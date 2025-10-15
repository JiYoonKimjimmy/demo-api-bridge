package http

import (
	"demo-api-bridge/internal/core/port"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// NewLoggingMiddleware는 로깅 미들웨어를 생성합니다.
func NewLoggingMiddleware(log port.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		log.Info("HTTP Request",
			"timestamp", param.TimeStamp.Format(time.RFC3339),
			"status", param.StatusCode,
			"latency_ms", param.Latency.Milliseconds(),
			"client_ip", param.ClientIP,
			"method", param.Method,
			"path", param.Path,
			"user_agent", param.Request.UserAgent(),
			"error", param.ErrorMessage,
		)
		return ""
	})
}

// NewMetricsMiddleware는 메트릭 수집 미들웨어를 생성합니다.
func NewMetricsMiddleware(m port.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		m.RecordRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), duration)
	}
}

// NewCORSMiddleware는 CORS 미들웨어를 생성합니다.
func NewCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// rateLimiter는 간단한 레이트 리미터입니다.
var rateLimiter = rate.NewLimiter(rate.Limit(100), 200) // 100 req/sec, burst 200

// NewRateLimitMiddleware는 레이트 리미팅 미들웨어를 생성합니다.
func NewRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rateLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
