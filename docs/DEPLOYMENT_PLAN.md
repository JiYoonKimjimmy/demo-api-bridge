# API Bridge 시스템 - 개발 계획

API Bridge 시스템 개발을 위한 단계별 계획 및 일정

---

## 📅 전체 개발 일정

### 개발 기간: 약 8-10주

| 단계 | 기간 | 주요 작업 |
|------|------|----------|
| **Phase 1: 기반 구축** | 2주 | 프로젝트 초기화, DB 스키마, 기본 구조 |
| **Phase 2: 핵심 기능** | 3주 | 라우팅, 병렬 호출, 응답 비교 |
| **Phase 3: 전환 로직** | 2주 | 자동 전환, 모니터링, 대시보드 |
| **Phase 4: 안정화** | 2주 | 성능 튜닝, 테스트, 문서화 |
| **Phase 5: 배포 준비** | 1주 | 운영 환경 구축, 롤아웃 |

---

## Phase 1: 기반 구축 (2주)

### Week 1: 프로젝트 초기화 및 기본 구조

#### 1.1. 프로젝트 설정
- [x] Go 프로젝트 초기화 (`go mod init`)
- [x] 기본 디렉토리 구조 생성
  ```
  cmd/
  ├── api-bridge/        # 메인 애플리케이션
  internal/
  ├── gateway/           # API Gateway Layer
  ├── routing/           # Routing Layer
  ├── orchestration/     # Orchestration Layer
  ├── client/            # HTTP Client Layer
  ├── comparison/        # Comparison Engine
  ├── decision/          # Decision Engine
  ├── repository/        # Data Repository
  └── config/            # Configuration
  pkg/
  ├── logger/            # Logging
  ├── metrics/           # Prometheus Metrics
  └── cache/             # Redis Cache
  ```
- [x] 외부 라이브러리 설치
  - [x] Gin/Fiber 프레임워크
  - [ ] GORM (OracleDB 드라이버) *(Mock Repository로 대체)*
  - [x] go-redis
  - [x] prometheus/client_golang
  - [x] zap/zerolog
  - [ ] viper *(기본 설정으로 대체)*

#### 1.2. 설정 관리
- [x] 설정 파일 구조 설계 (config.yaml)
- [ ] Viper 기반 설정 로더 구현 *(기본 설정으로 대체)*
- [x] 환경변수 오버라이드 지원
- [ ] 다중 인스턴스 설정 파일 (config-1.yaml, config-2.yaml 등)

#### 1.3. 로깅 시스템
- [x] 구조화된 로깅 (JSON 형식)
- [x] Trace ID 생성 및 컨텍스트 전파
- [ ] 로그 레벨 동적 변경 API
- [ ] 로그 파일 로테이션 (lumberjack)

### Week 2: 데이터베이스 및 캐시

#### 2.1. OracleDB 스키마
- [x] 테이블 설계 및 DDL 작성 *(Mock Repository로 구현)*
  - [x] `api_mappings`: API 매핑 정보
  - [x] `comparison_history`: 비교 결과 이력
  - [x] `transition_history`: 전환 이력
- [ ] 인덱스 생성 *(Mock에서는 불필요)*
- [x] GORM 모델 정의 *(Domain 모델로 대체)*
- [x] Repository 패턴 구현

#### 2.2. Redis 캐시
- [x] Redis 연결 풀 설정
- [x] 캐시 키 설계
- [x] TTL 전략 구현
- [x] Cache Aside 패턴 구현

#### 2.3. Health Check
- [x] `/health` 엔드포인트 (기본 헬스체크)
- [x] `/ready` 엔드포인트 (DB, Redis 연결 확인)

---

## Phase 2: 핵심 기능 구현 (3주)

### Week 3: API Gateway & Routing Layer

#### 3.1. API Gateway Layer
- [x] HTTP Server 초기화 (Gin/Fiber)
- [ ] Middleware 구현 *(기본 미들웨어만 구현)*
  - [x] Request Logger
  - [ ] CORS Handler
  - [ ] Rate Limiter
  - [ ] Request Validator
- [ ] 가상IP 바인딩 구현
- [x] Graceful Shutdown

#### 3.2. Routing Layer
- [x] APIMapping 구조체 및 Repository *(Domain 및 Mock Repository로 구현)*
- [ ] Radix Tree 기반 URL 매칭 *(기본 매칭으로 대체)*
- [x] 캐시 조회 로직
- [x] 라우팅 전략 선택

### Week 4: Orchestration Layer

#### 4.1. 병렬 호출 구현
- [ ] 고루틴/채널 기반 병렬 호출
- [ ] Context 기반 타임아웃 관리
- [ ] Worker Pool 패턴 구현
- [ ] 응답 수집 및 aggregation

#### 4.2. Circuit Breaker
- [ ] gobreaker 통합
- [ ] 레거시/모던 각각 Circuit Breaker
- [ ] 상태 변경 이벤트 처리
- [ ] Circuit Breaker 메트릭

### Week 5: HTTP Client & Comparison Engine

#### 5.1. HTTP Client Layer
- [x] HTTP Client 구현
- [ ] Connection Pool 최적화
- [x] Retry 로직 (Exponential Backoff)
- [x] 타임아웃 설정 (Dial, Response Header 등)
- [ ] HTTP/2 지원

#### 5.2. Comparison Engine
- [ ] JSON Diff 알고리즘 구현
- [ ] 재귀적 비교 로직
- [ ] 일치율 계산
- [ ] Difference 타입 분류 (MISSING, EXTRA, VALUE_MISMATCH, TYPE_MISMATCH)
- [ ] 비교 규칙 엔진 (제외 필드, 허용 오차)

---

## Phase 3: 전환 로직 및 모니터링 (2주)

### Week 6: Decision Engine

#### 6.1. 전환 결정 로직
- [ ] Threshold Evaluator 구현
- [ ] 임계값 체크 로직
- [ ] 최근 N개 요청 일치율 확인 (안정성)
- [ ] 자동 전환 조건 평가

#### 6.2. Transition Controller
- [ ] 전환 실행 로직 (PARALLEL → MODERN_ONLY)
- [ ] 롤백 로직 (MODERN_ONLY → PARALLEL)
- [ ] 전환 이력 저장
- [ ] 캐시 무효화

#### 6.3. Event System
- [ ] 이벤트 버스 구현
- [ ] 전환 이벤트 발행
- [ ] 이벤트 구독 및 처리

### Week 7: 모니터링 및 메트릭

#### 7.1. Prometheus 메트릭
- [ ] 커스텀 메트릭 정의
  - API 호출 카운터
  - 응답 시간 히스토그램
  - 일치율 게이지
  - 전환율 게이지
  - Circuit Breaker 상태
- [ ] `/metrics` 엔드포인트
- [ ] 메트릭 수집 로직

#### 7.2. Grafana 대시보드
- [ ] 대시보드 설계
- [ ] Prometheus 데이터소스 연동
- [ ] 패널 구성
  - API별 일치율 그래프
  - 전환율 현황
  - 응답 시간 비교
  - 에러율 추이

#### 7.3. 알림 시스템
- [ ] AlertManager 연동
- [ ] 알림 규칙 정의
  - 응답 불일치 발생
  - 에러율 임계값 초과
  - 전환 완료

---

## Phase 4: 안정화 및 테스트 (2주)

### Week 8: 테스트

#### 8.1. 단위 테스트
- [ ] 각 계층별 단위 테스트 작성
- [ ] 테이블 드리븐 테스트
- [ ] Mock 서버 구현 (httptest)
- [ ] 테스트 커버리지 80% 이상

#### 8.2. 통합 테스트
- [ ] 전체 플로우 테스트
- [ ] 병렬 호출 테스트
- [ ] Circuit Breaker 동작 테스트
- [ ] 전환 로직 테스트

#### 8.3. 성능 테스트
- [ ] 벤치마크 테스트 작성
- [ ] 부하 테스트 (vegeta, k6)
- [ ] 레이턴시 측정 및 최적화
- [ ] TPS 목표 달성 검증 (5,000 TPS)

### Week 9: 성능 최적화

#### 9.1. 프로파일링
- [ ] CPU 프로파일링 (pprof)
- [ ] 메모리 프로파일링
- [ ] 고루틴 누수 체크
- [ ] 병목 지점 식별

#### 9.2. 최적화
- [ ] Connection Pool 튜닝
- [ ] 고루틴 워커 풀 크기 조정
- [ ] 캐시 TTL 최적화
- [ ] DB 쿼리 최적화

#### 9.3. 문서화
- [ ] API 문서 작성 (Swagger/OpenAPI)
- [ ] 코드 주석 보완
- [ ] README 업데이트
- [ ] 운영 매뉴얼 작성

---

## Phase 5: 배포 준비 (1주)

### Week 10: 운영 환경 구축

#### 10.1. 인프라 준비
- [ ] 가상IP 설정
- [ ] 방화벽 규칙 설정
- [ ] OracleDB 접속 정보 확인
- [ ] Redis 서버 구축

#### 10.2. 배포 스크립트
- [ ] Shell Script 작성
  - start.sh
  - stop.sh
  - restart.sh
  - status.sh
  - watchdog.sh
  - deploy.sh
  - rollback.sh
- [ ] Cron 기반 watchdog 설정
- [ ] 로그 로테이션 설정 (logrotate)

#### 10.3. 모니터링 구축
- [ ] Prometheus 서버 설정
- [ ] Grafana 대시보드 배포
- [ ] AlertManager 설정
- [ ] 로그 수집 (Filebeat/Fluentd)

#### 10.4. 운영 테스트
- [ ] 개발 환경 배포 테스트
- [ ] 스테이징 환경 배포
- [ ] 무중단 배포 검증
- [ ] 롤백 시나리오 테스트

#### 10.5. 프로덕션 배포
- [ ] 프로덕션 환경 배포
- [ ] 로드밸런서 설정
- [ ] 헬스체크 확인
- [ ] 모니터링 확인

---

## 🎯 마일스톤

### Milestone 1: 기반 완성 (Week 2) ✅
- ✅ 프로젝트 구조 및 설정
- ✅ DB 스키마 완성 *(Mock Repository)*
- ✅ Health Check 동작

### Milestone 2: 병렬 호출 (Week 5)
- ✅ 레거시/모던 API 병렬 호출 성공
- ✅ 응답 비교 로직 동작
- ✅ Circuit Breaker 동작

### Milestone 3: 자동 전환 (Week 7)
- ✅ 일치율 기반 자동 전환 동작
- ✅ 모니터링 대시보드 완성
- ✅ 전환 이력 저장

### Milestone 4: 프로덕션 준비 (Week 10)
- ✅ 모든 테스트 통과
- ✅ 성능 목표 달성
- ✅ 운영 환경 배포 완료

---

## 🏃 스프린트 계획

### Sprint 1 (Week 1-2): 기반 구축
**목표**: 프로젝트 초기화 및 데이터 계층 완성

**주요 Task**:
1. 프로젝트 구조 생성
2. OracleDB 스키마 작성 및 적용
3. Redis 캐시 구현
4. 설정 관리 시스템
5. 로깅 시스템

**완료 조건**:
- [x] Health Check API 동작
- [x] DB 연결 정상 *(Mock Repository로 구현)*
- [x] Redis 캐시 동작
- [x] 구조화된 로그 출력

---

### Sprint 2 (Week 3-4): 라우팅 및 병렬 호출

**목표**: API Gateway 및 병렬 호출 메커니즘 구현

**주요 Task**:
1. API Gateway Layer 구현
2. Routing Layer 구현
3. Orchestration Layer 구현
4. HTTP Client Layer 구현
5. Circuit Breaker 통합

**완료 조건**:
- [ ] 클라이언트 요청 수신 및 라우팅
- [ ] 레거시/모던 API 병렬 호출 성공
- [ ] Circuit Breaker 동작 확인
- [ ] 레거시 응답 반환

---

### Sprint 3 (Week 5-6): 응답 비교 및 전환 로직

**목표**: 응답 비교 및 자동 전환 구현

**주요 Task**:
1. Comparison Engine 구현
2. Decision Engine 구현
3. Transition Controller 구현
4. 이벤트 시스템 구현

**완료 조건**:
- [ ] JSON Diff 정확도 100%
- [ ] 일치율 계산 정확
- [ ] 자동 전환 동작
- [ ] 롤백 기능 동작

---

### Sprint 4 (Week 7-8): 모니터링 및 테스트

**목표**: 모니터링 시스템 구축 및 테스트

**주요 Task**:
1. Prometheus 메트릭 구현
2. Grafana 대시보드 구성
3. 단위 테스트 작성
4. 통합 테스트 작성
5. 성능 테스트

**완료 조건**:
- [ ] 메트릭 정상 수집
- [ ] 대시보드 동작
- [ ] 테스트 커버리지 80% 이상
- [ ] 성능 목표 달성 (5,000 TPS, < 30ms)

---

### Sprint 5 (Week 9-10): 안정화 및 배포

**목표**: 성능 최적화 및 프로덕션 배포

**주요 Task**:
1. 프로파일링 및 최적화
2. 배포 스크립트 작성
3. 운영 환경 구축
4. 문서화
5. 프로덕션 배포

**완료 조건**:
- [ ] 메모리 사용량 < 200MB
- [ ] 무중단 배포 검증
- [ ] 모든 문서 완성
- [ ] 프로덕션 안정 운영

---

## 📋 상세 Task 리스트

### 기술 부채 관리

| 우선순위 | Task | 이유 |
|---------|------|------|
| **P0 (Critical)** | Circuit Breaker 구현 | 장애 격리 필수 |
| **P0** | Graceful Shutdown | 무중단 배포 필수 |
| **P0** | 응답 비교 정확도 | 핵심 기능 |
| **P1 (High)** | Connection Pool 최적화 | 성능 목표 달성 |
| **P1** | 캐시 전략 | 성능 개선 |
| **P2 (Medium)** | OpenTelemetry 통합 | 분산 추적 |
| **P2** | Admin API | 관리 편의성 |
| **P3 (Low)** | gRPC 지원 | 프로토콜 확장 |

---

## 🧪 테스트 전략

### 단위 테스트
- **대상**: 모든 계층의 핵심 로직
- **도구**: Go testing, testify
- **목표**: 커버리지 80% 이상

### 통합 테스트
- **대상**: 전체 플로우 (요청 → 응답 → 비교 → 전환)
- **도구**: httptest, testcontainers
- **시나리오**:
  - PARALLEL 모드 정상 동작
  - MODERN_ONLY 전환
  - Circuit Breaker 동작
  - 장애 복구

### 성능 테스트
- **도구**: vegeta, k6
- **시나리오**:
  - 5,000 TPS 부하 테스트
  - 레이턴시 측정 (p50, p95, p99)
  - 메모리 사용량 모니터링
  - Spike Test (급격한 트래픽 증가)

### 부하 테스트 계획
```bash
# vegeta를 이용한 부하 테스트
echo "GET http://192.168.1.101:10019/api/users" | \
  vegeta attack -duration=60s -rate=5000 | \
  vegeta report -type=text

# 목표
# - Latency p95: < 30ms
# - Success Rate: > 99.9%
# - Throughput: 5,000 req/s
```

---

## 🚀 배포 전략

### 단계별 배포

#### 1단계: 개발 환경 (Week 9)
- 단일 인스턴스 배포
- 기능 검증
- 성능 테스트

#### 2단계: 스테이징 환경 (Week 10)
- 다중 인스턴스 배포 (가상IP)
- 무중단 배포 검증
- 모니터링 확인

#### 3단계: 프로덕션 Canary (Week 10)
- Instance 1만 배포 (10% 트래픽)
- 24시간 모니터링
- 문제 없으면 전체 배포

#### 4단계: 프로덕션 전체 (Week 10)
- Rolling Update 방식
- 인스턴스별 순차 배포
- 실시간 모니터링

---

## 📊 성공 지표 (KPI)

### 기능 지표
- [ ] API 매핑 설정 성공률: 100%
- [ ] 응답 비교 정확도: 100%
- [ ] 자동 전환 성공률: 100%
- [ ] 롤백 성공률: 100%

### 성능 지표
- [ ] 브리지 추가 레이턴시: < 30ms (p95)
- [ ] 처리량: ≥ 5,000 TPS
- [ ] 메모리 사용량: < 200MB
- [ ] CPU 사용률: < 50% (정상 부하)

### 안정성 지표
- [ ] 가용성: 99.9% 이상
- [ ] Circuit Breaker 정상 동작률: 100%
- [ ] 무중단 배포 성공률: 100%

### 운영 지표
- [ ] 평균 배포 시간: < 10분
- [ ] 평균 롤백 시간: < 3분
- [ ] 장애 복구 시간 (MTTR): < 5분

---

## 🔄 리스크 관리

### 주요 리스크

| 리스크 | 영향 | 확률 | 대응 방안 |
|--------|------|------|----------|
| **OracleDB 드라이버 이슈** | 높음 | 중간 | 사전 PoC, 대체 드라이버 검토 |
| **성능 목표 미달성** | 높음 | 낮음 | 프로파일링, 단계별 최적화 |
| **가상IP 설정 제약** | 중간 | 낮음 | 인프라팀 사전 협의 |
| **레거시/모던 API 응답 불일치** | 중간 | 높음 | 비교 규칙 유연화, 허용 오차 설정 |
| **배포 중 장애** | 높음 | 낮음 | Rolling Update, 즉시 롤백 |

### 리스크 대응

**기술적 리스크**:
- 주요 라이브러리 사전 검증 (PoC)
- 성능 테스트 조기 실시
- 단계별 코드 리뷰

**일정 리스크**:
- 버퍼 기간 확보 (각 Phase 10% 여유)
- 우선순위 기반 개발
- Critical Path 집중

**운영 리스크**:
- 충분한 테스트 기간
- Canary 배포로 리스크 최소화
- 롤백 계획 사전 수립

---

## 👥 역할 및 책임

### 개발팀

| 역할 | 담당자 | 책임 |
|------|--------|------|
| **Tech Lead** | TBD | 아키텍처 설계, 코드 리뷰 |
| **Backend Developer 1** | TBD | Gateway, Routing Layer |
| **Backend Developer 2** | TBD | Orchestration, HTTP Client |
| **Backend Developer 3** | TBD | Comparison, Decision Engine |

### 운영팀

| 역할 | 담당자 | 책임 |
|------|--------|------|
| **DevOps** | TBD | 인프라 구축, 배포 자동화 |
| **DBA** | TBD | DB 스키마, 성능 튜닝 |
| **Monitoring** | TBD | 모니터링 시스템 구축 |

---

## 📝 의사결정 기록

### ADR (Architecture Decision Record)

#### ADR-001: Go 언어 선택
- **날짜**: 2025-09-30
- **결정**: API Bridge를 Go로 개발
- **이유**: 고성능, 동시성, 낮은 메모리 사용, 운영 편의성
- **대안**: Spring Boot WebFlux
- **상태**: Accepted

#### ADR-002: Shell Script 기반 프로세스 관리
- **날짜**: 2025-09-30
- **결정**: Systemd 대신 Shell Script 사용
- **이유**: Root 권한 불필요, 빠른 배포, 가상IP 관리 용이
- **대안**: Systemd, Supervisor
- **상태**: Accepted

#### ADR-003: 가상IP 기반 다중 인스턴스
- **날짜**: 2025-09-30
- **결정**: 포트 분리 대신 가상IP 분리
- **이유**: 현재 인프라 구조, 동일 포트 사용
- **대안**: 포트 분리 (8080, 8081, 8082)
- **상태**: Accepted

---

## 📌 체크리스트

### 개발 시작 전
- [x] 인프라 요구사항 확인
- [ ] 레거시/모던 API 명세 확보
- [x] OracleDB 접속 정보 확보 *(Mock Repository로 대체)*
- [x] 개발 환경 구축 (Go 1.25.1)
- [x] Git Repository 생성

### 개발 중
- [ ] 코드 리뷰 프로세스
- [ ] 주간 진행 상황 공유
- [ ] 기술 부채 관리
- [ ] 테스트 커버리지 유지

### 배포 전
- [ ] 모든 테스트 통과
- [ ] 성능 목표 달성 검증
- [ ] 운영 매뉴얼 작성
- [ ] 롤백 계획 수립
- [ ] 모니터링 대시보드 구축

### 배포 후
- [ ] 실시간 모니터링
- [ ] 에러 로그 확인
- [ ] 성능 지표 추적
- [ ] 사용자 피드백 수집

---

## 🔗 참고 문서

- [README.md](../README.md): 프로젝트 개요
- [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md): 상세 구현 가이드
- [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md): 배포 가이드

---

## 📞 Contact

- **프로젝트 관리자**: TBD
- **기술 리더**: TBD
- **문의**: TBD
