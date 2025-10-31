package metrics

import (
	"demo-api-bridge/internal/core/port"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// prometheusMetrics는 Prometheus 기반 MetricsCollector 구현체입니다.
type prometheusMetrics struct {
	namespace string // 메트릭 네임스페이스

	// HTTP 요청 메트릭
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec

	// 외부 API 호출 메트릭
	externalAPICallsTotal   *prometheus.CounterVec
	externalAPICallDuration *prometheus.HistogramVec

	// 캐시 메트릭
	cacheHitsTotal   prometheus.Counter
	cacheMissesTotal prometheus.Counter

	// 라우팅 메트릭
	defaultRoutingUsedTotal *prometheus.CounterVec

	// 일반 메트릭
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

// New는 새로운 Prometheus MetricsCollector를 생성합니다.
func New(namespace string) port.MetricsCollector {
	m := &prometheusMetrics{
		namespace:  namespace,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}

	// HTTP 요청 메트릭
	m.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code"},
	)

	m.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// 외부 API 호출 메트릭
	m.externalAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "external_api_calls_total",
			Help:      "Total number of external API calls",
		},
		[]string{"endpoint", "success"},
	)

	m.externalAPICallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "external_api_call_duration_seconds",
			Help:      "External API call duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// 캐시 메트릭
	m.cacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_hits_total",
			Help:      "Total number of cache hits",
		},
	)

	m.cacheMissesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_misses_total",
			Help:      "Total number of cache misses",
		},
	)

	// 라우팅 메트릭
	m.defaultRoutingUsedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "default_routing_used_total",
			Help:      "Total number of times default routing was used (no matching rule found)",
		},
		[]string{"method", "path"},
	)

	return m
}

// RecordRequest는 요청 메트릭을 기록합니다.
func (m *prometheusMetrics) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	m.httpRequestsTotal.WithLabelValues(
		method,
		path,
		strconv.Itoa(statusCode),
	).Inc()

	m.httpRequestDuration.WithLabelValues(
		method,
		path,
	).Observe(duration.Seconds())
}

// RecordExternalAPICall은 외부 API 호출 메트릭을 기록합니다.
func (m *prometheusMetrics) RecordExternalAPICall(endpoint string, success bool, duration time.Duration) {
	successStr := "false"
	if success {
		successStr = "true"
	}

	m.externalAPICallsTotal.WithLabelValues(
		endpoint,
		successStr,
	).Inc()

	m.externalAPICallDuration.WithLabelValues(
		endpoint,
	).Observe(duration.Seconds())
}

// RecordCacheHit는 캐시 히트 메트릭을 기록합니다.
func (m *prometheusMetrics) RecordCacheHit(hit bool) {
	if hit {
		m.cacheHitsTotal.Inc()
	} else {
		m.cacheMissesTotal.Inc()
	}
}

// RecordDefaultRoutingUsed는 기본 라우팅 사용 메트릭을 기록합니다.
func (m *prometheusMetrics) RecordDefaultRoutingUsed(method, path string) {
	m.defaultRoutingUsedTotal.WithLabelValues(method, path).Inc()
}

// IncrementCounter는 카운터를 증가시킵니다.
func (m *prometheusMetrics) IncrementCounter(name string, labels map[string]string) {
	// namespace를 포함한 고유 이름 생성
	fullName := m.namespace + "_" + name

	counter, exists := m.counters[fullName]
	if !exists {
		labelNames := make([]string, 0, len(labels))
		for k := range labels {
			labelNames = append(labelNames, k)
		}

		counter = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: fullName,
				Help: name,
			},
			labelNames,
		)
		m.counters[fullName] = counter
	}

	labelValues := make([]string, 0, len(labels))
	for _, v := range labels {
		labelValues = append(labelValues, v)
	}

	counter.WithLabelValues(labelValues...).Inc()
}

// RecordGauge는 게이지 값을 기록합니다.
func (m *prometheusMetrics) RecordGauge(name string, value float64, labels map[string]string) {
	// namespace를 포함한 고유 이름 생성
	fullName := m.namespace + "_" + name

	gauge, exists := m.gauges[fullName]
	if !exists {
		labelNames := make([]string, 0, len(labels))
		for k := range labels {
			labelNames = append(labelNames, k)
		}

		gauge = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: fullName,
				Help: name,
			},
			labelNames,
		)
		m.gauges[fullName] = gauge
	}

	labelValues := make([]string, 0, len(labels))
	for _, v := range labels {
		labelValues = append(labelValues, v)
	}

	gauge.WithLabelValues(labelValues...).Set(value)
}

// RecordHistogram은 히스토그램 값을 기록합니다.
func (m *prometheusMetrics) RecordHistogram(name string, value float64, labels map[string]string) {
	// namespace를 포함한 고유 이름 생성
	fullName := m.namespace + "_" + name

	histogram, exists := m.histograms[fullName]
	if !exists {
		labelNames := make([]string, 0, len(labels))
		for k := range labels {
			labelNames = append(labelNames, k)
		}

		histogram = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    fullName,
				Help:    name,
				Buckets: prometheus.DefBuckets,
			},
			labelNames,
		)
		m.histograms[fullName] = histogram
	}

	labelValues := make([]string, 0, len(labels))
	for _, v := range labels {
		labelValues = append(labelValues, v)
	}

	histogram.WithLabelValues(labelValues...).Observe(value)
}

// NoOpMetrics는 메트릭을 수집하지 않는 구현체입니다 (테스트용).
type NoOpMetrics struct{}

// NewNoOp는 NoOp MetricsCollector를 생성합니다.
func NewNoOp() port.MetricsCollector {
	return &NoOpMetrics{}
}

func (m *NoOpMetrics) RecordRequest(method, path string, statusCode int, duration time.Duration)   {}
func (m *NoOpMetrics) RecordExternalAPICall(endpoint string, success bool, duration time.Duration) {}
func (m *NoOpMetrics) RecordCacheHit(hit bool)                                                     {}
func (m *NoOpMetrics) RecordDefaultRoutingUsed(method, path string)                                {}
func (m *NoOpMetrics) IncrementCounter(name string, labels map[string]string)                      {}
func (m *NoOpMetrics) RecordGauge(name string, value float64, labels map[string]string)            {}
func (m *NoOpMetrics) RecordHistogram(name string, value float64, labels map[string]string)        {}

// NewMetricsCollector는 새로운 메트릭 수집기를 생성합니다.
func NewMetricsCollector() port.MetricsCollector {
	return New("api_bridge")
}
