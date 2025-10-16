# API Bridge 성능 테스트 스크립트

param(
    [string]$TestType = "all",
    [int]$Duration = 30,
    [int]$Concurrency = 100,
    [int]$RPS = 1000
)

Write-Host "🚀 API Bridge Performance Testing" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Green

# 테스트 타입별 실행
switch ($TestType) {
    "benchmark" {
        Write-Host "Running benchmark tests..." -ForegroundColor Cyan
        go test -bench=. -benchmem -run=^$ ./test/performance_test.go
    }
    
    "load" {
        Write-Host "Running load test..." -ForegroundColor Cyan
        Write-Host "Duration: $Duration seconds, Concurrency: $Concurrency, Target RPS: $RPS" -ForegroundColor Yellow
        go test -run=TestLoadTest -timeout=$($Duration + 60)s ./test/performance_test.go
    }
    
    "concurrent" {
        Write-Host "Running concurrent request test..." -ForegroundColor Cyan
        go test -run=TestConcurrentRequests -timeout=60s ./test/performance_test.go
    }
    
    "response-time" {
        Write-Host "Running response time test..." -ForegroundColor Cyan
        go test -run=TestResponseTime -timeout=30s ./test/performance_test.go
    }
    
    "all" {
        Write-Host "Running all performance tests..." -ForegroundColor Cyan
        
        Write-Host "`n1. Response Time Test" -ForegroundColor Yellow
        go test -run=TestResponseTime -timeout=30s ./test/performance_test.go
        
        Write-Host "`n2. Concurrent Requests Test" -ForegroundColor Yellow
        go test -run=TestConcurrentRequests -timeout=60s ./test/performance_test.go
        
        Write-Host "`n3. Benchmark Tests" -ForegroundColor Yellow
        go test -bench=. -benchmem -run=^$ ./test/performance_test.go
        
        if ($Duration -gt 0) {
            Write-Host "`n4. Load Test" -ForegroundColor Yellow
            go test -run=TestLoadTest -timeout=$($Duration + 60)s ./test/performance_test.go
        }
    }
    
    default {
        Write-Host "Invalid test type: $TestType" -ForegroundColor Red
        Write-Host "Available types: benchmark, load, concurrent, response-time, all" -ForegroundColor Yellow
        exit 1
    }
}

Write-Host "`n✅ Performance testing completed!" -ForegroundColor Green
