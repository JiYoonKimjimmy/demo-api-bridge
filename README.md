# API Bridge 시스템 😎

## Introduce

- 기존 `Spring Boot` 기반 API 애플리케이션을 업그레이드 후 **레거시 시스템** to **모던 시스템** 으로 점진적인 API 호출 마이그레이션 처리하는 시스템 구축
- 레거시 시스템에서 제공하는 `레거시 API` 와 동일한 기능을 하는 모던 시스템의 `모던 API` 의 응답값 일치 여부를 검증하여 응답 일치율이 100% 도달하는 경우, `레거시 API` 호출하지 않고 `모던 API` 만 호출하여 응답하도록 마이그레이션 처리

---

## Requirement

### 기능적 요구사항

#### 1. API 라우팅 및 매핑 관리
- 레거시 API와 모던 API 엔드포인트 매핑 구조 관리
- 요청 URL 패턴 기반 동적 라우팅
- API 버전별 라우팅 전략 설정 (레거시 전용, 병렬 호출, 모던 전용)

#### 2. 응답 검증 및 비교
- 레거시 API와 모던 API의 응답값 실시간 비교
- JSON 구조 비교 (필드명, 데이터 타입, 값)
- 응답 일치율 계산 및 집계
- 불일치 항목 상세 기록 (diff 정보)

#### 3. 점진적 전환 로직
- API별 응답 일치율 임계값 설정 (기본: 100%)
- 임계값 도달 시 자동/수동 전환 지원
- 전환 롤백 기능 (문제 발생 시 레거시로 복구)
- Canary 배포 방식 지원 (트래픽 비율 조정)

#### 4. 모니터링 및 대시보드
- **실시간 현황**
  - API별 응답 일치율 현황
  - API별 전환 상태 (레거시/병렬/모던)
  - API 호출 성공률 및 에러율
- **통계 정보**
  - 전체 API 전환율 (전환 완료/전체)
  - API별 평균 응답 시간 비교
  - 일치율 추이 (시계열 그래프)
- **알림 기능**
  - 응답 불일치 발생 시 알림
  - 전환 완료 시 알림
  - 에러율 임계값 초과 시 알림

#### 5. 관리 기능
- API 매핑 설정 CRUD (추가/수정/삭제)
- 전환 상태 수동 제어 (강제 전환/롤백)
- 검증 규칙 설정 (특정 필드 제외, 허용 오차 설정 등)
- 테스트 모드 (실제 전환 없이 검증만 수행)

### 기술적 요구사항

#### 1. 개발 언어 및 프레임워크
- **Go 1.21+** 기반 구현
- **Gin** 또는 **Fiber** 프레임워크 (고성능 HTTP 라우터)
- 표준 라이브러리 `net/http` 패키지 활용
- Context 기반 요청 추적 및 타임아웃 관리

#### 2. 시스템 아키텍처
- **고루틴 기반 동시성**: 레거시/모던 API 병렬 호출
- **채널 기반 통신**: 응답 수집 및 비교 결과 전달
- **워커 풀 패턴**: 고루틴 수 제한 및 리소스 관리
- **Circuit Breaker 패턴**: `gobreaker` 또는 `hystrix-go` 활용
- 이벤트 드리븐 아키텍처 (상태 변경 이벤트 처리)

#### 3. 성능 요구사항
- API 브리지 추가 레이턴시 < 30ms (p95)
- 병렬 호출 시 타임아웃 설정 (기본: 5초, Context로 제어)
- 고루틴 기반 논블로킹 I/O 처리
- 대용량 트래픽 처리 (최소 5000 TPS)
- Connection Pool 최적화 (Keep-Alive, 재사용)
- 메모리 사용량 < 200MB (안정 상태)

#### 4. HTTP 클라이언트
- `net/http` 표준 클라이언트 커스터마이징
- Connection Pool 설정 (MaxIdleConns, MaxConnsPerHost)
- 타임아웃 설정 (DialTimeout, ResponseHeaderTimeout, IdleConnTimeout)
- 재시도 로직 (Exponential Backoff with Jitter)
- HTTP/2 지원

#### 5. 데이터 저장소
- **메타데이터 저장**: OracleDB
  - GORM 또는 sqlx 라이브러리 사용
  - API 매핑, 전환 상태, 설정 관리
- **캐시 레이어**: Redis
  - API 매핑 정보 캐싱 (빠른 조회)
  - 일치율 임시 집계
- **검증 결과 저장**: 
  - 로그 파일 (JSON Lines 형식)
  - 선택적으로 Elasticsearch 연동
- **통계 데이터**: Prometheus Metrics
  - `prometheus/client_golang` 사용
  - Counter, Gauge, Histogram 메트릭 노출

#### 6. 로깅 및 모니터링
- **구조화된 로깅**: `zap` 또는 `zerolog`
  - JSON 형식 로그
  - 요청별 Trace ID/Correlation ID
  - 로그 레벨 동적 변경 지원
- **메트릭 노출**:
  - Prometheus 메트릭 엔드포인트 (`/metrics`)
  - 커스텀 메트릭: 일치율, 전환율, 응답시간 등
- **분산 추적**: OpenTelemetry 또는 Jaeger
- **Health Check**: `/health`, `/ready` 엔드포인트

#### 7. 안정성 및 보안
- **Graceful Shutdown**: 시그널 핸들링 (SIGTERM, SIGINT)
- **Fallback 전략**: 
  - 레거시 API 우선 응답
  - 모던 API 실패 시 레거시만 사용
- **Rate Limiting**: `golang.org/x/time/rate` 활용
- **인증 정보 관리**: 
  - 환경변수 또는 Secret Manager 연동
  - TLS/mTLS 지원
- **민감 정보 마스킹**: 로그 출력 시 자동 마스킹

#### 8. 운영 요구사항 (온프레미스 환경)

##### 8.1. 배포 및 실행
- **바이너리 배포**:
  - 단일 실행 파일 (정적 링크 바이너리)
  - 크로스 컴파일 지원 (Linux amd64/arm64)
  - 버전 정보 임베딩 (빌드 시간, Git 커밋 해시)
  - 외부 의존성 없음 (Go 정적 빌드)
- **실행 환경**:
  - Shell Script 기반 프로세스 관리
  - 일반 사용자 권한으로 실행 (non-root)
  - 백그라운드 실행 (nohup, &)
  - PID 파일 기반 프로세스 추적
- **다중 인스턴스 운영 (가상IP 기반)**:
  - **서버 구성**: 멀티 서버 + 각 서버당 멀티 인스턴스
  - **네트워크 구성**: 
    - 각 인스턴스는 가상IP(Virtual IP) 할당
    - 모든 인스턴스는 동일 포트(10019) 사용
    - 가상IP를 통한 인스턴스 식별 및 분리
  - **가상IP 설정 (예시)**:
    - 서버1: 192.168.1.101 (Instance 1), 192.168.1.102 (Instance 2), 192.168.1.103 (Instance 3)
    - 서버2: 192.168.2.101 (Instance 1), 192.168.2.102 (Instance 2), 192.168.2.103 (Instance 3)
  - **바인딩 설정**: 각 인스턴스가 특정 가상IP에 바인딩하여 기동
  - **로드밸런서 연동**:
    - L4/L7 로드밸런서가 모든 가상IP를 백엔드 풀로 관리
    - Health Check는 각 가상IP:10019로 수행
    - 트래픽 분산 알고리즘 (Round-Robin, Least Connection 등)

##### 8.2. 프로세스 관리 (Shell Script 방식)

**선택 이유**:
- Root 권한 불필요 (일반 사용자 권한으로 운영)
- 간단한 구조로 빠른 배포 가능
- 가상IP 기반 다중 인스턴스 관리 용이
- 배포 스크립트와 쉬운 통합
- PID 기반 프로세스 추적 및 관리

**주요 기능**:
- 백그라운드 실행 (`nohup`, `&`)
- PID 파일 기반 프로세스 관리
- Graceful Shutdown (SIGTERM)
- 헬스체크 통합
- 인스턴스별 독립 로그 관리

> **참고**: 다른 프로세스 관리 방법(Systemd, Supervisor 등) 비교는 [PROCESS_MANAGEMENT.md](./docs/PROCESS_MANAGEMENT.md) 참고

##### 8.3. 설정 관리
- **설정 파일**:
  - YAML/JSON/TOML 포맷 지원
  - 환경별 설정 분리 (dev/stg/prod)
  - `viper` 라이브러리 활용
- **환경변수 오버라이드**:
  - 설정 파일 < 환경변수 우선순위
  - 민감 정보는 환경변수로 관리
- **설정 Hot Reload**:
  - SIGHUP 시그널로 설정 재로드
  - 서비스 중단 없이 설정 변경 반영

##### 8.4. 로그 관리
- **로그 파일 출력**:
  - 로그 파일 경로 설정 가능
  - 로그 로테이션 (logrotate 또는 lumberjack)
  - 파일 크기/일자 기반 자동 압축 및 삭제
- **로그 레벨**:
  - 런타임 로그 레벨 변경 (HTTP API 또는 시그널)
  - 디버그 모드 동적 활성화
- **중앙 로그 수집**:
  - Fluentd/Filebeat를 통한 로그 수집
  - Syslog 프로토콜 지원

##### 8.5. 무중단 배포
- **Graceful Shutdown**:
  - SIGTERM 수신 시 신규 요청 거부
  - 진행 중인 요청 완료 대기 (30초 타임아웃)
  - 리소스 정리 (DB 커넥션, 캐시 등)
- **Rolling Deployment**:
  - 인스턴스별 순차 배포
  - 헬스체크 통과 후 다음 인스턴스 배포
  - 로드밸런서에서 인스턴스 제거/추가
- **Blue-Green 배포** (선택):
  - 신규 버전 전체 배포 후 스위칭
  - 문제 시 즉시 롤백

##### 8.6. 모니터링 및 헬스체크
- **Health Check 엔드포인트**:
  - `/health`: 기본 헬스체크 (프로세스 살아있음)
  - `/ready`: Readiness 체크 (DB, Redis 연결 확인)
  - 로드밸런서 헬스체크 연동
- **메트릭 수집**:
  - Prometheus 메트릭 엔드포인트 (`/metrics`)
  - 별도 Prometheus 서버에서 scrape
  - Grafana 대시보드 연동
- **APM 연동**:
  - Pinpoint, Scouter 등 APM 에이전트 연동 (선택)
  - 분산 추적 (OpenTelemetry)

##### 8.7. 보안 및 네트워크
- **방화벽 설정**:
  - 필요한 포트만 오픈 (애플리케이션, 메트릭)
  - 내부 관리 API는 특정 IP만 허용
- **TLS/SSL**:
  - 인증서 관리 (파일 기반 또는 볼트)
  - 주기적인 인증서 갱신
- **프로세스 권한**:
  - 전용 시스템 계정으로 실행 (non-root)
  - 파일 시스템 권한 제한

##### 8.8. 백업 및 복구
- **애플리케이션 백업**:
  - 바이너리 버전별 보관
  - 설정 파일 버전 관리 (Git)
- **데이터베이스 백업**:
  - 정기 백업 스케줄
  - 백업 검증 프로세스
- **장애 복구 계획**:
  - 롤백 절차 문서화
  - 복구 시간 목표(RTO) 정의

#### 9. 확장성
- **Stateless 설계**: 멀티 인스턴스 운영
- **수평 확장**: 로드밸런서 뒤 N개 인스턴스
- **상태 공유**: Redis를 통한 전환 상태 동기화
- **플러그인 구조**: 
  - 인터페이스 기반 비교 로직 확장
  - 다양한 프로토콜 어댑터 (REST, gRPC)
- **고루틴 제어**: 워커 풀로 동시 실행 수 제한

#### 10. 개발 및 테스트
- **의존성 관리**: Go Modules (`go.mod`)
- **코드 품질**:
  - `golangci-lint` 정적 분석
  - `gofmt`, `goimports` 코드 포맷팅
- **테스트**:
  - 단위 테스트 (testify 활용)
  - 테이블 드리븐 테스트
  - Mock 서버 (httptest)
  - 벤치마크 테스트
- **CI/CD**: GitHub Actions, GitLab CI 등

---

## Architecture

### 전체 시스템 구성도

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
    │  ┌─────────────────────────────────┐ │
    │  │   Routing Layer                 │ │
    │  │  - Request Handler              │ │
    │  │  - Route Matcher                │ │
    │  │  - Strategy Selector            │ │
    │  └──────────────┬──────────────────┘ │
    │                 │                     │
    │  ┌──────────────┴──────────────────┐ │
    │  │   Orchestration Layer           │ │
    │  │  - Parallel Caller (Goroutines) │ │
    │  │  - Response Aggregator          │ │
    │  │  - Circuit Breaker              │ │
    │  └──────────────┬──────────────────┘ │
    │                 │                     │
    │       ┌─────────┴─────────┐          │
    │       │                   │          │
    │       ▼                   ▼          │
    │  ┌─────────┐        ┌─────────┐     │
    │  │ Legacy  │        │ Modern  │     │
    │  │API      │        │API      │     │
    │  │Client   │        │Client   │     │
    │  └────┬────┘        └────┬────┘     │
    │       │                  │          │
    │  ┌────┴──────────────────┴────┐     │
    │  │   Comparison Engine        │     │
    │  │  - JSON Diff               │     │
    │  │  - Match Rate Calculator   │     │
    │  └──────────────┬─────────────┘     │
    │                 │                    │
    │  ┌──────────────┴─────────────────┐ │
    │  │   Decision Engine              │ │
    │  │  - Threshold Evaluator         │ │
    │  │  - Transition Controller       │ │
    │  │  - Fallback Handler            │ │
    │  └────────────────────────────────┘ │
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

### 계층별 아키텍처 개요

> **상세 구현 코드**: 각 계층별 Go 코드 샘플 및 데이터베이스 스키마는 [ARCHITECTURE_DETAIL.md](./docs/ARCHITECTURE_DETAIL.md) 참고

#### 1. API Gateway Layer (진입점)

**역할**: 클라이언트 요청 수신 및 기본 검증

**구성요소**:
- Gin/Fiber 프레임워크
- Middleware Stack (Logger, CORS, Rate Limiter, Auth, Validator)

**처리 흐름**:
1. 클라이언트 요청 수신 → 2. Trace ID 생성 → 3. Rate Limiting 체크 → 4. 요청 유효성 검증 → 5. 라우팅 레이어로 전달

---

#### 2. Routing Layer (라우팅 계층)

**역할**: 요청 URL 매핑 및 호출 전략 결정

**핵심 데이터 구조**:
- `APIMapping`: API 매핑 정보 (ClientPath, LegacyURL, ModernURL, Strategy, MatchRate, Threshold)
- `RoutingStrategy`: LEGACY_ONLY, PARALLEL, MODERN_ONLY

**처리 로직**:
1. 요청 URL 매칭 (Trie/Radix Tree) → 2. 캐시 조회 → 3. 캐시 미스 시 DB 조회 → 4. 전략에 따라 Orchestration Layer 호출

---

#### 3. Orchestration Layer (오케스트레이션 계층)

**역할**: 레거시/모던 API 병렬 호출 및 응답 처리

**핵심 패턴**:
- 고루틴/채널 기반 병렬 호출
- Circuit Breaker 패턴 (장애 격리)
- 비동기 응답 비교 (클라이언트 응답에 영향 없음)

**처리 흐름**:
1. 전략 판단 → 2. 병렬 호출 (고루틴) → 3. 채널로 응답 수집 → 4. 레거시 응답 우선 반환 → 5. [비동기] 응답 비교

---

#### 4. HTTP Client Layer (API 호출 계층)

**역할**: 실제 레거시/모던 API 호출

**주요 기능**:
- Connection Pool 최적화 (Keep-Alive, 재사용)
- Retry 로직 (Exponential Backoff with Jitter)
- 타임아웃 설정 (Dial, Response Header, Idle Conn)
- HTTP/2 지원

---

#### 5. Comparison Engine (비교 엔진)

**역할**: 레거시/모던 API 응답 비교 및 일치율 계산

**핵심 기능**:
- JSON Diff 알고리즘 (재귀적 비교)
- 일치율 계산 (필드 단위)
- 불일치 항목 상세 기록 (경로, 타입, 값)
- 비교 규칙 엔진 (제외 필드, 허용 오차, 순서 무시)

---

#### 6. Decision Engine (의사결정 엔진)

**역할**: 일치율 기반 전환 결정 및 상태 관리

**핵심 로직**:
- 임계값 평가 (기본 100%)
- 자동 전환 (PARALLEL → MODERN_ONLY)
- 롤백 기능
- 전환 이벤트 발행

**전환 프로세스**:
1. 임계값 도달 확인 → 2. DB 상태 업데이트 → 3. 캐시 무효화 → 4. 이벤트 발행 → 5. 메트릭 기록

---

#### 7. Data Layer (데이터 계층)

**OracleDB (메타데이터)**:
- `api_mappings`: API 매핑 정보
- `comparison_history`: 비교 결과 이력
- `transition_history`: 전환 이력

**Redis (캐시 및 실시간 상태)**:
- API 매핑 캐시 (TTL: 10분)
- 일치율 임시 집계 (Sorted Set)
- Circuit Breaker 상태 (TTL: 30초)

**Prometheus (메트릭)**:
- API 호출 카운터
- 응답 시간 히스토그램
- 일치율/전환율 게이지
- Circuit Breaker 상태

---

### 배포 아키텍처 (온프레미스 - 가상IP 기반)

```
┌─────────────────────────────────────────────────────────────────┐
│                   Load Balancer (L4/L7)                         │
│  Backend Pool: 192.168.1.101:10019, 192.168.1.102:10019,       │
│                192.168.1.103:10019, 192.168.2.101:10019, ...   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
        ┌───────────────────┴───────────────────┐
        │                                       │
        ▼                                       ▼
┌──────────────────────────────┐    ┌──────────────────────────────┐
│   Server 1 (Physical)        │    │   Server 2 (Physical)        │
│   Hostname: app-server-01    │    │   Hostname: app-server-02    │
│                              │    │                              │
│  ┌────────────────────────┐  │    │  ┌────────────────────────┐  │
│  │ Virtual IP Aliases     │  │    │  │ Virtual IP Aliases     │  │
│  │                        │  │    │  │                        │  │
│  │ eth0:0 → 192.168.1.101 │  │    │  │ eth0:0 → 192.168.2.101 │  │
│  │ eth0:1 → 192.168.1.102 │  │    │  │ eth0:1 → 192.168.2.102 │  │
│  │ eth0:2 → 192.168.1.103 │  │    │  │ eth0:2 → 192.168.2.103 │  │
│  └────────────────────────┘  │    │  └────────────────────────┘  │
│                              │    │                              │
│  ┌────────────────────────┐  │    │  ┌────────────────────────┐  │
│  │ Systemd Services       │  │    │  │ Systemd Services       │  │
│  │                        │  │    │  │                        │  │
│  │ api-bridge-1.service   │  │    │  │ api-bridge-1.service   │  │
│  │  ├─ VIP: .101:10019   │  │    │  │  ├─ VIP: .101:10019   │  │
│  │  └─ PID: 12345        │  │    │  │  └─ PID: 12345        │  │
│  │                        │  │    │  │                        │  │
│  │ api-bridge-2.service   │  │    │  │ api-bridge-2.service   │  │
│  │  ├─ VIP: .102:10019   │  │    │  │  ├─ VIP: .102:10019   │  │
│  │  └─ PID: 12346        │  │    │  │  └─ PID: 12346        │  │
│  │                        │  │    │  │                        │  │
│  │ api-bridge-3.service   │  │    │  │ api-bridge-3.service   │  │
│  │  ├─ VIP: .103:10019   │  │    │  │  ├─ VIP: .103:10019   │  │
│  │  └─ PID: 12347        │  │    │  │  └─ PID: 12347        │  │
│  └────────────────────────┘  │    │  └────────────────────────┘  │
│                              │    │                              │
│  ┌────────────────────────┐  │    │  ┌────────────────────────┐  │
│  │ Application Files      │  │    │  │ Application Files      │  │
│  │                        │  │    │  │                        │  │
│  │ /opt/api-bridge/       │  │    │  │ /opt/api-bridge/       │  │
│  │  ├─ bin/               │  │    │  │  ├─ bin/               │  │
│  │  │  └─ api-bridge     │  │    │  │  │  └─ api-bridge     │  │
│  │  ├─ config/           │  │    │  │  ├─ config/           │  │
│  │  │  ├─ config-1.yaml │  │    │  │  │  ├─ config-1.yaml │  │
│  │  │  ├─ config-2.yaml │  │    │  │  │  ├─ config-2.yaml │  │
│  │  │  └─ config-3.yaml │  │    │  │  │  └─ config-3.yaml │  │
│  │  └─ logs/             │  │    │  │  └─ logs/             │  │
│  │     ├─ instance-1/    │  │    │  │     ├─ instance-1/    │  │
│  │     ├─ instance-2/    │  │    │  │     ├─ instance-2/    │  │
│  │     └─ instance-3/    │  │    │  │     └─ instance-3/    │  │
│  └────────────────────────┘  │    │  └────────────────────────┘  │
│                              │    │                              │
│  ┌────────────────────────┐  │    │  ┌────────────────────────┐  │
│  │ Monitoring Agents      │  │    │  │ Monitoring Agents      │  │
│  │                        │  │    │  │                        │  │
│  │ - Node Exporter        │  │    │  │ - Node Exporter        │  │
│  │ - Filebeat             │  │    │  │ - Filebeat             │  │
│  └────────────────────────┘  │    │  └────────────────────────┘  │
└──────────────────────────────┘    └──────────────────────────────┘
```

#### 가상IP 및 프로세스 관리

> **상세 가이드**: 가상IP 설정, Shell Script 구현, Go 애플리케이션 코드, 배포 스크립트는 [PROCESS_MANAGEMENT.md](./docs/PROCESS_MANAGEMENT.md#가상ip-설정-및-shell-script-구현) 참고

---

### 데이터 흐름

#### 시나리오 1: 병렬 호출 모드 (PARALLEL)

```
1. Client Request
   ↓
2. Load Balancer → Bridge Instance
   ↓
3. Routing Layer
   - 캐시에서 매핑 조회
   - Strategy = PARALLEL 확인
   ↓
4. Orchestration Layer
   - 고루틴 2개 생성 (레거시/모던)
   ↓
5. HTTP Client Layer
   ├─ 레거시 API 호출 (5초 타임아웃)
   └─ 모던 API 호출 (5초 타임아웃)
   ↓
6. 응답 수집 (채널)
   ↓
7. 클라이언트에 레거시 응답 즉시 반환
   ↓
8. [비동기] Comparison Engine
   - JSON Diff 수행
   - 일치율 계산 (예: 98.5%)
   ↓
9. [비동기] Decision Engine
   - 임계값 체크 (98.5% < 100%)
   - DB에 비교 결과 저장
   - Redis에 일치율 업데이트
   - Prometheus 메트릭 기록
```

#### 시나리오 2: 자동 전환 (100% 도달)

```
1. Comparison Engine에서 100% 일치 감지
   ↓
2. Decision Engine
   - Threshold Evaluator가 전환 조건 확인
   - 최근 N개 요청 모두 100% 확인
   ↓
3. Transition Controller
   - DB 업데이트: Strategy = MODERN_ONLY
   - 캐시 무효화
   - 전환 이벤트 발행
   ↓
4. 이후 요청
   - Routing Layer에서 MODERN_ONLY 확인
   - 모던 API만 호출
   - 비교 로직 스킵
```

---

### 확장 및 장애 대응

#### 수평 확장
- Stateless 설계로 인스턴스 추가만으로 확장
- 로드밸런서가 자동 분산

#### 장애 시나리오

**1. 레거시 API 장애**
```
Circuit Breaker Open
   ↓
모던 API만 호출
   ↓
응답 반환 (Fallback)
```

**2. 모던 API 장애**
```
Circuit Breaker Open
   ↓
레거시 API만 호출
   ↓
비교 스킵
```

**3. DB 장애**
```
캐시에서 매핑 정보 제공
   ↓
읽기는 정상 동작
   ↓
쓰기는 실패 로그 기록
```

**4. Redis 장애**
```
DB 직접 조회로 Fallback
   ↓
성능 저하 알림
   ↓
계속 서비스 제공
```

