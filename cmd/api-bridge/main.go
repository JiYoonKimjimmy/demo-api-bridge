package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultPort = "10019"
	serviceName = "api-bridge"
	version     = "0.1.0"
)

func main() {
	fmt.Printf("Starting %s v%s...\n", serviceName, version)

	// Gin 모드 설정
	gin.SetMode(gin.ReleaseMode)

	// 라우터 초기화
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 라우트 등록
	setupRoutes(router)

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
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
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
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exited")
}

func setupRoutes(router *gin.Engine) {
	// Health Check Endpoint
	router.GET("/health", healthHandler)
	router.GET("/ready", readyHandler)

	// API v1 그룹
	v1 := router.Group("/api/v1")
	{
		v1.GET("/status", statusHandler)
	}
}

// healthHandler는 서버 상태를 확인하는 헬스체크 엔드포인트입니다.
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": serviceName,
		"version": version,
	})
}

// readyHandler는 서버가 요청을 받을 준비가 되었는지 확인합니다.
func readyHandler(c *gin.Context) {
	// TODO: DB, Redis 등 의존성 연결 확인 로직 추가
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// statusHandler는 상세한 서버 상태 정보를 반환합니다.
func statusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":   serviceName,
		"version":   version,
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    "N/A", // TODO: 실제 uptime 계산 로직 추가
	})
}
