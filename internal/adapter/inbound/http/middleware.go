package http

import (
	"demo-api-bridge/internal/core/port"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const requestIDKey = "request_id"

// NewLoggingMiddleware는 로깅 미들웨어를 생성합니다.
func NewLoggingMiddleware(log port.Logger) gin.HandlerFunc {
	// 로깅에서 제외할 경로 패턴 정의
	skipPaths := []string{
		"/abs/swagger",      // Swagger UI
		"/abs/swagger-yaml", // Swagger YAML
		"/abs/debug/pprof",  // pprof 프로파일링
		"/favicon.ico",
	}

	return func(c *gin.Context) {
		// 제외 경로 확인
		for _, skipPath := range skipPaths {
			if strings.HasPrefix(c.Request.URL.Path, skipPath) {
				c.Next()
				return
			}
		}

		// Request ID 생성 및 Context에 저장
		requestID := generateRequestID()
		c.Set(requestIDKey, requestID)

		// 요청 시작 로그
		log.Info("▶ REQ " + requestID + " | " + c.Request.Method + " " + c.Request.URL.Path)

		// DEBUG 레벨: user_agent 정보
		if userAgent := c.Request.UserAgent(); userAgent != "" {
			log.Debug("Request " + requestID + " from " + userAgent)
		}

		// 요청 시작 시간
		start := time.Now()

		// 핸들러 실행
		c.Next()

		// 응답 완료 후 로그
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// 에러 메시지 확인
		var errorMsg string
		if len(c.Errors) > 0 {
			errorMsg = c.Errors.String()
		}

		// 상태 코드에 따라 로그 레벨 결정
		if statusCode >= 400 {
			// 에러 응답
			if errorMsg != "" {
				log.Error("◀ RES " + requestID + " | " + formatStatus(statusCode) + " | " + formatLatency(latency) + " | " + errorMsg)
			} else {
				log.Error("◀ RES " + requestID + " | " + formatStatus(statusCode) + " | " + formatLatency(latency))
			}
		} else {
			// 정상 응답
			log.Info("◀ RES " + requestID + " | " + formatStatus(statusCode) + " | " + formatLatency(latency))
		}
	}
}

// formatStatus는 HTTP 상태 코드를 문자열로 포맷합니다.
func formatStatus(code int) string {
	return fmt.Sprintf("%d", code)
}

// formatLatency는 레이턴시를 보기 좋게 포맷합니다.
func formatLatency(d time.Duration) string {
	ms := d.Milliseconds()
	return fmt.Sprintf("%dms", ms)
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
	// Rate limit에서 제외할 경로 정의
	// 관리 API, 모니터링, Swagger는 Rate Limit에서 제외
	skipPaths := []string{
		"/abs/", // 관리 API (Health, CRUD, Metrics, Debug, Swagger 등)
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
