# API Bridge 프로파일링 결과

이 문서는 API Bridge 시스템의 프로파일링 결과 및 성능 분석을 기록합니다.

---

## 📊 프로파일링 개요

### 프로파일링의 목적

프로메테우스 메트릭은 **"무엇이 일어나고 있는가"**(What)를 모니터링하는 데 사용되지만, pprof 프로파일링은 **"왜 느린가? 어디를 최적화해야 하는가"**(Why, Where)를 분석하는 데 사용됩니다.

| 구분 | 프로메테우스 메트릭 | pprof 프로파일링 |
|------|-------------------|-----------------|
| **목적** | 운영 중 실시간 모니터링 | 성능 병목 지점 상세 분석 |
| **데이터** | 비즈니스/시스템 메트릭 | 함수/라인 단위 상세 정보 |
| **사용 시점** | 상시 | 온디맨드 (필요할 때만) |
| **예시** | TPS, 응답시간, 에러율 | CPU 사용 함수, 메모리 할당 위치 |

### 프로파일링 도구 사용법

```bash
# Windows
.\scripts\profile.ps1 -Type cpu -Duration 30
.\scripts\profile.ps1 -Type mem
.\scripts\profile.ps1 -Type all

# Linux/macOS
./scripts/profile.sh cpu 30
./scripts/profile.sh mem
./scripts/profile.sh all
```

---

## 🔍 베이스라인 (최적화 전)

### 측정 환경

- **측정 일시**: YYYY-MM-DD HH:MM:SS
- **서버 환경**: 
  - OS: Windows/Linux
  - CPU: 
  - Memory: 
  - Go 버전: 1.25.1
- **부하 조건**:
  - 동시 사용자: 
  - 요청 속도: req/s
  - 테스트 시간: 초

### 성능 지표 (최적화 전)

| 지표 | 측정값 | 목표값 | 달성률 |
|------|--------|--------|--------|
| **TPS** | 1,624 req/s | 5,000 req/s | 32% |
| **응답시간 (p95)** | 17.9ms | < 30ms | ✅ 달성 |
| **메모리 사용량** | - | < 200MB | - |
| **CPU 사용률** | - | < 50% | - |
| **고루틴 수** | - | < 1,000 | - |

---

## 🔥 CPU 프로파일링 결과

### 프로파일 파일
- **파일명**: `profiling-results/cpu_profile_YYYYMMDD_HHMMSS.pprof`
- **수집 시간**: 30초
- **분석 명령어**: `go tool pprof -http=:8081 cpu_profile_YYYYMMDD_HHMMSS.pprof`

### CPU 사용량 Top 10

| 순위 | 함수 | CPU 사용률 | 누적 | 비고 |
|------|------|------------|------|------|
| 1 | `json.Marshal` | 40% | 40% | JSON 직렬화 |
| 2 | `service.compareJSON` | 25% | 65% | 응답 비교 로직 |
| 3 | `http.Client.Do` | 15% | 80% | HTTP 호출 |
| 4 | `redis.Set` | 10% | 90% | 캐시 저장 |
| 5 | `logger.Log` | 5% | 95% | 로깅 |

*(실제 프로파일링 후 업데이트)*

### 병목 지점 분석

#### 1. JSON 직렬화 (40%)
**문제점**: JSON Marshal/Unmarshal이 CPU의 40%를 사용
**원인**: 
- 큰 JSON 객체의 반복적인 직렬화
- 불필요한 중복 직렬화

**최적화 방안**:
- [ ] JSON 캐싱: 동일한 응답은 직렬화된 결과를 재사용
- [ ] `easyjson` 또는 `jsoniter` 라이브러리로 전환
- [ ] 필요한 필드만 직렬화

#### 2. 응답 비교 로직 (25%)
**문제점**: JSON Diff 알고리즘이 CPU의 25%를 사용
**원인**:
- 재귀적 비교의 오버헤드
- 모든 필드를 비교하는 완전 비교

**최적화 방안**:
- [ ] 주요 필드만 비교하는 경량 비교 모드 추가
- [ ] 비교 결과 캐싱
- [ ] 병렬 비교 로직

---

## 💾 메모리 프로파일링 결과

### 프로파일 파일
- **파일명**: `profiling-results/mem_profile_YYYYMMDD_HHMMSS.pprof`
- **분석 명령어**: `go tool pprof -http=:8082 mem_profile_YYYYMMDD_HHMMSS.pprof`

### 메모리 할당 Top 10

| 순위 | 함수 | 할당량 | 비율 | 비고 |
|------|------|--------|------|------|
| 1 | `json.Unmarshal` | 50MB | 35% | JSON 파싱 |
| 2 | `strings.Builder` | 30MB | 21% | 문자열 조합 |
| 3 | `http.Request` | 20MB | 14% | HTTP 요청 객체 |
| 4 | `logger.Fields` | 15MB | 11% | 로그 필드 |
| 5 | `redis.Client` | 10MB | 7% | Redis 연결 |

*(실제 프로파일링 후 업데이트)*

### 메모리 이슈 분석

#### 1. JSON 파싱 메모리 (35%)
**문제점**: JSON Unmarshal이 많은 메모리를 할당
**최적화 방안**:
- [ ] `sync.Pool`을 사용한 버퍼 재사용
- [ ] 스트리밍 JSON 파서 사용 (큰 응답의 경우)

#### 2. 문자열 조합 (21%)
**문제점**: 많은 문자열 연결 연산
**최적화 방안**:
- [ ] `strings.Builder` 사용 확대
- [ ] 문자열 템플릿 사전 컴파일

---

## 🔄 고루틴 프로파일링 결과

### 프로파일 파일
- **파일명**: `profiling-results/goroutine_profile_YYYYMMDD_HHMMSS.pprof`
- **분석 명령어**: `go tool pprof goroutine_profile_YYYYMMDD_HHMMSS.pprof`

### 고루틴 현황

| 상태 | 개수 | 비율 | 위치 |
|------|------|------|------|
| **Running** | - | - | - |
| **Waiting** | - | - | `chan receive`, `sync.Mutex.Lock` |
| **Blocked** | - | - | `http.Client.Do` |

*(실제 프로파일링 후 업데이트)*

### 고루틴 누수 체크

- [ ] 장시간 실행 후 고루틴 수 증가 추이 확인
- [ ] 주요 고루틴 생성 위치 파악
- [ ] Context 취소 처리 확인

---

## 📈 최적화 우선순위

프로파일링 결과를 바탕으로 최적화 우선순위를 결정합니다.

### Priority 1 (High Impact, Low Effort)

1. **Connection Pool 튜닝**
   - 현재 상태: MaxIdleConnsPerHost = 10
   - 최적화 목표: 동시 요청 수에 맞게 조정
   - 예상 효과: TPS 20-30% 개선

2. **JSON 직렬화 최적화**
   - 현재 이슈: CPU 40% 사용
   - 최적화 방법: 캐싱 추가
   - 예상 효과: CPU 사용량 10-15% 감소

### Priority 2 (High Impact, Medium Effort)

3. **응답 비교 로직 경량화**
   - 현재 이슈: CPU 25% 사용
   - 최적화 방법: 주요 필드만 비교
   - 예상 효과: CPU 사용량 5-10% 감소

4. **워커 풀 크기 조정**
   - 현재 상태: 동적 고루틴 생성
   - 최적화 방법: 고정 워커 풀 사용
   - 예상 효과: 메모리 사용량 안정화

### Priority 3 (Medium Impact, Low Effort)

5. **캐시 TTL 최적화**
   - 현재 상태: 고정 TTL
   - 최적화 방법: 엔드포인트별 적응형 TTL
   - 예상 효과: 캐시 히트율 10-15% 개선

---

## ✅ 최적화 적용 결과

### Optimization 1: Connection Pool 튜닝

**적용 내용**:
```go
// Before
MaxIdleConnsPerHost: 10

// After
MaxIdleConnsPerHost: 50
MaxIdleConns: 200
```

**결과**:
- TPS: 1,624 → ??? req/s (??% 개선)
- 응답시간 p95: 17.9ms → ??? ms

*(최적화 후 업데이트)*

### Optimization 2: ...

*(추가 최적화 적용 후 업데이트)*

---

## 📊 최종 성능 비교

### Before vs After

| 지표 | Before | After | 개선율 |
|------|--------|-------|--------|
| **TPS** | 1,624 req/s | ??? req/s | +??% |
| **응답시간 (p95)** | 17.9ms | ??? ms | -??% |
| **메모리 사용량** | ??? MB | ??? MB | -??% |
| **CPU 사용률** | ??? % | ??? % | -??% |
| **고루틴 수** | ??? | ??? | -??% |

### 목표 달성 현황

- [ ] TPS 5,000 달성 (현재: ???%)
- [x] 응답시간 p95 < 30ms 달성
- [ ] 메모리 < 200MB 달성
- [ ] CPU 사용률 < 50% 달성

---

## 🔗 참고 자료

### 프로파일링 가이드
- [Go 공식 프로파일링 문서](https://go.dev/blog/pprof)
- [pprof 사용법](https://github.com/google/pprof)

### 최적화 기법
- [Effective Go](https://go.dev/doc/effective_go)
- [High Performance Go Workshop](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)

---

**Last Updated**: YYYY-MM-DD
**Author**: Backend Team

