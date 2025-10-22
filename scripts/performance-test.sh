#!/bin/bash

# API Bridge 성능 테스트 스크립트 (Bash)

# 기본 매개변수 설정
TEST_TYPE=${1:-"all"}
DURATION=${2:-30}
CONCURRENCY=${3:-100}
RPS=${4:-1000}

echo "🚀 API Bridge Performance Testing"
echo "================================="

# 테스트 타입별 실행
case $TEST_TYPE in
    "benchmark")
        echo "Running benchmark tests..."
        go test -bench=. -benchmem -run=^$ ./test/performance_test.go
        ;;
    
    "load")
        echo "Running load test..."
        echo "Duration: $DURATION seconds, Concurrency: $CONCURRENCY, Target RPS: $RPS"
        go test -run=TestLoadTest -timeout=$((DURATION + 60))s ./test/performance_test.go
        ;;
    
    "concurrent")
        echo "Running concurrent request test..."
        go test -run=TestConcurrentRequests -timeout=60s ./test/performance_test.go
        ;;
    
    "response-time")
        echo "Running response time test..."
        go test -run=TestResponseTime -timeout=30s ./test/performance_test.go
        ;;
    
    "all")
        echo "Running all performance tests..."
        
        echo -e "\n1. Response Time Test"
        go test -run=TestResponseTime -timeout=30s ./test/performance_test.go
        
        echo -e "\n2. Concurrent Requests Test"
        go test -run=TestConcurrentRequests -timeout=60s ./test/performance_test.go
        
        echo -e "\n3. Benchmark Tests"
        go test -bench=. -benchmem -run=^$ ./test/performance_test.go
        
        if [ $DURATION -gt 0 ]; then
            echo -e "\n4. Load Test"
            go test -run=TestLoadTest -timeout=$((DURATION + 60))s ./test/performance_test.go
        fi
        ;;
    
    *)
        echo "Invalid test type: $TEST_TYPE"
        echo "Available types: benchmark, load, concurrent, response-time, all"
        exit 1
        ;;
esac

echo -e "\n✅ Performance testing completed!"
