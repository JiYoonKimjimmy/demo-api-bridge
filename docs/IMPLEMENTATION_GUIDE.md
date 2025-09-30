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

// HTTP Server 초기화
func NewAPIServer(config *Config) *gin.Engine {
    router := gin.New()
    
    // Middleware 등록
    router.Use(
        RequestLoggerMiddleware(),
        CORSMiddleware(),
        RateLimiterMiddleware(),
        AuthenticationMiddleware(),
        RequestValidatorMiddleware(),
    )
    
    // Health Check
    router.GET("/health", HealthCheckHandler)
    router.GET("/ready", ReadinessCheckHandler)
    
    // API 브리지 엔드포인트
    router.Any("/*path", BridgeHandler)
    
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
    limiter := rate.NewLimiter(1000, 2000) // 1000 req/sec, burst 2000
    
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{
                "error": "Too Many Requests",
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
package routing

// Routing Service
type RoutingService struct {
    mappingRepo  MappingRepository
    cache        Cache
    strategyMgr  StrategyManager
    metrics      *RoutingMetrics
}

// API Mapping
type APIMapping struct {
    ID            string          `json:"id" db:"id"`
    ClientPath    string          `json:"client_path" db:"client_path"`
    LegacyURL     string          `json:"legacy_url" db:"legacy_url"`
    ModernURL     string          `json:"modern_url" db:"modern_url"`
    Strategy      RoutingStrategy `json:"strategy" db:"strategy"`
    MatchRate     float64         `json:"match_rate" db:"match_rate"`
    Threshold     float64         `json:"threshold" db:"threshold"`
    IsActive      bool            `json:"is_active" db:"is_active"`
    CreatedAt     time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// Routing Strategy
type RoutingStrategy string

const (
    LEGACY_ONLY  RoutingStrategy = "legacy_only"   // 레거시만 호출
    PARALLEL     RoutingStrategy = "parallel"       // 병렬 호출 + 검증
    MODERN_ONLY  RoutingStrategy = "modern_only"   // 모던만 호출
)
```

### 라우팅 로직

```go
// 라우팅 결정
func (s *RoutingService) Route(ctx context.Context, clientPath string) (*APIMapping, error) {
    // 1. 캐시 조회
    cacheKey := fmt.Sprintf("mapping:%s", clientPath)
    if mapping, err := s.cache.Get(cacheKey); err == nil {
        return mapping, nil
    }
    
    // 2. DB 조회
    mapping, err := s.mappingRepo.FindByClientPath(ctx, clientPath)
    if err != nil {
        return nil, fmt.Errorf("mapping not found: %w", err)
    }
    
    // 3. 캐시 저장
    s.cache.Set(cacheKey, mapping, 10*time.Minute)
    
    return mapping, nil
}

// URL 매칭 (Radix Tree 사용)
type RadixTree struct {
    root *node
}

type node struct {
    prefix   string
    mapping  *APIMapping
    children map[string]*node
}

func (t *RadixTree) Match(path string) (*APIMapping, error) {
    current := t.root
    remaining := path
    
    for {
        if current.mapping != nil && remaining == "" {
            return current.mapping, nil
        }
        
        matched := false
        for prefix, child := range current.children {
            if strings.HasPrefix(remaining, prefix) {
                current = child
                remaining = strings.TrimPrefix(remaining, prefix)
                matched = true
                break
            }
        }
        
        if !matched {
            return nil, fmt.Errorf("no mapping found for path: %s", path)
        }
    }
}
```

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
package orchestration

import "github.com/sony/gobreaker"

// Circuit Breaker
type CircuitBreaker struct {
    legacyBreaker  *gobreaker.CircuitBreaker
    modernBreaker  *gobreaker.CircuitBreaker
}

// Circuit Breaker 초기화
func NewCircuitBreaker() *CircuitBreaker {
    settings := gobreaker.Settings{
        Name:        "API",
        MaxRequests: 3,           // Half-Open 상태에서 허용 요청 수
        Interval:    10 * time.Second,
        Timeout:     30 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 5 && failureRatio >= 0.6
        },
        OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
            logger.Warn("circuit breaker state changed",
                "name", name,
                "from", from.String(),
                "to", to.String(),
            )
        },
    }
    
    return &CircuitBreaker{
        legacyBreaker:  gobreaker.NewCircuitBreaker(settings),
        modernBreaker:  gobreaker.NewCircuitBreaker(settings),
    }
}

// 실행
func (cb *CircuitBreaker) ExecuteLegacy(fn func() (interface{}, error)) (interface{}, error) {
    return cb.legacyBreaker.Execute(fn)
}

func (cb *CircuitBreaker) ExecuteModern(fn func() (interface{}, error)) (interface{}, error) {
    return cb.modernBreaker.Execute(fn)
}
```

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
package client

import (
    "math"
    "math/rand"
    "time"
)

// Retry Policy
type RetryPolicy struct {
    MaxRetries   int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    Jitter       bool
}

// 지연 시간 계산 (Exponential Backoff with Jitter)
func (p *RetryPolicy) CalculateDelay(attempt int) time.Duration {
    delay := float64(p.InitialDelay) * math.Pow(p.Multiplier, float64(attempt))
    
    if delay > float64(p.MaxDelay) {
        delay = float64(p.MaxDelay)
    }
    
    if p.Jitter {
        // Full Jitter
        delay = rand.Float64() * delay
    }
    
    return time.Duration(delay)
}
```

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

---

## 7. Data Layer

### OracleDB 스키마

```sql
-- API 매핑 테이블
CREATE TABLE api_mappings (
    id VARCHAR2(36) PRIMARY KEY,
    client_path VARCHAR2(500) NOT NULL,
    legacy_url VARCHAR2(1000) NOT NULL,
    modern_url VARCHAR2(1000) NOT NULL,
    strategy VARCHAR2(20) NOT NULL,
    match_rate NUMBER(5,2) DEFAULT 0,
    threshold NUMBER(5,2) DEFAULT 100,
    is_active NUMBER(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_strategy CHECK (strategy IN ('legacy_only', 'parallel', 'modern_only')),
    CONSTRAINT uk_client_path UNIQUE (client_path)
);

-- 비교 결과 이력 테이블
CREATE TABLE comparison_history (
    id VARCHAR2(36) PRIMARY KEY,
    mapping_id VARCHAR2(36) NOT NULL,
    is_match NUMBER(1) NOT NULL,
    match_rate NUMBER(5,2) NOT NULL,
    differences CLOB,
    trace_id VARCHAR2(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_comparison_mapping FOREIGN KEY (mapping_id) REFERENCES api_mappings(id)
);

-- 전환 이력 테이블
CREATE TABLE transition_history (
    id VARCHAR2(36) PRIMARY KEY,
    mapping_id VARCHAR2(36) NOT NULL,
    from_strategy VARCHAR2(20) NOT NULL,
    to_strategy VARCHAR2(20) NOT NULL,
    reason VARCHAR2(500),
    performed_by VARCHAR2(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_transition_mapping FOREIGN KEY (mapping_id) REFERENCES api_mappings(id)
);

-- 인덱스
CREATE INDEX idx_mappings_path ON api_mappings(client_path);
CREATE INDEX idx_mappings_strategy ON api_mappings(strategy, is_active);
CREATE INDEX idx_comparison_mapping ON comparison_history(mapping_id, created_at);
CREATE INDEX idx_comparison_created ON comparison_history(created_at);
CREATE INDEX idx_transition_mapping ON transition_history(mapping_id, created_at);

-- 코멘트
COMMENT ON TABLE api_mappings IS 'API 매핑 정보';
COMMENT ON TABLE comparison_history IS '응답 비교 결과 이력';
COMMENT ON TABLE transition_history IS 'API 전환 이력';
```

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

```yaml
# config.yaml
server:
  bind_address: "192.168.1.101"
  bind_port: 10019
  read_timeout: 30s
  write_timeout: 30s

legacy:
  base_url: "http://legacy-api.example.com"
  timeout: 5s

modern:
  base_url: "http://modern-api.example.com"
  timeout: 5s

database:
  driver: "oracle"
  dsn: "oracle://user:password@localhost:1521/ORCL"
  max_open_conns: 25
  max_idle_conns: 5

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

circuit_breaker:
  max_requests: 3
  interval: 10s
  timeout: 30s
  failure_threshold: 0.6

comparison:
  excluded_fields:
    - "timestamp"
    - "requestId"
    - "trace_id"
  numeric_tolerance: 0.01
  ignore_order: false

logging:
  level: "info"
  format: "json"
  output: "/opt/api-bridge/logs/app.log"
```

---

이 문서는 실제 개발 시 참고용으로 사용되며, 프로젝트 진행 중 지속적으로 업데이트됩니다.
