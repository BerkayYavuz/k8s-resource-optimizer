# API Documentation

## Base URL

When running locally: `http://localhost:8080/api/v1`

In Kubernetes: `http://k8s-resource-optimizer:8080/api/v1`

## Endpoints

### Health Check

**GET** `/health`

Returns the health status of the service.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-06T19:00:00Z"
}
```

---

### List All Pods

**GET** `/pods`

Returns a list of all pods that have been analyzed.

**Response:**
```json
{
  "pods": [
    "default/web-app-7d4f8b9c",
    "production/api-server-5g8h2k",
    "staging/worker-queue-3d9f4"
  ],
  "count": 3
}
```

---

### List All Recommendations

**GET** `/recommendations`

Returns all pod recommendations.

**Response:**
```json
{
  "recommendations": [
    {
      "pod_name": "web-app-7d4f8b9c",
      "namespace": "default",
      "timestamp": "2026-01-06T19:00:00Z",
      "containers": {
        "nginx": {
          "container": "nginx",
          "current": {
            "cpu_request": 1.0,
            "cpu_limit": 2.0,
            "memory_request": 1024,
            "memory_limit": 2048
          },
          "predicted": {
            "avg_cpu": 0.15,
            "peak_cpu": 0.35,
            "avg_memory": 256,
            "peak_memory": 512
          },
          "recommended_cpu_request": 0.173,
          "recommended_cpu_limit": 0.438,
          "recommended_memory_request": 294,
          "recommended_memory_limit": 640,
          "cpu_request_change_percent": -82.7,
          "cpu_limit_change_percent": -78.1,
          "memory_request_change_percent": -71.3,
          "memory_limit_change_percent": -68.8,
          "confidence": 0.87
        }
      },
      "potential_cpu_saving_cores": 0.827,
      "potential_memory_saving_mb": 730,
      "cpu_waste_percentage": 82.7,
      "memory_waste_percentage": 71.3,
      "overall_confidence": 0.87,
      "dry_run": true
    }
  ],
  "count": 1
}
```

---

### Get Pod Recommendation

**GET** `/recommendations/{namespace}/{pod}`

Returns recommendation for a specific pod.

**Path Parameters:**
- `namespace` - Kubernetes namespace
- `pod` - Pod name

**Query Parameters:**
- `format` (optional) - Response format: `json` (default) or `yaml`

**Response (JSON):**
```json
{
  "pod_name": "web-app-7d4f8b9c",
  "namespace": "default",
  "containers": {
    "nginx": {
      "recommended_cpu_request": 0.173,
      "recommended_cpu_limit": 0.438,
      "recommended_memory_request": 294,
      "recommended_memory_limit": 640
    }
  }
}
```

**Response (YAML - with `?format=yaml`):**
```yaml
# Resource recommendation for pod: default/web-app-7d4f8b9c
# Generated: 2026-01-06T19:00:00Z
# Confidence: 87.00%
# DRY RUN MODE - Review before applying

spec:
  containers:
  - name: nginx
    resources:
      requests:
        cpu: "173m"  # was: 1000m (change: -82.7%)
        memory: "294Mi"  # was: 1024Mi (change: -71.3%)
      limits:
        cpu: "438m"  # was: 2000m (change: -78.1%)
        memory: "640Mi"  # was: 2048Mi (change: -68.8%)
```

---

### Prometheus Metrics

**GET** `/metrics`

Returns Prometheus-format metrics for monitoring.

**Response:**
```
# HELP k8s_optimizer_pods_analyzed Total number of pods analyzed
# TYPE k8s_optimizer_pods_analyzed gauge
k8s_optimizer_pods_analyzed 42

# HELP k8s_optimizer_cpu_savings_cores Potential CPU savings in cores
# TYPE k8s_optimizer_cpu_savings_cores gauge
k8s_optimizer_cpu_savings_cores 15.750

# HELP k8s_optimizer_memory_savings_mb Potential memory savings in MB
# TYPE k8s_optimizer_memory_savings_mb gauge
k8s_optimizer_memory_savings_mb 32768

# HELP k8s_optimizer_avg_confidence Average prediction confidence
# TYPE k8s_optimizer_avg_confidence gauge
k8s_optimizer_avg_confidence 0.835
```

## ML Service API

The ML service runs on port 5000 and is internal to the system.

### Predict CPU

**POST** `/predict/cpu`

```json
{
  "container": "nginx",
  "cpu": [
    {"timestamp": "2026-01-01T00:00:00Z", "value": 0.15},
    {"timestamp": "2026-01-01T01:00:00Z", "value": 0.18}
  ]
}
```

### Predict Memory

**POST** `/predict/memory`

```json
{
  "container": "nginx",
  "memory": [
    {"timestamp": "2026-01-01T00:00:00Z", "value": 268435456},
    {"timestamp": "2026-01-01T01:00:00Z", "value": 283115520}
  ]
}
```

### Predict All

**POST** `/predict/all`

```json
{
  "pod_name": "web-app",
  "namespace": "default",
  "metrics": {
    "nginx": {
      "cpu": [...],
      "memory": [...]
    }
  }
}
```

**Response:**
```json
{
  "success": true,
  "pod_name": "web-app",
  "namespace": "default",
  "predictions": {
    "nginx": {
      "predicted_avg_cpu": 0.15,
      "predicted_peak_cpu": 0.35,
      "predicted_avg_memory": 256,
      "predicted_peak_memory": 512,
      "confidence": 0.87,
      "model": "ensemble"
    }
  },
  "timestamp": "2026-01-06T19:00:00Z"
}
```

## Error Responses

All endpoints return standard error responses:

```json
{
  "error": "Error message describing what went wrong"
}
```

**Status Codes:**
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `404` - Not Found (pod/recommendation doesn't exist)
- `500` - Internal Server Error
