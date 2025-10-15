package database

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"sync"
)

// mockOrchestrationRepository는 OrchestrationRepository의 Mock 구현체입니다.
type mockOrchestrationRepository struct {
	rules map[string]*domain.OrchestrationRule
	mutex sync.RWMutex
}

// NewMockOrchestrationRepository는 새로운 Mock OrchestrationRepository를 생성합니다.
func NewMockOrchestrationRepository() port.OrchestrationRepository {
	repo := &mockOrchestrationRepository{
		rules: make(map[string]*domain.OrchestrationRule),
	}

	// 초기 테스트 데이터 생성
	repo.initializeTestData()

	return repo
}

// initializeTestData는 테스트용 초기 데이터를 생성합니다.
func (r *mockOrchestrationRepository) initializeTestData() {
	// 예시 오케스트레이션 규칙
	rule1 := domain.NewOrchestrationRule(
		"orch-rule-1",
		"User API Orchestration",
		"route-1",
		"legacy-user-endpoint",
		"modern-user-endpoint",
	)
	rule1.Description = "사용자 API 병렬 호출 및 전환 규칙"

	rule2 := domain.NewOrchestrationRule(
		"orch-rule-2",
		"Product API Orchestration",
		"route-2",
		"legacy-product-endpoint",
		"modern-product-endpoint",
	)
	rule2.Description = "상품 API 병렬 호출 및 전환 규칙"

	r.rules[rule1.ID] = rule1
	r.rules[rule2.ID] = rule2
}

// Create는 오케스트레이션 규칙을 생성합니다.
func (r *mockOrchestrationRepository) Create(ctx context.Context, rule *domain.OrchestrationRule) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.rules[rule.ID]; exists {
		return fmt.Errorf("orchestration rule with ID %s already exists", rule.ID)
	}

	// 복사본 생성 (원본 보호)
	ruleCopy := *rule
	r.rules[rule.ID] = &ruleCopy

	return nil
}

// Update는 오케스트레이션 규칙을 수정합니다.
func (r *mockOrchestrationRepository) Update(ctx context.Context, rule *domain.OrchestrationRule) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.rules[rule.ID]; !exists {
		return fmt.Errorf("orchestration rule with ID %s not found", rule.ID)
	}

	// 복사본 생성 (원본 보호)
	ruleCopy := *rule
	r.rules[rule.ID] = &ruleCopy

	return nil
}

// Delete는 오케스트레이션 규칙을 삭제합니다.
func (r *mockOrchestrationRepository) Delete(ctx context.Context, ruleID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.rules[ruleID]; !exists {
		return fmt.Errorf("orchestration rule with ID %s not found", ruleID)
	}

	delete(r.rules, ruleID)
	return nil
}

// FindByID는 ID로 오케스트레이션 규칙을 조회합니다.
func (r *mockOrchestrationRepository) FindByID(ctx context.Context, ruleID string) (*domain.OrchestrationRule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	rule, exists := r.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("orchestration rule with ID %s not found", ruleID)
	}

	// 복사본 반환 (원본 보호)
	ruleCopy := *rule
	return &ruleCopy, nil
}

// FindByRoutingRuleID는 라우팅 규칙 ID로 오케스트레이션 규칙을 조회합니다.
func (r *mockOrchestrationRepository) FindByRoutingRuleID(ctx context.Context, routingRuleID string) (*domain.OrchestrationRule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, rule := range r.rules {
		if rule.RoutingRuleID == routingRuleID {
			// 복사본 반환 (원본 보호)
			ruleCopy := *rule
			return &ruleCopy, nil
		}
	}

	return nil, fmt.Errorf("orchestration rule for routing rule ID %s not found", routingRuleID)
}

// FindAll은 모든 오케스트레이션 규칙을 조회합니다.
func (r *mockOrchestrationRepository) FindAll(ctx context.Context) ([]*domain.OrchestrationRule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	rules := make([]*domain.OrchestrationRule, 0, len(r.rules))
	for _, rule := range r.rules {
		// 복사본 생성 (원본 보호)
		ruleCopy := *rule
		rules = append(rules, &ruleCopy)
	}

	return rules, nil
}

// FindActive는 활성화된 오케스트레이션 규칙만 조회합니다.
func (r *mockOrchestrationRepository) FindActive(ctx context.Context) ([]*domain.OrchestrationRule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var activeRules []*domain.OrchestrationRule
	for _, rule := range r.rules {
		if rule.IsActive {
			// 복사본 생성 (원본 보호)
			ruleCopy := *rule
			activeRules = append(activeRules, &ruleCopy)
		}
	}

	return activeRules, nil
}
