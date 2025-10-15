package domain

import (
	"time"
)

// CircuitBreakerState는 Circuit Breaker의 상태를 나타냅니다.
type CircuitBreakerState string

const (
	// CLOSED: 정상 상태, 요청 허용
	CLOSED CircuitBreakerState = "CLOSED"
	// OPEN: 장애 상태, 요청 차단
	OPEN CircuitBreakerState = "OPEN"
	// HALF_OPEN: 복구 시도 상태, 제한적 요청 허용
	HALF_OPEN CircuitBreakerState = "HALF_OPEN"
)

// CircuitBreakerConfig는 Circuit Breaker 설정을 나타냅니다.
type CircuitBreakerConfig struct {
	Name          string                                                              // Circuit Breaker 이름
	MaxRequests   uint32                                                              // HALF_OPEN 상태에서 허용할 최대 요청 수
	Interval      time.Duration                                                       // 상태 변경을 확인하는 간격
	Timeout       time.Duration                                                       // OPEN 상태에서 HALF_OPEN으로 전환하기까지의 대기 시간
	ReadyToTrip   func(counts Counts) bool                                            // OPEN 상태로 전환할 조건
	OnStateChange func(name string, from CircuitBreakerState, to CircuitBreakerState) // 상태 변경 콜백
}

// Counts는 Circuit Breaker의 카운터를 나타냅니다.
type Counts struct {
	Requests             uint32 // 총 요청 수
	TotalSuccesses       uint32 // 성공한 요청 수
	TotalFailures        uint32 // 실패한 요청 수
	ConsecutiveSuccesses uint32 // 연속 성공 수
	ConsecutiveFailures  uint32 // 연속 실패 수
}

// GetRequests는 현재 간격 내의 요청 수를 반환합니다.
func (c Counts) GetRequests() uint32 {
	return c.Requests
}

// TotalRequests는 총 요청 수를 반환합니다.
func (c Counts) TotalRequests() uint32 {
	return c.TotalSuccesses + c.TotalFailures
}

// IsSuccessful은 성공률을 기반으로 성공 여부를 판단합니다.
func (c Counts) IsSuccessful() bool {
	if c.TotalRequests() == 0 {
		return true
	}
	return float64(c.TotalSuccesses)/float64(c.TotalRequests()) >= 0.5 // 50% 이상 성공 시 성공으로 간주
}

// NewCircuitBreakerConfig는 기본 Circuit Breaker 설정을 생성합니다.
func NewCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			// 연속 5회 실패 시 OPEN 상태로 전환
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from CircuitBreakerState, to CircuitBreakerState) {
			// 상태 변경 로깅 (기본 구현)
		},
	}
}

// CircuitBreakerInfo는 Circuit Breaker의 현재 상태 정보를 나타냅니다.
type CircuitBreakerInfo struct {
	Name        string              `json:"name"`
	State       CircuitBreakerState `json:"state"`
	Counts      Counts              `json:"counts"`
	MaxRequests uint32              `json:"max_requests"`
	Expiry      time.Time           `json:"expiry"`
	LastError   error               `json:"last_error,omitempty"`
}

// IsHealthy는 Circuit Breaker가 건강한 상태인지 확인합니다.
func (c CircuitBreakerInfo) IsHealthy() bool {
	return c.State == CLOSED || c.State == HALF_OPEN
}

// CanExecute는 요청 실행이 가능한지 확인합니다.
func (c CircuitBreakerInfo) CanExecute() bool {
	switch c.State {
	case CLOSED:
		return true
	case OPEN:
		return false
	case HALF_OPEN:
		return c.Counts.GetRequests() < c.MaxRequests
	default:
		return false
	}
}

// MaxRequests는 HALF_OPEN 상태에서 허용할 최대 요청 수입니다.
// 실제 구현에서는 config에서 가져와야 하지만, 여기서는 간단히 상수로 정의
const defaultMaxRequests = 3
