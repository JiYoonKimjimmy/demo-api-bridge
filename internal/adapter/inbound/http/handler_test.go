package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/pkg/logger"

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

func (m *MockBridgeService) GetRoutingRule(ctx context.Context, request *domain.Request) (*domain.RoutingRule, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*domain.RoutingRule), args.Error(1)
}

func (m *MockBridgeService) GetEndpoint(ctx context.Context, endpointID string) (*domain.APIEndpoint, error) {
	args := m.Called(ctx, endpointID)
	return args.Get(0).(*domain.APIEndpoint), args.Error(1)
}

type MockRoutingService struct {
	mock.Mock
}

func (m *MockRoutingService) CreateRule(ctx context.Context, rule *domain.RoutingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockRoutingService) UpdateRule(ctx context.Context, rule *domain.RoutingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockRoutingService) DeleteRule(ctx context.Context, ruleID string) error {
	args := m.Called(ctx, ruleID)
	return args.Error(0)
}

func (m *MockRoutingService) GetRule(ctx context.Context, ruleID string) (*domain.RoutingRule, error) {
	args := m.Called(ctx, ruleID)
	return args.Get(0).(*domain.RoutingRule), args.Error(1)
}

func (m *MockRoutingService) ListRules(ctx context.Context) ([]*domain.RoutingRule, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.RoutingRule), args.Error(1)
}

type MockEndpointService struct {
	mock.Mock
}

func (m *MockEndpointService) ListEndpoints(ctx context.Context) ([]*domain.APIEndpoint, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.APIEndpoint), args.Error(1)
}

func (m *MockEndpointService) GetEndpoint(ctx context.Context, endpointID string) (*domain.APIEndpoint, error) {
	args := m.Called(ctx, endpointID)
	return args.Get(0).(*domain.APIEndpoint), args.Error(1)
}

func (m *MockEndpointService) CreateEndpoint(ctx context.Context, endpoint *domain.APIEndpoint) error {
	args := m.Called(ctx, endpoint)
	return args.Error(0)
}

func (m *MockEndpointService) UpdateEndpoint(ctx context.Context, endpoint *domain.APIEndpoint) error {
	args := m.Called(ctx, endpoint)
	return args.Error(0)
}

func (m *MockEndpointService) DeleteEndpoint(ctx context.Context, endpointID string) error {
	args := m.Called(ctx, endpointID)
	return args.Error(0)
}

type MockHealthService struct {
	mock.Mock
}

func (m *MockHealthService) CheckHealth(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockHealthService) CheckReadiness(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockHealthService) GetServiceStatus(ctx context.Context) map[string]interface{} {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{})
}

type MockOrchestrationService struct {
	mock.Mock
}

func (m *MockOrchestrationService) ProcessParallelRequest(ctx context.Context, request *domain.Request, legacyEndpoint, modernEndpoint *domain.APIEndpoint) (*domain.APIComparison, error) {
	args := m.Called(ctx, request, legacyEndpoint, modernEndpoint)
	return args.Get(0).(*domain.APIComparison), args.Error(1)
}

func (m *MockOrchestrationService) GetOrchestrationRule(ctx context.Context, routingRuleID string) (*domain.OrchestrationRule, error) {
	args := m.Called(ctx, routingRuleID)
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

func setupTestHandler() (*Handler, *MockBridgeService, *MockRoutingService, *MockEndpointService, *MockHealthService, *MockOrchestrationService, *gin.Engine) {
	// Create mock services
	mockBridge := &MockBridgeService{}
	mockHealth := &MockHealthService{}
	mockEndpoint := &MockEndpointService{}
	mockRouting := &MockRoutingService{}
	mockOrchestration := &MockOrchestrationService{}

	// Create logger
	testLogger := logger.NewLogger()

	// Create handler
	handler := NewHandler(
		mockBridge,
		mockHealth,
		mockEndpoint,
		mockRouting,
		mockOrchestration,
		testLogger,
	)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register routes manually for testing - matching main.go structure
	// Internal Management API under /abs
	abs := router.Group("/abs")
	{
		// Health Check & Monitoring
		abs.GET("/health", handler.HealthCheck)
		abs.GET("/ready", handler.ReadinessCheck)
		abs.GET("/status", handler.Status)

		// Endpoint CRUD routes
		abs.GET("/v1/endpoints", handler.ListEndpoints)
		abs.POST("/v1/endpoints", handler.CreateEndpoint)
		abs.GET("/v1/endpoints/:id", handler.GetEndpoint)
		abs.PUT("/v1/endpoints/:id", handler.UpdateEndpoint)
		abs.DELETE("/v1/endpoints/:id", handler.DeleteEndpoint)

		// Routing rule CRUD routes
		abs.GET("/v1/routing-rules", handler.ListRoutingRules)
		abs.POST("/v1/routing-rules", handler.CreateRoutingRule)
		abs.GET("/v1/routing-rules/:id", handler.GetRoutingRule)
		abs.PUT("/v1/routing-rules/:id", handler.UpdateRoutingRule)
		abs.DELETE("/v1/routing-rules/:id", handler.DeleteRoutingRule)
	}

	// External API Bridge - all other requests
	router.NoRoute(handler.ProcessBridgeRequest)

	return handler, mockBridge, mockRouting, mockEndpoint, mockHealth, mockOrchestration, router
}

func TestHealthCheck(t *testing.T) {
	_, _, _, _, mockHealth, _, router := setupTestHandler()

	// Mock health check - CheckHealth returns error only
	mockHealth.On("CheckHealth", mock.Anything).Return(nil)

	// Create request
	req, _ := http.NewRequest("GET", "/abs/health", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.NotEmpty(t, response["timestamp"])

	mockHealth.AssertExpectations(t)
}

func TestReadinessCheck(t *testing.T) {
	_, _, _, _, mockHealth, _, router := setupTestHandler()

	// Mock readiness check - CheckReadiness returns error only
	mockHealth.On("CheckReadiness", mock.Anything).Return(nil)

	// Create request
	req, _ := http.NewRequest("GET", "/abs/ready", nil)
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
	_, _, _, _, mockHealth, _, router := setupTestHandler()

	// Mock status response - GetServiceStatus returns map[string]interface{}
	expectedStatus := map[string]interface{}{
		"service":     "api-bridge",
		"version":     "0.1.0",
		"uptime":      "1h30m",
		"environment": "test",
		"metrics":     map[string]interface{}{"requests": 100},
	}

	mockHealth.On("GetServiceStatus", mock.Anything).Return(expectedStatus)

	// Create request
	req, _ := http.NewRequest("GET", "/abs/status", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "api-bridge", response["service"])
	assert.Equal(t, "0.1.0", response["version"])

	mockHealth.AssertExpectations(t)
}

func TestHandleAPIRequest(t *testing.T) {
	_, mockBridge, _, _, _, _, router := setupTestHandler()

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
	_, _, _, mockEndpoint, _, _, router := setupTestHandler()

	// Mock endpoints response
	expectedEndpoints := []*domain.APIEndpoint{
		{
			ID:          "test-endpoint-1",
			Name:        "test-endpoint",
			Description: "Test endpoint",
			BaseURL:     "http://test.com",
			IsActive:    true,
		},
	}

	mockEndpoint.On("ListEndpoints", mock.Anything).Return(expectedEndpoints, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/abs/v1/endpoints", nil)
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
	_, _, _, mockEndpoint, _, _, router := setupTestHandler()

	// Mock create endpoint - CreateEndpoint now returns error only
	mockEndpoint.On("CreateEndpoint", mock.Anything, mock.AnythingOfType("*domain.APIEndpoint")).Return(nil)

	// Create request body
	requestBody := CreateEndpointRequest{
		Name:        "new-endpoint",
		Description: "New endpoint",
		BaseURL:     "http://new.com",
		Method:      "GET",
		IsActive:    true,
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/abs/v1/endpoints", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response EndpointResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.ID)
	assert.Equal(t, "new-endpoint", response.Name)

	mockEndpoint.AssertExpectations(t)
}

func TestListRoutingRules(t *testing.T) {
	_, _, mockRouting, _, _, _, router := setupTestHandler()

	// Mock routing rules response
	expectedRules := []*domain.RoutingRule{
		{
			ID:          "test-rule-1",
			Name:        "test-rule",
			PathPattern: "/test/*",
			Method:      "GET",
			Priority:    1,
			IsActive:    true,
		},
	}

	mockRouting.On("ListRules", mock.Anything).Return(expectedRules, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/abs/v1/routing-rules", nil)
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
	_, _, mockRouting, _, _, _, router := setupTestHandler()

	// Mock create routing rule - CreateRule returns error only
	mockRouting.On("CreateRule", mock.Anything, mock.AnythingOfType("*domain.RoutingRule")).Return(nil)

	// Create request body
	requestBody := CreateRoutingRuleRequest{
		Name:        "new-rule",
		PathPattern: "/new/*",
		Method:      "GET",
		Priority:    1,
		IsActive:    true,
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/abs/v1/routing-rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response RoutingRuleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.ID)
	assert.Equal(t, "new-rule", response.Name)

	mockRouting.AssertExpectations(t)
}

func TestHealthCheckFailure(t *testing.T) {
	_, _, _, _, mockHealth, _, router := setupTestHandler()

	// Mock health check failure
	mockHealth.On("CheckHealth", mock.Anything).Return(assert.AnError)

	// Create request
	req, _ := http.NewRequest("GET", "/abs/health", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", response["status"])
	assert.NotNil(t, response["error"])
	assert.NotNil(t, response["timestamp"])

	mockHealth.AssertExpectations(t)
}

func TestHandleAPIRequestError(t *testing.T) {
	_, mockBridge, _, _, _, _, router := setupTestHandler()

	// Mock bridge service error
	mockBridge.On("ProcessRequest", mock.Anything, mock.AnythingOfType("*domain.Request")).Return((*domain.Response)(nil), assert.AnError)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response gin.H
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["error"])

	mockBridge.AssertExpectations(t)
}

// Benchmark tests
func BenchmarkHandleAPIRequest(b *testing.B) {
	_, mockBridge, _, _, _, _, router := setupTestHandler()

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
	_, _, _, _, mockHealth, _, router := setupTestHandler()

	// Mock health check - CheckHealth returns error only
	mockHealth.On("CheckHealth", mock.Anything).Return(nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/abs/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
