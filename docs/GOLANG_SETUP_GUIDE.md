# Go 개발 환경 설정 가이드 (Windows)

API Bridge 시스템 개발을 위한 Go 언어 설치 및 환경 설정 가이드입니다.

---

## 📋 목차

1. [시스템 요구사항](#시스템-요구사항)
2. [Go 설치](#go-설치)
3. [환경 변수 설정](#환경-변수-설정)
4. [설치 확인](#설치-확인)
5. [IDE 설정](#ide-설정)
6. [프로젝트 초기화](#프로젝트-초기화)
7. [유용한 도구 설치](#유용한-도구-설치)
8. [트러블슈팅](#트러블슈팅)

---

## 시스템 요구사항

- **OS**: Windows 10 이상 (64-bit)
- **메모리**: 최소 4GB RAM (권장 8GB 이상)
- **디스크**: 최소 500MB 여유 공간
- **필수**: PowerShell 5.1 이상 또는 Windows Terminal

---

## Go 설치

### 1. Go 다운로드

#### 방법 1: 공식 웹사이트 (권장)

1. [Go 공식 다운로드 페이지](https://go.dev/dl/) 접속
2. **Windows 64-bit MSI 인스톨러** 다운로드
   - 권장 버전: **Go 1.21.x** 이상 (API Bridge 프로젝트 요구사항)
   - 파일명 예시: `go1.21.x.windows-amd64.msi`

#### 방법 2: Chocolatey (선택사항)

```powershell
# PowerShell (관리자 권한)
choco install golang --version=1.21.0
```

### 2. Go 설치 실행

1. 다운로드한 MSI 파일 실행
2. 설치 마법사 진행:
   - **설치 경로**: 기본값 사용 권장 (`C:\Program Files\Go`)
   - **Add to PATH**: 자동으로 추가됨
3. "Install" 클릭 후 완료

> **💡 Tip**: 설치 경로에 한글이나 공백이 포함되지 않도록 주의하세요.

---

## 환경 변수 설정

Go 설치 후 추가로 설정해야 할 환경 변수입니다.

### 1. 환경 변수 확인

```powershell
# PowerShell에서 확인
$env:GOROOT
# 출력 예: C:\Program Files\Go

$env:PATH
# 출력에 C:\Program Files\Go\bin 포함되어 있어야 함
```

### 2. GOPATH 설정 (선택사항)

Go 1.11 이후부터는 Go Modules를 사용하므로 GOPATH 설정은 선택사항입니다.
하지만 전역 도구 설치를 위해 설정을 권장합니다.

#### 수동 설정 방법

1. **시스템 환경 변수 편집** 열기:
   - `Win + X` → "시스템" → "고급 시스템 설정" → "환경 변수"
   
2. **사용자 변수** 추가:
   ```
   변수 이름: GOPATH
   변수 값: C:\Users\{사용자명}\go
   ```

3. **PATH 변수에 추가**:
   ```
   C:\Users\{사용자명}\go\bin
   ```

#### PowerShell로 설정 (영구적)

```powershell
# 사용자 프로필에 영구적으로 추가
[System.Environment]::SetEnvironmentVariable("GOPATH", "$HOME\go", "User")
[System.Environment]::SetEnvironmentVariable("PATH", "$env:PATH;$HOME\go\bin", "User")

# 현재 세션에 즉시 반영
$env:GOPATH = "$HOME\go"
$env:PATH += ";$HOME\go\bin"
```

### 3. 환경 변수 적용

설정 후 PowerShell 또는 터미널을 **재시작**해야 적용됩니다.

---

## 설치 확인

### 1. Go 버전 확인

```powershell
go version
# 출력 예: go version go1.21.5 windows/amd64
```

### 2. Go 환경 정보 확인

```powershell
go env

# 주요 확인 항목:
# GOROOT=C:\Program Files\Go
# GOPATH=C:\Users\{사용자명}\go
# GOMODCACHE=C:\Users\{사용자명}\go\pkg\mod
# GOOS=windows
# GOARCH=amd64
```

### 3. 간단한 테스트 프로그램 실행

```powershell
# 임시 디렉토리에서 테스트
mkdir $env:TEMP\go-test
cd $env:TEMP\go-test

# 간단한 Go 파일 생성
@"
package main
import "fmt"
func main() {
    fmt.Println("Hello, API Bridge!")
}
"@ | Out-File -Encoding UTF8 main.go

# 실행
go run main.go
# 출력: Hello, API Bridge!

# 빌드
go build -o hello.exe main.go
.\hello.exe
# 출력: Hello, API Bridge!

# 정리
cd ..
Remove-Item -Recurse -Force go-test
```

---

## IDE 설정

### Visual Studio Code (권장)

#### 1. VS Code 설치
- [VS Code 다운로드](https://code.visualstudio.com/)

#### 2. Go 확장 설치

1. VS Code 실행
2. Extensions (`Ctrl + Shift + X`)
3. "Go" 검색 → **Go for Visual Studio Code** (Go Team at Google) 설치

#### 3. Go Tools 설치

VS Code에서 Go 파일을 처음 열면 자동으로 필요한 도구 설치를 제안합니다.
또는 수동 설치:

```powershell
# Command Palette (Ctrl+Shift+P) → "Go: Install/Update Tools" → "Select All"
```

필수 도구:
- `gopls`: Go 언어 서버
- `go-outline`: 문서 아웃라인
- `dlv`: 디버거
- `staticcheck`: 정적 분석 도구

#### 4. VS Code 설정 (선택사항)

`.vscode/settings.json` 예시:

```json
{
  "go.toolsManagement.autoUpdate": true,
  "go.useLanguageServer": true,
  "go.lintTool": "staticcheck",
  "go.lintOnSave": "workspace",
  "go.formatTool": "gofmt",
  "editor.formatOnSave": true,
  "go.coverOnSave": false,
  "[go]": {
    "editor.tabSize": 4,
    "editor.insertSpaces": false,
    "editor.defaultFormatter": "golang.go"
  },
  "go.testFlags": ["-v", "-count=1"],
  "go.buildFlags": ["-v"],
  "go.testTimeout": "30s"
}
```

### GoLand (JetBrains, 선택사항)

Go 전용 IDE로 고급 기능을 제공합니다.
- [GoLand 다운로드](https://www.jetbrains.com/go/)
- 유료 (30일 무료 평가판)

---

## 프로젝트 초기화

### 1. 프로젝트 디렉토리 생성

```powershell
# 작업 디렉토리로 이동
cd D:\00_kjy\workspace

# API Bridge 프로젝트 클론 (이미 있다면 스킵)
# git clone <repository-url> demo-api-bridge
cd demo-api-bridge
```

### 2. Go 모듈 초기화

```powershell
# go.mod 파일 생성
go mod init github.com/yourusername/demo-api-bridge

# 또는 내부 프로젝트인 경우
go mod init demo-api-bridge
```

### 3. 프로젝트 구조 생성

```powershell
# API Bridge 프로젝트 구조
mkdir -p cmd/api-bridge
mkdir -p internal/adapter/inbound/http
mkdir -p internal/adapter/outbound/httpclient
mkdir -p internal/adapter/outbound/database
mkdir -p internal/adapter/outbound/cache
mkdir -p internal/core/domain
mkdir -p internal/core/port
mkdir -p internal/core/service
mkdir -p pkg/logger
mkdir -p pkg/metrics
mkdir -p config
mkdir -p scripts
mkdir -p test
```

### 4. 기본 의존성 설치

```powershell
# Gin 프레임워크
go get -u github.com/gin-gonic/gin

# 로깅
go get -u go.uber.org/zap

# 설정 관리
go get -u github.com/spf13/viper

# Oracle DB 드라이버
go get -u github.com/godror/godror

# Redis
go get -u github.com/redis/go-redis/v9

# Prometheus
go get -u github.com/prometheus/client_golang/prometheus

# 의존성 정리
go mod tidy
```

### 5. 첫 번째 코드 작성

`cmd/api-bridge/main.go`:

```go
package main

import (
    "fmt"
    "log"

    "github.com/gin-gonic/gin"
)

func main() {
    fmt.Println("Starting API Bridge System...")

    r := gin.Default()
    
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status": "ok",
            "service": "api-bridge",
        })
    })

    if err := r.Run(":10019"); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

### 6. 실행 및 테스트

```powershell
# 개발 모드로 실행
go run cmd/api-bridge/main.go

# 다른 터미널에서 테스트
curl http://localhost:10019/health
# 또는 PowerShell
Invoke-WebRequest -Uri http://localhost:10019/health
```

---

## 유용한 도구 설치

### 1. 코드 품질 도구

```powershell
# gofmt: 코드 포맷팅 (기본 포함)
go install golang.org/x/tools/cmd/goimports@latest

# golangci-lint: 통합 린터 (권장)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# staticcheck: 정적 분석
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### 2. 개발 도구

```powershell
# air: 핫 리로드 (개발 시 자동 재시작)
go install github.com/cosmtrek/air@latest

# swag: Swagger 문서 생성
go install github.com/swaggo/swag/cmd/swag@latest

# mockgen: Mock 생성
go install go.uber.org/mock/mockgen@latest
```

### 3. 디버깅 도구

```powershell
# delve: Go 디버거
go install github.com/go-delve/delve/cmd/dlv@latest
```

---

## 트러블슈팅

### 문제 1: "go: command not found"

**원인**: PATH 환경 변수가 설정되지 않음

**해결**:
```powershell
# PATH 확인
$env:PATH -split ';' | Select-String "Go"

# 없으면 수동 추가 (현재 세션)
$env:PATH += ";C:\Program Files\Go\bin"

# 영구적 추가 (관리자 권한)
[System.Environment]::SetEnvironmentVariable("PATH", "$env:PATH;C:\Program Files\Go\bin", "Machine")
```

### 문제 2: "go get" 실패 (프록시 오류)

**원인**: 회사 방화벽/프록시

**해결**:
```powershell
# Go 프록시 설정
go env -w GOPROXY=https://goproxy.io,direct
# 또는 한국 프록시
go env -w GOPROXY=https://goproxy.kr,direct

# 비공개 모듈 설정 (필요시)
go env -w GOPRIVATE=github.com/yourcompany/*
```

### 문제 3: "gcc: command not found" (CGO 사용 시)

**원인**: C 컴파일러 필요 (Oracle 드라이버 등)

**해결**:
```powershell
# 방법 1: TDM-GCC 설치 (권장)
# https://jmeubank.github.io/tdm-gcc/download/

# 방법 2: CGO 비활성화 (가능한 경우)
go env -w CGO_ENABLED=0
```

### 문제 4: 모듈 의존성 오류

**해결**:
```powershell
# 의존성 정리
go mod tidy

# 모듈 캐시 정리
go clean -modcache

# vendor 디렉토리 사용 (오프라인 환경)
go mod vendor
go build -mod=vendor
```

### 문제 5: 포트 충돌 (10019 포트 사용 중)

**해결**:
```powershell
# 포트 사용 중인 프로세스 확인
netstat -ano | findstr :10019

# 프로세스 종료 (PID 확인 후)
taskkill /PID <PID> /F
```

---

## 추가 참고 자료

### 공식 문서
- [Go 공식 문서](https://go.dev/doc/)
- [Go Tour (튜토리얼)](https://go.dev/tour/)
- [Effective Go](https://go.dev/doc/effective_go)

### 학습 자료
- [Go by Example](https://gobyexample.com/)
- [Go 표준 라이브러리](https://pkg.go.dev/std)

### API Bridge 관련
- [Gin 프레임워크](https://gin-gonic.com/docs/)
- [Oracle godror 드라이버](https://github.com/godror/godror)
- [go-redis](https://redis.uptrace.dev/)

---

## 다음 단계

Go 설치가 완료되었다면:

1. **[구현 가이드](./IMPLEMENTATION_GUIDE.md)** 참고하여 계층별 구현 시작
2. **[배포 가이드](./DEPLOYMENT_GUIDE.md)** 참고하여 배포 환경 준비
3. **[개발 계획](./DEPLOYMENT_PLAN.md)** 참고하여 스프린트 계획 수립

---

**작성일**: 2025-09-30  
**대상 버전**: Go 1.21+  
**업데이트**: 프로젝트 진행 중 필요시 업데이트 예정

