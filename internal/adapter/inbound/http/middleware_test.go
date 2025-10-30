package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// TestNewLoggingMiddleware는 로깅 미들웨어를 테스트합니다.
func TestNewLoggingMiddleware(t *testing.T) {
	testLogger := logger.NewLogger()
	router := setupTestRouter()

	// 로깅 미들웨어 적용
	router.Use(NewLoggingMiddleware(testLogger))

	// 테스트 라우트 추가
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 요청 생성 및 실행
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestLoggingMiddlewareSkipPaths는 로깅 제외 경로를 테스트합니다.
func TestLoggingMiddlewareSkipPaths(t *testing.T) {
	testLogger := logger.NewLogger()
	router := setupTestRouter()

	router.Use(NewLoggingMiddleware(testLogger))

	// Swagger 경로 추가
	router.GET("/swagger/index.html", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증 - 로깅이 스킵되어도 요청은 정상 처리
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestNewMetricsMiddleware는 메트릭 미들웨어를 테스트합니다.
func TestNewMetricsMiddleware(t *testing.T) {
	testMetrics := metrics.New("test")
	router := setupTestRouter()

	// 메트릭 미들웨어 적용
	router.Use(NewMetricsMiddleware(testMetrics))

	// 테스트 라우트 추가
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 요청 생성 및 실행
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestNewCORSMiddleware는 CORS 미들웨어를 테스트합니다.
func TestNewCORSMiddleware(t *testing.T) {
	router := setupTestRouter()

	// CORS 미들웨어 적용
	router.Use(NewCORSMiddleware())

	// 테스트 라우트 추가
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 요청 생성 및 실행
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증 - CORS 헤더 확인
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

// TestCORSMiddlewareOPTIONS는 OPTIONS 요청을 테스트합니다.
func TestCORSMiddlewareOPTIONS(t *testing.T) {
	router := setupTestRouter()

	router.Use(NewCORSMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// OPTIONS 요청 생성 및 실행
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증 - OPTIONS 요청은 204 No Content 반환
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestNewRateLimitMiddleware는 Rate Limit 미들웨어를 테스트합니다.
func TestNewRateLimitMiddleware(t *testing.T) {
	router := setupTestRouter()

	// Rate Limit 미들웨어 적용
	router.Use(NewRateLimitMiddleware())

	// 테스트 라우트 추가
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 요청 생성 및 실행
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증 - 정상 요청은 통과
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRateLimitMiddlewareSkipPaths는 Rate Limit 제외 경로를 테스트합니다.
func TestRateLimitMiddlewareSkipPaths(t *testing.T) {
	router := setupTestRouter()

	router.Use(NewRateLimitMiddleware())

	// 관리 API 경로 추가 (Rate Limit 제외)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증 - 제외 경로는 항상 통과
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestMiddlewareChain는 여러 미들웨어를 체인으로 연결하여 테스트합니다.
func TestMiddlewareChain(t *testing.T) {
	testLogger := logger.NewLogger()
	testMetrics := metrics.New("test_chain") // 고유한 네임스페이스 사용
	router := setupTestRouter()

	// 모든 미들웨어 적용
	router.Use(NewLoggingMiddleware(testLogger))
	router.Use(NewMetricsMiddleware(testMetrics))
	router.Use(NewCORSMiddleware())
	router.Use(NewRateLimitMiddleware())

	// 테스트 라우트 추가
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 요청 생성 및 실행
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestLoggingMiddlewareWithError는 에러 응답 시 로깅을 테스트합니다.
func TestLoggingMiddlewareWithError(t *testing.T) {
	testLogger := logger.NewLogger()
	router := setupTestRouter()

	router.Use(NewLoggingMiddleware(testLogger))

	// 에러를 반환하는 라우트
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})

	req, _ := http.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증 - 에러 상태 코드
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Benchmark tests
func BenchmarkLoggingMiddleware(b *testing.B) {
	testLogger := logger.NewLogger()
	router := setupTestRouter()
	router.Use(NewLoggingMiddleware(testLogger))
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
	router := setupTestRouter()
	router.Use(NewCORSMiddleware())
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

func BenchmarkRateLimitMiddleware(b *testing.B) {
	router := setupTestRouter()
	router.Use(NewRateLimitMiddleware())
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

func BenchmarkMetricsMiddleware(b *testing.B) {
	testMetrics := metrics.New("bench_metrics")
	router := setupTestRouter()
	router.Use(NewMetricsMiddleware(testMetrics))
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

func BenchmarkMiddlewareChain(b *testing.B) {
	testLogger := logger.NewLogger()
	testMetrics := metrics.New("bench_chain")
	router := setupTestRouter()

	router.Use(NewLoggingMiddleware(testLogger))
	router.Use(NewMetricsMiddleware(testMetrics))
	router.Use(NewCORSMiddleware())
	router.Use(NewRateLimitMiddleware())

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
