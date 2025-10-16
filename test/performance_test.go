package test

import (
	"context"
	"demo-api-bridge/internal/adapter/outbound/cache"
	"demo-api-bridge/internal/adapter/outbound/database"
	"demo-api-bridge/internal/adapter/outbound/httpclient"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/service"
	"demo-api-bridge/pkg/logger"
	"demo-api-bridge/pkg/metrics"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mock HTTP 서버 생성
func createMockHTTPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 간단한 JSON 응답 반환
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id": 1, "name": "Test User", "email": "test@example.com"}`)
	}))
}

// 성능 테스트용 서비스 설정
func setupPerformanceTest() (*service.BridgeService, func()) {
	// Mock 서버 생성
	mockServer := createMockHTTPServer()

	// 의존성 설정
	log := logger.NewDefault()
	metricsCollector := metrics.New("performance-test")

	// Mock Repository들
	routingRepo := database.NewMockRoutingRepository()
	endpointRepo := database.NewMockEndpointRepository()
	orchestrationRepo := database.NewMockOrchestrationRepository()
	comparisonRepo := database.NewMockComparisonRepository()

	// Mock Cache
	cacheRepo := cache.NewMockCacheRepository()

	// HTTP Client
	httpClient := httpclient.NewHTTPClientAdapter(30 * time.Second)

	// Orchestration Service
	orchestrationSvc := service.NewOrchestrationService(
		orchestrationRepo,
		comparisonRepo,
		httpClient,
		log,
		metricsCollector,
	)

	// Bridge Service
	bridgeSvc := service.NewBridgeService(
		routingRepo,
		endpointRepo,
		orchestrationRepo,
		comparisonRepo,
		orchestrationSvc,
		httpClient,
		cacheRepo,
		log,
		metricsCollector,
	)

	// 초기 데이터 설정
	setupTestData(routingRepo, endpointRepo, mockServer.URL)

	cleanup := func() {
		mockServer.Close()
	}

	return bridgeSvc, cleanup
}

// 테스트 데이터 설정
func setupTestData(routingRepo *database.MockRoutingRepository, endpointRepo *database.MockEndpointRepository, serverURL string) {
	ctx := context.Background()

	// 엔드포인트 생성
	endpoint := domain.NewAPIEndpoint("test-endpoint", "Test API", serverURL, "/users", "GET")
	endpoint.IsActive = true
	endpoint.Timeout = 5 * time.Second

	// 라우팅 규칙 생성
	rule := domain.NewRoutingRule("test-rule", "Test Rule", "/api/users", "GET", "test-endpoint")
	rule.CacheEnabled = false
	rule.Priority = 1

	// Repository에 저장
	endpointRepo.Create(ctx, endpoint)
	routingRepo.Create(ctx, rule)
}

// 단일 요청 처리 벤치마크
func BenchmarkSingleRequest(b *testing.B) {
	bridgeSvc, cleanup := setupPerformanceTest()
	defer cleanup()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			request := domain.NewRequest("benchmark-request", "GET", "/api/users")
			_, err := bridgeSvc.ProcessRequest(ctx, request)
			if err != nil {
				b.Errorf("Request failed: %v", err)
			}
		}
	})
}

// 캐시 활성화된 요청 벤치마크
func BenchmarkCachedRequest(b *testing.B) {
	bridgeSvc, cleanup := setupPerformanceTest()
	defer cleanup()

	ctx := context.Background()

	// 첫 번째 요청으로 캐시 워밍업
	request := domain.NewRequest("warmup", "GET", "/api/users")
	bridgeSvc.ProcessRequest(ctx, request)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := domain.NewRequest(fmt.Sprintf("cached-request-%d", i), "GET", "/api/users")
		_, err := bridgeSvc.ProcessRequest(ctx, request)
		if err != nil {
			b.Errorf("Cached request failed: %v", err)
		}
	}
}

// 병렬 요청 처리 벤치마크
func BenchmarkParallelRequests(b *testing.B) {
	bridgeSvc, cleanup := setupPerformanceTest()
	defer cleanup()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			request := domain.NewRequest("parallel-request", "GET", "/api/users")
			_, err := bridgeSvc.ProcessRequest(ctx, request)
			if err != nil {
				b.Errorf("Parallel request failed: %v", err)
			}
		}
	})
}

// 메모리 사용량 벤치마크
func BenchmarkMemoryUsage(b *testing.B) {
	bridgeSvc, cleanup := setupPerformanceTest()
	defer cleanup()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		request := domain.NewRequest(fmt.Sprintf("memory-test-%d", i), "GET", "/api/users")
		_, err := bridgeSvc.ProcessRequest(ctx, request)
		if err != nil {
			b.Errorf("Memory test request failed: %v", err)
		}
	}
}

// 응답 시간 측정 테스트
func TestResponseTime(t *testing.T) {
	bridgeSvc, cleanup := setupPerformanceTest()
	defer cleanup()

	ctx := context.Background()
	request := domain.NewRequest("response-time-test", "GET", "/api/users")

	start := time.Now()
	_, err := bridgeSvc.ProcessRequest(ctx, request)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// 응답 시간이 30ms 미만인지 확인
	if duration > 30*time.Millisecond {
		t.Errorf("Response time too slow: %v (expected < 30ms)", duration)
	}

	t.Logf("Response time: %v", duration)
}

// 동시 요청 처리 테스트
func TestConcurrentRequests(t *testing.T) {
	bridgeSvc, cleanup := setupPerformanceTest()
	defer cleanup()

	ctx := context.Background()
	concurrency := 100
	requestsPerGoroutine := 10

	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency*requestsPerGoroutine)

	start := time.Now()

	// 동시 요청 생성
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < requestsPerGoroutine; j++ {
				request := domain.NewRequest(
					fmt.Sprintf("concurrent-request-%d-%d", workerID, j),
					"GET",
					"/api/users",
				)

				_, err := bridgeSvc.ProcessRequest(ctx, request)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	// 모든 고루틴 완료 대기
	for i := 0; i < concurrency; i++ {
		<-done
	}

	close(done)
	close(errors)

	duration := time.Since(start)
	totalRequests := concurrency * requestsPerGoroutine

	// 에러 확인
	var errorCount int
	for err := range errors {
		t.Logf("Request error: %v", err)
		errorCount++
	}

	// 성능 지표 계산
	tps := float64(totalRequests) / duration.Seconds()
	avgResponseTime := duration / time.Duration(totalRequests)

	t.Logf("Total requests: %d", totalRequests)
	t.Logf("Total time: %v", duration)
	t.Logf("Requests per second: %.2f", tps)
	t.Logf("Average response time: %v", avgResponseTime)
	t.Logf("Error count: %d", errorCount)

	// 성능 목표 확인
	if tps < 1000 {
		t.Errorf("TPS too low: %.2f (expected >= 1000)", tps)
	}

	if avgResponseTime > 100*time.Millisecond {
		t.Errorf("Average response time too high: %v (expected < 100ms)", avgResponseTime)
	}

	if errorCount > totalRequests/10 { // 에러율 10% 미만
		t.Errorf("Too many errors: %d/%d", errorCount, totalRequests)
	}
}

// 부하 테스트 (vegeta 스타일)
func TestLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	bridgeSvc, cleanup := setupPerformanceTest()
	defer cleanup()

	ctx := context.Background()
	duration := 30 * time.Second
	targetRPS := 1000 // 초당 1000 요청

	requestCount := 0
	errorCount := 0
	start := time.Now()

	ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
	defer ticker.Stop()

	timeout := time.After(duration)

	for {
		select {
		case <-timeout:
			goto finish
		case <-ticker.C:
			go func() {
				request := domain.NewRequest(
					fmt.Sprintf("load-test-%d", requestCount),
					"GET",
					"/api/users",
				)

				_, err := bridgeSvc.ProcessRequest(ctx, request)
				requestCount++

				if err != nil {
					errorCount++
				}
			}()
		}
	}

finish:
	elapsed := time.Since(start)
	actualRPS := float64(requestCount) / elapsed.Seconds()
	errorRate := float64(errorCount) / float64(requestCount) * 100

	t.Logf("Load test completed:")
	t.Logf("Duration: %v", elapsed)
	t.Logf("Total requests: %d", requestCount)
	t.Logf("Actual RPS: %.2f", actualRPS)
	t.Logf("Error count: %d", errorCount)
	t.Logf("Error rate: %.2f%%", errorRate)

	// 목표 RPS의 80% 이상 달성
	if actualRPS < float64(targetRPS)*0.8 {
		t.Errorf("Actual RPS too low: %.2f (expected >= %.2f)", actualRPS, float64(targetRPS)*0.8)
	}

	// 에러율 1% 미만
	if errorRate > 1.0 {
		t.Errorf("Error rate too high: %.2f%% (expected < 1%%)", errorRate)
	}
}
