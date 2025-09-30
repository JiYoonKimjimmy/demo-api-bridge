# API Bridge 시스템 😎

레거시 시스템에서 모던 시스템으로 안전하게 마이그레이션하기 위한 API 브리지

---

## 📋 개요

**목적**: 레거시 API와 모던 API 응답을 실시간 비교하여 100% 일치 시 자동 전환

**핵심 가치**:
- 🔄 **무중단 마이그레이션**: 서비스 중단 없이 점진적 전환
- ✅ **검증 기반 전환**: 응답 일치율 100% 도달 시 자동 전환
- 🛡️ **안전한 롤백**: 문제 발생 시 즉시 레거시로 복구

---

## 🎯 핵심 기능

### 1. 라우팅 전략
- **LEGACY_ONLY**: 레거시 API만 호출
- **PARALLEL**: 레거시 + 모던 병렬 호출 및 응답 비교
- **MODERN_ONLY**: 모던 API만 호출 (전환 완료)

### 2. 응답 검증
- JSON 구조 및 값 비교 (재귀적 Diff)
- 일치율 계산 및 실시간 집계
- 불일치 항목 상세 기록

### 3. 자동 전환
- 일치율 임계값 도달 시 자동 전환 (기본: 100%)
- 안정성 보장 (최근 N개 요청 모두 100% 확인)
- 전환 이력 관리

### 4. 모니터링
- API별 일치율/전환율 실시간 추적
- Prometheus + Grafana 대시보드
- 응답 시간, 에러율 메트릭

## 💻 기술 스택

### 개발
- **언어**: Go 1.21+
- **프레임워크**: Gin/Fiber
- **동시성**: Goroutine, Channel, Worker Pool
- **패턴**: Circuit Breaker, Event-Driven

### 데이터
- **DB**: OracleDB (메타데이터)
- **캐시**: Redis (매핑, 일치율)
- **메트릭**: Prometheus

### 모니터링
- **로깅**: zap/zerolog (JSON)
- **추적**: OpenTelemetry
- **대시보드**: Grafana

---

## 🏗️ 시스템 아키텍처

### 전체 구성

```
┌─────────────┐
│   Client    │
│ Application │
└──────┬──────┘
       │ HTTP Request
       ▼
┌─────────────────────────────────────────────────────────────┐
│                     Load Balancer                           │
│                  (L4/L7, Nginx, HAProxy)                    │
└─────────┬───────────────────────┬───────────────────────────┘
          │                       │
          ▼                       ▼
    ┌─────────┐            ┌─────────┐
    │ Bridge  │            │ Bridge  │  ... (N instances)
    │Instance1│            │Instance2│
    └────┬────┘            └────┬────┘
         │                      │
         └──────────┬───────────┘
                    ▼
    ┌───────────────────────────────────────┐
    │      API Bridge Core System           │
    │         (Go Application)              │
    │                                       │
    │  ┌─────────────────────────────────┐  │
    │  │   Routing Layer                 │  │
    │  │  - Request Handler              │  │
    │  │  - Route Matcher                │  │
    │  │  - Strategy Selector            │  │
    │  └──────────────┬──────────────────┘  │
    │                 │                     │
    │  ┌──────────────┴──────────────────┐  │
    │  │   Orchestration Layer           │  │
    │  │  - Parallel Caller (Goroutines) │  │
    │  │  - Response Aggregator          │  │
    │  │  - Circuit Breaker              │  │
    │  └──────────────┬──────────────────┘  │
    │                 │                     │
    │       ┌─────────┴─────────┐           │
    │       │                   │           │
    │       ▼                   ▼           │
    │  ┌─────────┐        ┌─────────┐       │
    │  │ Legacy  │        │ Modern  │       │
    │  │API      │        │API      │       │
    │  │Client   │        │Client   │       │
    │  └────┬────┘        └────┬────┘       │
    │       │                  │            │
    │  ┌────┴──────────────────┴────┐       │
    │  │   Comparison Engine        │       │
    │  │  - JSON Diff               │       │
    │  │  - Match Rate Calculator   │       │
    │  └──────────────┬─────────────┘       │
    │                 │                     │
    │  ┌──────────────┴─────────────────┐   │
    │  │   Decision Engine              │   │
    │  │  - Threshold Evaluator         │   │
    │  │  - Transition Controller       │   │
    │  │  - Fallback Handler            │   │
    │  └────────────────────────────────┘   │
    └───────────────────────────────────────┘
         │           │           │
         ▼           ▼           ▼
    ┌────────┐  ┌────────┐  ┌─────────┐
    │OracleDB│  │ Redis  │  │Prometheu│
    │        │  │(Cache) │  │  s      │
    └────────┘  └────────┘  └─────────┘
         │           │           │
         ▼           ▼           ▼
    [Metadata]  [Runtime]  [Metrics]
                [State]
```

### 7개 핵심 계층

| 계층 | 역할 | 핵심 기능 |
|------|------|----------|
| **1. API Gateway** | 요청 수신 | Middleware (Logger, CORS, Rate Limiter) |
| **2. Routing** | 매핑 조회 | 캐시 기반 라우팅, 전략 선택 |
| **3. Orchestration** | 병렬 호출 | 고루틴/채널, Circuit Breaker |
| **4. HTTP Client** | API 호출 | Connection Pool, Retry 로직 |
| **5. Comparison** | 응답 비교 | JSON Diff, 일치율 계산 |
| **6. Decision** | 전환 결정 | 임계값 평가, 자동 전환 |
| **7. Data Layer** | 데이터 저장 | OracleDB, Redis, Prometheus |

> **계층별 상세 구현**: [ARCHITECTURE_DETAIL.md](./docs/ARCHITECTURE_DETAIL.md)

### 배포 구성 (온프레미스 - 가상IP 기반)

**특징**:
- 멀티 서버 + 각 서버당 멀티 인스턴스
- 가상IP로 인스턴스 분리, 동일 포트(10019) 사용
- Shell Script 기반 프로세스 관리 (Root 권한 불필요)

**예시**:
```
Server 1: 192.168.1.101, 102, 103 (Instance 1, 2, 3)
Server 2: 192.168.2.101, 102, 103
→ L4/L7 로드밸런서가 모든 가상IP:10019를 백엔드 풀로 관리
```

> **배포 가이드**: [PROCESS_MANAGEMENT.md](./docs/PROCESS_MANAGEMENT.md)

---

## 📊 데이터 흐름

### PARALLEL 모드 (검증 단계)

```
1. 클라이언트 요청
2. Routing: 캐시에서 매핑 조회 (Strategy = PARALLEL)
3. Orchestration: 고루틴으로 레거시/모던 병렬 호출
4. 응답 수집 → 레거시 응답 즉시 반환
5. [비동기] 응답 비교 → 일치율 계산 → DB 저장 → 메트릭 기록
```

### MODERN_ONLY 모드 (전환 완료)

```
1. 클라이언트 요청
2. Routing: Strategy = MODERN_ONLY 확인
3. 모던 API만 호출
4. 응답 반환 (비교 스킵)
```

---

## 🛡️ 장애 대응

| 장애 상황 | 대응 전략 |
|----------|----------|
| **레거시 API 장애** | Circuit Breaker Open → 모던 API만 호출 |
| **모던 API 장애** | Circuit Breaker Open → 레거시 API만 호출 |
| **DB 장애** | 캐시에서 읽기 계속 제공 (쓰기만 실패) |
| **Redis 장애** | DB 직접 조회 (성능 저하, 서비스 지속) |

---

## 📈 성능 목표

- **레이턴시**: < 30ms (p95, 브리지 추가 오버헤드)
- **처리량**: 최소 5,000 TPS
- **메모리**: < 200MB (안정 상태)
- **일치율**: 100% 도달 시 자동 전환

---

## 📚 문서

- **[ARCHITECTURE_DETAIL.md](./docs/ARCHITECTURE_DETAIL.md)**: 계층별 상세 구현 코드, DB 스키마
- **[PROCESS_MANAGEMENT.md](./docs/PROCESS_MANAGEMENT.md)**: 프로세스 관리, 가상IP 설정, 배포 스크립트

---

## 🗂️ 프로젝트 구조

```
demo-api-bridge/
├── README.md                      # 프로젝트 개요 (본 문서)
└── docs/
    ├── ARCHITECTURE_DETAIL.md     # 상세 구현 가이드
    └── PROCESS_MANAGEMENT.md      # 운영 가이드
```

---

## 🔧 빠른 시작

- *TODO 작성 예정*
