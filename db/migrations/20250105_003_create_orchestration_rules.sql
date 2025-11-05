-- +migrate Up
-- OrchestrationRule 테이블 생성
CREATE TABLE orchestration_rules (
    id VARCHAR2(36) PRIMARY KEY,
    routing_rule_id VARCHAR2(36) NOT NULL,
    name VARCHAR2(100) NOT NULL,
    execution_type VARCHAR2(20) DEFAULT 'sequential',
    steps CLOB NOT NULL,
    is_active NUMBER(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_orc_routing FOREIGN KEY (routing_rule_id) REFERENCES routing_rules(id) ON DELETE CASCADE,
    CONSTRAINT chk_exec_type CHECK (execution_type IN ('sequential', 'parallel')),
    CONSTRAINT chk_orc_is_active CHECK (is_active IN (0, 1))
);

-- 인덱스 생성
CREATE INDEX idx_orc_routing ON orchestration_rules(routing_rule_id);
CREATE INDEX idx_orc_name ON orchestration_rules(name);

-- 코멘트 추가
COMMENT ON TABLE orchestration_rules IS 'API 오케스트레이션 규칙';
COMMENT ON COLUMN orchestration_rules.steps IS 'JSON 배열 형식의 실행 스텝';

-- +migrate Down
DROP TABLE orchestration_rules CASCADE CONSTRAINTS;
