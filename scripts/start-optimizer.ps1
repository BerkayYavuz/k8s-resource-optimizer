#!/usr/bin/env pwsh
# Quick start script for Go Optimizer Service
# Run this from the project root: .\scripts\start-optimizer.ps1

param(
    [string]$Config = "configs/config-local.yaml"
)

Write-Host "[START] Starting Kubernetes Resource Optimizer..." -ForegroundColor Green

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] Go is not installed! Please install Go 1.21+ from https://go.dev/dl/" -ForegroundColor Red
    exit 1
}

# Check if config file exists
if (-not (Test-Path $Config)) {
    Write-Host "[WARNING] Config file not found: $Config" -ForegroundColor Yellow
    Write-Host "Creating default local config..." -ForegroundColor Yellow
    
    if (-not (Test-Path "configs")) {
        New-Item -ItemType Directory -Path "configs" | Out-Null
    }
    
    Copy-Item "configs/config.yaml" $Config
    Write-Host "[OK] Created $Config - please review and update if needed" -ForegroundColor Green
}

# Download dependencies if needed
Write-Host "[SETUP] Checking Go dependencies..." -ForegroundColor Yellow
go mod download
go mod tidy

Write-Host ""
Write-Host "[RUNNING] Starting Optimizer Service on http://localhost:8080" -ForegroundColor Cyan
Write-Host "  Using config: $Config" -ForegroundColor Gray
Write-Host "  Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""

# Run the optimizer
go run cmd/optimizer/main.go -config $Config
