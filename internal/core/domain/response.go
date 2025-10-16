package domain

import (
	"encoding/json"
	"time"
)

// Response는 API Bridge를 통과하는 응답을 나타냅니다.
type Response struct {
	RequestID   string            // 원본 요청 ID (Trace ID)
	StatusCode  int               // HTTP 상태 코드
	Headers     map[string]string // 응답 헤더
	Body        []byte            // 응답 본문
	ContentType string            // 응답 콘텐츠 타입
	Timestamp   time.Time         // 응답 시간
	Duration    time.Duration     // 처리 시간
	Source      string            // 응답 소스 (예: external-api, cache, database)
	Error       error             // 에러 (있는 경우)
}

// NewResponse는 새로운 Response를 생성합니다.
func NewResponse(requestID string) *Response {
	return &Response{
		RequestID: requestID,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
	}
}

// SetSuccess는 성공 응답을 설정합니다.
func (r *Response) SetSuccess(statusCode int, body []byte) {
	r.StatusCode = statusCode
	r.Body = body
	r.Error = nil
}

// SetError는 에러 응답을 설정합니다.
func (r *Response) SetError(statusCode int, err error) {
	r.StatusCode = statusCode
	r.Error = err
}

// GetHeader는 헤더 값을 조회합니다.
func (r *Response) GetHeader(key string) (string, bool) {
	value, exists := r.Headers[key]
	return value, exists
}

// SetHeader는 헤더를 설정합니다.
func (r *Response) SetHeader(key, value string) {
	r.Headers[key] = value
}

// IsSuccess는 응답이 성공인지 확인합니다.
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300 && r.Error == nil
}

// IsFromCache는 응답이 캐시에서 온 것인지 확인합니다.
func (r *Response) IsFromCache() bool {
	return r.Source == "cache"
}

// SetDuration은 처리 시간을 설정합니다.
func (r *Response) SetDuration(start time.Time) {
	r.Duration = time.Since(start)
}

// ToJSON은 Response를 JSON으로 직렬화합니다.
func (r *Response) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ResponseFromJSON은 JSON에서 Response를 생성합니다.
func ResponseFromJSON(data []byte) (*Response, error) {
	var response Response
	err := json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
