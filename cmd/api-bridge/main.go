package main

import (
	"context"
	"fmt"
	"net/http"
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
	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"

	"github.com/gin-gonic/gin"
)

const (
	defaultPort = "10019"
	serviceName = "api_bridge"
	version     = "0.1.0"
)

func main() {
	fmt.Printf("Starting %s v%s...\n", serviceName, version)

	// 의존성 초기화
	dependencies := initializeDependencies()
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
		fmt.Printf("Server listening on port %s\n", port)
		fmt.Println("API Endpoints:")
		fmt.Println("  GET  /health                    - Health check")
		fmt.Println("  GET  /ready                     - Readiness check")
		fmt.Println("  GET  /api/v1/status             - Service status")
		fmt.Println("  ANY  /api/v1/bridge/*           - API Bridge")
		fmt.Println("Press Ctrl+C to stop")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	// Graceful Shutdown 설정
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
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
	Logger            port.Logger
	Metrics           port.MetricsCollector
	Cache             port.CacheRepository
	RoutingRepo       port.RoutingRepository
	EndpointRepo      port.EndpointRepository
	OrchestrationRepo port.OrchestrationRepository
	ComparisonRepo    port.ComparisonRepository
	CircuitBreaker    port.CircuitBreakerService
	ExternalAPI       port.ExternalAPIClient
	BridgeService     port.BridgeService
	HealthService     port.HealthCheckService
}

// initializeDependencies는 모든 의존성을 초기화합니다.
func initializeDependencies() *Dependencies {
	// 로거 초기화
	log := logger.NewLogger()

	// 메트릭 초기화
	metricsCollector := metrics.NewMetricsCollector()

	// 캐시 초기화 (Mock)
	cacheRepo := cache.NewMockCacheRepository()

	// 데이터베이스 리포지토리 초기화 (Mock)
	routingRepo := database.NewMockRoutingRepository()
	endpointRepo := database.NewMockEndpointRepository()
	orchestrationRepo := database.NewMockOrchestrationRepository()
	comparisonRepo := database.NewMockComparisonRepository()

	// Circuit Breaker 서비스 초기화
	circuitBreakerService := service.NewCircuitBreakerService(log, metricsCollector)

	// HTTP 클라이언트 초기화 (Circuit Breaker 포함)
	httpClient := httpclient.NewHTTPClientAdapterWithCircuitBreaker(30*time.Second, circuitBreakerService)

	// 서비스 초기화
	healthService := service.NewHealthCheckService(routingRepo, endpointRepo, cacheRepo, log)
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
		Logger:            log,
		Metrics:           metricsCollector,
		Cache:             cacheRepo,
		RoutingRepo:       routingRepo,
		EndpointRepo:      endpointRepo,
		OrchestrationRepo: orchestrationRepo,
		ComparisonRepo:    comparisonRepo,
		CircuitBreaker:    circuitBreakerService,
		ExternalAPI:       httpClient,
		BridgeService:     bridgeService,
		HealthService:     healthService,
	}
}

// setupRoutes는 라우트를 설정합니다.
func setupRoutes(router *gin.Engine, handler *httpadapter.Handler) {
	// Health Check
	router.GET("/health", handler.HealthCheck)
	router.GET("/ready", handler.ReadinessCheck)
	router.GET("/api/v1/status", handler.Status)

	// API Bridge - 모든 요청 처리
	router.Any("/api/v1/bridge/*path", handler.ProcessBridgeRequest)

	// Metrics
	router.GET("/metrics", handler.Metrics)
}

// cleanup은 리소스를 정리합니다.
func cleanup(deps *Dependencies) {
	// HTTP 클라이언트 정리는 필요시 구현
	// 현재 Mock 구현체는 정리가 필요하지 않음
}
