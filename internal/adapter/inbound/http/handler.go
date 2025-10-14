package http

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// Handler는 HTTP 인바운드 어댑터의 핵심 구조체입니다.
// Core Layer의 서비스들을 사용하여 HTTP 요청을 처리합니다.
type Handler struct {
	bridgeService   port.BridgeService
	routingService  port.RoutingService
	endpointService port.EndpointService
	healthService   port.HealthService
	logger          *logger.Logger
	metrics         *metrics.Metrics
}

// NewHandler는 새로운 HTTP 핸들러를 생성합니다.
func NewHandler(
	bridgeService port.BridgeService,
	routingService port.RoutingService,
	endpointService port.EndpointService,
	healthService port.HealthService,
	logger *logger.Logger,
	metrics *metrics.Metrics,
) *Handler {
	return &Handler{
		bridgeService:   bridgeService,
		routingService:  routingService,
		endpointService: endpointService,
		healthService:   healthService,
		logger:          logger,
		metrics:         metrics,
	}
}

// RegisterRoutes는 모든 라우트를 등록합니다.
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	// API v1 그룹
	v1 := router.Group("/api/v1")
	{
		// API Bridge 메인 엔드포인트 - 모든 요청을 처리
		v1.Any("/*path", h.handleAPIRequest)

		// 관리 API
		v1.GET("/endpoints", h.listEndpoints)
		v1.POST("/endpoints", h.createEndpoint)
		v1.GET("/endpoints/:id", h.getEndpoint)
		v1.PUT("/endpoints/:id", h.updateEndpoint)
		v1.DELETE("/endpoints/:id", h.deleteEndpoint)

		v1.GET("/routing-rules", h.listRoutingRules)
		v1.POST("/routing-rules", h.createRoutingRule)
		v1.GET("/routing-rules/:id", h.getRoutingRule)
		v1.PUT("/routing-rules/:id", h.updateRoutingRule)
		v1.DELETE("/routing-rules/:id", h.deleteRoutingRule)
	}

	// Health Check 엔드포인트
	router.GET("/health", h.healthCheck)
	router.GET("/ready", h.readinessCheck)
	router.GET("/api/v1/status", h.status)
}

// handleAPIRequest는 메인 API Bridge 요청을 처리합니다.
// 모든 외부 API 요청이 이 핸들러를 통해 라우팅됩니다.
func (h *Handler) handleAPIRequest(c *gin.Context) {
	startTime := time.Now()
	ctx := c.Request.Context()

	// 요청 정보 추출
	path := c.Param("path")
	method := c.Request.Method

	// Trace ID 생성 (없는 경우)
	traceID := c.GetHeader("X-Trace-ID")
	if traceID == "" {
		traceID = generateTraceID()
	}

	// Context에 Trace ID 추가
	ctx = logger.WithTraceID(ctx, traceID)
	c.Request = c.Request.WithContext(ctx)

	// 요청 로깅
	h.logger.Info(ctx, "API Bridge request received",
		"method", method,
		"path", path,
		"trace_id", traceID,
		"user_agent", c.GetHeader("User-Agent"),
		"remote_addr", c.ClientIP(),
	)

	// 메트릭 시작
	h.metrics.RecordHTTPRequestStarted(method, path)

	// 요청 변환
	req, err := h.buildDomainRequest(c)
	if err != nil {
		h.handleError(c, err, "Failed to build domain request")
		return
	}

	// Bridge Service 호출
	response, err := h.bridgeService.ProcessRequest(ctx, req)
	if err != nil {
		h.handleError(c, err, "Failed to process request")
		return
	}

	// 응답 변환 및 전송
	h.sendResponse(c, response)

	// 메트릭 기록
	duration := time.Since(startTime)
	h.metrics.RecordHTTPRequestCompleted(method, path, response.StatusCode, duration)

	// 응답 로깅
	h.logger.Info(ctx, "API Bridge request completed",
		"method", method,
		"path", path,
		"trace_id", traceID,
		"status_code", response.StatusCode,
		"duration_ms", duration.Milliseconds(),
	)
}

// buildDomainRequest는 Gin Context를 Domain Request로 변환합니다.
func (h *Handler) buildDomainRequest(c *gin.Context) (*domain.Request, error) {
	// 헤더 추출
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0] // 첫 번째 값만 사용
		}
	}

	// 쿼리 파라미터 추출
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0] // 첫 번째 값만 사용
		}
	}

	// 요청 본문 읽기
	var body []byte
	if c.Request.Body != nil {
		body = make([]byte, 0)
		// 실제 구현에서는 본문을 읽어야 하지만,
		// Gin의 경우 이미 읽혀졌을 수 있으므로 주의
	}

	// Request 생성
	req := &domain.Request{
		Method:      c.Request.Method,
		Path:        c.Param("path"),
		Headers:     headers,
		QueryParams: queryParams,
		Body:        body,
		SourceIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Timestamp:   time.Now(),
	}

	return req, nil
}

// sendResponse는 Domain Response를 HTTP 응답으로 변환하여 전송합니다.
func (h *Handler) sendResponse(c *gin.Context, response *domain.Response) {
	// 헤더 설정
	for key, value := range response.Headers {
		c.Header(key, value)
	}

	// Trace ID 헤더 추가
	if traceID := logger.GetTraceIDFromContext(c.Request.Context()); traceID != "" {
		c.Header("X-Trace-ID", traceID)
	}

	// 상태 코드 설정
	c.Status(response.StatusCode)

	// 본문 전송
	if len(response.Body) > 0 {
		c.Data(response.StatusCode, response.ContentType, response.Body)
	} else {
		c.Status(response.StatusCode)
	}
}

// handleError는 에러를 처리하고 적절한 HTTP 응답을 반환합니다.
func (h *Handler) handleError(c *gin.Context, err error, message string) {
	ctx := c.Request.Context()

	h.logger.Error(ctx, message, "error", err)

	// 에러 타입에 따른 상태 코드 결정
	var statusCode int
	var errorMessage string

	switch e := err.(type) {
	case *domain.APIError:
		statusCode = e.StatusCode
		errorMessage = e.Message
	default:
		statusCode = http.StatusInternalServerError
		errorMessage = "Internal Server Error"
	}

	// 메트릭 기록
	path := c.Param("path")
	method := c.Request.Method
	h.metrics.RecordHTTPRequestCompleted(method, path, statusCode, time.Since(time.Now()))

	// 에러 응답 전송
	c.JSON(statusCode, gin.H{
		"error":     true,
		"message":   errorMessage,
		"trace_id":  logger.GetTraceIDFromContext(ctx),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// === Health Check 핸들러들 ===

// healthCheck는 기본 헬스체크를 수행합니다.
func (h *Handler) healthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := h.healthService.CheckHealth(ctx)
	if err != nil {
		h.logger.Error(ctx, "Health check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"error":   err.Error(),
			"service": "api-bridge",
			"version": "0.1.0",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    status.Status,
		"service":   status.ServiceName,
		"version":   status.Version,
		"timestamp": status.Timestamp.Format(time.RFC3339),
		"uptime":    status.Uptime,
	})
}

// readinessCheck는 서비스 준비 상태를 확인합니다.
func (h *Handler) readinessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	ready, err := h.healthService.CheckReadiness(ctx)
	if err != nil {
		h.logger.Error(ctx, "Readiness check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  err.Error(),
		})
		return
	}

	statusCode := http.StatusOK
	if !ready.Ready {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    ready.Status,
		"ready":     ready.Ready,
		"checks":    ready.Checks,
		"timestamp": ready.Timestamp.Format(time.RFC3339),
	})
}

// status는 상세한 서버 상태를 반환합니다.
func (h *Handler) status(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := h.healthService.GetStatus(ctx)
	if err != nil {
		h.logger.Error(ctx, "Status check failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service":     status.ServiceName,
		"version":     status.Version,
		"timestamp":   status.Timestamp.Format(time.RFC3339),
		"uptime":      status.Uptime,
		"environment": status.Environment,
		"metrics":     status.Metrics,
	})
}

// === 엔드포인트 관리 핸들러들 ===

// listEndpoints는 모든 엔드포인트를 조회합니다.
func (h *Handler) listEndpoints(c *gin.Context) {
	ctx := c.Request.Context()

	endpoints, err := h.endpointService.GetAllEndpoints(ctx)
	if err != nil {
		h.handleError(c, err, "Failed to get endpoints")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoints": endpoints,
		"count":     len(endpoints),
	})
}

// createEndpoint는 새로운 엔드포인트를 생성합니다.
func (h *Handler) createEndpoint(c *gin.Context) {
	ctx := c.Request.Context()

	var endpoint domain.Endpoint
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		h.handleError(c, domain.NewValidationError("Invalid endpoint data: "+err.Error()), "Failed to parse endpoint")
		return
	}

	createdEndpoint, err := h.endpointService.CreateEndpoint(ctx, &endpoint)
	if err != nil {
		h.handleError(c, err, "Failed to create endpoint")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"endpoint": createdEndpoint,
	})
}

// getEndpoint는 특정 엔드포인트를 조회합니다.
func (h *Handler) getEndpoint(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.handleError(c, domain.NewValidationError("Invalid endpoint ID"), "Invalid endpoint ID")
		return
	}

	endpoint, err := h.endpointService.GetEndpointByID(ctx, id)
	if err != nil {
		h.handleError(c, err, "Failed to get endpoint")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoint": endpoint,
	})
}

// updateEndpoint는 엔드포인트를 업데이트합니다.
func (h *Handler) updateEndpoint(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.handleError(c, domain.NewValidationError("Invalid endpoint ID"), "Invalid endpoint ID")
		return
	}

	var endpoint domain.Endpoint
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		h.handleError(c, domain.NewValidationError("Invalid endpoint data: "+err.Error()), "Failed to parse endpoint")
		return
	}
	endpoint.ID = id

	updatedEndpoint, err := h.endpointService.UpdateEndpoint(ctx, &endpoint)
	if err != nil {
		h.handleError(c, err, "Failed to update endpoint")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoint": updatedEndpoint,
	})
}

// deleteEndpoint는 엔드포인트를 삭제합니다.
func (h *Handler) deleteEndpoint(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.handleError(c, domain.NewValidationError("Invalid endpoint ID"), "Invalid endpoint ID")
		return
	}

	err = h.endpointService.DeleteEndpoint(ctx, id)
	if err != nil {
		h.handleError(c, err, "Failed to delete endpoint")
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// === 라우팅 규칙 관리 핸들러들 ===

// listRoutingRules는 모든 라우팅 규칙을 조회합니다.
func (h *Handler) listRoutingRules(c *gin.Context) {
	ctx := c.Request.Context()

	rules, err := h.routingService.GetAllRoutingRules(ctx)
	if err != nil {
		h.handleError(c, err, "Failed to get routing rules")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"routing_rules": rules,
		"count":         len(rules),
	})
}

// createRoutingRule은 새로운 라우팅 규칙을 생성합니다.
func (h *Handler) createRoutingRule(c *gin.Context) {
	ctx := c.Request.Context()

	var rule domain.RoutingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		h.handleError(c, domain.NewValidationError("Invalid routing rule data: "+err.Error()), "Failed to parse routing rule")
		return
	}

	createdRule, err := h.routingService.CreateRoutingRule(ctx, &rule)
	if err != nil {
		h.handleError(c, err, "Failed to create routing rule")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"routing_rule": createdRule,
	})
}

// getRoutingRule은 특정 라우팅 규칙을 조회합니다.
func (h *Handler) getRoutingRule(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.handleError(c, domain.NewValidationError("Invalid routing rule ID"), "Invalid routing rule ID")
		return
	}

	rule, err := h.routingService.GetRoutingRuleByID(ctx, id)
	if err != nil {
		h.handleError(c, err, "Failed to get routing rule")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"routing_rule": rule,
	})
}

// updateRoutingRule은 라우팅 규칙을 업데이트합니다.
func (h *Handler) updateRoutingRule(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.handleError(c, domain.NewValidationError("Invalid routing rule ID"), "Invalid routing rule ID")
		return
	}

	var rule domain.RoutingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		h.handleError(c, domain.NewValidationError("Invalid routing rule data: "+err.Error()), "Failed to parse routing rule")
		return
	}
	rule.ID = id

	updatedRule, err := h.routingService.UpdateRoutingRule(ctx, &rule)
	if err != nil {
		h.handleError(c, err, "Failed to update routing rule")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"routing_rule": updatedRule,
	})
}

// deleteRoutingRule은 라우팅 규칙을 삭제합니다.
func (h *Handler) deleteRoutingRule(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.handleError(c, domain.NewValidationError("Invalid routing rule ID"), "Invalid routing rule ID")
		return
	}

	err = h.routingService.DeleteRoutingRule(ctx, id)
	if err != nil {
		h.handleError(c, err, "Failed to delete routing rule")
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// generateTraceID는 새로운 Trace ID를 생성합니다.
func generateTraceID() string {
	return fmt.Sprintf("trace_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}
