package service

import (
	"context"
	"demo-api-bridge/internal/core/port"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthService_CheckHealth_Success(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}

	service := NewHealthCheckService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockCache,
		mockLogger,
	)

	ctx := context.Background()

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Debug", "performing health check").Return()

	// When
	err := service.CheckHealth(ctx)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
}

func TestHealthService_CheckReadiness_Success(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}

	service := NewHealthCheckService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockCache,
		mockLogger,
	)

	ctx := context.Background()

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Debug", "performing readiness check").Return()
	mockCache.On("Exists", ctx, "health_check").Return(true, nil)

	// When
	err := service.CheckReadiness(ctx)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestHealthService_CheckReadiness_CacheError(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}

	service := NewHealthCheckService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockCache,
		mockLogger,
	)

	ctx := context.Background()
	cacheError := errors.New("cache connection failed")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Debug", "performing readiness check").Return()
	mockCache.On("Exists", ctx, "health_check").Return(false, cacheError)
	mockLogger.On("Warn", "cache connection check failed", "error", cacheError).Return()

	// When
	err := service.CheckReadiness(ctx)

	// Then
	assert.NoError(t, err) // 캐시 에러는 치명적이지 않음

	mockLogger.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestHealthService_GetServiceStatus_Success(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}

	service := NewHealthCheckService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockCache,
		mockLogger,
	)

	ctx := context.Background()

	// When
	status := service.GetServiceStatus(ctx)

	// Then
	assert.NotNil(t, status)
	assert.Equal(t, "api-bridge", status["service"])
	assert.Equal(t, "0.1.0", status["version"])
	assert.Contains(t, status, "uptime")
	assert.Contains(t, status, "timestamp")

	// uptime이 올바른 형식인지 확인
	uptime, ok := status["uptime"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, uptime)

	// timestamp가 올바른 형식인지 확인
	timestamp, ok := status["timestamp"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, timestamp)

	// RFC3339 형식인지 확인
	_, err := time.Parse(time.RFC3339, timestamp)
	assert.NoError(t, err)
}

func TestHealthService_GetServiceStatus_ContainsExpectedFields(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}

	service := NewHealthCheckService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockCache,
		mockLogger,
	)

	ctx := context.Background()

	// When
	status := service.GetServiceStatus(ctx)

	// Then
	expectedFields := []string{"service", "version", "uptime", "timestamp"}
	for _, field := range expectedFields {
		assert.Contains(t, status, field, "Status should contain field: %s", field)
		assert.NotEmpty(t, status[field], "Field %s should not be empty", field)
	}
}

func TestHealthService_NewHealthCheckService_CreatesValidInstance(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}

	// When
	service := NewHealthCheckService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockCache,
		mockLogger,
	)

	// Then
	assert.NotNil(t, service)

	// 서비스가 올바른 인터페이스를 구현하는지 확인
	_, ok := service.(port.HealthCheckService)
	assert.True(t, ok)
}
