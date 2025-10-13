# Quick Start Guide

API Bridge 프로젝트를 빠르게 시작하기 위한 가이드입니다.

## ⚡ 5분 안에 시작하기

### 1. Go 설치 확인

```powershell
go version
# go version go1.21.x 이상이어야 함
```

Go가 설치되어 있지 않다면 [GOLANG_SETUP_GUIDE.md](./GOLANG_SETUP_GUIDE.md)를 참고하세요.

### 2. 프로젝트 클론 및 의존성 설치

```powershell
# 프로젝트 디렉토리로 이동
cd demo-api-bridge

# 의존성 다운로드
go mod download

# 의존성 정리
go mod tidy
```

### 3. 애플리케이션 실행

#### 방법 1: 직접 실행 (권장)

```powershell
go run cmd/api-bridge/main.go
```

#### 방법 2: PowerShell 스크립트 사용

```powershell
.\scripts\run.ps1
```

#### 방법 3: 빌드 후 실행

```powershell
# 빌드
go build -o bin/api-bridge.exe cmd/api-bridge/main.go

# 실행
.\bin\api-bridge.exe
```

#### 방법 4: Makefile 사용

```powershell
make run-direct
```

### 4. 서버 확인

다른 터미널에서 헬스체크 엔드포인트를 호출합니다:

```powershell
# PowerShell
Invoke-WebRequest -Uri http://localhost:10019/health

# 또는 curl
curl http://localhost:10019/health
```

예상 응답:
```json
{
  "status": "ok",
  "service": "api-bridge",
  "version": "0.1.0"
}
```

## 🔧 개발 환경 설정

### VS Code 설정

1. **Go 확장 설치**
   - Extensions (`Ctrl + Shift + X`)
   - "Go" 검색 → 설치

2. **Go Tools 설치**
   - Command Palette (`Ctrl + Shift + P`)
   - "Go: Install/Update Tools" → "Select All"

### 유용한 도구 설치

```powershell
# 핫 리로드 (권장)
go install github.com/cosmtrek/air@latest

# 린터
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 코드 포맷팅
go install golang.org/x/tools/cmd/goimports@latest
```

### Air를 사용한 핫 리로드

```powershell
# Air 실행 (코드 변경 시 자동 재시작)
air

# 또는
make run
```

## 📡 API 테스트

### 기본 엔드포인트

#### 1. Health Check
```powershell
curl http://localhost:10019/health
```

#### 2. Readiness Check
```powershell
curl http://localhost:10019/ready
```

#### 3. Status
```powershell
curl http://localhost:10019/api/v1/status
```

### PowerShell에서 테스트

```powershell
# GET 요청
$response = Invoke-WebRequest -Uri http://localhost:10019/health
$response.Content | ConvertFrom-Json

# POST 요청 (예시)
$body = @{
    key = "value"
} | ConvertTo-Json

Invoke-WebRequest -Uri http://localhost:10019/api/v1/endpoint `
    -Method POST `
    -ContentType "application/json" `
    -Body $body
```

## 🧪 테스트 실행

### 전체 테스트

```powershell
go test ./...
```

### 커버리지 포함

```powershell
go test -cover ./...
```

### 상세 출력

```powershell
go test -v ./...
```

### 스크립트 사용

```powershell
.\scripts\test.ps1
```

## 🏗️ 빌드

### 개발 빌드

```powershell
go build -o bin/api-bridge.exe cmd/api-bridge/main.go
```

### 프로덕션 빌드 (최적화)

```powershell
go build -ldflags="-s -w" -o bin/api-bridge.exe cmd/api-bridge/main.go
```

### 스크립트 사용

```powershell
.\scripts\build.ps1
```

## 📝 설정 파일

### 1. 설정 파일 생성

```powershell
# 예시 파일 복사
cp config/config.example.yaml config/config.yaml
```

### 2. 설정 파일 수정

`config/config.yaml` 파일을 열어 환경에 맞게 수정:

```yaml
server:
  port: 10019
  mode: debug

log:
  level: debug
  format: console
```

### 3. 환경 변수 설정 (선택)

```powershell
# 포트 변경
$env:PORT = "8080"

# Gin 모드 변경
$env:GIN_MODE = "release"
```

## 🐛 디버깅

### VS Code 디버깅 설정

`.vscode/launch.json` 파일 생성:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch API Bridge",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/api-bridge",
      "env": {
        "PORT": "10019"
      }
    }
  ]
}
```

`F5` 키를 눌러 디버깅 시작!

### 로그 레벨 조정

```powershell
# 더 상세한 로그를 보려면
$env:LOG_LEVEL = "debug"
go run cmd/api-bridge/main.go
```

## 🚀 다음 단계

### 1. 프로젝트 구조 이해
- [PROJECT_STRUCTURE.md](./PROJECT_STRUCTURE.md) - 디렉토리 구조 상세 설명

### 2. 아키텍처 학습
- [HEXAGONAL_ARCHITECTURE.md](./HEXAGONAL_ARCHITECTURE.md) - 헥사고날 아키텍처 개념

### 3. 개발 시작
- [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) - 계층별 구현 가이드

### 4. 배포 준비
- [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md) - 배포 가이드

## 💡 유용한 명령어 모음

```powershell
# 의존성 관리
go mod tidy                          # 의존성 정리
go mod download                      # 의존성 다운로드
go mod verify                        # 의존성 검증

# 코드 품질
gofmt -s -w .                        # 코드 포맷팅
go vet ./...                         # 정적 분석
golangci-lint run ./...              # 린트 실행

# 빌드 및 실행
go run cmd/api-bridge/main.go        # 직접 실행
go build -o bin/api-bridge.exe cmd/api-bridge/main.go  # 빌드
air                                  # 핫 리로드

# 테스트
go test ./...                        # 전체 테스트
go test -v ./...                     # 상세 테스트
go test -cover ./...                 # 커버리지 포함
go test -race ./...                  # Race condition 검사

# 정리
go clean                             # 빌드 캐시 정리
go clean -modcache                   # 모듈 캐시 정리
```

## ❓ 자주 묻는 질문 (FAQ)

### Q: 포트가 이미 사용 중이라는 오류가 발생해요

```powershell
# 포트 사용 중인 프로세스 확인
netstat -ano | findstr :10019

# 프로세스 종료 (PID 확인 후)
taskkill /PID <PID> /F

# 또는 다른 포트 사용
$env:PORT = "8080"
go run cmd/api-bridge/main.go
```

### Q: 의존성 다운로드가 실패해요

```powershell
# 프록시 설정
go env -w GOPROXY=https://goproxy.io,direct

# 또는 한국 프록시
go env -w GOPROXY=https://goproxy.kr,direct
```

### Q: Air가 설치되지 않아요

```powershell
# GOPATH/bin이 PATH에 있는지 확인
echo $env:PATH

# Air 재설치
go install github.com/cosmtrek/air@latest

# 직접 경로로 실행
$env:GOPATH\bin\air.exe
```

## 🆘 문제 해결

문제가 해결되지 않으면:

1. [트러블슈팅 가이드](./GOLANG_SETUP_GUIDE.md#트러블슈팅) 참고
2. 이슈 등록
3. 로그 파일 확인 (`logs/`)

---

**작성일**: 2025-10-13  
**업데이트**: 프로젝트 초기 구조 완성  
**소요 시간**: 설정 5분 + 학습 10분 = 총 15분

