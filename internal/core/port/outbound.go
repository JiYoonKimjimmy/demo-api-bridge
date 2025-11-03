package port

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"time"
)

// ExternalAPIClient는 외부 API 호출을 담당하는 아웃바운드 포트입니다.
// 이 인터페이스는 서비스 레이어에서 사용되며, HTTP Client 어댑터에서 구현됩니다.
type ExternalAPIClient interface {
	// SendRequest는 외부 API에 요청을 전송하고 응답을 받습니다.
	SendRequest(ctx context.Context, endpoint *domain.APIEndpoint, request *domain.Request) (*domain.Response, error)

	// SendWithRetry는 재시도 로직을 포함하여 외부 API에 요청을 전송합니다.
	SendWithRetry(ctx context.Context, endpoint *domain.APIEndpoint, request *domain.Request) (*domain.Response, error)
}

// CacheRepository는 캐시 저장소를 담당하는 아웃바운드 포트입니다.
// 이 인터페이스는 서비스 레이어에서 사용되며, Redis 어댑터에서 구현됩니다.
type CacheRepository interface {
	// Get은 캐시에서 값을 조회합니다.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set은 캐시에 값을 저장합니다.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete는 캐시에서 값을 삭제합니다.
	Delete(ctx context.Context, key string) error

	// Exists는 캐시에 키가 존재하는지 확인합니다.
	Exists(ctx context.Context, key string) (bool, error)

	// GetOrSet은 캐시에서 값을 조회하거나, 없으면 함수를 실행하여 저장합니다.
	GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error)
}

// RoutingRepository는 라우팅 규칙 저장소를 담당하는 아웃바운드 포트입니다.
// 이 인터페이스는 서비스 레이어에서 사용되며, Database 어댑터에서 구현됩니다.
type RoutingRepository interface {
	// Create는 라우팅 규칙을 생성합니다.
	Create(ctx context.Context, rule *domain.RoutingRule) error

	// Update는 라우팅 규칙을 수정합니다.
	Update(ctx context.Context, rule *domain.RoutingRule) error

	// Delete는 라우팅 규칙을 삭제합니다.
	Delete(ctx context.Context, ruleID string) error

	// FindByID는 ID로 라우팅 규칙을 조회합니다.
	FindByID(ctx context.Context, ruleID string) (*domain.RoutingRule, error)

	// FindAll은 모든 라우팅 규칙을 조회합니다.
	FindAll(ctx context.Context) ([]*domain.RoutingRule, error)

	// FindMatchingRules는 요청에 매칭되는 라우팅 규칙들을 조회합니다.
	FindMatchingRules(ctx context.Context, request *domain.Request) ([]*domain.RoutingRule, error)
}

// EndpointRepository는 엔드포인트 저장소를 담당하는 아웃바운드 포트입니다.
// 이 인터페이스는 서비스 레이어에서 사용되며, Database 어댑터에서 구현됩니다.
type EndpointRepository interface {
	// Create는 엔드포인트를 생성합니다.
	Create(ctx context.Context, endpoint *domain.APIEndpoint) error

	// Update는 엔드포인트를 수정합니다.
	Update(ctx context.Context, endpoint *domain.APIEndpoint) error

	// Delete는 엔드포인트를 삭제합니다.
	Delete(ctx context.Context, endpointID string) error

	// FindByID는 ID로 엔드포인트를 조회합니다.
	FindByID(ctx context.Context, endpointID string) (*domain.APIEndpoint, error)

	// FindAll은 모든 엔드포인트를 조회합니다.
	FindAll(ctx context.Context) ([]*domain.APIEndpoint, error)

	// FindActive는 활성화된 엔드포인트만 조회합니다.
	FindActive(ctx context.Context) ([]*domain.APIEndpoint, error)

	// FindDefaultLegacyEndpoint는 기본 레거시 엔드포인트를 조회합니다.
	FindDefaultLegacyEndpoint(ctx context.Context) (*domain.APIEndpoint, error)

	// FindDefaultModernEndpoint는 기본 모던 엔드포인트를 조회합니다.
	FindDefaultModernEndpoint(ctx context.Context) (*domain.APIEndpoint, error)
}

// Logger는 로깅을 담당하는 아웃바운드 포트입니다.
// 이 인터페이스는 서비스 레이어에서 사용되며, Logger 패키지에서 구현됩니다.
type Logger interface {
	// Debug는 디버그 레벨 로그를 출력합니다.
	Debug(msg string, fields ...interface{})

	// Info는 정보 레벨 로그를 출력합니다.
	Info(msg string, fields ...interface{})

	// Warn은 경고 레벨 로그를 출력합니다.
	Warn(msg string, fields ...interface{})

	// Error는 에러 레벨 로그를 출력합니다.
	Error(msg string, fields ...interface{})

	// WithContext는 컨텍스트를 포함한 로거를 반환합니다.
	WithContext(ctx context.Context) Logger

	// WithFields는 필드를 포함한 로거를 반환합니다.
	WithFields(fields map[string]interface{}) Logger
}

// MetricsCollector는 메트릭 수집을 담당하는 아웃바운드 포트입니다.
// 이 인터페이스는 서비스 레이어에서 사용되며, Metrics 패키지에서 구현됩니다.
type MetricsCollector interface {
	// RecordRequest는 요청 메트릭을 기록합니다.
	RecordRequest(method, path string, statusCode int, duration time.Duration)

	// RecordExternalAPICall은 외부 API 호출 메트릭을 기록합니다.
	RecordExternalAPICall(endpoint string, success bool, duration time.Duration)

	// RecordCacheHit는 캐시 히트 메트릭을 기록합니다.
	RecordCacheHit(hit bool)

	// RecordDefaultRoutingUsed는 기본 라우팅 사용 메트릭을 기록합니다.
	RecordDefaultRoutingUsed(method, path string)

	// RecordDefaultOrchestrationUsed는 기본 오케스트레이션 사용 메트릭을 기록합니다.
	RecordDefaultOrchestrationUsed(method, path string)

	// IncrementCounter는 카운터를 증가시킵니다.
	IncrementCounter(name string, labels map[string]string)

	// RecordGauge는 게이지 값을 기록합니다.
	RecordGauge(name string, value float64, labels map[string]string)

	// RecordHistogram은 히스토그램 값을 기록합니다.
	RecordHistogram(name string, value float64, labels map[string]string)
}

// OrchestrationRepository는 오케스트레이션 규칙 저장소를 담당하는 아웃바운드 포트입니다.
type OrchestrationRepository interface {
	// Create는 오케스트레이션 규칙을 생성합니다.
	Create(ctx context.Context, rule *domain.OrchestrationRule) error

	// Update는 오케스트레이션 규칙을 수정합니다.
	Update(ctx context.Context, rule *domain.OrchestrationRule) error

	// Delete는 오케스트레이션 규칙을 삭제합니다.
	Delete(ctx context.Context, ruleID string) error

	// FindByID는 ID로 오케스트레이션 규칙을 조회합니다.
	FindByID(ctx context.Context, ruleID string) (*domain.OrchestrationRule, error)

	// FindByRoutingRuleID는 라우팅 규칙 ID로 오케스트레이션 규칙을 조회합니다.
	FindByRoutingRuleID(ctx context.Context, routingRuleID string) (*domain.OrchestrationRule, error)

	// FindAll은 모든 오케스트레이션 규칙을 조회합니다.
	FindAll(ctx context.Context) ([]*domain.OrchestrationRule, error)

	// FindActive는 활성화된 오케스트레이션 규칙만 조회합니다.
	FindActive(ctx context.Context) ([]*domain.OrchestrationRule, error)
}

// ComparisonRepository는 API 비교 결과 저장소를 담당하는 아웃바운드 포트입니다.
type ComparisonRepository interface {
	// SaveComparison은 API 비교 결과를 저장합니다.
	SaveComparison(ctx context.Context, comparison *domain.APIComparison) error

	// GetRecentComparisons는 최근 비교 결과들을 조회합니다.
	GetRecentComparisons(ctx context.Context, routingRuleID string, limit int) ([]*domain.APIComparison, error)

	// GetComparisonStatistics는 비교 통계를 조회합니다.
	GetComparisonStatistics(ctx context.Context, routingRuleID string, from, to time.Time) (*ComparisonStatistics, error)
}

// ComparisonStatistics는 비교 통계를 나타냅니다.
type ComparisonStatistics struct {
	RoutingRuleID     string    // 라우팅 규칙 ID
	TotalComparisons  int       // 총 비교 횟수
	SuccessfulMatches int       // 성공적인 일치 횟수
	AverageMatchRate  float64   // 평균 일치율
	LastComparison    time.Time // 마지막 비교 시점
}
