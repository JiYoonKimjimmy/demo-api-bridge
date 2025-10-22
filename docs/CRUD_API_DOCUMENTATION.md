# API Bridge CRUD API 문서

## 개요

API Bridge 시스템의 각 모델(APIEndpoint, RoutingRule, OrchestrationRule)에 대한 완전한 CRUD API가 구현되었습니다.

## 기본 정보

- **Base URL**: `http://localhost:10019/api/v1`
- **Content-Type**: `application/json`
- **인증**: 현재 인증 없음 (향후 추가 예정)

## API 엔드포인트

### 1. APIEndpoint CRUD

#### 엔드포인트 생성
```http
POST /api/v1/endpoints
Content-Type: application/json

{
  "name": "Legacy User API",
  "description": "레거시 사용자 API",
  "base_url": "https://legacy-api.example.com",
  "health_url": "https://legacy-api.example.com/health",
  "is_active": true,
  "timeout": 30000,
  "retry_count": 3
}
```

**응답:**
```json
{
  "id": "endpoint-20250121123456-abc123",
  "name": "Legacy User API",
  "description": "레거시 사용자 API",
  "base_url": "https://legacy-api.example.com",
  "health_url": "https://legacy-api.example.com/health",
  "is_active": true,
  "timeout": 30000,
  "retry_count": 3,
  "created_at": "2025-01-21T12:34:56Z",
  "updated_at": "2025-01-21T12:34:56Z"
}
```

#### 엔드포인트 목록 조회
```http
GET /api/v1/endpoints
```

**응답:**
```json
{
  "endpoints": [
    {
      "id": "endpoint-20250121123456-abc123",
      "name": "Legacy User API",
      "description": "레거시 사용자 API",
      "base_url": "https://legacy-api.example.com",
      "health_url": "https://legacy-api.example.com/health",
      "is_active": true,
      "timeout": 30000,
      "retry_count": 3,
      "created_at": "2025-01-21T12:34:56Z",
      "updated_at": "2025-01-21T12:34:56Z"
    }
  ],
  "count": 1
}
```

#### 엔드포인트 조회
```http
GET /api/v1/endpoints/{id}
```

#### 엔드포인트 수정
```http
PUT /api/v1/endpoints/{id}
Content-Type: application/json

{
  "name": "Updated Legacy User API",
  "is_active": false,
  "timeout": 60000
}
```

#### 엔드포인트 삭제
```http
DELETE /api/v1/endpoints/{id}
```

### 2. RoutingRule CRUD

#### 라우팅 규칙 생성
```http
POST /api/v1/routing-rules
Content-Type: application/json

{
  "name": "User API Routing Rule",
  "description": "사용자 API 라우팅 규칙",
  "path_pattern": "/api/users/*",
  "method": "GET",
  "priority": 1,
  "is_active": true,
  "legacy_endpoint": {
    "id": "endpoint-legacy-id"
  },
  "modern_endpoint": {
    "id": "endpoint-modern-id"
  },
  "headers": {
    "Content-Type": "application/json"
  },
  "query_params": {
    "version": "v1"
  }
}
```

**응답:**
```json
{
  "id": "rule-20250121123456-def456",
  "name": "User API Routing Rule",
  "description": "사용자 API 라우팅 규칙",
  "path_pattern": "/api/users/*",
  "method": "GET",
  "priority": 1,
  "is_active": true,
  "legacy_endpoint": {
    "id": "endpoint-legacy-id"
  },
  "modern_endpoint": {
    "id": "endpoint-modern-id"
  },
  "headers": {
    "Content-Type": "application/json"
  },
  "query_params": {
    "version": "v1"
  },
  "created_at": "2025-01-21T12:34:56Z",
  "updated_at": "2025-01-21T12:34:56Z"
}
```

#### 라우팅 규칙 목록 조회
```http
GET /api/v1/routing-rules
```

#### 라우팅 규칙 조회
```http
GET /api/v1/routing-rules/{id}
```

#### 라우팅 규칙 수정
```http
PUT /api/v1/routing-rules/{id}
Content-Type: application/json

{
  "name": "Updated User API Routing Rule",
  "priority": 2,
  "is_active": false
}
```

#### 라우팅 규칙 삭제
```http
DELETE /api/v1/routing-rules/{id}
```

### 3. OrchestrationRule CRUD

#### 오케스트레이션 규칙 생성
```http
POST /api/v1/orchestration-rules
Content-Type: application/json

{
  "name": "User API Orchestration",
  "description": "사용자 API 오케스트레이션 규칙",
  "routing_rule_id": "rule-id",
  "legacy_endpoint": {
    "id": "endpoint-legacy-id"
  },
  "modern_endpoint": {
    "id": "endpoint-modern-id"
  },
  "current_mode": "PARALLEL",
  "is_active": true,
  "transition_config": {
    "auto_transition_enabled": true,
    "match_rate_threshold": 0.95,
    "stability_period_hours": 24,
    "min_requests_for_transition": 100,
    "rollback_threshold": 0.90
  },
  "comparison_config": {
    "enabled": true,
    "ignore_fields": ["timestamp", "requestId"],
    "allowable_difference": 0.01,
    "strict_mode": false,
    "save_comparison_history": true
  }
}
```

**응답:**
```json
{
  "id": "orch-20250121123456-ghi789",
  "name": "User API Orchestration",
  "description": "사용자 API 오케스트레이션 규칙",
  "routing_rule_id": "rule-id",
  "legacy_endpoint": {
    "id": "endpoint-legacy-id"
  },
  "modern_endpoint": {
    "id": "endpoint-modern-id"
  },
  "current_mode": "PARALLEL",
  "is_active": true,
  "transition_config": {
    "auto_transition_enabled": true,
    "match_rate_threshold": 0.95,
    "stability_period_hours": 24,
    "min_requests_for_transition": 100,
    "rollback_threshold": 0.90
  },
  "comparison_config": {
    "enabled": true,
    "ignore_fields": ["timestamp", "requestId"],
    "allowable_difference": 0.01,
    "strict_mode": false,
    "save_comparison_history": true
  },
  "created_at": "2025-01-21T12:34:56Z",
  "updated_at": "2025-01-21T12:34:56Z"
}
```

#### 오케스트레이션 규칙 조회
```http
GET /api/v1/orchestration-rules/{routing_rule_id}
```

#### 오케스트레이션 규칙 수정
```http
PUT /api/v1/orchestration-rules/{routing_rule_id}
Content-Type: application/json

{
  "name": "Updated User API Orchestration",
  "current_mode": "MODERN_ONLY",
  "is_active": false
}
```

### 4. 전환 관련 API

#### 전환 가능성 평가
```http
GET /api/v1/orchestration-rules/{routing_rule_id}/evaluate-transition
```

**응답:**
```json
{
  "can_transition": true,
  "current_mode": "PARALLEL",
  "rule_id": "orch-20250121123456-ghi789"
}
```

#### 전환 실행
```http
POST /api/v1/orchestration-rules/{routing_rule_id}/execute-transition
Content-Type: application/json

{
  "new_mode": "MODERN_ONLY"
}
```

**응답:**
```json
{
  "message": "transition executed successfully",
  "from_mode": "PARALLEL",
  "to_mode": "MODERN_ONLY",
  "rule_id": "orch-20250121123456-ghi789"
}
```

## API 모드

- **LEGACY_ONLY**: 레거시 API만 호출
- **MODERN_ONLY**: 모던 API만 호출
- **PARALLEL**: 레거시와 모던 API를 병렬로 호출하고 결과 비교

## 에러 응답

모든 API는 일관된 에러 응답 형식을 사용합니다:

```json
{
  "error": "에러 메시지",
  "details": "상세 에러 정보"
}
```

**HTTP 상태 코드:**
- `200`: 성공
- `201`: 생성 성공
- `204`: 삭제 성공 (내용 없음)
- `400`: 잘못된 요청
- `404`: 리소스를 찾을 수 없음
- `500`: 내부 서버 오류

## 테스트

테스트 스크립트를 실행하여 모든 CRUD API를 테스트할 수 있습니다:

```bash
# 서버가 실행 중인 상태에서
./test_crud_api.sh
```

## 사용 예시

1. **엔드포인트 생성**: 레거시와 모던 시스템의 API 엔드포인트를 등록
2. **라우팅 규칙 생성**: 요청 패턴과 엔드포인트를 매핑
3. **오케스트레이션 규칙 생성**: 병렬 호출 및 전환 정책 설정
4. **전환 평가**: 모던 시스템으로 전환 가능 여부 확인
5. **전환 실행**: 안전하게 모던 시스템으로 전환

이러한 API를 통해 API Bridge 시스템의 모든 설정을 동적으로 관리할 수 있습니다.
