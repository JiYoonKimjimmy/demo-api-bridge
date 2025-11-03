package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type cbMockLogger struct{ mock.Mock }

func (m *cbMockLogger) WithContext(ctx context.Context) port.Logger {
	args := m.Called(ctx)
	if ret := args.Get(0); ret != nil {
		return ret.(port.Logger)
	}
	return m
}

func (m *cbMockLogger) WithFields(fields map[string]interface{}) port.Logger {
	args := m.Called(fields)
	if ret := args.Get(0); ret != nil {
		return ret.(port.Logger)
	}
	return m
}

func (m *cbMockLogger) Debug(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}
func (m *cbMockLogger) Info(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}
func (m *cbMockLogger) Warn(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}
func (m *cbMockLogger) Error(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}
func (m *cbMockLogger) Fatal(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}

type cbMockMetrics struct{ mock.Mock }

func (m *cbMockMetrics) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	m.Called(method, path, statusCode, duration)
}
func (m *cbMockMetrics) RecordExternalAPICall(url string, success bool, duration time.Duration) {
	m.Called(url, success, duration)
}
func (m *cbMockMetrics) RecordCacheHit(hit bool) { m.Called(hit) }
func (m *cbMockMetrics) RecordDefaultRoutingUsed(method, path string) {
	m.Called(method, path)
}
func (m *cbMockMetrics) RecordDefaultOrchestrationUsed(method, path string) {
	m.Called(method, path)
}
func (m *cbMockMetrics) IncrementCounter(name string, labels map[string]string) {
	m.Called(name, labels)
}
func (m *cbMockMetrics) RecordGauge(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}
func (m *cbMockMetrics) RecordHistogram(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

// --- Tests ---

func TestCircuitBreakerService_GetOrCreateBreaker_ReusesInstance(t *testing.T) {
	logger := &cbMockLogger{}
	metrics := &cbMockMetrics{}

	svc := NewCircuitBreakerService(logger, metrics)

	cfg := domain.NewCircuitBreakerConfig("test-breaker")

	// No specific expectations for logger/metrics on creation aside from Info logs
	logger.On("Info", "Circuit breaker created", "name", "test-breaker").Return().Once()

	b1 := svc.(*circuitBreakerService).GetOrCreateBreaker("test-breaker", cfg)
	// Second call should reuse without creation log
	b2 := svc.(*circuitBreakerService).GetOrCreateBreaker("test-breaker", cfg)

	assert.Same(t, b1, b2)
	logger.AssertExpectations(t)
}

func TestCircuitBreakerService_Execute_SuccessAndFailure(t *testing.T) {
	logger := &cbMockLogger{}
	metrics := &cbMockMetrics{}
	svc := NewCircuitBreakerService(logger, metrics)

	cfg := domain.NewCircuitBreakerConfig("exec-breaker")

	// Creation log only once
	logger.On("Info", "Circuit breaker created", "name", "exec-breaker").Return().Once()

	// Success case expectations
	metrics.On("RecordHistogram", "circuit_breaker_execution_duration", mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return().Once()
	logger.On("Debug", "Circuit breaker execution succeeded", "name", "exec-breaker", "duration_ms", mock.Anything).Return().Once()

	// Execute success
	res, err := svc.Execute(context.Background(), "exec-breaker", cfg, func() (interface{}, error) { return 123, nil })
	assert.NoError(t, err)
	assert.Equal(t, 123, res)

	// Failure case expectations
	metrics.On("RecordHistogram", "circuit_breaker_execution_duration", mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return().Once()
	logger.On("Warn", "Circuit breaker execution failed", "name", "exec-breaker", "error", mock.Anything, "duration_ms", mock.Anything).Return().Once()

	// Execute failure
	failErr := errors.New("boom")
	res, err = svc.Execute(context.Background(), "exec-breaker", cfg, func() (interface{}, error) { return nil, failErr })
	assert.Error(t, err)
	assert.Nil(t, res)

	logger.AssertExpectations(t)
	metrics.AssertExpectations(t)
}

func TestCircuitBreakerService_GetBreakerInfo_And_Reset(t *testing.T) {
	logger := &cbMockLogger{}
	metrics := &cbMockMetrics{}
	svc := NewCircuitBreakerService(logger, metrics)

	cfg := domain.NewCircuitBreakerConfig("info-breaker")

	// Creation log
	logger.On("Info", "Circuit breaker created", "name", "info-breaker").Return().Once()

	// Create once
	_ = svc.(*circuitBreakerService).GetOrCreateBreaker("info-breaker", cfg)

	// Query info
	info, err := svc.(*circuitBreakerService).GetBreakerInfo("info-breaker")
	assert.NoError(t, err)
	assert.Equal(t, "info-breaker", info.Name)

	// Reset expectations
	logger.On("Info", "Circuit breaker reset", "name", "info-breaker").Return().Once()
	metrics.On("IncrementCounter", "circuit_breaker_reset", mock.AnythingOfType("map[string]string")).Return().Once()

	err = svc.(*circuitBreakerService).ResetBreaker("info-breaker")
	assert.NoError(t, err)

	logger.AssertExpectations(t)
	metrics.AssertExpectations(t)
}

func TestCircuitBreakerService_GetBreakerInfo_NotFound(t *testing.T) {
	logger := &cbMockLogger{}
	metrics := &cbMockMetrics{}
	svc := NewCircuitBreakerService(logger, metrics)

	_, err := svc.(*circuitBreakerService).GetBreakerInfo("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCircuitBreakerService_StateChange_TriggersLoggingAndMetrics(t *testing.T) {
	logger := &cbMockLogger{}
	metrics := &cbMockMetrics{}
	svc := NewCircuitBreakerService(logger, metrics)

	// Configure to open on first failure and very small intervals
	cfg := domain.CircuitBreakerConfig{
		Name:        "state-breaker",
		MaxRequests: 1,
		Interval:    5 * time.Millisecond,
		Timeout:     5 * time.Millisecond,
		ReadyToTrip: func(c domain.Counts) bool { return c.ConsecutiveFailures >= 1 },
		OnStateChange: func(name string, from domain.CircuitBreakerState, to domain.CircuitBreakerState) {
			// no-op in test; service wraps with logs/metrics
		},
	}

	// Expect creation log
	logger.On("Info", "Circuit breaker created", "name", "state-breaker").Return().Once()

	// Expect execution histogram recording
	metrics.On("RecordHistogram", "circuit_breaker_execution_duration", mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return().Once()

	// Expect execution failure warning
	logger.On("Warn", "Circuit breaker execution failed", "name", "state-breaker", "error", mock.Anything, "duration_ms", mock.Anything).Return().Once()

	// Expect at least one state change Closed->Open due to failure
	metrics.On("IncrementCounter", "circuit_breaker_state_change", mock.AnythingOfType("map[string]string")).Return().Once()
	logger.On("Info", "Circuit breaker state changed",
		"name", "state-breaker",
		"from", domain.CLOSED,
		"to", domain.OPEN,
	).Return().Once()

	// Trigger failure to OPEN
	_, _ = svc.Execute(context.Background(), "state-breaker", cfg, func() (interface{}, error) { return nil, errors.New("fail") })

	// After timeout, breaker should be HALF_OPEN then CLOSED on success; at least one more state change occurs.
	// We set loose expectations: allow additional calls without exact matching by not asserting them explicitly here.

	logger.AssertExpectations(t)
	metrics.AssertExpectations(t)
}
