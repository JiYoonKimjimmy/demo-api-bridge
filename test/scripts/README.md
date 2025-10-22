# Test Scripts

이 디렉토리는 API Bridge 시스템의 테스트 스크립트들을 포함합니다.

## 파일 구조

```
test/
├── scripts/                    # 테스트 스크립트
│   └── test_crud_api.sh       # CRUD API 통합 테스트 스크립트
├── crud_api_test.go           # CRUD API Go 테스트
├── db_test.go                 # 데이터베이스 테스트
├── performance_test.go        # 성능 테스트
├── redis_test.go              # Redis 테스트
└── integration/               # 통합 테스트
    ├── full_flow_test.go      # 전체 플로우 테스트
    └── parallel_calls_test.go # 병렬 호출 테스트
```

## 테스트 실행 방법

### 1. Go 테스트 실행

```bash
# 모든 테스트 실행
go test ./test/...

# 특정 테스트 실행
go test ./test/crud_api_test.go -v

# 통합 테스트 실행
go test ./test/integration/... -v
```

### 2. 스크립트 테스트 실행

```bash
# 서버가 실행 중인 상태에서
cd test/scripts
./test_crud_api.sh
```

## 테스트 종류

### Unit Tests (단위 테스트)
- `crud_api_test.go`: CRUD API의 각 엔드포인트별 단위 테스트
- `db_test.go`: 데이터베이스 연결 및 쿼리 테스트
- `redis_test.go`: Redis 캐시 기능 테스트

### Integration Tests (통합 테스트)
- `integration/full_flow_test.go`: 전체 API 플로우 통합 테스트
- `integration/parallel_calls_test.go`: 병렬 API 호출 통합 테스트

### Performance Tests (성능 테스트)
- `performance_test.go`: 부하 테스트 및 성능 벤치마크

### Script Tests (스크립트 테스트)
- `scripts/test_crud_api.sh`: 실제 HTTP API 호출을 통한 E2E 테스트

## 테스트 환경 설정

### 필수 요구사항
- Go 1.21 이상
- 서버 실행 (스크립트 테스트용)
- jq (JSON 파싱용, 스크립트 테스트에서 사용)

### 환경 변수
```bash
export API_BRIDGE_URL="http://localhost:10019"
export API_BRIDGE_TIMEOUT="30s"
```

## 테스트 데이터

테스트는 Mock Repository를 사용하므로 실제 데이터베이스나 Redis가 필요하지 않습니다. 하지만 통합 테스트를 위해서는 실제 서비스가 실행 중이어야 합니다.

## 테스트 결과 확인

### Go 테스트
```bash
# 상세한 테스트 결과 확인
go test ./test/... -v -cover

# 커버리지 리포트 생성
go test ./test/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 스크립트 테스트
스크립트 테스트는 각 단계별로 응답을 출력하므로 실시간으로 결과를 확인할 수 있습니다.
