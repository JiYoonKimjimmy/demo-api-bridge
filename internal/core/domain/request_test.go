package domain

import (
	"testing"
)

func TestNewRequest(t *testing.T) {
	id := "trace-123"
	method := "GET"
	path := "/api/v1/users"

	req := NewRequest(id, method, path)

	if req.ID != id {
		t.Errorf("expected ID %s, got %s", id, req.ID)
	}
	if req.Method != method {
		t.Errorf("expected Method %s, got %s", method, req.Method)
	}
	if req.Path != path {
		t.Errorf("expected Path %s, got %s", path, req.Path)
	}
	if req.Headers == nil {
		t.Error("Headers should be initialized")
	}
	if req.QueryParams == nil {
		t.Error("QueryParams should be initialized")
	}
	if req.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestRequest_GetSetHeader(t *testing.T) {
	req := NewRequest("test-id", "GET", "/test")

	// 헤더 설정
	req.SetHeader("Content-Type", "application/json")

	// 헤더 조회
	value, exists := req.GetHeader("Content-Type")
	if !exists {
		t.Error("Header should exist")
	}
	if value != "application/json" {
		t.Errorf("expected 'application/json', got '%s'", value)
	}

	// 존재하지 않는 헤더 조회
	_, exists = req.GetHeader("NonExistent")
	if exists {
		t.Error("NonExistent header should not exist")
	}
}

func TestRequest_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		request *Request
		wantErr bool
	}{
		{
			name:    "valid request",
			request: NewRequest("id-1", "GET", "/api/users"),
			wantErr: false,
		},
		{
			name: "missing ID",
			request: &Request{
				Method: "GET",
				Path:   "/api/users",
			},
			wantErr: true,
		},
		{
			name: "missing Method",
			request: &Request{
				ID:   "id-1",
				Path: "/api/users",
			},
			wantErr: true,
		},
		{
			name: "missing Path",
			request: &Request{
				ID:     "id-1",
				Method: "GET",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.IsValid()
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequest_QueryParams(t *testing.T) {
	req := NewRequest("test-id", "GET", "/test")

	// 쿼리 파라미터 설정
	req.SetQueryParam("page", "1")
	req.SetQueryParam("limit", "10")

	// 쿼리 파라미터 조회
	page, exists := req.GetQueryParam("page")
	if !exists || page != "1" {
		t.Error("page query param should be '1'")
	}

	limit, exists := req.GetQueryParam("limit")
	if !exists || limit != "10" {
		t.Error("limit query param should be '10'")
	}
}

func BenchmarkNewRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRequest("id-"+string(rune(i)), "GET", "/api/test")
	}
}
