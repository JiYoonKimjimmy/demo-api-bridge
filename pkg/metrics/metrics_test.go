package metrics

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	metrics := New("test_service")
	if metrics == nil {
		t.Error("New() returned nil")
	}
}

func TestPrometheusMetrics_RecordRequest(t *testing.T) {
	metrics := New("test_service")

	// 요청 메트릭 기록
	metrics.RecordRequest("GET", "/api/users", 200, 100*time.Millisecond)
	metrics.RecordRequest("POST", "/api/users", 201, 150*time.Millisecond)
	metrics.RecordRequest("GET", "/api/users", 404, 50*time.Millisecond)

	// 에러가 발생하지 않으면 성공
}

func TestPrometheusMetrics_RecordExternalAPICall(t *testing.T) {
	metrics := New("test_service")

	// 외부 API 호출 메트릭 기록
	metrics.RecordExternalAPICall("https://api.example.com", true, 200*time.Millisecond)
	metrics.RecordExternalAPICall("https://api.example.com", false, 5*time.Second)

	// 에러가 발생하지 않으면 성공
}

func TestPrometheusMetrics_RecordCacheHit(t *testing.T) {
	metrics := New("test_service")

	// 캐시 메트릭 기록
	metrics.RecordCacheHit(true)  // hit
	metrics.RecordCacheHit(false) // miss
	metrics.RecordCacheHit(true)  // hit

	// 에러가 발생하지 않으면 성공
}

func TestPrometheusMetrics_IncrementCounter(t *testing.T) {
	metrics := New("test_service")

	labels := map[string]string{
		"endpoint": "api1",
		"method":   "GET",
	}

	metrics.IncrementCounter("custom_counter", labels)
	metrics.IncrementCounter("custom_counter", labels)

	// 에러가 발생하지 않으면 성공
}

func TestPrometheusMetrics_RecordGauge(t *testing.T) {
	metrics := New("test_service")

	labels := map[string]string{
		"type": "active_connections",
	}

	metrics.RecordGauge("connection_count", 10, labels)
	metrics.RecordGauge("connection_count", 15, labels)

	// 에러가 발생하지 않으면 성공
}

func TestPrometheusMetrics_RecordHistogram(t *testing.T) {
	metrics := New("test_service")

	labels := map[string]string{
		"operation": "database_query",
	}

	metrics.RecordHistogram("query_duration", 0.5, labels)
	metrics.RecordHistogram("query_duration", 1.2, labels)

	// 에러가 발생하지 않으면 성공
}

func TestNoOpMetrics(t *testing.T) {
	metrics := NewNoOp()
	if metrics == nil {
		t.Error("NewNoOp() returned nil")
	}

	// 모든 메서드가 에러 없이 실행되어야 함
	metrics.RecordRequest("GET", "/test", 200, 100*time.Millisecond)
	metrics.RecordExternalAPICall("http://example.com", true, 200*time.Millisecond)
	metrics.RecordCacheHit(true)
	metrics.IncrementCounter("test", map[string]string{})
	metrics.RecordGauge("test", 1.0, map[string]string{})
	metrics.RecordHistogram("test", 1.0, map[string]string{})
}

func BenchmarkMetrics_RecordRequest(b *testing.B) {
	metrics := New("bench_service")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordRequest("GET", "/api/test", 200, 100*time.Millisecond)
	}
}
