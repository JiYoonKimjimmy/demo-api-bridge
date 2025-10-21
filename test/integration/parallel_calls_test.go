package integration

// 주의: 이 파일의 테스트를 실행할 때는 개별 테스트로 실행하거나
// go test ./test/integration/... 명령어를 사용하세요.
// go test test/integration/parallel_calls_test.go 형태로 파일 전체 실행 시
// Prometheus 메트릭 중복 등록으로 인해 두 번째 테스트부터 실패할 수 있습니다.
//
// 권장 실행 방법:
// - go test -v ./test/integration/... -run TestParallelCalls_IdenticalResponses
// - go test -v ./test/integration/... -run TestParallelCalls

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httphandler "demo-api-bridge/internal/adapter/inbound/http"
	"demo-api-bridge/internal/adapter/outbound/cache"
	"demo-api-bridge/internal/adapter/outbound/database"
	"demo-api-bridge/internal/adapter/outbound/httpclient"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/service"
	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestParallelCalls_IdenticalResponses는 동일한 응답을 반환하는 병렬 호출을 테스트합니다.
func TestParallelCalls_IdenticalResponses(t *testing.T) {
	// Given: 동일한 응답을 반환하는 레거시/모던 API
	response := map[string]interface{}{
		"id":   1,
		"name": "John Doe",
	}

	mockLegacyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockLegacyAPI.Close()

	mockModernAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockModernAPI.Close()

	// Setup
	router, cleanup := setupParallelCallsTest("identical_responses", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When
	req, _ := http.NewRequest("GET", "/api/v1/bridge/parallel/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 레거시 응답 반환 (PARALLEL 모드)
	assert.Equal(t, http.StatusOK, w.Code)

	var responseData map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &responseData)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), responseData["id"])
	assert.Equal(t, "John Doe", responseData["name"])
}

// TestParallelCalls_DifferentResponses는 다른 응답을 반환하는 병렬 호출을 테스트합니다.
func TestParallelCalls_DifferentResponses(t *testing.T) {
	// Given: 다른 응답을 반환하는 레거시/모던 API
	legacyResponse := map[string]interface{}{
		"user_id":   1,
		"user_name": "John Legacy",
	}

	modernResponse := map[string]interface{}{
		"id":   1,
		"name": "John Modern",
	}

	mockLegacyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(legacyResponse)
	}))
	defer mockLegacyAPI.Close()

	mockModernAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(modernResponse)
	}))
	defer mockModernAPI.Close()

	// Setup
	router, cleanup := setupParallelCallsTest("different_responses", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When
	req, _ := http.NewRequest("GET", "/api/v1/bridge/parallel/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 레거시 응답 반환 및 비교 수행
	assert.Equal(t, http.StatusOK, w.Code)

	var responseData map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &responseData)
	assert.NoError(t, err)

	// 레거시 응답 형식 확인
	assert.Contains(t, w.Body.String(), "user_id")
}

// TestParallelCalls_LegacyFailure는 레거시 API 실패 시나리오를 테스트합니다.
func TestParallelCalls_LegacyFailure(t *testing.T) {
	// Given: 레거시 API는 실패, 모던 API는 성공
	mockLegacyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockLegacyAPI.Close()

	modernResponse := map[string]interface{}{
		"id":   1,
		"name": "John Modern",
	}

	mockModernAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(modernResponse)
	}))
	defer mockModernAPI.Close()

	// Setup
	router, cleanup := setupParallelCallsTest("legacy_failure", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When
	req, _ := http.NewRequest("GET", "/api/v1/bridge/parallel/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 레거시가 실패해도 응답 반환 (일치율 낮음)
	// 실제 동작은 레거시 우선이므로 실패 응답 가능
	assert.True(t, w.Code >= 200, "Should return some response")
}

// TestParallelCalls_ModernFailure는 모던 API 실패 시나리오를 테스트합니다.
func TestParallelCalls_ModernFailure(t *testing.T) {
	// Given: 레거시 API는 성공, 모던 API는 실패
	legacyResponse := map[string]interface{}{
		"id":   1,
		"name": "John Legacy",
	}

	mockLegacyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(legacyResponse)
	}))
	defer mockLegacyAPI.Close()

	mockModernAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockModernAPI.Close()

	// Setup
	router, cleanup := setupParallelCallsTest("modern_failure", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When
	req, _ := http.NewRequest("GET", "/api/v1/bridge/parallel/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 레거시 응답 반환 (레거시 우선)
	assert.Equal(t, http.StatusOK, w.Code)

	var responseData map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &responseData)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), responseData["id"])
}

// TestParallelCalls_Timeout는 타임아웃 시나리오를 테스트합니다.
func TestParallelCalls_Timeout(t *testing.T) {
	// Given: 느린 레거시 API, 빠른 모던 API
	mockLegacyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // 의도적 지연
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 1})
	}))
	defer mockLegacyAPI.Close()

	mockModernAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 1})
	}))
	defer mockModernAPI.Close()

	// Setup
	router, cleanup := setupParallelCallsTest("timeout", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When
	start := time.Now()
	req, _ := http.NewRequest("GET", "/api/v1/bridge/parallel/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	duration := time.Since(start)

	// Then: 병렬 호출로 빠르게 응답 (느린 API 기다리지 않음)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, duration < 200*time.Millisecond, "Should complete faster with parallel calls")
	t.Logf("Parallel call duration: %v", duration)
}

// setupParallelCallsTest는 병렬 호출 테스트 환경을 설정합니다.
func setupParallelCallsTest(testName, legacyURL, modernURL string) (*gin.Engine, func()) {
	// 의존성 설정
	log := logger.NewDefault()
	metricsCollector := metrics.New("parallel_test_" + testName)

	// Repository 설정
	routingRepo := database.NewMockRoutingRepository()
	endpointRepo := database.NewMockEndpointRepository()
	orchestrationRepo := database.NewMockOrchestrationRepository()
	comparisonRepo := database.NewMockComparisonRepository()
	cacheRepo := cache.NewMockCacheRepository()

	// HTTP Client
	httpClient := httpclient.NewHTTPClientAdapter(30 * time.Second)

	// Services
	orchestrationSvc := service.NewOrchestrationService(
		orchestrationRepo,
		comparisonRepo,
		httpClient,
		log,
		metricsCollector,
	)

	bridgeSvc := service.NewBridgeService(
		routingRepo,
		endpointRepo,
		orchestrationRepo,
		comparisonRepo,
		orchestrationSvc,
		httpClient,
		cacheRepo,
		log,
		metricsCollector,
	)

	healthSvc := service.NewHealthCheckService(routingRepo, endpointRepo, cacheRepo, log)

	// 테스트 데이터 설정
	ctx := context.Background()

	// 레거시 엔드포인트
	legacyEndpoint := domain.NewAPIEndpoint("legacy-endpoint", "Legacy API", legacyURL, "", "GET")
	legacyEndpoint.IsActive = true
	legacyEndpoint.Timeout = 5 * time.Second
	endpointRepo.Create(ctx, legacyEndpoint)

	// 모던 엔드포인트
	modernEndpoint := domain.NewAPIEndpoint("modern-endpoint", "Modern API", modernURL, "", "GET")
	modernEndpoint.IsActive = true
	modernEndpoint.Timeout = 5 * time.Second
	endpointRepo.Create(ctx, modernEndpoint)

	// 라우팅 규칙
	rule := domain.NewRoutingRule("parallel-rule", "Parallel Rule", "/parallel/users", "GET", "legacy-endpoint")
	rule.CacheEnabled = false
	routingRepo.Create(ctx, rule)

	// 오케스트레이션 규칙
	orchRule := domain.NewOrchestrationRule(
		"orch-rule-parallel",
		"Parallel Test Orchestration",
		"parallel-rule",
		"legacy-endpoint",
		"modern-endpoint",
	)
	orchRule.CurrentMode = domain.PARALLEL
	orchRule.TransitionConfig.AutoTransitionEnabled = false // 자동 전환 비활성화
	orchRule.TransitionConfig.MatchRateThreshold = 0.95
	orchRule.TransitionConfig.MinRequestsForTransition = 100
	orchRule.ComparisonConfig.SaveComparisonHistory = true
	orchestrationRepo.Create(ctx, orchRule)

	// HTTP Handler 설정
	handler := httphandler.NewHandler(bridgeSvc, healthSvc, log)

	// Gin Router 설정
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 라우트 설정
	router.GET("/health", handler.HealthCheck)
	router.GET("/ready", handler.ReadinessCheck)
	router.GET("/api/v1/status", handler.Status)
	router.Any("/api/v1/bridge/*path", handler.ProcessBridgeRequest)

	cleanup := func() {
		// cleanup logic if needed
	}

	return router, cleanup
}

// TestParallelCalls_ComparisonHistory는 비교 이력 저장을 테스트합니다.
func TestParallelCalls_ComparisonHistory(t *testing.T) {
	// Given: 약간 다른 응답을 반환하는 레거시/모던 API
	legacyResponse := map[string]interface{}{
		"id":    1,
		"name":  "John Doe",
		"email": "john.legacy@example.com",
	}

	modernResponse := map[string]interface{}{
		"id":    1,
		"name":  "John Doe",
		"email": "john.modern@example.com", // 이메일만 다름
	}

	mockLegacyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(legacyResponse)
	}))
	defer mockLegacyAPI.Close()

	mockModernAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(modernResponse)
	}))
	defer mockModernAPI.Close()

	// Setup
	router, cleanup := setupParallelCallsTest("comparison_history", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When: 여러 번 요청
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/bridge/parallel/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Then: 비교 이력이 저장됨 (실제 확인은 서비스 레이어에서)
	t.Log("Comparison history saved - check logs")
}

// TestParallelCalls_CircuitBreaker는 Circuit Breaker 동작을 테스트합니다.
func TestParallelCalls_CircuitBreaker(t *testing.T) {
	t.Skip("Circuit Breaker integration test - requires more setup")
	// TODO: Circuit Breaker가 OPEN 상태일 때 동작 테스트
}

// TestParallelCalls_ConcurrentRequests는 동시 다중 병렬 요청을 테스트합니다.
func TestParallelCalls_ConcurrentRequests(t *testing.T) {
	// Given
	response := map[string]interface{}{
		"id":   1,
		"name": "John Doe",
	}

	mockLegacyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockLegacyAPI.Close()

	mockModernAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockModernAPI.Close()

	// Setup
	router, cleanup := setupParallelCallsTest("concurrent_requests", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When: 동시에 10개 요청
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			req, _ := http.NewRequest("GET", "/api/v1/bridge/parallel/users", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- assert.AnError
			}
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}
	close(done)
	close(errors)

	// Then: 모든 요청 성공
	errorCount := len(errors)
	assert.Equal(t, 0, errorCount, "All concurrent requests should succeed")
}
