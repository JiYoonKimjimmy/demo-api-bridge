package database

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"testing"
	"time"
)

func TestMockRoutingRepository_Create(t *testing.T) {
	repo := NewMockRoutingRepository()
	ctx := context.Background()

	rule := domain.NewRoutingRule("rule1", "Test Rule", "/api/*", "GET", "endpoint1")

	// Create
	if err := repo.Create(ctx, rule); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Duplicate create should fail
	if err := repo.Create(ctx, rule); err == nil {
		t.Error("Expected error for duplicate create")
	}
}

func TestMockRoutingRepository_Update(t *testing.T) {
	repo := NewMockRoutingRepository()
	ctx := context.Background()

	rule := domain.NewRoutingRule("rule1", "Test Rule", "/api/*", "GET", "endpoint1")

	// Create first
	if err := repo.Create(ctx, rule); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update
	rule.Name = "Updated Rule"
	if err := repo.Update(ctx, rule); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.FindByID(ctx, rule.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if retrieved.Name != "Updated Rule" {
		t.Errorf("Expected 'Updated Rule', got '%s'", retrieved.Name)
	}
}

func TestMockRoutingRepository_Delete(t *testing.T) {
	repo := NewMockRoutingRepository()
	ctx := context.Background()

	rule := domain.NewRoutingRule("rule1", "Test Rule", "/api/*", "GET", "endpoint1")

	// Create first
	if err := repo.Create(ctx, rule); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete
	if err := repo.Delete(ctx, rule.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should not exist after delete
	if _, err := repo.FindByID(ctx, rule.ID); err == nil {
		t.Error("Expected error after delete")
	}
}

func TestMockRoutingRepository_FindMatchingRules(t *testing.T) {
	repo := NewMockRoutingRepository()
	ctx := context.Background()

	// Create multiple rules
	rule1 := domain.NewRoutingRule("rule1", "API Rule", "/api/*", "GET", "endpoint1")
	rule2 := domain.NewRoutingRule("rule2", "Users Rule", "/api/users/*", "GET", "endpoint2")
	rule3 := domain.NewRoutingRule("rule3", "All Rule", "/*", "*", "endpoint3")

	if err := repo.Create(ctx, rule1); err != nil {
		t.Fatalf("Create rule1 failed: %v", err)
	}
	if err := repo.Create(ctx, rule2); err != nil {
		t.Fatalf("Create rule2 failed: %v", err)
	}
	if err := repo.Create(ctx, rule3); err != nil {
		t.Fatalf("Create rule3 failed: %v", err)
	}

	// Test matching
	request := domain.NewRequest("req1", "GET", "/api/users/123")
	rules, err := repo.FindMatchingRules(ctx, request)
	if err != nil {
		t.Fatalf("FindMatchingRules failed: %v", err)
	}

	if len(rules) != 3 { // All three rules should match
		t.Errorf("Expected 3 matching rules, got %d", len(rules))
	}
}

func TestMockEndpointRepository_CRUD(t *testing.T) {
	repo := NewMockEndpointRepository()
	ctx := context.Background()

	endpoint := domain.NewAPIEndpoint("endpoint1", "Test API", "https://api.example.com", "/v1/test", "GET")

	// Create
	if err := repo.Create(ctx, endpoint); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Read
	retrieved, err := repo.FindByID(ctx, endpoint.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if retrieved.Name != endpoint.Name {
		t.Errorf("Expected %s, got %s", endpoint.Name, retrieved.Name)
	}

	// Update
	endpoint.Name = "Updated API"
	if err := repo.Update(ctx, endpoint); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, err = repo.FindByID(ctx, endpoint.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if retrieved.Name != "Updated API" {
		t.Errorf("Expected 'Updated API', got '%s'", retrieved.Name)
	}

	// Delete
	if err := repo.Delete(ctx, endpoint.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should not exist after delete
	if _, err := repo.FindByID(ctx, endpoint.ID); err == nil {
		t.Error("Expected error after delete")
	}
}

func TestMockCacheRepository_CRUD(t *testing.T) {
	repo := NewMockCacheRepository()
	ctx := context.Background()

	key := "test_key"
	value := []byte("test_value")

	// Set
	if err := repo.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get
	retrieved, err := repo.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}

	// Exists
	exists, err := repo.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if !exists {
		t.Error("Key should exist")
	}

	// Delete
	if err := repo.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should not exist after delete
	if _, err := repo.Get(ctx, key); err == nil {
		t.Error("Expected error after delete")
	}
}

func TestMockCacheRepository_GetOrSet(t *testing.T) {
	repo := NewMockCacheRepository()
	ctx := context.Background()

	key := "test_key"
	value := []byte("test_value")

	// GetOrSet when key doesn't exist
	retrieved, err := repo.GetOrSet(ctx, key, time.Minute, func() ([]byte, error) {
		return value, nil
	})
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}

	// GetOrSet when key exists (should return cached value)
	retrieved, err = repo.GetOrSet(ctx, key, time.Minute, func() ([]byte, error) {
		return []byte("different_value"), nil
	})
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected cached value %s, got %s", string(value), string(retrieved))
	}
}
