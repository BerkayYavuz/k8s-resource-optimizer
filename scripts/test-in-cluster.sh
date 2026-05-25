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

# 4) Port-forward + HTTP testleri
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
body=$(curl -sf http://localhost:8080/api/v1/health 2>/dev/null)
if [ -n "$body" ]; then
  echo "[OK]"
  status=$(echo "$body" | grep -oE '"status"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*:"\([^"]*\)".*/\1/')
  [ -n "$status" ] && echo "    → status: $status"
else
  echo "[FAIL]"
fi

# ML health
echo -n "  ML service /health: "
body=$(curl -sf http://localhost:5000/health 2>/dev/null)
if [ -n "$body" ]; then
  echo "[OK]"
  status=$(echo "$body" | grep -oE '"status"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*:"\([^"]*\)".*/\1/')
  [ -n "$status" ] && echo "    → status: $status"
else
  echo "[FAIL]"
fi

# Optimizer recommendations (boş olabilir, 200 dönmeli)
echo -n "  Optimizer /api/v1/recommendations: "
body=$(curl -sf http://localhost:8080/api/v1/recommendations 2>/dev/null)
if [ -n "$body" ]; then
  echo "[OK]"
  count=$(echo "$body" | grep -oE '"count"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' | head -1)
  [ -z "$count" ] && count=0
  echo "    → Toplam: $count öneri"
  if [ "$count" -gt 0 ]; then
    if command -v jq &>/dev/null; then
      # Namespace dağılımı
      ns_summary=$(echo "$body" | jq -r '
        [.recommendations[].namespace] | group_by(.) | map("\(.[0]): \(length)") | join(", ")
      ' 2>/dev/null)
      [ -n "$ns_summary" ] && echo "    → Namespace: $ns_summary"
      # Toplam tasarruf (potential_cpu_saving_cores, potential_memory_saving_mb)
      total_cpu=$(echo "$body"  | jq -r '([.recommendations[] | .potential_cpu_saving_cores // 0] | add // 0) * 1000 | floor / 1000' 2>/dev/null)
      total_mem=$(echo "$body"  | jq -r '[.recommendations[] | .potential_memory_saving_mb // 0] | add // 0' 2>/dev/null)
      if [ -n "$total_cpu" ] && [ "$total_cpu" != "null" ]; then
        # Sadece makul aralıktaysa göster (aşırı değerler veri/optimizer anomalisi olabilir)
        tc=${total_cpu#-}; tm=${total_mem#-}
        if [ ${#tc} -le 10 ] && [ ${#tm} -le 12 ]; then
          echo "    → Tasarruf: CPU ${total_cpu} core, bellek ${total_mem} MB"
        else
          echo "    → Tasarruf: (özet atlandı — aşırı değerler; optimizer çıktısını kontrol edin)"
        fi
      fi
      # İlk 8 pod: namespace/pod | container sayısı | CPU tasarruf | Mem tasarruf | güven
      echo "    → Öneri özeti (en fazla 8):"
      echo "$body" | jq -r '
        .recommendations[:8][] |
        "      \(.namespace)/\(.pod_name)  ·  \(.containers | keys | length) container  ·  CPU \(.potential_cpu_saving_cores // 0 | . * 1000 | floor / 1000) core, Mem \(.potential_memory_saving_mb // 0) MB  ·  güven %\((.overall_confidence // 0) * 100 | floor)"
      ' 2>/dev/null
      if [ "$count" -gt 8 ]; then
        more=$((count - 8))
        echo "      … ve $more öneri daha"
      fi
    else
      # jq yok: namespace/pod listesi (grep ile)
      echo "    → Pod'lar (ilk 12):"
      echo "$body" | grep -oE '"(pod_name|namespace)"[[:space:]]*:[[:space:]]*"[^"]*"' | paste - - 2>/dev/null | head -12 | while read -r a b; do
        ns=$(echo "$b" | sed 's/.*"\([^"]*\)"[[:space:]]*$/\1/')
        pod=$(echo "$a" | sed 's/.*"\([^"]*\)"[[:space:]]*$/\1/')
        printf "      %s/%s\n" "$ns" "$pod"
      done
      [ "$count" -gt 12 ] && echo "      … ve $((count - 12)) öneri daha"
    fi
  fi
else
  echo "[FAIL]"
fi

# ML /predict/all (minimal payload)
echo -n "  ML service /predict/all: "
PAYLOAD='{"pod_name":"test","namespace":"default","metrics":{"c1":{"cpu":[{"timestamp":"2026-01-01T00:00:00Z","value":0.1}],"memory":[{"timestamp":"2026-01-01T00:00:00Z","value":256}]}}}'
if curl -sf -X POST http://localhost:5000/predict/all -H "Content-Type: application/json" -d "$PAYLOAD" 2>/dev/null | grep -q '"success":true'; then
  echo "[OK]"
else
  echo "[FAIL]"
  echo "    → Örnek: curl -s -X POST http://localhost:5000/predict/all -H 'Content-Type: application/json' -d '\$PAYLOAD'"
fi

echo ""
echo "=============================================="
echo "  Test tamamlandı."
echo "=============================================="
