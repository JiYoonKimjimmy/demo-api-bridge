-- +migrate Up
-- ComparisonLog 테이블 생성
CREATE TABLE comparison_logs (
    id VARCHAR2(36) PRIMARY KEY,
    routing_rule_id VARCHAR2(36) NOT NULL,
    request_id VARCHAR2(100),
    old_response CLOB,
    new_response CLOB,
    is_matched NUMBER(1) DEFAULT 0,
    difference_details CLOB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_cmp_routing FOREIGN KEY (routing_rule_id) REFERENCES routing_rules(id) ON DELETE CASCADE,
    CONSTRAINT chk_cmp_is_matched CHECK (is_matched IN (0, 1))
);

-- 인덱스 생성
CREATE INDEX idx_cmp_routing ON comparison_logs(routing_rule_id);
CREATE INDEX idx_cmp_created ON comparison_logs(created_at);
CREATE INDEX idx_cmp_matched ON comparison_logs(is_matched);

-- 코멘트 추가
COMMENT ON TABLE comparison_logs IS 'API 응답 비교 로그';

-- +migrate Down
DROP TABLE comparison_logs CASCADE CONSTRAINTS;
