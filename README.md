# Demo API Bridge

헥사고날 아키텍처 기반의 API Bridge 시스템입니다.

## 📋 프로젝트 개요

이 프로젝트는 외부 API와 내부 시스템(Oracle DB, Redis Cache) 간의 중계 역할을 하는 API Bridge 서비스입니다. 헥사고날 아키텍처(포트&어댑터)를 적용하여 유지보수성과 테스트 용이성을 극대화했습니다.

## 🏗️ 아키텍처

```
demo-api-bridge/
├── cmd/
│   └── api-bridge/          # 애플리케이션 진입점
│       └── main.go
├── internal/
│   ├── adapter/
│   │   ├── inbound/         # 인바운드 어댑터
│   │   │   └── http/        # HTTP API 핸들러
│   │   └── outbound/        # 아웃바운드 어댑터
│   │       ├── httpclient/  # 외부 API 클라이언트
│   │       ├── database/    # Oracle DB 어댑터
│   │       └── cache/       # Redis 캐시 어댑터
│   └── core/
│       ├── domain/          # 도메인 모델
│       ├── port/            # 포트 인터페이스
│       └── service/         # 비즈니스 로직
├── pkg/
│   ├── logger/              # 로깅 유틸리티
│   └── metrics/             # 모니터링 메트릭
├── config/                  # 설정 파일
├── docs/                    # 문서
├── scripts/                 # 유틸리티 스크립트
└── test/                    # 통합 테스트
```

## 🚀 시작하기

### 필수 요구사항

- Go 1.21 이상
- Oracle Database (선택)
- Redis (선택)

### 설치

1. 저장소 클론

```bash
git clone <repository-url>
cd demo-api-bridge
```

2. 의존성 설치

```bash
go mod download
```

3. 개발 도구 설치 (선택)

```bash
make install-tools
```

### 실행

#### 개발 모드 (핫 리로드)

```bash
make run
# 또는
air
```

#### 스크립트를 사용한 실행 (권장)

**Linux/macOS (Bash)**
```bash
# 서비스 시작
./start.sh

# 헬스 체크
./health.sh
```

**Windows (PowerShell)**
```powershell
# 서비스 시작
.\start.ps1

# 헬스 체크
.\health.ps1
```

#### 직접 실행

```bash
make run-direct
# 또는
go run cmd/api-bridge/main.go
```

#### 빌드 후 실행

```bash
make build
./bin/api-bridge.exe
```

### 스크립트 옵션

**start.sh / start.ps1**
- Linux/macOS: `./start.sh -p 8080`
- Windows: `.\start.ps1 -Port 8080`

**health.sh / health.ps1**
- Linux/macOS: `./health.sh -h localhost -p 10019 -v`
- Windows: `.\health.ps1 -TargetHost localhost -Port 10019 -Verbose`

## 🔧 설정

1. 설정 파일 복사

```bash
cp config/config.example.yaml config/config.yaml
```

2. `config/config.yaml` 파일을 환경에 맞게 수정

## 📚 API 엔드포인트

### Health Check

```bash
GET /health
```

응답:
```json
{
  "status": "ok",
  "service": "api-bridge",
  "version": "0.1.0"
}
```

### Readiness Check

```bash
GET /ready
```

### Status

```bash
GET /api/v1/status
```

## 🧪 테스트

```bash
# 전체 테스트 실행
make test

# 커버리지 확인
make test-coverage

# 린트 실행
make lint
```

## 📖 문서

- [헥사고날 아키텍처 가이드](./docs/HEXAGONAL_ARCHITECTURE.md)
- [구현 가이드](./docs/IMPLEMENTATION_GUIDE.md)
- [배포 가이드](./docs/DEPLOYMENT_GUIDE.md)
- [Go 개발 환경 설정](./docs/GOLANG_SETUP_GUIDE.md)
- [프레임워크 비교](./docs/FRAMEWORK_COMPARISON.md)

## 🛠️ 개발

### 코드 포맷팅

```bash
make fmt
```

### 의존성 정리

```bash
make tidy
```

### 빌드

```bash
make build
```

## 📊 모니터링

Prometheus 메트릭은 `/metrics` 엔드포인트에서 확인할 수 있습니다 (설정 시).

## 🔐 환경 변수

| 변수명 | 설명 | 기본값 |
|--------|------|--------|
| PORT | 서버 포트 | 10019 |
| GIN_MODE | Gin 모드 | release |

## 🤝 기여

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 라이선스

This project is licensed under the MIT License.

## 👥 작성자

- Backend Developer

## 📧 문의

프로젝트에 대한 문의사항이 있으시면 이슈를 등록해주세요.

---

**Last Updated**: 2025-10-13
**Version**: 0.1.0
