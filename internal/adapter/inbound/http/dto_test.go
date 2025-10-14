package http

import (
	"testing"
	"time"

	"demo-api-bridge/internal/core/domain"

	"github.com/stretchr/testify/assert"
)

func TestCreateEndpointRequest_ToDomain(t *testing.T) {
	req := &CreateEndpointRequest{
		Name:        "test-endpoint",
		Description: "Test endpoint",
		BaseURL:     "http://test.com",
		HealthURL:   "http://test.com/health",
		IsActive:    true,
		Timeout:     5000, // 5 seconds
		RetryCount:  3,
	}

	endpoint := req.ToDomain()

	assert.Equal(t, "test-endpoint", endpoint.Name)
	assert.Equal(t, "Test endpoint", endpoint.Description)
	assert.Equal(t, "http://test.com", endpoint.BaseURL)
	assert.Equal(t, "http://test.com/health", endpoint.HealthURL)
	assert.True(t, endpoint.IsActive)
	assert.Equal(t, 5*time.Second, endpoint.Timeout)
	assert.Equal(t, 3, endpoint.RetryCount)
}

func TestUpdateEndpointRequest_ApplyTo(t *testing.T) {
	original := &domain.Endpoint{
		ID:          1,
		Name:        "original",
		Description: "Original description",
		BaseURL:     "http://original.com",
		IsActive:    true,
		Timeout:     10 * time.Second,
		RetryCount:  5,
	}

	update := &UpdateEndpointRequest{
		Name:        stringPtr("updated"),
		Description: stringPtr("Updated description"),
		Timeout:     intPtr(3000), // 3 seconds
		RetryCount:  intPtr(2),
	}

	update.ApplyTo(original)

	assert.Equal(t, "updated", original.Name)
	assert.Equal(t, "Updated description", original.Description)
	assert.Equal(t, "http://original.com", original.BaseURL) // Unchanged
	assert.True(t, original.IsActive)                        // Unchanged
	assert.Equal(t, 3*time.Second, original.Timeout)
	assert.Equal(t, 2, original.RetryCount)
}

func TestCreateRoutingRuleRequest_ToDomain(t *testing.T) {
	req := &CreateRoutingRuleRequest{
		Name:        "test-rule",
		Description: "Test routing rule",
		PathPattern: "/test/*",
		Method:      "GET",
		Priority:    10,
		IsActive:    true,
		LegacyEndpoint: &EndpointReference{
			ID:   1,
			Name: "legacy",
		},
		ModernEndpoint: &EndpointReference{
			ID:   2,
			Name: "modern",
		},
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
		QueryParams: map[string]string{
			"version": "v1",
		},
	}

	rule := req.ToDomain()

	assert.Equal(t, "test-rule", rule.Name)
	assert.Equal(t, "Test routing rule", rule.Description)
	assert.Equal(t, "/test/*", rule.PathPattern)
	assert.Equal(t, "GET", rule.Method)
	assert.Equal(t, 10, rule.Priority)
	assert.True(t, rule.IsActive)
	assert.Equal(t, int64(1), rule.LegacyEndpointID)
	assert.Equal(t, int64(2), rule.ModernEndpointID)
	assert.Equal(t, "Bearer token", rule.Headers["Authorization"])
	assert.Equal(t, "v1", rule.QueryParams["version"])
}

func TestUpdateRoutingRuleRequest_ApplyTo(t *testing.T) {
	original := &domain.RoutingRule{
		ID:               1,
		Name:             "original",
		Description:      "Original description",
		PathPattern:      "/original/*",
		Method:           "POST",
		Priority:         5,
		IsActive:         false,
		LegacyEndpointID: 1,
		ModernEndpointID: 2,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		QueryParams: map[string]string{
			"debug": "true",
		},
	}

	update := &UpdateRoutingRuleRequest{
		Name:        stringPtr("updated"),
		Description: stringPtr("Updated description"),
		Method:      stringPtr("GET"),
		Priority:    intPtr(10),
		IsActive:    boolPtr(true),
		ModernEndpoint: &EndpointReference{
			ID: 3,
		},
		Headers: map[string]string{
			"Authorization": "Bearer new-token",
		},
	}

	update.ApplyTo(original)

	assert.Equal(t, "updated", original.Name)
	assert.Equal(t, "Updated description", original.Description)
	assert.Equal(t, "/original/*", original.PathPattern) // Unchanged
	assert.Equal(t, "GET", original.Method)
	assert.Equal(t, 10, original.Priority)
	assert.True(t, original.IsActive)
	assert.Equal(t, int64(1), original.LegacyEndpointID) // Unchanged
	assert.Equal(t, int64(3), original.ModernEndpointID) // Updated
	assert.Equal(t, "Bearer new-token", original.Headers["Authorization"])
	assert.Equal(t, "true", original.QueryParams["debug"]) // Unchanged
}

func TestEndpointResponse_FromDomain(t *testing.T) {
	endpoint := &domain.Endpoint{
		ID:          1,
		Name:        "test-endpoint",
		Description: "Test endpoint",
		BaseURL:     "http://test.com",
		HealthURL:   "http://test.com/health",
		IsActive:    true,
		Timeout:     5 * time.Second,
		RetryCount:  3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	resp := &EndpointResponse{}
	resp.FromDomain(endpoint)

	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "test-endpoint", resp.Name)
	assert.Equal(t, "Test endpoint", resp.Description)
	assert.Equal(t, "http://test.com", resp.BaseURL)
	assert.Equal(t, "http://test.com/health", resp.HealthURL)
	assert.True(t, resp.IsActive)
	assert.Equal(t, 5000, resp.Timeout) // 5 seconds in milliseconds
	assert.Equal(t, 3, resp.RetryCount)
}

func TestRoutingRuleResponse_FromDomain(t *testing.T) {
	rule := &domain.RoutingRule{
		ID:               1,
		Name:             "test-rule",
		Description:      "Test routing rule",
		PathPattern:      "/test/*",
		Method:           "GET",
		Priority:         10,
		IsActive:         true,
		LegacyEndpointID: 1,
		ModernEndpointID: 2,
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
		QueryParams: map[string]string{
			"version": "v1",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := &RoutingRuleResponse{}
	resp.FromDomain(rule)

	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "test-rule", resp.Name)
	assert.Equal(t, "Test routing rule", resp.Description)
	assert.Equal(t, "/test/*", resp.PathPattern)
	assert.Equal(t, "GET", resp.Method)
	assert.Equal(t, 10, resp.Priority)
	assert.True(t, resp.IsActive)
	assert.Equal(t, "Bearer token", resp.Headers["Authorization"])
	assert.Equal(t, "v1", resp.QueryParams["version"])
	assert.Equal(t, int64(1), resp.LegacyEndpoint.ID)
	assert.Equal(t, int64(2), resp.ModernEndpoint.ID)
}

func TestToEndpointResponse(t *testing.T) {
	endpoint := &domain.Endpoint{
		ID:          1,
		Name:        "test-endpoint",
		Description: "Test endpoint",
		BaseURL:     "http://test.com",
		IsActive:    true,
		Timeout:     5 * time.Second,
		RetryCount:  3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	resp := ToEndpointResponse(endpoint)

	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "test-endpoint", resp.Name)
	assert.Equal(t, "Test endpoint", resp.Description)
	assert.Equal(t, "http://test.com", resp.BaseURL)
	assert.True(t, resp.IsActive)
	assert.Equal(t, 5000, resp.Timeout)
	assert.Equal(t, 3, resp.RetryCount)
}

func TestToEndpointResponseList(t *testing.T) {
	endpoints := []*domain.Endpoint{
		{
			ID:   1,
			Name: "endpoint-1",
		},
		{
			ID:   2,
			Name: "endpoint-2",
		},
	}

	responses := ToEndpointResponseList(endpoints)

	assert.Len(t, responses, 2)
	assert.Equal(t, "endpoint-1", responses[0].Name)
	assert.Equal(t, "endpoint-2", responses[1].Name)
}

func TestToRoutingRuleResponse(t *testing.T) {
	rule := &domain.RoutingRule{
		ID:               1,
		Name:             "test-rule",
		PathPattern:      "/test/*",
		Method:           "GET",
		LegacyEndpointID: 1,
		ModernEndpointID: 2,
	}

	resp := ToRoutingRuleResponse(rule)

	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "test-rule", resp.Name)
	assert.Equal(t, "/test/*", resp.PathPattern)
	assert.Equal(t, "GET", resp.Method)
	assert.Equal(t, int64(1), resp.LegacyEndpoint.ID)
	assert.Equal(t, int64(2), resp.ModernEndpoint.ID)
}

func TestToRoutingRuleResponseList(t *testing.T) {
	rules := []*domain.RoutingRule{
		{
			ID:   1,
			Name: "rule-1",
		},
		{
			ID:   2,
			Name: "rule-2",
		},
	}

	responses := ToRoutingRuleResponseList(rules)

	assert.Len(t, responses, 2)
	assert.Equal(t, "rule-1", responses[0].Name)
	assert.Equal(t, "rule-2", responses[1].Name)
}

func TestToHealthResponse(t *testing.T) {
	status := &domain.HealthStatus{
		Status:      "healthy",
		ServiceName: "api-bridge",
		Version:     "0.1.0",
		Timestamp:   time.Now(),
		Uptime:      "1h30m",
	}

	resp := ToHealthResponse(status)

	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "api-bridge", resp.Service)
	assert.Equal(t, "0.1.0", resp.Version)
	assert.Equal(t, "1h30m", resp.Uptime)
}

func TestToReadinessResponse(t *testing.T) {
	status := &domain.ReadinessStatus{
		Status: "ready",
		Ready:  true,
		Checks: map[string]string{
			"database": "ok",
			"cache":    "ok",
		},
		Timestamp: time.Now(),
	}

	response := ToReadinessResponse(status)

	assert.Equal(t, "ready", response["status"])
	assert.True(t, response["ready"].(bool))
	assert.Equal(t, "ok", response["checks"].(map[string]string)["database"])
	assert.Equal(t, "ok", response["checks"].(map[string]string)["cache"])
}

func TestToStatusResponse(t *testing.T) {
	status := &domain.ServiceStatus{
		ServiceName: "api-bridge",
		Version:     "0.1.0",
		Timestamp:   time.Now(),
		Uptime:      "1h30m",
		Environment: "test",
		Metrics: map[string]interface{}{
			"requests": 100,
			"errors":   5,
		},
	}

	resp := ToStatusResponse(status)

	assert.Equal(t, "api-bridge", resp.Service)
	assert.Equal(t, "0.1.0", resp.Version)
	assert.Equal(t, "1h30m", resp.Uptime)
	assert.Equal(t, "test", resp.Environment)
	assert.Equal(t, 100, resp.Metrics["requests"])
	assert.Equal(t, 5, resp.Metrics["errors"])
}

func TestToErrorResponse(t *testing.T) {
	err := assert.AnError
	traceID := "test-trace-123"

	resp := ToErrorResponse(err, traceID)

	assert.True(t, resp.Error)
	assert.Equal(t, err.Error(), resp.Message)
	assert.Equal(t, traceID, resp.TraceID)
	assert.NotEmpty(t, resp.Timestamp)
}

// Helper functions for testing
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

// Benchmark tests
func BenchmarkCreateEndpointRequest_ToDomain(b *testing.B) {
	req := &CreateEndpointRequest{
		Name:        "test-endpoint",
		Description: "Test endpoint",
		BaseURL:     "http://test.com",
		HealthURL:   "http://test.com/health",
		IsActive:    true,
		Timeout:     5000,
		RetryCount:  3,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req.ToDomain()
	}
}

func BenchmarkEndpointResponse_FromDomain(b *testing.B) {
	endpoint := &domain.Endpoint{
		ID:          1,
		Name:        "test-endpoint",
		Description: "Test endpoint",
		BaseURL:     "http://test.com",
		HealthURL:   "http://test.com/health",
		IsActive:    true,
		Timeout:     5 * time.Second,
		RetryCount:  3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	resp := &EndpointResponse{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp.FromDomain(endpoint)
	}
}

func BenchmarkToEndpointResponseList(b *testing.B) {
	endpoints := make([]*domain.Endpoint, 100)
	for i := 0; i < 100; i++ {
		endpoints[i] = &domain.Endpoint{
			ID:          int64(i),
			Name:        "endpoint",
			Description: "Test endpoint",
			BaseURL:     "http://test.com",
			IsActive:    true,
			Timeout:     5 * time.Second,
			RetryCount:  3,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ToEndpointResponseList(endpoints)
	}
}
