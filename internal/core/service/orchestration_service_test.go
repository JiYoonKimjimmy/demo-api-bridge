package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOrchestrationService_ProcessParallelRequest_Success(t *testing.T) {
	// Given
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewOrchestrationService(
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockExternalAPI,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	legacyEndpoint := &domain.APIEndpoint{
		ID:       "legacy-endpoint-1",
		BaseURL:  "https://legacy-api.example.com",
		Path:     "/users",
		Method:   "GET",
		IsActive: true,
		Timeout:  30 * time.Second,
	}

	modernEndpoint := &domain.APIEndpoint{
		ID:       "modern-endpoint-1",
		BaseURL:  "https://modern-api.example.com",
		Path:     "/users",
		Method:   "GET",
		IsActive: true,
		Timeout:  30 * time.Second,
	}

	legacyResponse := &domain.Response{
		RequestID:  "test-request-id",
		StatusCode: 200,
		Body:       []byte(`{"users": [{"id": 1, "name": "John"}]}`),
	}

	modernResponse := &domain.Response{
		RequestID:  "test-request-id",
		StatusCode: 200,
		Body:       []byte(`{"users": [{"id": 1, "name": "John"}]}`),
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "starting parallel API calls", "request_id", "test-request-id", "legacy_endpoint", "https://legacy-api.example.com/users", "modern_endpoint", "https://modern-api.example.com/users").Return()
	mockExternalAPI.On("SendWithRetry", mock.AnythingOfType("*context.timerCtx"), legacyEndpoint, request).Return(legacyResponse, nil)
	mockExternalAPI.On("SendWithRetry", mock.AnythingOfType("*context.timerCtx"), modernEndpoint, request).Return(modernResponse, nil)
	mockMetrics.On("RecordHistogram", "parallel_api_call_duration", mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return()
	mockLogger.On("Info", "parallel API calls completed", "request_id", "test-request-id", "legacy_success", true, "modern_success", true, "duration_ms", mock.AnythingOfType("int64")).Return()
	mockMetrics.On("RecordGauge", "api_comparison_match_rate", mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return()
	mockLogger.On("Info", "API comparison completed", "request_id", "test-request-id", "match_rate", mock.AnythingOfType("float64"), "differences_count", mock.AnythingOfType("int")).Return()

	// When
	comparison, err := service.ProcessParallelRequest(ctx, request, legacyEndpoint, modernEndpoint)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, comparison)
	assert.Equal(t, "test-request-id", comparison.RequestID)
	assert.NotNil(t, comparison.LegacyResponse)
	assert.NotNil(t, comparison.ModernResponse)

	mockLogger.AssertExpectations(t)
	mockExternalAPI.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestOrchestrationService_ProcessParallelRequest_BothAPIsFail(t *testing.T) {
	// Given
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewOrchestrationService(
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockExternalAPI,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	legacyEndpoint := &domain.APIEndpoint{
		ID:       "legacy-endpoint-1",
		BaseURL:  "https://legacy-api.example.com",
		Path:     "/users",
		Method:   "GET",
		IsActive: true,
		Timeout:  30 * time.Second,
	}

	modernEndpoint := &domain.APIEndpoint{
		ID:       "modern-endpoint-1",
		BaseURL:  "https://modern-api.example.com",
		Path:     "/users",
		Method:   "GET",
		IsActive: true,
		Timeout:  30 * time.Second,
	}

	apiError := errors.New("API connection failed")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "starting parallel API calls", "request_id", "test-request-id", "legacy_endpoint", "https://legacy-api.example.com/users", "modern_endpoint", "https://modern-api.example.com/users").Return()
	mockExternalAPI.On("SendWithRetry", mock.AnythingOfType("*context.timerCtx"), legacyEndpoint, request).Return(nil, apiError)
	mockExternalAPI.On("SendWithRetry", mock.AnythingOfType("*context.timerCtx"), modernEndpoint, request).Return(nil, apiError)
	mockMetrics.On("RecordHistogram", "parallel_api_call_duration", mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return()
	mockLogger.On("Info", "parallel API calls completed", "request_id", "test-request-id", "legacy_success", false, "modern_success", false, "duration_ms", mock.AnythingOfType("int64")).Return()
	mockLogger.On("Error", "both API calls failed", "request_id", "test-request-id", "legacy_error", apiError, "modern_error", apiError).Return()
	mockMetrics.On("IncrementCounter", "parallel_api_calls_failed", mock.AnythingOfType("map[string]string")).Return()

	// When
	comparison, err := service.ProcessParallelRequest(ctx, request, legacyEndpoint, modernEndpoint)

	// Then
	assert.Error(t, err)
	assert.Nil(t, comparison)
	assert.Contains(t, err.Error(), "both legacy and modern API calls failed")

	mockLogger.AssertExpectations(t)
	mockExternalAPI.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestOrchestrationService_ProcessParallelRequest_OneAPIFails(t *testing.T) {
	// Given
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewOrchestrationService(
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockExternalAPI,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	legacyEndpoint := &domain.APIEndpoint{
		ID:       "legacy-endpoint-1",
		BaseURL:  "https://legacy-api.example.com",
		Path:     "/users",
		Method:   "GET",
		IsActive: true,
		Timeout:  30 * time.Second,
	}

	modernEndpoint := &domain.APIEndpoint{
		ID:       "modern-endpoint-1",
		BaseURL:  "https://modern-api.example.com",
		Path:     "/users",
		Method:   "GET",
		IsActive: true,
		Timeout:  30 * time.Second,
	}

	legacyResponse := &domain.Response{
		RequestID:  "test-request-id",
		StatusCode: 200,
		Body:       []byte(`{"users": [{"id": 1, "name": "John"}]}`),
	}

	apiError := errors.New("modern API connection failed")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "starting parallel API calls", "request_id", "test-request-id", "legacy_endpoint", "https://legacy-api.example.com/users", "modern_endpoint", "https://modern-api.example.com/users").Return()
	mockExternalAPI.On("SendWithRetry", mock.AnythingOfType("*context.timerCtx"), legacyEndpoint, request).Return(legacyResponse, nil)
	mockExternalAPI.On("SendWithRetry", mock.AnythingOfType("*context.timerCtx"), modernEndpoint, request).Return(nil, apiError)
	mockMetrics.On("RecordHistogram", "parallel_api_call_duration", mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return()
	mockLogger.On("Info", "parallel API calls completed", "request_id", "test-request-id", "legacy_success", true, "modern_success", false, "duration_ms", mock.AnythingOfType("int64")).Return()
	mockMetrics.On("RecordGauge", "api_comparison_match_rate", mock.AnythingOfType("float64"), mock.AnythingOfType("map[string]string")).Return()
	mockLogger.On("Info", "API comparison completed", "request_id", "test-request-id", "match_rate", mock.AnythingOfType("float64"), "differences_count", mock.AnythingOfType("int")).Return()

	// When
	comparison, err := service.ProcessParallelRequest(ctx, request, legacyEndpoint, modernEndpoint)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, comparison)
	assert.Equal(t, "test-request-id", comparison.RequestID)
	assert.NotNil(t, comparison.LegacyResponse)
	assert.Nil(t, comparison.ModernResponse)
	assert.Equal(t, 0.0, comparison.MatchRate)

	mockLogger.AssertExpectations(t)
	mockExternalAPI.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestOrchestrationService_EvaluateTransition_Success(t *testing.T) {
	// Given
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewOrchestrationService(
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockExternalAPI,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	rule := &domain.OrchestrationRule{
		ID:               "orch-1",
		RoutingRuleID:    "rule-1",
		LegacyEndpointID: "legacy-endpoint-1",
		ModernEndpointID: "modern-endpoint-1",
		CurrentMode:      domain.PARALLEL,
		TransitionConfig: domain.TransitionConfig{
			AutoTransitionEnabled:    true,
			MatchRateThreshold:       0.95,
			MinRequestsForTransition: 100,
		},
	}

	recentComparisons := make([]*domain.APIComparison, 100)
	for i := 0; i < 100; i++ {
		recentComparisons[i] = &domain.APIComparison{MatchRate: 0.98}
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockComparisonRepo.On("GetRecentComparisons", ctx, "rule-1", 100).Return(recentComparisons, nil)
	mockLogger.On("Info", "transition evaluation completed", "rule_id", "orch-1", "average_match_rate", mock.AnythingOfType("float64"), "can_transition", true, "comparisons_count", 100).Return()

	// When
	canTransition, err := service.EvaluateTransition(ctx, rule)

	// Then
	assert.NoError(t, err)
	assert.True(t, canTransition)

	mockLogger.AssertExpectations(t)
	mockComparisonRepo.AssertExpectations(t)
}

func TestOrchestrationService_EvaluateTransition_InsufficientData(t *testing.T) {
	// Given
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewOrchestrationService(
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockExternalAPI,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	rule := &domain.OrchestrationRule{
		ID:               "orch-1",
		RoutingRuleID:    "rule-1",
		LegacyEndpointID: "legacy-endpoint-1",
		ModernEndpointID: "modern-endpoint-1",
		CurrentMode:      domain.PARALLEL,
		TransitionConfig: domain.TransitionConfig{
			AutoTransitionEnabled:    true,
			MatchRateThreshold:       0.95,
			MinRequestsForTransition: 100,
		},
	}

	recentComparisons := []*domain.APIComparison{
		{MatchRate: 0.98},
		{MatchRate: 0.96},
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockComparisonRepo.On("GetRecentComparisons", ctx, "rule-1", 100).Return(recentComparisons, nil)
	mockLogger.On("Info", "insufficient comparison data for transition", "rule_id", "orch-1", "required", 100, "available", 2).Return()

	// When
	canTransition, err := service.EvaluateTransition(ctx, rule)

	// Then
	assert.NoError(t, err)
	assert.False(t, canTransition)

	mockLogger.AssertExpectations(t)
	mockComparisonRepo.AssertExpectations(t)
}

func TestOrchestrationService_ExecuteTransition_Success(t *testing.T) {
	// Given
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewOrchestrationService(
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockExternalAPI,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	rule := &domain.OrchestrationRule{
		ID:               "orch-1",
		RoutingRuleID:    "rule-1",
		LegacyEndpointID: "legacy-endpoint-1",
		ModernEndpointID: "modern-endpoint-1",
		CurrentMode:      domain.PARALLEL,
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "executing API mode transition", "rule_id", "orch-1", "from_mode", domain.PARALLEL, "to_mode", domain.MODERN_ONLY).Return()
	mockOrchestrationRepo.On("Update", ctx, mock.AnythingOfType("*domain.OrchestrationRule")).Return(nil)
	mockLogger.On("Info", "API mode transition completed successfully", "rule_id", "orch-1", "new_mode", domain.MODERN_ONLY).Return()
	mockMetrics.On("IncrementCounter", "api_mode_transitions", mock.AnythingOfType("map[string]string")).Return()

	// When
	err := service.ExecuteTransition(ctx, rule, domain.MODERN_ONLY)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, domain.MODERN_ONLY, rule.CurrentMode)

	mockLogger.AssertExpectations(t)
	mockOrchestrationRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestOrchestrationService_CreateOrchestrationRule_Success(t *testing.T) {
	// Given
	mockOrchestrationRepo := &MockOrchestrationRepository{}
	mockComparisonRepo := &MockComparisonRepository{}
	mockExternalAPI := &MockExternalAPIClient{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewOrchestrationService(
		mockOrchestrationRepo,
		mockComparisonRepo,
		mockExternalAPI,
		mockLogger,
		mockMetrics,
	)

	ctx := context.Background()
	rule := &domain.OrchestrationRule{
		ID:               "orch-1",
		Name:             "Test Rule",
		RoutingRuleID:    "rule-1",
		LegacyEndpointID: "legacy-endpoint-1",
		ModernEndpointID: "modern-endpoint-1",
		CurrentMode:      domain.PARALLEL,
		TransitionConfig: domain.TransitionConfig{
			MatchRateThreshold: 0.95,
		},
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "creating orchestration rule", "rule_id", "orch-1", "name", "Test Rule").Return()
	mockOrchestrationRepo.On("Create", ctx, rule).Return(nil)
	mockLogger.On("Info", "orchestration rule created successfully", "rule_id", "orch-1").Return()
	mockMetrics.On("IncrementCounter", "orchestration_rules_created", mock.AnythingOfType("map[string]string")).Return()

	// When
	err := service.CreateOrchestrationRule(ctx, rule)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
	mockOrchestrationRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}
