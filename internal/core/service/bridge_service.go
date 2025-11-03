// Package service는 API Bridge의 핵심 비즈니스 로직을 구현합니다.
//
// 이 패키지는 Hexagonal Architecture의 Application Layer에 해당하며,
// 도메인 로직을 조율하고 인바운드/아웃바운드 포트를 연결합니다.
package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"strings"
	"sync"
	"time"
)

// routingRuleCacheEntry는 라우팅 규칙 캐시 엔트리입니다.
type routingRuleCacheEntry struct {
	rules     []*domain.RoutingRule
	timestamp time.Time
}

// bridgeService
// : BridgeService 인터페이스를 구현하는 핵심 서비스입니다.
//
// 이 서비스는 다음의 책임을 가집니다:
//   - 클라이언트 요청을 적절한 외부 API로 라우팅
//   - 레거시/모던 API의 병렬 호출 및 응답 비교
//   - 자동 전환 로직 실행 및 모니터링
//   - 캐싱 및 메트릭 수집
//
// 사용 예:
//
//	service := NewBridgeService(...)
//	response, err := service.ProcessRequest(ctx, request)
type bridgeService struct {
	routingRepo       port.RoutingRepository       // 라우팅 규칙 저장소
	endpointRepo      port.EndpointRepository      // API 엔드포인트 저장소
	orchestrationRepo port.OrchestrationRepository // 오케스트레이션 규칙 저장소
	comparisonRepo    port.ComparisonRepository    // 비교 결과 저장소
	orchestrationSvc  port.OrchestrationService    // 오케스트레이션 서비스
	externalAPI       port.ExternalAPIClient       // 외부 API 클라이언트
	cache             port.CacheRepository         // 캐시 저장소
	logger            port.Logger                  // 로거
	metrics           port.MetricsCollector        // 메트릭 수집기

	// 라우팅 규칙 캐시
	routingRuleCache    map[string]*routingRuleCacheEntry // 캐시 맵 (key: method:path)
	routingRuleCacheMu  sync.RWMutex                      // 캐시 락
	routingRuleCacheTTL time.Duration                     // 캐시 TTL (기본: 60초)
}

// NewBridgeService
// : 새로운 BridgeService 인스턴스를 생성합니다.
//
// 이 팩토리 함수는 의존성 주입 패턴을 사용하여 필요한 모든 저장소와 서비스를 주입받습니다.
// 반환된 서비스는 즉시 요청 처리를 시작할 수 있습니다.
//
// Parameters:
//   - routingRepo: 라우팅 규칙을 조회하는 저장소
//   - endpointRepo: API 엔드포인트 정보를 조회하는 저장소
//   - orchestrationRepo: 오케스트레이션 규칙을 조회하는 저장소
//   - comparisonRepo: 비교 결과를 저장하는 저장소
//   - orchestrationSvc: 병렬 호출 및 응답 비교를 수행하는 서비스
//   - externalAPI: 외부 API 호출을 담당하는 클라이언트
//   - cache: 라우팅 규칙 및 응답을 캐싱하는 저장소
//   - logger: 구조화된 로깅을 제공하는 로거
//   - metrics: Prometheus 메트릭을 수집하는 컬렉터
//
// Returns:
//   - port.BridgeService: 완전히 초기화된 Bridge 서비스 인터페이스
func NewBridgeService(
	routingRepo port.RoutingRepository,
	endpointRepo port.EndpointRepository,
	orchestrationRepo port.OrchestrationRepository,
	comparisonRepo port.ComparisonRepository,
	orchestrationSvc port.OrchestrationService,
	externalAPI port.ExternalAPIClient,
	cache port.CacheRepository,
	logger port.Logger,
	metrics port.MetricsCollector,
) port.BridgeService {
	return &bridgeService{
		routingRepo:         routingRepo,
		endpointRepo:        endpointRepo,
		orchestrationRepo:   orchestrationRepo,
		comparisonRepo:      comparisonRepo,
		orchestrationSvc:    orchestrationSvc,
		externalAPI:         externalAPI,
		cache:               cache,
		logger:              logger,
		metrics:             metrics,
		routingRuleCache:    make(map[string]*routingRuleCacheEntry),
		routingRuleCacheTTL: 60 * time.Second, // 60초 TTL
	}
}

// ProcessRequest
// : 클라이언트로부터 받은 API 요청을 처리하고 응답을 반환합니다.
//
// 이 메서드는 API Bridge의 핵심 로직으로, 다음 단계를 수행합니다:
//  1. 요청 유효성 검증
//  2. 라우팅 규칙 조회 (캐시 우선, DB 조회)
//  3. 오케스트레이션 규칙 확인
//  4. 요청 처리 모드 결정:
//     - PARALLEL: 레거시/모던 API 병렬 호출 후 레거시 응답 반환
//     - MODERN_ONLY: 모던 API만 호출
//     - LEGACY_ONLY: 레거시 API만 호출
//  5. 응답 비교 및 전환 로직 실행 (PARALLEL 모드)
//  6. 메트릭 수집 및 로깅
//
// Parameters:
//   - ctx: 요청의 컨텍스트 (타임아웃, 취소, Trace ID 포함)
//   - request: 처리할 요청 객체
//
// Returns:
//   - *domain.Response: 외부 API로부터 받은 응답
//   - error: 요청 처리 중 발생한 에러
//
// 에러 케이스:
//   - domain.ErrInvalidRequest: 요청 검증 실패
//   - domain.ErrRouteNotFound: 매칭되는 라우팅 규칙 없음
//   - domain.ErrExternalAPIFailed: 외부 API 호출 실패
func (s *bridgeService) ProcessRequest(ctx context.Context, request *domain.Request) (*domain.Response, error) {
	start := time.Now()

	// 로깅
	s.logger.WithContext(ctx).Info("processing request",
		"request_id", request.ID,
		"method", request.Method,
		"path", request.Path,
	)

	// 1. 요청 검증
	if err := request.IsValid(); err != nil {
		s.logger.WithContext(ctx).Error("invalid request", "error", err)
		s.metrics.RecordRequest(request.Method, request.Path, 400, time.Since(start))
		return nil, err
	}

	// 2. 라우팅 규칙 조회
	rule, err := s.GetRoutingRule(ctx, request)
	if err != nil {
		s.logger.WithContext(ctx).Error("routing rule not found", "error", err)
		s.metrics.RecordRequest(request.Method, request.Path, 404, time.Since(start))
		return nil, err
	}

	// 3. 오케스트레이션 규칙 확인
	orchestrationRule, err := s.orchestrationRepo.FindByRoutingRuleID(ctx, rule.ID)
	if err != nil {
		// 기본 라우팅 규칙인 경우, 기본 오케스트레이션 규칙 생성 시도
		if rule.ID == "default-legacy-route" {
			defaultOrchRule, createErr := s.createDefaultOrchestrationRule(ctx, request, rule)
			if createErr == nil {
				s.logger.WithContext(ctx).Info("using default orchestration for unmatched route",
					"request", fmt.Sprintf("%s %s", request.Method, request.Path),
				)
				s.metrics.RecordDefaultOrchestrationUsed(request.Method, request.Path)
				return s.processOrchestratedRequest(ctx, request, rule, defaultOrchRule, start)
			}
			s.logger.WithContext(ctx).Warn("failed to create default orchestration, falling back to single API",
				"error", createErr,
			)
		}

		// 오케스트레이션 규칙이 없으면 단일 API 호출
		return s.processSingleAPIRequest(ctx, request, rule, start)
	}

	// 4. 오케스트레이션 모드에 따른 처리
	return s.processOrchestratedRequest(ctx, request, rule, orchestrationRule, start)
}

// GetRoutingRule
// : 요청에 매칭되는 라우팅 규칙을 조회합니다.
//
// 이 메서드는 다음 순서로 라우팅 규칙을 찾습니다:
//  1. 인메모리 캐시에서 조회 (TTL 기반)
//  2. 캐시 미스 시 DB에서 조회 후 캐시에 저장
//  3. 매칭되는 규칙이 없으면 기본 레거시 엔드포인트로 fallback
//
// Parameters:
//   - ctx: 요청 컨텍스트
//   - request: 처리할 요청 객체
//
// Returns:
//   - *domain.RoutingRule: 매칭된 라우팅 규칙 (또는 기본 규칙)
//   - error: 조회 중 발생한 에러
func (s *bridgeService) GetRoutingRule(ctx context.Context, request *domain.Request) (*domain.RoutingRule, error) {
	cacheKey := s.generateRoutingCacheKey(request)

	// 1. 캐시에서 조회
	s.routingRuleCacheMu.RLock()
	if entry, exists := s.routingRuleCache[cacheKey]; exists {
		// TTL 체크
		if time.Since(entry.timestamp) < s.routingRuleCacheTTL {
			s.routingRuleCacheMu.RUnlock()

			// 캐시된 규칙이 있으면 우선순위 기반 선택
			if len(entry.rules) > 0 {
				return s.selectHighestPriorityRule(entry.rules), nil
			}

			// 캐시에 규칙이 없으면 기본 레거시 엔드포인트로 fallback
			s.logger.WithContext(ctx).Info("no cached routing rules, using default legacy endpoint",
				"request", fmt.Sprintf("%s %s", request.Method, request.Path),
			)
			s.metrics.RecordDefaultRoutingUsed(request.Method, request.Path)
			return s.createDefaultRoutingRule(ctx, request)
		}
	}
	s.routingRuleCacheMu.RUnlock()

	// 2. DB에서 조회
	rules, err := s.routingRepo.FindMatchingRules(ctx, request)
	if err != nil {
		// 에러 메시지의 개행/탭 문자를 공백으로 치환하여 한 줄로 출력
		errorMsg := strings.ReplaceAll(err.Error(), "\n", " ")
		errorMsg = strings.ReplaceAll(errorMsg, "\t", " ")
		errorMsg = strings.TrimSpace(errorMsg)

		s.logger.WithContext(ctx).Warn("DB query failed, using default legacy endpoint",
			"error", errorMsg,
			"request", fmt.Sprintf("%s %s", request.Method, request.Path),
		)
		// DB 조회 실패 시에도 기본 레거시 엔드포인트로 fallback
		s.metrics.RecordDefaultRoutingUsed(request.Method, request.Path)
		return s.createDefaultRoutingRule(ctx, request)
	}

	// 3. 캐시에 저장 (규칙이 없어도 저장하여 반복 DB 조회 방지)
	s.routingRuleCacheMu.Lock()
	s.routingRuleCache[cacheKey] = &routingRuleCacheEntry{
		rules:     rules,
		timestamp: time.Now(),
	}
	s.routingRuleCacheMu.Unlock()

	// 4. 매칭된 규칙이 있으면 반환
	if len(rules) > 0 {
		return s.selectHighestPriorityRule(rules), nil
	}

	// 5. 매칭된 규칙이 없으면 기본 레거시 엔드포인트로 fallback
	s.logger.WithContext(ctx).Info("no matching routing rules, using default legacy endpoint",
		"request", fmt.Sprintf("%s %s", request.Method, request.Path),
	)
	s.metrics.RecordDefaultRoutingUsed(request.Method, request.Path)
	return s.createDefaultRoutingRule(ctx, request)
}

// selectHighestPriorityRule은 우선순위가 가장 높은 (숫자가 낮은) 규칙을 선택합니다.
func (s *bridgeService) selectHighestPriorityRule(rules []*domain.RoutingRule) *domain.RoutingRule {
	if len(rules) == 0 {
		return nil
	}

	selectedRule := rules[0]
	for _, rule := range rules {
		if rule.Priority < selectedRule.Priority {
			selectedRule = rule
		}
	}

	return selectedRule
}

// createDefaultRoutingRule은 기본 레거시 엔드포인트를 사용하는 라우팅 규칙을 생성합니다.
func (s *bridgeService) createDefaultRoutingRule(ctx context.Context, request *domain.Request) (*domain.RoutingRule, error) {
	// 기본 레거시 엔드포인트 조회
	defaultEndpoint, err := s.endpointRepo.FindDefaultLegacyEndpoint(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to find default legacy endpoint", "error", err)
		return nil, fmt.Errorf("no routing rule found and failed to get default endpoint: %w", err)
	}

	// 동적으로 라우팅 규칙 생성
	defaultRule := &domain.RoutingRule{
		ID:           "default-legacy-route",
		Name:         "Default Legacy Route",
		PathPattern:  request.Path,
		Method:       request.Method,
		EndpointID:   defaultEndpoint.ID,
		Priority:     9999, // 가장 낮은 우선순위
		IsActive:     true,
		CacheEnabled: false,
		Description:  "Auto-generated default routing rule for legacy endpoint",
	}

	return defaultRule, nil
}

// createDefaultOrchestrationRule은 기본 오케스트레이션 규칙을 생성합니다.
//
// 라우팅 규칙이 없는 경우, 레거시와 모던 엔드포인트를 모두 사용하는
// 기본 오케스트레이션 규칙을 동적으로 생성합니다.
//
// Parameters:
//   - ctx: 요청 컨텍스트
//   - request: 처리할 요청 객체
//   - defaultRoutingRule: 기본 라우팅 규칙
//
// Returns:
//   - *domain.OrchestrationRule: 동적으로 생성된 오케스트레이션 규칙
//   - error: 모던 엔드포인트 조회 실패 시
func (s *bridgeService) createDefaultOrchestrationRule(
	ctx context.Context,
	request *domain.Request,
	defaultRoutingRule *domain.RoutingRule,
) (*domain.OrchestrationRule, error) {
	// 1. 기본 레거시 엔드포인트 조회
	defaultLegacyEndpoint, err := s.endpointRepo.FindDefaultLegacyEndpoint(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to find default legacy endpoint for orchestration", "error", err)
		return nil, fmt.Errorf("no default legacy endpoint for orchestration: %w", err)
	}

	// 2. 기본 모던 엔드포인트 조회
	defaultModernEndpoint, err := s.endpointRepo.FindDefaultModernEndpoint(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Warn("failed to find default modern endpoint, cannot use orchestration", "error", err)
		return nil, fmt.Errorf("no default modern endpoint for orchestration: %w", err)
	}

	// 3. 동적 오케스트레이션 규칙 생성
	now := time.Now()
	defaultOrchRule := &domain.OrchestrationRule{
		ID:               "default-orchestration-rule",
		Name:             "Default Parallel Orchestration",
		RoutingRuleID:    defaultRoutingRule.ID,
		LegacyEndpointID: defaultLegacyEndpoint.ID,
		ModernEndpointID: defaultModernEndpoint.ID,
		CurrentMode:      domain.PARALLEL,
		TransitionConfig: domain.TransitionConfig{
			AutoTransitionEnabled:    true,
			MatchRateThreshold:       0.95, // 95% 일치 시 전환
			StabilityPeriod:          24 * time.Hour,
			MinRequestsForTransition: 100,
			RollbackThreshold:        0.90, // 90% 미만 시 롤백
		},
		ComparisonConfig: domain.ComparisonConfig{
			Enabled:               true,
			IgnoreFields:          []string{"timestamp", "requestId"},
			AllowableDifference:   0.01, // 1% 허용 오차
			StrictMode:            false,
			SaveComparisonHistory: true,
		},
		IsActive:    true,
		Description: "Auto-generated default orchestration for unmatched routes",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.logger.WithContext(ctx).Info("created default orchestration rule",
		"legacy_endpoint", defaultLegacyEndpoint.ID,
		"modern_endpoint", defaultModernEndpoint.ID,
		"mode", defaultOrchRule.CurrentMode,
	)

	return defaultOrchRule, nil
}

// generateRoutingCacheKey는 라우팅 캐시 키를 생성합니다.
func (s *bridgeService) generateRoutingCacheKey(request *domain.Request) string {
	return fmt.Sprintf("abs:routing:%s:%s", request.Method, request.Path)
}

// GetEndpoint
// : 엔드포인트 ID로 엔드포인트 정보를 조회합니다.
func (s *bridgeService) GetEndpoint(ctx context.Context, endpointID string) (*domain.APIEndpoint, error) {
	endpoint, err := s.endpointRepo.FindByID(ctx, endpointID)
	if err != nil {
		return nil, err
	}

	if !endpoint.IsActive {
		return nil, domain.NewDomainError("ENDPOINT_INACTIVE", "endpoint is not active", nil)
	}

	return endpoint, nil
}

// processSingleAPIRequest는 단일 API 요청을 처리합니다 (기존 로직).
func (s *bridgeService) processSingleAPIRequest(ctx context.Context, request *domain.Request, rule *domain.RoutingRule, start time.Time) (*domain.Response, error) {
	// 엔드포인트 조회
	endpoint, err := s.GetEndpoint(ctx, rule.EndpointID)
	if err != nil {
		s.logger.WithContext(ctx).Error("endpoint not found", "error", err)
		s.metrics.RecordRequest(request.Method, request.Path, 404, time.Since(start))
		return nil, err
	}

	// 캐시 확인 (캐시가 활성화된 경우)
	if rule.CacheEnabled {
		cacheKey := s.generateCacheKey(request)
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
			s.logger.WithContext(ctx).Info("cache hit", "key", cacheKey)
			s.metrics.RecordCacheHit(true)
			s.metrics.RecordRequest(request.Method, request.Path, 200, time.Since(start))

			response := domain.NewResponse(request.ID)
			response.Body = cached
			response.StatusCode = 200
			response.Source = "cache"
			response.SetDuration(start)
			return response, nil
		}
		s.metrics.RecordCacheHit(false)
	}

	// 외부 API 호출
	apiStart := time.Now()
	response, err := s.externalAPI.SendWithRetry(ctx, endpoint, request)
	apiDuration := time.Since(apiStart)

	if err != nil {
		s.logger.WithContext(ctx).Error("external API call failed", "error", err)
		s.metrics.RecordExternalAPICall(endpoint.GetFullURL(), false, apiDuration)
		s.metrics.RecordRequest(request.Method, request.Path, 500, time.Since(start))
		return nil, err
	}

	s.metrics.RecordExternalAPICall(endpoint.GetFullURL(), true, apiDuration)

	// 캐시 저장 (성공한 경우)
	if rule.CacheEnabled && response.IsSuccess() {
		cacheKey := s.generateCacheKey(request)
		cacheTTL := time.Duration(rule.CacheTTL) * time.Second
		if err := s.cache.Set(ctx, cacheKey, response.Body, cacheTTL); err != nil {
			s.logger.WithContext(ctx).Warn("failed to save cache", "error", err)
		}
	}

	// 응답 반환
	response.SetDuration(start)
	s.metrics.RecordRequest(request.Method, request.Path, response.StatusCode, time.Since(start))

	s.logger.WithContext(ctx).Info("single API request processed successfully",
		"request_id", request.ID,
		"status_code", response.StatusCode,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return response, nil
}

// processOrchestratedRequest는 오케스트레이션된 요청을 처리합니다.
func (s *bridgeService) processOrchestratedRequest(ctx context.Context, request *domain.Request, rule *domain.RoutingRule, orchestrationRule *domain.OrchestrationRule, start time.Time) (*domain.Response, error) {
	s.logger.WithContext(ctx).Info("processing orchestrated request",
		"request_id", request.ID,
		"current_mode", orchestrationRule.CurrentMode,
	)

	switch orchestrationRule.CurrentMode {
	case domain.LEGACY_ONLY:
		return s.processLegacyOnlyRequest(ctx, request, orchestrationRule, start)
	case domain.MODERN_ONLY:
		return s.processModernOnlyRequest(ctx, request, orchestrationRule, start)
	case domain.PARALLEL:
		return s.processParallelRequest(ctx, request, orchestrationRule, start)
	default:
		return s.processParallelRequest(ctx, request, orchestrationRule, start)
	}
}

// processLegacyOnlyRequest는 레거시 API만 호출합니다.
func (s *bridgeService) processLegacyOnlyRequest(ctx context.Context, request *domain.Request, rule *domain.OrchestrationRule, start time.Time) (*domain.Response, error) {
	legacyEndpoint, err := s.GetEndpoint(ctx, rule.LegacyEndpointID)
	if err != nil {
		s.logger.WithContext(ctx).Error("legacy endpoint not found", "error", err)
		return nil, err
	}

	response, err := s.externalAPI.SendWithRetry(ctx, legacyEndpoint, request)
	if err != nil {
		s.logger.WithContext(ctx).Error("legacy API call failed", "error", err)
		return nil, err
	}

	response.SetDuration(start)
	s.metrics.RecordRequest(request.Method, request.Path, response.StatusCode, time.Since(start))

	s.logger.WithContext(ctx).Info("legacy-only request processed successfully",
		"request_id", request.ID,
		"status_code", response.StatusCode,
		"source", "legacy",
	)

	return response, nil
}

// processModernOnlyRequest는 모던 API만 호출합니다.
func (s *bridgeService) processModernOnlyRequest(ctx context.Context, request *domain.Request, rule *domain.OrchestrationRule, start time.Time) (*domain.Response, error) {
	modernEndpoint, err := s.GetEndpoint(ctx, rule.ModernEndpointID)
	if err != nil {
		s.logger.WithContext(ctx).Error("modern endpoint not found", "error", err)
		return nil, err
	}

	response, err := s.externalAPI.SendWithRetry(ctx, modernEndpoint, request)
	if err != nil {
		s.logger.WithContext(ctx).Error("modern API call failed", "error", err)
		return nil, err
	}

	response.SetDuration(start)
	s.metrics.RecordRequest(request.Method, request.Path, response.StatusCode, time.Since(start))

	s.logger.WithContext(ctx).Info("modern-only request processed successfully",
		"request_id", request.ID,
		"status_code", response.StatusCode,
		"source", "modern",
	)

	return response, nil
}

// processParallelRequest : 레거시와 모던 API를 병렬로 호출합니다.
func (s *bridgeService) processParallelRequest(ctx context.Context, request *domain.Request, rule *domain.OrchestrationRule, start time.Time) (*domain.Response, error) {
	// 엔드포인트 조회
	legacyEndpoint, err := s.GetEndpoint(ctx, rule.LegacyEndpointID)
	if err != nil {
		s.logger.WithContext(ctx).Error("legacy endpoint not found", "error", err)
		return nil, err
	}

	modernEndpoint, err := s.GetEndpoint(ctx, rule.ModernEndpointID)
	if err != nil {
		s.logger.WithContext(ctx).Error("modern endpoint not found", "error", err)
		return nil, err
	}

	// 병렬 호출 및 비교
	comparison, err := s.orchestrationSvc.ProcessParallelRequest(ctx, request, legacyEndpoint, modernEndpoint)
	if err != nil {
		s.logger.WithContext(ctx).Error("parallel request processing failed", "error", err)
		return nil, err
	}

	// 비교 결과 저장
	if rule.ComparisonConfig.SaveComparisonHistory {
		if err := s.comparisonRepo.SaveComparison(ctx, comparison); err != nil {
			s.logger.WithContext(ctx).Warn("failed to save comparison result", "error", err)
		}
	}

	// 응답 결정 (레거시 우선)
	var response *domain.Response
	if comparison.LegacyResponse != nil {
		response = comparison.LegacyResponse
		response.Source = "legacy"
	} else if comparison.ModernResponse != nil {
		response = comparison.ModernResponse
		response.Source = "modern"
	} else {
		return nil, fmt.Errorf("both API calls failed")
	}

	response.SetDuration(start)
	s.metrics.RecordRequest(request.Method, request.Path, response.StatusCode, time.Since(start))

	s.logger.WithContext(ctx).Info("parallel request processed successfully",
		"request_id", request.ID,
		"match_rate", comparison.MatchRate,
		"differences_count", len(comparison.Differences),
		"returned_source", response.Source,
	)

	// 전환 평가 (백그라운드)
	go s.evaluateTransitionAsync(ctx, rule)

	return response, nil
}

// evaluateTransitionAsync는 백그라운드에서 전환을 평가합니다.
func (s *bridgeService) evaluateTransitionAsync(ctx context.Context, rule *domain.OrchestrationRule) {
	canTransition, err := s.orchestrationSvc.EvaluateTransition(ctx, rule)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to evaluate transition", "rule_id", rule.ID, "error", err)
		return
	}

	if canTransition {
		s.logger.WithContext(ctx).Info("transition condition met, executing transition", "rule_id", rule.ID)
		if err := s.orchestrationSvc.ExecuteTransition(ctx, rule, domain.MODERN_ONLY); err != nil {
			s.logger.WithContext(ctx).Error("failed to execute transition", "rule_id", rule.ID, "error", err)
		}
	}
}

// generateCacheKey는 요청으로부터 캐시 키를 생성합니다.
func (s *bridgeService) generateCacheKey(request *domain.Request) string {
	return fmt.Sprintf("api_bridge:%s:%s", request.Method, request.Path)
}
