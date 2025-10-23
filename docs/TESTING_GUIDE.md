# API Bridge 테스트 가이드

API Bridge 시스템의 모든 테스트 방법을 통합한 종합 가이드입니다.

---

## 📋 목차

1. [개요 및 테스트 철학](#개요-및-테스트-철학)
2. [테스트 환경 준비](#테스트-환경-준비)
3. [단위 테스트 (Unit Test)](#단위-테스트-unit-test)
4. [통합 테스트 (Integration Test)](#통합-테스트-integration-test)
5. [성능 테스트 (Performance Test)](#성능-테스트-performance-test)
6. [부하 테스트 (Load Test) - Vegeta](#부하-테스트-load-test---vegeta)
7. [프로파일링](#프로파일링)
8. [테스트 워크플로우](#테스트-워크플로우)
9. [트러블슈팅](#트러블슈팅)

---

## 개요 및 테스트 철학

API Bridge는 **헥사고날 아키텍처**를 기반으로 설계되어 높은 테스트 가능성을 제공합니다. 다양한 레벨의 테스트를 통해 시스템의 안정성과 성능을 보장합니다.

### 테스트 피라미드

```
        🔺 E2E 테스트 (부하 테스트)
       🔺🔺 통합 테스트 (API 테스트)
     🔺🔺🔺 단위 테스트 (비즈니스 로직)
```

- **단위 테스트**: 비즈니스 로직 검증 (빠름, 안정적)
- **통합 테스트**: API 엔드포인트 검증 (중간 속도)
- **부하 테스트**: 성능 및 안정성 검증 (느림, 실제 환경)

---

## 테스트 환경 준비

### 필수 요구사항

- Go 1.21 이상
- 서비스 실행 환경 (포트 10019)
- 선택사항: Oracle DB, Redis

### 환경 설정

1. **의존성 설치**
   ```bash
   go mod download
   ```

2. **서비스 시작**
   ```bash
   # Windows
   .\scripts\start.ps1
   
   # Linux/macOS
   ./scripts/start.sh
   ```

3. **헬스 체크**
   ```bash
   curl http://localhost:10019/health
   ```

---

## 단위 테스트 (Unit Test)

### 목적
- 비즈니스 로직의 정확성 검증
- Mock 객체를 활용한 격리된 테스트
- 빠른 피드백 제공

### 실행 방법

**Windows (PowerShell)**
```powershell
.\scripts\unit-test.ps1
```

**Linux/macOS (Bash)**
```bash
./scripts/unit-test.sh
```

### 주요 기능

- **Race Condition 검사**: `-race` 플래그로 동시성 이슈 탐지
- **커버리지 분석**: `-coverprofile` 플래그로 코드 커버리지 측정
- **상세 출력**: `-v` 플래그로 각 테스트 결과 표시

### 커버리지 확인

```bash
# 커버리지 리포트 생성
go tool cover -func=coverage.out

# HTML 리포트 생성
go tool cover -html=coverage.out -o coverage.html
```

### 예상 결과

```
Running unit tests...
=== RUN   TestBridgeService_ProcessRequest
=== RUN   TestCircuitBreakerService_Execute
=== RUN   TestOrchestrationService_EvaluateTransition
...
Unit tests passed!

Generating coverage report...
api_bridge/internal/core/service/bridge_service.go:85.0%
api_bridge/internal/core/service/circuit_breaker_service.go:92.0%
...
```

---

## 통합 테스트 (Integration Test)

### 목적
- API 엔드포인트의 실제 동작 검증
- 데이터베이스 연동 테스트
- 전체 플로우 검증

### CRUD API 테스트

**실행 방법**
```bash
./scripts/test_crud_api.sh
```

**테스트 대상**
- APIEndpoint CRUD (생성, 조회, 수정, 삭제)
- RoutingRule CRUD
- OrchestrationRule CRUD

### 전체 플로우 테스트

**실행 방법**
```bash
# 통합 테스트 실행
go test -v ./test/integration/...

# 병렬 호출 테스트
go test -v ./test/integration/parallel_calls_test.go
```

### 예상 결과

```
=== RUN   TestCRUDAPIEndpoints
=== RUN   TestCRUDAPIRoutingRules
=== RUN   TestCRUDAPIOrchestrationRules
=== RUN   TestFullFlowIntegration
PASS
ok      api_bridge/test/integration    2.345s
```

---

## 성능 테스트 (Performance Test)

### 목적
- 응답 시간 측정
- 동시성 처리 능력 검증
- 벤치마크 성능 측정

### 실행 방법

**Windows (PowerShell)**
```powershell
# 모든 성능 테스트
.\scripts\performance-test.ps1 -TestType all

# 특정 테스트만
.\scripts\performance-test.ps1 -TestType benchmark
.\scripts\performance-test.ps1 -TestType load -Duration 60
```

**Linux/macOS (Bash)**
```bash
# 모든 성능 테스트
./scripts/performance-test.sh all

# 특정 테스트만
./scripts/performance-test.sh benchmark
./scripts/performance-test.sh load 60
```

### 테스트 유형

1. **Response Time Test**: 단일 요청 응답 시간
2. **Concurrent Requests Test**: 동시 요청 처리
3. **Benchmark Tests**: Go 벤치마크
4. **Load Test**: 지속적인 부하 테스트

### 예상 결과

```
🚀 API Bridge Performance Testing
=================================

1. Response Time Test
PASS: Average response time: 15.3ms

2. Concurrent Requests Test
PASS: 100 concurrent requests handled successfully

3. Benchmark Tests
BenchmarkBridgeService_ProcessRequest-8    1000    1.234ms/op
BenchmarkCircuitBreaker_Execute-8           2000    0.567ms/op

✅ Performance testing completed!
```

---

## 부하 테스트 (Load Test) - Vegeta

### Vegeta란?

**Vegeta**는 Go 언어로 작성된 HTTP 부하 테스트 도구입니다. 고루틴을 활용하여 병렬로 서버에 부하를 주는 방식으로 테스트를 수행합니다.

**주요 특징:**
- 고성능 부하 테스트 (초당 수천 개 요청)
- 명령줄 인터페이스로 간단한 사용
- 다양한 리포트 형태 제공
- 실시간 모니터링 지원

### 설치 방법

Vegeta는 스크립트에서 자동으로 설치됩니다:

```bash
# Go가 설치되어 있어야 함
go install github.com/tsenart/vegeta@latest
```

### 기본 사용법

**Windows (PowerShell)**
```powershell
# 기본 설정으로 실행
.\scripts\vegeta-load-test.ps1

# 커스텀 설정으로 실행
.\scripts\vegeta-load-test.ps1 -Target "http://localhost:10019/api/users" -Duration 120 -Rate 2000 -Method "GET"
```

**Linux/macOS (Bash)**
```bash
# 기본 설정으로 실행
./scripts/vegeta-load-test.sh

# 커스텀 설정으로 실행
./scripts/vegeta-load-test.sh http://localhost:10019/api/users 120 2000 GET results.txt
```

### 주요 파라미터

| 파라미터 | 설명 | 기본값 | 예시 |
|----------|------|--------|------|
| **Target** | 테스트 대상 URL | `http://localhost:10019/api/users` | `http://localhost:10019/api/endpoints` |
| **Duration** | 테스트 지속 시간 (초) | 60 | 120 (2분) |
| **Rate** | 초당 요청 수 | 1000 | 2000 |
| **Method** | HTTP 메서드 | GET | POST, PUT, DELETE |
| **Output** | 결과 파일명 | `results.txt` | `load_test_results.txt` |

### 실행 예시

**시나리오 1: 기본 부하 테스트**
```powershell
.\scripts\vegeta-load-test.ps1 -Rate 1000 -Duration 60
```

**시나리오 2: 고부하 테스트**
```powershell
.\scripts\vegeta-load-test.ps1 -Rate 3000 -Duration 120 -Target "http://localhost:10019/api/users"
```

**시나리오 3: POST 요청 테스트**
```powershell
.\scripts\vegeta-load-test.ps1 -Method POST -Rate 500 -Duration 30
```

### 결과 해석

#### 기본 리포트
```
Requests      [total, rate]            60000, 1000.00
Duration      [total, attack, wait]     1m0s, 1m0s, 15.2ms
Latencies     [mean, 50, 95, 99, max]   15.2ms, 12.1ms, 28.5ms, 45.2ms, 120.3ms
Bytes In      [total, mean]             2.4MB, 41.0B
Bytes Out     [total, mean]             0B, 0.00B
Success       [ratio]                   100.00%
Status Codes  [code:count]              200:60000
```

**주요 지표:**
- **Requests**: 총 요청 수, 초당 요청 수
- **Duration**: 총 시간, 공격 시간, 대기 시간
- **Latencies**: 평균, p50, p95, p99, 최대 응답 시간
- **Success**: 성공률 (목표: > 99.9%)

#### 히스토그램 리포트
```
Bucket         #     %       Histogram
[0s,    1ms]   5000   8.33%  ████████
[1ms,   2ms]   12000  20.00% ████████████████████
[2ms,   5ms]   18000  30.00% ████████████████████████████████
[5ms,   10ms]  15000  25.00% ██████████████████████████
[10ms,  20ms]  8000   13.33% ██████████████
[20ms,  50ms]  2000   3.33%  ████
[50ms,  100ms] 0      0.00%  
[100ms, 200ms] 0      0.00%  
[200ms, 500ms] 0      0.00%  
[500ms, 1s]    0      0.00%  
[1s,    2s]    0      0.00%  
[2s,    5s]    0      0.00%  
[5s,    10s]   0      0.00%  
[10s,   +Inf]  0      0.00%  
```

### 성능 목표 지표

| 지표 | 목표값 | 현재값 | 달성률 |
|------|--------|--------|--------|
| **TPS** | 5,000 req/s | 측정 필요 | - |
| **응답시간 (p95)** | < 30ms | 측정 필요 | - |
| **성공률** | > 99.9% | 측정 필요 | - |
| **메모리 사용량** | < 200MB | 측정 필요 | - |
| **CPU 사용률** | < 50% | 측정 필요 | - |

### 일반적인 시나리오

**1. 점진적 부하 증가 테스트**
```powershell
# 1단계: 기본 부하
.\scripts\vegeta-load-test.ps1 -Rate 500 -Duration 60

# 2단계: 중간 부하
.\scripts\vegeta-load-test.ps1 -Rate 1500 -Duration 60

# 3단계: 고부하
.\scripts\vegeta-load-test.ps1 -Rate 3000 -Duration 60
```

**2. 지속성 테스트**
```powershell
# 장시간 부하 테스트 (10분)
.\scripts\vegeta-load-test.ps1 -Rate 1000 -Duration 600
```

**3. 다양한 엔드포인트 테스트**
```powershell
# 사용자 API 테스트
.\scripts\vegeta-load-test.ps1 -Target "http://localhost:10019/api/users"

# 엔드포인트 API 테스트
.\scripts\vegeta-load-test.ps1 -Target "http://localhost:10019/api/v1/endpoints"

# 헬스 체크 테스트
.\scripts\vegeta-load-test.ps1 -Target "http://localhost:10019/health"
```

---

## 프로파일링

### 목적
- CPU 사용량 분석
- 메모리 할당 패턴 파악
- 고루틴 상태 확인
- 병목 지점 식별

### 실행 방법

**Windows (PowerShell)**
```powershell
# 모든 프로파일 수집
.\scripts\profile.ps1 -Type all

# 특정 프로파일만
.\scripts\profile.ps1 -Type cpu -Duration 60
.\scripts\profile.ps1 -Type mem
.\scripts\profile.ps1 -Type goroutine
```

**Linux/macOS (Bash)**
```bash
# 모든 프로파일 수집
./scripts/profile.sh all

# 특정 프로파일만
./scripts/profile.sh cpu 60
./scripts/profile.sh mem
./scripts/profile.sh goroutine
```

### 분석 방법

```bash
# 웹 UI로 분석 (권장)
go tool pprof -http=:8081 profiling-results/cpu_profile_*.pprof

# 터미널에서 분석
go tool pprof profiling-results/cpu_profile_*.pprof
(pprof) top10        # 상위 10개 함수
(pprof) list <func>  # 특정 함수의 라인별 분석
(pprof) web          # 그래프 시각화
```

### 상세 가이드

프로파일링에 대한 자세한 내용은 [프로파일링 결과 문서](./PROFILING_RESULTS.md)를 참조하세요.

---

## 테스트 워크플로우

### 개발 단계별 권장 테스트 순서

#### 1. 개발 중 (코드 작성 시)
```bash
# 단위 테스트 (빠른 피드백)
.\scripts\unit-test.ps1
```

#### 2. 기능 완성 후
```bash
# 통합 테스트
./scripts/test_crud_api.sh

# 성능 테스트
.\scripts\performance-test.ps1 -TestType all
```

#### 3. 배포 전
```bash
# 부하 테스트
.\scripts\vegeta-load-test.ps1 -Rate 2000 -Duration 120

# 프로파일링 (필요시)
.\scripts\profile.ps1 -Type all
```

#### 4. 배포 후 모니터링
```bash
# 헬스 체크
curl http://localhost:10019/health

# 메트릭 확인
curl http://localhost:10019/metrics
```

### CI/CD 파이프라인 권장사항

```yaml
# 예시: GitHub Actions
- name: Unit Tests
  run: ./scripts/unit-test.sh

- name: Integration Tests
  run: ./scripts/test_crud_api.sh

- name: Performance Tests
  run: ./scripts/performance-test.sh benchmark

- name: Load Test (Nightly)
  run: ./scripts/vegeta-load-test.sh
  if: github.event_name == 'schedule'
```

---

## 트러블슈팅

### 테스트 실행 시 흔한 문제와 해결방법

#### 1. 서비스가 시작되지 않음

**증상:**
```
❌ Failed to initialize dependencies
```

**해결방법:**
```bash
# 포트 사용 여부 확인
netstat -ano | findstr :10019  # Windows
lsof -i :10019                  # Linux/macOS

# 다른 포트로 실행
.\scripts\start.ps1 -Port 8080
```

#### 2. Vegeta 설치 실패

**증상:**
```
Vegeta not found. Installing...
Go not found. Please install Go first.
```

**해결방법:**
```bash
# Go 설치 확인
go version

# Go 설치 (Windows)
# https://golang.org/dl/ 에서 다운로드

# Go 설치 (Linux/macOS)
sudo apt install golang-go  # Ubuntu
brew install go             # macOS
```

#### 3. 테스트 타임아웃

**증상:**
```
panic: test timed out after 30s
```

**해결방법:**
```bash
# 타임아웃 시간 증가
go test -timeout=60s ./test/...

# 또는 스크립트에서 Duration 조정
.\scripts\vegeta-load-test.ps1 -Duration 30
```

#### 4. 메모리 부족

**증상:**
```
fatal error: runtime: out of memory
```

**해결방법:**
```bash
# 부하 테스트 강도 감소
.\scripts\vegeta-load-test.ps1 -Rate 500 -Duration 30

# 메모리 프로파일링으로 원인 분석
.\scripts\profile.ps1 -Type mem
```

#### 5. 네트워크 연결 실패

**증상:**
```
dial tcp [::1]:10019: connect: connection refused
```

**해결방법:**
```bash
# 서비스 상태 확인
curl http://localhost:10019/health

# 서비스 재시작
.\scripts\shutdown.ps1
.\scripts\start.ps1
```

### 성능 이슈 대응

#### 응답 시간이 느린 경우

1. **프로파일링 수행**
   ```bash
   .\scripts\profile.ps1 -Type cpu -Duration 60
   ```

2. **병목 지점 식별**
   - CPU: JSON 직렬화가 느린가?
   - Network: 외부 API 응답이 느린가?
   - Database: DB 쿼리가 느린가?

3. **최적화 적용**
   - Connection Pool 튜닝
   - 캐시 전략 개선
   - 불필요한 직렬화 제거

#### 메모리 사용량 증가

1. **메모리 프로파일링**
   ```bash
   .\scripts\profile.ps1 -Type mem
   ```

2. **분석 및 해결**
   - 메모리 누수 확인
   - 캐시 크기 조정
   - 고루틴 누수 확인

---

## 🔗 참고 자료

### 관련 문서
- [프로파일링 결과](./PROFILING_RESULTS.md)
- [운영 매뉴얼](./OPERATIONS_MANUAL.md)
- [배포 가이드](./DEPLOYMENT_GUIDE.md)

### 외부 자료
- [Go Testing 패키지](https://pkg.go.dev/testing)
- [Vegeta 공식 문서](https://github.com/tsenart/vegeta)
- [pprof 사용법](https://github.com/google/pprof)

---

**Last Updated**: 2025-01-23
**Version**: 1.0.0
