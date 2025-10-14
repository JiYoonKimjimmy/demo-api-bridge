package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock services for testing
type MockBridgeService struct {
	mock.Mock
}

func (m *MockBridgeService) ProcessRequest(ctx context.Context, req *domain.Request) (*domain.Response, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*domain.Response), args.Error(1)
}

type MockRoutingService struct {
	mock.Mock
}

func (m *MockRoutingService) GetAllRoutingRules(ctx context.Context) ([]*domain.RoutingRule, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.RoutingRule), args.Error(1)
}

func (m *MockRoutingService) GetRoutingRuleByID(ctx context.Context, id int64) (*domain.RoutingRule, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.RoutingRule), args.Error(1)
}

func (m *MockRoutingService) CreateRoutingRule(ctx context.Context, rule *domain.RoutingRule) (*domain.RoutingRule, error) {
	args := m.Called(ctx, rule)
	return args.Get(0).(*domain.RoutingRule), args.Error(1)
}

func (m *MockRoutingService) UpdateRoutingRule(ctx context.Context, rule *domain.RoutingRule) (*domain.RoutingRule, error) {
	args := m.Called(ctx, rule)
	return args.Get(0).(*domain.RoutingRule), args.Error(1)
}

func (m *MockRoutingService) DeleteRoutingRule(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoutingService) FindMatchingRule(ctx context.Context, method, path string) (*domain.RoutingRule, error) {
	args := m.Called(ctx, method, path)
	return args.Get(0).(*domain.RoutingRule), args.Error(1)
}

type MockEndpointService struct {
	mock.Mock
}

func (m *MockEndpointService) GetAllEndpoints(ctx context.Context) ([]*domain.Endpoint, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.Endpoint), args.Error(1)
}

func (m *MockEndpointService) GetEndpointByID(ctx context.Context, id int64) (*domain.Endpoint, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Endpoint), args.Error(1)
}

func (m *MockEndpointService) CreateEndpoint(ctx context.Context, endpoint *domain.Endpoint) (*domain.Endpoint, error) {
	args := m.Called(ctx, endpoint)
	return args.Get(0).(*domain.Endpoint), args.Error(1)
}

func (m *MockEndpointService) UpdateEndpoint(ctx context.Context, endpoint *domain.Endpoint) (*domain.Endpoint, error) {
	args := m.Called(ctx, endpoint)
	return args.Get(0).(*domain.Endpoint), args.Error(1)
}

func (m *MockEndpointService) DeleteEndpoint(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockHealthService struct {
	mock.Mock
}

func (m *MockHealthService) CheckHealth(ctx context.Context) (*domain.HealthStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*domain.HealthStatus), args.Error(1)
}

func (m *MockHealthService) CheckReadiness(ctx context.Context) (*domain.ReadinessStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*domain.ReadinessStatus), args.Error(1)
}

func (m *MockHealthService) GetStatus(ctx context.Context) (*domain.ServiceStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*domain.ServiceStatus), args.Error(1)
}

func setupTestHandler() (*Handler, *MockBridgeService, *MockRoutingService, *MockEndpointService, *MockHealthService, *gin.Engine) {
	// Create mock services
	mockBridge := &MockBridgeService{}
	mockRouting := &MockRoutingService{}
	mockEndpoint := &MockEndpointService{}
	mockHealth := &MockHealthService{}

	// Create logger and metrics
	testLogger := logger.NewLogger("test", "info")
	testMetrics := metrics.NewMetrics()

	// Create handler
	handler := NewHandler(
		mockBridge,
		mockRouting,
		mockEndpoint,
		mockHealth,
		testLogger,
		testMetrics,
	)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router)

	return handler, mockBridge, mockRouting, mockEndpoint, mockHealth, router
}

func TestHealthCheck(t *testing.T) {
	_, _, _, _, mockHealth, router := setupTestHandler()

	// Mock health check response
	expectedStatus := &domain.HealthStatus{
		Status:      "healthy",
		ServiceName: "api-bridge",
		Version:     "0.1.0",
		Timestamp:   time.Now(),
		Uptime:      "1h30m",
	}

	mockHealth.On("CheckHealth", mock.Anything).Return(expectedStatus, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "api-bridge", response.Service)
	assert.Equal(t, "0.1.0", response.Version)

	mockHealth.AssertExpectations(t)
}

func TestReadinessCheck(t *testing.T) {
	_, _, _, _, mockHealth, router := setupTestHandler()

	// Mock readiness check response
	expectedReadiness := &domain.ReadinessStatus{
		Status:    "ready",
		Ready:     true,
		Checks:    map[string]string{"database": "ok", "cache": "ok"},
		Timestamp: time.Now(),
	}

	mockHealth.On("CheckReadiness", mock.Anything).Return(expectedReadiness, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ready", response["status"])
	assert.Equal(t, true, response["ready"])

	mockHealth.AssertExpectations(t)
}

func TestStatus(t *testing.T) {
	_, _, _, _, mockHealth, router := setupTestHandler()

	// Mock status response
	expectedStatus := &domain.ServiceStatus{
		ServiceName: "api-bridge",
		Version:     "0.1.0",
		Timestamp:   time.Now(),
		Uptime:      "1h30m",
		Environment: "test",
		Metrics:     map[string]interface{}{"requests": 100},
	}

	mockHealth.On("GetStatus", mock.Anything).Return(expectedStatus, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response StatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "api-bridge", response.Service)
	assert.Equal(t, "0.1.0", response.Version)

	mockHealth.AssertExpectations(t)
}

func TestHandleAPIRequest(t *testing.T) {
	_, mockBridge, _, _, _, router := setupTestHandler()

	// Mock bridge service response
	expectedResponse := &domain.Response{
		StatusCode:  http.StatusOK,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Body:        []byte(`{"message": "success"}`),
		ContentType: "application/json",
	}

	mockBridge.On("ProcessRequest", mock.Anything, mock.AnythingOfType("*domain.Request")).Return(expectedResponse, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "success")

	mockBridge.AssertExpectations(t)
}

func TestListEndpoints(t *testing.T) {
	_, _, _, mockEndpoint, _, router := setupTestHandler()

	// Mock endpoints response
	expectedEndpoints := []*domain.Endpoint{
		{
			ID:          1,
			Name:        "test-endpoint",
			Description: "Test endpoint",
			BaseURL:     "http://test.com",
			IsActive:    true,
		},
	}

	mockEndpoint.On("GetAllEndpoints", mock.Anything).Return(expectedEndpoints, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/endpoints", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["endpoints"])
	assert.Equal(t, float64(1), response["count"])

	mockEndpoint.AssertExpectations(t)
}

func TestCreateEndpoint(t *testing.T) {
	_, _, _, mockEndpoint, _, router := setupTestHandler()

	// Mock create endpoint response
	expectedEndpoint := &domain.Endpoint{
		ID:          1,
		Name:        "new-endpoint",
		Description: "New endpoint",
		BaseURL:     "http://new.com",
		IsActive:    true,
	}

	mockEndpoint.On("CreateEndpoint", mock.Anything, mock.AnythingOfType("*domain.Endpoint")).Return(expectedEndpoint, nil)

	// Create request body
	requestBody := CreateEndpointRequest{
		Name:        "new-endpoint",
		Description: "New endpoint",
		BaseURL:     "http://new.com",
		IsActive:    true,
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/endpoints", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["endpoint"])

	mockEndpoint.AssertExpectations(t)
}

func TestListRoutingRules(t *testing.T) {
	_, _, mockRouting, _, _, router := setupTestHandler()

	// Mock routing rules response
	expectedRules := []*domain.RoutingRule{
		{
			ID:          1,
			Name:        "test-rule",
			PathPattern: "/test/*",
			Method:      "GET",
			Priority:    1,
			IsActive:    true,
		},
	}

	mockRouting.On("GetAllRoutingRules", mock.Anything).Return(expectedRules, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/routing-rules", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["routing_rules"])
	assert.Equal(t, float64(1), response["count"])

	mockRouting.AssertExpectations(t)
}

func TestCreateRoutingRule(t *testing.T) {
	_, _, mockRouting, _, _, router := setupTestHandler()

	// Mock create routing rule response
	expectedRule := &domain.RoutingRule{
		ID:          1,
		Name:        "new-rule",
		PathPattern: "/new/*",
		Method:      "GET",
		Priority:    1,
		IsActive:    true,
	}

	mockRouting.On("CreateRoutingRule", mock.Anything, mock.AnythingOfType("*domain.RoutingRule")).Return(expectedRule, nil)

	// Create request body
	requestBody := CreateRoutingRuleRequest{
		Name:        "new-rule",
		PathPattern: "/new/*",
		Method:      "GET",
		Priority:    1,
		IsActive:    true,
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/routing-rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["routing_rule"])

	mockRouting.AssertExpectations(t)
}

func TestHealthCheckFailure(t *testing.T) {
	_, _, _, _, mockHealth, router := setupTestHandler()

	// Mock health check failure
	mockHealth.On("CheckHealth", mock.Anything).Return((*domain.HealthStatus)(nil), assert.AnError)

	// Create request
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", response["status"])
	assert.Equal(t, "api-bridge", response["service"])

	mockHealth.AssertExpectations(t)
}

func TestHandleAPIRequestError(t *testing.T) {
	_, mockBridge, _, _, _, router := setupTestHandler()

	// Mock bridge service error
	mockBridge.On("ProcessRequest", mock.Anything, mock.AnythingOfType("*domain.Request")).Return((*domain.Response)(nil), assert.AnError)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Error)
	assert.NotEmpty(t, response.Message)

	mockBridge.AssertExpectations(t)
}

// Benchmark tests
func BenchmarkHandleAPIRequest(b *testing.B) {
	_, mockBridge, _, _, _, router := setupTestHandler()

	// Mock bridge service response
	expectedResponse := &domain.Response{
		StatusCode:  http.StatusOK,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Body:        []byte(`{"message": "success"}`),
		ContentType: "application/json",
	}

	mockBridge.On("ProcessRequest", mock.Anything, mock.AnythingOfType("*domain.Request")).Return(expectedResponse, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	_, _, _, _, mockHealth, router := setupTestHandler()

	// Mock health check response
	expectedStatus := &domain.HealthStatus{
		Status:      "healthy",
		ServiceName: "api-bridge",
		Version:     "0.1.0",
		Timestamp:   time.Now(),
		Uptime:      "1h30m",
	}

	mockHealth.On("CheckHealth", mock.Anything).Return(expectedStatus, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
