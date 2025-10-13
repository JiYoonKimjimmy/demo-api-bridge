package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
)

// routingService는 RoutingService 인터페이스를 구현합니다.
type routingService struct {
	repo    port.RoutingRepository
	cache   port.CacheRepository
	logger  port.Logger
	metrics port.MetricsCollector
}

// NewRoutingService는 새로운 RoutingService를 생성합니다.
func NewRoutingService(
	repo port.RoutingRepository,
	cache port.CacheRepository,
	logger port.Logger,
	metrics port.MetricsCollector,
) port.RoutingService {
	return &routingService{
		repo:    repo,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// CreateRule은 새로운 라우팅 규칙을 생성합니다.
func (s *routingService) CreateRule(ctx context.Context, rule *domain.RoutingRule) error {
	s.logger.WithContext(ctx).Info("creating routing rule", "rule_id", rule.ID, "name", rule.Name)

	// 검증
	if err := rule.IsValid(); err != nil {
		s.logger.WithContext(ctx).Error("invalid routing rule", "error", err)
		return err
	}

	// 저장
	if err := s.repo.Create(ctx, rule); err != nil {
		s.logger.WithContext(ctx).Error("failed to create routing rule", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("routing rule created successfully", "rule_id", rule.ID)
	s.metrics.IncrementCounter("routing_rules_created", map[string]string{"rule_id": rule.ID})

	return nil
}

// UpdateRule은 라우팅 규칙을 수정합니다.
func (s *routingService) UpdateRule(ctx context.Context, rule *domain.RoutingRule) error {
	s.logger.WithContext(ctx).Info("updating routing rule", "rule_id", rule.ID)

	// 검증
	if err := rule.IsValid(); err != nil {
		s.logger.WithContext(ctx).Error("invalid routing rule", "error", err)
		return err
	}

	// 수정
	if err := s.repo.Update(ctx, rule); err != nil {
		s.logger.WithContext(ctx).Error("failed to update routing rule", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("routing rule updated successfully", "rule_id", rule.ID)
	s.metrics.IncrementCounter("routing_rules_updated", map[string]string{"rule_id": rule.ID})

	return nil
}

// DeleteRule은 라우팅 규칙을 삭제합니다.
func (s *routingService) DeleteRule(ctx context.Context, ruleID string) error {
	s.logger.WithContext(ctx).Info("deleting routing rule", "rule_id", ruleID)

	if err := s.repo.Delete(ctx, ruleID); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete routing rule", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("routing rule deleted successfully", "rule_id", ruleID)
	s.metrics.IncrementCounter("routing_rules_deleted", map[string]string{"rule_id": ruleID})

	return nil
}

// GetRule은 라우팅 규칙을 조회합니다.
func (s *routingService) GetRule(ctx context.Context, ruleID string) (*domain.RoutingRule, error) {
	rule, err := s.repo.FindByID(ctx, ruleID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get routing rule", "rule_id", ruleID, "error", err)
		return nil, err
	}

	return rule, nil
}

// ListRules는 모든 라우팅 규칙을 조회합니다.
func (s *routingService) ListRules(ctx context.Context) ([]*domain.RoutingRule, error) {
	rules, err := s.repo.FindAll(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list routing rules", "error", err)
		return nil, err
	}

	s.logger.WithContext(ctx).Info("routing rules listed successfully", "count", len(rules))
	return rules, nil
}
