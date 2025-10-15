package database

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"sort"
	"sync"
	"time"
)

// mockComparisonRepository는 ComparisonRepository의 Mock 구현체입니다.
type mockComparisonRepository struct {
	comparisons map[string][]*domain.APIComparison
	mutex       sync.RWMutex
}

// NewMockComparisonRepository는 새로운 Mock ComparisonRepository를 생성합니다.
func NewMockComparisonRepository() port.ComparisonRepository {
	return &mockComparisonRepository{
		comparisons: make(map[string][]*domain.APIComparison),
	}
}

// SaveComparison은 API 비교 결과를 저장합니다.
func (r *mockComparisonRepository) SaveComparison(ctx context.Context, comparison *domain.APIComparison) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 라우팅 규칙 ID를 키로 사용 (실제로는 비교 객체에서 추출해야 함)
	// 임시로 request ID의 일부를 사용
	key := r.extractRoutingRuleID(comparison.RequestID)

	// 복사본 생성 (원본 보호)
	comparisonCopy := *comparison
	if comparison.LegacyResponse != nil {
		legacyCopy := *comparison.LegacyResponse
		comparisonCopy.LegacyResponse = &legacyCopy
	}
	if comparison.ModernResponse != nil {
		modernCopy := *comparison.ModernResponse
		comparisonCopy.ModernResponse = &modernCopy
	}

	r.comparisons[key] = append(r.comparisons[key], &comparisonCopy)

	// 최대 저장 개수 제한 (메모리 관리)
	maxComparisons := 1000
	if len(r.comparisons[key]) > maxComparisons {
		// 오래된 것부터 삭제
		r.comparisons[key] = r.comparisons[key][len(r.comparisons[key])-maxComparisons:]
	}

	return nil
}

// GetRecentComparisons는 최근 비교 결과들을 조회합니다.
func (r *mockComparisonRepository) GetRecentComparisons(ctx context.Context, routingRuleID string, limit int) ([]*domain.APIComparison, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	comparisons, exists := r.comparisons[routingRuleID]
	if !exists || len(comparisons) == 0 {
		return []*domain.APIComparison{}, nil
	}

	// 시간순 정렬 (최신이 뒤에)
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].Timestamp.Before(comparisons[j].Timestamp)
	})

	// 최근 limit개 반환
	start := len(comparisons) - limit
	if start < 0 {
		start = 0
	}

	recentComparisons := comparisons[start:]
	result := make([]*domain.APIComparison, len(recentComparisons))
	for i, comp := range recentComparisons {
		// 복사본 생성 (원본 보호)
		compCopy := *comp
		if comp.LegacyResponse != nil {
			legacyCopy := *comp.LegacyResponse
			compCopy.LegacyResponse = &legacyCopy
		}
		if comp.ModernResponse != nil {
			modernCopy := *comp.ModernResponse
			compCopy.ModernResponse = &modernCopy
		}
		result[i] = &compCopy
	}

	return result, nil
}

// GetComparisonStatistics는 비교 통계를 조회합니다.
func (r *mockComparisonRepository) GetComparisonStatistics(ctx context.Context, routingRuleID string, from, to time.Time) (*port.ComparisonStatistics, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	comparisons, exists := r.comparisons[routingRuleID]
	if !exists || len(comparisons) == 0 {
		return &port.ComparisonStatistics{
			RoutingRuleID:     routingRuleID,
			TotalComparisons:  0,
			SuccessfulMatches: 0,
			AverageMatchRate:  0.0,
			LastComparison:    time.Time{},
		}, nil
	}

	// 기간 필터링
	var filteredComparisons []*domain.APIComparison
	for _, comp := range comparisons {
		if comp.Timestamp.After(from) && comp.Timestamp.Before(to) {
			filteredComparisons = append(filteredComparisons, comp)
		}
	}

	if len(filteredComparisons) == 0 {
		return &port.ComparisonStatistics{
			RoutingRuleID:     routingRuleID,
			TotalComparisons:  0,
			SuccessfulMatches: 0,
			AverageMatchRate:  0.0,
			LastComparison:    time.Time{},
		}, nil
	}

	// 통계 계산
	totalComparisons := len(filteredComparisons)
	successfulMatches := 0
	var totalMatchRate float64

	var lastComparison time.Time
	for _, comp := range filteredComparisons {
		if comp.IsSuccessful() {
			successfulMatches++
		}
		totalMatchRate += comp.MatchRate

		if comp.Timestamp.After(lastComparison) {
			lastComparison = comp.Timestamp
		}
	}

	averageMatchRate := totalMatchRate / float64(totalComparisons)

	return &port.ComparisonStatistics{
		RoutingRuleID:     routingRuleID,
		TotalComparisons:  totalComparisons,
		SuccessfulMatches: successfulMatches,
		AverageMatchRate:  averageMatchRate,
		LastComparison:    lastComparison,
	}, nil
}

// extractRoutingRuleID는 요청 ID에서 라우팅 규칙 ID를 추출합니다.
// 실제 구현에서는 더 정교한 로직이 필요합니다.
func (r *mockComparisonRepository) extractRoutingRuleID(requestID string) string {
	// 임시 구현: request ID의 일부를 사용
	// 실제로는 요청과 연결된 라우팅 규칙 ID를 저장해야 함
	if len(requestID) > 10 {
		return "route-" + requestID[:5]
	}
	return "route-default"
}
