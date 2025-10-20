package service

import (
	"context"
	"errors"
	"testing"

	"demo-api-bridge/internal/core/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRoutingService_CreateRule_Success(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	rule := &domain.RoutingRule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Method:     "GET",
		PathPattern: "/api/users",
		EndpointID: "endpoint-1",
		Priority:   1,
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "creating routing rule", "rule_id", "rule-1", "name", "Test Rule").Return()
	mockRepo.On("Create", ctx, rule).Return(nil)
	mockLogger.On("Info", "routing rule created successfully", "rule_id", "rule-1").Return()
	mockMetrics.On("IncrementCounter", "routing_rules_created", map[string]string{"rule_id": "rule-1"}).Return()

	// When
	err := service.CreateRule(ctx, rule)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestRoutingService_CreateRule_ValidationError(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	invalidRule := &domain.RoutingRule{
		ID:         "", // Invalid: empty ID
		Name:       "Test Rule",
		Method:     "GET",
		PathPattern: "/api/users",
		EndpointID: "endpoint-1",
		Priority:   1,
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "creating routing rule", "rule_id", "", "name", "Test Rule").Return()
	mockLogger.On("Error", "invalid routing rule", "error", mock.AnythingOfType("*domain.ValidationError")).Return()

	// When
	err := service.CreateRule(ctx, invalidRule)

	// Then
	assert.Error(t, err)

	mockLogger.AssertExpectations(t)
}

func TestRoutingService_CreateRule_RepositoryError(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	rule := &domain.RoutingRule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Method:     "GET",
		PathPattern: "/api/users",
		EndpointID: "endpoint-1",
		Priority:   1,
	}

	repoError := errors.New("database error")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "creating routing rule", "rule_id", "rule-1", "name", "Test Rule").Return()
	mockRepo.On("Create", ctx, rule).Return(repoError)
	mockLogger.On("Error", "failed to create routing rule", "error", repoError).Return()

	// When
	err := service.CreateRule(ctx, rule)

	// Then
	assert.Error(t, err)
	assert.Equal(t, repoError, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestRoutingService_UpdateRule_Success(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	rule := &domain.RoutingRule{
		ID:         "rule-1",
		Name:       "Updated Rule",
		Method:     "POST",
		PathPattern: "/api/users",
		EndpointID: "endpoint-1",
		Priority:   2,
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "updating routing rule", "rule_id", "rule-1").Return()
	mockRepo.On("Update", ctx, rule).Return(nil)
	mockLogger.On("Info", "routing rule updated successfully", "rule_id", "rule-1").Return()
	mockMetrics.On("IncrementCounter", "routing_rules_updated", map[string]string{"rule_id": "rule-1"}).Return()

	// When
	err := service.UpdateRule(ctx, rule)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestRoutingService_UpdateRule_ValidationError(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	invalidRule := &domain.RoutingRule{
		ID:          "rule-1",
		Name:        "Updated Rule",
		Method:      "POST",
		PathPattern: "", // Invalid: empty PathPattern
		EndpointID:  "endpoint-1",
		Priority:    2,
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "updating routing rule", "rule_id", "rule-1").Return()
	mockLogger.On("Error", "invalid routing rule", "error", mock.AnythingOfType("*domain.ValidationError")).Return()

	// When
	err := service.UpdateRule(ctx, invalidRule)

	// Then
	assert.Error(t, err)

	mockLogger.AssertExpectations(t)
}

func TestRoutingService_DeleteRule_Success(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	ruleID := "rule-1"

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "deleting routing rule", "rule_id", ruleID).Return()
	mockRepo.On("Delete", ctx, ruleID).Return(nil)
	mockLogger.On("Info", "routing rule deleted successfully", "rule_id", ruleID).Return()
	mockMetrics.On("IncrementCounter", "routing_rules_deleted", map[string]string{"rule_id": ruleID}).Return()

	// When
	err := service.DeleteRule(ctx, ruleID)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestRoutingService_DeleteRule_RepositoryError(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	ruleID := "rule-1"
	repoError := errors.New("database error")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "deleting routing rule", "rule_id", ruleID).Return()
	mockRepo.On("Delete", ctx, ruleID).Return(repoError)
	mockLogger.On("Error", "failed to delete routing rule", "error", repoError).Return()

	// When
	err := service.DeleteRule(ctx, ruleID)

	// Then
	assert.Error(t, err)
	assert.Equal(t, repoError, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestRoutingService_GetRule_Success(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	ruleID := "rule-1"
	expectedRule := &domain.RoutingRule{
		ID:         ruleID,
		Name:       "Test Rule",
		Method:     "GET",
		PathPattern: "/api/users",
		EndpointID: "endpoint-1",
		Priority:   1,
	}

	mockRepo.On("FindByID", ctx, ruleID).Return(expectedRule, nil)

	// When
	rule, err := service.GetRule(ctx, ruleID)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, rule)
	assert.Equal(t, expectedRule.ID, rule.ID)
	assert.Equal(t, expectedRule.Name, rule.Name)

	mockRepo.AssertExpectations(t)
}

func TestRoutingService_GetRule_NotFound(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	ruleID := "non-existent"
	notFoundError := errors.New("routing rule not found")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockRepo.On("FindByID", ctx, ruleID).Return(nil, notFoundError)
	mockLogger.On("Error", "failed to get routing rule", "rule_id", ruleID, "error", notFoundError).Return()

	// When
	rule, err := service.GetRule(ctx, ruleID)

	// Then
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Equal(t, notFoundError, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestRoutingService_ListRules_Success(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	expectedRules := []*domain.RoutingRule{
		{
			ID:         "rule-1",
			Name:       "Rule 1",
			Method:     "GET",
			PathPattern: "/api/users",
			EndpointID: "endpoint-1",
			Priority:   1,
		},
		{
			ID:         "rule-2",
			Name:       "Rule 2",
			Method:     "POST",
			PathPattern: "/api/products",
			EndpointID: "endpoint-2",
			Priority:   2,
		},
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockRepo.On("FindAll", ctx).Return(expectedRules, nil)
	mockLogger.On("Info", "routing rules listed successfully", "count", 2).Return()

	// When
	rules, err := service.ListRules(ctx)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, rules)
	assert.Len(t, rules, 2)
	assert.Equal(t, "rule-1", rules[0].ID)
	assert.Equal(t, "rule-2", rules[1].ID)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestRoutingService_ListRules_Empty(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	emptyRules := []*domain.RoutingRule{}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockRepo.On("FindAll", ctx).Return(emptyRules, nil)
	mockLogger.On("Info", "routing rules listed successfully", "count", 0).Return()

	// When
	rules, err := service.ListRules(ctx)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, rules)
	assert.Len(t, rules, 0)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestRoutingService_ListRules_RepositoryError(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	ctx := context.Background()
	repoError := errors.New("database connection failed")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockRepo.On("FindAll", ctx).Return([]*domain.RoutingRule(nil), repoError)
	mockLogger.On("Error", "failed to list routing rules", "error", repoError).Return()

	// When
	rules, err := service.ListRules(ctx)

	// Then
	assert.Error(t, err)
	assert.Nil(t, rules)
	assert.Equal(t, repoError, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestRoutingService_NewRoutingService_CreatesValidInstance(t *testing.T) {
	// Given
	mockRepo := &MockRoutingRepository{}
	mockCache := &MockCacheRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	// When
	service := NewRoutingService(mockRepo, mockCache, mockLogger, mockMetrics)

	// Then
	assert.NotNil(t, service)
}

