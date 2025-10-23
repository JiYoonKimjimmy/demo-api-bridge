# Vegeta를 사용한 부하 테스트 스크립트

param(
    [string]$Target = "http://localhost:10019/api/users",
    [int]$Duration = 60,
    [int]$Rate = 1000,
    [string]$Method = "GET",
    [string]$Output = "results.txt"
)

Write-Host "Vegeta Load Testing" -ForegroundColor Green
Write-Host "=====================" -ForegroundColor Green

# Vegeta 설치 확인
$vegetaPath = Get-Command vegeta -ErrorAction SilentlyContinue
if (-not $vegetaPath) {
    Write-Host "Vegeta not found. Installing..." -ForegroundColor Yellow
    
    # Go가 설치되어 있는지 확인
    $goPath = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goPath) {
        Write-Host "Go not found. Please install Go first." -ForegroundColor Red
        exit 1
    }
    
    # Vegeta 설치
    go install github.com/tsenart/vegeta@latest
    
    # PATH에 GOPATH/bin 추가
    $gopath = go env GOPATH
    $env:PATH += ";$gopath\bin"
}

Write-Host "Target: $Target" -ForegroundColor Cyan
Write-Host "Duration: $Duration seconds" -ForegroundColor Cyan
Write-Host "Rate: $Rate requests/second" -ForegroundColor Cyan
Write-Host "Method: $Method" -ForegroundColor Cyan

# Vegeta 명령어 구성
$vegetaCmd = "echo `"$Method $Target`" | vegeta attack -duration=${Duration}s -rate=$Rate | vegeta report -type=text"

Write-Host "`nStarting load test..." -ForegroundColor Yellow

# 테스트 실행
Invoke-Expression $vegetaCmd

Write-Host "`nDetailed Results:" -ForegroundColor Green

# 상세 결과 생성
$detailedCmd = "echo `"$Method $Target`" | vegeta attack -duration=${Duration}s -rate=$Rate | vegeta report -type=hist[0,1ms,2ms,5ms,10ms,20ms,50ms,100ms,200ms,500ms,1s,2s,5s,10s]"
Invoke-Expression $detailedCmd

# 결과를 파일로 저장
Write-Host "`nSaving results to $Output..." -ForegroundColor Yellow
$saveCmd = "echo `"$Method $Target`" | vegeta attack -duration=${Duration}s -rate=$Rate | vegeta report -type=text > $Output"
Invoke-Expression $saveCmd

Write-Host "`nLoad test completed!" -ForegroundColor Green
Write-Host "Results saved to: $Output" -ForegroundColor Cyan

# 성능 목표 확인
Write-Host "`nPerformance Targets:" -ForegroundColor Yellow
Write-Host "- Target RPS: 5,000 (current: $Rate)" -ForegroundColor White
Write-Host "- Target Latency (p95): < 30ms" -ForegroundColor White
Write-Host "- Target Success Rate: > 99.9%" -ForegroundColor White

Write-Host "`nTo view results in real-time:" -ForegroundColor Cyan
Write-Host "echo `"$Method $Target`" | vegeta attack -duration=${Duration}s -rate=$Rate | vegeta report -type=text" -ForegroundColor Gray
