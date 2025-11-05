-- +migrate Up
-- RoutingRule 테이블 생성
CREATE TABLE routing_rules (
    id VARCHAR2(36) PRIMARY KEY,
    endpoint_id VARCHAR2(36) NOT NULL,
    request_path VARCHAR2(500) NOT NULL,
    method VARCHAR2(10) NOT NULL,
    strategy VARCHAR2(50) NOT NULL,
    priority NUMBER(10) DEFAULT 0,
    is_active NUMBER(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_method CHECK (method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')),
    CONSTRAINT chk_strategy CHECK (strategy IN ('direct', 'orchestration', 'comparison', 'ab_test')),
    CONSTRAINT chk_is_active CHECK (is_active IN (0, 1))
);

-- 인덱스 생성
CREATE INDEX idx_routing_path ON routing_rules(request_path);
CREATE INDEX idx_routing_endpoint ON routing_rules(endpoint_id);
CREATE INDEX idx_routing_active ON routing_rules(is_active);

-- 코멘트 추가
COMMENT ON TABLE routing_rules IS 'API 라우팅 규칙 관리';
COMMENT ON COLUMN routing_rules.strategy IS 'direct: 단일 전달, orchestration: 오케스트레이션, comparison: AB 비교';

-- +migrate Down
-- 테이블 삭제 (롤백)
DROP TABLE routing_rules CASCADE CONSTRAINTS;
