package database

import (
	"context"
	"database/sql"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"demo-api-bridge/pkg/config"
	"fmt"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

// oracleRoutingRepository는 OracleDB 기반 RoutingRepository 구현체입니다.
type oracleRoutingRepository struct {
	db *sql.DB
}

// NewOracleRoutingRepository는 새로운 Oracle 라우팅 레포지토리를 생성합니다.
func NewOracleRoutingRepository(cfg *config.DatabaseConfig) (port.RoutingRepository, error) {
	// DSN 생성
	dsn := cfg.GetDSN()

	// 데이터베이스 연결
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 연결 설정
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &oracleRoutingRepository{db: db}, nil
}

// Create는 라우팅 규칙을 생성합니다.
func (r *oracleRoutingRepository) Create(ctx context.Context, rule *domain.RoutingRule) error {
	query := `
		INSERT INTO routing_rules (
			id, name, description, method, path_pattern, 
			headers, query_params, legacy_endpoint_id, modern_endpoint_id,
			created_at, updated_at
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.Method,
		rule.PathPattern,
		rule.Headers,
		rule.QueryParams,
		rule.LegacyEndpointID,
		rule.ModernEndpointID,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create routing rule: %w", err)
	}

	return nil
}

// Update는 라우팅 규칙을 수정합니다.
func (r *oracleRoutingRepository) Update(ctx context.Context, rule *domain.RoutingRule) error {
	query := `
		UPDATE routing_rules SET
			name = :1,
			description = :2,
			method = :3,
			path_pattern = :4,
			headers = :5,
			query_params = :6,
			legacy_endpoint_id = :7,
			modern_endpoint_id = :8,
			updated_at = :9
		WHERE id = :10
	`

	result, err := r.db.ExecContext(ctx, query,
		rule.Name,
		rule.Description,
		rule.Method,
		rule.PathPattern,
		rule.Headers,
		rule.QueryParams,
		rule.LegacyEndpointID,
		rule.ModernEndpointID,
		time.Now(),
		rule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update routing rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("routing rule with ID %s not found", rule.ID)
	}

	return nil
}

// Delete는 라우팅 규칙을 삭제합니다.
func (r *oracleRoutingRepository) Delete(ctx context.Context, ruleID string) error {
	query := `DELETE FROM routing_rules WHERE id = :1`

	result, err := r.db.ExecContext(ctx, query, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete routing rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("routing rule with ID %s not found", ruleID)
	}

	return nil
}

// FindByID는 ID로 라우팅 규칙을 조회합니다.
func (r *oracleRoutingRepository) FindByID(ctx context.Context, ruleID string) (*domain.RoutingRule, error) {
	query := `
		SELECT id, name, description, method, path_pattern,
		       headers, query_params, legacy_endpoint_id, modern_endpoint_id,
		       created_at, updated_at
		FROM routing_rules
		WHERE id = :1
	`

	var rule domain.RoutingRule
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, ruleID).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&rule.Method,
		&rule.PathPattern,
		&rule.Headers,
		&rule.QueryParams,
		&rule.LegacyEndpointID,
		&rule.ModernEndpointID,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("routing rule with ID %s not found", ruleID)
		}
		return nil, fmt.Errorf("failed to query routing rule: %w", err)
	}

	rule.CreatedAt = createdAt
	rule.UpdatedAt = updatedAt

	return &rule, nil
}

// FindAll은 모든 라우팅 규칙을 조회합니다.
func (r *oracleRoutingRepository) FindAll(ctx context.Context) ([]*domain.RoutingRule, error) {
	query := `
		SELECT id, name, description, method, path_pattern,
		       headers, query_params, legacy_endpoint_id, modern_endpoint_id,
		       created_at, updated_at
		FROM routing_rules
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query routing rules: %w", err)
	}
	defer rows.Close()

	var rules []*domain.RoutingRule
	for rows.Next() {
		var rule domain.RoutingRule
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&rule.Method,
			&rule.PathPattern,
			&rule.Headers,
			&rule.QueryParams,
			&rule.LegacyEndpointID,
			&rule.ModernEndpointID,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan routing rule: %w", err)
		}

		rule.CreatedAt = createdAt
		rule.UpdatedAt = updatedAt
		rules = append(rules, &rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating routing rules: %w", err)
	}

	return rules, nil
}

// FindMatchingRules는 요청에 매칭되는 라우팅 규칙들을 조회합니다.
func (r *oracleRoutingRepository) FindMatchingRules(ctx context.Context, request *domain.Request) ([]*domain.RoutingRule, error) {
	query := `
		SELECT id, name, description, method, path_pattern,
		       headers, query_params, legacy_endpoint_id, modern_endpoint_id,
		       created_at, updated_at
		FROM routing_rules
		WHERE method = :1 AND path_pattern LIKE :2
		ORDER BY created_at DESC
	`

	// 간단한 패턴 매칭 (실제로는 더 정교한 로직 필요)
	pathPattern := "%" + request.Path + "%"

	rows, err := r.db.QueryContext(ctx, query, request.Method, pathPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query matching routing rules: %w", err)
	}
	defer rows.Close()

	var rules []*domain.RoutingRule
	for rows.Next() {
		var rule domain.RoutingRule
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&rule.Method,
			&rule.PathPattern,
			&rule.Headers,
			&rule.QueryParams,
			&rule.LegacyEndpointID,
			&rule.ModernEndpointID,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan routing rule: %w", err)
		}

		rule.CreatedAt = createdAt
		rule.UpdatedAt = updatedAt

		// 실제 매칭 로직 확인
		if match, err := rule.Matches(request); err != nil {
			return nil, err
		} else if match {
			rules = append(rules, &rule)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating matching routing rules: %w", err)
	}

	return rules, nil
}

// Close는 데이터베이스 연결을 닫습니다.
func (r *oracleRoutingRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// oracleEndpointRepository는 OracleDB 기반 EndpointRepository 구현체입니다.
type oracleEndpointRepository struct {
	db *sql.DB
}

// NewOracleEndpointRepository는 새로운 Oracle 엔드포인트 레포지토리를 생성합니다.
func NewOracleEndpointRepository(cfg *config.DatabaseConfig) (port.EndpointRepository, error) {
	// DSN 생성
	dsn := cfg.GetDSN()

	// 데이터베이스 연결
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 연결 설정
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &oracleEndpointRepository{db: db}, nil
}

// Create는 엔드포인트를 생성합니다.
func (r *oracleEndpointRepository) Create(ctx context.Context, endpoint *domain.APIEndpoint) error {
	query := `
		INSERT INTO api_endpoints (
			id, name, description, base_url, health_url,
			is_active, created_at, updated_at
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		endpoint.ID,
		endpoint.Name,
		endpoint.Description,
		endpoint.BaseURL,
		endpoint.HealthURL,
		endpoint.IsActive,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}

	return nil
}

// Update는 엔드포인트를 수정합니다.
func (r *oracleEndpointRepository) Update(ctx context.Context, endpoint *domain.APIEndpoint) error {
	query := `
		UPDATE api_endpoints SET
			name = :1,
			description = :2,
			base_url = :3,
			health_url = :4,
			is_active = :5,
			updated_at = :6
		WHERE id = :7
	`

	result, err := r.db.ExecContext(ctx, query,
		endpoint.Name,
		endpoint.Description,
		endpoint.BaseURL,
		endpoint.HealthURL,
		endpoint.IsActive,
		time.Now(),
		endpoint.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update endpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("endpoint with ID %s not found", endpoint.ID)
	}

	return nil
}

// Delete는 엔드포인트를 삭제합니다.
func (r *oracleEndpointRepository) Delete(ctx context.Context, endpointID string) error {
	query := `DELETE FROM api_endpoints WHERE id = :1`

	result, err := r.db.ExecContext(ctx, query, endpointID)
	if err != nil {
		return fmt.Errorf("failed to delete endpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("endpoint with ID %s not found", endpointID)
	}

	return nil
}

// FindByID는 ID로 엔드포인트를 조회합니다.
func (r *oracleEndpointRepository) FindByID(ctx context.Context, endpointID string) (*domain.APIEndpoint, error) {
	query := `
		SELECT id, name, description, base_url, health_url,
		       is_active, created_at, updated_at
		FROM api_endpoints
		WHERE id = :1
	`

	var endpoint domain.APIEndpoint
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, endpointID).Scan(
		&endpoint.ID,
		&endpoint.Name,
		&endpoint.Description,
		&endpoint.BaseURL,
		&endpoint.HealthURL,
		&endpoint.IsActive,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("endpoint with ID %s not found", endpointID)
		}
		return nil, fmt.Errorf("failed to query endpoint: %w", err)
	}

	endpoint.CreatedAt = createdAt
	endpoint.UpdatedAt = updatedAt

	return &endpoint, nil
}

// FindAll은 모든 엔드포인트를 조회합니다.
func (r *oracleEndpointRepository) FindAll(ctx context.Context) ([]*domain.APIEndpoint, error) {
	query := `
		SELECT id, name, description, base_url, health_url,
		       is_active, created_at, updated_at
		FROM api_endpoints
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []*domain.APIEndpoint
	for rows.Next() {
		var endpoint domain.APIEndpoint
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&endpoint.ID,
			&endpoint.Name,
			&endpoint.Description,
			&endpoint.BaseURL,
			&endpoint.HealthURL,
			&endpoint.IsActive,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}

		endpoint.CreatedAt = createdAt
		endpoint.UpdatedAt = updatedAt
		endpoints = append(endpoints, &endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoints: %w", err)
	}

	return endpoints, nil
}

// FindByType는 타입별로 엔드포인트를 조회합니다. (현재는 이름으로 필터링)
func (r *oracleEndpointRepository) FindByType(ctx context.Context, endpointType string) ([]*domain.APIEndpoint, error) {
	query := `
		SELECT id, name, description, base_url, health_url,
		       is_active, created_at, updated_at
		FROM api_endpoints
		WHERE name LIKE :1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, "%"+endpointType+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoints by type: %w", err)
	}
	defer rows.Close()

	var endpoints []*domain.APIEndpoint
	for rows.Next() {
		var endpoint domain.APIEndpoint
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&endpoint.ID,
			&endpoint.Name,
			&endpoint.Description,
			&endpoint.BaseURL,
			&endpoint.HealthURL,
			&endpoint.IsActive,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}

		endpoint.CreatedAt = createdAt
		endpoint.UpdatedAt = updatedAt
		endpoints = append(endpoints, &endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoints by type: %w", err)
	}

	return endpoints, nil
}

// FindActive는 활성화된 엔드포인트만 조회합니다.
func (r *oracleEndpointRepository) FindActive(ctx context.Context) ([]*domain.APIEndpoint, error) {
	query := `
		SELECT id, name, description, base_url, health_url,
		       is_active, created_at, updated_at
		FROM api_endpoints
		WHERE is_active = 1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []*domain.APIEndpoint
	for rows.Next() {
		var endpoint domain.APIEndpoint
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&endpoint.ID,
			&endpoint.Name,
			&endpoint.Description,
			&endpoint.BaseURL,
			&endpoint.HealthURL,
			&endpoint.IsActive,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan active endpoint: %w", err)
		}

		endpoint.CreatedAt = createdAt
		endpoint.UpdatedAt = updatedAt
		endpoints = append(endpoints, &endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active endpoints: %w", err)
	}

	return endpoints, nil
}

// Close는 데이터베이스 연결을 닫습니다.
func (r *oracleEndpointRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// oracleOrchestrationRepository는 OracleDB 기반 OrchestrationRepository 구현체입니다.
type oracleOrchestrationRepository struct {
	db *sql.DB
}

// NewOracleOrchestrationRepository는 새로운 Oracle 오케스트레이션 레포지토리를 생성합니다.
func NewOracleOrchestrationRepository(cfg *config.DatabaseConfig) (port.OrchestrationRepository, error) {
	// DSN 생성
	dsn := cfg.GetDSN()

	// 데이터베이스 연결
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 연결 설정
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &oracleOrchestrationRepository{db: db}, nil
}

// Create는 오케스트레이션 규칙을 생성합니다.
func (r *oracleOrchestrationRepository) Create(ctx context.Context, rule *domain.OrchestrationRule) error {
	query := `
		INSERT INTO orchestration_rules (
			id, name, description, routing_rule_id,
			legacy_endpoint_id, modern_endpoint_id,
			current_mode, transition_config,
			comparison_config, created_at, updated_at
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.RoutingRuleID,
		rule.LegacyEndpointID,
		rule.ModernEndpointID,
		string(rule.CurrentMode),
		rule.TransitionConfig, // JSON으로 저장
		rule.ComparisonConfig, // JSON으로 저장
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create orchestration rule: %w", err)
	}

	return nil
}

// Update는 오케스트레이션 규칙을 수정합니다.
func (r *oracleOrchestrationRepository) Update(ctx context.Context, rule *domain.OrchestrationRule) error {
	query := `
		UPDATE orchestration_rules SET
			name = :1,
			description = :2,
			routing_rule_id = :3,
			legacy_endpoint_id = :4,
			modern_endpoint_id = :5,
			current_mode = :6,
			transition_config = :7,
			comparison_config = :8,
			updated_at = :9
		WHERE id = :10
	`

	result, err := r.db.ExecContext(ctx, query,
		rule.Name,
		rule.Description,
		rule.RoutingRuleID,
		rule.LegacyEndpointID,
		rule.ModernEndpointID,
		string(rule.CurrentMode),
		rule.TransitionConfig,
		rule.ComparisonConfig,
		time.Now(),
		rule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update orchestration rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("orchestration rule with ID %s not found", rule.ID)
	}

	return nil
}

// Delete는 오케스트레이션 규칙을 삭제합니다.
func (r *oracleOrchestrationRepository) Delete(ctx context.Context, ruleID string) error {
	query := `DELETE FROM orchestration_rules WHERE id = :1`

	result, err := r.db.ExecContext(ctx, query, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete orchestration rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("orchestration rule with ID %s not found", ruleID)
	}

	return nil
}

// FindByID는 ID로 오케스트레이션 규칙을 조회합니다.
func (r *oracleOrchestrationRepository) FindByID(ctx context.Context, ruleID string) (*domain.OrchestrationRule, error) {
	query := `
		SELECT id, name, description, routing_rule_id,
		       legacy_endpoint_id, modern_endpoint_id,
		       current_mode, transition_config, comparison_config,
		       created_at, updated_at
		FROM orchestration_rules
		WHERE id = :1
	`

	var rule domain.OrchestrationRule
	var currentModeStr string
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, ruleID).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&rule.RoutingRuleID,
		&rule.LegacyEndpointID,
		&rule.ModernEndpointID,
		&currentModeStr,
		&rule.TransitionConfig,
		&rule.ComparisonConfig,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("orchestration rule with ID %s not found", ruleID)
		}
		return nil, fmt.Errorf("failed to query orchestration rule: %w", err)
	}

	rule.CurrentMode = domain.APIMode(currentModeStr)
	rule.CreatedAt = createdAt
	rule.UpdatedAt = updatedAt

	return &rule, nil
}

// FindByRoutingRuleID는 라우팅 규칙 ID로 오케스트레이션 규칙을 조회합니다.
func (r *oracleOrchestrationRepository) FindByRoutingRuleID(ctx context.Context, routingRuleID string) (*domain.OrchestrationRule, error) {
	query := `
		SELECT id, name, description, routing_rule_id,
		       legacy_endpoint_id, modern_endpoint_id,
		       current_mode, transition_config, comparison_config,
		       created_at, updated_at
		FROM orchestration_rules
		WHERE routing_rule_id = :1
	`

	var rule domain.OrchestrationRule
	var currentModeStr string
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, routingRuleID).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&rule.RoutingRuleID,
		&rule.LegacyEndpointID,
		&rule.ModernEndpointID,
		&currentModeStr,
		&rule.TransitionConfig,
		&rule.ComparisonConfig,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("orchestration rule for routing rule %s not found", routingRuleID)
		}
		return nil, fmt.Errorf("failed to query orchestration rule by routing rule ID: %w", err)
	}

	rule.CurrentMode = domain.APIMode(currentModeStr)
	rule.CreatedAt = createdAt
	rule.UpdatedAt = updatedAt

	return &rule, nil
}

// FindAll은 모든 오케스트레이션 규칙을 조회합니다.
func (r *oracleOrchestrationRepository) FindAll(ctx context.Context) ([]*domain.OrchestrationRule, error) {
	query := `
		SELECT id, name, description, routing_rule_id,
		       legacy_endpoint_id, modern_endpoint_id,
		       current_mode, transition_config, comparison_config,
		       created_at, updated_at
		FROM orchestration_rules
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query orchestration rules: %w", err)
	}
	defer rows.Close()

	var rules []*domain.OrchestrationRule
	for rows.Next() {
		var rule domain.OrchestrationRule
		var currentModeStr string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&rule.RoutingRuleID,
			&rule.LegacyEndpointID,
			&rule.ModernEndpointID,
			&currentModeStr,
			&rule.TransitionConfig,
			&rule.ComparisonConfig,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan orchestration rule: %w", err)
		}

		rule.CurrentMode = domain.APIMode(currentModeStr)
		rule.CreatedAt = createdAt
		rule.UpdatedAt = updatedAt
		rules = append(rules, &rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orchestration rules: %w", err)
	}

	return rules, nil
}

// FindActive는 활성화된 오케스트레이션 규칙만 조회합니다.
func (r *oracleOrchestrationRepository) FindActive(ctx context.Context) ([]*domain.OrchestrationRule, error) {
	// 현재는 모든 규칙을 활성으로 간주 (필요시 활성화 플래그 추가)
	return r.FindAll(ctx)
}

// Close는 데이터베이스 연결을 닫습니다.
func (r *oracleOrchestrationRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// oracleComparisonRepository는 OracleDB 기반 ComparisonRepository 구현체입니다.
type oracleComparisonRepository struct {
	db *sql.DB
}

// NewOracleComparisonRepository는 새로운 Oracle 비교 레포지토리를 생성합니다.
func NewOracleComparisonRepository(cfg *config.DatabaseConfig) (port.ComparisonRepository, error) {
	// DSN 생성
	dsn := cfg.GetDSN()

	// 데이터베이스 연결
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 연결 설정
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &oracleComparisonRepository{db: db}, nil
}

// SaveComparison은 API 비교 결과를 저장합니다.
func (r *oracleComparisonRepository) SaveComparison(ctx context.Context, comparison *domain.APIComparison) error {
	query := `
		INSERT INTO api_comparisons (
			id, request_id, routing_rule_id,
			legacy_response, modern_response,
			match_rate, differences, comparison_duration,
			created_at
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9
		)
	`

	// 응답을 JSON으로 직렬화
	legacyResponseJSON, err := comparison.LegacyResponse.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize legacy response: %w", err)
	}

	modernResponseJSON, err := comparison.ModernResponse.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize modern response: %w", err)
	}

	differencesJSON, err := comparison.DifferencesToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize differences: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		comparison.ID,
		comparison.RequestID,
		comparison.RoutingRuleID,
		legacyResponseJSON,
		modernResponseJSON,
		comparison.MatchRate,
		differencesJSON,
		comparison.ComparisonDuration.Milliseconds(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save comparison: %w", err)
	}

	return nil
}

// GetRecentComparisons는 최근 비교 결과들을 조회합니다.
func (r *oracleComparisonRepository) GetRecentComparisons(ctx context.Context, routingRuleID string, limit int) ([]*domain.APIComparison, error) {
	query := `
		SELECT id, request_id, routing_rule_id,
		       legacy_response, modern_response,
		       match_rate, differences, comparison_duration,
		       created_at
		FROM api_comparisons
		WHERE routing_rule_id = :1
		ORDER BY created_at DESC
		FETCH FIRST :2 ROWS ONLY
	`

	rows, err := r.db.QueryContext(ctx, query, routingRuleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent comparisons: %w", err)
	}
	defer rows.Close()

	var comparisons []*domain.APIComparison
	for rows.Next() {
		var comparison domain.APIComparison
		var legacyResponseJSON, modernResponseJSON, differencesJSON string
		var comparisonDurationMs int64
		var createdAt time.Time

		err := rows.Scan(
			&comparison.ID,
			&comparison.RequestID,
			&comparison.RoutingRuleID,
			&legacyResponseJSON,
			&modernResponseJSON,
			&comparison.MatchRate,
			&differencesJSON,
			&comparisonDurationMs,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan comparison: %w", err)
		}

		// JSON을 도메인 객체로 역직렬화
		if legacyResponseJSON != "" {
			comparison.LegacyResponse, err = domain.ResponseFromJSON([]byte(legacyResponseJSON))
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize legacy response: %w", err)
			}
		}

		if modernResponseJSON != "" {
			comparison.ModernResponse, err = domain.ResponseFromJSON([]byte(modernResponseJSON))
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize modern response: %w", err)
			}
		}

		comparison.Differences, err = domain.ResponseDiffsFromJSON([]byte(differencesJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize differences: %w", err)
		}

		comparison.ComparisonDuration = time.Duration(comparisonDurationMs) * time.Millisecond
		comparison.CreatedAt = createdAt

		comparisons = append(comparisons, &comparison)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comparisons: %w", err)
	}

	return comparisons, nil
}

// GetComparisonStatistics는 비교 통계를 조회합니다.
func (r *oracleComparisonRepository) GetComparisonStatistics(ctx context.Context, routingRuleID string, from, to time.Time) (*port.ComparisonStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total_comparisons,
			COUNT(CASE WHEN match_rate >= 0.95 THEN 1 END) as successful_matches,
			AVG(match_rate) as average_match_rate,
			MAX(created_at) as last_comparison
		FROM api_comparisons
		WHERE routing_rule_id = :1
		AND created_at BETWEEN :2 AND :3
	`

	var stats port.ComparisonStatistics
	stats.RoutingRuleID = routingRuleID

	var totalComparisons, successfulMatches int
	var averageMatchRate sql.NullFloat64
	var lastComparison sql.NullTime

	err := r.db.QueryRowContext(ctx, query, routingRuleID, from, to).Scan(
		&totalComparisons,
		&successfulMatches,
		&averageMatchRate,
		&lastComparison,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query comparison statistics: %w", err)
	}

	stats.TotalComparisons = totalComparisons
	stats.SuccessfulMatches = successfulMatches

	if averageMatchRate.Valid {
		stats.AverageMatchRate = averageMatchRate.Float64
	}

	if lastComparison.Valid {
		stats.LastComparison = lastComparison.Time
	}

	return &stats, nil
}

// Close는 데이터베이스 연결을 닫습니다.
func (r *oracleComparisonRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
