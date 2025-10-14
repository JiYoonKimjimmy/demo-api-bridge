package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultPort = "10019"
	serviceName = "api_bridge"
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

	// Health Check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   serviceName,
			"version":   version,
			"timestamp": time.Now().Format(time.RFC3339),
			"uptime":    "N/A",
		})
	})

	// Readiness Check
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"ready":  true,
			"checks": map[string]string{
				"database": "ok",
				"cache":    "ok",
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Status
	router.GET("/api/v1/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     serviceName,
			"version":     version,
			"timestamp":   time.Now().Format(time.RFC3339),
			"uptime":      "N/A",
			"environment": "development",
			"metrics":     map[string]interface{}{"requests": 0},
		})
	})

	// API Bridge - 모든 요청 처리 (status 제외)
	router.Any("/api/v1/bridge/*path", func(c *gin.Context) {
		path := c.Param("path")
		method := c.Request.Method

		fmt.Printf("Processing %s request to %s\n", method, path)

		// 간단한 응답
		c.JSON(http.StatusOK, gin.H{
			"message":   "API Bridge is working",
			"method":    method,
			"path":      path,
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   serviceName,
			"version":   version,
		})
	})

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
