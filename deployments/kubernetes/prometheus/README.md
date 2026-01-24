# Prometheus Kurulumu

k8s-resource-optimizer için gerekli **container_cpu_usage_seconds_total** ve **container_memory_working_set_bytes** metrikleri, kubelet/cAdvisor'dan toplanır.

## Kurulum sırası

```bash
# 1. Namespace
kubectl apply -f prometheus/namespace.yaml

# 2. RBAC, ConfigMap, Deployment, Service
kubectl apply -f prometheus/rbac.yaml
kubectl apply -f prometheus/configmap.yaml
kubectl apply -f prometheus/deployment.yaml
kubectl apply -f prometheus/service.yaml
```

Veya tümü (namespace zaten varsa):

```bash
kubectl apply -f prometheus/
```

## Doğrulama

```bash
kubectl get pods -n monitoring -l app=prometheus
kubectl port-forward -n monitoring svc/prometheus-server 9090:9090
# Tarayıcı: http://localhost:9090 → Status → Targets → kubernetes-cadvisor "UP" olmalı
```

## Optimizer bağlantısı

Optimizer ConfigMap'te Prometheus adresi `http://prometheus-server.monitoring:9090` olarak ayarlı. Optimizer `default` namespace'den bu adrese erişir.
