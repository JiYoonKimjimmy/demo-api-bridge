# API Bridge Health Check Script (PowerShell)
# This script tests the health endpoints of the API Bridge service

param(
    [string]$TargetHost = "localhost",
    [string]$Port = "10019",
    [int]$Timeout = 10,
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
    Write-Host "Usage: .\health.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Host HOST        Target host (default: localhost)"
    Write-Host "  -Port PORT        Target port (default: 10019)"
    Write-Host "  -Timeout SEC      Request timeout in seconds (default: 10)"
    Write-Host "  -Verbose          Show detailed response"
    Write-Host "  -Help             Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\health.ps1                          # Test localhost:10019"
    Write-Host "  .\health.ps1 -Host 192.168.1.100 -Port 8080  # Test specific host and port"
    Write-Host "  .\health.ps1 -Verbose                 # Show detailed response"
    Write-Host ""
}

if ($Help) {
    Show-Usage
    exit 0
}

# Base URL
$BaseUrl = "http://$TargetHost`:$Port"

# Function to test an endpoint
function Test-Endpoint {
    param(
        [string]$Endpoint,
        [int]$ExpectedStatus,
        [string]$Description
    )
    
    $url = "$BaseUrl$Endpoint"
    
    Write-Info "Testing: $Description"
    Write-Info "URL: $url"
    
    try {
        # Make the request
        $response = Invoke-WebRequest -Uri $url -TimeoutSec $Timeout -UseBasicParsing -ErrorAction Stop
        
        # Check HTTP status code
        if ($response.StatusCode -eq $ExpectedStatus) {
            Write-Success "✓ Status: $($response.StatusCode) (Expected: $ExpectedStatus)"
        }
        else {
            Write-Error "✗ Status: $($response.StatusCode) (Expected: $ExpectedStatus)"
            return $false
        }
        
        # Show response body if verbose
        if ($Verbose) {
            Write-Host "Response:"
            try {
                $jsonResponse = $response.Content | ConvertFrom-Json
                $jsonResponse | ConvertTo-Json -Depth 10
            }
            catch {
                Write-Host $response.Content
            }
        }
        
        Write-Host ""
        return $true
    }
    catch {
        Write-Error "Failed to connect to $url"
        Write-Error "Error: $($_.Exception.Message)"
        Write-Host ""
        return $false
    }
}

# Function to test service availability
function Test-ServiceAvailability {
    Write-Info "Testing service availability..."

    try {
        $response = Invoke-WebRequest -Uri "$BaseUrl/management/health" -TimeoutSec $Timeout -UseBasicParsing -ErrorAction Stop
        Write-Success "Service is available at $BaseUrl"
        return $true
    }
    catch {
        Write-Error "Service is not available at $BaseUrl"
        Write-Info "Please make sure the API Bridge service is running"
        Write-Info "You can start it with: .\start.ps1"
        return $false
    }
}

# Function to run comprehensive health checks
function Invoke-HealthChecks {
    Write-Host "=================================="
    Write-Host "API Bridge Health Check"
    Write-Host "=================================="
    Write-Info "Host: $TargetHost"
    Write-Info "Port: $Port"
    Write-Info "Timeout: ${Timeout}s"
    Write-Info "Verbose: $Verbose"
    Write-Host "=================================="
    Write-Host ""
    
    $failedTests = 0
    
    # Test service availability first
    if (-not (Test-ServiceAvailability)) {
        exit 1
    }
    
    # Test health endpoint
    if (-not (Test-Endpoint "/management/health" 200 "Health Check Endpoint")) {
        $failedTests++
    }

    # Test readiness endpoint
    if (-not (Test-Endpoint "/management/ready" 200 "Readiness Check Endpoint")) {
        $failedTests++
    }

    # Test status endpoint
    if (-not (Test-Endpoint "/management/v1/status" 200 "Service Status Endpoint")) {
        $failedTests++
    }

    # Test metrics endpoint
    if (-not (Test-Endpoint "/management/metrics" 200 "Prometheus Metrics Endpoint")) {
        $failedTests++
    }
    
    Write-Host "=================================="
    if ($failedTests -eq 0) {
        Write-Success "All health checks passed! ✓"
        Write-Info "API Bridge service is healthy and ready"
        exit 0
    }
    else {
        Write-Error "$failedTests test(s) failed! ✗"
        Write-Info "Please check the service logs for more details"
        exit 1
    }
}

# Function to run quick health check
function Invoke-QuickCheck {
    Write-Info "Quick health check..."

    try {
        $response = Invoke-WebRequest -Uri "$BaseUrl/management/health" -TimeoutSec $Timeout -UseBasicParsing -ErrorAction Stop

        if ($response.StatusCode -eq 200) {
            Write-Success "Service is healthy ✓"
            Write-Host "$BaseUrl/management/health returned status $($response.StatusCode)"
        }
        else {
            Write-Error "Service is not healthy ✗"
            Write-Host "$BaseUrl/management/health returned status $($response.StatusCode)"
            exit 1
        }
    }
    catch {
        Write-Error "Service is not healthy ✗"
        Write-Error "Error: $($_.Exception.Message)"
        exit 1
    }
}

# Main execution
function Main {
    if ($Verbose) {
        Invoke-HealthChecks
    }
    else {
        Invoke-QuickCheck
    }
}

# Run main function
Main
