# API Bridge Profiling Script (Windows PowerShell)
# Usage: .\scripts\profile.ps1 [-Type cpu|mem|goroutine|all] [-Duration 30] [-Port 10019]

param(
    [string]$Type = "cpu",
    [int]$Duration = 30,
    [int]$Port = 10019,
    [string]$OutputDir = "profiling-results"
)

$ErrorActionPreference = "Stop"

# Color output function
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

Write-ColorOutput "========================================" "Cyan"
Write-ColorOutput "API Bridge Profiling Tool" "Cyan"
Write-ColorOutput "========================================" "Cyan"
Write-Host ""

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
    Write-ColorOutput "Output directory created: $OutputDir" "Green"
}

$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$baseUrl = "http://localhost:$Port"

# CPU profiling
function Start-CPUProfile {
    param([int]$Duration)
    
    Write-ColorOutput "`n[CPU Profiling]" "Yellow"
    Write-ColorOutput "Collection time: $Duration seconds" "Gray"
    
    $outputFile = "$OutputDir/cpu_profile_$timestamp.pprof"
    
    try {
        Write-ColorOutput "Collecting profile..." "Gray"
        Invoke-WebRequest -Uri "$baseUrl/debug/pprof/profile?seconds=$Duration" `
            -OutFile $outputFile -TimeoutSec ($Duration + 10)
        
        Write-ColorOutput "CPU profile saved: $outputFile" "Green"
        
        Write-ColorOutput "`nAnalysis commands:" "Cyan"
        Write-Host "  go tool pprof -http=:8081 $outputFile" -ForegroundColor White
        Write-Host "  or" -ForegroundColor Gray
        Write-Host "  go tool pprof $outputFile" -ForegroundColor White
        
    } catch {
        Write-ColorOutput "CPU profiling failed: $_" "Red"
    }
}

# Memory profiling
function Start-MemoryProfile {
    Write-ColorOutput "`n[Memory Profiling]" "Yellow"
    
    $outputFile = "$OutputDir/mem_profile_$timestamp.pprof"
    
    try {
        Write-ColorOutput "Collecting profile..." "Gray"
        Invoke-WebRequest -Uri "$baseUrl/debug/pprof/heap" `
            -OutFile $outputFile -TimeoutSec 10
        
        Write-ColorOutput "Memory profile saved: $outputFile" "Green"
        
        Write-ColorOutput "`nAnalysis commands:" "Cyan"
        Write-Host "  go tool pprof -http=:8082 $outputFile" -ForegroundColor White
        
    } catch {
        Write-ColorOutput "Memory profiling failed: $_" "Red"
    }
}

# Goroutine profiling
function Start-GoroutineProfile {
    Write-ColorOutput "`n[Goroutine Profiling]" "Yellow"
    
    $outputFile = "$OutputDir/goroutine_profile_$timestamp.pprof"
    
    try {
        Write-ColorOutput "Collecting profile..." "Gray"
        Invoke-WebRequest -Uri "$baseUrl/debug/pprof/goroutine" `
            -OutFile $outputFile -TimeoutSec 10
        
        Write-ColorOutput "Goroutine profile saved: $outputFile" "Green"
        
    } catch {
        Write-ColorOutput "Goroutine profiling failed: $_" "Red"
    }
}

Write-Host ""

# Execute profiling based on type
switch ($Type.ToLower()) {
    "cpu" {
        Start-CPUProfile -Duration $Duration
    }
    "mem" {
        Start-MemoryProfile
    }
    "memory" {
        Start-MemoryProfile
    }
    "goroutine" {
        Start-GoroutineProfile
    }
    "all" {
        Write-ColorOutput "Starting full profiling..." "Cyan"
        Start-CPUProfile -Duration $Duration
        Start-Sleep -Seconds 2
        Start-MemoryProfile
        Start-Sleep -Seconds 1
        Start-GoroutineProfile
    }
    default {
        Write-ColorOutput "Invalid profiling type: $Type" "Red"
        Write-ColorOutput "Available types: cpu, mem, goroutine, all" "Yellow"
        exit 1
    }
}

Write-Host ""
Write-ColorOutput "========================================" "Cyan"
Write-ColorOutput "Profiling completed!" "Green"
Write-ColorOutput "Results location: $OutputDir" "Cyan"
Write-ColorOutput "========================================" "Cyan"
Write-Host ""
Write-ColorOutput "Recommended workflow:" "Yellow"
Write-Host "1. Run load test: .\scripts\vegeta-load-test.ps1" -ForegroundColor Gray
Write-Host "2. Collect profiles: .\scripts\profile.ps1 -Type all" -ForegroundColor Gray
Write-Host "3. Analyze results: go tool pprof -http=:8081 <profile_file>" -ForegroundColor Gray
Write-Host ""
