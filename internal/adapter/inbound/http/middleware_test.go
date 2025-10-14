package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestMiddleware() (*Middleware, *gin.Engine) {
	// Create logger and metrics
	testLogger := logger.NewLogger("test", "info")
	testMetrics := metrics.NewMetrics()

	// Create middleware
	middleware := NewMiddleware(testLogger, testMetrics)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Apply middlewares
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.Metrics())
	router.Use(middleware.TraceIDMiddleware())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORSMiddleware())

	// Add test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return middleware, router
}

func TestTraceIDMiddleware(t *testing.T) {
	_, router := setupTestMiddleware()

	// Test without existing Trace ID
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Trace-ID"))
}

func TestTraceIDMiddlewareWithExistingID(t *testing.T) {
	_, router := setupTestMiddleware()

	// Test with existing Trace ID
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "test-trace-id-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-trace-id-123", w.Header().Get("X-Trace-ID"))
}

func TestSecurityHeaders(t *testing.T) {
	_, router := setupTestMiddleware()

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestCORSMiddleware(t *testing.T) {
	_, router := setupTestMiddleware()

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddlewareOPTIONS(t *testing.T) {
	_, router := setupTestMiddleware()

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRequestSizeLimiter(t *testing.T) {
	middleware, router := setupTestMiddleware()

	// Apply size limiter
	router.Use(middleware.RequestSizeLimiter(1024)) // 1KB limit

	// Test with small request
	req, _ := http.NewRequest("POST", "/test", nil)
	req.ContentLength = 512
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should pass
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestSizeLimiterExceeded(t *testing.T) {
	middleware, router := setupTestMiddleware()

	// Apply size limiter
	router.Use(middleware.RequestSizeLimiter(1024)) // 1KB limit

	// Test with large request
	req, _ := http.NewRequest("POST", "/test", nil)
	req.ContentLength = 2048 // 2KB
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fail
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestTimeoutMiddleware(t *testing.T) {
	middleware, router := setupTestMiddleware()

	// Apply timeout middleware
	router.Use(middleware.TimeoutMiddleware(100 * time.Millisecond))

	// Add a slow route
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond) // Longer than timeout
		c.JSON(http.StatusOK, gin.H{"message": "slow"})
	})

	req, _ := http.NewRequest("GET", "/slow", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should timeout (context deadline exceeded)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRecoveryMiddleware(t *testing.T) {
	middleware, router := setupTestMiddleware()

	// Add a route that panics
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req, _ := http.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should recover and return 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMetricsMiddleware(t *testing.T) {
	_, router := setupTestMiddleware()

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should record metrics (we can't easily test the metrics recording without exposing internal state)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthCheckSkipMiddleware(t *testing.T) {
	middleware, router := setupTestMiddleware()

	// Apply health check skip middleware
	router.Use(middleware.HealthCheckSkip())

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should work normally
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Header().Get("X-Status"))
}

// Test middleware chain
func TestMiddlewareChain(t *testing.T) {
	middleware, router := setupTestMiddleware()

	// Apply all middlewares in order
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.Metrics())
	router.Use(middleware.TraceIDMiddleware())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RequestSizeLimiter(1024))
	router.Use(middleware.TimeoutMiddleware(5 * time.Second))

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "test-trace-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should apply all middlewares
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-trace-123", w.Header().Get("X-Trace-ID"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// Benchmark tests
func BenchmarkMiddlewareChain(b *testing.B) {
	_, router := setupTestMiddleware()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkTraceIDMiddleware(b *testing.B) {
	middleware, router := gin.New(), gin.New()
	router.Use(middleware.TraceIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkSecurityHeaders(b *testing.B) {
	middleware, router := gin.New(), gin.New()
	router.Use(middleware.SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkCORSMiddleware(b *testing.B) {
	middleware, router := gin.New(), gin.New()
	router.Use(middleware.CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
