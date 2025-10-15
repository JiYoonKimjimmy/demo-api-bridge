package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"time"
)

// bridgeService는 BridgeService 인터페이스를 구현합니다.
type bridgeService struct {
	routingRepo       port.RoutingRepository
	endpointRepo      port.EndpointRepository
	orchestrationRepo port.OrchestrationRepository
	comparisonRepo    port.ComparisonRepository
	orchestrationSvc  port.OrchestrationService
	externalAPI       port.ExternalAPIClient
	cache             port.CacheRepository
	logger            port.Logger
	metrics           port.MetricsCollector
}

// NewBridgeService는 새로운 BridgeService를 생성합니다.
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
		routingRepo:       routingRepo,
		endpointRepo:      endpointRepo,
		orchestrationRepo: orchestrationRepo,
		comparisonRepo:    comparisonRepo,
		orchestrationSvc:  orchestrationSvc,
		externalAPI:       externalAPI,
		cache:             cache,
		logger:            logger,
		metrics:           metrics,
	}
}

// ProcessRequest는 API 요청을 처리하고 응답을 반환합니다.
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
		// 오케스트레이션 규칙이 없으면 단일 API 호출
		return s.processSingleAPIRequest(ctx, request, rule, start)
	}

	// 4. 오케스트레이션 모드에 따른 처리
	return s.processOrchestratedRequest(ctx, request, rule, orchestrationRule, start)
}

// GetRoutingRule은 요청에 매칭되는 라우팅 규칙을 조회합니다.
func (s *bridgeService) GetRoutingRule(ctx context.Context, request *domain.Request) (*domain.RoutingRule, error) {
	rules, err := s.routingRepo.FindMatchingRules(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, domain.ErrRouteNotFound
	}

	// 우선순위가 가장 높은 (숫자가 낮은) 규칙 선택
	selectedRule := rules[0]
	for _, rule := range rules {
		if rule.Priority < selectedRule.Priority {
			selectedRule = rule
		}
	}

	return selectedRule, nil
}

// GetEndpoint는 엔드포인트 ID로 엔드포인트 정보를 조회합니다.
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

// processParallelRequest는 레거시와 모던 API를 병렬로 호출합니다.
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
