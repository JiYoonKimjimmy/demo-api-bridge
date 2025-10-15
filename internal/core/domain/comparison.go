package domain

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

// ComparisonEngine은 JSON 응답 비교를 수행하는 엔진입니다.
type ComparisonEngine struct {
	config ComparisonConfig
}

// NewComparisonEngine은 새로운 비교 엔진을 생성합니다.
func NewComparisonEngine(config ComparisonConfig) *ComparisonEngine {
	return &ComparisonEngine{
		config: config,
	}
}

// CompareResponses는 두 응답을 비교하고 결과를 반환합니다.
func (e *ComparisonEngine) CompareResponses(legacyResponse, modernResponse *Response) *ComparisonResult {
	result := &ComparisonResult{
		MatchRate:     0.0,
		Differences:   []ResponseDiff{},
		TotalFields:   0,
		MatchedFields: 0,
	}

	if legacyResponse == nil || modernResponse == nil {
		result.Differences = append(result.Differences, ResponseDiff{
			Type:    MISSING,
			Path:    "response",
			Message: "One or both responses are nil",
		})
		return result
	}

	// JSON 파싱
	var legacyData, modernData interface{}
	var err error

	if len(legacyResponse.Body) > 0 {
		err = json.Unmarshal(legacyResponse.Body, &legacyData)
		if err != nil {
			result.Differences = append(result.Differences, ResponseDiff{
				Type:        TYPE_MISMATCH,
				Path:        "legacy_response.body",
				LegacyValue: "invalid JSON",
				ModernValue: legacyResponse.Body,
				Message:     "Legacy response is not valid JSON",
			})
		}
	}

	if len(modernResponse.Body) > 0 {
		err = json.Unmarshal(modernResponse.Body, &modernData)
		if err != nil {
			result.Differences = append(result.Differences, ResponseDiff{
				Type:        TYPE_MISMATCH,
				Path:        "modern_response.body",
				LegacyValue: modernResponse.Body,
				ModernValue: "invalid JSON",
				Message:     "Modern response is not valid JSON",
			})
		}
	}

	// JSON 비교 수행
	e.compareJSON(legacyData, modernData, "", result)

	// 일치율 계산
	if result.TotalFields > 0 {
		result.MatchRate = float64(result.MatchedFields) / float64(result.TotalFields)
	} else {
		result.MatchRate = 1.0 // 빈 응답인 경우
	}

	return result
}

// compareJSON는 JSON 객체를 재귀적으로 비교합니다.
func (e *ComparisonEngine) compareJSON(legacy, modern interface{}, path string, result *ComparisonResult) {
	// 무시할 필드 확인
	if e.shouldIgnoreField(path) {
		return
	}

	result.TotalFields++

	// 타입이 다른 경우
	if reflect.TypeOf(legacy) != reflect.TypeOf(modern) {
		result.Differences = append(result.Differences, ResponseDiff{
			Type:        TYPE_MISMATCH,
			Path:        path,
			LegacyValue: e.formatValue(legacy),
			ModernValue: e.formatValue(modern),
			Message:     "Type mismatch",
		})
		return
	}

	switch legacyVal := legacy.(type) {
	case map[string]interface{}:
		modernVal, ok := modern.(map[string]interface{})
		if !ok {
			result.Differences = append(result.Differences, ResponseDiff{
				Type:        TYPE_MISMATCH,
				Path:        path,
				LegacyValue: "object",
				ModernValue: e.formatValue(modern),
				Message:     "Type mismatch",
			})
			return
		}

		// 모든 키 비교
		allKeys := make(map[string]bool)
		for k := range legacyVal {
			allKeys[k] = true
		}
		for k := range modernVal {
			allKeys[k] = true
		}

		for key := range allKeys {
			newPath := e.buildPath(path, key)
			legacyValue, legacyExists := legacyVal[key]
			modernValue, modernExists := modernVal[key]

			if !legacyExists {
				result.Differences = append(result.Differences, ResponseDiff{
					Type:        MISSING,
					Path:        newPath,
					ModernValue: e.formatValue(modernValue),
					Message:     "Field missing in legacy response",
				})
			} else if !modernExists {
				result.Differences = append(result.Differences, ResponseDiff{
					Type:        EXTRA,
					Path:        newPath,
					LegacyValue: e.formatValue(legacyValue),
					Message:     "Extra field in legacy response",
				})
			} else {
				e.compareJSON(legacyValue, modernValue, newPath, result)
			}
		}

	case []interface{}:
		modernVal, ok := modern.([]interface{})
		if !ok {
			result.Differences = append(result.Differences, ResponseDiff{
				Type:        TYPE_MISMATCH,
				Path:        path,
				LegacyValue: "array",
				ModernValue: e.formatValue(modern),
				Message:     "Type mismatch",
			})
			return
		}

		// 배열 길이 비교
		if len(legacyVal) != len(modernVal) {
			result.Differences = append(result.Differences, ResponseDiff{
				Type:        VALUE_MISMATCH,
				Path:        path,
				LegacyValue: len(legacyVal),
				ModernValue: len(modernVal),
				Message:     "Array length mismatch",
			})
		}

		// 배열 요소 비교 (최대 10개까지만)
		maxLen := len(legacyVal)
		if len(modernVal) > maxLen {
			maxLen = len(modernVal)
		}
		if maxLen > 10 {
			maxLen = 10 // 성능을 위해 제한
		}

		for i := 0; i < maxLen; i++ {
			newPath := e.buildPath(path, "["+strconv.Itoa(i)+"]")
			if i < len(legacyVal) && i < len(modernVal) {
				e.compareJSON(legacyVal[i], modernVal[i], newPath, result)
			} else if i < len(legacyVal) {
				result.Differences = append(result.Differences, ResponseDiff{
					Type:        MISSING,
					Path:        newPath,
					LegacyValue: e.formatValue(legacyVal[i]),
					Message:     "Array element missing in modern response",
				})
			} else {
				result.Differences = append(result.Differences, ResponseDiff{
					Type:        EXTRA,
					Path:        newPath,
					ModernValue: e.formatValue(modernVal[i]),
					Message:     "Extra array element in modern response",
				})
			}
		}

	default:
		// 기본값 비교
		if !e.valuesEqual(legacyVal, modern.(interface{})) {
			result.Differences = append(result.Differences, ResponseDiff{
				Type:        VALUE_MISMATCH,
				Path:        path,
				LegacyValue: e.formatValue(legacyVal),
				ModernValue: e.formatValue(modern.(interface{})),
				Message:     "Value mismatch",
			})
			return
		}
	}

	// 차이점이 없으면 매치된 필드로 카운트
	if len(result.Differences) == 0 || !e.hasDifferenceAtPath(result.Differences, path) {
		result.MatchedFields++
	}
}

// shouldIgnoreField는 필드를 무시해야 하는지 확인합니다.
func (e *ComparisonEngine) shouldIgnoreField(path string) bool {
	for _, ignoreField := range e.config.IgnoreFields {
		if strings.Contains(path, ignoreField) {
			return true
		}
	}
	return false
}

// valuesEqual는 두 값을 비교합니다 (허용 오차 고려).
func (e *ComparisonEngine) valuesEqual(legacy, modern interface{}) bool {
	// 문자열 비교
	if legacyStr, ok := legacy.(string); ok {
		if modernStr, ok := modern.(string); ok {
			return legacyStr == modernStr
		}
		return false
	}

	// 숫자 비교 (허용 오차 고려)
	if legacyNum, ok := e.toFloat64(legacy); ok {
		if modernNum, ok := e.toFloat64(modern); ok {
			diff := legacyNum - modernNum
			if diff < 0 {
				diff = -diff
			}
			return diff <= e.config.AllowableDifference
		}
		return false
	}

	// 불린 비교
	if legacyBool, ok := legacy.(bool); ok {
		if modernBool, ok := modern.(bool); ok {
			return legacyBool == modernBool
		}
		return false
	}

	// nil 비교
	if legacy == nil && modern == nil {
		return true
	}

	// 기본 비교
	return reflect.DeepEqual(legacy, modern)
}

// toFloat64는 값을 float64로 변환합니다.
func (e *ComparisonEngine) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// formatValue는 값을 문자열로 포맷팅합니다.
func (e *ComparisonEngine) formatValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// JSON으로 직렬화하여 포맷팅
	if jsonBytes, err := json.Marshal(value); err == nil {
		var result interface{}
		if err := json.Unmarshal(jsonBytes, &result); err == nil {
			return result
		}
	}

	return value
}

// buildPath는 경로를 구성합니다.
func (e *ComparisonEngine) buildPath(basePath, key string) string {
	if basePath == "" {
		return key
	}
	return basePath + "." + key
}

// hasDifferenceAtPath는 특정 경로에 차이점이 있는지 확인합니다.
func (e *ComparisonEngine) hasDifferenceAtPath(differences []ResponseDiff, path string) bool {
	for _, diff := range differences {
		if diff.Path == path {
			return true
		}
	}
	return false
}

// ComparisonResult는 비교 결과를 나타냅니다.
type ComparisonResult struct {
	MatchRate     float64        `json:"match_rate"`
	Differences   []ResponseDiff `json:"differences"`
	TotalFields   int            `json:"total_fields"`
	MatchedFields int            `json:"matched_fields"`
}

// IsSuccessful은 비교가 성공적인지 확인합니다.
func (r *ComparisonResult) IsSuccessful() bool {
	return r.MatchRate >= 0.95 // 95% 이상 일치 시 성공
}
