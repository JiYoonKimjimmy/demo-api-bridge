package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // pprof 프로파일링 엔드포인트 활성화
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "demo-api-bridge/internal/adapter/inbound/http"
	"demo-api-bridge/internal/adapter/outbound/cache"
	"demo-api-bridge/internal/adapter/outbound/database"
	"demo-api-bridge/internal/adapter/outbound/httpclient"
	"demo-api-bridge/internal/core/port"
	"demo-api-bridge/internal/core/service"
	"demo-api-bridge/pkg/config"
	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	defaultPort = "10019"
	serviceName = "api_bridge"
	version     = "0.1.0"
)

func main() {
	fmt.Printf("Starting %s v%s...\n", serviceName, version)
	fmt.Println("DEBUG: Main function started")

	// 설정 로드
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		fmt.Println("Using default configuration...")
		cfg = config.GetDefaultConfig()
	}

	// 의존성 초기화
	dependencies, err := initializeDependencies(cfg)
	if err != nil {
		fmt.Printf("❌ Failed to initialize dependencies: %v\n", err)
		os.Exit(1)
	}
	defer cleanup(dependencies)

	// Gin 모드 설정
	gin.SetMode(gin.ReleaseMode)

	// 라우터 초기화
	router := gin.New()

	// 미들웨어 설정
	router.Use(gin.Recovery())
	router.Use(httpadapter.NewLoggingMiddleware(dependencies.Logger))
	router.Use(httpadapter.NewMetricsMiddleware(dependencies.Metrics))
	router.Use(httpadapter.NewCORSMiddleware())
	router.Use(httpadapter.NewRateLimitMiddleware())

	// HTTP 핸들러 설정
	httpHandler := httpadapter.NewHandler(
		dependencies.BridgeService,
		dependencies.HealthService,
		dependencies.EndpointService,
		dependencies.RoutingService,
		dependencies.OrchestrationService,
		dependencies.Logger,
	)

	// 라우트 설정
	setupRoutes(router, httpHandler)

	// 포트 설정
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// 서버 시작 (고루틴)
	go func() {
		fmt.Printf("🚀 API Bridge service is now running on port %s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("❌ Failed to start server: %v\n", err)
		}
	}()

	// Graceful Shutdown 설정
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Handler의 shutdown 채널도 함께 처리
	shutdownChannel := httpHandler.GetShutdownChannel()

	// 여러 채널을 동시에 처리
	go func() {
		select {
		case <-quit:
			fmt.Println("Received system signal for shutdown")
		case <-shutdownChannel:
			fmt.Println("Received API shutdown request")
		}
		quit <- os.Interrupt // 다른 고루틴에 신호 전달
	}()

	<-quit

	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server exited")
}

// Dependencies는 애플리케이션의 모든 의존성을 포함합니다.
type Dependencies struct {
	Logger               port.Logger
	Metrics              port.MetricsCollector
	Cache                port.CacheRepository
	RoutingRepo          port.RoutingRepository
	EndpointRepo         port.EndpointRepository
	OrchestrationRepo    port.OrchestrationRepository
	ComparisonRepo       port.ComparisonRepository
	CircuitBreaker       port.CircuitBreakerService
	ExternalAPI          port.ExternalAPIClient
	BridgeService        port.BridgeService
	HealthService        port.HealthCheckService
	EndpointService      port.EndpointService
	RoutingService       port.RoutingService
	OrchestrationService port.OrchestrationService
	RedisClient          *redis.Client
}

// initializeDependencies는 모든 의존성을 초기화합니다.
func initializeDependencies(cfg *config.Config) (*Dependencies, error) {
	// 로거 초기화
	log := logger.NewLogger()

	// 메트릭 초기화
	metricsCollector := metrics.NewMetricsCollector()

	// Redis 클라이언트 초기화
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})

	// Redis 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Warn(fmt.Sprintf("Failed to connect to Redis: %v", err))
		log.Info("Using Mock cache repository instead")
	}

	// 캐시 리포지토리 초기화 (Redis 또는 Mock)
	var cacheRepo port.CacheRepository
	if err := redisClient.Ping(ctx).Err(); err != nil {
		cacheRepo = cache.NewMockCacheRepository()
	} else {
		cacheRepo = cache.NewRedisAdapterWithClient(redisClient)
		log.Info("✅ Redis cache repository initialized")
	}

	// 데이터베이스 리포지토리 초기화 (OracleDB 또는 Mock)
	var routingRepo port.RoutingRepository
	var endpointRepo port.EndpointRepository
	var orchestrationRepo port.OrchestrationRepository
	var comparisonRepo port.ComparisonRepository

	// OracleDB 연결 시도
	oracleRoutingRepo, err := database.NewOracleRoutingRepository(&cfg.Database)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to connect to OracleDB: %v", err))
		log.Info("Using Mock database repositories instead")

		// Mock 리포지토리 사용
		routingRepo = database.NewMockRoutingRepository()
		endpointRepo = database.NewMockEndpointRepository()
		orchestrationRepo = database.NewMockOrchestrationRepository()
		comparisonRepo = database.NewMockComparisonRepository()
	} else {
		// OracleDB 리포지토리 사용
		routingRepo = oracleRoutingRepo

		oracleEndpointRepo, err := database.NewOracleEndpointRepository(&cfg.Database)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to create Oracle endpoint repository: %v", err))
			endpointRepo = database.NewMockEndpointRepository()
		} else {
			endpointRepo = oracleEndpointRepo
		}

		// Orchestration Repository OracleDB 구현
		oracleOrchestrationRepo, err := database.NewOracleOrchestrationRepository(&cfg.Database)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to create Oracle orchestration repository: %v", err))
			orchestrationRepo = database.NewMockOrchestrationRepository()
		} else {
			orchestrationRepo = oracleOrchestrationRepo
		}

		// Comparison Repository OracleDB 구현
		oracleComparisonRepo, err := database.NewOracleComparisonRepository(&cfg.Database)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to create Oracle comparison repository: %v", err))
			comparisonRepo = database.NewMockComparisonRepository()
		} else {
			comparisonRepo = oracleComparisonRepo
		}

		log.Info("✅ OracleDB repositories initialized")
	}

	// Circuit Breaker 서비스 초기화
	circuitBreakerService := service.NewCircuitBreakerService(log, metricsCollector)

	// HTTP 클라이언트 초기화 (Circuit Breaker 포함)
	httpClient := httpclient.NewHTTPClientAdapterWithCircuitBreaker(cfg.ExternalAPI.Timeout, circuitBreakerService)

	// 서비스 초기화
	healthService := service.NewHealthCheckService(routingRepo, endpointRepo, cacheRepo, log)

	endpointService := service.NewEndpointService(endpointRepo, log, metricsCollector)
	routingService := service.NewRoutingService(routingRepo, cacheRepo, log, metricsCollector)

	orchestrationService := service.NewOrchestrationService(
		orchestrationRepo,
		comparisonRepo,
		httpClient,
		log,
		metricsCollector,
	)

	bridgeService := service.NewBridgeService(
		routingRepo,
		endpointRepo,
		orchestrationRepo,
		comparisonRepo,
		orchestrationService,
		httpClient,
		cacheRepo,
		log,
		metricsCollector,
	)

	return &Dependencies{
		Logger:               log,
		Metrics:              metricsCollector,
		Cache:                cacheRepo,
		RoutingRepo:          routingRepo,
		EndpointRepo:         endpointRepo,
		OrchestrationRepo:    orchestrationRepo,
		ComparisonRepo:       comparisonRepo,
		CircuitBreaker:       circuitBreakerService,
		ExternalAPI:          httpClient,
		BridgeService:        bridgeService,
		HealthService:        healthService,
		EndpointService:      endpointService,
		RoutingService:       routingService,
		OrchestrationService: orchestrationService,
		RedisClient:          redisClient,
	}, nil
}

// setupRoutes는 라우트를 설정합니다.
func setupRoutes(router *gin.Engine, handler *httpadapter.Handler) {
	// Swagger YAML 파일 제공
	router.Static("/swagger-yaml", "./api-docs")

	// Swagger UI - swagger.yaml 파일 기반 (절대 URL 사용)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("http://localhost:10019/swagger-yaml/swagger.yaml")))

	// Health Check
	router.GET("/health", handler.HealthCheck)
	router.GET("/ready", handler.ReadinessCheck)
	router.GET("/api/v1/status", handler.Status)

	// Graceful Shutdown
	router.POST("/api/v1/shutdown", handler.GracefulShutdown)

	// API Bridge - 모든 요청 처리
	router.Any("/api/v1/bridge/*path", handler.ProcessBridgeRequest)

	// === CRUD API 엔드포인트 ===

	// APIEndpoint CRUD
	router.POST("/api/v1/endpoints", handler.CreateEndpoint)
	router.GET("/api/v1/endpoints", handler.ListEndpoints)
	router.GET("/api/v1/endpoints/:id", handler.GetEndpoint)
	router.PUT("/api/v1/endpoints/:id", handler.UpdateEndpoint)
	router.DELETE("/api/v1/endpoints/:id", handler.DeleteEndpoint)

	// RoutingRule CRUD
	router.POST("/api/v1/routing-rules", handler.CreateRoutingRule)
	router.GET("/api/v1/routing-rules", handler.ListRoutingRules)
	router.GET("/api/v1/routing-rules/:id", handler.GetRoutingRule)
	router.PUT("/api/v1/routing-rules/:id", handler.UpdateRoutingRule)
	router.DELETE("/api/v1/routing-rules/:id", handler.DeleteRoutingRule)

	// OrchestrationRule CRUD
	router.POST("/api/v1/orchestration-rules", handler.CreateOrchestrationRule)
	router.GET("/api/v1/orchestration-rules/:id", handler.GetOrchestrationRule)
	router.PUT("/api/v1/orchestration-rules/:id", handler.UpdateOrchestrationRule)

	// OrchestrationRule 전환 관련
	router.GET("/api/v1/orchestration-rules/:id/evaluate-transition", handler.EvaluateTransition)
	router.POST("/api/v1/orchestration-rules/:id/execute-transition", handler.ExecuteTransition)

	// Metrics
	router.GET("/metrics", handler.Metrics)

	// pprof 프로파일링 엔드포인트 (디버그 전용)
	// /debug/pprof/ - 프로파일링 인덱스
	// /debug/pprof/cmdline - 커맨드라인
	// /debug/pprof/profile - CPU 프로파일 (30초)
	// /debug/pprof/symbol - 심볼 조회
	// /debug/pprof/trace - 실행 트레이스
	// /debug/pprof/heap - 힙 메모리 프로파일
	// /debug/pprof/goroutine - 고루틴 프로파일
	// /debug/pprof/threadcreate - 스레드 생성 프로파일
	// /debug/pprof/block - 블록 프로파일
	// /debug/pprof/mutex - 뮤텍스 프로파일
	router.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
}

// cleanup은 리소스를 정리합니다.
func cleanup(deps *Dependencies) {
	// Redis 클라이언트 정리
	if deps.RedisClient != nil {
		if err := deps.RedisClient.Close(); err != nil {
			fmt.Printf("Failed to close Redis client: %v\n", err)
		} else {
			fmt.Println("✅ Redis client closed")
		}
	}

	// 데이터베이스 리포지토리 정리 (Close 메서드가 있는 경우)
	if closer, ok := deps.RoutingRepo.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			fmt.Printf("Failed to close routing repository: %v\n", err)
		} else {
			fmt.Println("✅ Routing repository closed")
		}
	}

	if closer, ok := deps.EndpointRepo.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			fmt.Printf("Failed to close endpoint repository: %v\n", err)
		} else {
			fmt.Println("✅ Endpoint repository closed")
		}
	}

	fmt.Println("✅ Cleanup completed")
}
