-- API Bridge Database Schema Rollback
-- Oracle Database DDL Rollback for API Bridge Service
-- 외래키 참조 순서를 고려하여 역순으로 DROP

-- 1. Drop Orchestration Rules Table (외래키가 가장 많음)
DROP TABLE MAP.ABS_ORCHESTRATION_RULES CASCADE CONSTRAINTS;

-- 2. Drop API Comparisons Table
DROP TABLE MAP.ABS_API_COMPARISONS CASCADE CONSTRAINTS;

-- 3. Drop Routing Rules Table
DROP TABLE MAP.ABS_ROUTING_RULES CASCADE CONSTRAINTS;

-- 4. Drop API Endpoints Table (기본 테이블)
DROP TABLE MAP.ABS_API_ENDPOINTS CASCADE CONSTRAINTS;

-- Note: 인덱스와 코멘트는 테이블 DROP 시 자동으로 삭제됩니다.
