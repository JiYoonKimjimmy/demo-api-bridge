# API Bridge Service Start Script (PowerShell)
# This script starts the API Bridge service with proper configuration

param(
    [string]$Port = "10019",
    [string]$TargetHost = "localhost",
    [switch]$Verbose,
    [switch]$Help
)

# Colors for output
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Blue"
    White = "White"
}

# Function to print colored output
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor $Colors.Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor $Colors.Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor $Colors.Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Colors.Red
}

# Function to show usage
function Show-Usage {
    Write-Host "Usage: .\scripts\start.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Port PORT        Target port (default: 10019)"
    Write-Host "  -Host HOST        Target host (default: localhost)"
    Write-Host "  -Verbose          Show detailed output"
    Write-Host "  -Help             Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\scripts\start.ps1                    # Start with default settings"
    Write-Host "  .\scripts\start.ps1 -Port 8080         # Start on port 8080"
    Write-Host "  .\scripts\start.ps1 -Verbose           # Show detailed output"
    Write-Host ""
}

if ($Help) {
    Show-Usage
    exit 0
}

# Check if Go is installed
function Test-Go {
    try {
        $goVersion = go version
        Write-Info "Found Go: $goVersion"
        return $true
    }
    catch {
        Write-Error "Go is not installed or not in PATH"
        Write-Info "Please install Go 1.25.1 or later"
        return $false
    }
}

# Check if required directories exist
function Test-ProjectStructure {
    Write-Info "Checking project structure..."
    
    if (-not (Test-Path "go.mod")) {
        Write-Error "go.mod not found. Please run this script from the project root directory."
        exit 1
    }
    
    if (-not (Test-Path "cmd/api-bridge/main.go")) {
        Write-Error "Main application file not found: cmd/api-bridge/main.go"
        exit 1
    }
    
    if (-not (Test-Path "config/config.yaml")) {
        Write-Warning "config/config.yaml not found. Using default configuration."
    }
    
    Write-Success "Project structure validated"
}

# Download dependencies
function Invoke-DownloadDependencies {
    Write-Info "Downloading Go dependencies..."
    try {
        go mod download
        Write-Success "Dependencies downloaded successfully"
    }
    catch {
        Write-Error "Failed to download dependencies"
        exit 1
    }
}


# Build the application
function Invoke-BuildApplication {
    Write-Info "Building the application..."
    try {
        # 기존 바이너리 삭제 (캐시 제거)
        if (Test-Path "bin/api-bridge.exe") {
            Write-Info "Removing existing binary to ensure clean build..."
            
            # 실행 중인 프로세스 종료
            $processes = Get-Process -Name "api-bridge" -ErrorAction SilentlyContinue
            if ($processes) {
                Write-Info "Stopping existing api-bridge processes..."
                $processes | Stop-Process -Force
                Start-Sleep -Seconds 2
            }
            
            # 바이너리 파일 삭제
            try {
                Remove-Item "bin/api-bridge.exe" -Force
                Write-Info "Existing binary removed successfully"
            }
            catch {
                Write-Warning "Could not remove existing binary: $($_.Exception.Message)"
                Write-Info "Continuing with build..."
            }
        }
        
        # 새로 빌드
        go build -o bin/api-bridge.exe cmd/api-bridge/main.go
        Write-Success "Application built successfully"
    }
    catch {
        Write-Error "Failed to build application"
        exit 1
    }
}

# Create necessary directories
function New-Directories {
    Write-Info "Creating necessary directories..."
    New-Item -ItemType Directory -Force -Path "bin" | Out-Null
    New-Item -ItemType Directory -Force -Path "logs" | Out-Null
    Write-Success "Directories created"
}

# Set environment variables
function Set-Environment {
    Write-Info "Setting environment variables..."
    
    # Set environment variables
    $env:PORT = $Port
    $env:GIN_MODE = "release"
    $env:TZ = "Asia/Seoul"
    
    Write-Info "PORT: $env:PORT"
    Write-Info "GIN_MODE: $env:GIN_MODE"
    Write-Info "TZ: $env:TZ"
}

# Start the application
function Start-Application {
    Write-Info "Starting API Bridge Service..."
    Write-Info "Service will be available at: http://$TargetHost`:$Port"
    Write-Host ""
    Write-Info "Available endpoints:"
    Write-Info "  ANY  /api/*path                 - API Bridge (handles all /api/* requests)"
    Write-Info "  GET  /management/health         - Health check"
    Write-Info "  GET  /management/ready          - Readiness check"
    Write-Info "  GET  /management/v1/status      - Service status"
    Write-Info "  GET  /management/metrics        - Prometheus metrics"
    Write-Host ""
    Write-Info "Press Ctrl+C to stop the service"
    Write-Host "=================================="
    
    # Start the application
    if (Test-Path "bin/api-bridge.exe") {
        # Use built binary
        & ".\bin\api-bridge.exe"
    }
    else {
        # Run directly with go run
        go run cmd/api-bridge/main.go
    }
}

# Cleanup function
function Invoke-Cleanup {
    Write-Info "Shutting down API Bridge Service..."
    Write-Success "Service stopped"
}

# Set trap for cleanup
$null = Register-EngineEvent PowerShell.Exiting -Action { Invoke-Cleanup }

# Main execution
function Main {
    Write-Host "=================================="
    Write-Host "API Bridge Service Startup Script"
    Write-Host "=================================="
    
    if (-not (Test-Go)) { exit 1 }
    Test-ProjectStructure
    New-Directories
    Invoke-DownloadDependencies
    Invoke-BuildApplication
    Set-Environment
    
    Write-Success "All checks passed. Starting service..."
    Write-Host "=================================="
    
    Start-Application
}

# Run main function
Main
