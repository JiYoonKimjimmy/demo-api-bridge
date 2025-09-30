# ABS - 헥사고날 아키텍처 설계

API Bridge System (ABS)의 헥사고날 아키텍처 설계 및 구현 가이드

---

## 📋 프로젝트 정보

- **프로젝트명**: API Bridge System (ABS)
- **약자**: ABS
- **아키텍처**: Hexagonal Architecture (Ports & Adapters)
- **언어**: Go 1.21+
- **프레임워크**: Gin

---

## 🏗️ 헥사고날 아키텍처 개요

### 핵심 개념

```
                    외부 세계
                       ↓
              ┌────────────────┐
              │ Primary Adapter│  (HTTP, CLI 등)
              └────────┬───────┘
                       ↓
              ┌────────────────┐
              │ Primary Port   │  (Inbound Interface)
              └────────┬───────┘
                       ↓
         ┌─────────────────────────┐
         │                         │
         │    Domain (Core)        │  비즈니스 로직
         │    순수한 Go 코드        │  
         │                         │
         └─────────┬───────────────┘
                   ↓
         ┌─────────────────┐
         │ Secondary Port  │  (Outbound Interface)
         └─────────┬───────┘
                   ↓
         ┌─────────────────┐
         │Secondary Adapter│  (DB, Cache, API Client 등)
         └─────────────────┘
                   ↓
               외부 세계
```

### 의존성 방향

```
Primary Adapter → Primary Port → Domain ← Secondary Port ← Secondary Adapter
                                   ↑
                              모든 의존성은
                              Domain을 향함
```

---

## 📁 상세 디렉토리 구조

```
api-bridge-system/
│
├── cmd/
│   └── api-bridge/
│       └── main.go                           # 애플리케이션 진입점, DI 설정
│
├── internal/
│   │
│   ├── domain/                               # 🔵 Domain Layer (Core)
│   │   │
│   │   ├── model/                            # 도메인 모델 (엔티티, 값 객체)
│   │   │   ├── api_mapping.go                # API 매핑 엔티티
│   │   │   ├── comparison_result.go          # 비교 결과 값 객체
│   │   │   ├── routing_strategy.go           # 라우팅 전략 (LEGACY_ONLY, PARALLEL, MODERN_ONLY)
│   │   │   ├── difference.go                 # 차이점 값 객체
│   │   │   └── event/
│   │   │       ├── transition_event.go       # 전환 이벤트
│   │   │       └── rollback_event.go         # 롤백 이벤트
│   │   │
│   │   ├── service/                          # 도메인 서비스 (비즈니스 로직)
│   │   │   ├── routing_service.go            # 라우팅 전략 결정
│   │   │   ├── orchestration_service.go      # 병렬 호출 조율
│   │   │   ├── comparison_service.go         # 응답 비교 로직
│   │   │   └── decision_service.go           # 전환 결정 로직
│   │   │
│   │   └── port/                             # 🔌 Ports (인터페이스)
│   │       │
│   │       ├── inbound/                      # Primary Ports (애플리케이션으로 들어오는 포트)
│   │       │   ├── bridge_service.go         # 브리지 서비스 포트
│   │       │   └── admin_service.go          # 관리 서비스 포트
│   │       │
│   │       └── outbound/                     # Secondary Ports (애플리케이션에서 나가는 포트)
│   │           ├── api_client.go             # API 호출 포트
│   │           ├── mapping_repository.go     # 매핑 저장소 포트
│   │           ├── comparison_repository.go  # 비교 이력 저장소 포트
│   │           ├── cache.go                  # 캐시 포트
│   │           ├── event_publisher.go        # 이벤트 발행 포트
│   │           └── metrics.go                # 메트릭 포트
│   │
│   ├── adapter/                              # 🔌 Adapters (구현체)
│   │   │
│   │   ├── primary/                          # Primary Adapters (Driving)
│   │   │   │
│   │   │   ├── http/                         # HTTP Adapter (Gin)
│   │   │   │   ├── server.go                 # Gin 서버 초기화
│   │   │   │   ├── router.go                 # 라우팅 설정
│   │   │   │   ├── handler/
│   │   │   │   │   ├── bridge_handler.go     # 브리지 핸들러
│   │   │   │   │   ├── health_handler.go     # Health Check
│   │   │   │   │   └── admin_handler.go      # 관리 API
│   │   │   │   ├── middleware/
│   │   │   │   │   ├── logger.go             # 로깅 미들웨어
│   │   │   │   │   ├── cors.go               # CORS 미들웨어
│   │   │   │   │   ├── rate_limiter.go       # Rate Limiting
│   │   │   │   │   └── recovery.go           # Panic Recovery
│   │   │   │   └── dto/
│   │   │   │       ├── request.go            # HTTP 요청 DTO
│   │   │   │       └── response.go           # HTTP 응답 DTO
│   │   │   │
│   │   │   └── cli/                          # CLI Adapter (선택)
│   │   │       └── command.go
│   │   │
│   │   └── secondary/                        # Secondary Adapters (Driven)
│   │       │
│   │       ├── httpclient/                   # HTTP Client Adapter
│   │       │   ├── legacy_client.go          # 레거시 API 클라이언트
│   │       │   ├── modern_client.go          # 모던 API 클라이언트
│   │       │   ├── client.go                 # 공통 HTTP Client
│   │       │   └── circuit_breaker.go        # Circuit Breaker 적용
│   │       │
│   │       ├── repository/                   # Repository Adapter
│   │       │   └── oracle/
│   │       │       ├── mapping_repository.go # 매핑 저장소 구현
│   │       │       ├── comparison_repository.go
│   │       │       ├── transition_repository.go
│   │       │       └── entity/               # DB 엔티티 (GORM)
│   │       │           ├── api_mapping.go
│   │       │           ├── comparison_history.go
│   │       │           └── transition_history.go
│   │       │
│   │       ├── cache/                        # Cache Adapter
│   │       │   └── redis/
│   │       │       ├── cache.go              # Redis 캐시 구현
│   │       │       └── client.go             # Redis 클라이언트
│   │       │
│   │       ├── event/                        # Event Publisher Adapter
│   │       │   └── publisher.go              # 이벤트 발행 구현
│   │       │
│   │       └── metrics/                      # Metrics Adapter
│   │           └── prometheus/
│   │               └── collector.go          # Prometheus 메트릭 수집
│   │
│   └── infrastructure/                       # 🔧 Infrastructure
│       ├── config/
│       │   ├── config.go                     # 설정 구조체
│       │   └── loader.go                     # Viper 설정 로더
│       ├── database/
│       │   └── oracle.go                     # OracleDB 연결
│       ├── logger/
│       │   └── zap.go                        # Zap 로거 설정
│       └── telemetry/
│           └── tracer.go                     # OpenTelemetry 설정
│
├── pkg/                                      # 공용 유틸리티
│   ├── errors/
│   │   └── errors.go                         # 커스텀 에러
│   ├── validator/
│   │   └── validator.go                      # 검증기
│   └── util/
│       ├── json.go                           # JSON 유틸
│       └── time.go                           # 시간 유틸
│
├── configs/                                  # 설정 파일
│   ├── config.yaml
│   ├── config-dev.yaml
│   ├── config-stg.yaml
│   └── config-prod.yaml
│
├── scripts/                                  # 운영 스크립트
│   ├── start.sh
│   ├── stop.sh
│   ├── restart.sh
│   ├── status.sh
│   ├── watchdog.sh
│   └── deploy.sh
│
├── test/                                     # 테스트
│   ├── integration/
│   └── mock/
│       ├── mock_api_client.go
│       ├── mock_repository.go
│       └── mock_cache.go
│
├── docs/                                     # 문서
│   ├── IMPLEMENTATION_GUIDE.md
│   ├── DEPLOYMENT_GUIDE.md
│   ├── DEPLOYMENT_PLAN.md
│   ├── FRAMEWORK_COMPARISON.md
│   └── HEXAGONAL_ARCHITECTURE.md
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 🔵 Domain Layer 상세 설계

### 1. Domain Model

#### api_mapping.go
```go
// internal/domain/model/api_mapping.go
package model

import "time"

// APIMapping은 레거시 API와 모던 API의 매핑 정보를 나타냅니다.
type APIMapping struct {
    ID          string
    ClientPath  string
    LegacyURL   string
    ModernURL   string
    Strategy    RoutingStrategy
    MatchRate   float64
    Threshold   float64
    IsActive    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// ShouldTransition은 전환 조건을 만족하는지 확인합니다.
func (m *APIMapping) ShouldTransition() bool {
    return m.Strategy == PARALLEL && m.MatchRate >= m.Threshold
}

// CanRollback은 롤백 가능한지 확인합니다.
func (m *APIMapping) CanRollback() bool {
    return m.Strategy == MODERN_ONLY
}
```

#### routing_strategy.go
```go
// internal/domain/model/routing_strategy.go
package model

type RoutingStrategy string

const (
    LEGACY_ONLY RoutingStrategy = "legacy_only"
    PARALLEL    RoutingStrategy = "parallel"
    MODERN_ONLY RoutingStrategy = "modern_only"
)

func (s RoutingStrategy) IsValid() bool {
    switch s {
    case LEGACY_ONLY, PARALLEL, MODERN_ONLY:
        return true
    default:
        return false
    }
}
```

#### comparison_result.go
```go
// internal/domain/model/comparison_result.go
package model

import "time"

type ComparisonResult struct {
    MappingID   string
    IsMatch     bool
    MatchRate   float64
    Differences []Difference
    TraceID     string
    Timestamp   time.Time
}

type Difference struct {
    Path      string      // JSON 경로 (예: "data.user.email")
    LegacyVal interface{}
    ModernVal interface{}
    DiffType  DifferenceType
}

type DifferenceType string

const (
    MISSING        DifferenceType = "MISSING"
    EXTRA          DifferenceType = "EXTRA"
    VALUE_MISMATCH DifferenceType = "VALUE_MISMATCH"
    TYPE_MISMATCH  DifferenceType = "TYPE_MISMATCH"
)
```

---

### 2. Domain Service

#### routing_service.go
```go
// internal/domain/service/routing_service.go
package service

import (
    "context"
    "github.com/yourorg/api-bridge-system/internal/domain/model"
    "github.com/yourorg/api-bridge-system/internal/domain/port/outbound"
)

type RoutingService struct {
    mappingRepo outbound.MappingRepository
    cache       outbound.Cache
}

func NewRoutingService(repo outbound.MappingRepository, cache outbound.Cache) *RoutingService {
    return &RoutingService{
        mappingRepo: repo,
        cache:       cache,
    }
}

func (s *RoutingService) GetMapping(ctx context.Context, clientPath string) (*model.APIMapping, error) {
    // 1. 캐시 조회
    cacheKey := "mapping:" + clientPath
    if mapping, err := s.cache.Get(ctx, cacheKey); err == nil {
        return mapping.(*model.APIMapping), nil
    }
    
    // 2. DB 조회
    mapping, err := s.mappingRepo.FindByClientPath(ctx, clientPath)
    if err != nil {
        return nil, err
    }
    
    // 3. 캐시 저장
    s.cache.Set(ctx, cacheKey, mapping, 10*time.Minute)
    
    return mapping, nil
}

func (s *RoutingService) DetermineStrategy(mapping *model.APIMapping) model.RoutingStrategy {
    // 비즈니스 규칙: 일치율이 임계값 이상이면 모던만 사용
    if mapping.MatchRate >= mapping.Threshold {
        return model.MODERN_ONLY
    }
    return mapping.Strategy
}
```

---

## 🔌 Port 정의

### Primary Ports (Inbound)

```go
// internal/domain/port/inbound/bridge_service.go
package inbound

import (
    "context"
    "github.com/yourorg/api-bridge-system/internal/domain/model"
)

// BridgeService는 API 브리지의 핵심 기능을 제공합니다.
type BridgeService interface {
    // HandleRequest는 클라이언트 요청을 처리합니다.
    HandleRequest(ctx context.Context, req *BridgeRequest) (*BridgeResponse, error)
    
    // GetMappingStatus는 API 매핑 상태를 조회합니다.
    GetMappingStatus(ctx context.Context, mappingID string) (*MappingStatus, error)
}

type BridgeRequest struct {
    ClientPath string
    Method     string
    Headers    map[string]string
    Body       []byte
    TraceID    string
}

type BridgeResponse struct {
    StatusCode int
    Headers    map[string]string
    Body       []byte
    Source     string  // "legacy" or "modern"
}

type MappingStatus struct {
    Mapping     *model.APIMapping
    MatchRate   float64
    TotalCalls  int64
    Transitoned bool
}
```

### Secondary Ports (Outbound)

```go
// internal/domain/port/outbound/api_client.go
package outbound

import "context"

type APIClient interface {
    Call(ctx context.Context, url string, req *APIRequest) (*APIResponse, error)
}

type APIRequest struct {
    Method  string
    Headers map[string]string
    Body    []byte
}

type APIResponse struct {
    StatusCode int
    Headers    map[string]string
    Body       []byte
    Duration   time.Duration
}
```

```go
// internal/domain/port/outbound/mapping_repository.go
package outbound

import (
    "context"
    "github.com/yourorg/api-bridge-system/internal/domain/model"
)

type MappingRepository interface {
    FindByClientPath(ctx context.Context, path string) (*model.APIMapping, error)
    FindByID(ctx context.Context, id string) (*model.APIMapping, error)
    Save(ctx context.Context, mapping *model.APIMapping) error
    UpdateStrategy(ctx context.Context, id string, strategy model.RoutingStrategy) error
    UpdateMatchRate(ctx context.Context, id string, matchRate float64) error
    ListAll(ctx context.Context) ([]*model.APIMapping, error)
}
```

```go
// internal/domain/port/outbound/cache.go
package outbound

import (
    "context"
    "time"
)

type Cache interface {
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

---

## 🔧 Adapter 구현

### Primary Adapter (HTTP)

```go
// internal/adapter/primary/http/handler/bridge_handler.go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/yourorg/api-bridge-system/internal/domain/port/inbound"
)

type BridgeHandler struct {
    bridgeService inbound.BridgeService  // Port에 의존
}

func NewBridgeHandler(service inbound.BridgeService) *BridgeHandler {
    return &BridgeHandler{
        bridgeService: service,
    }
}

func (h *BridgeHandler) Handle(c *gin.Context) {
    // 1. HTTP 요청을 Domain Request로 변환
    req := &inbound.BridgeRequest{
        ClientPath: c.Request.URL.Path,
        Method:     c.Request.Method,
        Headers:    extractHeaders(c),
        Body:       readBody(c),
        TraceID:    c.GetString("trace_id"),
    }
    
    // 2. Domain Service 호출 (Port 사용)
    resp, err := h.bridgeService.HandleRequest(c.Request.Context(), req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 3. Domain Response를 HTTP 응답으로 변환
    for key, val := range resp.Headers {
        c.Header(key, val)
    }
    c.Data(resp.StatusCode, "application/json", resp.Body)
}
```

### Secondary Adapter (HTTP Client)

```go
// internal/adapter/secondary/httpclient/legacy_client.go
package httpclient

import (
    "context"
    "net/http"
    "github.com/yourorg/api-bridge-system/internal/domain/port/outbound"
)

type LegacyClient struct {
    baseURL string
    client  *http.Client
}

func NewLegacyClient(baseURL string) outbound.APIClient {
    return &LegacyClient{
        baseURL: baseURL,
        client: &http.Client{
            Timeout: 5 * time.Second,
        },
    }
}

// Port 인터페이스 구현
func (c *LegacyClient) Call(ctx context.Context, url string, req *outbound.APIRequest) (*outbound.APIResponse, error) {
    fullURL := c.baseURL + url
    
    httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bytes.NewReader(req.Body))
    if err != nil {
        return nil, err
    }
    
    // 헤더 설정
    for key, val := range req.Headers {
        httpReq.Header.Set(key, val)
    }
    
    // HTTP 호출
    start := time.Now()
    httpResp, err := c.client.Do(httpReq)
    duration := time.Since(start)
    
    if err != nil {
        return nil, err
    }
    defer httpResp.Body.Close()
    
    body, _ := io.ReadAll(httpResp.Body)
    
    // Domain 모델로 변환
    return &outbound.APIResponse{
        StatusCode: httpResp.StatusCode,
        Headers:    extractHeaders(httpResp),
        Body:       body,
        Duration:   duration,
    }, nil
}
```

### Secondary Adapter (Repository)

```go
// internal/adapter/secondary/repository/oracle/mapping_repository.go
package oracle

import (
    "context"
    "gorm.io/gorm"
    "github.com/yourorg/api-bridge-system/internal/domain/model"
    "github.com/yourorg/api-bridge-system/internal/domain/port/outbound"
)

type MappingRepository struct {
    db *gorm.DB
}

func NewMappingRepository(db *gorm.DB) outbound.MappingRepository {
    return &MappingRepository{db: db}
}

// Port 인터페이스 구현
func (r *MappingRepository) FindByClientPath(ctx context.Context, path string) (*model.APIMapping, error) {
    var entity APIMappingEntity
    
    err := r.db.WithContext(ctx).
        Where("client_path = ? AND is_active = 1", path).
        First(&entity).Error
    
    if err != nil {
        return nil, err
    }
    
    // DB Entity를 Domain Model로 변환
    return entity.ToDomain(), nil
}

func (r *MappingRepository) UpdateStrategy(ctx context.Context, id string, strategy model.RoutingStrategy) error {
    return r.db.WithContext(ctx).
        Model(&APIMappingEntity{}).
        Where("id = ?", id).
        Update("strategy", string(strategy)).Error
}
```

---

## 🔗 의존성 주입 (DI)

### main.go

```go
// cmd/api-bridge/main.go
package main

import (
    "flag"
    "log"
    
    "github.com/gin-gonic/gin"
    
    // Domain
    "github.com/yourorg/api-bridge-system/internal/domain/service"
    
    // Primary Adapters
    httpHandler "github.com/yourorg/api-bridge-system/internal/adapter/primary/http/handler"
    
    // Secondary Adapters
    "github.com/yourorg/api-bridge-system/internal/adapter/secondary/httpclient"
    "github.com/yourorg/api-bridge-system/internal/adapter/secondary/repository/oracle"
    redisCache "github.com/yourorg/api-bridge-system/internal/adapter/secondary/cache/redis"
    
    // Infrastructure
    "github.com/yourorg/api-bridge-system/internal/infrastructure/config"
    "github.com/yourorg/api-bridge-system/internal/infrastructure/database"
    "github.com/yourorg/api-bridge-system/internal/infrastructure/logger"
)

func main() {
    // 플래그 파싱
    bindAddress := flag.String("bind-address", "0.0.0.0", "Bind IP address")
    bindPort := flag.Int("bind-port", 10019, "Bind port")
    configFile := flag.String("config", "config.yaml", "Config file path")
    flag.Parse()
    
    // 1. 설정 로드
    cfg, err := config.Load(*configFile)
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // 2. Infrastructure 초기화
    logger.Init(cfg.Logging)
    db := database.NewOracleDB(cfg.Database)
    redisClient := redis.NewClient(cfg.Redis)
    
    // 3. Secondary Adapters 생성 (Driven)
    legacyClient := httpclient.NewLegacyClient(cfg.Legacy.BaseURL)
    modernClient := httpclient.NewModernClient(cfg.Modern.BaseURL)
    mappingRepo := oracle.NewMappingRepository(db)
    cache := redisCache.NewCache(redisClient)
    
    // 4. Domain Services 생성 (DI)
    routingService := service.NewRoutingService(mappingRepo, cache)
    orchestrationService := service.NewOrchestrationService(legacyClient, modernClient)
    comparisonService := service.NewComparisonService()
    decisionService := service.NewDecisionService(mappingRepo, cache)
    
    bridgeService := service.NewBridgeService(
        routingService,
        orchestrationService,
        comparisonService,
        decisionService,
    )
    
    // 5. Primary Adapters 생성 (Driving)
    bridgeHandler := httpHandler.NewBridgeHandler(bridgeService)
    healthHandler := httpHandler.NewHealthHandler(db, redisClient)
    
    // 6. HTTP Server 설정 (Gin)
    router := gin.Default()
    
    // Middleware
    router.Use(middleware.Logger())
    router.Use(middleware.Recovery())
    
    // Routes
    router.GET("/health", healthHandler.Health)
    router.GET("/ready", healthHandler.Ready)
    router.Any("/*path", bridgeHandler.Handle)
    
    // 7. 서버 시작
    listenAddr := fmt.Sprintf("%s:%d", *bindAddress, *bindPort)
    log.Printf("Starting ABS on %s", listenAddr)
    
    if err := router.Run(listenAddr); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}
```

---

## 🧪 테스트 전략

### 1. Domain Service 테스트 (Mock 사용)

```go
// internal/domain/service/routing_service_test.go
package service_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockMappingRepository struct {
    mock.Mock
}

func (m *MockMappingRepository) FindByClientPath(ctx context.Context, path string) (*model.APIMapping, error) {
    args := m.Called(ctx, path)
    return args.Get(0).(*model.APIMapping), args.Error(1)
}

func TestRoutingService_GetMapping(t *testing.T) {
    // Given
    mockRepo := new(MockMappingRepository)
    mockCache := new(MockCache)
    
    expected := &model.APIMapping{
        ID:         "mapping-1",
        ClientPath: "/api/users",
        Strategy:   model.PARALLEL,
    }
    
    mockRepo.On("FindByClientPath", mock.Anything, "/api/users").Return(expected, nil)
    mockCache.On("Get", mock.Anything, "mapping:/api/users").Return(nil, errors.New("not found"))
    mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
    
    service := NewRoutingService(mockRepo, mockCache)
    
    // When
    result, err := service.GetMapping(context.Background(), "/api/users")
    
    // Then
    assert.NoError(t, err)
    assert.Equal(t, expected.ID, result.ID)
    mockRepo.AssertExpectations(t)
}
```

---

## 📝 헥사고날 아키텍처 원칙

### 1. Domain은 순수해야 함
```go
// ✅ Good: Domain에 기술 의존성 없음
package service

type ComparisonService struct {
    // 순수 Go 타입만
}

func (s *ComparisonService) Compare(legacy, modern []byte) (*model.ComparisonResult, error) {
    // 순수 로직
}

// ❌ Bad: Domain에 기술 의존성
package service

import "github.com/gin-gonic/gin"  // ❌ HTTP 프레임워크 의존

type ComparisonService struct {
    router *gin.Engine  // ❌ 기술 상세에 의존
}
```

### 2. Port는 Domain에 위치
```go
// ✅ Good: Port는 Domain이 정의
// internal/domain/port/outbound/cache.go
package outbound

type Cache interface {
    Get(ctx context.Context, key string) (interface{}, error)
}

// ❌ Bad: Adapter가 Port 정의
// internal/adapter/secondary/cache/redis/port.go  ❌ 잘못된 위치
```

### 3. 의존성은 항상 안쪽으로
```go
// ✅ Good
Adapter (HTTP Handler) → Port (Interface) → Domain Service

// ❌ Bad
Domain Service → Adapter (Redis Client)  // 직접 의존 ❌
```

---

## 🎯 ABS 헥사고날 적용 이점

### 1. 테스트 용이성
```go
// Domain 테스트: Mock만으로 완전한 테스트
func TestDecisionService_ShouldTransition(t *testing.T) {
    mockRepo := &MockRepository{}
    service := NewDecisionService(mockRepo)
    // 실제 DB 불필요
}
```

### 2. 기술 스택 교체
```
OracleDB → PostgreSQL:
- Domain: 변경 없음
- Port: 변경 없음
- Adapter: PostgreSQL Adapter만 추가

Redis → Memcached:
- Domain: 변경 없음
- Port: 변경 없음
- Adapter: Memcached Adapter만 추가
```

### 3. 비즈니스 로직 명확성
```
/internal/domain/service/
  - 비즈니스 규칙만 집중
  - HTTP, DB, 캐시 등 기술 상세 배제
  - 읽기 쉽고 이해하기 쉬움
```

### 4. 확장성
```
새로운 프로토콜 추가 (gRPC):
- gRPC Primary Adapter만 추가
- Domain 로직 재사용
```

---

## 🚀 초기화 순서

### Step 1: 프로젝트 초기화
```bash
# Go 모듈 초기화
go mod init github.com/yourorg/api-bridge-system

# 디렉토리 생성
mkdir -p cmd/api-bridge
mkdir -p internal/{domain/{model,service,port/{inbound,outbound}},adapter/{primary/http/{handler,middleware},secondary/{httpclient,repository/oracle,cache/redis}},infrastructure/{config,database,logger}}
mkdir -p pkg/{errors,validator}
mkdir -p configs
mkdir -p scripts
mkdir -p test/{integration,mock}
```

### Step 2: 기본 파일 생성
```bash
# main.go
touch cmd/api-bridge/main.go

# Domain Models
touch internal/domain/model/{api_mapping,routing_strategy,comparison_result}.go

# Ports
touch internal/domain/port/inbound/bridge_service.go
touch internal/domain/port/outbound/{api_client,mapping_repository,cache}.go

# go.mod 의존성 추가
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get github.com/redis/go-redis/v9
go get github.com/prometheus/client_golang/prometheus
go get go.uber.org/zap
```

### Step 3: 기본 구현
1. Domain Model 정의
2. Port 인터페이스 정의
3. Health Check Handler 구현
4. main.go DI 설정

---

## 📚 참고 자료

- **Hexagonal Architecture**: Alistair Cockburn
- **Clean Architecture**: Robert C. Martin
- **Ports and Adapters Pattern**

---

이 문서는 ABS 개발 시 헥사고날 아키텍처 가이드로 사용됩니다.
