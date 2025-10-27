# API Bridge Shutdown Script (PowerShell)
# This script uses the API endpoint to gracefully shutdown the service

param(
    [string]$Port = "10019",
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
    Write-Host "Usage: .\scripts\shutdown.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Port PORT        Target port (default: 10019)"
    Write-Host "  -Help             Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\scripts\shutdown.ps1                    # Graceful shutdown on default port"
    Write-Host "  .\scripts\shutdown.ps1 -Port 8080         # Graceful shutdown on port 8080"
    Write-Host ""
}

if ($Help) {
    Show-Usage
    exit 0
}

# Function to check if service is running
function Test-ServiceRunning {
    param([string]$Port)

    try {
        $response = Invoke-WebRequest -Uri "http://localhost:$Port/management/health" -Method GET -TimeoutSec 5 -ErrorAction SilentlyContinue
        return $response.StatusCode -eq 200
    }
    catch {
        return $false
    }
}

# Function to send graceful shutdown request
function Send-GracefulShutdown {
    param([string]$Port)

    try {
        Write-Info "Sending graceful shutdown request to http://localhost:$Port/management/v1/shutdown..."

        $response = Invoke-WebRequest -Uri "http://localhost:$Port/management/v1/shutdown" -Method POST -TimeoutSec 10
        
        if ($response.StatusCode -eq 200) {
            $responseBody = $response.Content | ConvertFrom-Json
            Write-Success "Graceful shutdown initiated successfully"
            Write-Host "Response: $($responseBody.message)"
            Write-Host "Timestamp: $($responseBody.timestamp)"
            return $true
        } else {
            Write-Warning "Unexpected response code: $($response.StatusCode)"
            return $false
        }
    }
    catch {
        Write-Error "Failed to send graceful shutdown request: $($_.Exception.Message)"
        return $false
    }
}

# Function to wait for service to stop
function Wait-ForServiceStop {
    param([string]$Port, [int]$TimeoutSeconds = 30)
    
    Write-Info "Waiting for service to stop (timeout: $TimeoutSeconds seconds)..."
    
    $elapsed = 0
    while ($elapsed -lt $TimeoutSeconds) {
        if (-not (Test-ServiceRunning -Port $Port)) {
            Write-Success "Service has stopped successfully"
            return $true
        }
        
        Start-Sleep -Seconds 1
        $elapsed++
        
        if ($elapsed % 5 -eq 0) {
            Write-Info "Still waiting... ($elapsed/$TimeoutSeconds seconds)"
        }
    }
    
    Write-Warning "Service did not stop within $TimeoutSeconds seconds"
    return $false
}

# Main function
function Main {
    Write-Host "=================================="
    Write-Host "API Bridge Shutdown Script"
    Write-Host "=================================="
    
    Write-Info "Checking if service is running on port $Port..."
    
    if (-not (Test-ServiceRunning -Port $Port)) {
        Write-Warning "Service is not running on port $Port"
        return
    }
    
    Write-Success "Service is running on port $Port"
    
    # Send graceful shutdown request
    if (Send-GracefulShutdown -Port $Port) {
        # Wait for service to stop
        if (Wait-ForServiceStop -Port $Port) {
            Write-Success "Graceful shutdown completed successfully"
        } else {
            Write-Warning "Graceful shutdown may not have completed properly"
            Write-Info "You may need to use force shutdown methods"
        }
    } else {
        Write-Error "Failed to initiate graceful shutdown"
        Write-Info "You may need to use force shutdown methods"
    }
    
    Write-Host "=================================="
}

# Run main function
Main
