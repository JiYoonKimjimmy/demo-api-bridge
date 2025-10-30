#!/bin/bash

# API Bridge CRUD API 테스트 스크립트
# 서버가 실행 중이어야 합니다 (포트 10019)

BASE_URL="http://localhost:10019/abs/v1"

echo "🚀 API Bridge CRUD API 테스트 시작"
echo "=================================="

# 1. 엔드포인트 생성 테스트
echo "📝 1. 엔드포인트 생성 테스트"
echo "POST $BASE_URL/endpoints"

LEGACY_ENDPOINT_RESPONSE=$(curl -s -X POST "$BASE_URL/endpoints" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Legacy User API",
    "description": "레거시 사용자 API",
    "base_url": "https://legacy-api.example.com",
    "path": "/api/v1/users",
    "health_url": "https://legacy-api.example.com/health",
    "method": "GET",
    "is_active": true,
    "timeout": 30000,
    "retry_count": 3,
    "priority": 1
  }')

echo "레거시 엔드포인트 생성 응답:"
echo "$LEGACY_ENDPOINT_RESPONSE" | jq '.' 2>/dev/null || echo "$LEGACY_ENDPOINT_RESPONSE"
echo ""

MODERN_ENDPOINT_RESPONSE=$(curl -s -X POST "$BASE_URL/endpoints" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Modern User API",
    "description": "모던 사용자 API",
    "base_url": "https://modern-api.example.com",
    "path": "/api/v2/users",
    "health_url": "https://modern-api.example.com/health",
    "method": "GET",
    "is_active": true,
    "timeout": 30000,
    "retry_count": 3,
    "priority": 2
  }')

echo "모던 엔드포인트 생성 응답:"
echo "$MODERN_ENDPOINT_RESPONSE" | jq '.' 2>/dev/null || echo "$MODERN_ENDPOINT_RESPONSE"
echo ""

# 엔드포인트 ID 추출 (jq 사용 또는 첫 번째 ID만 추출)
LEGACY_ENDPOINT_ID=$(echo "$LEGACY_ENDPOINT_RESPONSE" | jq -r '.id' 2>/dev/null || echo "$LEGACY_ENDPOINT_RESPONSE" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
MODERN_ENDPOINT_ID=$(echo "$MODERN_ENDPOINT_RESPONSE" | jq -r '.id' 2>/dev/null || echo "$MODERN_ENDPOINT_RESPONSE" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)

echo "추출된 ID들:"
echo "레거시 엔드포인트 ID: $LEGACY_ENDPOINT_ID"
echo "모던 엔드포인트 ID: $MODERN_ENDPOINT_ID"
echo ""

# 2. 엔드포인트 목록 조회 테스트
echo "📋 2. 엔드포인트 목록 조회 테스트"
echo "GET $BASE_URL/endpoints"

ENDPOINTS_RESPONSE=$(curl -s -X GET "$BASE_URL/endpoints")
echo "엔드포인트 목록:"
echo "$ENDPOINTS_RESPONSE" | jq '.' 2>/dev/null || echo "$ENDPOINTS_RESPONSE"
echo ""

# 3. 라우팅 규칙 생성 테스트
echo "🛣️ 3. 라우팅 규칙 생성 테스트"
echo "POST $BASE_URL/routing-rules"

ROUTING_RULE_RESPONSE=$(curl -s -X POST "$BASE_URL/routing-rules" \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "name": "User API Routing Rule",
  "description": "사용자 API 라우팅 규칙",
  "path_pattern": "/api/users/*",
  "method": "GET",
  "priority": 1,
  "is_active": true,
  "legacy_endpoint": {
    "id": "$LEGACY_ENDPOINT_ID"
  },
  "modern_endpoint": {
    "id": "$MODERN_ENDPOINT_ID"
  }
}
EOF
)

echo "라우팅 규칙 생성 응답:"
echo "$ROUTING_RULE_RESPONSE" | jq '.' 2>/dev/null || echo "$ROUTING_RULE_RESPONSE"
echo ""

# 라우팅 규칙 ID 추출 (jq 사용 또는 첫 번째 ID만 추출)
ROUTING_RULE_ID=$(echo "$ROUTING_RULE_RESPONSE" | jq -r '.id' 2>/dev/null || echo "$ROUTING_RULE_RESPONSE" | grep -o '"id":"[^"]*"' | head -n1 | cut -d'"' -f4)
echo "라우팅 규칙 ID: $ROUTING_RULE_ID"
echo ""

# 4. 오케스트레이션 규칙 생성 테스트
echo "🎭 4. 오케스트레이션 규칙 생성 테스트"
echo "POST $BASE_URL/orchestration-rules"

ORCHESTRATION_RULE_RESPONSE=$(curl -s -X POST "$BASE_URL/orchestration-rules" \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "name": "User API Orchestration",
  "description": "사용자 API 오케스트레이션 규칙",
  "routing_rule_id": "$ROUTING_RULE_ID",
  "legacy_endpoint": {
    "id": "$LEGACY_ENDPOINT_ID"
  },
  "modern_endpoint": {
    "id": "$MODERN_ENDPOINT_ID"
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
EOF
)

echo "오케스트레이션 규칙 생성 응답:"
echo "$ORCHESTRATION_RULE_RESPONSE" | jq '.' 2>/dev/null || echo "$ORCHESTRATION_RULE_RESPONSE"
echo ""

# 5. 오케스트레이션 규칙 조회 테스트
echo "🔍 5. 오케스트레이션 규칙 조회 테스트"
echo "GET $BASE_URL/orchestration-rules/$ROUTING_RULE_ID"

ORCHESTRATION_GET_RESPONSE=$(curl -s -X GET "$BASE_URL/orchestration-rules/$ROUTING_RULE_ID")
echo "오케스트레이션 규칙 조회 응답:"
echo "$ORCHESTRATION_GET_RESPONSE" | jq '.' 2>/dev/null || echo "$ORCHESTRATION_GET_RESPONSE"
echo ""

# 6. 전환 평가 테스트
echo "⚖️ 6. 전환 평가 테스트"
echo "GET $BASE_URL/orchestration-rules/$ROUTING_RULE_ID/evaluate-transition"

EVALUATE_RESPONSE=$(curl -s -X GET "$BASE_URL/orchestration-rules/$ROUTING_RULE_ID/evaluate-transition")
echo "전환 평가 응답:"
echo "$EVALUATE_RESPONSE" | jq '.' 2>/dev/null || echo "$EVALUATE_RESPONSE"
echo ""

# 7. 전환 실행 테스트 (PARALLEL -> MODERN_ONLY)
echo "🔄 7. 전환 실행 테스트 (PARALLEL -> MODERN_ONLY)"
echo "POST $BASE_URL/orchestration-rules/$ROUTING_RULE_ID/execute-transition"

EXECUTE_RESPONSE=$(curl -s -X POST "$BASE_URL/orchestration-rules/$ROUTING_RULE_ID/execute-transition" \
  -H "Content-Type: application/json" \
  -d '{
    "new_mode": "MODERN_ONLY"
  }')

echo "전환 실행 응답:"
echo "$EXECUTE_RESPONSE" | jq '.' 2>/dev/null || echo "$EXECUTE_RESPONSE"
echo ""

echo "✅ CRUD API 테스트 완료!"
echo "=================================="
