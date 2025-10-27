# API Bridge CRUD API 테스트 스크립트
# 서버가 실행 중이어야 합니다 (포트 10019)

$BaseUrl = "http://localhost:10019/api/v1"

Write-Host "🚀 API Bridge CRUD API 테스트 시작" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# 1. 엔드포인트 생성 테스트
Write-Host "📝 1. 엔드포인트 생성 테스트" -ForegroundColor Yellow
Write-Host "POST $BaseUrl/endpoints" -ForegroundColor Gray

$legacyEndpointBody = @{
    name = "Legacy User API"
    description = "레거시 사용자 API"
    base_url = "https://legacy-api.example.com"
    path = "/api/v1/users"
    health_url = "https://legacy-api.example.com/health"
    method = "GET"
    is_active = $true
    timeout = 30000
    retry_count = 3
    priority = 1
} | ConvertTo-Json

try {
    $legacyEndpointResponse = Invoke-RestMethod -Uri "$BaseUrl/endpoints" `
        -Method Post `
        -ContentType "application/json" `
        -Body $legacyEndpointBody

    Write-Host "레거시 엔드포인트 생성 응답:" -ForegroundColor Green
    $legacyEndpointResponse | ConvertTo-Json -Depth 10
    Write-Host ""
} catch {
    Write-Host "에러 발생: $_" -ForegroundColor Red
    exit 1
}

$modernEndpointBody = @{
    name = "Modern User API"
    description = "모던 사용자 API"
    base_url = "https://modern-api.example.com"
    path = "/api/v2/users"
    health_url = "https://modern-api.example.com/health"
    method = "GET"
    is_active = $true
    timeout = 30000
    retry_count = 3
    priority = 2
} | ConvertTo-Json

try {
    $modernEndpointResponse = Invoke-RestMethod -Uri "$BaseUrl/endpoints" `
        -Method Post `
        -ContentType "application/json" `
        -Body $modernEndpointBody

    Write-Host "모던 엔드포인트 생성 응답:" -ForegroundColor Green
    $modernEndpointResponse | ConvertTo-Json -Depth 10
    Write-Host ""
} catch {
    Write-Host "에러 발생: $_" -ForegroundColor Red
    exit 1
}

# 엔드포인트 ID 추출
$legacyEndpointId = $legacyEndpointResponse.id
$modernEndpointId = $modernEndpointResponse.id

Write-Host "추출된 ID들:" -ForegroundColor Cyan
Write-Host "레거시 엔드포인트 ID: $legacyEndpointId" -ForegroundColor White
Write-Host "모던 엔드포인트 ID: $modernEndpointId" -ForegroundColor White
Write-Host ""

# 2. 엔드포인트 목록 조회 테스트
Write-Host "📋 2. 엔드포인트 목록 조회 테스트" -ForegroundColor Yellow
Write-Host "GET $BaseUrl/endpoints" -ForegroundColor Gray

try {
    $endpointsResponse = Invoke-RestMethod -Uri "$BaseUrl/endpoints" -Method Get
    Write-Host "엔드포인트 목록:" -ForegroundColor Green
    $endpointsResponse | ConvertTo-Json -Depth 10
    Write-Host ""
} catch {
    Write-Host "에러 발생: $_" -ForegroundColor Red
    exit 1
}

# 3. 라우팅 규칙 생성 테스트
Write-Host "🛣️ 3. 라우팅 규칙 생성 테스트" -ForegroundColor Yellow
Write-Host "POST $BaseUrl/routing-rules" -ForegroundColor Gray

$routingRuleBody = @{
    name = "User API Routing Rule"
    description = "사용자 API 라우팅 규칙"
    path_pattern = "/api/users/*"
    method = "GET"
    priority = 1
    is_active = $true
    legacy_endpoint = @{
        id = $legacyEndpointId
    }
    modern_endpoint = @{
        id = $modernEndpointId
    }
} | ConvertTo-Json

try {
    $routingRuleResponse = Invoke-RestMethod -Uri "$BaseUrl/routing-rules" `
        -Method Post `
        -ContentType "application/json" `
        -Body $routingRuleBody

    Write-Host "라우팅 규칙 생성 응답:" -ForegroundColor Green
    $routingRuleResponse | ConvertTo-Json -Depth 10
    Write-Host ""
} catch {
    Write-Host "에러 발생: $_" -ForegroundColor Red
    exit 1
}

# 라우팅 규칙 ID 추출
$routingRuleId = $routingRuleResponse.id
Write-Host "라우팅 규칙 ID: $routingRuleId" -ForegroundColor Cyan
Write-Host ""

# 4. 오케스트레이션 규칙 생성 테스트
Write-Host "🎭 4. 오케스트레이션 규칙 생성 테스트" -ForegroundColor Yellow
Write-Host "POST $BaseUrl/orchestration-rules" -ForegroundColor Gray

$orchestrationRuleBody = @{
    name = "User API Orchestration"
    description = "사용자 API 오케스트레이션 규칙"
    routing_rule_id = $routingRuleId
    legacy_endpoint = @{
        id = $legacyEndpointId
    }
    modern_endpoint = @{
        id = $modernEndpointId
    }
    current_mode = "PARALLEL"
    is_active = $true
    transition_config = @{
        auto_transition_enabled = $true
        match_rate_threshold = 0.95
        stability_period_hours = 24
        min_requests_for_transition = 100
        rollback_threshold = 0.90
    }
    comparison_config = @{
        enabled = $true
        ignore_fields = @("timestamp", "requestId")
        allowable_difference = 0.01
        strict_mode = $false
        save_comparison_history = $true
    }
} | ConvertTo-Json -Depth 10

try {
    $orchestrationRuleResponse = Invoke-RestMethod -Uri "$BaseUrl/orchestration-rules" `
        -Method Post `
        -ContentType "application/json" `
        -Body $orchestrationRuleBody

    Write-Host "오케스트레이션 규칙 생성 응답:" -ForegroundColor Green
    $orchestrationRuleResponse | ConvertTo-Json -Depth 10
    Write-Host ""
} catch {
    Write-Host "에러 발생: $_" -ForegroundColor Red
    exit 1
}

# 5. 오케스트레이션 규칙 조회 테스트
Write-Host "🔍 5. 오케스트레이션 규칙 조회 테스트" -ForegroundColor Yellow
Write-Host "GET $BaseUrl/orchestration-rules/$routingRuleId" -ForegroundColor Gray

try {
    $orchestrationGetResponse = Invoke-RestMethod -Uri "$BaseUrl/orchestration-rules/$routingRuleId" -Method Get
    Write-Host "오케스트레이션 규칙 조회 응답:" -ForegroundColor Green
    $orchestrationGetResponse | ConvertTo-Json -Depth 10
    Write-Host ""
} catch {
    Write-Host "에러 발생: $_" -ForegroundColor Red
    exit 1
}

# 6. 전환 평가 테스트
Write-Host "⚖️ 6. 전환 평가 테스트" -ForegroundColor Yellow
Write-Host "GET $BaseUrl/orchestration-rules/$routingRuleId/evaluate-transition" -ForegroundColor Gray

try {
    $evaluateResponse = Invoke-RestMethod -Uri "$BaseUrl/orchestration-rules/$routingRuleId/evaluate-transition" -Method Get
    Write-Host "전환 평가 응답:" -ForegroundColor Green
    $evaluateResponse | ConvertTo-Json -Depth 10
    Write-Host ""
} catch {
    Write-Host "에러 발생: $_" -ForegroundColor Red
    exit 1
}

# 7. 전환 실행 테스트 (PARALLEL -> MODERN_ONLY)
Write-Host "🔄 7. 전환 실행 테스트 (PARALLEL -> MODERN_ONLY)" -ForegroundColor Yellow
Write-Host "POST $BaseUrl/orchestration-rules/$routingRuleId/execute-transition" -ForegroundColor Gray

$executeBody = @{
    new_mode = "MODERN_ONLY"
} | ConvertTo-Json

try {
    $executeResponse = Invoke-RestMethod -Uri "$BaseUrl/orchestration-rules/$routingRuleId/execute-transition" `
        -Method Post `
        -ContentType "application/json" `
        -Body $executeBody

    Write-Host "전환 실행 응답:" -ForegroundColor Green
    $executeResponse | ConvertTo-Json -Depth 10
    Write-Host ""
} catch {
    Write-Host "에러 발생: $_" -ForegroundColor Red
    exit 1
}

Write-Host "✅ CRUD API 테스트 완료!" -ForegroundColor Green
Write-Host "==================================" -ForegroundColor Cyan
