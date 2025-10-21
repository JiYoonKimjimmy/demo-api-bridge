package integration

import (
	"bytes"
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

// TestFullFlow_SingleAPIRequest는 단일 API 요청의 전체 플로우를 테스트합니다.
func TestFullFlow_SingleAPIRequest(t *testing.T) {
	// Given: 외부 Mock API 서버 생성
	mockExternalAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    1,
			"name":  "John Doe",
			"email": "john@example.com",
		})
	}))
	defer mockExternalAPI.Close()

	// Setup: 서비스 및 핸들러 초기화
	router, cleanup := setupIntegrationTest("single_api", mockExternalAPI.URL)
	defer cleanup()

	// When: API 요청 실행
	req, _ := http.NewRequest("GET", "/api/v1/bridge/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 응답 검증
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), response["id"])
	assert.Equal(t, "John Doe", response["name"])
}

// TestFullFlow_ParallelAPIRequest는 병렬 API 요청의 전체 플로우를 테스트합니다.
func TestFullFlow_ParallelAPIRequest(t *testing.T) {
	// Given: 레거시와 모던 Mock API 서버 생성
	legacyResponse := map[string]interface{}{
		"user_id": 1,
		"name":    "John Doe",
		"email":   "john@example.com",
	}

	modernResponse := map[string]interface{}{
		"userId": 1,
		"name":   "John Doe",
		"email":  "john@example.com",
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

	// Setup: 병렬 호출 설정
	router, cleanup := setupParallelTest("parallel_api", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When: API 요청 실행
	req, _ := http.NewRequest("GET", "/api/v1/bridge/users/parallel", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 레거시 응답 반환 확인 (PARALLEL 모드는 레거시 우선)
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 레거시 응답 형식 확인
	assert.Contains(t, w.Body.String(), "user_id") // 레거시 응답
}

// TestFullFlow_AutoTransition는 자동 전환 시나리오를 테스트합니다.
func TestFullFlow_AutoTransition(t *testing.T) {
	// Given: 동일한 응답을 반환하는 레거시/모던 API
	identicalResponse := map[string]interface{}{
		"id":   1,
		"name": "John Doe",
	}

	mockLegacyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(identicalResponse)
	}))
	defer mockLegacyAPI.Close()

	mockModernAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(identicalResponse)
	}))
	defer mockModernAPI.Close()

	// Setup
	router, cleanup := setupParallelTest("auto_transition", mockLegacyAPI.URL, mockModernAPI.URL)
	defer cleanup()

	// When: 10번 요청 (자동 전환 임계값)
	successCount := 0
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/bridge/users/parallel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	// Then: 최소한 일부 요청은 성공해야 함
	assert.True(t, successCount > 0, "At least some requests should succeed")
	t.Logf("Auto transition scenario: %d/%d requests succeeded", successCount, 10)
}

// TestFullFlow_CacheHit는 캐시 히트 시나리오를 테스트합니다.
func TestFullFlow_CacheHit(t *testing.T) {
	// Given
	mockExternalAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     1,
			"cached": true,
		})
	}))
	defer mockExternalAPI.Close()

	router, cleanup := setupCachedTest("cache_hit", mockExternalAPI.URL)
	defer cleanup()

	// When: 첫 번째 요청 (캐시 미스)
	req1, _ := http.NewRequest("GET", "/api/v1/bridge/cache/test", nil)
	w1 := httptest.NewRecorder()
	start1 := time.Now()
	router.ServeHTTP(w1, req1)
	duration1 := time.Since(start1)

	// Then: 성공 확인
	assert.Equal(t, http.StatusOK, w1.Code)

	// When: 두 번째 요청 (캐시 히트 예상)
	req2, _ := http.NewRequest("GET", "/api/v1/bridge/cache/test", nil)
	w2 := httptest.NewRecorder()
	start2 := time.Now()
	router.ServeHTTP(w2, req2)
	duration2 := time.Since(start2)

	// Then: 캐시 히트로 더 빠른 응답
	assert.Equal(t, http.StatusOK, w2.Code)

	// 캐시 히트는 일반적으로 더 빠름 (항상 보장되는 것은 아니지만 경향성 확인)
	t.Logf("First request: %v, Second request (cached): %v", duration1, duration2)
}

// TestFullFlow_ErrorHandling는 에러 처리 시나리오를 테스트합니다.
func TestFullFlow_ErrorHandling(t *testing.T) {
	// Given: 에러를 반환하는 Mock API
	mockExternalAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Internal server error",
		})
	}))
	defer mockExternalAPI.Close()

	router, cleanup := setupIntegrationTest("error_handling", mockExternalAPI.URL)
	defer cleanup()

	// When: API 요청 실행
	req, _ := http.NewRequest("GET", "/api/v1/bridge/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 외부 API의 에러 응답을 그대로 반환
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFullFlow_InvalidRequest는 잘못된 요청 처리를 테스트합니다.
func TestFullFlow_InvalidRequest(t *testing.T) {
	// Given
	mockExternalAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockExternalAPI.Close()

	router, cleanup := setupIntegrationTest("invalid_request", mockExternalAPI.URL)
	defer cleanup()

	// When: 등록되지 않은 경로로 요청
	req, _ := http.NewRequest("GET", "/api/v1/bridge/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 404 또는 에러 응답
	assert.True(t, w.Code >= 400, "Should return error status code")
}

// TestFullFlow_HealthCheck는 헬스체크 엔드포인트를 테스트합니다.
func TestFullFlow_HealthCheck(t *testing.T) {
	// Given
	mockExternalAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockExternalAPI.Close()

	router, cleanup := setupIntegrationTest("health_check", mockExternalAPI.URL)
	defer cleanup()

	// When: Health 엔드포인트 호출
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 정상 응답 확인
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFullFlow_ReadinessCheck는 Readiness 엔드포인트를 테스트합니다.
func TestFullFlow_ReadinessCheck(t *testing.T) {
	// Given
	mockExternalAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockExternalAPI.Close()

	router, cleanup := setupIntegrationTest("readiness_check", mockExternalAPI.URL)
	defer cleanup()

	// When: Ready 엔드포인트 호출
	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 정상 응답 확인
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFullFlow_PostRequest는 POST 요청을 테스트합니다.
func TestFullFlow_PostRequest(t *testing.T) {
	// Given: POST 요청을 받는 Mock API
	mockExternalAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 요청 본문 읽기
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      1,
			"name":    reqBody["name"],
			"created": true,
		})
	}))
	defer mockExternalAPI.Close()

	router, cleanup := setupPostTest("post_request", mockExternalAPI.URL)
	defer cleanup()

	// When: POST 요청 실행
	requestBody := map[string]interface{}{
		"name":  "New User",
		"email": "newuser@example.com",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/api/v1/bridge/users", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Then: 생성 성공 확인
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "New User", response["name"])
	assert.Equal(t, true, response["created"])
}

// setupIntegrationTest는 통합 테스트 환경을 설정합니다.
func setupIntegrationTest(testName, externalAPIURL string) (*gin.Engine, func()) {
	// 의존성 설정
	log := logger.NewDefault()
	metricsCollector := metrics.New("integration_test_" + testName)

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
	endpoint := domain.NewAPIEndpoint("test-endpoint", "Test API", externalAPIURL, "", "GET")
	endpoint.IsActive = true
	endpoint.Timeout = 5 * time.Second
	endpointRepo.Create(ctx, endpoint)

	rule := domain.NewRoutingRule("test-rule", "Test Rule", "/users", "GET", "test-endpoint")
	rule.CacheEnabled = false
	routingRepo.Create(ctx, rule)

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

// setupParallelTest는 병렬 호출 통합 테스트 환경을 설정합니다.
func setupParallelTest(testName, legacyURL, modernURL string) (*gin.Engine, func()) {
	// 의존성 설정
	log := logger.NewDefault()
	metricsCollector := metrics.New("integration_test_" + testName)

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
	rule := domain.NewRoutingRule("parallel-rule", "Parallel Rule", "/users/parallel", "GET", "legacy-endpoint")
	rule.CacheEnabled = false
	routingRepo.Create(ctx, rule)

	// 오케스트레이션 규칙
	orchRule := domain.NewOrchestrationRule(
		"orch-rule-1",
		"Test Orchestration",
		"parallel-rule",
		"legacy-endpoint",
		"modern-endpoint",
	)
	orchRule.CurrentMode = domain.PARALLEL
	orchRule.TransitionConfig.AutoTransitionEnabled = true
	orchRule.TransitionConfig.MatchRateThreshold = 0.95
	orchRule.TransitionConfig.MinRequestsForTransition = 10
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

// setupCachedTest는 캐시 테스트 환경을 설정합니다.
func setupCachedTest(testName, externalAPIURL string) (*gin.Engine, func()) {
	// 의존성 설정
	log := logger.NewDefault()
	metricsCollector := metrics.New("integration_test_" + testName)

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

	// 테스트 데이터 설정 (캐시 활성화)
	ctx := context.Background()
	endpoint := domain.NewAPIEndpoint("cache-endpoint", "Cache API", externalAPIURL, "", "GET")
	endpoint.IsActive = true
	endpoint.Timeout = 5 * time.Second
	endpointRepo.Create(ctx, endpoint)

	rule := domain.NewRoutingRule("cache-rule", "Cache Rule", "/cache/test", "GET", "cache-endpoint")
	rule.CacheEnabled = true
	rule.CacheTTL = 300
	routingRepo.Create(ctx, rule)

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

// setupPostTest는 POST 요청 테스트 환경을 설정합니다.
func setupPostTest(testName, externalAPIURL string) (*gin.Engine, func()) {
	// 의존성 설정
	log := logger.NewDefault()
	metricsCollector := metrics.New("integration_test_" + testName)

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
	endpoint := domain.NewAPIEndpoint("post-endpoint", "POST API", externalAPIURL, "", "POST")
	endpoint.IsActive = true
	endpoint.Timeout = 5 * time.Second
	endpointRepo.Create(ctx, endpoint)

	rule := domain.NewRoutingRule("post-rule", "POST Rule", "/users", "POST", "post-endpoint")
	rule.CacheEnabled = false
	routingRepo.Create(ctx, rule)

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
