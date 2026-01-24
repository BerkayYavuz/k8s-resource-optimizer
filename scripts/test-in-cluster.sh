#!/bin/bash
# AKS/K8s cluster'da optimizer ve ml-service pod'larının çalışıp çalışmadığını test eder.
# Kullanım: ./scripts/test-in-cluster.sh [namespace]
#   namespace: default: default

set -e

NS="${1:-default}"

echo "=============================================="
echo "  k8s-resource-optimizer cluster test"
echo "  namespace: $NS"
echo "=============================================="

# 1) Pod durumları
echo ""
echo "--- 1) Pod durumları ---"
kubectl get pods -n "$NS" -l 'app in (k8s-resource-optimizer)' -o wide

RUNNING_OPT=$(kubectl get pods -n "$NS" -l component=optimizer --no-headers 2>/dev/null | grep -c "Running" || true)
RUNNING_ML=$(kubectl get pods -n "$NS" -l component=ml-service --no-headers 2>/dev/null | grep -c "Running" || true)

if [ "${RUNNING_OPT}" -lt 1 ] || [ "${RUNNING_ML}" -lt 1 ]; then
  echo ""
  echo "Uyarı: Her iki pod da Running olmalı (optimizer: $RUNNING_OPT, ml-service: $RUNNING_ML)."
  exit 1
fi

# 2) Son loglar (kısa)
echo ""
echo "--- 2) Optimizer son loglar ---"
kubectl logs -n "$NS" -l component=optimizer --tail=5 2>/dev/null || echo "(log yok)"

echo ""
echo "--- 3) ML service son loglar ---"
kubectl logs -n "$NS" -l component=ml-service --tail=5 2>/dev/null || echo "(log yok)"

# 3) Port-forward + HTTP testleri
echo ""
echo "--- 4) HTTP health kontrolleri (port-forward) ---"

OPT_PF_PID=""
ML_PF_PID=""

cleanup() {
  [ -n "$OPT_PF_PID" ] && kill $OPT_PF_PID 2>/dev/null || true
  [ -n "$ML_PF_PID" ] && kill $ML_PF_PID 2>/dev/null || true
}
trap cleanup EXIT

kubectl port-forward -n "$NS" svc/k8s-resource-optimizer 8080:8080 &>/dev/null &
OPT_PF_PID=$!
kubectl port-forward -n "$NS" svc/ml-service 5000:5000 &>/dev/null &
ML_PF_PID=$!

sleep 2

# Optimizer health
echo -n "  Optimizer /api/v1/health: "
if curl -sf http://localhost:8080/api/v1/health | head -c 200; then
  echo " [OK]"
else
  echo " [FAIL]"
fi

# ML health
echo -n "  ML service /health: "
if curl -sf http://localhost:5000/health | head -c 200; then
  echo " [OK]"
else
  echo " [FAIL]"
fi

# Optimizer recommendations (boş olabilir, 200 dönmeli)
echo -n "  Optimizer /api/v1/recommendations: "
if curl -sf http://localhost:8080/api/v1/recommendations >/dev/null; then
  echo " [OK]"
  curl -sf http://localhost:8080/api/v1/recommendations | head -c 300
  echo ""
else
  echo " [FAIL]"
fi

# ML /predict/all (minimal payload)
echo -n "  ML service /predict/all: "
PAYLOAD='{"pod_name":"test","namespace":"default","metrics":{"c1":{"cpu":[{"timestamp":"2026-01-01T00:00:00Z","value":0.1}],"memory":[{"timestamp":"2026-01-01T00:00:00Z","value":256}]}}}'
if curl -sf -X POST http://localhost:5000/predict/all -H "Content-Type: application/json" -d "$PAYLOAD" | grep -q '"success":true'; then
  echo " [OK]"
else
  echo " [FAIL] (curl ile dene: curl -s -X POST http://localhost:5000/predict/all -H 'Content-Type: application/json' -d '$PAYLOAD')"
fi

echo ""
echo "=============================================="
echo "  Test tamamlandı."
echo "=============================================="
