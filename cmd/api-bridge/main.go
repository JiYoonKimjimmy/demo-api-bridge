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

	configadapter "demo-api-bridge/internal/adapter/config"
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

	// 캐시 리포지토리 초기화
	var cacheRepo port.CacheRepository
	var redisClient *redis.Client // 호환성을 위해 유지 (향후 제거 가능)

	switch cfg.Cache.Type {
	case "local", "ristretto":
		// Ristretto 로컬 캐시 초기화
		ristrettoConfig := &cache.RistrettoConfig{
			MaxSizeMB:      cfg.Cache.MaxSizeMB,
			NumCounters:    cfg.Cache.NumCounters,
			BufferItems:    cfg.Cache.BufferItems,
			MetricsEnabled: cfg.Cache.MetricsEnabled,
		}
		var err error
		cacheRepo, err = cache.NewRistrettoAdapter(ristrettoConfig)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to create Ristretto cache: %v", err))
			log.Info("Using Mock cache repository instead")
			cacheRepo = cache.NewMockCacheRepository()
		} else {
			log.Info("✅ Ristretto local cache initialized (1GB max memory)")
		}

	case "redis":
		// Redis 클라이언트 초기화 (레거시 지원)
		redisClient = redis.NewClient(&redis.Options{
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
			cacheRepo = cache.NewMockCacheRepository()
		} else {
			cacheRepo = cache.NewRedisAdapterWithClient(redisClient)
			log.Info("✅ Redis cache repository initialized")
		}

	case "mock":
		// Mock 캐시 (테스트 또는 캐시 비활성화)
		cacheRepo = cache.NewMockCacheRepository()
		log.Info("✅ Mock cache repository initialized")

	default:
		// 기본값: Ristretto 로컬 캐시
		log.Warn(fmt.Sprintf("Unknown cache type '%s', using Ristretto as default", cfg.Cache.Type))
		ristrettoConfig := &cache.RistrettoConfig{
			MaxSizeMB:      1024,
			NumCounters:    10000000,
			BufferItems:    64,
			MetricsEnabled: true,
		}
		var err error
		cacheRepo, err = cache.NewRistrettoAdapter(ristrettoConfig)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to create Ristretto cache: %v", err))
			cacheRepo = cache.NewMockCacheRepository()
		} else {
			log.Info("✅ Ristretto local cache initialized (default)")
		}
	}

	// Endpoint Repository 초기화 (Config 기반 - 메모리에서 로드, DB 조회 불필요)
	endpointRepo, err := configadapter.NewConfigEndpointRepository(&cfg.Endpoints)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to create config-based endpoint repository: %v", err))
		log.Info("Falling back to Mock endpoint repository")
		endpointRepo = database.NewMockEndpointRepository()
	} else {
		log.Info("✅ Config-based endpoint repository initialized (memory-based, no DB queries)")
	}

	// 데이터베이스 리포지토리 초기화 (OracleDB 또는 Mock)
	var routingRepo port.RoutingRepository
	var orchestrationRepo port.OrchestrationRepository
	var comparisonRepo port.ComparisonRepository

	// OracleDB 연결 시도
	oracleRoutingRepo, err := database.NewOracleRoutingRepository(&cfg.Database)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to connect to OracleDB: %v", err))
		log.Info("Using Mock database repositories instead")

		// Mock 리포지토리 사용
		routingRepo = database.NewMockRoutingRepository()
		orchestrationRepo = database.NewMockOrchestrationRepository()
		comparisonRepo = database.NewMockComparisonRepository()
	} else {
		// OracleDB 리포지토리 사용
		routingRepo = oracleRoutingRepo

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
	// === Internal Management API (높은 우선순위 - 먼저 등록) ===
	abs := router.Group("/abs")
	{
		// Swagger YAML 파일 제공
		abs.Static("/swagger-yaml", "./api-docs")

		// Swagger UI - swagger.yaml 파일 기반
		abs.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
			ginSwagger.URL("http://localhost:10019/abs/swagger-yaml/swagger.yaml")))

		// pprof 프로파일링 엔드포인트 (디버그 전용)
		abs.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))

		// Health Check & Monitoring
		abs.GET("/health", handler.HealthCheck)
		abs.GET("/ready", handler.ReadinessCheck)
		abs.GET("/metrics", handler.Metrics)
		abs.GET("/status", handler.Status)

		// Graceful Shutdown
		abs.POST("/shutdown", handler.GracefulShutdown)

		// APIEndpoint CRUD
		abs.POST("/v1/endpoints", handler.CreateEndpoint)
		abs.GET("/v1/endpoints", handler.ListEndpoints)
		abs.GET("/v1/endpoints/:id", handler.GetEndpoint)
		abs.PUT("/v1/endpoints/:id", handler.UpdateEndpoint)
		abs.DELETE("/v1/endpoints/:id", handler.DeleteEndpoint)

		// RoutingRule CRUD
		abs.POST("/v1/routing-rules", handler.CreateRoutingRule)
		abs.GET("/v1/routing-rules", handler.ListRoutingRules)
		abs.GET("/v1/routing-rules/:id", handler.GetRoutingRule)
		abs.PUT("/v1/routing-rules/:id", handler.UpdateRoutingRule)
		abs.DELETE("/v1/routing-rules/:id", handler.DeleteRoutingRule)

		// OrchestrationRule CRUD
		abs.POST("/v1/orchestration-rules", handler.CreateOrchestrationRule)
		abs.GET("/v1/orchestration-rules/:id", handler.GetOrchestrationRule)
		abs.PUT("/v1/orchestration-rules/:id", handler.UpdateOrchestrationRule)

		// OrchestrationRule 전환 관련
		abs.GET("/v1/orchestration-rules/:id/evaluate-transition", handler.EvaluateTransition)
		abs.POST("/v1/orchestration-rules/:id/execute-transition", handler.ExecuteTransition)
	}

	// === API Bridge - 모든 외부 요청 처리 (반드시 마지막에 등록!) ===
	// /abs/abs/* 를 제외한 모든 경로를 브릿지로 라우팅
	router.NoRoute(handler.ProcessBridgeRequest)
}

// cleanup은 리소스를 정리합니다.
func cleanup(deps *Dependencies) {
	// 캐시 리포지토리 정리 (Ristretto의 경우 Close 호출 필요)
	// ristrettoAdapter가 아닌 인터페이스를 통한 Close 메서드 확인
	type cacheCloser interface {
		Close()
	}
	if closer, ok := deps.Cache.(cacheCloser); ok {
		closer.Close()
		fmt.Println("✅ Cache repository closed")
	}

	// Redis 클라이언트 정리 (레거시 지원)
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
