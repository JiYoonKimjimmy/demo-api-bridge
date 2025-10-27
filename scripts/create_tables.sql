-- API Bridge Database Schema
-- Oracle Database DDL for API Bridge Service

-- 1. API Endpoints Table
CREATE TABLE api_endpoints (
    id VARCHAR2(100) PRIMARY KEY,
    name VARCHAR2(200) NOT NULL,
    description VARCHAR2(500),
    base_url VARCHAR2(500) NOT NULL,
    health_url VARCHAR2(500),
    is_active NUMBER(1) DEFAULT 1 NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT chk_endpoints_is_active CHECK (is_active IN (0, 1))
);

-- 2. Routing Rules Table
CREATE TABLE routing_rules (
    id VARCHAR2(100) PRIMARY KEY,
    name VARCHAR2(200) NOT NULL,
    description VARCHAR2(500),
    method VARCHAR2(10),
    path_pattern VARCHAR2(500) NOT NULL,
    headers CLOB,
    query_params CLOB,
    legacy_endpoint_id VARCHAR2(100),
    modern_endpoint_id VARCHAR2(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_routing_legacy FOREIGN KEY (legacy_endpoint_id)
        REFERENCES api_endpoints(id) ON DELETE SET NULL,
    CONSTRAINT fk_routing_modern FOREIGN KEY (modern_endpoint_id)
        REFERENCES api_endpoints(id) ON DELETE SET NULL
);

-- 3. Orchestration Rules Table
CREATE TABLE orchestration_rules (
    id VARCHAR2(100) PRIMARY KEY,
    name VARCHAR2(200) NOT NULL,
    description VARCHAR2(500),
    routing_rule_id VARCHAR2(100) NOT NULL,
    legacy_endpoint_id VARCHAR2(100) NOT NULL,
    modern_endpoint_id VARCHAR2(100) NOT NULL,
    current_mode VARCHAR2(20) DEFAULT 'PARALLEL' NOT NULL,
    transition_config CLOB,
    comparison_config CLOB,
    is_active NUMBER(1) DEFAULT 1 NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT chk_orchestration_mode CHECK (
        current_mode IN ('LEGACY_ONLY', 'PARALLEL', 'MODERN_ONLY')
    ),
    CONSTRAINT chk_orchestration_active CHECK (is_active IN (0, 1)),
    CONSTRAINT fk_orchestration_routing FOREIGN KEY (routing_rule_id)
        REFERENCES routing_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_orchestration_legacy FOREIGN KEY (legacy_endpoint_id)
        REFERENCES api_endpoints(id) ON DELETE CASCADE,
    CONSTRAINT fk_orchestration_modern FOREIGN KEY (modern_endpoint_id)
        REFERENCES api_endpoints(id) ON DELETE CASCADE
);

-- 4. API Comparisons Table (Comparison History)
CREATE TABLE api_comparisons (
    id VARCHAR2(100) PRIMARY KEY,
    request_id VARCHAR2(100) NOT NULL,
    routing_rule_id VARCHAR2(100),
    match_rate NUMBER(5,4) DEFAULT 0,
    differences CLOB,
    comparison_duration_ms NUMBER(10),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_comparison_routing FOREIGN KEY (routing_rule_id)
        REFERENCES routing_rules(id) ON DELETE SET NULL
);

-- Indexes for Performance
CREATE INDEX idx_endpoints_active ON api_endpoints(is_active);
CREATE INDEX idx_endpoints_created ON api_endpoints(created_at);

CREATE INDEX idx_routing_path ON routing_rules(path_pattern);
CREATE INDEX idx_routing_method ON routing_rules(method);
CREATE INDEX idx_routing_legacy ON routing_rules(legacy_endpoint_id);
CREATE INDEX idx_routing_modern ON routing_rules(modern_endpoint_id);

CREATE INDEX idx_orchestration_routing ON orchestration_rules(routing_rule_id);
CREATE INDEX idx_orchestration_mode ON orchestration_rules(current_mode);
CREATE INDEX idx_orchestration_active ON orchestration_rules(is_active);

CREATE INDEX idx_comparison_request ON api_comparisons(request_id);
CREATE INDEX idx_comparison_routing ON api_comparisons(routing_rule_id);
CREATE INDEX idx_comparison_timestamp ON api_comparisons(timestamp);
CREATE INDEX idx_comparison_match_rate ON api_comparisons(match_rate);

-- Comments
COMMENT ON TABLE api_endpoints IS 'API 엔드포인트 정보';
COMMENT ON TABLE routing_rules IS 'API 라우팅 규칙';
COMMENT ON TABLE orchestration_rules IS 'API 오케스트레이션 규칙';
COMMENT ON TABLE api_comparisons IS 'API 응답 비교 이력';

COMMENT ON COLUMN api_endpoints.id IS '엔드포인트 고유 ID';
COMMENT ON COLUMN api_endpoints.name IS '엔드포인트 이름';
COMMENT ON COLUMN api_endpoints.base_url IS '기본 URL';
COMMENT ON COLUMN api_endpoints.health_url IS '헬스 체크 URL';
COMMENT ON COLUMN api_endpoints.is_active IS '활성화 여부 (0: 비활성, 1: 활성)';

COMMENT ON COLUMN routing_rules.path_pattern IS '경로 패턴 (예: /api/users/*)';
COMMENT ON COLUMN routing_rules.method IS 'HTTP 메서드';
COMMENT ON COLUMN routing_rules.legacy_endpoint_id IS '레거시 엔드포인트 ID';
COMMENT ON COLUMN routing_rules.modern_endpoint_id IS '모던 엔드포인트 ID';

COMMENT ON COLUMN orchestration_rules.current_mode IS '현재 API 모드 (LEGACY_ONLY, PARALLEL, MODERN_ONLY)';
COMMENT ON COLUMN orchestration_rules.transition_config IS '전환 설정 (JSON)';
COMMENT ON COLUMN orchestration_rules.comparison_config IS '비교 설정 (JSON)';

COMMENT ON COLUMN api_comparisons.match_rate IS '일치율 (0.0 ~ 1.0)';
COMMENT ON COLUMN api_comparisons.differences IS '차이점 목록 (JSON)';
COMMENT ON COLUMN api_comparisons.comparison_duration_ms IS '비교 소요 시간 (밀리초)';
