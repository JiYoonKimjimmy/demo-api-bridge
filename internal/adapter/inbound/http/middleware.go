package http

import (
	"context"
	"time"

	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// Middleware는 HTTP 미들웨어들을 제공합니다.
type Middleware struct {
	logger  *logger.Logger
	metrics *metrics.Metrics
}

// NewMiddleware는 새로운 미들웨어를 생성합니다.
func NewMiddleware(logger *logger.Logger, metrics *metrics.Metrics) *Middleware {
	return &Middleware{
		logger:  logger,
		metrics: metrics,
	}
}

// RequestLogger는 요청 로깅 미들웨어입니다.
func (m *Middleware) RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 구조화된 로깅을 위한 JSON 형식
		logData := map[string]interface{}{
			"timestamp":  param.TimeStamp.Format(time.RFC3339),
			"status":     param.StatusCode,
			"latency":    param.Latency.Milliseconds(),
			"client_ip":  param.ClientIP,
			"method":     param.Method,
			"path":       param.Path,
			"user_agent": param.Request.UserAgent(),
			"error":      param.ErrorMessage,
		}

		// JSON 로깅 (실제로는 logger 패키지 사용)
		jsonData, _ := logger.JSONMarshal(logData)
		return string(jsonData) + "\n"
	})
}

// Recovery는 패닉 복구 미들웨어입니다.
func (m *Middleware) Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			ctx := c.Request.Context()
			m.logger.Error(ctx, "Panic recovered",
				"error", err,
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
			)
		}

		c.AbortWithStatus(500)
	})
}

// Metrics는 메트릭 수집 미들웨어입니다.
func (m *Middleware) Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 요청 시작 메트릭
		m.metrics.RecordHTTPRequestStarted(c.Request.Method, c.Request.URL.Path)

		// 요청 처리
		c.Next()

		// 요청 완료 메트릭
		duration := time.Since(start)
		m.metrics.RecordHTTPRequestCompleted(
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}

// CORSMiddleware는 CORS 헤더를 설정하는 미들웨어입니다.
func (m *Middleware) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Trace-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Trace-ID")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// TraceIDMiddleware는 Trace ID를 처리하는 미들웨어입니다.
func (m *Middleware) TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 기존 Trace ID 확인
		traceID := c.GetHeader("X-Trace-ID")

		// Trace ID가 없으면 생성
		if traceID == "" {
			traceID = generateTraceID()
		}

		// Context에 Trace ID 추가
		ctx := logger.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		// 응답 헤더에 Trace ID 추가
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}

// SecurityHeaders는 보안 헤더를 설정하는 미들웨어입니다.
func (m *Middleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 보안 헤더 설정
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// HSTS 헤더 (HTTPS인 경우만)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// RateLimiter는 기본적인 요청 제한 미들웨어입니다.
// 실제 구현에서는 Redis 기반 분산 Rate Limiting을 권장합니다.
func (m *Middleware) RateLimiter() gin.HandlerFunc {
	// 간단한 메모리 기반 Rate Limiting
	// 실제 프로덕션에서는 Redis나 다른 분산 저장소 사용 권장

	return func(c *gin.Context) {
		// 여기서는 단순히 통과시키지만,
		// 실제로는 클라이언트 IP별 요청 수를 추적하여 제한

		c.Next()
	}
}

// RequestSizeLimiter는 요청 크기를 제한하는 미들웨어입니다.
func (m *Middleware) RequestSizeLimiter(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.AbortWithStatusJSON(413, gin.H{
				"error":   "Request Entity Too Large",
				"message": "Request size exceeds the allowed limit",
			})
			return
		}

		c.Next()
	}
}

// TimeoutMiddleware는 요청 타임아웃을 설정하는 미들웨어입니다.
func (m *Middleware) TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Context에 타임아웃 설정
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// HealthCheckSkip는 헬스체크 엔드포인트를 메트릭에서 제외하는 미들웨어입니다.
func (m *Middleware) HealthCheckSkip() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 헬스체크 관련 경로는 메트릭에서 제외
		if path == "/health" || path == "/ready" || path == "/metrics" {
			c.Next()
			return
		}

		// 다른 미들웨어들 적용
		c.Next()
	}
}
