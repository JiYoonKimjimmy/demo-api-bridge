package domain

import (
	"time"
)

// Request는 API Bridge를 통과하는 요청을 나타냅니다.
type Request struct {
	ID          string            // 요청 고유 ID (Trace ID)
	Method      string            // HTTP 메서드 (GET, POST, PUT, DELETE 등)
	Path        string            // 요청 경로
	Headers     map[string]string // 요청 헤더
	QueryParams map[string]string // 쿼리 파라미터
	Body        []byte            // 요청 본문
	Timestamp   time.Time         // 요청 시간
	ClientIP    string            // 클라이언트 IP
}

// NewRequest는 새로운 Request를 생성합니다.
func NewRequest(id, method, path string) *Request {
	return &Request{
		ID:          id,
		Method:      method,
		Path:        path,
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
		Timestamp:   time.Now(),
	}
}

// GetHeader는 헤더 값을 조회합니다.
func (r *Request) GetHeader(key string) (string, bool) {
	value, exists := r.Headers[key]
	return value, exists
}

// SetHeader는 헤더를 설정합니다.
func (r *Request) SetHeader(key, value string) {
	r.Headers[key] = value
}

// GetQueryParam은 쿼리 파라미터 값을 조회합니다.
func (r *Request) GetQueryParam(key string) (string, bool) {
	value, exists := r.QueryParams[key]
	return value, exists
}

// SetQueryParam은 쿼리 파라미터를 설정합니다.
func (r *Request) SetQueryParam(key, value string) {
	r.QueryParams[key] = value
}

// IsValid는 요청이 유효한지 검증합니다.
func (r *Request) IsValid() error {
	if r.ID == "" {
		return ErrInvalidRequestID
	}
	if r.Method == "" {
		return ErrInvalidMethod
	}
	if r.Path == "" {
		return ErrInvalidPath
	}
	return nil
}
