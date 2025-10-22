# API Documentation Generation Script (PowerShell)
# Generates docs.go automatically based on swagger.yaml file.

Write-Host "Starting API documentation generation..." -ForegroundColor Green

# Move to project root directory
$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

Write-Host "Project root: $projectRoot" -ForegroundColor Yellow

# 2. Execute swag init to regenerate docs.go
Write-Host "Running swag init..." -ForegroundColor Yellow
try {
    swag init -g cmd/api-bridge/main.go -o api-docs
    Write-Host "docs.go generated successfully!" -ForegroundColor Green
} catch {
    Write-Host "Failed to run swag init: $_" -ForegroundColor Red
    exit 1
}

# 3. Delete swagger.json file (keep YAML only)
if (Test-Path "api-docs\swagger.json") {
    Remove-Item "api-docs\swagger.json"
    Write-Host "swagger.json file deleted (YAML only)" -ForegroundColor Yellow
}

# 4. Remove LeftDelim, RightDelim from docs.go (swag version compatibility)
Write-Host "Fixing docs.go compatibility..." -ForegroundColor Yellow
$docsContent = Get-Content "api-docs\docs.go" -Raw
$docsContent = $docsContent -replace 'LeftDelim:\s*"[^"]*",', ''
$docsContent = $docsContent -replace 'RightDelim:\s*"[^"]*",', ''
$docsContent = $docsContent -replace ',\s*LeftDelim:\s*"[^"]*"', ''
$docsContent = $docsContent -replace ',\s*RightDelim:\s*"[^"]*"', ''
Set-Content "api-docs\docs.go" $docsContent -NoNewline
Write-Host "docs.go compatibility fixed!" -ForegroundColor Green

Write-Host "API documentation generation completed!" -ForegroundColor Green
Write-Host "Updated files:" -ForegroundColor Cyan
Write-Host "   - api-docs/docs.go" -ForegroundColor White
Write-Host "   - api-docs/swagger.yaml" -ForegroundColor White
Write-Host ""
Write-Host "Usage:" -ForegroundColor Magenta
Write-Host "   Run this script after modifying swagger.yaml file" -ForegroundColor White
Write-Host "   .\scripts\generate-docs.ps1" -ForegroundColor Gray
