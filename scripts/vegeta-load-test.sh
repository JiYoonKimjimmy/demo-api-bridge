#!/bin/bash

# Vegeta를 사용한 부하 테스트 스크립트 (Bash)

# 기본 매개변수 설정
TARGET=${1:-"http://localhost:10019/api/users"}
DURATION=${2:-60}
RATE=${3:-1000}
METHOD=${4:-"GET"}
OUTPUT=${5:-"results.txt"}

echo "Vegeta Load Testing"
echo "====================="

# Vegeta 설치 확인
if ! command -v vegeta &> /dev/null; then
    echo "Vegeta not found. Installing..."
    
    # Go가 설치되어 있는지 확인
    if ! command -v go &> /dev/null; then
        echo "Go not found. Please install Go first."
        exit 1
    fi
    
    # Vegeta 설치
    go install github.com/tsenart/vegeta@latest
    
    # PATH에 GOPATH/bin 추가
    export PATH="$PATH:$(go env GOPATH)/bin"
fi

echo "Target: $TARGET"
echo "Duration: $DURATION seconds"
echo "Rate: $RATE requests/second"
echo "Method: $METHOD"

# Vegeta 명령어 구성
VEGETA_CMD="echo '$METHOD $TARGET' | vegeta attack -duration=${DURATION}s -rate=$RATE | vegeta report -type=text"

echo -e "\nStarting load test..."

# 테스트 실행
eval $VEGETA_CMD

echo -e "\nDetailed Results:"

# 상세 결과 생성
DETAILED_CMD="echo '$METHOD $TARGET' | vegeta attack -duration=${DURATION}s -rate=$RATE | vegeta report -type=hist[0,1ms,2ms,5ms,10ms,20ms,50ms,100ms,200ms,500ms,1s,2s,5s,10s]"
eval $DETAILED_CMD

# 결과를 파일로 저장
echo -e "\nSaving results to $OUTPUT..."
SAVE_CMD="echo '$METHOD $TARGET' | vegeta attack -duration=${DURATION}s -rate=$RATE | vegeta report -type=text > $OUTPUT"
eval $SAVE_CMD

echo -e "\nLoad test completed!"
echo "Results saved to: $OUTPUT"

# 성능 목표 확인
echo -e "\nPerformance Targets:"
echo "- Target RPS: 5,000 (current: $RATE)"
echo "- Target Latency (p95): < 30ms"
echo "- Target Success Rate: > 99.9%"

echo -e "\nTo view results in real-time:"
echo "echo '$METHOD $TARGET' | vegeta attack -duration=${DURATION}s -rate=$RATE | vegeta report -type=text"
