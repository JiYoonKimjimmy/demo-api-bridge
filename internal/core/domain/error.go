package domain

import (
	"errors"
	"fmt"
)

// 도메인 에러 정의
var (
	// Request 관련 에러
	ErrInvalidRequestID = errors.New("invalid request ID")
	ErrInvalidMethod    = errors.New("invalid HTTP method")
	ErrInvalidPath      = errors.New("invalid request path")
	ErrInvalidBody      = errors.New("invalid request body")

	// Response 관련 에러
	ErrInvalidResponse = errors.New("invalid response")
	ErrEmptyResponse   = errors.New("empty response")

	// Routing 관련 에러
	ErrRouteNotFound    = errors.New("route not found")
	ErrInvalidRoute     = errors.New("invalid routing rule")
	ErrEndpointNotFound = errors.New("endpoint not found")
	ErrInvalidEndpoint  = errors.New("invalid endpoint configuration")

	// External API 관련 에러
	ErrExternalAPITimeout     = errors.New("external API timeout")
	ErrExternalAPIFailed      = errors.New("external API request failed")
	ErrExternalAPIUnavailable = errors.New("external API unavailable")

	// Cache 관련 에러
	ErrCacheNotFound    = errors.New("cache entry not found")
	ErrCacheExpired     = errors.New("cache entry expired")
	ErrCacheWriteFailed = errors.New("cache write failed")

	// Database 관련 에러
	ErrDatabaseConnection = errors.New("database connection failed")
	ErrDatabaseQuery      = errors.New("database query failed")
	ErrRecordNotFound     = errors.New("record not found")
)

// DomainError는 도메인 레이어의 커스텀 에러입니다.
type DomainError struct {
	Code    string // 에러 코드
	Message string // 에러 메시지
	Cause   error  // 원인 에러
}

// Error는 error 인터페이스를 구현합니다.
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap은 원인 에러를 반환합니다.
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// NewDomainError는 새로운 DomainError를 생성합니다.
func NewDomainError(code, message string, cause error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// ValidationError는 검증 실패 에러입니다.
type ValidationError struct {
	Field   string // 필드명
	Message string // 에러 메시지
}

// Error는 error 인터페이스를 구현합니다.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// NewValidationError는 새로운 ValidationError를 생성합니다.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
