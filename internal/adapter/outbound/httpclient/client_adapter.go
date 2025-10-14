package httpclient

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpClientAdapter는 HTTP 기반 ExternalAPIClient 구현체입니다.
type httpClientAdapter struct {
	client  *http.Client
	timeout time.Duration
}

// NewHTTPClientAdapter는 새로운 HTTP 클라이언트 어댑터를 생성합니다.
func NewHTTPClientAdapter(timeout time.Duration) port.ExternalAPIClient {
	return &httpClientAdapter{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		timeout: timeout,
	}
}

// NewHTTPClientAdapterWithClient는 기존 HTTP 클라이언트로 어댑터를 생성합니다.
func NewHTTPClientAdapterWithClient(client *http.Client) port.ExternalAPIClient {
	return &httpClientAdapter{
		client:  client,
		timeout: client.Timeout,
	}
}

// SendRequest는 외부 API에 요청을 전송하고 응답을 받습니다.
func (h *httpClientAdapter) SendRequest(ctx context.Context, endpoint *domain.APIEndpoint, request *domain.Request) (*domain.Response, error) {
	start := time.Now()

	// URL 구성
	url := h.buildURL(endpoint, request)

	// HTTP 요청 생성
	httpReq, err := h.buildHTTPRequest(ctx, request, url)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %w", err)
	}

	// 요청 전송
	httpResp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// 응답 본문 읽기
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 응답 생성
	response := domain.NewResponse(request.ID)
	response.StatusCode = httpResp.StatusCode
	response.Body = body
	response.SetDuration(start)
	response.Source = "external-api"

	// 응답 헤더 복사
	for key, values := range httpResp.Header {
		if len(values) > 0 {
			response.SetHeader(key, values[0])
		}
	}

	return response, nil
}

// SendWithRetry는 재시도 로직을 포함하여 외부 API에 요청을 전송합니다.
func (h *httpClientAdapter) SendWithRetry(ctx context.Context, endpoint *domain.APIEndpoint, request *domain.Request) (*domain.Response, error) {
	var lastErr error
	attempt := 0

	for attempt <= endpoint.RetryCount {
		if attempt > 0 {
			// 재시도 전 대기
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		response, err := h.SendRequest(ctx, endpoint, request)
		if err == nil {
			return response, nil
		}

		lastErr = err
		attempt++

		// 재시도 가능한 에러인지 확인
		if !h.isRetryableError(err) {
			break
		}

		// 최대 재시도 횟수 확인
		if !endpoint.ShouldRetry(attempt) {
			break
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", attempt, lastErr)
}

// buildURL은 엔드포인트와 요청으로부터 URL을 구성합니다.
func (h *httpClientAdapter) buildURL(endpoint *domain.APIEndpoint, request *domain.Request) string {
	baseURL := endpoint.GetFullURL()

	// 요청 경로가 있으면 추가
	if request.Path != "" && request.Path != "/" {
		baseURL = strings.TrimSuffix(baseURL, "/") + request.Path
	}

	// 쿼리 파라미터 추가
	if len(request.QueryParams) > 0 {
		baseURL += "?"
		params := make([]string, 0, len(request.QueryParams))
		for key, value := range request.QueryParams {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
		baseURL += strings.Join(params, "&")
	}

	return baseURL
}

// buildHTTPRequest는 HTTP 요청을 생성합니다.
func (h *httpClientAdapter) buildHTTPRequest(ctx context.Context, request *domain.Request, url string) (*http.Request, error) {
	var body io.Reader
	if len(request.Body) > 0 {
		body = strings.NewReader(string(request.Body))
	}

	httpReq, err := http.NewRequestWithContext(ctx, request.Method, url, body)
	if err != nil {
		return nil, err
	}

	// 요청 헤더 복사
	for key, value := range request.Headers {
		httpReq.Header.Set(key, value)
	}

	return httpReq, nil
}

// isRetryableError는 재시도 가능한 에러인지 확인합니다.
func (h *httpClientAdapter) isRetryableError(err error) bool {
	// 네트워크 타임아웃, 연결 실패 등은 재시도 가능
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connection reset")
}

// Close는 HTTP 클라이언트를 종료합니다.
func (h *httpClientAdapter) Close() error {
	if transport, ok := h.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}
