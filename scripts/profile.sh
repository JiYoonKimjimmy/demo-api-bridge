#!/bin/bash

# API Bridge 프로파일링 스크립트 (Linux/macOS)
# 사용법: ./scripts/profile.sh [cpu|mem|goroutine|all] [duration] [port]

set -e

# 기본 설정
TYPE="${1:-cpu}"
DURATION="${2:-30}"
PORT="${3:-10019}"
OUTPUT_DIR="profiling-results"

# 색상 코드
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# 색상 출력 함수
print_color() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

print_color "$CYAN" "========================================"
print_color "$CYAN" "API Bridge 프로파일링 도구"
print_color "$CYAN" "========================================"
echo ""

# 출력 디렉토리 생성
if [ ! -d "$OUTPUT_DIR" ]; then
    mkdir -p "$OUTPUT_DIR"
    print_color "$GREEN" "✓ 출력 디렉토리 생성: $OUTPUT_DIR"
fi

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BASE_URL="http://localhost:$PORT"

# 서버 상태 확인
check_server_health() {
    if curl -s -f "$BASE_URL/management/health" > /dev/null 2>&1; then
        print_color "$GREEN" "✓ 서버가 실행 중입니다 (Port: $PORT)"
        return 0
    else
        print_color "$RED" "✗ 서버가 실행 중이지 않거나 응답하지 않습니다"
        print_color "$YELLOW" "  먼저 서비스를 시작하세요: ./scripts/start.sh"
        return 1
    fi
}

# CPU 프로파일링
profile_cpu() {
    local duration=$1
    
    print_color "$YELLOW" "\n[CPU 프로파일링]"
    print_color "$GRAY" "수집 시간: $duration 초"
    
    local output_file="$OUTPUT_DIR/cpu_profile_$TIMESTAMP.pprof"
    
    print_color "$GRAY" "프로파일링 수집 중..."
    if curl -s -o "$output_file" "$BASE_URL/debug/pprof/profile?seconds=$duration"; then
        print_color "$GREEN" "✓ CPU 프로파일 저장: $output_file"
        
        # 분석 명령어 안내
        print_color "$CYAN" "\n분석 명령어:"
        echo "  go tool pprof -http=:8081 $output_file"
        print_color "$GRAY" "  또는"
        echo "  go tool pprof $output_file"
    else
        print_color "$RED" "✗ CPU 프로파일링 실패"
    fi
}

# 메모리 프로파일링
profile_memory() {
    print_color "$YELLOW" "\n[메모리 프로파일링]"
    
    local output_file="$OUTPUT_DIR/mem_profile_$TIMESTAMP.pprof"
    
    print_color "$GRAY" "프로파일링 수집 중..."
    if curl -s -o "$output_file" "$BASE_URL/debug/pprof/heap"; then
        print_color "$GREEN" "✓ 메모리 프로파일 저장: $output_file"
        
        # 분석 명령어 안내
        print_color "$CYAN" "\n분석 명령어:"
        echo "  go tool pprof -http=:8082 $output_file"
        print_color "$GRAY" "  또는"
        echo "  go tool pprof -alloc_space $output_file  # 할당된 총 메모리"
        echo "  go tool pprof -inuse_space $output_file  # 현재 사용 중인 메모리"
    else
        print_color "$RED" "✗ 메모리 프로파일링 실패"
    fi
}

# 고루틴 프로파일링
profile_goroutine() {
    print_color "$YELLOW" "\n[고루틴 프로파일링]"
    
    local output_file="$OUTPUT_DIR/goroutine_profile_$TIMESTAMP.pprof"
    
    print_color "$GRAY" "프로파일링 수집 중..."
    if curl -s -o "$output_file" "$BASE_URL/debug/pprof/goroutine"; then
        print_color "$GREEN" "✓ 고루틴 프로파일 저장: $output_file"
        
        # 분석 명령어 안내
        print_color "$CYAN" "\n분석 명령어:"
        echo "  go tool pprof $output_file"
        print_color "$GRAY" "  (pprof) top      # 상위 고루틴"
        print_color "$GRAY" "  (pprof) list     # 코드 레벨 분석"
    else
        print_color "$RED" "✗ 고루틴 프로파일링 실패"
    fi
}

# 블록 프로파일링
profile_block() {
    print_color "$YELLOW" "\n[블록 프로파일링]"
    
    local output_file="$OUTPUT_DIR/block_profile_$TIMESTAMP.pprof"
    
    print_color "$GRAY" "프로파일링 수집 중..."
    if curl -s -o "$output_file" "$BASE_URL/debug/pprof/block"; then
        print_color "$GREEN" "✓ 블록 프로파일 저장: $output_file"
    else
        print_color "$RED" "✗ 블록 프로파일링 실패"
    fi
}

# 뮤텍스 프로파일링
profile_mutex() {
    print_color "$YELLOW" "\n[뮤텍스 프로파일링]"
    
    local output_file="$OUTPUT_DIR/mutex_profile_$TIMESTAMP.pprof"
    
    print_color "$GRAY" "프로파일링 수집 중..."
    if curl -s -o "$output_file" "$BASE_URL/debug/pprof/mutex"; then
        print_color "$GREEN" "✓ 뮤텍스 프로파일 저장: $output_file"
    else
        print_color "$RED" "✗ 뮤텍스 프로파일링 실패"
    fi
}

# 서버 상태 확인
if ! check_server_health; then
    exit 1
fi

echo ""

# 프로파일링 유형에 따라 실행
case "$TYPE" in
    cpu)
        profile_cpu "$DURATION"
        ;;
    mem|memory)
        profile_memory
        ;;
    goroutine)
        profile_goroutine
        ;;
    block)
        profile_block
        ;;
    mutex)
        profile_mutex
        ;;
    all)
        print_color "$CYAN" "전체 프로파일링을 시작합니다..."
        profile_cpu "$DURATION"
        sleep 2
        profile_memory
        sleep 1
        profile_goroutine
        sleep 1
        profile_block
        sleep 1
        profile_mutex
        ;;
    *)
        print_color "$RED" "✗ 잘못된 프로파일링 유형: $TYPE"
        print_color "$YELLOW" "사용 가능한 유형: cpu, mem, goroutine, block, mutex, all"
        exit 1
        ;;
esac

echo ""
print_color "$CYAN" "========================================"
print_color "$GREEN" "프로파일링 완료!"
print_color "$CYAN" "결과 위치: $OUTPUT_DIR"
print_color "$CYAN" "========================================"
echo ""
print_color "$YELLOW" "💡 추천 워크플로우:"
print_color "$GRAY" "1. 부하 테스트 실행: ./scripts/vegeta-load-test.sh"
print_color "$GRAY" "2. 프로파일링 수집: ./scripts/profile.sh all"
print_color "$GRAY" "3. 결과 분석: go tool pprof -http=:8081 <profile_file>"
echo ""

