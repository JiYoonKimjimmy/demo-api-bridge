-- +migrate Up
-- APIEndpoint 테이블 생성
CREATE TABLE api_endpoints (
    id VARCHAR2(36) PRIMARY KEY,
    name VARCHAR2(100) NOT NULL,
    base_url VARCHAR2(500) NOT NULL,
    path VARCHAR2(500),
    method VARCHAR2(10) NOT NULL,
    timeout_ms NUMBER(10) DEFAULT 5000,
    retry_count NUMBER(3) DEFAULT 3,
    headers CLOB,
    is_active NUMBER(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_ep_method CHECK (method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')),
    CONSTRAINT chk_ep_is_active CHECK (is_active IN (0, 1))
);

-- 인덱스 생성
CREATE INDEX idx_ep_name ON api_endpoints(name);
CREATE INDEX idx_ep_active ON api_endpoints(is_active);

-- 코멘트 추가
COMMENT ON TABLE api_endpoints IS '외부 API 엔드포인트 정보';
COMMENT ON COLUMN api_endpoints.headers IS 'JSON 형식의 HTTP 헤더';

-- +migrate Down
DROP TABLE api_endpoints CASCADE CONSTRAINTS;
