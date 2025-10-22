package http

// @title API Bridge Service
// @version 1.0
// @description API Bridge Service for Legacy and Modern System Integration
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:10019
// @BasePath /api/v1
// @schemes http https
// @produce json
// @consume json
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

import (
	"crypto/rand"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"encoding/hex"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler는 HTTP 인바운드 어댑터의 핵심 구조체입니다.
// Core Layer의 서비스들을 사용하여 HTTP 요청을 처리합니다.
type Handler struct {
	bridgeService        port.BridgeService
	healthService        port.HealthCheckService
	endpointService      port.EndpointService
	routingService       port.RoutingService
	orchestrationService port.OrchestrationService
	logger               port.Logger
	shutdownChannel      chan os.Signal
}

// NewHandler는 새로운 HTTP 핸들러를 생성합니다.
func NewHandler(
	bridgeService port.BridgeService,
	healthService port.HealthCheckService,
	endpointService port.EndpointService,
	routingService port.RoutingService,
	orchestrationService port.OrchestrationService,
	logger port.Logger,
) *Handler {
	shutdownChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownChannel, os.Interrupt, syscall.SIGTERM)

	return &Handler{
		bridgeService:        bridgeService,
		healthService:        healthService,
		endpointService:      endpointService,
		routingService:       routingService,
		orchestrationService: orchestrationService,
		logger:               logger,
		shutdownChannel:      shutdownChannel,
	}
}

// HealthCheck는 서비스의 헬스체크를 처리합니다.
// @Summary Health Check
// @Description Check if the service is healthy
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{} "Service is healthy"
// @Failure 503 {object} map[string]interface{} "Service is unhealthy"
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	err := h.healthService.CheckHealth(ctx)
	if err != nil {
		h.logger.WithContext(ctx).Error("health check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "unhealthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"error":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ReadinessCheck는 서비스의 준비 상태를 확인합니다.
// @Summary Readiness Check
// @Description Check if the service is ready to accept requests
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{} "Service is ready"
// @Failure 503 {object} map[string]interface{} "Service is not ready"
// @Router /ready [get]
func (h *Handler) ReadinessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	err := h.healthService.CheckReadiness(ctx)
	if err != nil {
		h.logger.WithContext(ctx).Error("readiness check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"ready":  false,
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"ready":  true,
	})
}

// Status는 상세한 서비스 상태를 반환합니다.
// @Summary Service Status
// @Description Get detailed service status information
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{} "Service status retrieved successfully"
// @Router /api/v1/status [get]
func (h *Handler) Status(c *gin.Context) {
	ctx := c.Request.Context()

	status := h.healthService.GetServiceStatus(ctx)
	c.JSON(http.StatusOK, status)
}

// ProcessBridgeRequest는 API Bridge 요청을 처리합니다.
// @Summary Process Bridge Request
// @Description Process API bridge request using any HTTP method
// @Tags Bridge
// @Accept json
// @Produce json
// @Param path path string true "API path to bridge"
// @Success 200 {object} map[string]interface{} "Request processed successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/bridge/{path} [get]
// @Router /api/v1/bridge/{path} [post]
// @Router /api/v1/bridge/{path} [put]
// @Router /api/v1/bridge/{path} [delete]
func (h *Handler) ProcessBridgeRequest(c *gin.Context) {
	ctx := c.Request.Context()

	// 요청 파라미터 추출
	path := c.Param("path")
	method := c.Request.Method

	// 도메인 요청 객체 생성
	requestID := generateRequestID()
	request := domain.NewRequest(requestID, method, path)

	// 헤더 복사
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			request.SetHeader(key, values[0])
		}
	}

	// 쿼리 파라미터 복사
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			request.SetQueryParam(key, values[0])
		}
	}

	// 요청 본문 읽기 (POST, PUT 등)
	if method == "POST" || method == "PUT" || method == "PATCH" {
		body, err := c.GetRawData()
		if err != nil {
			h.logger.WithContext(ctx).Error("failed to read request body", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		request.Body = body
	}

	// 브리지 서비스로 요청 처리
	response, err := h.bridgeService.ProcessRequest(ctx, request)
	if err != nil {
		h.logger.WithContext(ctx).Error("bridge request processing failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 응답 헤더 설정
	for key, value := range response.Headers {
		c.Header(key, value)
	}

	// 응답 반환
	c.Data(response.StatusCode, "application/json", response.Body)
}

// Metrics는 Prometheus 메트릭을 반환합니다.
// @Summary Prometheus Metrics
// @Description Get Prometheus metrics for monitoring
// @Tags metrics
// @Produce text/plain
// @Success 200 {string} string "Metrics retrieved successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /metrics [get]
func (h *Handler) Metrics(c *gin.Context) {
	// TODO: Prometheus 메트릭 엔드포인트 구현
	c.JSON(http.StatusOK, gin.H{
		"message": "Metrics endpoint - to be implemented",
	})
}

// generateRequestID는 요청 ID를 생성합니다.
func generateRequestID() string {
	return "req-" + time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString은 랜덤 문자열을 생성합니다.
func randomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// fallback to time-based approach
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405")))[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}

// === APIEndpoint CRUD 핸들러 ===

// CreateEndpoint는 새로운 엔드포인트를 생성합니다.
// @Summary Create a new API endpoint
// @Description Create a new API endpoint with the provided information
// @Tags endpoints
// @Accept json
// @Produce json
// @Param endpoint body CreateEndpointRequest true "Endpoint information"
// @Success 201 {object} EndpointResponse "Endpoint created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/endpoints [post]
func (h *Handler) CreateEndpoint(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 도메인 객체 생성
	endpoint := req.ToDomain()
	endpoint.ID = generateEndpointID()

	// 서비스 호출
	if err := h.endpointService.CreateEndpoint(ctx, endpoint); err != nil {
		h.logger.WithContext(ctx).Error("failed to create endpoint", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create endpoint", "details": err.Error()})
		return
	}

	// 응답 생성
	response := ToEndpointResponse(endpoint)
	c.JSON(http.StatusCreated, response)
}

// GetEndpoint는 엔드포인트를 조회합니다.
// @Summary Get API endpoint by ID
// @Description Retrieve a specific API endpoint by its ID
// @Tags endpoints
// @Produce json
// @Param id path string true "Endpoint ID"
// @Success 200 {object} map[string]interface{} "Endpoint retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Endpoint not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/endpoints/{id} [get]
func (h *Handler) GetEndpoint(c *gin.Context) {
	ctx := c.Request.Context()
	endpointID := c.Param("id")

	endpoint, err := h.endpointService.GetEndpoint(ctx, endpointID)
	if err != nil {
		h.logger.WithContext(ctx).Error("failed to get endpoint", "endpoint_id", endpointID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found", "details": err.Error()})
		return
	}

	response := ToEndpointResponse(endpoint)
	c.JSON(http.StatusOK, response)
}

// ListEndpoints는 모든 엔드포인트를 조회합니다.
// @Summary Get all API endpoints
// @Description Retrieve a list of all API endpoints
// @Tags endpoints
// @Produce json
// @Success 200 {object} map[string]interface{} "List of endpoints"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/endpoints [get]
func (h *Handler) ListEndpoints(c *gin.Context) {
	ctx := c.Request.Context()

	endpoints, err := h.endpointService.ListEndpoints(ctx)
	if err != nil {
		h.logger.WithContext(ctx).Error("failed to list endpoints", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list endpoints", "details": err.Error()})
		return
	}

	responses := ToEndpointResponseList(endpoints)
	c.JSON(http.StatusOK, gin.H{"endpoints": responses, "count": len(responses)})
}

// UpdateEndpoint는 엔드포인트를 수정합니다.
// @Summary Update API endpoint
// @Description Update an existing API endpoint
// @Tags endpoints
// @Accept json
// @Produce json
// @Param id path string true "Endpoint ID"
// @Param endpoint body object true "Endpoint information"
// @Success 200 {object} map[string]interface{} "Endpoint updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 404 {object} map[string]interface{} "Endpoint not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/endpoints/{id} [put]
func (h *Handler) UpdateEndpoint(c *gin.Context) {
	ctx := c.Request.Context()
	endpointID := c.Param("id")

	var req UpdateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 기존 엔드포인트 조회
	endpoint, err := h.endpointService.GetEndpoint(ctx, endpointID)
	if err != nil {
		h.logger.WithContext(ctx).Error("endpoint not found", "endpoint_id", endpointID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found", "details": err.Error()})
		return
	}

	// 업데이트 적용
	req.ApplyTo(endpoint)

	// 서비스 호출
	if err := h.endpointService.UpdateEndpoint(ctx, endpoint); err != nil {
		h.logger.WithContext(ctx).Error("failed to update endpoint", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update endpoint", "details": err.Error()})
		return
	}

	// 응답 생성
	response := ToEndpointResponse(endpoint)
	c.JSON(http.StatusOK, response)
}

// DeleteEndpoint는 엔드포인트를 삭제합니다.
// @Summary Delete API endpoint
// @Description Delete an API endpoint by its ID
// @Tags endpoints
// @Produce json
// @Param id path string true "Endpoint ID"
// @Success 200 {object} map[string]interface{} "Endpoint deleted successfully"
// @Failure 404 {object} map[string]interface{} "Endpoint not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/endpoints/{id} [delete]
func (h *Handler) DeleteEndpoint(c *gin.Context) {
	ctx := c.Request.Context()
	endpointID := c.Param("id")

	if err := h.endpointService.DeleteEndpoint(ctx, endpointID); err != nil {
		h.logger.WithContext(ctx).Error("failed to delete endpoint", "endpoint_id", endpointID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete endpoint", "details": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// generateEndpointID는 엔드포인트 ID를 생성합니다.
func generateEndpointID() string {
	return "endpoint-" + time.Now().Format("20060102150405") + "-" + randomString(6)
}

// === RoutingRule CRUD 핸들러 ===

// CreateRoutingRule은 새로운 라우팅 규칙을 생성합니다.
// @Summary Create a new routing rule
// @Description Create a new routing rule with the provided information
// @Tags routing-rules
// @Accept json
// @Produce json
// @Param routingRule body object true "Routing rule information"
// @Success 201 {object} map[string]interface{} "Routing rule created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/routing-rules [post]
func (h *Handler) CreateRoutingRule(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateRoutingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 도메인 객체 생성
	rule := req.ToDomain()
	rule.ID = generateRoutingRuleID()

	// 서비스 호출
	if err := h.routingService.CreateRule(ctx, rule); err != nil {
		h.logger.WithContext(ctx).Error("failed to create routing rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create routing rule", "details": err.Error()})
		return
	}

	// 응답 생성
	response := ToRoutingRuleResponse(rule)
	c.JSON(http.StatusCreated, response)
}

// GetRoutingRule은 라우팅 규칙을 조회합니다.
// @Summary Get routing rule by ID
// @Description Retrieve a specific routing rule by its ID
// @Tags routing-rules
// @Produce json
// @Param id path string true "Routing rule ID"
// @Success 200 {object} map[string]interface{} "Routing rule retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Routing rule not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/routing-rules/{id} [get]
func (h *Handler) GetRoutingRule(c *gin.Context) {
	ctx := c.Request.Context()
	ruleID := c.Param("id")

	rule, err := h.routingService.GetRule(ctx, ruleID)
	if err != nil {
		h.logger.WithContext(ctx).Error("failed to get routing rule", "rule_id", ruleID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "routing rule not found", "details": err.Error()})
		return
	}

	response := ToRoutingRuleResponse(rule)
	c.JSON(http.StatusOK, response)
}

// ListRoutingRules는 모든 라우팅 규칙을 조회합니다.
// @Summary Get all routing rules
// @Description Retrieve a list of all routing rules
// @Tags routing-rules
// @Produce json
// @Success 200 {object} map[string]interface{} "List of routing rules retrieved successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/routing-rules [get]
func (h *Handler) ListRoutingRules(c *gin.Context) {
	ctx := c.Request.Context()

	rules, err := h.routingService.ListRules(ctx)
	if err != nil {
		h.logger.WithContext(ctx).Error("failed to list routing rules", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list routing rules", "details": err.Error()})
		return
	}

	responses := ToRoutingRuleResponseList(rules)
	c.JSON(http.StatusOK, gin.H{"routing_rules": responses, "count": len(responses)})
}

// UpdateRoutingRule은 라우팅 규칙을 수정합니다.
// @Summary Update routing rule
// @Description Update an existing routing rule
// @Tags routing-rules
// @Accept json
// @Produce json
// @Param id path string true "Routing rule ID"
// @Param routingRule body object true "Routing rule information"
// @Success 200 {object} map[string]interface{} "Routing rule updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 404 {object} map[string]interface{} "Routing rule not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/routing-rules/{id} [put]
func (h *Handler) UpdateRoutingRule(c *gin.Context) {
	ctx := c.Request.Context()
	ruleID := c.Param("id")

	var req UpdateRoutingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 기존 라우팅 규칙 조회
	rule, err := h.routingService.GetRule(ctx, ruleID)
	if err != nil {
		h.logger.WithContext(ctx).Error("routing rule not found", "rule_id", ruleID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "routing rule not found", "details": err.Error()})
		return
	}

	// 업데이트 적용
	req.ApplyTo(rule)

	// 서비스 호출
	if err := h.routingService.UpdateRule(ctx, rule); err != nil {
		h.logger.WithContext(ctx).Error("failed to update routing rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update routing rule", "details": err.Error()})
		return
	}

	// 응답 생성
	response := ToRoutingRuleResponse(rule)
	c.JSON(http.StatusOK, response)
}

// DeleteRoutingRule은 라우팅 규칙을 삭제합니다.
// @Summary Delete routing rule
// @Description Delete a routing rule by its ID
// @Tags routing-rules
// @Produce json
// @Param id path string true "Routing rule ID"
// @Success 200 {object} map[string]interface{} "Routing rule deleted successfully"
// @Failure 404 {object} map[string]interface{} "Routing rule not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/routing-rules/{id} [delete]
func (h *Handler) DeleteRoutingRule(c *gin.Context) {
	ctx := c.Request.Context()
	ruleID := c.Param("id")

	if err := h.routingService.DeleteRule(ctx, ruleID); err != nil {
		h.logger.WithContext(ctx).Error("failed to delete routing rule", "rule_id", ruleID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete routing rule", "details": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// generateRoutingRuleID는 라우팅 규칙 ID를 생성합니다.
func generateRoutingRuleID() string {
	return "rule-" + time.Now().Format("20060102150405") + "-" + randomString(6)
}

// === OrchestrationRule CRUD 핸들러 ===

// CreateOrchestrationRule은 새로운 오케스트레이션 규칙을 생성합니다.
// @Summary Create a new orchestration rule
// @Description Create a new orchestration rule with the provided information
// @Tags orchestration-rules
// @Accept json
// @Produce json
// @Param orchestrationRule body object true "Orchestration rule information"
// @Success 201 {object} map[string]interface{} "Orchestration rule created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/orchestration-rules [post]
func (h *Handler) CreateOrchestrationRule(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateOrchestrationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 도메인 객체 생성
	rule := req.ToDomain()
	rule.ID = generateOrchestrationRuleID()

	// 서비스 호출
	if err := h.orchestrationService.CreateOrchestrationRule(ctx, rule); err != nil {
		h.logger.WithContext(ctx).Error("failed to create orchestration rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create orchestration rule", "details": err.Error()})
		return
	}

	// 응답 생성
	response := ToOrchestrationRuleResponse(rule)
	c.JSON(http.StatusCreated, response)
}

// GetOrchestrationRule은 오케스트레이션 규칙을 조회합니다.
// @Summary Get orchestration rule by ID
// @Description Retrieve a specific orchestration rule by its ID
// @Tags orchestration-rules
// @Produce json
// @Param id path string true "Orchestration rule ID"
// @Success 200 {object} map[string]interface{} "Orchestration rule retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Orchestration rule not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/orchestration-rules/{id} [get]
func (h *Handler) GetOrchestrationRule(c *gin.Context) {
	ctx := c.Request.Context()
	routingRuleID := c.Param("id")

	rule, err := h.orchestrationService.GetOrchestrationRule(ctx, routingRuleID)
	if err != nil {
		h.logger.WithContext(ctx).Error("failed to get orchestration rule", "routing_rule_id", routingRuleID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "orchestration rule not found", "details": err.Error()})
		return
	}

	response := ToOrchestrationRuleResponse(rule)
	c.JSON(http.StatusOK, response)
}

// UpdateOrchestrationRule은 오케스트레이션 규칙을 수정합니다.
// @Summary Update orchestration rule
// @Description Update an existing orchestration rule
// @Tags orchestration-rules
// @Accept json
// @Produce json
// @Param id path string true "Orchestration rule ID"
// @Param orchestrationRule body object true "Orchestration rule information"
// @Success 200 {object} map[string]interface{} "Orchestration rule updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 404 {object} map[string]interface{} "Orchestration rule not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/orchestration-rules/{id} [put]
func (h *Handler) UpdateOrchestrationRule(c *gin.Context) {
	ctx := c.Request.Context()
	routingRuleID := c.Param("id")

	var req UpdateOrchestrationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 기존 오케스트레이션 규칙 조회
	rule, err := h.orchestrationService.GetOrchestrationRule(ctx, routingRuleID)
	if err != nil {
		h.logger.WithContext(ctx).Error("orchestration rule not found", "routing_rule_id", routingRuleID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "orchestration rule not found", "details": err.Error()})
		return
	}

	// 업데이트 적용
	req.ApplyTo(rule)

	// 서비스 호출
	if err := h.orchestrationService.UpdateOrchestrationRule(ctx, rule); err != nil {
		h.logger.WithContext(ctx).Error("failed to update orchestration rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update orchestration rule", "details": err.Error()})
		return
	}

	// 응답 생성
	response := ToOrchestrationRuleResponse(rule)
	c.JSON(http.StatusOK, response)
}

// EvaluateTransition은 전환 가능성을 평가합니다.
// @Summary Evaluate transition
// @Description Evaluate if a transition should be made for the orchestration rule
// @Tags orchestration-rules
// @Produce json
// @Param id path string true "Orchestration rule ID"
// @Success 200 {object} map[string]interface{} "Transition evaluation completed"
// @Failure 404 {object} map[string]interface{} "Orchestration rule not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/orchestration-rules/{id}/evaluate-transition [get]
func (h *Handler) EvaluateTransition(c *gin.Context) {
	ctx := c.Request.Context()
	routingRuleID := c.Param("id")

	// 오케스트레이션 규칙 조회
	rule, err := h.orchestrationService.GetOrchestrationRule(ctx, routingRuleID)
	if err != nil {
		h.logger.WithContext(ctx).Error("orchestration rule not found", "routing_rule_id", routingRuleID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "orchestration rule not found", "details": err.Error()})
		return
	}

	// 전환 평가
	canTransition, err := h.orchestrationService.EvaluateTransition(ctx, rule)
	if err != nil {
		h.logger.WithContext(ctx).Error("failed to evaluate transition", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to evaluate transition", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"can_transition": canTransition,
		"current_mode":   string(rule.CurrentMode),
		"rule_id":        rule.ID,
	})
}

// ExecuteTransition은 API 모드를 전환합니다.
// @Summary Execute transition
// @Description Execute a transition to a new mode for the orchestration rule
// @Tags orchestration-rules
// @Accept json
// @Produce json
// @Param id path string true "Orchestration rule ID"
// @Param transition body object true "Transition information"
// @Success 200 {object} map[string]interface{} "Transition executed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 404 {object} map[string]interface{} "Orchestration rule not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/orchestration-rules/{id}/execute-transition [post]
func (h *Handler) ExecuteTransition(c *gin.Context) {
	ctx := c.Request.Context()
	routingRuleID := c.Param("id")

	var req struct {
		NewMode string `json:"new_mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// 오케스트레이션 규칙 조회
	rule, err := h.orchestrationService.GetOrchestrationRule(ctx, routingRuleID)
	if err != nil {
		h.logger.WithContext(ctx).Error("orchestration rule not found", "routing_rule_id", routingRuleID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "orchestration rule not found", "details": err.Error()})
		return
	}

	// 새로운 모드 변환
	var newMode domain.APIMode
	switch req.NewMode {
	case "LEGACY_ONLY":
		newMode = domain.LEGACY_ONLY
	case "MODERN_ONLY":
		newMode = domain.MODERN_ONLY
	case "PARALLEL":
		newMode = domain.PARALLEL
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mode", "details": "mode must be LEGACY_ONLY, MODERN_ONLY, or PARALLEL"})
		return
	}

	// 전환 실행
	if err := h.orchestrationService.ExecuteTransition(ctx, rule, newMode); err != nil {
		h.logger.WithContext(ctx).Error("failed to execute transition", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to execute transition", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "transition executed successfully",
		"from_mode": string(rule.CurrentMode),
		"to_mode":   string(newMode),
		"rule_id":   rule.ID,
	})
}

// generateOrchestrationRuleID는 오케스트레이션 규칙 ID를 생성합니다.
func generateOrchestrationRuleID() string {
	return "orch-" + time.Now().Format("20060102150405") + "-" + randomString(6)
}

// GracefulShutdown는 서비스를 안전하게 종료합니다.
// @Summary Graceful shutdown
// @Description Gracefully shutdown the API Bridge service
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Shutdown initiated successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/shutdown [post]
func (h *Handler) GracefulShutdown(c *gin.Context) {
	ctx := c.Request.Context()

	h.logger.WithContext(ctx).Info("Graceful shutdown requested via API")

	// 응답을 먼저 보냅니다
	c.JSON(http.StatusOK, gin.H{
		"message":   "Graceful shutdown initiated",
		"timestamp": time.Now().Format(time.RFC3339),
	})

	// 응답을 보낸 후 shutdown 신호를 전송합니다
	go func() {
		// 잠시 대기하여 응답이 완전히 전송되도록 합니다
		time.Sleep(100 * time.Millisecond)
		h.shutdownChannel <- os.Interrupt
	}()
}

// GetShutdownChannel는 shutdown 채널을 반환합니다.
func (h *Handler) GetShutdownChannel() <-chan os.Signal {
	return h.shutdownChannel
}
