#!/usr/bin/env pwsh
# Quick start script for Python ML Service
# Run this from the project root: .\scripts\start-ml-service.ps1

Write-Host "[START] Starting ML Service..." -ForegroundColor Green

# Navigate to ml-service directory
$mlServicePath = Join-Path $PSScriptRoot "..\ml-service"
Set-Location $mlServicePath

# Check if virtual environment exists
if (-not (Test-Path "venv")) {
    Write-Host "[SETUP] Creating virtual environment..." -ForegroundColor Yellow
    python -m venv venv
}

# Activate virtual environment
Write-Host "[SETUP] Activating virtual environment..." -ForegroundColor Yellow
& ".\venv\Scripts\Activate.ps1"

# Check if dependencies are installed
$requirementsHash = Get-FileHash "requirements.txt" -Algorithm MD5
$installedHash = $null
if (Test-Path ".requirements.hash") {
    $installedHash = Get-Content ".requirements.hash"
}

if ($requirementsHash.Hash -ne $installedHash) {
    Write-Host "[SETUP] Installing/updating dependencies..." -ForegroundColor Yellow
    pip install --upgrade pip
    pip install -r requirements.txt
    
    # Save hash to avoid reinstalling unnecessarily
    $requirementsHash.Hash | Out-File ".requirements.hash"
} else {
    Write-Host "[OK] Dependencies already up to date" -ForegroundColor Green
}

# Start the ML service
Write-Host ""
Write-Host "[RUNNING] Starting Flask ML Service on http://localhost:5000" -ForegroundColor Cyan
Write-Host "  Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""

python api/app.py
