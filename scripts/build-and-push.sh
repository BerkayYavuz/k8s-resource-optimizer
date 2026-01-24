#!/bin/bash
# Docker imajlarını build edip Docker Hub'a push eder.
# Önce: docker login
#
# Kullanım: ./scripts/build-and-push.sh [tag]
#   tag: optional, default "latest"

set -e

REGISTRY="berkayyvz"
OPTIMIZER_IMAGE="${REGISTRY}/k8s-resource-optimizer"
ML_IMAGE="${REGISTRY}/k8s-resource-optimizer-ml"
TAG="${1:-latest}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "=== Building optimizer (linux/amd64 for AKS/GKE/EKS) ==="
docker build --platform linux/amd64 -f deployments/docker/Dockerfile.optimizer -t "${OPTIMIZER_IMAGE}:${TAG}" .

echo "=== Building ML service (linux/amd64) ==="
docker build --platform linux/amd64 -f deployments/docker/Dockerfile.ml -t "${ML_IMAGE}:${TAG}" .

echo "=== Pushing ${OPTIMIZER_IMAGE}:${TAG} ==="
docker push "${OPTIMIZER_IMAGE}:${TAG}"

echo "=== Pushing ${ML_IMAGE}:${TAG} ==="
docker push "${ML_IMAGE}:${TAG}"

echo "=== Done ==="
echo "Images:"
echo "  - ${OPTIMIZER_IMAGE}:${TAG}"
echo "  - ${ML_IMAGE}:${TAG}"
