# Example Prometheus Queries

This document contains the Prometheus queries used by the Kubernetes Resource Optimizer.

## CPU Metrics

### Container CPU Usage Rate

```promql
rate(container_cpu_usage_seconds_total{
  namespace="production",
  pod="web-app-7d4f8b9c",
  container!="",
  container!="POD"
}[5m])
```

**Description**: CPU usage rate in cores over a 5-minute window.

**Returns**: Time series of CPU usage per container

---

### Pod CPU Usage (All Containers)

```promql
sum(rate(container_cpu_usage_seconds_total{
  namespace="production",
  pod="web-app-7d4f8b9c",
  container!="POD"
}[5m])) by (pod)
```

**Description**: Total CPU usage for a pod across all containers.

---

### CPU Usage by Namespace

```promql
sum(rate(container_cpu_usage_seconds_total{
  namespace="production",
  container!="POD"
}[5m])) by (namespace, pod)
```

---

## Memory Metrics

### Container Memory Working Set

```promql
container_memory_working_set_bytes{
  namespace="production",
  pod="web-app-7d4f8b9c",
  container!="",
  container!="POD"
}
```

**Description**: Current memory working set (actual memory in use).

**Returns**: Memory in bytes per container

---

### Memory Usage in MB

```promql
container_memory_working_set_bytes{
  namespace="production",
  pod="web-app-7d4f8b9c",
  container!="POD"
} / 1024 / 1024
```

**Description**: Memory converted to megabytes.

---

## Resource Requests vs Usage

### CPU Request Utilization

```promql
sum(rate(container_cpu_usage_seconds_total{
  container!="POD"
}[5m])) by (pod, namespace)
/
sum(kube_pod_container_resource_requests{
  resource="cpu"
}) by (pod, namespace)
* 100
```

**Description**: Percentage of CPU requests actually used.

**Returns**: 0-100+ percentage (>100 means over-limit)

---

### Memory Request Utilization

```promql
sum(container_memory_working_set_bytes{
  container!="POD"
}) by (pod, namespace)
/
sum(kube_pod_container_resource_requests{
  resource="memory"
}) by (pod, namespace)
* 100
```

---

## Historical Analysis

### 14-Day CPU Average

```promql
avg_over_time(
  rate(container_cpu_usage_seconds_total{
    namespace="production",
    pod="web-app-7d4f8b9c",
    container!="POD"
  }[5m])[14d:5m]
)
```

**Description**: Average CPU usage over 14 days.

---

### 14-Day CPU Peak (P95)

```promql
quantile_over_time(0.95,
  rate(container_cpu_usage_seconds_total{
    namespace="production",
    pod="web-app-7d4f8b9c",
    container!="POD"
  }[5m])[14d:5m]
)
```

**Description**: 95th percentile CPU usage over 14 days.

---

### 14-Day Memory Average

```promql
avg_over_time(
  container_memory_working_set_bytes{
    namespace="production",
    pod="web-app-7d4f8b9c",
    container!="POD"
  }[14d:5m]
) / 1024 / 1024
```

---

### 14-Day Memory Peak (P95)

```promql
quantile_over_time(0.95,
  container_memory_working_set_bytes{
    namespace="production",
    pod="web-app-7d4f8b9c",
    container!="POD"
  }[14d:5m]
) / 1024 / 1024
```

---

## Waste Detection

### Over-Provisioned CPU (Pods Using <50% of Requests)

```promql
(
  sum(rate(container_cpu_usage_seconds_total{container!="POD"}[5m])) by (pod, namespace)
  /
  sum(kube_pod_container_resource_requests{resource="cpu"}) by (pod, namespace)
) < 0.5
```

---

### Over-Provisioned Memory

```promql
(
  sum(container_memory_working_set_bytes{container!="POD"}) by (pod, namespace)
  /
  sum(kube_pod_container_resource_requests{resource="memory"}) by (pod, namespace)
) < 0.5
```

---

## System-Wide Metrics

### Cluster CPU Waste

```promql
sum(kube_pod_container_resource_requests{resource="cpu"})
-
sum(rate(container_cpu_usage_seconds_total{container!="POD"}[5m]))
```

**Description**: Total unused CPU cores across the cluster.

---

### Cluster Memory Waste

```promql
(
  sum(kube_pod_container_resource_requests{resource="memory"})
  -
  sum(container_memory_working_set_bytes{container!="POD"})
) / 1024 / 1024 / 1024
```

**Description**: Total unused memory in GB.

---

## Time Range Parameters

The optimizer uses these queries with configurable time ranges:

- **Short term** (5m): Current state
- **Medium term** (1h): Recent trends
- **Long term** (7d, 14d, 30d): Historical analysis

Example with time range:
```promql
container_cpu_usage_seconds_total{
  namespace="production",
  pod="web-app-7d4f8b9c"
}[14d:5m]
```

This queries 14 days of data with 5-minute resolution.

---

## Notes

1. **Container filtering**: `container!="POD"` excludes the pause container
2. **Resolution**: 5m is standard but configurable
3. **Labels**: Adjust namespace, pod, container labels to match your cluster
4. **kube-state-metrics**: Required for resource request queries
