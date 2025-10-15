package domain

import (
	"time"
)

// APIEndpoint는 외부 API 엔드포인트 정보를 나타냅니다.
type APIEndpoint struct {
	ID          string        // 엔드포인트 고유 ID
	Name        string        // 엔드포인트 이름
	BaseURL     string        // 기본 URL (예: https://api.example.com)
	Path        string        // 경로 (예: /v1/users)
	HealthURL   string        // 헬스 체크 URL
	Method      string        // HTTP 메서드
	Timeout     time.Duration // 타임아웃
	RetryCount  int           // 재시도 횟수
	IsActive    bool          // 활성화 여부
	Priority    int           // 우선순위 (여러 엔드포인트가 있을 경우)
	Description string        // 설명
	CreatedAt   time.Time     // 생성 시간
	UpdatedAt   time.Time     // 수정 시간
}

// NewAPIEndpoint는 새로운 APIEndpoint를 생성합니다.
func NewAPIEndpoint(id, name, baseURL, path, method string) *APIEndpoint {
	return &APIEndpoint{
		ID:         id,
		Name:       name,
		BaseURL:    baseURL,
		Path:       path,
		Method:     method,
		Timeout:    30 * time.Second,
		RetryCount: 3,
		IsActive:   true,
		Priority:   1,
	}
}

// GetFullURL은 전체 URL을 반환합니다.
func (e *APIEndpoint) GetFullURL() string {
	return e.BaseURL + e.Path
}

// IsValid는 엔드포인트가 유효한지 검증합니다.
func (e *APIEndpoint) IsValid() error {
	if e.ID == "" {
		return NewValidationError("ID", "endpoint ID is required")
	}
	if e.BaseURL == "" {
		return NewValidationError("BaseURL", "base URL is required")
	}
	if e.Method == "" {
		return NewValidationError("Method", "HTTP method is required")
	}
	return nil
}

// ShouldRetry는 재시도 여부를 판단합니다.
func (e *APIEndpoint) ShouldRetry(attemptCount int) bool {
	return attemptCount < e.RetryCount
}
