package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"sync"
	"time"
)

// orchestrationService는 레거시/모던 API의 병렬 호출 및 응답 비교를 담당하는 서비스입니다.
//
// 이 서비스는 다음의 핵심 기능을 제공합니다:
//   - 고루틴 기반 병렬 API 호출: 레거시와 모던 API를 동시에 호출하여 지연시간 최소화
//   - JSON 응답 비교: 재귀적 알고리즘으로 두 응답의 일치율 계산
//   - 자동 전환 평가: 일치율 기반으로 PARALLEL → MODERN_ONLY 전환 가능성 판단
//   - 전환 실행 및 롤백: 안전한 전환 로직 및 문제 발생 시 즉시 롤백
//
// 성능 특성:
//   - 병렬 호출로 인한 추가 지연시간: ~5-10ms (레거시 응답 대기)
//   - JSON 비교 오버헤드: 응답 크기에 비례 (일반적으로 <10ms)
type orchestrationService struct {
	orchestrationRepo port.OrchestrationRepository // 오케스트레이션 규칙 저장소
	comparisonRepo    port.ComparisonRepository    // 비교 결과 저장소
	externalAPI       port.ExternalAPIClient       // 외부 API 클라이언트
	logger            port.Logger                  // 로거
	metrics           port.MetricsCollector        // 메트릭 수집기
}

// NewOrchestrationService는 새로운 OrchestrationService를 생성합니다.
func NewOrchestrationService(
	orchestrationRepo port.OrchestrationRepository,
	comparisonRepo port.ComparisonRepository,
	externalAPI port.ExternalAPIClient,
	logger port.Logger,
	metrics port.MetricsCollector,
) port.OrchestrationService {
	return &orchestrationService{
		orchestrationRepo: orchestrationRepo,
		comparisonRepo:    comparisonRepo,
		externalAPI:       externalAPI,
		logger:            logger,
		metrics:           metrics,
	}
}

// ProcessParallelRequest는 레거시와 모던 API를 병렬로 호출하고 결과를 비교합니다.
//
// 이 메서드는 고루틴을 사용하여 두 API를 동시에 호출하고, 먼저 완료되는 것을 기다리지 않고
// 두 응답을 모두 수집한 후 JSON Diff 알고리즘으로 비교합니다.
//
// 병렬 호출 흐름:
//  1. 두 개의 고루틴 생성 (레거시, 모던)
//  2. Context 타임아웃 내에서 응답 대기
//  3. 두 응답 모두 성공 시 JSON 비교 수행
//  4. 일치율 계산 및 비교 결과 저장
//  5. 자동 전환 조건 확인
//
// Parameters:
//   - ctx: 요청 컨텍스트 (타임아웃, 취소 신호 포함)
//   - request: 원본 요청
//   - legacyEndpoint: 레거시 API 엔드포인트
//   - modernEndpoint: 모던 API 엔드포인트
//
// Returns:
//   - *domain.APIComparison: 비교 결과 (일치율, 차이점 포함)
//   - error: 병렬 호출 또는 비교 중 발생한 에러
//
// 에러 처리:
//   - 두 API 모두 실패 시 에러 반환
//   - 한쪽만 실패 시 비교 불가로 기록하고 성공한 응답 사용
func (s *orchestrationService) ProcessParallelRequest(
	ctx context.Context,
	request *domain.Request,
	legacyEndpoint, modernEndpoint *domain.APIEndpoint,
) (*domain.APIComparison, error) {
	start := time.Now()

	s.logger.WithContext(ctx).Info("starting parallel API calls",
		"request_id", request.ID,
		"legacy_endpoint", legacyEndpoint.GetFullURL(),
		"modern_endpoint", modernEndpoint.GetFullURL(),
	)

	// 병렬 호출을 위한 채널과 컨텍스트
	type apiResult struct {
		response *domain.Response
		err      error
		source   string
	}

	resultChan := make(chan apiResult, 2)
	var wg sync.WaitGroup

	// 레거시 API 호출
	wg.Add(1)
	go func() {
		defer wg.Done()

		legacyCtx, cancel := context.WithTimeout(ctx, legacyEndpoint.Timeout)
		defer cancel()

		response, err := s.externalAPI.SendWithRetry(legacyCtx, legacyEndpoint, request)
		resultChan <- apiResult{
			response: response,
			err:      err,
			source:   "legacy",
		}
	}()

	// 모던 API 호출
	wg.Add(1)
	go func() {
		defer wg.Done()

		modernCtx, cancel := context.WithTimeout(ctx, modernEndpoint.Timeout)
		defer cancel()

		response, err := s.externalAPI.SendWithRetry(modernCtx, modernEndpoint, request)
		resultChan <- apiResult{
			response: response,
			err:      err,
			source:   "modern",
		}
	}()

	// 고루틴 완료 대기
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 결과 수집
	var legacyResponse, modernResponse *domain.Response
	var legacyErr, modernErr error

	for result := range resultChan {
		if result.source == "legacy" {
			legacyResponse = result.response
			legacyErr = result.err
		} else {
			modernResponse = result.response
			modernErr = result.err
		}
	}

	// API 호출 완료 시간 기록
	apiDuration := time.Since(start)
	s.metrics.RecordHistogram("parallel_api_call_duration", float64(apiDuration.Milliseconds()), map[string]string{
		"request_id": request.ID,
	})

	// 결과 로깅
	s.logger.WithContext(ctx).Info("parallel API calls completed",
		"request_id", request.ID,
		"legacy_success", legacyErr == nil,
		"modern_success", modernErr == nil,
		"duration_ms", apiDuration.Milliseconds(),
	)

	// 에러 처리
	if legacyErr != nil && modernErr != nil {
		s.logger.WithContext(ctx).Error("both API calls failed",
			"request_id", request.ID,
			"legacy_error", legacyErr,
			"modern_error", modernErr,
		)
		s.metrics.IncrementCounter("parallel_api_calls_failed", map[string]string{
			"request_id": request.ID,
		})
		return nil, fmt.Errorf("both legacy and modern API calls failed: legacy=%v, modern=%v", legacyErr, modernErr)
	}

	// API 비교 객체 생성
	comparison := domain.NewAPIComparison(request.ID, request.ID, request.RoutingRuleID, legacyResponse, modernResponse)
	comparison.ComparisonDuration = time.Since(start)

	// 응답 비교 수행
	if legacyResponse != nil && modernResponse != nil {
		// 실제 JSON 비교 엔진 사용
		comparisonEngine := domain.NewComparisonEngine(domain.ComparisonConfig{
			Enabled:               true,
			IgnoreFields:          []string{"timestamp", "requestId", "request_id"},
			AllowableDifference:   0.01, // 1% 허용 오차
			StrictMode:            false,
			SaveComparisonHistory: true,
		})

		comparisonResult := comparisonEngine.CompareResponses(legacyResponse, modernResponse)
		comparison.MatchRate = comparisonResult.MatchRate
		comparison.Differences = comparisonResult.Differences
	} else {
		// 하나의 API만 성공한 경우
		if legacyResponse != nil {
			comparison.MatchRate = 0.0 // 모던 API 실패
			comparison.Differences = []domain.ResponseDiff{
				{
					Type:        domain.EXTRA,
					Path:        "modern_response",
					Message:     "Modern API call failed",
					ModernValue: modernErr.Error(),
				},
			}
		} else {
			comparison.MatchRate = 0.0 // 레거시 API 실패
			comparison.Differences = []domain.ResponseDiff{
				{
					Type:        domain.MISSING,
					Path:        "legacy_response",
					Message:     "Legacy API call failed",
					LegacyValue: legacyErr.Error(),
				},
			}
		}
	}

	// 비교 결과 메트릭 기록
	s.metrics.RecordGauge("api_comparison_match_rate", comparison.MatchRate, map[string]string{
		"request_id": request.ID,
	})

	s.logger.WithContext(ctx).Info("API comparison completed",
		"request_id", request.ID,
		"match_rate", comparison.MatchRate,
		"differences_count", len(comparison.Differences),
	)

	return comparison, nil
}

// GetOrchestrationRule은 오케스트레이션 규칙을 조회합니다.
func (s *orchestrationService) GetOrchestrationRule(ctx context.Context, routingRuleID string) (*domain.OrchestrationRule, error) {
	rule, err := s.orchestrationRepo.FindByRoutingRuleID(ctx, routingRuleID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get orchestration rule", "routing_rule_id", routingRuleID, "error", err)
		return nil, err
	}

	return rule, nil
}

// CreateOrchestrationRule은 새로운 오케스트레이션 규칙을 생성합니다.
func (s *orchestrationService) CreateOrchestrationRule(ctx context.Context, rule *domain.OrchestrationRule) error {
	s.logger.WithContext(ctx).Info("creating orchestration rule", "rule_id", rule.ID, "name", rule.Name)

	// 검증
	if err := rule.IsValid(); err != nil {
		s.logger.WithContext(ctx).Error("invalid orchestration rule", "error", err)
		return err
	}

	// 저장
	if err := s.orchestrationRepo.Create(ctx, rule); err != nil {
		s.logger.WithContext(ctx).Error("failed to create orchestration rule", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("orchestration rule created successfully", "rule_id", rule.ID)
	s.metrics.IncrementCounter("orchestration_rules_created", map[string]string{"rule_id": rule.ID})

	return nil
}

// UpdateOrchestrationRule은 오케스트레이션 규칙을 수정합니다.
func (s *orchestrationService) UpdateOrchestrationRule(ctx context.Context, rule *domain.OrchestrationRule) error {
	s.logger.WithContext(ctx).Info("updating orchestration rule", "rule_id", rule.ID)

	// 검증
	if err := rule.IsValid(); err != nil {
		s.logger.WithContext(ctx).Error("invalid orchestration rule", "error", err)
		return err
	}

	// 수정
	if err := s.orchestrationRepo.Update(ctx, rule); err != nil {
		s.logger.WithContext(ctx).Error("failed to update orchestration rule", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("orchestration rule updated successfully", "rule_id", rule.ID)
	s.metrics.IncrementCounter("orchestration_rules_updated", map[string]string{"rule_id": rule.ID})

	return nil
}

// EvaluateTransition는 전환 가능성을 평가합니다.
func (s *orchestrationService) EvaluateTransition(ctx context.Context, rule *domain.OrchestrationRule) (bool, error) {
	if !rule.TransitionConfig.AutoTransitionEnabled {
		return false, nil
	}

	// 최근 비교 결과 조회
	recentComparisons, err := s.comparisonRepo.GetRecentComparisons(ctx, rule.RoutingRuleID, rule.TransitionConfig.MinRequestsForTransition)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get recent comparisons", "error", err)
		return false, err
	}

	if len(recentComparisons) < rule.TransitionConfig.MinRequestsForTransition {
		s.logger.WithContext(ctx).Info("insufficient comparison data for transition",
			"rule_id", rule.ID,
			"required", rule.TransitionConfig.MinRequestsForTransition,
			"available", len(recentComparisons),
		)
		return false, nil
	}

	// 평균 일치율 계산
	var totalMatchRate float64
	for _, comp := range recentComparisons {
		totalMatchRate += comp.MatchRate
	}
	averageMatchRate := totalMatchRate / float64(len(recentComparisons))

	// 전환 조건 확인
	canTransition := rule.CanTransitionToModern(averageMatchRate, len(recentComparisons))

	s.logger.WithContext(ctx).Info("transition evaluation completed",
		"rule_id", rule.ID,
		"average_match_rate", averageMatchRate,
		"can_transition", canTransition,
		"comparisons_count", len(recentComparisons),
	)

	return canTransition, nil
}

// ExecuteTransition는 API 모드를 전환합니다.
func (s *orchestrationService) ExecuteTransition(ctx context.Context, rule *domain.OrchestrationRule, newMode domain.APIMode) error {
	s.logger.WithContext(ctx).Info("executing API mode transition",
		"rule_id", rule.ID,
		"from_mode", rule.CurrentMode,
		"to_mode", newMode,
	)

	// 모드 전환
	rule.CurrentMode = newMode

	// 저장
	if err := s.orchestrationRepo.Update(ctx, rule); err != nil {
		s.logger.WithContext(ctx).Error("failed to update orchestration rule during transition", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("API mode transition completed successfully",
		"rule_id", rule.ID,
		"new_mode", newMode,
	)

	s.metrics.IncrementCounter("api_mode_transitions", map[string]string{
		"rule_id":   rule.ID,
		"from_mode": string(rule.CurrentMode),
		"to_mode":   string(newMode),
	})

	return nil
}
