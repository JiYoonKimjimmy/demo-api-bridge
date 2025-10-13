# API Bridge 테스트 스크립트 (PowerShell)

Write-Host "Running tests..." -ForegroundColor Green

# 테스트 실행
go test -v -race -coverprofile=coverage.out ./...

if ($LASTEXITCODE -eq 0) {
    Write-Host "`nTests passed!" -ForegroundColor Green
    
    # 커버리지 확인
    Write-Host "`nGenerating coverage report..." -ForegroundColor Cyan
    go tool cover -func=coverage.out
} else {
    Write-Host "`nTests failed!" -ForegroundColor Red
    exit 1
}

