package test

import (
	"bytes"
	httpadapter "demo-api-bridge/internal/adapter/inbound/http"
	"demo-api-bridge/internal/adapter/outbound/cache"
	"demo-api-bridge/internal/adapter/outbound/database"
	"demo-api-bridge/internal/adapter/outbound/httpclient"
	"demo-api-bridge/internal/core/service"
	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCRUDAPI는 CRUD API의 전체 플로우를 테스트합니다.
func TestCRUDAPI(t *testing.T) {
	// 테스트 환경 설정
	gin.SetMode(gin.TestMode)

	// 의존성 초기화
	log := logger.NewLogger()
	metricsCollector := metrics.NewMetricsCollector()

	// Mock 리포지토리들
	endpointRepo := database.NewMockEndpointRepository()
	routingRepo := database.NewMockRoutingRepository()
	orchestrationRepo := database.NewMockOrchestrationRepository()
	comparisonRepo := database.NewMockComparisonRepository()
	cacheRepo := cache.NewMockCacheRepository()

	// 서비스들 초기화
	endpointService := service.NewEndpointService(endpointRepo, log, metricsCollector)
	routingService := service.NewRoutingService(routingRepo, cacheRepo, log, metricsCollector)

	circuitBreakerService := service.NewCircuitBreakerService(log, metricsCollector)
	httpClient := httpclient.NewHTTPClientAdapterWithCircuitBreaker(30*time.Second, circuitBreakerService)

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

	healthService := service.NewHealthCheckService(routingRepo, endpointRepo, cacheRepo, log)

	// HTTP 핸들러 생성
	handler := httpadapter.NewHandler(
		bridgeService,
		healthService,
		endpointService,
		routingService,
		orchestrationService,
		log,
	)

	// 라우터 설정
	router := gin.New()
	router.POST("/api/v1/endpoints", handler.CreateEndpoint)
	router.GET("/api/v1/endpoints", handler.ListEndpoints)
	router.GET("/api/v1/endpoints/:id", handler.GetEndpoint)
	router.PUT("/api/v1/endpoints/:id", handler.UpdateEndpoint)
	router.DELETE("/api/v1/endpoints/:id", handler.DeleteEndpoint)

	router.POST("/api/v1/routing-rules", handler.CreateRoutingRule)
	router.GET("/api/v1/routing-rules", handler.ListRoutingRules)
	router.GET("/api/v1/routing-rules/:id", handler.GetRoutingRule)
	router.PUT("/api/v1/routing-rules/:id", handler.UpdateRoutingRule)
	router.DELETE("/api/v1/routing-rules/:id", handler.DeleteRoutingRule)

	router.POST("/api/v1/orchestration-rules", handler.CreateOrchestrationRule)
	router.GET("/api/v1/orchestration-rules/:id", handler.GetOrchestrationRule)
	router.PUT("/api/v1/orchestration-rules/:id", handler.UpdateOrchestrationRule)
	router.GET("/api/v1/orchestration-rules/:id/evaluate-transition", handler.EvaluateTransition)
	router.POST("/api/v1/orchestration-rules/:id/execute-transition", handler.ExecuteTransition)

	t.Run("Complete CRUD Flow", func(t *testing.T) {
		// 1. 엔드포인트 생성
		legacyEndpointReq := httpadapter.CreateEndpointRequest{
			Name:        "Legacy User API",
			Description: "레거시 사용자 API",
			BaseURL:     "https://legacy-api.example.com",
			Path:        "/api/v1/users",
			HealthURL:   "https://legacy-api.example.com/health",
			Method:      "GET",
			IsActive:    true,
			Timeout:     30000,
			RetryCount:  3,
			Priority:    1,
		}

		legacyEndpointResp := testCreateEndpoint(t, router, legacyEndpointReq)
		require.NotEmpty(t, legacyEndpointResp.ID)

		modernEndpointReq := httpadapter.CreateEndpointRequest{
			Name:        "Modern User API",
			Description: "모던 사용자 API",
			BaseURL:     "https://modern-api.example.com",
			Path:        "/api/v2/users",
			HealthURL:   "https://modern-api.example.com/health",
			Method:      "GET",
			IsActive:    true,
			Timeout:     30000,
			RetryCount:  3,
			Priority:    2,
		}

		modernEndpointResp := testCreateEndpoint(t, router, modernEndpointReq)
		require.NotEmpty(t, modernEndpointResp.ID)

		// 2. 엔드포인트 목록 조회
		endpoints := testListEndpoints(t, router)
		assert.Len(t, endpoints, 2)

		// 3. 라우팅 규칙 생성
		routingRuleReq := httpadapter.CreateRoutingRuleRequest{
			Name:        "User API Routing Rule",
			Description: "사용자 API 라우팅 규칙",
			PathPattern: "/api/users/*",
			Method:      "GET",
			Priority:    1,
			IsActive:    true,
			LegacyEndpoint: &httpadapter.EndpointReference{
				ID: legacyEndpointResp.ID,
			},
			ModernEndpoint: &httpadapter.EndpointReference{
				ID: modernEndpointResp.ID,
			},
		}

		routingRuleResp := testCreateRoutingRule(t, router, routingRuleReq)
		require.NotEmpty(t, routingRuleResp.ID)

		// 4. 라우팅 규칙 목록 조회
		routingRules := testListRoutingRules(t, router)
		assert.Len(t, routingRules, 1)

		// 5. 오케스트레이션 규칙 생성
		orchestrationRuleReq := httpadapter.CreateOrchestrationRuleRequest{
			Name:          "User API Orchestration",
			Description:   "사용자 API 오케스트레이션 규칙",
			RoutingRuleID: routingRuleResp.ID,
			LegacyEndpoint: &httpadapter.EndpointReference{
				ID: legacyEndpointResp.ID,
			},
			ModernEndpoint: &httpadapter.EndpointReference{
				ID: modernEndpointResp.ID,
			},
			CurrentMode: "PARALLEL",
			IsActive:    true,
			TransitionConfig: &httpadapter.TransitionConfigRequest{
				AutoTransitionEnabled:    true,
				MatchRateThreshold:       0.95,
				StabilityPeriodHours:     24,
				MinRequestsForTransition: 100,
				RollbackThreshold:        0.90,
			},
			ComparisonConfig: &httpadapter.ComparisonConfigRequest{
				Enabled:               true,
				IgnoreFields:          []string{"timestamp", "requestId"},
				AllowableDifference:   0.01,
				StrictMode:            false,
				SaveComparisonHistory: true,
			},
		}

		orchestrationRuleResp := testCreateOrchestrationRule(t, router, orchestrationRuleReq)
		require.NotEmpty(t, orchestrationRuleResp.ID)

		// 6. 오케스트레이션 규칙 조회
		retrievedRule := testGetOrchestrationRule(t, router, routingRuleResp.ID)
		assert.Equal(t, orchestrationRuleResp.ID, retrievedRule.ID)
		assert.Equal(t, "PARALLEL", retrievedRule.CurrentMode)

		// 7. 전환 평가
		evaluation := testEvaluateTransition(t, router, routingRuleResp.ID)
		assert.NotNil(t, evaluation)

		// 8. 전환 실행
		transitionReq := struct {
			NewMode string `json:"new_mode"`
		}{
			NewMode: "MODERN_ONLY",
		}

		transitionResp := testExecuteTransition(t, router, routingRuleResp.ID, transitionReq)
		assert.Equal(t, "transition executed successfully", transitionResp["message"])

		// 9. 엔드포인트 수정
		updateReq := httpadapter.UpdateEndpointRequest{
			Name:     stringPtr("Updated Legacy User API"),
			IsActive: boolPtr(false),
		}

		updatedEndpoint := testUpdateEndpoint(t, router, legacyEndpointResp.ID, updateReq)
		assert.Equal(t, "Updated Legacy User API", updatedEndpoint.Name)
		assert.False(t, updatedEndpoint.IsActive)

		// 10. 엔드포인트 삭제
		testDeleteEndpoint(t, router, modernEndpointResp.ID)

		// 삭제 후 목록 확인
		finalEndpoints := testListEndpoints(t, router)
		assert.Len(t, finalEndpoints, 1) // 하나만 남아야 함
	})
}

// 헬퍼 함수들

func testCreateEndpoint(t *testing.T, router *gin.Engine, req httpadapter.CreateEndpointRequest) *httpadapter.EndpointResponse {
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/v1/endpoints", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp httpadapter.EndpointResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return &resp
}

func testListEndpoints(t *testing.T, router *gin.Engine) []*httpadapter.EndpointResponse {
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/v1/endpoints", nil)

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Endpoints []*httpadapter.EndpointResponse `json:"endpoints"`
		Count     int                             `json:"count"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return resp.Endpoints
}

func testCreateRoutingRule(t *testing.T, router *gin.Engine, req httpadapter.CreateRoutingRuleRequest) *httpadapter.RoutingRuleResponse {
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/v1/routing-rules", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp httpadapter.RoutingRuleResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return &resp
}

func testListRoutingRules(t *testing.T, router *gin.Engine) []*httpadapter.RoutingRuleResponse {
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/v1/routing-rules", nil)

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		RoutingRules []*httpadapter.RoutingRuleResponse `json:"routing_rules"`
		Count        int                                `json:"count"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return resp.RoutingRules
}

func testCreateOrchestrationRule(t *testing.T, router *gin.Engine, req httpadapter.CreateOrchestrationRuleRequest) *httpadapter.OrchestrationRuleResponse {
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/v1/orchestration-rules", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp httpadapter.OrchestrationRuleResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return &resp
}

func testGetOrchestrationRule(t *testing.T, router *gin.Engine, routingRuleID string) *httpadapter.OrchestrationRuleResponse {
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/v1/orchestration-rules/"+routingRuleID, nil)

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpadapter.OrchestrationRuleResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return &resp
}

func testEvaluateTransition(t *testing.T, router *gin.Engine, routingRuleID string) map[string]interface{} {
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/v1/orchestration-rules/"+routingRuleID+"/evaluate-transition", nil)

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return resp
}

func testExecuteTransition(t *testing.T, router *gin.Engine, routingRuleID string, req interface{}) map[string]interface{} {
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/v1/orchestration-rules/"+routingRuleID+"/execute-transition", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return resp
}

func testUpdateEndpoint(t *testing.T, router *gin.Engine, endpointID string, req httpadapter.UpdateEndpointRequest) *httpadapter.EndpointResponse {
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("PUT", "/api/v1/endpoints/"+endpointID, bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpadapter.EndpointResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return &resp
}

func testDeleteEndpoint(t *testing.T, router *gin.Engine, endpointID string) {
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("DELETE", "/api/v1/endpoints/"+endpointID, nil)

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// 유틸리티 함수들
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
