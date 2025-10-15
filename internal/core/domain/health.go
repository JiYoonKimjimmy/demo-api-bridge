package domain

import "time"

// HealthStatus는 서비스의 헬스 상태를 나타냅니다.
type HealthStatus string

const (
	HEALTHY   HealthStatus = "healthy"
	UNHEALTHY HealthStatus = "unhealthy"
	UNKNOWN   HealthStatus = "unknown"
)

// ReadinessStatus는 서비스의 준비 상태를 나타냅니다.
type ReadinessStatus string

const (
	READY         ReadinessStatus = "ready"
	NOT_READY     ReadinessStatus = "not_ready"
	UNKNOWN_READY ReadinessStatus = "unknown"
)

// ServiceStatus는 서비스의 전체 상태 정보를 나타냅니다.
type ServiceStatus struct {
	ServiceName     string                 `json:"service_name"`
	Version         string                 `json:"version"`
	HealthStatus    HealthStatus           `json:"health_status"`
	ReadinessStatus ReadinessStatus        `json:"readiness_status"`
	Uptime          time.Duration          `json:"uptime"`
	LastChecked     time.Time              `json:"last_checked"`
	Timestamp       time.Time              `json:"timestamp"`
	Environment     string                 `json:"environment"`
	Metrics         map[string]interface{} `json:"metrics"`
	Details         map[string]interface{} `json:"details,omitempty"`
}

// HealthCheck는 개별 헬스 체크 결과를 나타냅니다.
type HealthCheck struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	LastCheck time.Time     `json:"last_check"`
}

// ReadinessCheck는 개별 준비 상태 체크 결과를 나타냅니다.
type ReadinessCheck struct {
	Name      string          `json:"name"`
	Status    ReadinessStatus `json:"status"`
	Message   string          `json:"message,omitempty"`
	Duration  time.Duration   `json:"duration"`
	LastCheck time.Time       `json:"last_check"`
}
