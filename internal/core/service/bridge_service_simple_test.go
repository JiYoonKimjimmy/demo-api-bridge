package service

import (
	"demo-api-bridge/internal/core/domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBridgeService_Simple(t *testing.T) {
	// 간단한 테스트로 시작
	request := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	assert.Equal(t, "test-request-id", request.ID)
	assert.Equal(t, "GET", request.Method)
	assert.Equal(t, "/api/users", request.Path)
}

func TestRequest_Validation(t *testing.T) {
	// 유효한 요청
	validRequest := &domain.Request{
		ID:     "test-request-id",
		Method: "GET",
		Path:   "/api/users",
	}

	err := validRequest.IsValid()
	assert.NoError(t, err)

	// 유효하지 않은 요청 (ID가 비어있음)
	invalidRequest := &domain.Request{
		ID:     "", // Invalid: empty ID
		Method: "GET",
		Path:   "/api/users",
	}

	err = invalidRequest.IsValid()
	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidRequestID, err)
}
