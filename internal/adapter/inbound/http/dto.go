package http

import (
	"encoding/json"
	"time"

	"demo-api-bridge/internal/core/domain"

	"github.com/gin-gonic/gin"
)

// === 요청 DTO ===

// CreateEndpointRequest는 엔드포인트 생성을 위한 요청 DTO입니다.
type CreateEndpointRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	BaseURL     string `json:"base_url" binding:"required,url"`
	HealthURL   string `json:"health_url"`
	IsActive    bool   `json:"is_active"`
	Timeout     int    `json:"timeout"` // milliseconds
	RetryCount  int    `json:"retry_count"`
}

// ToDomain는 CreateEndpointRequest를 Domain APIEndpoint로 변환합니다.
func (req *CreateEndpointRequest) ToDomain() *domain.APIEndpoint {
	return &domain.APIEndpoint{
		Name:        req.Name,
		Description: req.Description,
		BaseURL:     req.BaseURL,
		HealthURL:   req.HealthURL,
		IsActive:    req.IsActive,
		Timeout:     time.Duration(req.Timeout) * time.Millisecond,
		RetryCount:  req.RetryCount,
	}
}

// UpdateEndpointRequest는 엔드포인트 업데이트를 위한 요청 DTO입니다.
type UpdateEndpointRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	BaseURL     *string `json:"base_url,omitempty"`
	HealthURL   *string `json:"health_url,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
	Timeout     *int    `json:"timeout,omitempty"` // milliseconds
	RetryCount  *int    `json:"retry_count,omitempty"`
}

// ApplyTo는 UpdateEndpointRequest의 값을 Domain Endpoint에 적용합니다.
func (req *UpdateEndpointRequest) ApplyTo(endpoint *domain.APIEndpoint) {
	if req.Name != nil {
		endpoint.Name = *req.Name
	}
	if req.Description != nil {
		endpoint.Description = *req.Description
	}
	if req.BaseURL != nil {
		endpoint.BaseURL = *req.BaseURL
	}
	if req.HealthURL != nil {
		endpoint.HealthURL = *req.HealthURL
	}
	if req.IsActive != nil {
		endpoint.IsActive = *req.IsActive
	}
	if req.Timeout != nil {
		endpoint.Timeout = time.Duration(*req.Timeout) * time.Millisecond
	}
	if req.RetryCount != nil {
		endpoint.RetryCount = *req.RetryCount
	}
}

// CreateRoutingRuleRequest는 라우팅 규칙 생성을 위한 요청 DTO입니다.
type CreateRoutingRuleRequest struct {
	Name           string             `json:"name" binding:"required"`
	Description    string             `json:"description"`
	PathPattern    string             `json:"path_pattern" binding:"required"`
	Method         string             `json:"method" binding:"required"`
	Priority       int                `json:"priority"`
	IsActive       bool               `json:"is_active"`
	LegacyEndpoint *EndpointReference `json:"legacy_endpoint"`
	ModernEndpoint *EndpointReference `json:"modern_endpoint"`
	Headers        map[string]string  `json:"headers"`
	QueryParams    map[string]string  `json:"query_params"`
}

// ToDomain는 CreateRoutingRuleRequest를 Domain RoutingRule로 변환합니다.
func (req *CreateRoutingRuleRequest) ToDomain() *domain.RoutingRule {
	rule := &domain.RoutingRule{
		Name:        req.Name,
		Description: req.Description,
		PathPattern: req.PathPattern,
		Method:      req.Method,
		Priority:    req.Priority,
		IsActive:    req.IsActive,
		Headers:     req.Headers,
		QueryParams: req.QueryParams,
	}

	if req.LegacyEndpoint != nil {
		rule.LegacyEndpointID = req.LegacyEndpoint.ID
	}
	if req.ModernEndpoint != nil {
		rule.ModernEndpointID = req.ModernEndpoint.ID
	}

	return rule
}

// UpdateRoutingRuleRequest는 라우팅 규칙 업데이트를 위한 요청 DTO입니다.
type UpdateRoutingRuleRequest struct {
	Name           *string            `json:"name,omitempty"`
	Description    *string            `json:"description,omitempty"`
	PathPattern    *string            `json:"path_pattern,omitempty"`
	Method         *string            `json:"method,omitempty"`
	Priority       *int               `json:"priority,omitempty"`
	IsActive       *bool              `json:"is_active,omitempty"`
	LegacyEndpoint *EndpointReference `json:"legacy_endpoint,omitempty"`
	ModernEndpoint *EndpointReference `json:"modern_endpoint,omitempty"`
	Headers        map[string]string  `json:"headers,omitempty"`
	QueryParams    map[string]string  `json:"query_params,omitempty"`
}

// ApplyTo는 UpdateRoutingRuleRequest의 값을 Domain RoutingRule에 적용합니다.
func (req *UpdateRoutingRuleRequest) ApplyTo(rule *domain.RoutingRule) {
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.PathPattern != nil {
		rule.PathPattern = *req.PathPattern
	}
	if req.Method != nil {
		rule.Method = *req.Method
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}
	if req.LegacyEndpoint != nil {
		rule.LegacyEndpointID = req.LegacyEndpoint.ID
	}
	if req.ModernEndpoint != nil {
		rule.ModernEndpointID = req.ModernEndpoint.ID
	}
	if req.Headers != nil {
		rule.Headers = req.Headers
	}
	if req.QueryParams != nil {
		rule.QueryParams = req.QueryParams
	}
}

// EndpointReference는 엔드포인트 참조를 위한 DTO입니다.
type EndpointReference struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name,omitempty"`
}

// === 응답 DTO ===

// EndpointResponse는 엔드포인트 응답 DTO입니다.
type EndpointResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	BaseURL     string    `json:"base_url"`
	HealthURL   string    `json:"health_url"`
	IsActive    bool      `json:"is_active"`
	Timeout     int       `json:"timeout"` // milliseconds
	RetryCount  int       `json:"retry_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FromDomain는 Domain Endpoint를 EndpointResponse로 변환합니다.
func (resp *EndpointResponse) FromDomain(endpoint *domain.APIEndpoint) {
	resp.ID = endpoint.ID
	resp.Name = endpoint.Name
	resp.Description = endpoint.Description
	resp.BaseURL = endpoint.BaseURL
	resp.HealthURL = endpoint.HealthURL
	resp.IsActive = endpoint.IsActive
	resp.Timeout = int(endpoint.Timeout.Milliseconds())
	resp.RetryCount = endpoint.RetryCount
	resp.CreatedAt = endpoint.CreatedAt
	resp.UpdatedAt = endpoint.UpdatedAt
}

// RoutingRuleResponse는 라우팅 규칙 응답 DTO입니다.
type RoutingRuleResponse struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	PathPattern    string             `json:"path_pattern"`
	Method         string             `json:"method"`
	Priority       int                `json:"priority"`
	IsActive       bool               `json:"is_active"`
	LegacyEndpoint *EndpointReference `json:"legacy_endpoint"`
	ModernEndpoint *EndpointReference `json:"modern_endpoint"`
	Headers        map[string]string  `json:"headers"`
	QueryParams    map[string]string  `json:"query_params"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// FromDomain는 Domain RoutingRule을 RoutingRuleResponse로 변환합니다.
func (resp *RoutingRuleResponse) FromDomain(rule *domain.RoutingRule) {
	resp.ID = rule.ID
	resp.Name = rule.Name
	resp.Description = rule.Description
	resp.PathPattern = rule.PathPattern
	resp.Method = rule.Method
	resp.Priority = rule.Priority
	resp.IsActive = rule.IsActive
	resp.Headers = rule.Headers
	resp.QueryParams = rule.QueryParams
	resp.CreatedAt = rule.CreatedAt
	resp.UpdatedAt = rule.UpdatedAt

	// 엔드포인트 참조 설정 (실제로는 엔드포인트 정보를 조회해야 함)
	if rule.LegacyEndpointID != "" {
		resp.LegacyEndpoint = &EndpointReference{
			ID: rule.LegacyEndpointID,
		}
	}
	if rule.ModernEndpointID != "" {
		resp.ModernEndpoint = &EndpointReference{
			ID: rule.ModernEndpointID,
		}
	}
}

// HealthResponse는 헬스체크 응답 DTO입니다.
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Timestamp string            `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// StatusResponse는 상세 상태 응답 DTO입니다.
type StatusResponse struct {
	Service     string                 `json:"service"`
	Version     string                 `json:"version"`
	Timestamp   string                 `json:"timestamp"`
	Uptime      string                 `json:"uptime"`
	Environment string                 `json:"environment"`
	Metrics     map[string]interface{} `json:"metrics"`
}

// ErrorResponse는 에러 응답 DTO입니다.
type ErrorResponse struct {
	Error     bool   `json:"error"`
	Message   string `json:"message"`
	TraceID   string `json:"trace_id,omitempty"`
	Timestamp string `json:"timestamp"`
	Details   string `json:"details,omitempty"`
}

// APIRequest는 API 요청을 위한 DTO입니다.
type APIRequest struct {
	Method      string            `json:"method" binding:"required"`
	Path        string            `json:"path" binding:"required"`
	Headers     map[string]string `json:"headers"`
	QueryParams map[string]string `json:"query_params"`
	Body        json.RawMessage   `json:"body"`
}

// APIResponse는 API 응답을 위한 DTO입니다.
type APIResponse struct {
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        json.RawMessage   `json:"body"`
	ContentType string            `json:"content_type"`
	Duration    int64             `json:"duration"` // milliseconds
}

// === 유틸리티 함수 ===

// ToEndpointResponse는 Domain Endpoint를 EndpointResponse로 변환합니다.
func ToEndpointResponse(endpoint *domain.APIEndpoint) *EndpointResponse {
	resp := &EndpointResponse{}
	resp.FromDomain(endpoint)
	return resp
}

// ToEndpointResponseList는 Domain Endpoint 리스트를 EndpointResponse 리스트로 변환합니다.
func ToEndpointResponseList(endpoints []*domain.APIEndpoint) []*EndpointResponse {
	responses := make([]*EndpointResponse, len(endpoints))
	for i, endpoint := range endpoints {
		responses[i] = ToEndpointResponse(endpoint)
	}
	return responses
}

// ToRoutingRuleResponse는 Domain RoutingRule을 RoutingRuleResponse로 변환합니다.
func ToRoutingRuleResponse(rule *domain.RoutingRule) *RoutingRuleResponse {
	resp := &RoutingRuleResponse{}
	resp.FromDomain(rule)
	return resp
}

// ToRoutingRuleResponseList는 Domain RoutingRule 리스트를 RoutingRuleResponse 리스트로 변환합니다.
func ToRoutingRuleResponseList(rules []*domain.RoutingRule) []*RoutingRuleResponse {
	responses := make([]*RoutingRuleResponse, len(rules))
	for i, rule := range rules {
		responses[i] = ToRoutingRuleResponse(rule)
	}
	return responses
}

// ToHealthResponse는 Domain HealthStatus를 HealthResponse로 변환합니다.
func ToHealthResponse(status domain.HealthStatus) *HealthResponse {
	return &HealthResponse{
		Status:    string(status),
		Service:   "api-bridge",
		Version:   "0.1.0",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    "0s",
	}
}

// ToReadinessResponse는 Domain ReadinessStatus를 응답으로 변환합니다.
func ToReadinessResponse(status domain.ReadinessStatus) gin.H {
	return gin.H{
		"status":    string(status),
		"ready":     status == domain.READY,
		"checks":    make(map[string]string),
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

// ToStatusResponse는 Domain ServiceStatus를 StatusResponse로 변환합니다.
func ToStatusResponse(status *domain.ServiceStatus) *StatusResponse {
	return &StatusResponse{
		Service:     status.ServiceName,
		Version:     status.Version,
		Timestamp:   status.Timestamp.Format(time.RFC3339),
		Uptime:      status.Uptime.String(),
		Environment: status.Environment,
		Metrics:     status.Metrics,
	}
}

// ToErrorResponse는 에러를 ErrorResponse로 변환합니다.
func ToErrorResponse(err error, traceID string) *ErrorResponse {
	return &ErrorResponse{
		Error:     true,
		Message:   err.Error(),
		TraceID:   traceID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}
