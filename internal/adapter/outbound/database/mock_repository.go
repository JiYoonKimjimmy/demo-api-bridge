package database

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"sync"
	"time"
)

// mockRoutingRepository는 메모리 기반 RoutingRepository 구현체입니다.
type mockRoutingRepository struct {
	rules map[string]*domain.RoutingRule
	mutex sync.RWMutex
}

// NewMockRoutingRepository는 새로운 Mock 라우팅 레포지토리를 생성합니다.
func NewMockRoutingRepository() port.RoutingRepository {
	return &mockRoutingRepository{
		rules: make(map[string]*domain.RoutingRule),
	}
}

// Create는 라우팅 규칙을 생성합니다.
func (m *mockRoutingRepository) Create(ctx context.Context, rule *domain.RoutingRule) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.rules[rule.ID]; exists {
		return fmt.Errorf("routing rule with ID %s already exists", rule.ID)
	}

	// 복사본 저장
	ruleCopy := *rule
	m.rules[rule.ID] = &ruleCopy
	return nil
}

// Update는 라우팅 규칙을 수정합니다.
func (m *mockRoutingRepository) Update(ctx context.Context, rule *domain.RoutingRule) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.rules[rule.ID]; !exists {
		return fmt.Errorf("routing rule with ID %s not found", rule.ID)
	}

	// 복사본 저장
	ruleCopy := *rule
	m.rules[rule.ID] = &ruleCopy
	return nil
}

// Delete는 라우팅 규칙을 삭제합니다.
func (m *mockRoutingRepository) Delete(ctx context.Context, ruleID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.rules[ruleID]; !exists {
		return fmt.Errorf("routing rule with ID %s not found", ruleID)
	}

	delete(m.rules, ruleID)
	return nil
}

// FindByID는 ID로 라우팅 규칙을 조회합니다.
func (m *mockRoutingRepository) FindByID(ctx context.Context, ruleID string) (*domain.RoutingRule, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	rule, exists := m.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("routing rule with ID %s not found", ruleID)
	}

	// 복사본 반환
	ruleCopy := *rule
	return &ruleCopy, nil
}

// FindAll은 모든 라우팅 규칙을 조회합니다.
func (m *mockRoutingRepository) FindAll(ctx context.Context) ([]*domain.RoutingRule, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	rules := make([]*domain.RoutingRule, 0, len(m.rules))
	for _, rule := range m.rules {
		ruleCopy := *rule
		rules = append(rules, &ruleCopy)
	}

	return rules, nil
}

// FindMatchingRules는 요청에 매칭되는 라우팅 규칙들을 조회합니다.
func (m *mockRoutingRepository) FindMatchingRules(ctx context.Context, request *domain.Request) ([]*domain.RoutingRule, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var matchingRules []*domain.RoutingRule
	for _, rule := range m.rules {
		if match, err := rule.Matches(request); err != nil {
			return nil, err
		} else if match {
			ruleCopy := *rule
			matchingRules = append(matchingRules, &ruleCopy)
		}
	}

	return matchingRules, nil
}

// mockEndpointRepository는 메모리 기반 EndpointRepository 구현체입니다.
type mockEndpointRepository struct {
	endpoints map[string]*domain.APIEndpoint
	mutex     sync.RWMutex
}

// NewMockEndpointRepository는 새로운 Mock 엔드포인트 레포지토리를 생성합니다.
func NewMockEndpointRepository() port.EndpointRepository {
	return &mockEndpointRepository{
		endpoints: make(map[string]*domain.APIEndpoint),
	}
}

// Create는 엔드포인트를 생성합니다.
func (m *mockEndpointRepository) Create(ctx context.Context, endpoint *domain.APIEndpoint) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.endpoints[endpoint.ID]; exists {
		return fmt.Errorf("endpoint with ID %s already exists", endpoint.ID)
	}

	// 복사본 저장
	endpointCopy := *endpoint
	m.endpoints[endpoint.ID] = &endpointCopy
	return nil
}

// Update는 엔드포인트를 수정합니다.
func (m *mockEndpointRepository) Update(ctx context.Context, endpoint *domain.APIEndpoint) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.endpoints[endpoint.ID]; !exists {
		return fmt.Errorf("endpoint with ID %s not found", endpoint.ID)
	}

	// 복사본 저장
	endpointCopy := *endpoint
	m.endpoints[endpoint.ID] = &endpointCopy
	return nil
}

// Delete는 엔드포인트를 삭제합니다.
func (m *mockEndpointRepository) Delete(ctx context.Context, endpointID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.endpoints[endpointID]; !exists {
		return fmt.Errorf("endpoint with ID %s not found", endpointID)
	}

	delete(m.endpoints, endpointID)
	return nil
}

// FindByID는 ID로 엔드포인트를 조회합니다.
func (m *mockEndpointRepository) FindByID(ctx context.Context, endpointID string) (*domain.APIEndpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	endpoint, exists := m.endpoints[endpointID]
	if !exists {
		return nil, fmt.Errorf("endpoint with ID %s not found", endpointID)
	}

	// 복사본 반환
	endpointCopy := *endpoint
	return &endpointCopy, nil
}

// FindAll은 모든 엔드포인트를 조회합니다.
func (m *mockEndpointRepository) FindAll(ctx context.Context) ([]*domain.APIEndpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	endpoints := make([]*domain.APIEndpoint, 0, len(m.endpoints))
	for _, endpoint := range m.endpoints {
		endpointCopy := *endpoint
		endpoints = append(endpoints, &endpointCopy)
	}

	return endpoints, nil
}

// FindActive는 활성화된 엔드포인트만 조회합니다.
func (m *mockEndpointRepository) FindActive(ctx context.Context) ([]*domain.APIEndpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var activeEndpoints []*domain.APIEndpoint
	for _, endpoint := range m.endpoints {
		if endpoint.IsActive {
			endpointCopy := *endpoint
			activeEndpoints = append(activeEndpoints, &endpointCopy)
		}
	}

	return activeEndpoints, nil
}

// FindDefaultLegacyEndpoint는 기본 레거시 엔드포인트를 조회합니다.
func (m *mockEndpointRepository) FindDefaultLegacyEndpoint(ctx context.Context) (*domain.APIEndpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 1차 우선순위: IsDefault=true && IsLegacy=true && IsActive=true
	for _, endpoint := range m.endpoints {
		if endpoint.IsDefault && endpoint.IsLegacy && endpoint.IsActive {
			endpointCopy := *endpoint
			return &endpointCopy, nil
		}
	}

	// 2차 우선순위: IsLegacy=true && IsActive=true (첫 번째 발견)
	for _, endpoint := range m.endpoints {
		if endpoint.IsLegacy && endpoint.IsActive {
			endpointCopy := *endpoint
			return &endpointCopy, nil
		}
	}

	// 3차 우선순위: IsActive=true (첫 번째 발견)
	for _, endpoint := range m.endpoints {
		if endpoint.IsActive {
			endpointCopy := *endpoint
			return &endpointCopy, nil
		}
	}

	return nil, fmt.Errorf("no default legacy endpoint found")
}

// FindDefaultModernEndpoint는 기본 모던 엔드포인트를 조회합니다.
func (m *mockEndpointRepository) FindDefaultModernEndpoint(ctx context.Context) (*domain.APIEndpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 1차 우선순위: IsDefault=true && IsLegacy=false && IsActive=true
	for _, endpoint := range m.endpoints {
		if endpoint.IsDefault && !endpoint.IsLegacy && endpoint.IsActive {
			endpointCopy := *endpoint
			return &endpointCopy, nil
		}
	}

	// 2차 우선순위: IsLegacy=false && IsActive=true (첫 번째 발견)
	for _, endpoint := range m.endpoints {
		if !endpoint.IsLegacy && endpoint.IsActive {
			endpointCopy := *endpoint
			return &endpointCopy, nil
		}
	}

	return nil, fmt.Errorf("no default modern endpoint found")
}

// mockCacheRepository는 메모리 기반 CacheRepository 구현체입니다.
type mockCacheRepository struct {
	data  map[string]cacheEntry
	mutex sync.RWMutex
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMockCacheRepository는 새로운 Mock 캐시 레포지토리를 생성합니다.
func NewMockCacheRepository() port.CacheRepository {
	return &mockCacheRepository{
		data: make(map[string]cacheEntry),
	}
}

// Get은 캐시에서 값을 조회합니다.
func (m *mockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entry, exists := m.data[key]
	if !exists {
		return nil, domain.ErrCacheNotFound
	}

	// 만료 확인
	if time.Now().After(entry.expiresAt) {
		delete(m.data, key)
		return nil, domain.ErrCacheExpired
	}

	// 복사본 반환
	value := make([]byte, len(entry.value))
	copy(value, entry.value)
	return value, nil
}

// Set은 캐시에 값을 저장합니다.
func (m *mockCacheRepository) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 복사본 저장
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	m.data[key] = cacheEntry{
		value:     valueCopy,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Delete는 캐시에서 값을 삭제합니다.
func (m *mockCacheRepository) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	return nil
}

// Exists는 캐시에 키가 존재하는지 확인합니다.
func (m *mockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entry, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 만료 확인
	if time.Now().After(entry.expiresAt) {
		delete(m.data, key)
		return false, nil
	}

	return true, nil
}

// GetOrSet은 캐시에서 값을 조회하거나, 없으면 함수를 실행하여 저장합니다.
func (m *mockCacheRepository) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
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
