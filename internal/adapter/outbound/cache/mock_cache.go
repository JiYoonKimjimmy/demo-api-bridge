package cache

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"sync"
	"time"
)

// MockCacheRepository는 테스트를 위한 Mock 캐시 구현체입니다.
type MockCacheRepository struct {
	data  map[string][]byte
	mutex sync.RWMutex
}

// NewMockCacheRepository는 새로운 Mock 캐시 리포지토리를 생성합니다.
func NewMockCacheRepository() port.CacheRepository {
	return &MockCacheRepository{
		data: make(map[string][]byte),
	}
}

// Get은 캐시에서 값을 조회합니다.
func (m *MockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if value, exists := m.data[key]; exists {
		return value, nil
	}
	return nil, domain.NewDomainError("CACHE_MISS", "key not found", nil)
}

// Set은 캐시에 값을 저장합니다.
func (m *MockCacheRepository) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data[key] = value
	return nil
}

// Delete는 캐시에서 값을 삭제합니다.
func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	return nil
}

// Exists는 키가 존재하는지 확인합니다.
func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	_, exists := m.data[key]
	return exists, nil
}

// Clear는 모든 캐시를 삭제합니다.
func (m *MockCacheRepository) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data = make(map[string][]byte)
	return nil
}

// GetOrSet은 캐시에서 값을 조회하거나, 없으면 함수를 실행하여 저장합니다.
func (m *MockCacheRepository) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	// 먼저 캐시에서 조회
	if value, err := m.Get(ctx, key); err == nil {
		return value, nil
	}

	// 캐시에 없으면 함수 실행
	value, err := fn()
	if err != nil {
		return nil, err
	}

	// 결과를 캐시에 저장
	if err := m.Set(ctx, key, value, ttl); err != nil {
		return nil, err
	}

	return value, nil
}

// GetCacheStats는 캐시 통계를 반환합니다.
func (m *MockCacheRepository) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return &CacheStats{
		HitCount:  0, // Mock에서는 실제 통계를 추적하지 않음
		MissCount: 0,
		HitRate:   0.0,
		Size:      int64(len(m.data)),
	}, nil
}

// CacheStats는 캐시 통계를 나타냅니다.
type CacheStats struct {
	HitCount  int64   `json:"hit_count"`
	MissCount int64   `json:"miss_count"`
	HitRate   float64 `json:"hit_rate"`
	Size      int64   `json:"size"`
}
