package port

import (
	"context"
	"demo-api-bridge/internal/core/domain"

	"github.com/sony/gobreaker"
)

// BridgeService는 API Bridge의 핵심 비즈니스 로직을 정의하는 인바운드 포트입니다.
// 이 인터페이스는 외부(HTTP Handler)에서 호출되며, 서비스 레이어에서 구현됩니다.
type BridgeService interface {
	// ProcessRequest는 API 요청을 처리하고 응답을 반환합니다.
	ProcessRequest(ctx context.Context, request *domain.Request) (*domain.Response, error)

	// GetRoutingRule은 요청에 매칭되는 라우팅 규칙을 조회합니다.
	GetRoutingRule(ctx context.Context, request *domain.Request) (*domain.RoutingRule, error)

	// GetEndpoint는 엔드포인트 ID로 엔드포인트 정보를 조회합니다.
	GetEndpoint(ctx context.Context, endpointID string) (*domain.APIEndpoint, error)
}

// RoutingService는 라우팅 관리를 담당하는 인바운드 포트입니다.
type RoutingService interface {
	// CreateRule은 새로운 라우팅 규칙을 생성합니다.
	CreateRule(ctx context.Context, rule *domain.RoutingRule) error

	// UpdateRule은 라우팅 규칙을 수정합니다.
	UpdateRule(ctx context.Context, rule *domain.RoutingRule) error

	// DeleteRule은 라우팅 규칙을 삭제합니다.
	DeleteRule(ctx context.Context, ruleID string) error

	// GetRule은 라우팅 규칙을 조회합니다.
	GetRule(ctx context.Context, ruleID string) (*domain.RoutingRule, error)

	// ListRules는 모든 라우팅 규칙을 조회합니다.
	ListRules(ctx context.Context) ([]*domain.RoutingRule, error)
}

// EndpointService는 엔드포인트 관리를 담당하는 인바운드 포트입니다.
type EndpointService interface {
	// CreateEndpoint는 새로운 엔드포인트를 생성합니다.
	CreateEndpoint(ctx context.Context, endpoint *domain.APIEndpoint) error

	// UpdateEndpoint는 엔드포인트를 수정합니다.
	UpdateEndpoint(ctx context.Context, endpoint *domain.APIEndpoint) error

	// DeleteEndpoint는 엔드포인트를 삭제합니다.
	DeleteEndpoint(ctx context.Context, endpointID string) error

	// GetEndpoint는 엔드포인트를 조회합니다.
	GetEndpoint(ctx context.Context, endpointID string) (*domain.APIEndpoint, error)

	// ListEndpoints는 모든 엔드포인트를 조회합니다.
	ListEndpoints(ctx context.Context) ([]*domain.APIEndpoint, error)
}

// HealthCheckService는 서비스 상태 확인을 담당하는 인바운드 포트입니다.
type HealthCheckService interface {
	// CheckHealth는 서비스의 전반적인 상태를 확인합니다.
	CheckHealth(ctx context.Context) error

	// CheckReadiness는 서비스가 요청을 받을 준비가 되었는지 확인합니다.
	CheckReadiness(ctx context.Context) error

	// GetServiceStatus는 상세한 서비스 상태 정보를 반환합니다.
	GetServiceStatus(ctx context.Context) map[string]interface{}
}

// OrchestrationService는 API 오케스트레이션을 담당하는 인바운드 포트입니다.
type OrchestrationService interface {
	// ProcessParallelRequest는 레거시와 모던 API를 병렬로 호출하고 결과를 비교합니다.
	ProcessParallelRequest(ctx context.Context, request *domain.Request, legacyEndpoint, modernEndpoint *domain.APIEndpoint) (*domain.APIComparison, error)

	// GetOrchestrationRule은 오케스트레이션 규칙을 조회합니다.
	GetOrchestrationRule(ctx context.Context, routingRuleID string) (*domain.OrchestrationRule, error)

	// CreateOrchestrationRule은 새로운 오케스트레이션 규칙을 생성합니다.
	CreateOrchestrationRule(ctx context.Context, rule *domain.OrchestrationRule) error

	// UpdateOrchestrationRule은 오케스트레이션 규칙을 수정합니다.
	UpdateOrchestrationRule(ctx context.Context, rule *domain.OrchestrationRule) error

	// EvaluateTransition는 전환 가능성을 평가합니다.
	EvaluateTransition(ctx context.Context, rule *domain.OrchestrationRule) (bool, error)

	// ExecuteTransition는 API 모드를 전환합니다.
	ExecuteTransition(ctx context.Context, rule *domain.OrchestrationRule, newMode domain.APIMode) error
}

// CircuitBreakerService는 Circuit Breaker 관리를 담당하는 인바운드 포트입니다.
type CircuitBreakerService interface {
	// GetOrCreateBreaker는 이름에 해당하는 Circuit Breaker를 가져오거나 생성합니다.
	GetOrCreateBreaker(name string, config domain.CircuitBreakerConfig) *gobreaker.CircuitBreaker

	// Execute는 Circuit Breaker를 통해 함수를 실행합니다.
	Execute(ctx context.Context, breakerName string, config domain.CircuitBreakerConfig, fn func() (interface{}, error)) (interface{}, error)

	// GetBreakerInfo는 Circuit Breaker의 현재 상태 정보를 반환합니다.
	GetBreakerInfo(breakerName string) (*domain.CircuitBreakerInfo, error)

	// GetAllBreakerInfos는 모든 Circuit Breaker의 상태 정보를 반환합니다.
	GetAllBreakerInfos() map[string]*domain.CircuitBreakerInfo

	// ResetBreaker는 Circuit Breaker를 리셋합니다.
	ResetBreaker(breakerName string) error
}
