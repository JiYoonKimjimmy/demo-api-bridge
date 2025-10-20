package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for dependencies
type MockRoutingRepository struct {
	mock.Mock
}

func (m *MockRoutingRepository) FindMatchingRules(ctx context.Context, request *domain.Request) ([]*domain.RoutingRule, error) {
	args := m.Called(ctx, request)
	return args.Get(0).([]*domain.RoutingRule), args.Error(1)
}

func (m *MockRoutingRepository) FindByID(ctx context.Context, id string) (*domain.RoutingRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RoutingRule), args.Error(1)
}

func (m *MockRoutingRepository) Create(ctx context.Context, rule *domain.RoutingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockRoutingRepository) Update(ctx context.Context, rule *domain.RoutingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockRoutingRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoutingRepository) FindAll(ctx context.Context) ([]*domain.RoutingRule, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.RoutingRule), args.Error(1)
}

type MockEndpointRepository struct {
	mock.Mock
}

func (m *MockEndpointRepository) FindByID(ctx context.Context, id string) (*domain.APIEndpoint, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIEndpoint), args.Error(1)
}

func (m *MockEndpointRepository) Create(ctx context.Context, endpoint *domain.APIEndpoint) error {
	args := m.Called(ctx, endpoint)
	return args.Error(0)
}

func (m *MockEndpointRepository) Update(ctx context.Context, endpoint *domain.APIEndpoint) error {
	args := m.Called(ctx, endpoint)
	return args.Error(0)
}

func (m *MockEndpointRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockEndpointRepository) FindAll(ctx context.Context) ([]*domain.APIEndpoint, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.APIEndpoint), args.Error(1)
}

func (m *MockEndpointRepository) FindActive(ctx context.Context) ([]*domain.APIEndpoint, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.APIEndpoint), args.Error(1)
}

type MockOrchestrationRepository struct {
	mock.Mock
}

func (m *MockOrchestrationRepository) FindByRoutingRuleID(ctx context.Context, routingRuleID string) (*domain.OrchestrationRule, error) {
	args := m.Called(ctx, routingRuleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OrchestrationRule), args.Error(1)
}

func (m *MockOrchestrationRepository) Create(ctx context.Context, rule *domain.OrchestrationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockOrchestrationRepository) Update(ctx context.Context, rule *domain.OrchestrationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockOrchestrationRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrchestrationRepository) FindByID(ctx context.Context, id string) (*domain.OrchestrationRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OrchestrationRule), args.Error(1)
}

func (m *MockOrchestrationRepository) FindAll(ctx context.Context) ([]*domain.OrchestrationRule, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.OrchestrationRule), args.Error(1)
}

func (m *MockOrchestrationRepository) FindActive(ctx context.Context) ([]*domain.OrchestrationRule, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.OrchestrationRule), args.Error(1)
}

type MockComparisonRepository struct {
	mock.Mock
}

func (m *MockComparisonRepository) SaveComparison(ctx context.Context, comparison *domain.APIComparison) error {
	args := m.Called(ctx, comparison)
	return args.Error(0)
}

func (m *MockComparisonRepository) GetRecentComparisons(ctx context.Context, routingRuleID string, limit int) ([]*domain.APIComparison, error) {
	args := m.Called(ctx, routingRuleID, limit)
	return args.Get(0).([]*domain.APIComparison), args.Error(1)
}

func (m *MockComparisonRepository) GetComparisonStatistics(ctx context.Context, routingRuleID string, from, to time.Time) (*port.ComparisonStatistics, error) {
	args := m.Called(ctx, routingRuleID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.ComparisonStatistics), args.Error(1)
}

type MockOrchestrationService struct {
	mock.Mock
}

func (m *MockOrchestrationService) ProcessParallelRequest(ctx context.Context, request *domain.Request, legacyEndpoint, modernEndpoint *domain.APIEndpoint) (*domain.APIComparison, error) {
	args := m.Called(ctx, request, legacyEndpoint, modernEndpoint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIComparison), args.Error(1)
}

func (m *MockOrchestrationService) GetOrchestrationRule(ctx context.Context, routingRuleID string) (*domain.OrchestrationRule, error) {
	args := m.Called(ctx, routingRuleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OrchestrationRule), args.Error(1)
}

func (m *MockOrchestrationService) CreateOrchestrationRule(ctx context.Context, rule *domain.OrchestrationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockOrchestrationService) UpdateOrchestrationRule(ctx context.Context, rule *domain.OrchestrationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockOrchestrationService) EvaluateTransition(ctx context.Context, rule *domain.OrchestrationRule) (bool, error) {
	args := m.Called(ctx, rule)
	return args.Bool(0), args.Error(1)
}

func (m *MockOrchestrationService) ExecuteTransition(ctx context.Context, rule *domain.OrchestrationRule, newMode domain.APIMode) error {
	args := m.Called(ctx, rule, newMode)
	return args.Error(0)
}

type MockExternalAPIClient struct {
	mock.Mock
}

func (m *MockExternalAPIClient) SendRequest(ctx context.Context, endpoint *domain.APIEndpoint, request *domain.Request) (*domain.Response, error) {
	args := m.Called(ctx, endpoint, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Response), args.Error(1)
}

func (m *MockExternalAPIClient) SendWithRetry(ctx context.Context, endpoint *domain.APIEndpoint, request *domain.Request) (*domain.Response, error) {
	args := m.Called(ctx, endpoint, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Response), args.Error(1)
}

type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	args := m.Called(ctx, key, ttl, fn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) WithContext(ctx context.Context) port.Logger {
	args := m.Called(ctx)
	return args.Get(0).(port.Logger)
}

func (m *MockLogger) WithFields(fields map[string]interface{}) port.Logger {
	args := m.Called(fields)
	return args.Get(0).(port.Logger)
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}

func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	m.Called(append([]interface{}{msg}, fields...)...)
}

type MockMetricsCollector struct {
	mock.Mock
}

func (m *MockMetricsCollector) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	m.Called(method, path, statusCode, duration)
}

func (m *MockMetricsCollector) RecordExternalAPICall(url string, success bool, duration time.Duration) {
	m.Called(url, success, duration)
}

func (m *MockMetricsCollector) RecordCacheHit(hit bool) {
	m.Called(hit)
}

func (m *MockMetricsCollector) IncrementCounter(name string, labels map[string]string) {
	m.Called(name, labels)
}

func (m *MockMetricsCollector) RecordGauge(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func TestBridgeService_ProcessRequest_InvalidRequest(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockOrchestrationSvc := &MockOrchestrationService{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewBridgeService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockOrchestrationSvc,
		mockExternalAPI,
		mockCache,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	invalidRequest := &domain.Request{
		ID:     "", // Invalid: empty ID
		Method: "GET",
		Path:   "/api/users",
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "processing request", "request_id", "", "method", "GET", "path", "/api/users").Return()
	mockLogger.On("Error", "invalid request", "error", domain.ErrInvalidRequestID).Return()
	mockMetrics.On("RecordRequest", "GET", "/api/users", 400, mock.AnythingOfType("time.Duration")).Return()

	// When
	response, err := service.ProcessRequest(ctx, invalidRequest)

	// Then
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidRequestID, err)

	mockLogger.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestBridgeService_ProcessRequest_RouteNotFound(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockOrchestrationSvc := &MockOrchestrationService{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewBridgeService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockOrchestrationSvc,
		mockExternalAPI,
		mockCache,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "processing request", "request_id", "test-request-id", "method", "GET", "path", "/api/users").Return()
	mockRoutingRepo.On("FindMatchingRules", ctx, request).Return([]*domain.RoutingRule{}, nil)
	mockLogger.On("Error", "routing rule not found", "error", domain.ErrRouteNotFound).Return()
	mockMetrics.On("RecordRequest", "GET", "/api/users", 404, mock.AnythingOfType("time.Duration")).Return()

	// When
	response, err := service.ProcessRequest(ctx, request)

	// Then
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrRouteNotFound, err)

	mockLogger.AssertExpectations(t)
	mockRoutingRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestBridgeService_ProcessRequest_SingleAPIRequest_Success(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockOrchestrationSvc := &MockOrchestrationService{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewBridgeService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockOrchestrationSvc,
		mockExternalAPI,
		mockCache,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	routingRule := &domain.RoutingRule{
		ID:           "rule-1",
		EndpointID:   "endpoint-1",
		CacheEnabled: false,
		CacheTTL:     300,
	}

	endpoint := &domain.APIEndpoint{
		ID:       "endpoint-1",
		BaseURL:  "https://api.example.com",
		Path:     "/users",
		Method:   "GET",
		IsActive: true,
	}

	expectedResponse := &domain.Response{
		RequestID:  "test-request-id",
		StatusCode: 200,
		Body:       []byte(`{"users": []}`),
		Source:     "external",
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "processing request", "request_id", "test-request-id", "method", "GET", "path", "/api/users").Return()
	mockRoutingRepo.On("FindMatchingRules", ctx, request).Return([]*domain.RoutingRule{routingRule}, nil)
	mockOrchestrationRepo.On("FindByRoutingRuleID", ctx, "rule-1").Return(nil, errors.New("not found"))
	mockEndpointRepo.On("FindByID", ctx, "endpoint-1").Return(endpoint, nil)
	mockExternalAPI.On("SendWithRetry", ctx, endpoint, request).Return(expectedResponse, nil)
	mockMetrics.On("RecordExternalAPICall", "https://api.example.com/users", true, mock.AnythingOfType("time.Duration")).Return()
	mockMetrics.On("RecordRequest", "GET", "/api/users", 200, mock.AnythingOfType("time.Duration")).Return()
	mockLogger.On("Info", "single API request processed successfully", "request_id", "test-request-id", "status_code", 200, "duration_ms", mock.AnythingOfType("int64")).Return()

	// When
	response, err := service.ProcessRequest(ctx, request)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedResponse.RequestID, response.RequestID)
	assert.Equal(t, expectedResponse.StatusCode, response.StatusCode)
	assert.Equal(t, expectedResponse.Body, response.Body)
	assert.Equal(t, expectedResponse.Source, response.Source)

	mockLogger.AssertExpectations(t)
	mockRoutingRepo.AssertExpectations(t)
	mockOrchestrationRepo.AssertExpectations(t)
	mockEndpointRepo.AssertExpectations(t)
	mockExternalAPI.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestBridgeService_ProcessRequest_SingleAPIRequest_CacheHit(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockOrchestrationSvc := &MockOrchestrationService{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewBridgeService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockOrchestrationSvc,
		mockExternalAPI,
		mockCache,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	routingRule := &domain.RoutingRule{
		ID:           "rule-1",
		EndpointID:   "endpoint-1",
		CacheEnabled: true,
		CacheTTL:     300,
	}

	cachedData := []byte(`{"users": [{"id": 1, "name": "John"}]}`)

	endpoint := &domain.APIEndpoint{
		ID:       "endpoint-1",
		BaseURL:  "https://api.example.com",
		Path:     "/users",
		Method:   "GET",
		IsActive: true,
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "processing request", "request_id", "test-request-id", "method", "GET", "path", "/api/users").Return()
	mockRoutingRepo.On("FindMatchingRules", ctx, request).Return([]*domain.RoutingRule{routingRule}, nil)
	mockOrchestrationRepo.On("FindByRoutingRuleID", ctx, "rule-1").Return(nil, errors.New("not found"))
	mockEndpointRepo.On("FindByID", ctx, "endpoint-1").Return(endpoint, nil)
	mockCache.On("Get", ctx, "api_bridge:GET:/api/users").Return(cachedData, nil)
	mockLogger.On("Info", "cache hit", "key", "api_bridge:GET:/api/users").Return()
	mockMetrics.On("RecordCacheHit", true).Return()
	mockMetrics.On("RecordRequest", "GET", "/api/users", 200, mock.AnythingOfType("time.Duration")).Return()

	// When
	response, err := service.ProcessRequest(ctx, request)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "test-request-id", response.RequestID)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, cachedData, response.Body)
	assert.Equal(t, "cache", response.Source)

	mockLogger.AssertExpectations(t)
	mockRoutingRepo.AssertExpectations(t)
	mockOrchestrationRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestBridgeService_ProcessRequest_ParallelRequest_Success(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockOrchestrationSvc := &MockOrchestrationService{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewBridgeService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockOrchestrationSvc,
		mockExternalAPI,
		mockCache,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	routingRule := &domain.RoutingRule{
		ID:           "rule-1",
		EndpointID:   "endpoint-1",
		CacheEnabled: false,
		CacheTTL:     300,
	}

	orchestrationRule := &domain.OrchestrationRule{
		ID:               "orch-1",
		RoutingRuleID:    "rule-1",
		LegacyEndpointID: "legacy-endpoint-1",
		ModernEndpointID: "modern-endpoint-1",
		CurrentMode:      domain.PARALLEL,
		ComparisonConfig: domain.ComparisonConfig{SaveComparisonHistory: true},
		TransitionConfig: domain.TransitionConfig{},
	}

	legacyEndpoint := &domain.APIEndpoint{
		ID:       "legacy-endpoint-1",
		BaseURL:  "https://legacy-api.example.com",
		Path:     "/users",
		IsActive: true,
	}

	modernEndpoint := &domain.APIEndpoint{
		ID:       "modern-endpoint-1",
		BaseURL:  "https://modern-api.example.com",
		Path:     "/users",
		IsActive: true,
	}

	comparison := &domain.APIComparison{
		RequestID:     "test-request-id",
		RoutingRuleID: "rule-1",
		MatchRate:     0.95,
		LegacyResponse: &domain.Response{
			RequestID:  "test-request-id",
			StatusCode: 200,
			Body:       []byte(`{"users": [{"id": 1}]}`),
		},
		ModernResponse: &domain.Response{
			RequestID:  "test-request-id",
			StatusCode: 200,
			Body:       []byte(`{"users": [{"id": 1}]}`),
		},
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "processing request", "request_id", "test-request-id", "method", "GET", "path", "/api/users").Return()
	mockRoutingRepo.On("FindMatchingRules", ctx, request).Return([]*domain.RoutingRule{routingRule}, nil)
	mockOrchestrationRepo.On("FindByRoutingRuleID", ctx, "rule-1").Return(orchestrationRule, nil)
	mockLogger.On("Info", "processing orchestrated request", "request_id", "test-request-id", "current_mode", domain.PARALLEL).Return()
	mockEndpointRepo.On("FindByID", ctx, "legacy-endpoint-1").Return(legacyEndpoint, nil)
	mockEndpointRepo.On("FindByID", ctx, "modern-endpoint-1").Return(modernEndpoint, nil)
	mockOrchestrationSvc.On("ProcessParallelRequest", ctx, request, legacyEndpoint, modernEndpoint).Return(comparison, nil)
	mockComparisonRepo.On("SaveComparison", ctx, comparison).Return(nil)
	mockMetrics.On("RecordRequest", "GET", "/api/users", 200, mock.AnythingOfType("time.Duration")).Return()
	mockLogger.On("Info", "parallel request processed successfully", "request_id", "test-request-id", "match_rate", 0.95, "differences_count", 0, "returned_source", "legacy").Return()
	mockOrchestrationSvc.On("EvaluateTransition", mock.Anything, mock.Anything).Return(false, nil)

	// When
	response, err := service.ProcessRequest(ctx, request)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "test-request-id", response.RequestID)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "legacy", response.Source)

	mockLogger.AssertExpectations(t)
	mockRoutingRepo.AssertExpectations(t)
	mockOrchestrationRepo.AssertExpectations(t)
	mockEndpointRepo.AssertExpectations(t)
	mockOrchestrationSvc.AssertExpectations(t)
	mockComparisonRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestBridgeService_GetRoutingRule_SelectsHighestPriority(t *testing.T) {
	// Given
	mockRoutingRepo := &MockRoutingRepository{}
	mockEndpointRepo := &MockEndpointRepository{}
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockOrchestrationSvc := &MockOrchestrationService{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewBridgeService(
		mockRoutingRepo,
		mockEndpointRepo,
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockOrchestrationSvc,
		mockExternalAPI,
		mockCache,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	// Multiple rules with different priorities
	rules := []*domain.RoutingRule{
		{ID: "rule-1", Priority: 10},
		{ID: "rule-2", Priority: 5}, // Highest priority (lowest number)
		{ID: "rule-3", Priority: 15},
	}

	mockRoutingRepo.On("FindMatchingRules", ctx, request).Return(rules, nil)

	// When
	selectedRule, err := service.GetRoutingRule(ctx, request)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, selectedRule)
	assert.Equal(t, "rule-2", selectedRule.ID)
	assert.Equal(t, 5, selectedRule.Priority)

	mockRoutingRepo.AssertExpectations(t)
}
