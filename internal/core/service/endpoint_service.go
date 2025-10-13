package service

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
)

// endpointService는 EndpointService 인터페이스를 구현합니다.
type endpointService struct {
	repo    port.EndpointRepository
	logger  port.Logger
	metrics port.MetricsCollector
}

// NewEndpointService는 새로운 EndpointService를 생성합니다.
func NewEndpointService(
	repo port.EndpointRepository,
	logger port.Logger,
	metrics port.MetricsCollector,
) port.EndpointService {
	return &endpointService{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
	}
}

// CreateEndpoint는 새로운 엔드포인트를 생성합니다.
func (s *endpointService) CreateEndpoint(ctx context.Context, endpoint *domain.APIEndpoint) error {
	s.logger.WithContext(ctx).Info("creating endpoint", "endpoint_id", endpoint.ID, "name", endpoint.Name)

	// 검증
	if err := endpoint.IsValid(); err != nil {
		s.logger.WithContext(ctx).Error("invalid endpoint", "error", err)
		return err
	}

	// 저장
	if err := s.repo.Create(ctx, endpoint); err != nil {
		s.logger.WithContext(ctx).Error("failed to create endpoint", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("endpoint created successfully", "endpoint_id", endpoint.ID)
	s.metrics.IncrementCounter("endpoints_created", map[string]string{"endpoint_id": endpoint.ID})

	return nil
}

// UpdateEndpoint는 엔드포인트를 수정합니다.
func (s *endpointService) UpdateEndpoint(ctx context.Context, endpoint *domain.APIEndpoint) error {
	s.logger.WithContext(ctx).Info("updating endpoint", "endpoint_id", endpoint.ID)

	// 검증
	if err := endpoint.IsValid(); err != nil {
		s.logger.WithContext(ctx).Error("invalid endpoint", "error", err)
		return err
	}

	// 수정
	if err := s.repo.Update(ctx, endpoint); err != nil {
		s.logger.WithContext(ctx).Error("failed to update endpoint", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("endpoint updated successfully", "endpoint_id", endpoint.ID)
	s.metrics.IncrementCounter("endpoints_updated", map[string]string{"endpoint_id": endpoint.ID})

	return nil
}

// DeleteEndpoint는 엔드포인트를 삭제합니다.
func (s *endpointService) DeleteEndpoint(ctx context.Context, endpointID string) error {
	s.logger.WithContext(ctx).Info("deleting endpoint", "endpoint_id", endpointID)

	if err := s.repo.Delete(ctx, endpointID); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete endpoint", "error", err)
		return err
	}

	s.logger.WithContext(ctx).Info("endpoint deleted successfully", "endpoint_id", endpointID)
	s.metrics.IncrementCounter("endpoints_deleted", map[string]string{"endpoint_id": endpointID})

	return nil
}

// GetEndpoint는 엔드포인트를 조회합니다.
func (s *endpointService) GetEndpoint(ctx context.Context, endpointID string) (*domain.APIEndpoint, error) {
	endpoint, err := s.repo.FindByID(ctx, endpointID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get endpoint", "endpoint_id", endpointID, "error", err)
		return nil, err
	}

	return endpoint, nil
}

// ListEndpoints는 모든 엔드포인트를 조회합니다.
func (s *endpointService) ListEndpoints(ctx context.Context) ([]*domain.APIEndpoint, error) {
	endpoints, err := s.repo.FindAll(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list endpoints", "error", err)
		return nil, err
	}

	s.logger.WithContext(ctx).Info("endpoints listed successfully", "count", len(endpoints))
	return endpoints, nil
}
