// Package config는 설정 파일 기반의 Repository 구현체를 제공합니다.
package config

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"demo-api-bridge/pkg/config"
	"fmt"
	"sync"
	"time"
)

// configEndpointRepository : 설정 파일 기반 EndpointRepository 구현체입니다.
//
// 이 구현체는 다음 특징을 가집니다:
//   - 메모리 기반으로 동작하여 매우 빠름 (DB 쿼리 없음)
//   - 애플리케이션 시작 시 설정 파일에서 한 번만 로드
//   - 불변(Immutable) 데이터로 동시성 안전
//   - 런타임 변경 불가 (재배포 필요)
type configEndpointRepository struct {
	endpoints map[string]*domain.APIEndpoint // endpointID -> APIEndpoint
	mu        sync.RWMutex                   // 읽기 최적화 락
}

// NewConfigEndpointRepository : 설정 기반 엔드포인트 저장소를 생성합니다.
//
// Parameters:
//   - cfg: 엔드포인트 설정 정보
//
// Returns:
//   - port.EndpointRepository: 엔드포인트 저장소 인터페이스
//   - error: 초기화 중 발생한 에러
//
// Example:
//
//	cfg := config.LoadConfig("config.yaml")
//	repo, err := NewConfigEndpointRepository(&cfg.Endpoints)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewConfigEndpointRepository(cfg *config.EndpointsConfig) (port.EndpointRepository, error) {
	if cfg == nil {
		return nil, fmt.Errorf("endpoints config is nil")
	}

	endpoints := make(map[string]*domain.APIEndpoint)

	// 설정 파일의 엔드포인트를 도메인 객체로 변환
	for key, epCfg := range cfg.Endpoints {
		endpoint, err := convertToEndpoint(key, &epCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert endpoint '%s': %w", key, err)
		}
		endpoints[endpoint.ID] = endpoint
	}

	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}

	return &configEndpointRepository{
		endpoints: endpoints,
	}, nil
}

// convertToEndpoint : 설정을 도메인 엔드포인트로 변환합니다.
func convertToEndpoint(key string, cfg *config.EndpointConfig) (*domain.APIEndpoint, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("endpoint ID is required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("endpoint base_url is required")
	}

	now := time.Now()

	endpoint := &domain.APIEndpoint{
		ID:          cfg.ID,
		Name:        cfg.Name,
		Description: cfg.Description,
		BaseURL:     cfg.BaseURL,
		HealthURL:   cfg.HealthURL,
		IsActive:    cfg.IsActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return endpoint, nil
}

// FindByID : ID로 엔드포인트를 조회합니다.
//
// 메모리에서 조회하므로 매우 빠릅니다 (O(1), ~나노초 수준).
//
// Parameters:
//   - ctx: 컨텍스트 (이 구현체에서는 사용하지 않음)
//   - endpointID: 조회할 엔드포인트 ID
//
// Returns:
//   - *domain.APIEndpoint: 엔드포인트 정보
//   - error: 엔드포인트를 찾지 못한 경우
func (r *configEndpointRepository) FindByID(ctx context.Context, endpointID string) (*domain.APIEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	endpoint, exists := r.endpoints[endpointID]
	if !exists {
		return nil, fmt.Errorf("endpoint with ID '%s' not found in configuration", endpointID)
	}

	// 포인터 복사본 반환 (원본 보호)
	result := *endpoint
	return &result, nil
}

// FindAll : 모든 엔드포인트를 조회합니다.
func (r *configEndpointRepository) FindAll(ctx context.Context) ([]*domain.APIEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	endpoints := make([]*domain.APIEndpoint, 0, len(r.endpoints))
	for _, ep := range r.endpoints {
		epCopy := *ep
		endpoints = append(endpoints, &epCopy)
	}

	return endpoints, nil
}

// FindByType : 타입별로 엔드포인트를 조회합니다.
//
// Config 기반 구현에서는 Name 필드로 간단한 매칭을 수행합니다.
func (r *configEndpointRepository) FindByType(ctx context.Context, endpointType string) ([]*domain.APIEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var endpoints []*domain.APIEndpoint
	for _, ep := range r.endpoints {
		// 이름에 타입이 포함되어 있으면 매칭
		if contains(ep.Name, endpointType) || contains(ep.ID, endpointType) {
			epCopy := *ep
			endpoints = append(endpoints, &epCopy)
		}
	}

	return endpoints, nil
}

// FindActive : 활성화된 엔드포인트만 조회합니다.
func (r *configEndpointRepository) FindActive(ctx context.Context) ([]*domain.APIEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var endpoints []*domain.APIEndpoint
	for _, ep := range r.endpoints {
		if ep.IsActive {
			epCopy := *ep
			endpoints = append(endpoints, &epCopy)
		}
	}

	return endpoints, nil
}

// Create : 설정 기반 저장소에서는 지원하지 않습니다.
//
// Config 파일로 관리되므로 런타임 생성은 불가능합니다.
// 엔드포인트를 추가하려면 설정 파일을 수정하고 재배포해야 합니다.
func (r *configEndpointRepository) Create(ctx context.Context, endpoint *domain.APIEndpoint) error {
	return fmt.Errorf("create operation is not supported for config-based endpoint repository")
}

// Update : 설정 기반 저장소에서는 지원하지 않습니다.
func (r *configEndpointRepository) Update(ctx context.Context, endpoint *domain.APIEndpoint) error {
	return fmt.Errorf("update operation is not supported for config-based endpoint repository")
}

// Delete : 설정 기반 저장소에서는 지원하지 않습니다.
func (r *configEndpointRepository) Delete(ctx context.Context, endpointID string) error {
	return fmt.Errorf("delete operation is not supported for config-based endpoint repository")
}

// Close : 리소스를 정리합니다 (메모리 기반이므로 특별한 작업 없음).
func (r *configEndpointRepository) Close() error {
	return nil
}

// contains : 문자열 포함 여부를 확인하는 헬퍼 함수입니다.
func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
