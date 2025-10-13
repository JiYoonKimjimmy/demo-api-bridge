# API Bridge 빌드 스크립트 (PowerShell)

Write-Host "Building API Bridge..." -ForegroundColor Green

# 빌드 디렉토리 생성
if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
}

# 빌드 실행
$output = "bin\api-bridge.exe"
go build -ldflags="-s -w" -o $output cmd\api-bridge\main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful: $output" -ForegroundColor Green
    
    # 파일 크기 확인
    $size = (Get-Item $output).Length / 1MB
    Write-Host "Binary size: $([math]::Round($size, 2)) MB" -ForegroundColor Cyan
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

