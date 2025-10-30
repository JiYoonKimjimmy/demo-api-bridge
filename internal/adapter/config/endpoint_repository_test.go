package config

import (
	"context"
	"demo-api-bridge/pkg/config"
	"testing"
	"time"
)

func TestNewConfigEndpointRepository(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.EndpointsConfig
		wantErr bool
	}{
		{
			name: "Valid configuration with multiple endpoints",
			cfg: &config.EndpointsConfig{
				Endpoints: map[string]config.EndpointConfig{
					"test-endpoint-1": {
						ID:          "test-endpoint-1",
						Name:        "Test Endpoint 1",
						Description: "Test description",
						BaseURL:     "https://test1.example.com",
						HealthURL:   "/health",
						IsActive:    true,
						Timeout:     5 * time.Second,
					},
					"test-endpoint-2": {
						ID:          "test-endpoint-2",
						Name:        "Test Endpoint 2",
						Description: "Test description 2",
						BaseURL:     "https://test2.example.com",
						HealthURL:   "/health",
						IsActive:    true,
						Timeout:     3 * time.Second,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Nil configuration",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "Empty endpoints",
			cfg: &config.EndpointsConfig{
				Endpoints: map[string]config.EndpointConfig{},
			},
			wantErr: true,
		},
		{
			name: "Missing endpoint ID",
			cfg: &config.EndpointsConfig{
				Endpoints: map[string]config.EndpointConfig{
					"test": {
						Name:      "Test",
						BaseURL:   "https://test.com",
						IsActive:  true,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing base URL",
			cfg: &config.EndpointsConfig{
				Endpoints: map[string]config.EndpointConfig{
					"test": {
						ID:        "test-id",
						Name:      "Test",
						IsActive:  true,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewConfigEndpointRepository(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfigEndpointRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && repo == nil {
				t.Error("NewConfigEndpointRepository() returned nil repository")
			}
		})
	}
}

func TestConfigEndpointRepository_FindByID(t *testing.T) {
	cfg := &config.EndpointsConfig{
		Endpoints: map[string]config.EndpointConfig{
			"test-endpoint": {
				ID:          "test-endpoint",
				Name:        "Test Endpoint",
				Description: "Test description",
				BaseURL:     "https://test.example.com",
				HealthURL:   "/health",
				IsActive:    true,
				Timeout:     5 * time.Second,
			},
		},
	}

	repo, err := NewConfigEndpointRepository(cfg)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	tests := []struct {
		name       string
		endpointID string
		wantErr    bool
		checkFunc  func(t *testing.T, repo *configEndpointRepository, endpointID string)
	}{
		{
			name:       "Find existing endpoint",
			endpointID: "test-endpoint",
			wantErr:    false,
			checkFunc: func(t *testing.T, r *configEndpointRepository, endpointID string) {
				endpoint, err := r.FindByID(context.Background(), endpointID)
				if err != nil {
					t.Errorf("FindByID() error = %v", err)
					return
				}
				if endpoint.ID != endpointID {
					t.Errorf("FindByID() got ID = %v, want %v", endpoint.ID, endpointID)
				}
				if endpoint.Name != "Test Endpoint" {
					t.Errorf("FindByID() got Name = %v, want Test Endpoint", endpoint.Name)
				}
				if endpoint.BaseURL != "https://test.example.com" {
					t.Errorf("FindByID() got BaseURL = %v, want https://test.example.com", endpoint.BaseURL)
				}
			},
		},
		{
			name:       "Find non-existing endpoint",
			endpointID: "non-existing",
			wantErr:    true,
			checkFunc: func(t *testing.T, r *configEndpointRepository, endpointID string) {
				_, err := r.FindByID(context.Background(), endpointID)
				if err == nil {
					t.Error("FindByID() expected error for non-existing endpoint")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := repo.(*configEndpointRepository)
			tt.checkFunc(t, r, tt.endpointID)
		})
	}
}

func TestConfigEndpointRepository_FindAll(t *testing.T) {
	cfg := &config.EndpointsConfig{
		Endpoints: map[string]config.EndpointConfig{
			"endpoint-1": {
				ID:       "endpoint-1",
				Name:     "Endpoint 1",
				BaseURL:  "https://test1.com",
				IsActive: true,
			},
			"endpoint-2": {
				ID:       "endpoint-2",
				Name:     "Endpoint 2",
				BaseURL:  "https://test2.com",
				IsActive: true,
			},
		},
	}

	repo, err := NewConfigEndpointRepository(cfg)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	endpoints, err := repo.FindAll(context.Background())
	if err != nil {
		t.Errorf("FindAll() error = %v", err)
		return
	}

	if len(endpoints) != 2 {
		t.Errorf("FindAll() got %d endpoints, want 2", len(endpoints))
	}
}

func TestConfigEndpointRepository_FindActive(t *testing.T) {
	cfg := &config.EndpointsConfig{
		Endpoints: map[string]config.EndpointConfig{
			"active-endpoint": {
				ID:       "active-endpoint",
				Name:     "Active Endpoint",
				BaseURL:  "https://active.com",
				IsActive: true,
			},
			"inactive-endpoint": {
				ID:       "inactive-endpoint",
				Name:     "Inactive Endpoint",
				BaseURL:  "https://inactive.com",
				IsActive: false,
			},
		},
	}

	repo, err := NewConfigEndpointRepository(cfg)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	endpoints, err := repo.FindActive(context.Background())
	if err != nil {
		t.Errorf("FindActive() error = %v", err)
		return
	}

	if len(endpoints) != 1 {
		t.Errorf("FindActive() got %d endpoints, want 1", len(endpoints))
	}

	if len(endpoints) > 0 && endpoints[0].ID != "active-endpoint" {
		t.Errorf("FindActive() got endpoint ID = %v, want active-endpoint", endpoints[0].ID)
	}
}

func TestConfigEndpointRepository_UnsupportedOperations(t *testing.T) {
	cfg := &config.EndpointsConfig{
		Endpoints: map[string]config.EndpointConfig{
			"test": {
				ID:       "test",
				Name:     "Test",
				BaseURL:  "https://test.com",
				IsActive: true,
			},
		},
	}

	repo, err := NewConfigEndpointRepository(cfg)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	ctx := context.Background()

	// Test Create (should fail)
	err = repo.Create(ctx, nil)
	if err == nil {
		t.Error("Create() expected error, got nil")
	}

	// Test Update (should fail)
	err = repo.Update(ctx, nil)
	if err == nil {
		t.Error("Update() expected error, got nil")
	}

	// Test Delete (should fail)
	err = repo.Delete(ctx, "test")
	if err == nil {
		t.Error("Delete() expected error, got nil")
	}
}

func TestConfigEndpointRepository_ConcurrentAccess(t *testing.T) {
	cfg := &config.EndpointsConfig{
		Endpoints: map[string]config.EndpointConfig{
			"test-endpoint": {
				ID:       "test-endpoint",
				Name:     "Test Endpoint",
				BaseURL:  "https://test.com",
				IsActive: true,
			},
		},
	}

	repo, err := NewConfigEndpointRepository(cfg)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// 동시에 여러 고루틴에서 읽기 작업 수행
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			_, err := repo.FindByID(context.Background(), "test-endpoint")
			if err != nil {
				t.Errorf("Concurrent FindByID() error = %v", err)
			}
			done <- true
		}()
	}

	// 모든 고루틴이 완료될 때까지 대기
	for i := 0; i < 100; i++ {
		<-done
	}
}
