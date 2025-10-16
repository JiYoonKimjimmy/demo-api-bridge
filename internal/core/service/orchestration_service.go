package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"sync"
	"time"
)

// orchestrationService는 OrchestrationService 인터페이스를 구현합니다.
type orchestrationService struct {
	orchestrationRepo port.OrchestrationRepository
	comparisonRepo    port.ComparisonRepository
	externalAPI       port.ExternalAPIClient
	logger            port.Logger
	metrics           port.MetricsCollector
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
