package domain

import (
	"regexp"
)

// RoutingRule은 요청을 적절한 엔드포인트로 라우팅하는 규칙을 나타냅니다.
type RoutingRule struct {
	ID            string         // 규칙 고유 ID
	Name          string         // 규칙 이름
	PathPattern   string         // 경로 패턴 (예: /api/v1/users/*)
	MethodPattern string         // HTTP 메서드 패턴 (예: GET, POST, *)
	EndpointID    string         // 대상 엔드포인트 ID
	Priority      int            // 우선순위 (낮을수록 먼저 매칭)
	IsActive      bool           // 활성화 여부
	CacheEnabled  bool           // 캐시 사용 여부
	CacheTTL      int            // 캐시 TTL (초)
	Description   string         // 설명
	compiledRegex *regexp.Regexp // 컴파일된 정규식 (private)
}

// NewRoutingRule은 새로운 RoutingRule을 생성합니다.
func NewRoutingRule(id, name, pathPattern, methodPattern, endpointID string) *RoutingRule {
	return &RoutingRule{
		ID:            id,
		Name:          name,
		PathPattern:   pathPattern,
		MethodPattern: methodPattern,
		EndpointID:    endpointID,
		Priority:      100,
		IsActive:      true,
		CacheEnabled:  false,
		CacheTTL:      300, // 기본 5분
	}
}

// Matches는 요청이 이 라우팅 규칙에 매칭되는지 확인합니다.
func (r *RoutingRule) Matches(request *Request) (bool, error) {
	if !r.IsActive {
		return false, nil
	}

	// 메서드 매칭
	if r.MethodPattern != "*" && r.MethodPattern != request.Method {
		return false, nil
	}

	// 경로 패턴 매칭
	if r.compiledRegex == nil {
		pattern := convertPatternToRegex(r.PathPattern)
		var err error
		r.compiledRegex, err = regexp.Compile(pattern)
		if err != nil {
			return false, NewDomainError("REGEX_COMPILE_ERROR", "failed to compile path pattern", err)
		}
	}

	return r.compiledRegex.MatchString(request.Path), nil
}

// IsValid는 라우팅 규칙이 유효한지 검증합니다.
func (r *RoutingRule) IsValid() error {
	if r.ID == "" {
		return NewValidationError("ID", "routing rule ID is required")
	}
	if r.PathPattern == "" {
		return NewValidationError("PathPattern", "path pattern is required")
	}
	if r.EndpointID == "" {
		return NewValidationError("EndpointID", "endpoint ID is required")
	}
	return nil
}

// convertPatternToRegex는 간단한 패턴을 정규식으로 변환합니다.
// 예: /api/v1/users/* -> ^/api/v1/users/.*$
func convertPatternToRegex(pattern string) string {
	// 특수문자 이스케이프
	regex := regexp.QuoteMeta(pattern)
	// * 를 .* 로 변환
	regex = regexp.MustCompile(`\\\*`).ReplaceAllString(regex, ".*")
	// 시작과 끝 앵커 추가
	return "^" + regex + "$"
}
