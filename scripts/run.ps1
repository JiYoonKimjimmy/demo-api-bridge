# API Bridge 실행 스크립트 (PowerShell)

Write-Host "Starting API Bridge..." -ForegroundColor Green

# 환경 변수 설정
$env:PORT = "10019"
$env:GIN_MODE = "debug"

# 애플리케이션 실행
go run cmd/api-bridge/main.go

