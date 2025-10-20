package service

import (
	"context"
	"errors"
	"testing"

	"demo-api-bridge/internal/core/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEndpointService_CreateEndpoint_Success(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	endpoint := &domain.APIEndpoint{
		ID:      "endpoint-1",
		Name:    "Test Endpoint",
		BaseURL: "https://api.example.com",
		Path:    "/v1/users",
		Method:  "GET",
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "creating endpoint", "endpoint_id", "endpoint-1", "name", "Test Endpoint").Return()
	mockRepo.On("Create", ctx, endpoint).Return(nil)
	mockLogger.On("Info", "endpoint created successfully", "endpoint_id", "endpoint-1").Return()
	mockMetrics.On("IncrementCounter", "endpoints_created", map[string]string{"endpoint_id": "endpoint-1"}).Return()

	// When
	err := service.CreateEndpoint(ctx, endpoint)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestEndpointService_CreateEndpoint_ValidationError(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	invalidEndpoint := &domain.APIEndpoint{
		ID:      "", // Invalid: empty ID
		Name:    "Test Endpoint",
		BaseURL: "https://api.example.com",
		Path:    "/v1/users",
		Method:  "GET",
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "creating endpoint", "endpoint_id", "", "name", "Test Endpoint").Return()
	mockLogger.On("Error", "invalid endpoint", "error", mock.AnythingOfType("*domain.ValidationError")).Return()

	// When
	err := service.CreateEndpoint(ctx, invalidEndpoint)

	// Then
	assert.Error(t, err)

	mockLogger.AssertExpectations(t)
}

func TestEndpointService_CreateEndpoint_RepositoryError(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	endpoint := &domain.APIEndpoint{
		ID:      "endpoint-1",
		Name:    "Test Endpoint",
		BaseURL: "https://api.example.com",
		Path:    "/v1/users",
		Method:  "GET",
	}

	repoError := errors.New("database error")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "creating endpoint", "endpoint_id", "endpoint-1", "name", "Test Endpoint").Return()
	mockRepo.On("Create", ctx, endpoint).Return(repoError)
	mockLogger.On("Error", "failed to create endpoint", "error", repoError).Return()

	// When
	err := service.CreateEndpoint(ctx, endpoint)

	// Then
	assert.Error(t, err)
	assert.Equal(t, repoError, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestEndpointService_UpdateEndpoint_Success(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	endpoint := &domain.APIEndpoint{
		ID:      "endpoint-1",
		Name:    "Updated Endpoint",
		BaseURL: "https://api.example.com",
		Path:    "/v1/users",
		Method:  "GET",
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "updating endpoint", "endpoint_id", "endpoint-1").Return()
	mockRepo.On("Update", ctx, endpoint).Return(nil)
	mockLogger.On("Info", "endpoint updated successfully", "endpoint_id", "endpoint-1").Return()
	mockMetrics.On("IncrementCounter", "endpoints_updated", map[string]string{"endpoint_id": "endpoint-1"}).Return()

	// When
	err := service.UpdateEndpoint(ctx, endpoint)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestEndpointService_UpdateEndpoint_ValidationError(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	invalidEndpoint := &domain.APIEndpoint{
		ID:      "endpoint-1",
		Name:    "Updated Endpoint",
		BaseURL: "", // Invalid: empty BaseURL
		Path:    "/v1/users",
		Method:  "GET",
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "updating endpoint", "endpoint_id", "endpoint-1").Return()
	mockLogger.On("Error", "invalid endpoint", "error", mock.AnythingOfType("*domain.ValidationError")).Return()

	// When
	err := service.UpdateEndpoint(ctx, invalidEndpoint)

	// Then
	assert.Error(t, err)

	mockLogger.AssertExpectations(t)
}

func TestEndpointService_DeleteEndpoint_Success(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	endpointID := "endpoint-1"

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "deleting endpoint", "endpoint_id", endpointID).Return()
	mockRepo.On("Delete", ctx, endpointID).Return(nil)
	mockLogger.On("Info", "endpoint deleted successfully", "endpoint_id", endpointID).Return()
	mockMetrics.On("IncrementCounter", "endpoints_deleted", map[string]string{"endpoint_id": endpointID}).Return()

	// When
	err := service.DeleteEndpoint(ctx, endpointID)

	// Then
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestEndpointService_DeleteEndpoint_RepositoryError(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	endpointID := "endpoint-1"
	repoError := errors.New("database error")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockLogger.On("Info", "deleting endpoint", "endpoint_id", endpointID).Return()
	mockRepo.On("Delete", ctx, endpointID).Return(repoError)
	mockLogger.On("Error", "failed to delete endpoint", "error", repoError).Return()

	// When
	err := service.DeleteEndpoint(ctx, endpointID)

	// Then
	assert.Error(t, err)
	assert.Equal(t, repoError, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestEndpointService_GetEndpoint_Success(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	endpointID := "endpoint-1"
	expectedEndpoint := &domain.APIEndpoint{
		ID:      endpointID,
		Name:    "Test Endpoint",
		BaseURL: "https://api.example.com",
		Path:    "/v1/users",
		Method:  "GET",
	}

	mockRepo.On("FindByID", ctx, endpointID).Return(expectedEndpoint, nil)

	// When
	endpoint, err := service.GetEndpoint(ctx, endpointID)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, expectedEndpoint.ID, endpoint.ID)
	assert.Equal(t, expectedEndpoint.Name, endpoint.Name)

	mockRepo.AssertExpectations(t)
}

func TestEndpointService_GetEndpoint_NotFound(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	endpointID := "non-existent"
	notFoundError := errors.New("endpoint not found")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockRepo.On("FindByID", ctx, endpointID).Return(nil, notFoundError)
	mockLogger.On("Error", "failed to get endpoint", "endpoint_id", endpointID, "error", notFoundError).Return()

	// When
	endpoint, err := service.GetEndpoint(ctx, endpointID)

	// Then
	assert.Error(t, err)
	assert.Nil(t, endpoint)
	assert.Equal(t, notFoundError, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestEndpointService_ListEndpoints_Success(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	expectedEndpoints := []*domain.APIEndpoint{
		{
			ID:      "endpoint-1",
			Name:    "Endpoint 1",
			BaseURL: "https://api.example.com",
			Path:    "/v1/users",
			Method:  "GET",
		},
		{
			ID:      "endpoint-2",
			Name:    "Endpoint 2",
			BaseURL: "https://api.example.com",
			Path:    "/v1/products",
			Method:  "POST",
		},
	}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockRepo.On("FindAll", ctx).Return(expectedEndpoints, nil)
	mockLogger.On("Info", "endpoints listed successfully", "count", 2).Return()

	// When
	endpoints, err := service.ListEndpoints(ctx)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, endpoints)
	assert.Len(t, endpoints, 2)
	assert.Equal(t, "endpoint-1", endpoints[0].ID)
	assert.Equal(t, "endpoint-2", endpoints[1].ID)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestEndpointService_ListEndpoints_Empty(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	emptyEndpoints := []*domain.APIEndpoint{}

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockRepo.On("FindAll", ctx).Return(emptyEndpoints, nil)
	mockLogger.On("Info", "endpoints listed successfully", "count", 0).Return()

	// When
	endpoints, err := service.ListEndpoints(ctx)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, endpoints)
	assert.Len(t, endpoints, 0)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestEndpointService_ListEndpoints_RepositoryError(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	ctx := context.Background()
	repoError := errors.New("database connection failed")

	mockLogger.On("WithContext", ctx).Return(mockLogger)
	mockRepo.On("FindAll", ctx).Return([]*domain.APIEndpoint(nil), repoError)
	mockLogger.On("Error", "failed to list endpoints", "error", repoError).Return()

	// When
	endpoints, err := service.ListEndpoints(ctx)

	// Then
	assert.Error(t, err)
	assert.Nil(t, endpoints)
	assert.Equal(t, repoError, err)

	mockLogger.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestEndpointService_NewEndpointService_CreatesValidInstance(t *testing.T) {
	// Given
	mockRepo := &MockEndpointRepository{}
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsCollector{}

	// When
	service := NewEndpointService(mockRepo, mockLogger, mockMetrics)

	// Then
	assert.NotNil(t, service)
}

