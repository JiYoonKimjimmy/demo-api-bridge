package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

// circuitBreakerService는 Circuit Breaker 패턴을 구현하는 서비스입니다.
//
// Circuit Breaker는 장애가 발생한 외부 시스템에 대한 호출을 차단하여
// 시스템 전체의 안정성을 보호합니다. Sony의 gobreaker 라이브러리를 사용하여
// 다음 세 가지 상태를 관리합니다:
//
// States:
//   - Closed (정상): 모든 요청이 통과, 에러 발생 시 카운트 증가
//   - Open (차단): 모든 요청이 즉시 실패, Timeout 후 Half-Open으로 전환
//   - Half-Open (반개방): 제한된 수의 요청만 통과, 성공 시 Closed로 복구
//
// 장애 격리 메커니즘:
//   - 연속 실패 임계값 초과 시 자동으로 Circuit Open
//   - 일정 시간(Timeout) 후 자동으로 Half-Open 상태로 복구 시도
//   - Half-Open에서 성공하면 Closed로 복구, 실패하면 다시 Open
//
// 성능 특성:
//   - Circuit Open 상태에서는 즉시 에러 반환 (지연시간 0ms)
//   - 상태 확인은 Thread-Safe한 RWMutex 사용
type circuitBreakerService struct {
	breakers map[string]*gobreaker.CircuitBreaker // 이름별 Circuit Breaker 맵
	mutex    sync.RWMutex                         // Thread-Safe 접근을 위한 뮤텍스
	logger   port.Logger                          // 로거
	metrics  port.MetricsCollector                // 메트릭 수집기
}

// NewCircuitBreakerService는 새로운 Circuit Breaker 서비스를 생성합니다.
func NewCircuitBreakerService(logger port.Logger, metrics port.MetricsCollector) port.CircuitBreakerService {
	return &circuitBreakerService{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		logger:   logger,
		metrics:  metrics,
	}
}

// GetOrCreateBreaker는 이름에 해당하는 Circuit Breaker를 가져오거나 생성합니다.
func (s *circuitBreakerService) GetOrCreateBreaker(name string, config domain.CircuitBreakerConfig) *gobreaker.CircuitBreaker {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if breaker, exists := s.breakers[name]; exists {
		return breaker
	}

	// gobreaker 설정 변환
	gobreakerConfig := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// domain.Counts로 변환
			domainCounts := domain.Counts{
				Requests:             counts.Requests,
				TotalSuccesses:       counts.TotalSuccesses,
				TotalFailures:        counts.TotalFailures,
				ConsecutiveSuccesses: counts.ConsecutiveSuccesses,
				ConsecutiveFailures:  counts.ConsecutiveFailures,
			}
			return config.ReadyToTrip(domainCounts)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// domain.CircuitBreakerState로 변환
			var fromState, toState domain.CircuitBreakerState
			switch from {
			case gobreaker.StateClosed:
				fromState = domain.CLOSED
			case gobreaker.StateOpen:
				fromState = domain.OPEN
			case gobreaker.StateHalfOpen:
				fromState = domain.HALF_OPEN
			}
			switch to {
			case gobreaker.StateClosed:
				toState = domain.CLOSED
			case gobreaker.StateOpen:
				toState = domain.OPEN
			case gobreaker.StateHalfOpen:
				toState = domain.HALF_OPEN
			}

			config.OnStateChange(name, fromState, toState)

			// 메트릭 기록
			s.metrics.IncrementCounter("circuit_breaker_state_change", map[string]string{
				"name": name,
				"from": string(fromState),
				"to":   string(toState),
			})

			s.logger.Info("Circuit breaker state changed",
				"name", name,
				"from", fromState,
				"to", toState,
			)
		},
	}

	breaker := gobreaker.NewCircuitBreaker(gobreakerConfig)
	s.breakers[name] = breaker

	s.logger.Info("Circuit breaker created", "name", name)
	return breaker
}

// Execute는 Circuit Breaker를 통해 함수를 실행합니다.
func (s *circuitBreakerService) Execute(ctx context.Context, breakerName string, config domain.CircuitBreakerConfig, fn func() (interface{}, error)) (interface{}, error) {
	breaker := s.GetOrCreateBreaker(breakerName, config)

	start := time.Now()
	result, err := breaker.Execute(fn)
	duration := time.Since(start)

	// 메트릭 기록
	s.metrics.RecordHistogram("circuit_breaker_execution_duration", float64(duration.Milliseconds()), map[string]string{
		"name":   breakerName,
		"result": s.getResultLabel(err),
	})

	if err != nil {
		s.logger.Warn("Circuit breaker execution failed",
			"name", breakerName,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
	} else {
		s.logger.Debug("Circuit breaker execution succeeded",
			"name", breakerName,
			"duration_ms", duration.Milliseconds(),
		)
	}

	return result, err
}

// GetBreakerInfo는 Circuit Breaker의 현재 상태 정보를 반환합니다.
func (s *circuitBreakerService) GetBreakerInfo(breakerName string) (*domain.CircuitBreakerInfo, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	breaker, exists := s.breakers[breakerName]
	if !exists {
		return nil, fmt.Errorf("circuit breaker '%s' not found", breakerName)
	}

	state := breaker.State()
	counts := breaker.Counts()

	// gobreaker.State를 domain.CircuitBreakerState로 변환
	var domainState domain.CircuitBreakerState
	switch state {
	case gobreaker.StateClosed:
		domainState = domain.CLOSED
	case gobreaker.StateOpen:
		domainState = domain.OPEN
	case gobreaker.StateHalfOpen:
		domainState = domain.HALF_OPEN
	}

	// gobreaker.Counts를 domain.Counts로 변환
	domainCounts := domain.Counts{
		Requests:             counts.Requests,
		TotalSuccesses:       counts.TotalSuccesses,
		TotalFailures:        counts.TotalFailures,
		ConsecutiveSuccesses: counts.ConsecutiveSuccesses,
		ConsecutiveFailures:  counts.ConsecutiveFailures,
	}

	return &domain.CircuitBreakerInfo{
		Name:        breakerName,
		State:       domainState,
		Counts:      domainCounts,
		MaxRequests: 3,                                // 기본값
		Expiry:      time.Now().Add(10 * time.Second), // 임시 값
	}, nil
}

// GetAllBreakerInfos는 모든 Circuit Breaker의 상태 정보를 반환합니다.
func (s *circuitBreakerService) GetAllBreakerInfos() map[string]*domain.CircuitBreakerInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	infos := make(map[string]*domain.CircuitBreakerInfo)
	for name, breaker := range s.breakers {
		state := breaker.State()
		counts := breaker.Counts()

		var domainState domain.CircuitBreakerState
		switch state {
		case gobreaker.StateClosed:
			domainState = domain.CLOSED
		case gobreaker.StateOpen:
			domainState = domain.OPEN
		case gobreaker.StateHalfOpen:
			domainState = domain.HALF_OPEN
		}

		domainCounts := domain.Counts{
			Requests:             counts.Requests,
			TotalSuccesses:       counts.TotalSuccesses,
			TotalFailures:        counts.TotalFailures,
			ConsecutiveSuccesses: counts.ConsecutiveSuccesses,
			ConsecutiveFailures:  counts.ConsecutiveFailures,
		}

		infos[name] = &domain.CircuitBreakerInfo{
			Name:        name,
			State:       domainState,
			Counts:      domainCounts,
			MaxRequests: 3,                                // 기본값
			Expiry:      time.Now().Add(10 * time.Second), // 임시 값
		}
	}

	return infos
}

// ResetBreaker는 Circuit Breaker를 리셋합니다.
func (s *circuitBreakerService) ResetBreaker(breakerName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, exists := s.breakers[breakerName]
	if !exists {
		return fmt.Errorf("circuit breaker '%s' not found", breakerName)
	}

	// Circuit Breaker를 새로 생성하여 리셋 효과
	config := domain.NewCircuitBreakerConfig(breakerName)
	gobreakerConfig := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return config.ReadyToTrip(domain.Counts{
				Requests:             counts.Requests,
				TotalSuccesses:       counts.TotalSuccesses,
				TotalFailures:        counts.TotalFailures,
				ConsecutiveSuccesses: counts.ConsecutiveSuccesses,
				ConsecutiveFailures:  counts.ConsecutiveFailures,
			})
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			config.OnStateChange(name, domain.CircuitBreakerState(from.String()), domain.CircuitBreakerState(to.String()))
		},
	}

	s.breakers[breakerName] = gobreaker.NewCircuitBreaker(gobreakerConfig)
	s.logger.Info("Circuit breaker reset", "name", breakerName)

	s.metrics.IncrementCounter("circuit_breaker_reset", map[string]string{
		"name": breakerName,
	})

	return nil
}

// getResultLabel은 실행 결과에 따른 라벨을 반환합니다.
func (s *circuitBreakerService) getResultLabel(err error) string {
	if err != nil {
		return "failure"
	}
	return "success"
}
