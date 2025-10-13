package service

import (
	"context"
	"demo-api-bridge/internal/core/port"
	"time"
)

// healthService는 HealthCheckService 인터페이스를 구현합니다.
type healthService struct {
	routingRepo  port.RoutingRepository
	endpointRepo port.EndpointRepository
	cache        port.CacheRepository
	logger       port.Logger
	startTime    time.Time
}

// NewHealthCheckService는 새로운 HealthCheckService를 생성합니다.
func NewHealthCheckService(
	routingRepo port.RoutingRepository,
	endpointRepo port.EndpointRepository,
	cache port.CacheRepository,
	logger port.Logger,
) port.HealthCheckService {
	return &healthService{
		routingRepo:  routingRepo,
		endpointRepo: endpointRepo,
		cache:        cache,
		logger:       logger,
		startTime:    time.Now(),
	}
}

// CheckHealth는 서비스의 전반적인 상태를 확인합니다.
func (s *healthService) CheckHealth(ctx context.Context) error {
	s.logger.WithContext(ctx).Debug("performing health check")

	// TODO: 실제 상태 확인 로직 구현
	// 예: DB 연결 확인, 캐시 연결 확인 등

	return nil
}

// CheckReadiness는 서비스가 요청을 받을 준비가 되었는지 확인합니다.
func (s *healthService) CheckReadiness(ctx context.Context) error {
	s.logger.WithContext(ctx).Debug("performing readiness check")

	// TODO: 실제 준비 상태 확인 로직 구현
	// 예: 필수 의존성 연결 확인

	// 캐시 연결 확인 (예시)
	if s.cache != nil {
		if _, err := s.cache.Exists(ctx, "health_check"); err != nil {
			s.logger.WithContext(ctx).Warn("cache connection check failed", "error", err)
			// 캐시는 선택사항이므로 에러를 반환하지 않음
		}
	}

	return nil
}

// GetServiceStatus는 상세한 서비스 상태 정보를 반환합니다.
func (s *healthService) GetServiceStatus(ctx context.Context) map[string]interface{} {
	uptime := time.Since(s.startTime)

	status := map[string]interface{}{
		"service":   "api-bridge",
		"version":   "0.1.0",
		"uptime":    uptime.String(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// TODO: 추가 상태 정보
	// 예: 활성 연결 수, 처리 중인 요청 수, 메모리 사용량 등

	return status
}
