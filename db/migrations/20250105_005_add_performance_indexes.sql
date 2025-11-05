-- +migrate Up
-- 복합 인덱스 추가 (성능 최적화)
CREATE INDEX idx_routing_path_method ON routing_rules(request_path, method, is_active);
CREATE INDEX idx_ep_url_method ON api_endpoints(base_url, method, is_active);

-- 통계 정보 수집 (Oracle Optimizer)
BEGIN
    DBMS_STATS.GATHER_TABLE_STATS(USER, 'ROUTING_RULES');
    DBMS_STATS.GATHER_TABLE_STATS(USER, 'API_ENDPOINTS');
    DBMS_STATS.GATHER_TABLE_STATS(USER, 'ORCHESTRATION_RULES');
    DBMS_STATS.GATHER_TABLE_STATS(USER, 'COMPARISON_LOGS');
END;
/

-- +migrate Down
DROP INDEX idx_routing_path_method;
DROP INDEX idx_ep_url_method;
