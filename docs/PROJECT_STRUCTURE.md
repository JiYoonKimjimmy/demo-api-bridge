# 프로젝트 구조 문서

## 📁 전체 디렉토리 구조

```
demo-api-bridge/
├── cmd/                          # 애플리케이션 진입점
│   └── api-bridge/
│       └── main.go              # 메인 애플리케이션
│
├── internal/                     # 내부 애플리케이션 코드
│   ├── adapter/                 # 어댑터 레이어
│   │   ├── inbound/            # 인바운드 어댑터 (외부 → 내부)
│   │   │   └── http/           # HTTP API 핸들러
│   │   │       ├── handler.go  # 요청 핸들러
│   │   │       ├── middleware.go # 미들웨어
│   │   │       └── dto.go      # DTO (Data Transfer Object)
│   │   │
│   │   └── outbound/           # 아웃바운드 어댑터 (내부 → 외부)
│   │       ├── httpclient/     # 외부 API 클라이언트
│   │       │   ├── client.go   # HTTP 클라이언트 구현
│   │       │   └── mapper.go   # 응답 매핑
│   │       │
│   │       ├── database/       # 데이터베이스 어댑터
│   │       │   ├── oracle.go   # Oracle DB 연결
│   │       │   ├── repository.go # 레포지토리 구현
│   │       │   └── mapper.go   # DB 엔티티 매핑
│   │       │
│   │       └── cache/          # 캐시 어댑터
│   │           ├── redis.go    # Redis 연결
│   │           └── cache.go    # 캐시 레포지토리 구현
│   │
│   └── core/                    # 핵심 비즈니스 로직 (의존성 없음)
│       ├── domain/              # 도메인 모델
│       │   ├── entity.go       # 비즈니스 엔티티
│       │   ├── value_object.go # 값 객체
│       │   └── error.go        # 도메인 에러
│       │
│       ├── port/                # 포트 인터페이스
│       │   ├── inbound.go      # 인바운드 포트 (유즈케이스)
│       │   └── outbound.go     # 아웃바운드 포트 (레포지토리)
│       │
│       └── service/             # 비즈니스 로직 서비스
│           ├── service.go      # 서비스 구현
│           └── service_test.go # 서비스 테스트
│
├── pkg/                         # 공용 패키지 (다른 프로젝트에서도 사용 가능)
│   ├── logger/                  # 로깅 유틸리티
│   │   ├── logger.go           # 로거 인터페이스 및 구현
│   │   └── zap.go              # Zap 로거 래퍼
│   │
│   └── metrics/                 # 메트릭 수집
│       ├── metrics.go          # 메트릭 인터페이스
│       └── prometheus.go       # Prometheus 메트릭
│
├── config/                      # 설정 파일
│   ├── config.example.yaml     # 설정 파일 예시
│   └── config.yaml             # 실제 설정 (git에서 제외)
│
├── docs/                        # 문서
│   ├── HEXAGONAL_ARCHITECTURE.md
│   ├── IMPLEMENTATION_GUIDE.md
│   ├── DEPLOYMENT_GUIDE.md
│   ├── GOLANG_SETUP_GUIDE.md
│   ├── FRAMEWORK_COMPARISON.md
│   ├── DEPLOYMENT_PLAN.md
│   └── PROJECT_STRUCTURE.md    # 이 문서
│
├── scripts/                     # 유틸리티 스크립트
│   ├── build.ps1               # 빌드 스크립트
│   ├── run.ps1                 # 실행 스크립트
│   └── test.ps1                # 테스트 스크립트
│
├── test/                        # 통합 테스트
│   ├── integration/            # 통합 테스트
│   └── e2e/                    # E2E 테스트
│
├── .gitignore                   # Git 제외 파일 설정
├── .air.toml                    # Air (핫 리로드) 설정
├── go.mod                       # Go 모듈 정의
├── go.sum                       # 의존성 체크섬
├── Makefile                     # Make 명령어 정의
└── README.md                    # 프로젝트 소개
```

## 🏗️ 아키텍처 레이어 설명

### 1. Core (핵심 레이어)

**위치**: `internal/core/`

**특징**:
- 외부 의존성이 **전혀 없음**
- 순수한 비즈니스 로직만 포함
- 테스트가 가장 쉬움
- 가장 안정적인 레이어

**구성 요소**:
- **Domain**: 비즈니스 엔티티와 규칙
- **Port**: 인터페이스 정의 (의존성 역전)
- **Service**: 비즈니스 로직 구현

### 2. Adapter (어댑터 레이어)

**위치**: `internal/adapter/`

**특징**:
- Core를 외부 세계와 연결
- 기술 구현 세부사항 포함
- 교체 가능한 구조

#### 2.1 Inbound Adapter (인바운드 어댑터)

**역할**: 외부 요청 → 애플리케이션 내부

**예시**:
- HTTP REST API Handler
- gRPC Handler
- GraphQL Resolver
- Message Queue Consumer

**위치**: `internal/adapter/inbound/http/`

#### 2.2 Outbound Adapter (아웃바운드 어댑터)

**역할**: 애플리케이션 내부 → 외부 시스템

**예시**:
- Database Repository
- External API Client
- Cache Client
- Message Queue Producer

**위치**: `internal/adapter/outbound/{httpclient,database,cache}/`

### 3. Pkg (공용 패키지)

**위치**: `pkg/`

**특징**:
- 다른 프로젝트에서도 재사용 가능
- 비즈니스 로직과 무관한 유틸리티
- 외부에 공개 가능한 코드

**구성 요소**:
- Logger: 구조화된 로깅
- Metrics: 성능 모니터링
- Utils: 공용 유틸리티 함수

## 📦 의존성 흐름

```
┌─────────────────────────────────────────────┐
│          Inbound Adapter (HTTP)             │
│        (외부 요청을 받아 처리)                 │
└────────────────┬────────────────────────────┘
                 │ depends on
                 ↓
┌─────────────────────────────────────────────┐
│              Core (Service)                 │
│         (비즈니스 로직 실행)                   │
└────────────────┬────────────────────────────┘
                 │ depends on (interface)
                 ↓
┌─────────────────────────────────────────────┐
│          Outbound Adapter (DB, API)         │
│    (외부 시스템과 통신하여 데이터 처리)          │
└─────────────────────────────────────────────┘
```

**핵심 원칙**:
- Core는 **어떤 것에도 의존하지 않음**
- Adapter는 Core에 의존
- 의존성은 항상 **내부(Core)를 향함**

## 🔄 데이터 흐름 예시

### 사용자 요청 처리 흐름

```
1. HTTP Request
   ↓
2. [HTTP Handler] (Inbound Adapter)
   - 요청 파싱
   - DTO 변환
   ↓
3. [Service] (Core)
   - 비즈니스 로직 실행
   - Port 인터페이스 호출
   ↓
4. [Repository] (Outbound Adapter)
   - DB 쿼리 실행
   - 외부 API 호출
   - 캐시 조회
   ↓
5. [Service] (Core)
   - 결과 가공
   - 비즈니스 규칙 적용
   ↓
6. [HTTP Handler] (Inbound Adapter)
   - 응답 DTO 변환
   - JSON 응답 생성
   ↓
7. HTTP Response
```

## 📝 파일 명명 규칙

### Go 파일명
- **소문자 + 언더스코어**: `user_service.go`, `http_handler.go`
- **테스트 파일**: `{파일명}_test.go`
- **인터페이스 파일**: `port.go`, `interface.go`

### 디렉토리명
- **소문자 단수형**: `service`, `handler`, `domain`
- **복수형은 컬렉션일 때만**: `scripts`, `docs`

### 패키지명
- 디렉토리명과 동일
- 간결하고 명확하게

## 🧪 테스트 구조

```
테스트 레벨별 위치:

1. Unit Test (단위 테스트)
   - 위치: 각 파일과 동일한 디렉토리
   - 파일명: *_test.go
   - 예: internal/core/service/service_test.go

2. Integration Test (통합 테스트)
   - 위치: test/integration/
   - 여러 컴포넌트 간 상호작용 테스트

3. E2E Test (종단 간 테스트)
   - 위치: test/e2e/
   - 전체 시스템 흐름 테스트
```

## 🚀 빌드 결과물

```
build/
└── api-bridge.exe    # 실행 파일 (Windows)
└── api-bridge        # 실행 파일 (Linux/Mac)
```

## 📊 설정 파일 우선순위

```
1. 환경 변수 (최우선)
2. config/config.yaml (로컬 설정)
3. config/config.example.yaml (기본값)
```

## 🔒 보안 주의사항

### Git에서 제외해야 할 파일
- `config/config.yaml` (실제 설정 파일)
- `.env` (환경 변수)
- `*.log` (로그 파일)
- `bin/`, `tmp/` (빌드 산출물)

### 민감 정보 관리
- 비밀번호, API 키는 환경 변수로 관리
- 설정 파일에는 예시 값만 포함
- 프로덕션 설정은 별도 관리

## 📌 다음 단계

1. **구현 시작**: [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) 참고
2. **배포 준비**: [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md) 참고
3. **아키텍처 학습**: [HEXAGONAL_ARCHITECTURE.md](./HEXAGONAL_ARCHITECTURE.md) 참고

---

**작성일**: 2025-10-13  
**버전**: 1.0  
**업데이트**: 프로젝트 초기 구조 완성

