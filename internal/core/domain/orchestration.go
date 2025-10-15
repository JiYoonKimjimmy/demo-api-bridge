package domain

import (
	"time"
)

// APIMode는 API 호출 모드를 나타냅니다.
type APIMode string

const (
	// LEGACY_ONLY: 레거시 API만 호출
	LEGACY_ONLY APIMode = "LEGACY_ONLY"
	// MODERN_ONLY: 모던 API만 호출
	MODERN_ONLY APIMode = "MODERN_ONLY"
	// PARALLEL: 레거시와 모던 API를 병렬로 호출
	PARALLEL APIMode = "PARALLEL"
)

// APIComparison는 API 응답 비교 결과를 나타냅니다.
type APIComparison struct {
	RequestID          string         // 요청 ID
	LegacyResponse     *Response      // 레거시 API 응답
	ModernResponse     *Response      // 모던 API 응답
	MatchRate          float64        // 일치율 (0.0 ~ 1.0)
	Differences        []ResponseDiff // 차이점 목록
	ComparisonDuration time.Duration  // 비교 소요 시간
	Timestamp          time.Time      // 비교 시점
}

// ResponseDiff는 응답 차이점을 나타냅니다.
type ResponseDiff struct {
	Type        DiffType `json:"type"`         // 차이점 유형
	Path        string   `json:"path"`         // 차이점 경로
	LegacyValue any      `json:"legacy_value"` // 레거시 값
	ModernValue any      `json:"modern_value"` // 모던 값
	Message     string   `json:"message"`      // 설명
}

// DiffType은 응답 차이점 유형을 나타냅니다.
type DiffType string

const (
	MISSING        DiffType = "MISSING"        // 레거시에만 있는 필드
	EXTRA          DiffType = "EXTRA"          // 모던에만 있는 필드
	VALUE_MISMATCH DiffType = "VALUE_MISMATCH" // 값이 다른 경우
	TYPE_MISMATCH  DiffType = "TYPE_MISMATCH"  // 타입이 다른 경우
)

// OrchestrationRule은 오케스트레이션 규칙을 나타냅니다.
type OrchestrationRule struct {
	ID               string           // 규칙 고유 ID
	Name             string           // 규칙 이름
	RoutingRuleID    string           // 연결된 라우팅 규칙 ID
	LegacyEndpointID string           // 레거시 엔드포인트 ID
	ModernEndpointID string           // 모던 엔드포인트 ID
	CurrentMode      APIMode          // 현재 API 모드
	TransitionConfig TransitionConfig // 전환 설정
	ComparisonConfig ComparisonConfig // 비교 설정
	IsActive         bool             // 활성화 여부
	Description      string           // 설명
}

// TransitionConfig는 전환 설정을 나타냅니다.
type TransitionConfig struct {
	AutoTransitionEnabled    bool          // 자동 전환 활성화
	MatchRateThreshold       float64       // 전환 임계값 (0.0 ~ 1.0)
	StabilityPeriod          time.Duration // 안정성 확인 기간
	MinRequestsForTransition int           // 전환을 위한 최소 요청 수
	RollbackThreshold        float64       // 롤백 임계값
}

// ComparisonConfig는 비교 설정을 나타냅니다.
type ComparisonConfig struct {
	Enabled               bool     // 비교 활성화
	IgnoreFields          []string // 무시할 필드 목록
	AllowableDifference   float64  // 허용 가능한 차이 (숫자 필드용)
	StrictMode            bool     // 엄격 모드 (모든 차이점을 에러로 처리)
	SaveComparisonHistory bool     // 비교 이력 저장 여부
}

// NewOrchestrationRule은 새로운 OrchestrationRule을 생성합니다.
func NewOrchestrationRule(id, name, routingRuleID, legacyEndpointID, modernEndpointID string) *OrchestrationRule {
	return &OrchestrationRule{
		ID:               id,
		Name:             name,
		RoutingRuleID:    routingRuleID,
		LegacyEndpointID: legacyEndpointID,
		ModernEndpointID: modernEndpointID,
		CurrentMode:      PARALLEL,
		TransitionConfig: TransitionConfig{
			AutoTransitionEnabled:    true,
			MatchRateThreshold:       0.95, // 95% 일치 시 전환
			StabilityPeriod:          24 * time.Hour,
			MinRequestsForTransition: 100,
			RollbackThreshold:        0.90, // 90% 미만 시 롤백
		},
		ComparisonConfig: ComparisonConfig{
			Enabled:               true,
			IgnoreFields:          []string{"timestamp", "requestId"},
			AllowableDifference:   0.01, // 1% 허용 오차
			StrictMode:            false,
			SaveComparisonHistory: true,
		},
		IsActive: true,
	}
}

// CanTransitionToModern는 모던 API로 전환 가능한지 확인합니다.
func (o *OrchestrationRule) CanTransitionToModern(recentMatchRate float64, requestCount int) bool {
	if !o.TransitionConfig.AutoTransitionEnabled {
		return false
	}

	if o.CurrentMode != PARALLEL {
		return false
	}

	if requestCount < o.TransitionConfig.MinRequestsForTransition {
		return false
	}

	return recentMatchRate >= o.TransitionConfig.MatchRateThreshold
}

// ShouldRollback는 롤백이 필요한지 확인합니다.
func (o *OrchestrationRule) ShouldRollback(recentMatchRate float64) bool {
	if o.CurrentMode != MODERN_ONLY {
		return false
	}

	return recentMatchRate < o.TransitionConfig.RollbackThreshold
}

// IsValid는 오케스트레이션 규칙이 유효한지 검증합니다.
func (o *OrchestrationRule) IsValid() error {
	if o.ID == "" {
		return NewValidationError("ID", "orchestration rule ID is required")
	}
	if o.RoutingRuleID == "" {
		return NewValidationError("RoutingRuleID", "routing rule ID is required")
	}
	if o.LegacyEndpointID == "" {
		return NewValidationError("LegacyEndpointID", "legacy endpoint ID is required")
	}
	if o.ModernEndpointID == "" {
		return NewValidationError("ModernEndpointID", "modern endpoint ID is required")
	}
	if o.TransitionConfig.MatchRateThreshold < 0.0 || o.TransitionConfig.MatchRateThreshold > 1.0 {
		return NewValidationError("MatchRateThreshold", "match rate threshold must be between 0.0 and 1.0")
	}
	return nil
}

// NewAPIComparison은 새로운 APIComparison을 생성합니다.
func NewAPIComparison(requestID string, legacyResponse, modernResponse *Response) *APIComparison {
	return &APIComparison{
		RequestID:      requestID,
		LegacyResponse: legacyResponse,
		ModernResponse: modernResponse,
		Timestamp:      time.Now(),
	}
}

// CalculateMatchRate는 일치율을 계산합니다.
func (c *APIComparison) CalculateMatchRate() float64 {
	if c.LegacyResponse == nil || c.ModernResponse == nil {
		return 0.0
	}

	// 기본적인 일치율 계산 (간단한 버전)
	// TODO: 실제 JSON 비교 로직으로 대체
	if c.LegacyResponse.StatusCode == c.ModernResponse.StatusCode {
		return 1.0
	}

	return 0.0
}

// IsSuccessful은 비교가 성공적인지 확인합니다.
func (c *APIComparison) IsSuccessful() bool {
	return c.MatchRate >= 0.95 // 95% 이상 일치 시 성공
}
