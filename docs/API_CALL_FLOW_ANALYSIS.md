# API 호출 흐름 분석

> 현재 프로젝트에서 레거시 API와 모던 API를 어떻게 호출하고 처리하는지에 대한 상세 분석 문서

**작성일**: 2025-10-27
**버전**: 1.0.0

---

## 목차

1. [전체 요청 흐름 개요](#1-전체-요청-흐름-개요)
2. [3가지 호출 전략](#2-3가지-호출-전략)
3. [병렬 호출 메커니즘](#3-병렬-호출-메커니즘)
4. [응답 비교 엔진](#4-응답-비교-엔진)
5. [자동 전환 로직](#5-자동-전환-로직)
6. [HTTP 클라이언트 최적화](#6-http-클라이언트-최적화)
7. [완전한 요청 흐름 예시](#7-완전한-요청-흐름-예시)
8. [성능 특성](#8-성능-특성)
9. [주요 파일 위치](#9-주요-파일-위치)
10. [핵심 요약](#10-핵심-요약)

---

## 1. 전체 요청 흐름 개요

```
HTTP 요청 → 미들웨어 스택 → 핸들러 → Bridge Service → 라우팅 → 오케스트레이션 → 외부 API 호출 → 응답
```

### 요청 수신 프로세스

모든 API 브리지 요청은 `/api/*path` 와일드카드 엔드포인트를 통해 수신됩니다.
관리 API는 `/management/` prefix로 별도 분리되어 있습니다.

```go
// internal/adapter/inbound/http/handler.go
func (h *Handler) ProcessBridgeRequest(c *gin.Context) {
    ctx := c.Request.Context()

    // 1. 요청 파라미터 추출
    // 변경: c.Param("path") 대신 c.Request.URL.Path 사용
    path := c.Request.URL.Path        // 전체 URL 경로 추출
    method := c.Request.Method         // HTTP 메서드

    // 2. 고유 요청 ID 생성 (분산 추적용)
    requestID := generateRequestID()   // 형식: "req-timestamp-randomstring"

    // 3. 도메인 Request 객체 생성
    request := domain.NewRequest(requestID, method, path)

    // 4. 헤더, 쿼리 파라미터, 바디 복사
    // ... (코드 생략)

    // 5. Bridge Service에 위임
    response, err := h.bridgeService.ProcessRequest(ctx, request)

    // 6. 응답 반환
    c.Data(response.StatusCode, "application/json", response.Body)
}
```

### 미들웨어 스택

요청은 다음 순서로 미들웨어를 거칩니다:

```go
// cmd/api-bridge/main.go의 setupRoutes()에서 설정
router.Use(gin.Recovery())                               // 패닉 복구
router.Use(httpadapter.NewLoggingMiddleware(...))        // 구조화된 로깅
router.Use(httpadapter.NewMetricsMiddleware(...))        // 성능 메트릭
router.Use(httpadapter.NewCORSMiddleware())              // CORS 헤더
router.Use(httpadapter.NewRateLimitMiddleware())         // Rate Limiting
```

**Rate Limiting 설정**:
- **제한**: 100 requests/second (버스트: 200)
- **제외 경로**: `/debug/pprof/`, `/management/`, `/swagger/`, `/swagger-yaml/`

---

## 2. 3가지 호출 전략

프로젝트는 `OrchestrationRule`의 `CurrentMode`에 따라 3가지 방식으로 동작합니다.

### 전략 유형

```go
// internal/core/domain/orchestration.go
type APIMode string

const (
    LEGACY_ONLY  APIMode = "LEGACY_ONLY"   // 레거시 API만 호출
    MODERN_ONLY  APIMode = "MODERN_ONLY"   // 모던 API만 호출
    PARALLEL     APIMode = "PARALLEL"      // 두 API 병렬 호출 + 비교
)
```

### 전략 선택 로직

```go
// internal/core/service/bridge_service.go
func (s *bridgeService) processOrchestratedRequest(
    ctx context.Context,
    request *domain.Request,
    rule *domain.RoutingRule,
    orchestrationRule *domain.OrchestrationRule,
    start time.Time,
) (*domain.Response, error) {

    // 현재 모드에 따라 분기
    switch orchestrationRule.CurrentMode {
    case domain.LEGACY_ONLY:
        return s.processLegacyOnlyRequest(ctx, request, orchestrationRule, start)

    case domain.MODERN_ONLY:
        return s.processModernOnlyRequest(ctx, request, orchestrationRule, start)

    case domain.PARALLEL:
        return s.processParallelRequest(ctx, request, orchestrationRule, start)

    default:
        // 기본값: 병렬 처리
        return s.processParallelRequest(ctx, request, orchestrationRule, start)
    }
}
```

### OrchestrationRule 구조

```go
type OrchestrationRule struct {
    ID                string           // 규칙 ID
    RoutingRuleID     string          // 연결된 라우팅 규칙
    LegacyEndpointID  string          // 레거시 API 엔드포인트 ID
    ModernEndpointID  string          // 모던 API 엔드포인트 ID
    CurrentMode       APIMode         // 현재 동작 모드
    TransitionConfig  TransitionConfig // 자동 전환 설정
    ComparisonConfig  ComparisonConfig // 비교 설정
    IsActive          bool            // 활성화 여부
}

// 전환 설정
type TransitionConfig struct {
    AutoTransitionEnabled    bool          // 자동 전환 활성화
    MatchRateThreshold       float64       // 전환 임계값 (예: 0.95 = 95%)
    StabilityPeriod          time.Duration // 안정화 기간
    MinRequestsForTransition int           // 최소 요청 수 (예: 100)
    RollbackThreshold        float64       // 롤백 임계값
}

// 비교 설정
type ComparisonConfig struct {
    Enabled               bool     // 비교 활성화
    IgnoreFields          []string // 무시할 필드 (예: timestamp)
    AllowableDifference   float64  // 허용 오차 (예: 0.01 = 1%)
    StrictMode            bool     // 엄격 모드
    SaveComparisonHistory bool     // 비교 이력 저장
}
```

---

## 3. 병렬 호출 메커니즘

### 핵심 구현

`PARALLEL` 모드에서는 2개의 고루틴으로 레거시 API와 모던 API를 동시에 호출합니다.

```go
// internal/core/service/orchestration_service.go
func (s *orchestrationService) ProcessParallelRequest(
    ctx context.Context,
    request *domain.Request,
    legacyEndpoint, modernEndpoint *domain.APIEndpoint,
) (*domain.APIComparison, error) {
    start := time.Now()

    // 결과 컨테이너 정의
    type apiResult struct {
        response *domain.Response
        err      error
        source   string  // "legacy" 또는 "modern"
    }

    // 버퍼링된 채널 생성 (크기 2)
    resultChan := make(chan apiResult, 2)
    var wg sync.WaitGroup

    // ===== 레거시 API 고루틴 =====
    wg.Add(1)
    go func() {
        defer wg.Done()

        // 엔드포인트 타임아웃이 적용된 컨텍스트 생성
        legacyCtx, cancel := context.WithTimeout(ctx, legacyEndpoint.Timeout)
        defer cancel()

        // 재시도 로직이 포함된 API 호출
        response, err := s.externalAPI.SendWithRetry(legacyCtx, legacyEndpoint, request)

        // 즉시 결과 전송 (논블로킹)
        resultChan <- apiResult{
            response: response,
            err:      err,
            source:   "legacy",
        }
    }()

    // ===== 모던 API 고루틴 =====
    wg.Add(1)
    go func() {
        defer wg.Done()

        // 엔드포인트 타임아웃이 적용된 컨텍스트 생성
        modernCtx, cancel := context.WithTimeout(ctx, modernEndpoint.Timeout)
        defer cancel()

        // 재시도 로직이 포함된 API 호출
        response, err := s.externalAPI.SendWithRetry(modernCtx, modernEndpoint, request)

        // 즉시 결과 전송 (논블로킹)
        resultChan <- apiResult{
            response: response,
            err:      err,
            source:   "modern",
        }
    }()

    // 채널 닫기 (모든 고루틴 완료 후)
    go func() {
        wg.Wait()
        close(resultChan)
    }()

    // ===== 결과 수집 =====
    var legacyResponse, modernResponse *domain.Response
    var legacyErr, modernErr error

    // 두 고루틴으로부터 결과 수신
    for result := range resultChan {
        if result.source == "legacy" {
            legacyResponse = result.response
            legacyErr = result.err
        } else {
            modernResponse = result.response
            modernErr = result.err
        }
    }

    // 병렬 호출 총 시간 기록
    apiDuration := time.Since(start)
    s.metrics.RecordHistogram("parallel_api_call_duration",
        float64(apiDuration.Milliseconds()),
        map[string]string{"request_id": request.ID})

    // ===== 에러 처리 =====
    if legacyErr != nil && modernErr != nil {
        return nil, fmt.Errorf("both APIs failed: legacy=%v, modern=%v",
            legacyErr, modernErr)
    }

    // ===== 비교 객체 생성 및 비교 수행 =====
    comparison := domain.NewAPIComparison(request.ID, request.ID,
        request.RoutingRuleID, legacyResponse, modernResponse)

    if legacyResponse != nil && modernResponse != nil {
        // 양쪽 모두 성공 - JSON 비교 수행
        comparisonEngine := domain.NewComparisonEngine(config)
        comparisonResult := comparisonEngine.CompareResponses(legacyResponse, modernResponse)

        comparison.MatchRate = comparisonResult.MatchRate
        comparison.Differences = comparisonResult.Differences
    }

    return comparison, nil
}
```

### 병렬 호출의 특징

| 특징 | 설명 |
|------|------|
| **버퍼링된 채널** | 크기 2의 채널로 논블로킹 전송 |
| **독립 타임아웃** | 각 API마다 별도 context 생성 |
| **동시 실행** | 두 API 호출이 완전히 병렬로 실행 |
| **오버헤드** | 약 15-20ms (느린 API 대기 + 비교 시간) |
| **에러 처리** | 한쪽 실패 시에도 다른 쪽 결과 사용 가능 |

---

## 4. 응답 비교 엔진

### 비교 프로세스

두 API가 모두 성공하면 **재귀적 JSON 비교**를 수행합니다.

```go
// internal/core/domain/comparison.go
func (e *ComparisonEngine) CompareResponses(
    legacyResponse,
    modernResponse *Response,
) *ComparisonResult {
    result := &ComparisonResult{
        MatchRate:     0.0,
        Differences:   []ResponseDiff{},
        TotalFields:   0,
        MatchedFields: 0,
    }

    // 1. JSON 파싱
    var legacyData, modernData interface{}
    json.Unmarshal(legacyResponse.Body, &legacyData)
    json.Unmarshal(modernResponse.Body, &modernData)

    // 2. 재귀적으로 모든 필드 비교
    e.compareJSON(legacyData, modernData, "", result)

    // 3. 일치율 계산
    if result.TotalFields > 0 {
        result.MatchRate = float64(result.MatchedFields) / float64(result.TotalFields)
    } else {
        result.MatchRate = 1.0  // 빈 응답 = 완전 일치
    }

    return result
}
```

### 비교 차이 유형

```go
type DiffType string

const (
    MISSING         DiffType = "MISSING"          // 레거시에만 있는 필드
    EXTRA           DiffType = "EXTRA"            // 모던에만 있는 필드
    VALUE_MISMATCH  DiffType = "VALUE_MISMATCH"   // 값 불일치
    TYPE_MISMATCH   DiffType = "TYPE_MISMATCH"    // 타입 불일치
)

type ResponseDiff struct {
    Type        DiffType    // 차이 유형
    Path        string      // JSON 경로 (예: "user.address.city")
    LegacyValue interface{} // 레거시 값
    ModernValue interface{} // 모던 값
    Message     string      // 설명 메시지
}
```

### 재귀 비교 로직

```go
func (e *ComparisonEngine) compareJSON(
    legacy, modern interface{},
    path string,
    result *ComparisonResult,
) {
    // 무시할 필드 체크
    if e.shouldIgnoreField(path) {
        return
    }

    result.TotalFields++

    // 타입 체크
    if reflect.TypeOf(legacy) != reflect.TypeOf(modern) {
        result.Differences = append(result.Differences, ResponseDiff{
            Type:        TYPE_MISMATCH,
            Path:        path,
            LegacyValue: legacy,
            ModernValue: modern,
            Message:     "Type mismatch",
        })
        return
    }

    switch legacyVal := legacy.(type) {
    case map[string]interface{}:
        // 객체 비교 - 모든 키 재귀적으로 비교
        modernVal := modern.(map[string]interface{})

        // 모든 고유 키 수집
        allKeys := make(map[string]bool)
        for k := range legacyVal { allKeys[k] = true }
        for k := range modernVal { allKeys[k] = true }

        for key := range allKeys {
            newPath := e.buildPath(path, key)
            legacyValue, legacyExists := legacyVal[key]
            modernValue, modernExists := modernVal[key]

            if !legacyExists {
                // 레거시에 없는 필드
                result.Differences = append(result.Differences, ResponseDiff{
                    Type:        MISSING,
                    Path:        newPath,
                    ModernValue: modernValue,
                })
            } else if !modernExists {
                // 모던에 없는 필드
                result.Differences = append(result.Differences, ResponseDiff{
                    Type:        EXTRA,
                    Path:        newPath,
                    LegacyValue: legacyValue,
                })
            } else {
                // 재귀 비교
                e.compareJSON(legacyValue, modernValue, newPath, result)
            }
        }

    case []interface{}:
        // 배열 비교
        modernVal := modern.([]interface{})

        // 배열 길이 체크
        if len(legacyVal) != len(modernVal) {
            result.Differences = append(result.Differences, ResponseDiff{
                Type:        VALUE_MISMATCH,
                Path:        path,
                LegacyValue: len(legacyVal),
                ModernValue: len(modernVal),
                Message:     "Array length mismatch",
            })
        }

        // 배열 요소 비교 (최대 10개까지)
        maxLen := min(len(legacyVal), len(modernVal), 10)
        for i := 0; i < maxLen; i++ {
            newPath := fmt.Sprintf("%s[%d]", path, i)
            e.compareJSON(legacyVal[i], modernVal[i], newPath, result)
        }

    default:
        // 원시 값 비교
        if !e.valuesEqual(legacyVal, modern) {
            result.Differences = append(result.Differences, ResponseDiff{
                Type:        VALUE_MISMATCH,
                Path:        path,
                LegacyValue: legacyVal,
                ModernValue: modern,
            })
            return
        }
    }

    // 차이가 없으면 매칭 필드 증가
    if !e.hasDifferenceAtPath(result.Differences, path) {
        result.MatchedFields++
    }
}
```

### 값 비교 로직 (허용 오차 적용)

```go
func (e *ComparisonEngine) valuesEqual(legacy, modern interface{}) bool {
    // 문자열 비교
    if legacyStr, ok := legacy.(string); ok {
        modernStr, ok := modern.(string)
        return ok && legacyStr == modernStr
    }

    // 숫자 비교 (허용 오차 적용)
    if legacyNum, ok := e.toFloat64(legacy); ok {
        if modernNum, ok := e.toFloat64(modern); ok {
            diff := math.Abs(legacyNum - modernNum)
            return diff <= e.config.AllowableDifference  // 기본값: 0.01 (1%)
        }
    }

    // 불린 비교
    if legacyBool, ok := legacy.(bool); ok {
        modernBool, ok := modern.(bool)
        return ok && legacyBool == modernBool
    }

    // null 비교
    if legacy == nil && modern == nil {
        return true
    }

    // Deep equality
    return reflect.DeepEqual(legacy, modern)
}
```

### 비교 설정

```go
// 기본 비교 설정
config := domain.ComparisonConfig{
    Enabled:               true,
    IgnoreFields:          []string{"timestamp", "requestId", "request_id"},
    AllowableDifference:   0.01,  // 숫자 1% 차이 허용
    StrictMode:            false,
    SaveComparisonHistory: true,
}
```

---

## 5. 자동 전환 로직

### 전환 평가 프로세스

병렬 호출 후 **백그라운드 고루틴**에서 전환 조건을 평가합니다.

```go
// internal/core/service/bridge_service.go
// 비동기 전환 평가 시작
go s.evaluateTransitionAsync(ctx, orchestrationRule)

func (s *bridgeService) evaluateTransitionAsync(
    ctx context.Context,
    rule *domain.OrchestrationRule,
) {
    // 전환 가능 여부 평가
    canTransition, err := s.orchestrationSvc.EvaluateTransition(ctx, rule)
    if err != nil {
        s.logger.WithContext(ctx).Error("failed to evaluate transition",
            "rule_id", rule.ID, "error", err)
        return
    }

    // 전환 조건 충족 시 전환 실행
    if canTransition {
        s.logger.WithContext(ctx).Info("transition condition met",
            "rule_id", rule.ID)

        err := s.orchestrationSvc.ExecuteTransition(ctx, rule, domain.MODERN_ONLY)
        if err != nil {
            s.logger.WithContext(ctx).Error("failed to execute transition",
                "rule_id", rule.ID, "error", err)
        }
    }
}
```

### 전환 조건 평가

```go
// internal/core/service/orchestration_service.go
func (s *orchestrationService) EvaluateTransition(
    ctx context.Context,
    rule *domain.OrchestrationRule,
) (bool, error) {
    // 자동 전환이 비활성화되어 있으면 false
    if !rule.TransitionConfig.AutoTransitionEnabled {
        return false, nil
    }

    // 최근 비교 결과 조회
    recentComparisons, err := s.comparisonRepo.GetRecentComparisons(
        ctx,
        rule.RoutingRuleID,
        rule.TransitionConfig.MinRequestsForTransition,  // 예: 100
    )
    if err != nil {
        return false, err
    }

    // 최소 요청 수 미달 시 false
    if len(recentComparisons) < rule.TransitionConfig.MinRequestsForTransition {
        return false, nil
    }

    // 평균 일치율 계산
    var totalMatchRate float64
    for _, comp := range recentComparisons {
        totalMatchRate += comp.MatchRate
    }
    averageMatchRate := totalMatchRate / float64(len(recentComparisons))

    // 임계값 체크
    return rule.CanTransitionToModern(averageMatchRate, len(recentComparisons)), nil
}
```

### 전환 실행

```go
func (s *orchestrationService) ExecuteTransition(
    ctx context.Context,
    rule *domain.OrchestrationRule,
    newMode domain.APIMode,
) error {
    s.logger.WithContext(ctx).Info("executing transition",
        "rule_id", rule.ID,
        "from_mode", rule.CurrentMode,
        "to_mode", newMode,
    )

    // 1. 현재 모드 업데이트
    rule.CurrentMode = newMode

    // 2. 데이터베이스에 저장
    err := s.orchestrationRepo.Update(ctx, rule)
    if err != nil {
        return fmt.Errorf("failed to update orchestration rule: %w", err)
    }

    // 3. 캐시 무효화 (있는 경우)
    if s.cache != nil {
        cacheKey := fmt.Sprintf("orchestration_rule:%s", rule.ID)
        s.cache.Delete(ctx, cacheKey)
    }

    // 4. 전환 이력 기록
    s.recordTransitionHistory(ctx, rule, newMode)

    // 5. 메트릭 기록
    s.metrics.IncrementCounter("api_mode_transition",
        map[string]string{
            "rule_id":   rule.ID,
            "from_mode": string(rule.CurrentMode),
            "to_mode":   string(newMode),
        })

    s.logger.WithContext(ctx).Info("transition completed",
        "rule_id", rule.ID,
        "new_mode", newMode,
    )

    return nil
}
```

### 전환 조건

| 조건 | 값 |
|------|-----|
| **자동 전환 활성화** | `AutoTransitionEnabled = true` |
| **최소 요청 수** | 기본 100건 |
| **임계 일치율** | 기본 95% 이상 |
| **현재 모드** | `PARALLEL` 모드일 때만 |
| **전환 대상** | `MODERN_ONLY` 모드로 전환 |

---

## 6. HTTP 클라이언트 최적화

### 연결 풀링 설정

```go
// internal/adapter/outbound/httpclient/client_adapter.go
func NewHTTPClientAdapterWithCircuitBreaker(
    timeout time.Duration,
    circuitBreaker port.CircuitBreakerService,
) port.ExternalAPIClient {
    return &httpClientAdapter{
        client: &http.Client{
            Timeout: timeout,
            Transport: &http.Transport{
                // 연결 풀링 최적화
                MaxIdleConns:        200,   // 전체 유휴 연결 수
                MaxIdleConnsPerHost: 50,    // 호스트당 유휴 연결 수
                MaxConnsPerHost:     100,   // 호스트당 최대 연결 수
                IdleConnTimeout:     90 * time.Second,

                // 성능 최적화
                DisableKeepAlives:      false,     // Keep-Alive 활성화
                DisableCompression:     false,     // 압축 활성화
                ForceAttemptHTTP2:      true,      // HTTP/2 지원
                MaxResponseHeaderBytes: 1 << 20,   // 1MB 헤더 제한
                WriteBufferSize:        32 * 1024, // 32KB 쓰기 버퍼
                ReadBufferSize:         32 * 1024, // 32KB 읽기 버퍼
            },
        },
        timeout:        timeout,
        circuitBreaker: circuitBreaker,
    }
}
```

### 재시도 로직 (Exponential Backoff)

```go
func (h *httpClientAdapter) sendWithRetryInternal(
    ctx context.Context,
    endpoint *domain.APIEndpoint,
    request *domain.Request,
) (*domain.Response, error) {
    var lastErr error
    attempt := 0

    for attempt <= endpoint.RetryCount {
        // 재시도 대기 (1초, 2초, 3초... 증가)
        if attempt > 0 {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(time.Duration(attempt) * time.Second):
            }
        }

        // API 호출
        response, err := h.SendRequest(ctx, endpoint, request)
        if err == nil {
            return response, nil  // 성공
        }

        lastErr = err
        attempt++

        // 재시도 가능한 에러만 재시도
        if !h.isRetryableError(err) {
            break
        }

        // 엔드포인트 재시도 설정 체크
        if !endpoint.ShouldRetry(attempt) {
            break
        }
    }

    return nil, fmt.Errorf("request failed after %d attempts: %w",
        attempt, lastErr)
}

// 재시도 가능한 에러 판단
func (h *httpClientAdapter) isRetryableError(err error) bool {
    errMsg := err.Error()
    return strings.Contains(errMsg, "timeout") ||
        strings.Contains(errMsg, "connection refused") ||
        strings.Contains(errMsg, "connection reset")
}
```

### URL 및 요청 구성

```go
func (h *httpClientAdapter) buildURL(
    endpoint *domain.APIEndpoint,
    request *domain.Request,
) string {
    // 1. 엔드포인트 기본 URL 시작
    baseURL := endpoint.GetFullURL()  // 예: https://api.example.com/v1

    // 2. 요청 경로 추가
    if request.Path != "" && request.Path != "/" {
        baseURL = strings.TrimSuffix(baseURL, "/") + request.Path
    }

    // 3. 쿼리 파라미터 추가
    if len(request.QueryParams) > 0 {
        baseURL += "?"
        params := make([]string, 0, len(request.QueryParams))
        for key, value := range request.QueryParams {
            params = append(params, fmt.Sprintf("%s=%s", key, value))
        }
        baseURL += strings.Join(params, "&")
    }

    return baseURL
}

func (h *httpClientAdapter) buildHTTPRequest(
    ctx context.Context,
    request *domain.Request,
    url string,
) (*http.Request, error) {
    var body io.Reader
    if len(request.Body) > 0 {
        body = strings.NewReader(string(request.Body))
    }

    // 컨텍스트 포함 요청 생성 (취소/타임아웃 지원)
    httpReq, err := http.NewRequestWithContext(ctx, request.Method, url, body)
    if err != nil {
        return nil, err
    }

    // 요청 헤더 복사
    for key, value := range request.Headers {
        httpReq.Header.Set(key, value)
    }

    return httpReq, nil
}
```

---

## 7. 완전한 요청 흐름 예시

### 시나리오: `GET /api/v1/users/123` (PARALLEL 모드)

```
1. 클라이언트 요청 도착
   HTTP GET http://localhost:10019/api/v1/bridge/api/v1/users/123
   ↓

2. 미들웨어 스택 처리
   - gin.Recovery(): 패닉 복구 준비
   - LoggingMiddleware: 요청 로깅 시작, Trace ID 생성
   - MetricsMiddleware: 타이머 시작
   - CORSMiddleware: CORS 헤더 추가
   - RateLimitMiddleware: 100 req/sec 체크 (통과)
   ↓

3. ProcessBridgeRequest 핸들러 (HTTP Adapter)
   - path 추출: "/api/v1/users/123"
   - method 추출: "GET"
   - 요청 ID 생성: "req-1698123456-abc123"
   - Request 도메인 객체 생성
   - 헤더, 쿼리 파라미터 복사
   ↓

4. BridgeService.ProcessRequest()
   - 요청 유효성 검증 (ID, method, path 필수)
   - GetRoutingRule() 호출
     a. Repository.FindMatchingRules() 실행
        → 패턴 매칭: /api/v1/users/* 일치
     b. 우선순위 가장 낮은(높은) 룰 선택
     c. 룰 활성화 상태 확인
   - 오케스트레이션 룰 존재 여부 확인
     → 존재함 (PARALLEL 모드)
   ↓

5. processOrchestratedRequest() 호출
   - CurrentMode 확인: PARALLEL
   - processParallelRequest() 호출
   ↓

6. ProcessParallelRequest() (OrchestrationService)
   - 레거시 엔드포인트 조회: http://legacy-api.com/v1/users/123
   - 모던 엔드포인트 조회: http://modern-api.com/v2/users/123
   - 버퍼링된 채널 생성 (크기 2)
   ↓

7. 병렬 API 호출 (고루틴 2개 생성)

   [고루틴 1 - 레거시 API]
   - Context 생성 (타임아웃: 5초)
   - SendWithRetry() 호출
     * Circuit Breaker 체크: Closed (정상)
     * HTTP GET http://legacy-api.com/v1/users/123
     * 연결 풀에서 연결 재사용
     * 응답 수신: 200 OK, 응답 시간: 245ms
   - resultChan에 결과 전송

   [고루틴 2 - 모던 API]
   - Context 생성 (타임아웃: 5초)
   - SendWithRetry() 호출
     * Circuit Breaker 체크: Closed (정상)
     * HTTP GET http://modern-api.com/v2/users/123
     * 연결 풀에서 연결 재사용
     * 응답 수신: 200 OK, 응답 시간: 220ms
   - resultChan에 결과 전송
   ↓

8. 결과 수집 (채널에서 수신)
   - legacyResponse 수신 완료
   - modernResponse 수신 완료
   - 병렬 호출 총 시간: 250ms (두 API 중 느린 쪽 기준)
   - 메트릭 기록: parallel_api_call_duration = 250ms
   ↓

9. 응답 비교 수행
   - 두 API 모두 성공 → JSON 비교 실행
   - ComparisonEngine.CompareResponses() 호출
     a. JSON 파싱
        Legacy: {"id":123, "name":"John", "timestamp":"2025-10-27T10:00:00Z"}
        Modern: {"id":123, "name":"John", "timestamp":"2025-10-27T10:00:01Z"}
     b. 재귀적 필드 비교
        - id: 일치 ✓
        - name: 일치 ✓
        - timestamp: 무시 (IgnoreFields 설정)
     c. 일치율 계산: 2/2 = 100%
     d. 차이점: []
   - 비교 시간: 8ms
   - 메트릭 기록: api_comparison_match_rate = 1.0 (100%)
   ↓

10. 비교 결과 저장 (DB)
    - comparison_history 테이블에 INSERT
    - 필드: mapping_id, is_match=true, match_rate=100, differences=null
    ↓

11. 백그라운드 전환 평가 (비동기 고루틴)
    go evaluateTransitionAsync(ctx, orchestrationRule)

    - 자동 전환 활성화 체크: true
    - 최근 100건 비교 결과 조회
      → 평균 일치율: 97.2%
    - 임계값 체크: 97.2% >= 95% (임계값) → 조건 충족!
    - ExecuteTransition() 호출
      a. CurrentMode 업데이트: PARALLEL → MODERN_ONLY
      b. DB 저장: orchestration_rules 테이블 UPDATE
      c. 캐시 무효화
      d. 전환 이력 저장: transition_history 테이블 INSERT
      e. 메트릭 기록: api_mode_transition 카운터 증가
    - 로그: "transition completed, rule_id=xxx, new_mode=MODERN_ONLY"
    ↓

12. 클라이언트 응답 결정
    - PARALLEL 모드: 레거시 응답 우선 반환
    - 응답 데이터: legacyResponse
    - 상태 코드: 200
    - 응답 헤더: X-Trace-ID: req-1698123456-abc123
    - 응답 바디: {"id":123, "name":"John", "timestamp":"2025-10-27T10:00:00Z"}
    ↓

13. HTTP 응답 전송
    - c.Data(200, "application/json", responseBody)
    ↓

14. 미들웨어 후처리
    - LoggingMiddleware: 응답 로깅
      * "request completed, trace_id=req-1698123456-abc123,
         status=200, duration=268ms"
    - MetricsMiddleware: 메트릭 기록
      * api_bridge_requests_total{method="GET", path="/api/v1/users/123", status="200"}++
      * api_bridge_duration_seconds{method="GET", path="/api/v1/users/123"} = 0.268
    ↓

15. 클라이언트에 응답 도착
    ✓ 총 소요 시간: 268ms
      - 병렬 API 호출: 250ms
      - 비교: 8ms
      - 기타 처리: 10ms
```

### 타임라인 다이어그램

```
시간 →
0ms    클라이언트 요청
5ms    미들웨어 처리 완료
10ms   라우팅 완료, 병렬 호출 시작
       ├─ 레거시 API 호출 시작
       └─ 모던 API 호출 시작
230ms  모던 API 응답 수신 (220ms 소요)
255ms  레거시 API 응답 수신 (245ms 소요)
260ms  병렬 호출 완료, 비교 시작
263ms  비교 완료 (8ms 소요)
265ms  비교 결과 DB 저장
266ms  백그라운드 전환 평가 시작 (비동기)
268ms  클라이언트에 응답 반환
300ms  전환 평가 완료, MODERN_ONLY로 전환 (백그라운드)
```

---

## 8. 성능 특성

### 성능 메트릭

| 항목 | 값 | 설명 |
|------|-----|------|
| **병렬 호출 오버헤드** | 15-20ms | 느린 API 대기 시간 |
| **JSON 비교 오버헤드** | <10ms | 재귀 비교 알고리즘 |
| **Rate Limit** | 100 req/sec | 버스트: 200 |
| **연결 풀 크기** | 200개 | 호스트당 50개 유휴 |
| **재시도 횟수** | 설정 가능 | 기본 3회 |
| **재시도 간격** | Exponential | 1s, 2s, 3s... |
| **캐시 TTL** | 300초 | 5분 (설정 가능) |
| **타임아웃** | 엔드포인트별 | 기본 5초 |

### 모드별 성능 비교

| 모드 | 응답 시간 | 오버헤드 | 비교 수행 |
|------|----------|---------|----------|
| **LEGACY_ONLY** | ~100ms | 0ms | 없음 |
| **MODERN_ONLY** | ~80ms | 0ms | 없음 |
| **PARALLEL** | ~120ms | 15-20ms | 있음 |

### 최적화 포인트

1. **연결 재사용**: Keep-Alive로 연결 설정 비용 절감
2. **연결 풀링**: 호스트당 50개 유휴 연결 유지
3. **HTTP/2**: 멀티플렉싱으로 여러 요청 병렬 처리
4. **버퍼링 채널**: 논블로킹 고루틴 통신
5. **비동기 전환 평가**: 클라이언트 응답에 영향 없음
6. **캐시**: 반복 요청에 대한 빠른 응답

---

## 9. 주요 파일 위치

### 파일 구조

```
internal/
├── adapter/
│   ├── inbound/
│   │   └── http/
│   │       ├── handler.go              # HTTP 요청 수신 및 응답
│   │       └── middleware.go           # Rate Limit, Logging, CORS 등
│   └── outbound/
│       └── httpclient/
│           └── client_adapter.go       # 외부 API 호출 (연결 풀, 재시도)
│
├── core/
│   ├── domain/
│   │   ├── request.go                  # Request 도메인 모델
│   │   ├── response.go                 # Response 도메인 모델
│   │   ├── routing.go                  # RoutingRule 및 매칭 로직
│   │   ├── orchestration.go            # OrchestrationRule 및 전환 로직
│   │   ├── comparison.go               # JSON 비교 엔진
│   │   └── endpoint.go                 # APIEndpoint 모델
│   │
│   ├── port/
│   │   ├── inbound.go                  # 인바운드 포트 인터페이스
│   │   └── outbound.go                 # 아웃바운드 포트 인터페이스
│   │
│   └── service/
│       ├── bridge_service.go           # 라우팅 및 전략 결정
│       └── orchestration_service.go    # 병렬 호출 오케스트레이션
│
└── repository/
    ├── routing_repository.go           # RoutingRule 저장소
    ├── orchestration_repository.go     # OrchestrationRule 저장소
    └── comparison_repository.go        # 비교 이력 저장소
```

### 주요 파일별 역할

| 파일 | 역할 |
|------|------|
| `handler.go` | HTTP 요청/응답 처리, 파라미터 추출 |
| `middleware.go` | Rate Limit, 로깅, CORS, 메트릭 |
| `client_adapter.go` | 외부 API 호출, 연결 풀링, 재시도 |
| `bridge_service.go` | 라우팅 룰 매칭, 전략 선택 |
| `orchestration_service.go` | 병렬 호출, 응답 비교, 전환 평가 |
| `comparison.go` | JSON 재귀 비교, 일치율 계산 |
| `orchestration.go` | 전환 조건 검증, 모드 전환 |
| `routing.go` | 패턴 매칭, 우선순위 선택 |

---

## 10. 핵심 요약

### 아키텍처 특징

1. **3가지 호출 모드**
   - `LEGACY_ONLY`: 레거시 API만 호출
   - `MODERN_ONLY`: 모던 API만 호출
   - `PARALLEL`: 두 API 병렬 호출 + 응답 비교

2. **병렬 호출 메커니즘**
   - 2개의 고루틴으로 동시 실행
   - 버퍼링된 채널 (크기 2)로 논블로킹 통신
   - 독립적인 타임아웃 설정
   - 평균 오버헤드: 15-20ms

3. **응답 비교**
   - 재귀적 JSON 비교 알고리즘
   - 필드별 차이점 추적
   - 일치율 자동 계산
   - 설정 가능한 무시 필드

4. **응답 우선순위**
   - PARALLEL 모드: **레거시 응답 우선 반환**
   - 레거시 실패 시: 모던 응답 반환
   - 두 API 모두 실패 시: 에러 반환

5. **자동 전환**
   - 백그라운드 비동기 평가
   - 최소 요청 수: 100건
   - 임계 일치율: 95% 이상
   - PARALLEL → MODERN_ONLY 자동 전환

6. **성능 최적화**
   - 연결 풀링 (호스트당 50개 유휴 연결)
   - HTTP/2 지원
   - Keep-Alive 활성화
   - Circuit Breaker 패턴
   - Exponential Backoff 재시도

7. **회복력 (Resilience)**
   - Circuit Breaker: 장애 격리
   - 재시도 로직: 일시적 오류 복구
   - Rate Limiting: 과부하 방지
   - Graceful Degradation: 한쪽 API 실패 시 다른 쪽 사용

8. **관찰성 (Observability)**
   - 요청 ID 생성 및 추적
   - 구조화된 로깅
   - 상세 메트릭 수집
   - 비교 이력 저장
   - 전환 이력 기록

### 경로 구조 요약

| 용도 | 경로 | 설명 |
|------|------|------|
| **브리지 요청** | `/api/*path` | 레거시/모던 API로 프록시 (모든 /api/* 요청) |
| **Health** | `/management/health` | 헬스 체크 |
| **Readiness** | `/management/ready` | 준비 상태 |
| **Status** | `/management/v1/status` | 상세 상태 |
| **Metrics** | `/management/metrics` | Prometheus 메트릭 |
| **CRUD APIs** | `/management/v1/*` | 관리용 CRUD API |
| **Swagger** | `/swagger/*` | API 문서 |
| **Debug/Profiling** | `/debug/pprof/*` | 성능 프로파일링 |

### 데이터 흐름

```
요청 → 라우팅 → 전략 선택 → API 호출 → 비교 → 전환 평가 → 응답
  ↓        ↓         ↓          ↓        ↓        ↓         ↓
로깅    룰 매칭   모드 확인   병렬 실행  JSON Diff  조건 체크  레거시 우선
```

### 주요 이점

| 이점 | 설명 |
|------|------|
| **안전한 마이그레이션** | 레거시 응답 우선으로 안정성 보장 |
| **자동화된 전환** | 일치율 기반 자동 전환으로 수동 개입 최소화 |
| **상세한 모니터링** | 비교 결과 저장으로 마이그레이션 진행 상황 추적 |
| **높은 성능** | 연결 풀링 및 병렬 처리로 오버헤드 최소화 |
| **회복력** | Circuit Breaker 및 재시도로 일시적 장애 대응 |
| **유연성** | 3가지 모드로 다양한 마이그레이션 단계 지원 |

---

## 참고 문서

- [헥사고날 아키텍처 가이드](./HEXAGONAL_ARCHITECTURE.md)
- [구현 가이드](./IMPLEMENTATION_GUIDE.md)
- [배포 가이드](./DEPLOYMENT_GUIDE.md)
- [테스트 가이드](./TESTING_GUIDE.md)
- [운영 매뉴얼](./OPERATIONS_MANUAL.md)

---

**Last Updated**: 2025-10-27
**Version**: 1.0.0
