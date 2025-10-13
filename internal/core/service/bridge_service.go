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
	routingRepo  port.RoutingRepository
	endpointRepo port.EndpointRepository
	externalAPI  port.ExternalAPIClient
	cache        port.CacheRepository
	logger       port.Logger
	metrics      port.MetricsCollector
}

// NewBridgeService는 새로운 BridgeService를 생성합니다.
func NewBridgeService(
	routingRepo port.RoutingRepository,
	endpointRepo port.EndpointRepository,
	externalAPI port.ExternalAPIClient,
	cache port.CacheRepository,
	logger port.Logger,
	metrics port.MetricsCollector,
) port.BridgeService {
	return &bridgeService{
		routingRepo:  routingRepo,
		endpointRepo: endpointRepo,
		externalAPI:  externalAPI,
		cache:        cache,
		logger:       logger,
		metrics:      metrics,
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

	// 3. 엔드포인트 조회
	endpoint, err := s.GetEndpoint(ctx, rule.EndpointID)
	if err != nil {
		s.logger.WithContext(ctx).Error("endpoint not found", "error", err)
		s.metrics.RecordRequest(request.Method, request.Path, 404, time.Since(start))
		return nil, err
	}

	// 4. 캐시 확인 (캐시가 활성화된 경우)
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

	// 5. 외부 API 호출
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

	// 6. 캐시 저장 (성공한 경우)
	if rule.CacheEnabled && response.IsSuccess() {
		cacheKey := s.generateCacheKey(request)
		cacheTTL := time.Duration(rule.CacheTTL) * time.Second
		if err := s.cache.Set(ctx, cacheKey, response.Body, cacheTTL); err != nil {
			s.logger.WithContext(ctx).Warn("failed to save cache", "error", err)
		}
	}

	// 7. 응답 반환
	response.SetDuration(start)
	s.metrics.RecordRequest(request.Method, request.Path, response.StatusCode, time.Since(start))

	s.logger.WithContext(ctx).Info("request processed successfully",
		"request_id", request.ID,
		"status_code", response.StatusCode,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return response, nil
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

// generateCacheKey는 요청으로부터 캐시 키를 생성합니다.
func (s *bridgeService) generateCacheKey(request *domain.Request) string {
	return fmt.Sprintf("api_bridge:%s:%s", request.Method, request.Path)
}
