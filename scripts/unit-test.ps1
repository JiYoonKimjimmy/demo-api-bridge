# API Bridge Unit Test Script (PowerShell)
# Runs Go unit tests with coverage analysis

Write-Host "Running unit tests..." -ForegroundColor Green

# 테스트 실행
go test -v -race -coverprofile=coverage.out ./...

if ($LASTEXITCODE -eq 0) {
    Write-Host "`nUnit tests passed!" -ForegroundColor Green
    
    # 커버리지 확인
    Write-Host "`nGenerating coverage report..." -ForegroundColor Cyan
    go tool cover -func=coverage.out
} else {
    Write-Host "`nUnit tests failed!" -ForegroundColor Red
    exit 1
}
