package http

import (
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler는 HTTP 인바운드 어댑터의 핵심 구조체입니다.
// Core Layer의 서비스들을 사용하여 HTTP 요청을 처리합니다.
type Handler struct {
	bridgeService port.BridgeService
	healthService port.HealthCheckService
	logger        port.Logger
}

// NewHandler는 새로운 HTTP 핸들러를 생성합니다.
func NewHandler(
	bridgeService port.BridgeService,
	healthService port.HealthCheckService,
	logger port.Logger,
) *Handler {
	return &Handler{
		bridgeService: bridgeService,
		healthService: healthService,
		logger:        logger,
	}
}

// HealthCheck는 서비스의 헬스체크를 처리합니다.
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
func (h *Handler) Status(c *gin.Context) {
	ctx := c.Request.Context()

	status := h.healthService.GetServiceStatus(ctx)
	c.JSON(http.StatusOK, status)
}

// ProcessBridgeRequest는 API Bridge 요청을 처리합니다.
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
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
