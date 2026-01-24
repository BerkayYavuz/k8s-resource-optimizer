#!/usr/bin/env pwsh
# Complete local development startup script
# Starts both ML service and Optimizer in separate windows

Write-Host "=========================================================" -ForegroundColor Cyan
Write-Host "Starting K8s Resource Optimizer - Local Development" -ForegroundColor Cyan
Write-Host "=========================================================" -ForegroundColor Cyan
Write-Host ""

# Get the directory where this script is located
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir

# Change to project root
Set-Location $projectRoot

Write-Host "[INFO] Project root: $projectRoot" -ForegroundColor Gray
Write-Host ""

# Check prerequisites
Write-Host "[STEP 1/4] Checking prerequisites..." -ForegroundColor Yellow

$allGood = $true

# Check Python
if (Get-Command python -ErrorAction SilentlyContinue) {
    $pythonVersion = python --version
    Write-Host "  [OK] Python: $pythonVersion" -ForegroundColor Green
} else {
    Write-Host "  [ERROR] Python not found! Install from https://www.python.org/" -ForegroundColor Red
    $allGood = $false
}

# Check Go
if (Get-Command go -ErrorAction SilentlyContinue) {
    $goVersion = go version
    Write-Host "  [OK] Go: $goVersion" -ForegroundColor Green
} else {
    Write-Host "  [ERROR] Go not found! Install from https://go.dev/dl/" -ForegroundColor Red
    $allGood = $false
}

if (-not $allGood) {
    Write-Host ""
    Write-Host "[ERROR] Please install missing prerequisites and try again." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[STEP 2/4] All prerequisites found!" -ForegroundColor Green
Write-Host ""

# Start ML Service in new window
Write-Host "[STEP 3/4] Starting ML Service in new window..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-File", "$scriptDir\start-ml-service.ps1"

# Wait a bit for ML service to start
Write-Host "  [INFO] Waiting 10 seconds for ML service to initialize..." -ForegroundColor Gray
Start-Sleep -Seconds 10

# Start Optimizer in new window
Write-Host "[STEP 4/4] Starting Optimizer in new window..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-File", "$scriptDir\start-optimizer.ps1"

Write-Host ""
Write-Host "=========================================================" -ForegroundColor Cyan
Write-Host "[SUCCESS] Both services are starting!" -ForegroundColor Green
Write-Host ""
Write-Host "Services:" -ForegroundColor Yellow
Write-Host "  ML Service:  http://localhost:5000" -ForegroundColor White
Write-Host "  Optimizer:   http://localhost:8080" -ForegroundColor White
Write-Host ""
Write-Host "Test commands:" -ForegroundColor Yellow
Write-Host "  curl http://localhost:5000/health" -ForegroundColor White
Write-Host "  curl http://localhost:8080/api/v1/health" -ForegroundColor White
Write-Host ""
Write-Host "For more information, see LOCAL_SETUP.md" -ForegroundColor Gray
Write-Host "=========================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Press any key to close this window..." -ForegroundColor Gray
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
