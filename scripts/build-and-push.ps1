# Docker imajlarını build edip Docker Hub'a push eder.
# Önce: docker login
#
# Kullanım: .\scripts\build-and-push.ps1 [tag]
#   tag: optional, default "latest"

param([string]$Tag = "latest")

$ErrorActionPreference = "Stop"

$REGISTRY = "berkayyvz"
$OPTIMIZER_IMAGE = "${REGISTRY}/k8s-resource-optimizer"
$ML_IMAGE = "${REGISTRY}/k8s-resource-optimizer-ml"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Resolve-Path (Join-Path $ScriptDir "..")
Set-Location $ProjectRoot

Write-Host "=== Building optimizer (linux/amd64 for AKS/GKE/EKS) ===" -ForegroundColor Cyan
docker build --platform linux/amd64 -f deployments/docker/Dockerfile.optimizer -t "${OPTIMIZER_IMAGE}:${Tag}" .

Write-Host "=== Building ML service (linux/amd64) ===" -ForegroundColor Cyan
docker build --platform linux/amd64 -f deployments/docker/Dockerfile.ml -t "${ML_IMAGE}:${Tag}" .

Write-Host "=== Pushing ${OPTIMIZER_IMAGE}:${Tag} ===" -ForegroundColor Cyan
docker push "${OPTIMIZER_IMAGE}:${Tag}"

Write-Host "=== Pushing ${ML_IMAGE}:${Tag} ===" -ForegroundColor Cyan
docker push "${ML_IMAGE}:${Tag}"

Write-Host "=== Done ===" -ForegroundColor Green
Write-Host "Images:"
Write-Host "  - ${OPTIMIZER_IMAGE}:${Tag}"
Write-Host "  - ${ML_IMAGE}:${Tag}"
