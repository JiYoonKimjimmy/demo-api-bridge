#!/bin/bash

# Vegeta를 사용한 부하 테스트 스크립트 (Bash)

# 기본값 설정
TARGET="http://localhost:10019/api/users"
DURATION=60
RATE=1000
METHOD="GET"
OUTPUT="results.txt"

# 사용법 출력 함수
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
    -t TARGET    Target URL (default: http://localhost:10019/api/users)
    -d DURATION  Test duration in seconds (default: 60)
    -r RATE      Requests per second (default: 1000)
    -m METHOD    HTTP method (default: GET)
    -o OUTPUT    Output file (default: results.txt)
    -h           Show this help message

Examples:
    $0 -t "http://localhost:10019/api/test" -d 30
    $0 -r 2000 -d 120
    $0 -t "http://localhost:10019/api/users" -m POST -d 60

EOF
    exit 0
}

# 명명된 옵션 파싱
while getopts "t:d:r:m:o:h" opt; do
    case $opt in
        t) TARGET="$OPTARG" ;;
        d) DURATION="$OPTARG" ;;
        r) RATE="$OPTARG" ;;
        m) METHOD="$OPTARG" ;;
        o) OUTPUT="$OPTARG" ;;
        h) usage ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            echo "Use -h for help"
            exit 1
            ;;
        :)
            echo "Option -$OPTARG requires an argument" >&2
            exit 1
            ;;
    esac
done

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
