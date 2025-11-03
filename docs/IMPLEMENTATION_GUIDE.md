# API Bridge 시스템 - 레이어별 구현 가이드

> **Note**: 이 문서는 실제 개발 시 참고할 수 있는 계층별 상세 구현 코드와 스키마를 포함합니다.

---

## 목차

1. [API Gateway Layer](#1-api-gateway-layer)
2. [Routing Layer](#2-routing-layer)
3. [Orchestration Layer](#3-orchestration-layer)
4. [HTTP Client Layer](#4-http-client-layer)
5. [Comparison Engine](#5-comparison-engine)
6. [Decision Engine](#6-decision-engine)
7. [Data Layer](#7-data-layer)

---

## 1. API Gateway Layer

### HTTP Server 구성

```go
package main

import (
    "github.com/gin-gonic/gin"
    "time"
)

// HTTP Server 초기화 (cmd/api-bridge/main.go의 setupRoutes 참조)
func NewAPIServer(config *Config) *gin.Engine {
    router := gin.New()

    // Middleware 등록 (순서 중요!)
    router.Use(
        gin.Recovery(),                           // 패닉 복구
        NewLoggingMiddleware(logger),             // 구조화된 로깅
        NewMetricsMiddleware(metricsCollector),   // 성능 메트릭 수집
        NewCORSMiddleware(),                      // CORS 헤더 처리
        NewRateLimitMiddleware(),                 // Rate Limiting (100 req/sec)
    )

    // === Internal Management API (우선순위 높음 - 먼저 등록) ===
    abs := router.Group("/abs")
    {
        // Health Check & Monitoring
        abs.GET("/health", HealthCheckHandler)
        abs.GET("/ready", ReadinessCheckHandler)
        abs.GET("/metrics", MetricsHandler)
        abs.GET("/status", StatusHandler)

        // Swagger UI & Documentation
        abs.GET("/swagger/*any", SwaggerHandler)
        abs.Static("/swagger-yaml", "./api-docs")

        // pprof 프로파일링 (디버그 전용)
        abs.GET("/debug/pprof/*any", PprofHandler)

        // CRUD APIs
        abs.POST("/v1/endpoints", CreateEndpointHandler)
        abs.GET("/v1/endpoints", ListEndpointsHandler)
        // ... 기타 CRUD 엔드포인트
    }

    // === API Bridge - 모든 외부 요청 처리 (반드시 마지막에 등록!) ===
    // NoRoute 핸들러: /abs/* 를 제외한 모든 경로를 브릿지로 라우팅
    router.NoRoute(BridgeHandler)

    return router
}

// Request Logger Middleware
func RequestLoggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        // Trace ID 생성
        traceID := generateTraceID()
        c.Set("trace_id", traceID)
        c.Header("X-Trace-ID", traceID)
        
        // 요청 로깅
        logger.Info("request started",
            "trace_id", traceID,
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
        )
        
        c.Next()
        
        // 응답 로깅
        duration := time.Since(start)
        logger.Info("request completed",
            "trace_id", traceID,
            "status", c.Writer.Status(),
            "duration_ms", duration.Milliseconds(),
        )
    }
}

// Rate Limiter Middleware
func RateLimiterMiddleware() gin.HandlerFunc {
    limiter := rate.NewLimiter(100, 200) // 100 req/sec, burst 200

    // Rate limit에서 제외할 경로 정의
    // 관리 API, 모니터링, Swagger는 Rate Limit에서 제외
    skipPaths := []string{
        "/abs/", // 모든 관리 API (Health, CRUD, Metrics, Debug, Swagger 등)
    }

    return func(c *gin.Context) {
        // 제외 경로 확인
        for _, skipPath := range skipPaths {
            if strings.HasPrefix(c.Request.URL.Path, skipPath) {
                c.Next()
                return
            }
        }

        // Rate limit 적용
        if !limiter.Allow() {
            c.JSON(429, gin.H{
                "error": "Rate limit exceeded",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## 2. Routing Layer

### 데이터 구조

```go
package domain

import (
    "regexp"
    "time"
)

// RoutingRule은 요청을 적절한 엔드포인트로 라우팅하는 규칙을 나타냅니다.
type RoutingRule struct {
    ID               string            // 규칙 고유 ID
    Name             string            // 규칙 이름
    PathPattern      string            // 경로 패턴 (예: /api/v1/users/*)
    MethodPattern    string            // HTTP 메서드 패턴 (예: GET, POST, *)
    Method           string            // HTTP 메서드
    Headers          map[string]string // 헤더 매칭
    QueryParams      map[string]string // 쿼리 파라미터 매칭
    EndpointID       string            // 대상 엔드포인트 ID
    LegacyEndpointID string            // 레거시 엔드포인트 ID
    ModernEndpointID string            // 모던 엔드포인트 ID
    Priority         int               // 우선순위 (낮을수록 먼저 매칭)
    IsActive         bool              // 활성화 여부
    CacheEnabled     bool              // 캐시 사용 여부
    CacheTTL         int               // 캐시 TTL (초)
    Description      string            // 설명
    CreatedAt        time.Time         // 생성 시간
    UpdatedAt        time.Time         // 수정 시간
    compiledRegex    *regexp.Regexp    // 컴파일된 정규식 (private)
}

// NewRoutingRule은 새로운 RoutingRule을 생성합니다.
func NewRoutingRule(id, name, pathPattern, methodPattern, endpointID string) *RoutingRule {
    return &RoutingRule{
        ID:            id,
        Name:          name,
        PathPattern:   pathPattern,
        MethodPattern: methodPattern,
        EndpointID:    endpointID,
        Priority:      100,
        IsActive:      true,
        CacheEnabled:  false,
        CacheTTL:      300, // 기본 5분
    }
}
```

### 라우팅 로직

```go
package service

// bridgeService에 포함된 메모리 캐시 구조
type routingRuleCacheEntry struct {
    rules     []*domain.RoutingRule
    timestamp time.Time
}

type bridgeService struct {
    // ... 기타 필드
    routingRuleCache    map[string]*routingRuleCacheEntry // 캐시 맵 (key: method:path)
    routingRuleCacheMu  sync.RWMutex                      // 캐시 락
    routingRuleCacheTTL time.Duration                     // 캐시 TTL (기본: 60초)
}

// GetRoutingRule: 라우팅 규칙 조회 (메모리 캐시 → Redis → DB)
func (s *bridgeService) GetRoutingRule(ctx context.Context, request *domain.Request) (*domain.RoutingRule, error) {
    cacheKey := fmt.Sprintf("%s:%s", request.Method, request.Path)

    // 1. 메모리 캐시 조회
    s.routingRuleCacheMu.RLock()
    if entry, exists := s.routingRuleCache[cacheKey]; exists {
        if time.Since(entry.timestamp) < s.routingRuleCacheTTL {
            s.routingRuleCacheMu.RUnlock()
            // 캐시된 규칙 중 매칭되는 것 찾기
            for _, rule := range entry.rules {
                if matched, _ := rule.Matches(request); matched {
                    return rule, nil
                }
            }
        }
    }
    s.routingRuleCacheMu.RUnlock()

    // 2. DB에서 모든 활성 규칙 조회
    rules, err := s.routingRepo.FindMatchingRules(ctx, request.Path, request.Method)
    if err != nil {
        return nil, err
    }

    // 3. 메모리 캐시에 저장
    s.routingRuleCacheMu.Lock()
    s.routingRuleCache[cacheKey] = &routingRuleCacheEntry{
        rules:     rules,
        timestamp: time.Now(),
    }
    s.routingRuleCacheMu.Unlock()

    // 4. 매칭되는 규칙 찾기 (우선순위 순)
    for _, rule := range rules {
        if matched, _ := rule.Matches(request); matched {
            return rule, nil
        }
    }

    return nil, domain.ErrRouteNotFound
}

// Matches: 정규식 기반 패턴 매칭
func (r *RoutingRule) Matches(request *Request) (bool, error) {
    if !r.IsActive {
        return false, nil
    }

    // 메서드 매칭
    if r.MethodPattern != "*" && r.MethodPattern != request.Method {
        return false, nil
    }

    // 경로 패턴 매칭 (정규식)
    if r.compiledRegex == nil {
        pattern := convertPatternToRegex(r.PathPattern)
        var err error
        r.compiledRegex, err = regexp.Compile(pattern)
        if err != nil {
            return false, NewDomainError("REGEX_COMPILE_ERROR", "failed to compile path pattern", err)
        }
    }

    return r.compiledRegex.MatchString(request.Path), nil
}

// convertPatternToRegex: 간단한 패턴을 정규식으로 변환
// 예: /api/v1/users/* -> ^/api/v1/users/.*$
func convertPatternToRegex(pattern string) string {
    // 특수문자 이스케이프
    regex := regexp.QuoteMeta(pattern)
    // * 를 .* 로 변환
    regex = regexp.MustCompile(`\\\*`).ReplaceAllString(regex, ".*")
    // 시작과 끝 앵커 추가
    return "^" + regex + "$"
}
```

**라우팅 메커니즘 요약:**
1. **3-tier 캐시 전략**: 메모리 캐시(60초 TTL) → Redis 캐시 → OracleDB
2. **정규식 기반 매칭**: Radix Tree 대신 `regexp.Compile()` 사용
3. **우선순위 기반 선택**: 여러 규칙이 매칭되면 Priority 낮은(높은 우선순위) 것 선택
4. **컴파일 캐싱**: 정규식 객체를 RoutingRule 내부에 캐싱하여 성능 최적화

---

## 3. Orchestration Layer

### Orchestrator 구현

```go
package orchestration

// Orchestrator
type Orchestrator struct {
    legacyClient   *HTTPClient
    modernClient   *HTTPClient
    workerPool     *WorkerPool
    circuitBreaker *CircuitBreaker
    comparator     *ResponseComparator
    decisionEngine *DecisionEngine
    metrics        *OrchestratorMetrics
}

// 실행
func (o *Orchestrator) Execute(ctx context.Context, req *Request) (*Response, error) {
    strategy := req.Mapping.Strategy
    
    switch strategy {
    case LEGACY_ONLY:
        return o.callLegacyOnly(ctx, req)
    
    case PARALLEL:
        return o.callParallel(ctx, req)
    
    case MODERN_ONLY:
        return o.callModernOnly(ctx, req)
    
    default:
        return nil, fmt.Errorf("unknown strategy: %s", strategy)
    }
}

// 병렬 호출 구현
func (o *Orchestrator) callParallel(ctx context.Context, req *Request) (*Response, error) {
    // 채널 생성
    legacyCh := make(chan *APIResponse, 1)
    modernCh := make(chan *APIResponse, 1)
    errCh := make(chan error, 2)
    
    // 타임아웃 설정
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    // 고루틴으로 병렬 호출
    go func() {
        resp, err := o.callLegacyAPI(ctx, req)
        if err != nil {
            errCh <- err
            return
        }
        legacyCh <- resp
    }()
    
    go func() {
        resp, err := o.callModernAPI(ctx, req)
        if err != nil {
            errCh <- err
            return
        }
        modernCh <- resp
    }()
    
    // 응답 수집
    var legacyResp, modernResp *APIResponse
    receivedCount := 0
    
    for receivedCount < 2 {
        select {
        case legacyResp = <-legacyCh:
            receivedCount++
            o.metrics.LegacyCallsTotal.Inc()
            
        case modernResp = <-modernCh:
            receivedCount++
            o.metrics.ModernCallsTotal.Inc()
            
        case err := <-errCh:
            receivedCount++
            logger.Error("API call failed", "error", err)
            
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    // 레거시 응답이 없으면 에러
    if legacyResp == nil {
        return nil, fmt.Errorf("legacy API call failed")
    }
    
    // 비동기로 비교 처리 (클라이언트 응답에 영향 없음)
    if modernResp != nil {
        go o.compareAndRecord(req.Mapping.ID, legacyResp, modernResp)
    }
    
    // 레거시 응답 우선 반환
    return legacyResp.Response, nil
}

// 레거시 API만 호출
func (o *Orchestrator) callLegacyOnly(ctx context.Context, req *Request) (*Response, error) {
    resp, err := o.callLegacyAPI(ctx, req)
    if err != nil {
        return nil, err
    }
    return resp.Response, nil
}

// 모던 API만 호출
func (o *Orchestrator) callModernOnly(ctx context.Context, req *Request) (*Response, error) {
    resp, err := o.callModernAPI(ctx, req)
    if err != nil {
        return nil, err
    }
    return resp.Response, nil
}
```

### Circuit Breaker 구현

```go
package service

import (
    "context"
    "demo-api-bridge/internal/core/domain"
    "demo-api-bridge/internal/core/port"
    "fmt"
    "sync"
    "time"
)

// circuitBreakerService: 자체 구현 Circuit Breaker 서비스
type circuitBreakerService struct {
    breakers map[string]*domain.CircuitBreaker
    mu       sync.RWMutex
    logger   port.Logger
    metrics  port.MetricsCollector
}

// NewCircuitBreakerService: Circuit Breaker 서비스 생성
func NewCircuitBreakerService(logger port.Logger, metrics port.MetricsCollector) port.CircuitBreakerService {
    return &circuitBreakerService{
        breakers: make(map[string]*domain.CircuitBreaker),
        logger:   logger,
        metrics:  metrics,
    }
}

// Execute: Circuit Breaker를 통한 함수 실행
func (s *circuitBreakerService) Execute(
    ctx context.Context,
    name string,
    config *domain.CircuitBreakerConfig,
    fn func() (interface{}, error),
) (interface{}, error) {
    // 1. Circuit Breaker 조회 또는 생성
    breaker := s.getOrCreateBreaker(name, config)

    // 2. 현재 상태 확인
    if !breaker.CanExecute() {
        s.logger.Warn("circuit breaker is open, rejecting request", "name", name)
        s.metrics.IncrementCounter("circuit_breaker_rejected", map[string]string{"name": name})
        return nil, fmt.Errorf("circuit breaker '%s' is open", name)
    }

    // 3. 함수 실행
    start := time.Now()
    result, err := fn()
    duration := time.Since(start)

    // 4. 결과 기록
    if err != nil {
        breaker.RecordFailure()
        s.logger.Error("circuit breaker recorded failure",
            "name", name,
            "error", err,
            "duration_ms", duration.Milliseconds(),
        )
        s.metrics.IncrementCounter("circuit_breaker_failures", map[string]string{"name": name})
    } else {
        breaker.RecordSuccess()
        s.metrics.IncrementCounter("circuit_breaker_success", map[string]string{"name": name})
    }

    // 5. 메트릭 기록
    s.metrics.RecordHistogram("circuit_breaker_duration", duration.Seconds(), map[string]string{"name": name})

    return result, err
}

// getOrCreateBreaker: Circuit Breaker 조회 또는 생성
func (s *circuitBreakerService) getOrCreateBreaker(name string, config *domain.CircuitBreakerConfig) *domain.CircuitBreaker {
    s.mu.RLock()
    breaker, exists := s.breakers[name]
    s.mu.RUnlock()

    if exists {
        return breaker
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    // Double-check (다른 고루틴이 생성했을 수 있음)
    if breaker, exists := s.breakers[name]; exists {
        return breaker
    }

    // 새 Circuit Breaker 생성
    breaker = domain.NewCircuitBreaker(name, config)
    s.breakers[name] = breaker

    s.logger.Info("circuit breaker created", "name", name)

    return breaker
}
```

**Circuit Breaker 특징:**
1. **자체 구현**: `sony/gobreaker` 대신 도메인 모델 기반 구현
2. **상태 관리**: Closed → Open → Half-Open 자동 전환
3. **실패율 기반 트립**: 설정 가능한 임계값 (기본: 60%)
4. **메트릭 통합**: 성공/실패/거부 카운터 자동 기록
5. **스레드 세이프**: `sync.RWMutex`로 동시성 제어

---

## 4. HTTP Client Layer

### HTTP Client 구현

```go
package client

import (
    "net"
    "net/http"
    "time"
)

// HTTP Client
type HTTPClient struct {
    client      *http.Client
    baseURL     string
    timeout     time.Duration
    retryPolicy *RetryPolicy
    metrics     *ClientMetrics
}

// HTTP Client 생성
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        MaxConnsPerHost:     50,
        IdleConnTimeout:     90 * time.Second,
        DisableKeepAlives:   false,
        
        DialContext: (&net.Dialer{
            Timeout:   5 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        
        TLSHandshakeTimeout:   10 * time.Second,
        ResponseHeaderTimeout: 10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
        ForceAttemptHTTP2:     true,
    }
    
    return &HTTPClient{
        client: &http.Client{
            Transport: transport,
            Timeout:   timeout,
        },
        baseURL: baseURL,
        timeout: timeout,
        retryPolicy: &RetryPolicy{
            MaxRetries:   3,
            InitialDelay: 100 * time.Millisecond,
            MaxDelay:     2 * time.Second,
            Multiplier:   2.0,
            Jitter:       true,
        },
    }
}

// HTTP 요청
func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
    var resp *http.Response
    var err error
    
    for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
        // 메트릭 기록
        start := time.Now()
        
        resp, err = c.client.Do(req.WithContext(ctx))
        
        duration := time.Since(start)
        c.metrics.RequestDuration.Observe(duration.Seconds())
        
        if err == nil && resp.StatusCode < 500 {
            c.metrics.RequestsTotal.WithLabelValues("success").Inc()
            return resp, nil
        }
        
        c.metrics.RequestsTotal.WithLabelValues("error").Inc()
        
        // 재시도 지연
        if attempt < c.retryPolicy.MaxRetries {
            delay := c.retryPolicy.CalculateDelay(attempt)
            time.Sleep(delay)
        }
    }
    
    return nil, fmt.Errorf("max retries exceeded: %w", err)
}
```

### Retry Policy 구현

```go
package httpclient

import (
    "context"
    "strings"
    "time"
)

// sendWithRetryInternal: 내부 재시도 로직 (Linear Backoff)
func (h *httpClientAdapter) sendWithRetryInternal(
    ctx context.Context,
    endpoint *domain.APIEndpoint,
    request *domain.Request,
) (*domain.Response, error) {
    var lastErr error
    attempt := 0

    for attempt <= endpoint.RetryCount {
        // 재시도 대기 (Linear Backoff: 1초, 2초, 3초...)
        if attempt > 0 {
            // Context 취소 확인
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(time.Duration(attempt) * time.Second):
                // 대기 완료
            }
        }

        // API 호출
        response, err := h.SendRequest(ctx, endpoint, request)
        if err == nil {
            return response, nil // 성공
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

    return nil, fmt.Errorf("request failed after %d attempts: %w", attempt, lastErr)
}

// isRetryableError: 재시도 가능한 에러 판단
func (h *httpClientAdapter) isRetryableError(err error) bool {
    errMsg := err.Error()
    return strings.Contains(errMsg, "timeout") ||
        strings.Contains(errMsg, "connection refused") ||
        strings.Contains(errMsg, "connection reset")
}
```

**재시도 전략:**
1. **Linear Backoff**: 1초 → 2초 → 3초 (Exponential이 아님)
2. **재시도 조건**: Timeout, Connection Refused, Connection Reset
3. **최대 재시도**: 엔드포인트별 설정 가능 (기본: 3회)
4. **Context 취소 지원**: Graceful shutdown 시 즉시 중단

---

## 5. Comparison Engine

### Response Comparator 구현

```go
package comparison

// Response Comparator
type ResponseComparator struct {
    diffEngine  *DiffEngine
    ruleEngine  *ComparisonRuleEngine
    recorder    *ComparisonRecorder
}

// Comparison Result
type ComparisonResult struct {
    MappingID     string       `json:"mapping_id"`
    IsMatch       bool         `json:"is_match"`
    MatchRate     float64      `json:"match_rate"`
    Differences   []Difference `json:"differences"`
    Timestamp     time.Time    `json:"timestamp"`
    TraceID       string       `json:"trace_id"`
}

// Difference
type Difference struct {
    Path      string      `json:"path"`       // JSON 경로
    LegacyVal interface{} `json:"legacy_val"`
    ModernVal interface{} `json:"modern_val"`
    DiffType  string      `json:"diff_type"`  // MISSING, EXTRA, VALUE_MISMATCH, TYPE_MISMATCH
}

// 응답 비교
func (c *ResponseComparator) Compare(legacy, modern []byte) (*ComparisonResult, error) {
    var legacyJSON, modernJSON map[string]interface{}
    
    if err := json.Unmarshal(legacy, &legacyJSON); err != nil {
        return nil, fmt.Errorf("failed to unmarshal legacy response: %w", err)
    }
    
    if err := json.Unmarshal(modern, &modernJSON); err != nil {
        return nil, fmt.Errorf("failed to unmarshal modern response: %w", err)
    }
    
    // 재귀적 비교
    diffs := c.diffEngine.DeepCompare(legacyJSON, modernJSON, "")
    
    // 일치율 계산
    totalFields := c.countFields(legacyJSON)
    matchedFields := totalFields - len(diffs)
    matchRate := 0.0
    if totalFields > 0 {
        matchRate = float64(matchedFields) / float64(totalFields) * 100
    }
    
    return &ComparisonResult{
        IsMatch:     len(diffs) == 0,
        MatchRate:   matchRate,
        Differences: diffs,
        Timestamp:   time.Now(),
    }, nil
}

// JSON 재귀 비교
func (de *DiffEngine) DeepCompare(legacy, modern interface{}, path string) []Difference {
    var diffs []Difference
    
    // 타입 체크
    if reflect.TypeOf(legacy) != reflect.TypeOf(modern) {
        diffs = append(diffs, Difference{
            Path:      path,
            LegacyVal: legacy,
            ModernVal: modern,
            DiffType:  "TYPE_MISMATCH",
        })
        return diffs
    }
    
    // Map 비교
    if legacyMap, ok := legacy.(map[string]interface{}); ok {
        modernMap := modern.(map[string]interface{})
        
        // 레거시 키 체크
        for key, legacyVal := range legacyMap {
            newPath := path + "." + key
            if path == "" {
                newPath = key
            }
            
            modernVal, exists := modernMap[key]
            if !exists {
                diffs = append(diffs, Difference{
                    Path:      newPath,
                    LegacyVal: legacyVal,
                    ModernVal: nil,
                    DiffType:  "MISSING",
                })
                continue
            }
            
            // 재귀 비교
            diffs = append(diffs, de.DeepCompare(legacyVal, modernVal, newPath)...)
        }
        
        // 모던 추가 키 체크
        for key, modernVal := range modernMap {
            if _, exists := legacyMap[key]; !exists {
                newPath := path + "." + key
                if path == "" {
                    newPath = key
                }
                diffs = append(diffs, Difference{
                    Path:      newPath,
                    LegacyVal: nil,
                    ModernVal: modernVal,
                    DiffType:  "EXTRA",
                })
            }
        }
        
        return diffs
    }
    
    // 값 비교
    if !reflect.DeepEqual(legacy, modern) {
        diffs = append(diffs, Difference{
            Path:      path,
            LegacyVal: legacy,
            ModernVal: modern,
            DiffType:  "VALUE_MISMATCH",
        })
    }
    
    return diffs
}
```

### Comparison Rule Engine

```go
package comparison

// Comparison Rule
type ComparisonRule struct {
    ExcludedFields   []string                              // 비교 제외 필드
    NumericTolerance float64                               // 숫자 허용 오차
    IgnoreOrder      bool                                  // 배열 순서 무시
    CustomRules      map[string]func(a, b interface{}) bool // 커스텀 규칙
}

// 규칙 적용
func (r *ComparisonRule) ShouldCompare(path string) bool {
    for _, excluded := range r.ExcludedFields {
        if strings.Contains(path, excluded) {
            return false
        }
    }
    return true
}

// 숫자 비교 (허용 오차)
func (r *ComparisonRule) CompareNumeric(a, b float64) bool {
    diff := math.Abs(a - b)
    return diff <= r.NumericTolerance
}
```

**⚠️ 중요: 배열 비교 성능 최적화**

실제 구현에서는 배열 비교 시 **최대 10개 요소**만 비교합니다 (`internal/core/domain/comparison.go:175-182`):

```go
// 배열 요소 비교 (최대 10개까지만)
maxLen := len(legacyVal)
if len(modernVal) > maxLen {
    maxLen = len(modernVal)
}
if maxLen > 10 {
    maxLen = 10 // 성능을 위해 제한
}
```

**이유:**
- 대용량 배열 응답 시 비교 시간이 급격히 증가
- 10개 샘플로도 충분한 일치율 판단 가능
- 클라이언트 응답 지연 최소화 (목표: <10ms)

---

## 6. Decision Engine

### Decision Engine 구현

```go
package decision

// Decision Engine
type DecisionEngine struct {
    thresholdMgr   *ThresholdManager
    transitionCtrl *TransitionController
    eventPublisher *EventPublisher
}

// 전환 결정
func (d *DecisionEngine) Evaluate(ctx context.Context, mappingID string, matchRate float64) error {
    mapping, err := d.thresholdMgr.GetMapping(ctx, mappingID)
    if err != nil {
        return err
    }
    
    // 임계값 체크
    if matchRate >= mapping.Threshold && mapping.Strategy == PARALLEL {
        // 최근 N개 요청 모두 100% 확인 (안정성 보장)
        recentRates, err := d.thresholdMgr.GetRecentMatchRates(ctx, mappingID, 100)
        if err != nil {
            return err
        }
        
        allMatched := true
        for _, rate := range recentRates {
            if rate < mapping.Threshold {
                allMatched = false
                break
            }
        }
        
        if allMatched {
            // 전환 조건 충족
            return d.transitionCtrl.TransitionToModern(ctx, mappingID)
        }
    }
    
    return nil
}
```

### Transition Controller 구현

```go
package decision

// Transition Controller
type TransitionController struct {
    repo       MappingRepository
    cache      Cache
    eventBus   EventBus
    metrics    *TransitionMetrics
}

// 모던으로 전환
func (t *TransitionController) TransitionToModern(ctx context.Context, mappingID string) error {
    logger.Info("transitioning to modern", "mapping_id", mappingID)
    
    // 1. DB 상태 업데이트
    err := t.repo.UpdateStrategy(ctx, mappingID, MODERN_ONLY)
    if err != nil {
        return fmt.Errorf("failed to update strategy: %w", err)
    }
    
    // 2. 캐시 무효화
    cacheKey := fmt.Sprintf("mapping:%s", mappingID)
    t.cache.Delete(cacheKey)
    
    // 3. 전환 이력 저장
    history := &TransitionHistory{
        MappingID:     mappingID,
        FromStrategy:  PARALLEL,
        ToStrategy:    MODERN_ONLY,
        Reason:        "Match rate threshold reached",
        PerformedBy:   "system",
        CreatedAt:     time.Now(),
    }
    if err := t.repo.SaveTransitionHistory(ctx, history); err != nil {
        logger.Error("failed to save transition history", "error", err)
    }
    
    // 4. 전환 이벤트 발행
    t.eventBus.Publish(Event{
        Type: "API_TRANSITIONED",
        Data: map[string]interface{}{
            "mapping_id": mappingID,
            "strategy":   MODERN_ONLY,
            "timestamp":  time.Now(),
        },
    })
    
    // 5. 메트릭 기록
    t.metrics.TransitionCounter.Inc()
    
    logger.Info("transition completed", "mapping_id", mappingID)
    return nil
}

// 롤백
func (t *TransitionController) Rollback(ctx context.Context, mappingID string, reason string) error {
    logger.Warn("rolling back to parallel", "mapping_id", mappingID, "reason", reason)

    // 1. DB 상태 업데이트
    err := t.repo.UpdateStrategy(ctx, mappingID, PARALLEL)
    if err != nil {
        return fmt.Errorf("failed to rollback strategy: %w", err)
    }

    // 2. 캐시 무효화
    cacheKey := fmt.Sprintf("mapping:%s", mappingID)
    t.cache.Delete(cacheKey)

    // 3. 롤백 이력 저장
    history := &TransitionHistory{
        MappingID:     mappingID,
        FromStrategy:  MODERN_ONLY,
        ToStrategy:    PARALLEL,
        Reason:        reason,
        PerformedBy:   "system",
        CreatedAt:     time.Now(),
    }
    t.repo.SaveTransitionHistory(ctx, history)

    // 4. 이벤트 발행
    t.eventBus.Publish(Event{
        Type: "API_ROLLBACK",
        Data: map[string]interface{}{
            "mapping_id": mappingID,
            "reason":     reason,
            "timestamp":  time.Now(),
        },
    })

    return nil
}
```

**전환 설정 기본값 (domain/orchestration.go:97-103):**

```go
TransitionConfig: TransitionConfig{
    AutoTransitionEnabled:    true,
    MatchRateThreshold:       0.95,  // 95% 일치 시 전환
    StabilityPeriod:          24 * time.Hour,  // 24시간 안정화 기간
    MinRequestsForTransition: 100,
    RollbackThreshold:        0.90,  // 90% 미만 시 롤백
},
```

**롤백 조건 평가 (domain/orchestration.go:135-141):**

```go
// ShouldRollback는 롤백이 필요한지 확인합니다.
func (o *OrchestrationRule) ShouldRollback(recentMatchRate float64) bool {
    if o.CurrentMode != MODERN_ONLY {
        return false
    }

    return recentMatchRate < o.TransitionConfig.RollbackThreshold
}
```

---

## 7. Data Layer

### OracleDB 스키마

**실제 데이터베이스 구조는 4개의 독립적인 테이블로 구성됩니다** (`scripts/create_tables.sql`):

```sql
-- 1. API Endpoints Table (API 엔드포인트 정보)
CREATE TABLE MAP.ABS_API_ENDPOINTS (
    id VARCHAR2(100) PRIMARY KEY,
    name VARCHAR2(200) NOT NULL,
    description VARCHAR2(500),
    base_url VARCHAR2(500) NOT NULL,
    health_url VARCHAR2(500),
    is_active NUMBER(1) DEFAULT 1 NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT chk_endpoints_is_active CHECK (is_active IN (0, 1))
);

-- 2. Routing Rules Table (라우팅 규칙)
CREATE TABLE MAP.ABS_ROUTING_RULES (
    id VARCHAR2(100) PRIMARY KEY,
    name VARCHAR2(200) NOT NULL,
    description VARCHAR2(500),
    method VARCHAR2(10),
    path_pattern VARCHAR2(500) NOT NULL,
    headers CLOB,
    query_params CLOB,
    legacy_endpoint_id VARCHAR2(100),
    modern_endpoint_id VARCHAR2(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_routing_legacy FOREIGN KEY (legacy_endpoint_id)
        REFERENCES MAP.ABS_API_ENDPOINTS(id) ON DELETE SET NULL,
    CONSTRAINT fk_routing_modern FOREIGN KEY (modern_endpoint_id)
        REFERENCES MAP.ABS_API_ENDPOINTS(id) ON DELETE SET NULL
);

-- 3. Orchestration Rules Table (오케스트레이션 규칙)
CREATE TABLE MAP.ABS_ORCHESTRATION_RULES (
    id VARCHAR2(100) PRIMARY KEY,
    name VARCHAR2(200) NOT NULL,
    description VARCHAR2(500),
    routing_rule_id VARCHAR2(100) NOT NULL,
    legacy_endpoint_id VARCHAR2(100) NOT NULL,
    modern_endpoint_id VARCHAR2(100) NOT NULL,
    current_mode VARCHAR2(20) DEFAULT 'PARALLEL' NOT NULL,
    transition_config CLOB,  -- JSON: TransitionConfig
    comparison_config CLOB,  -- JSON: ComparisonConfig
    is_active NUMBER(1) DEFAULT 1 NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT chk_orchestration_mode CHECK (
        current_mode IN ('LEGACY_ONLY', 'PARALLEL', 'MODERN_ONLY')
    ),
    CONSTRAINT chk_orchestration_active CHECK (is_active IN (0, 1)),
    CONSTRAINT fk_orchestration_routing FOREIGN KEY (routing_rule_id)
        REFERENCES MAP.ABS_ROUTING_RULES(id) ON DELETE CASCADE,
    CONSTRAINT fk_orchestration_legacy FOREIGN KEY (legacy_endpoint_id)
        REFERENCES MAP.ABS_API_ENDPOINTS(id) ON DELETE CASCADE,
    CONSTRAINT fk_orchestration_modern FOREIGN KEY (modern_endpoint_id)
        REFERENCES MAP.ABS_API_ENDPOINTS(id) ON DELETE CASCADE
);

-- 4. API Comparisons Table (API 응답 비교 이력)
CREATE TABLE MAP.ABS_API_COMPARISONS (
    id VARCHAR2(100) PRIMARY KEY,
    request_id VARCHAR2(100) NOT NULL,
    routing_rule_id VARCHAR2(100),
    match_rate NUMBER(5,4) DEFAULT 0,  -- ⚠️ 정밀도: (5,4) = 0.0000~1.0000
    differences CLOB,  -- JSON: []ResponseDiff
    comparison_duration_ms NUMBER(10),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_comparison_routing FOREIGN KEY (routing_rule_id)
        REFERENCES MAP.ABS_ROUTING_RULES(id) ON DELETE SET NULL
);

-- Indexes for Performance
CREATE INDEX IDX_ABS_ENDPOINTS_ACTIVE ON MAP.ABS_API_ENDPOINTS(is_active);
CREATE INDEX IDX_ABS_ENDPOINTS_CREATED ON MAP.ABS_API_ENDPOINTS(created_at);

CREATE INDEX IDX_ABS_ROUTING_PATH ON MAP.ABS_ROUTING_RULES(path_pattern);
CREATE INDEX IDX_ABS_ROUTING_METHOD ON MAP.ABS_ROUTING_RULES(method);
CREATE INDEX IDX_ABS_ROUTING_LEGACY ON MAP.ABS_ROUTING_RULES(legacy_endpoint_id);
CREATE INDEX IDX_ABS_ROUTING_MODERN ON MAP.ABS_ROUTING_RULES(modern_endpoint_id);

CREATE INDEX IDX_ABS_ORCH_ROUTING ON MAP.ABS_ORCHESTRATION_RULES(routing_rule_id);
CREATE INDEX IDX_ABS_ORCH_MODE ON MAP.ABS_ORCHESTRATION_RULES(current_mode);
CREATE INDEX IDX_ABS_ORCH_ACTIVE ON MAP.ABS_ORCHESTRATION_RULES(is_active);

CREATE INDEX IDX_ABS_COMP_REQUEST ON MAP.ABS_API_COMPARISONS(request_id);
CREATE INDEX IDX_ABS_COMP_ROUTING ON MAP.ABS_API_COMPARISONS(routing_rule_id);
CREATE INDEX IDX_ABS_COMP_TIMESTAMP ON MAP.ABS_API_COMPARISONS(timestamp);
CREATE INDEX IDX_ABS_COMP_MATCH_RATE ON MAP.ABS_API_COMPARISONS(match_rate);
```

**테이블 관계도:**

```
ABS_API_ENDPOINTS (1) ─────┬───── (N) ABS_ROUTING_RULES
                           │            │
                           │            │ (1)
                           │            │
                           │            ↓
                           └───── (N) ABS_ORCHESTRATION_RULES
                                        │
                                        │ (1)
                                        │
                                        ↓
                                  (N) ABS_API_COMPARISONS
```

**주요 변경사항:**
1. **스키마 접두사**: `MAP.ABS_*` (네임스페이스 분리)
2. **match_rate 정밀도**: `NUMBER(5,4)` → 0.0000~1.0000 (기존 0.00~100.00에서 변경)
3. **JSON 저장**: `transition_config`, `comparison_config`, `differences` → CLOB 타입
4. **외래키 관계**: Cascade 삭제 지원

### Redis 데이터 구조

```go
package cache

// Cache Keys
type CacheKeys struct {
    Mapping      string // "mapping:{client_path}"
    MatchRate    string // "matchrate:{mapping_id}"
    CircuitState string // "circuit:{service}:{endpoint}"
}

// Cache TTL
const (
    MappingCacheTTL      = 10 * time.Minute
    MatchRateCacheTTL    = 1 * time.Minute
    CircuitStateCacheTTL = 30 * time.Second
)

// Redis 명령어 예시
func ExampleRedisCommands() {
    // 매핑 캐시
    rdb.Set(ctx, "mapping:/api/users", mappingJSON, MappingCacheTTL)
    
    // 일치율 임시 집계 (Sorted Set)
    rdb.ZAdd(ctx, "matchrate:mapping-id-1", &redis.Z{
        Score:  float64(time.Now().Unix()),
        Member: 98.5,
    })
    
    // 최근 100건 조회
    rates := rdb.ZRange(ctx, "matchrate:mapping-id-1", -100, -1)
    
    // Circuit Breaker 상태
    rdb.Set(ctx, "circuit:legacy:/api/users", "open", CircuitStateCacheTTL)
}
```

### Prometheus 메트릭

```go
package metrics

import "github.com/prometheus/client_golang/prometheus"

// 커스텀 메트릭 정의
var (
    // API 호출 카운터
    apiCallsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "api_bridge_calls_total",
            Help: "Total number of API calls",
        },
        []string{"mapping_id", "target", "status"},
    )
    
    // 응답 시간 히스토그램
    apiDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "api_bridge_duration_seconds",
            Help:    "API call duration in seconds",
            Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
        },
        []string{"mapping_id", "target"},
    )
    
    // 일치율 게이지
    matchRateGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "api_bridge_match_rate",
            Help: "Current match rate percentage",
        },
        []string{"mapping_id"},
    )
    
    // 전환율 게이지
    transitionRateGauge = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "api_bridge_transition_rate",
            Help: "Percentage of APIs transitioned to modern",
        },
    )
    
    // Circuit Breaker 상태 (0=closed, 1=open, 2=half-open)
    circuitBreakerState = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "api_bridge_circuit_breaker_state",
            Help: "Circuit breaker state",
        },
        []string{"target"},
    )
    
    // 비교 결과 카운터
    comparisonResultTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "api_bridge_comparison_result_total",
            Help: "Total number of comparison results",
        },
        []string{"mapping_id", "result"}, // result: match, mismatch
    )
)

// 메트릭 등록
func RegisterMetrics() {
    prometheus.MustRegister(
        apiCallsTotal,
        apiDuration,
        matchRateGauge,
        transitionRateGauge,
        circuitBreakerState,
        comparisonResultTotal,
    )
}

// 메트릭 사용 예시
func RecordAPICall(mappingID, target, status string, duration time.Duration) {
    apiCallsTotal.WithLabelValues(mappingID, target, status).Inc()
    apiDuration.WithLabelValues(mappingID, target).Observe(duration.Seconds())
}

func UpdateMatchRate(mappingID string, matchRate float64) {
    matchRateGauge.WithLabelValues(mappingID).Set(matchRate)
}

func RecordComparison(mappingID string, isMatch bool) {
    result := "match"
    if !isMatch {
        result = "mismatch"
    }
    comparisonResultTotal.WithLabelValues(mappingID, result).Inc()
}
```

---

## 참고 사항

### 외부 라이브러리

```go
// go.mod
module github.com/yourorg/api-bridge

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/sony/gobreaker v0.5.0
    github.com/prometheus/client_golang v1.17.0
    github.com/redis/go-redis/v9 v9.3.0
    github.com/spf13/viper v1.17.0
    go.uber.org/zap v1.26.0
    github.com/google/uuid v1.4.0
)
```

### 설정 파일 예시

**실제 config.yaml 구조** (`config/config.yaml`):

```yaml
# API Bridge 실제 운영 설정 파일

server:
  port: 10019
  mode: release  # debug, release
  read_timeout: 10s   # ⚠️ 30s → 10s로 변경
  write_timeout: 10s  # ⚠️ 30s → 10s로 변경
  max_header_bytes: 1048576  # 1MB

log:
  level: info  # debug, info, warn, error
  format: json  # json, console
  output: stdout  # stdout, file
  file_path: ./logs/app.log

# Oracle Database 설정
database:
  host: dev1-db.konadc.com
  port: 15322
  sid: kmdbp19
  username: "map"
  password: "StgMAP1104#"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
  connection_timeout: 10s

# Redis Cache 설정
redis:
  host: dev3.konadc.com
  port: 6379
  password: "123456"
  db: 0
  pool_size: 10
  min_idle_conns: 5
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s

# 외부 API 설정
external_api:
  base_url: https://api.example.com
  timeout: 30s
  retry_count: 3
  retry_delay: 1s
  max_retry_delay: 10s
  retry_backoff_multiplier: 2.0

# Circuit Breaker 설정
circuit_breaker:
  max_requests: 5      # ⚠️ 3 → 5로 변경
  interval: 10s
  timeout: 5s          # ⚠️ 30s → 5s로 변경

# 모니터링
metrics:
  enabled: true
  port: 9090
  path: /metrics

# 캐시 설정
cache:
  default_ttl: 300s  # 5분
  routing_rules_ttl: 3600s  # 1시간
  api_response_ttl: 600s  # 10분

# ⚠️ 중요: API 엔드포인트 설정 (메모리 기반, DB 조회 불필요)
endpoints:
  endpoints:
    # Legacy API 엔드포인트
    legacy-api:
      id: legacy-api
      name: Legacy API
      description: Legacy API Information
      base_url: http://dev3.konadc.com:10010
      health_url: /mobile-platform-1.0/api/health
      is_active: true
      is_legacy: true
      is_default: true  # 기본 레거시 엔드포인트로 지정
      timeout: 5s
      retry:
        max_attempts: 3
        initial_delay: 1s
        max_delay: 10s
        backoff_multiplier: 2.0
        retryable_http_codes: [500, 502, 503, 504]

    # Modern API 엔드포인트
    modern-user-api:
      id: modern-api
      name: Modern API
      description: Modern API Information
      base_url: http://dev3.konadc.com:10010
      health_url: /mobile-platform-1.0/api/health
      is_active: true
      is_legacy: false
      is_default: true  # 기본 모던 엔드포인트로 지정
      timeout: 3s
      retry:
        max_attempts: 3
        initial_delay: 500ms
        max_delay: 5s
        backoff_multiplier: 2.0
        retryable_http_codes: [500, 502, 503, 504]
```

**주요 변경사항:**
1. **Endpoints 섹션 추가**: Config 파일에서 직접 로드 (DB 조회 불필요)
2. **Timeout 단축**: Read/Write timeout 30s → 10s
3. **Circuit Breaker 완화**: max_requests 3 → 5, timeout 30s → 5s
4. **Redis 상세 설정**: connection pool, timeout 등 세분화

---

이 문서는 실제 개발 시 참고용으로 사용되며, 프로젝트 진행 중 지속적으로 업데이트됩니다.
