# System Architecture

## Overview

The Kubernetes Resource Optimizer AI is a microservices-based system designed to analyze, predict, and recommend optimal resource allocations for Kubernetes pods.

## Design Principles

1. **Safety First**: Dry-run by default, extensive safety margins, minimum thresholds
2. **Modularity**: Clear separation of concerns between components
3. **Extensibility**: Easy to add new ML models, data sources, or optimization strategies
4. **Production-Ready**: Comprehensive logging, metrics, error handling, and monitoring
5. **Cloud-Native**: Designed to run in Kubernetes, follows 12-factor app principles

## System Components

### 1. Metrics Collector (Go)

**Responsibility**: Query Prometheus for pod-level resource metrics

**Key Features**:
- Prometheus HTTP API client with authentication support
- Configurable time windows (7, 14, 30 days)
- Rate limiting and retry logic
- Metric normalization

**Data Flow**:
```
Prometheus → HTTP Query → Parse Response → Time Series Data
```

**Queries**:
- CPU: `rate(container_cpu_usage_seconds_total[5m])`
- Memory: `container_memory_working_set_bytes`

---

### 2. Kubernetes Client (Go)

**Responsibility**: Interface with Kubernetes API

**Key Features**:
- In-cluster and kubeconfig support
- Namespace filtering (excludes system namespaces)
- Read pod specifications and resource requests/limits
- YAML patch generation

**RBAC Requirements**:
- Read pods across all namespaces
- Read namespaces
- Optional: Patch pods (if auto-apply enabled)

---

### 3. Data Pipeline (Go)

**Responsibility**: Preprocess raw metrics into ML-ready format

**Processing Steps**:
1. **Gap Filling**: Linear interpolation for missing data points
2. **Smoothing**: Rolling average (5-point window)
3. **Outlier Removal**: IQR-based filtering (3x IQR threshold)
4. **Normalization**: Unit conversion (bytes → MB for memory)
5. **Statistics**: Calculate avg, peak, percentiles (P50, P95, P99), std dev

**Why Each Step**:
- **Gap filling**: ML models need continuous data
- **Smoothing**: Reduces noise from metric collection jitter
- **Outlier removal**: Prevents skew from anomalies/OOM events
- **Statistics**: Provides baseline for comparison with predictions

---

### 4. ML Service (Python)

**Responsibility**: Predict future resource usage using ML models

**Architecture**:
```
Input Data → Prophet Model → Trend/Seasonality Predictions
           ↘ LSTM Model   → Pattern-based Predictions
                         ↘ Ensemble → Weighted Average → Output
```

#### Prophet Model
- **Best for**: Workloads with daily/weekly patterns
- **Strengths**: Handles seasonality, holidays, trend changes
- **Output**: Average and peak predictions with confidence intervals
- **Confidence Calculation**: Based on prediction interval width

#### LSTM Model
- **Best for**: Complex, non-linear patterns
- **Architecture**: 2-layer LSTM (64→32 units) + Dense layers
- **Training**: Online (per-request), early stopping
- **Output**: Average and peak predictions with training loss

#### Ensemble Strategy
- **Method**: Weighted average based on confidence scores
- **Peak Selection**: Maximum of both models (safety-first)
- **Why Ensemble**: Combines Prophet's seasonality detection with LSTM's pattern recognition

---

### 5. Optimization Engine (Go)

**Responsibility**: Convert ML predictions into resource recommendations

**Algorithm**:

```
Predicted Avg Resource (from ML)
  ↓ × (1 + safety_margin_requests)  [default: 15%]
  ↓ Apply minimum threshold
  → Recommended Request

Predicted Peak Resource (from ML)
  ↓ × (1 + safety_margin_limits)    [default: 25%]
  ↓ Apply minimum threshold
  ↓ Ensure >= Request
  → Recommended Limit
```

**Safety Mechanisms**:
1. **Safety Margins**: Prevent under-provisioning
2. **Minimum Thresholds**: Never go below safe minimums (10m CPU, 32MB memory)
3. **Sanity Checks**: Limits must be >= Requests
4. **Confidence Thresholds**: Can skip low-confidence recommendations

**Output**:
- Recommended requests and limits
- Change percentages
- Potential savings
- YAML patch for kubectl apply

---

### 6. REST API (Go)

**Responsibility**: Expose recommendations via HTTP

**Endpoints**:
- `/api/v1/health` - Health check
- `/api/v1/pods` - List analyzed pods
- `/api/v1/recommendations` - Get all recommendations
- `/api/v1/recommendations/{ns}/{pod}` - Get specific recommendation
- `/api/v1/metrics` - Prometheus metrics

**Features**:
- JSON and YAML response formats
- CORS support
- Request logging
- Error handling

---

## Data Flow

### Complete Analysis Cycle

```
1. Trigger (Timer or Manual)
   ↓
2. List Pods (Kubernetes API)
   ↓
3. For each pod:
   ↓
4. Collect Metrics (Prometheus)
   - Query last 14 days
   - CPU and memory per container
   ↓
5. Preprocess Data (Pipeline)
   - Fill gaps
   - Smooth
   - Remove outliers
   - Calculate statistics
   ↓
6. Convert to ML Format (JSON)
   {
     "pod_name": "...",
     "metrics": {
       "container1": {
         "cpu": [{"timestamp": "...", "value": 0.15}, ...],
         "memory": [...]
       }
     }
   }
   ↓
7. Call ML Service (HTTP POST to /predict/all)
   ↓
8. ML Service:
   - Run Prophet for each metric
   - Run LSTM for each metric
   - Combine with ensemble
   - Return predictions
   ↓
9. Optimization Engine
   - Apply safety margins
   - Apply thresholds
   - Calculate recommended resources
   - Generate YAML patch
   ↓
10. Store Recommendation (API Server)
    ↓
11. Expose via API
    - JSON format for programmatic access
    - YAML format for kubectl apply
    - Metrics for Grafana
```

### Interaction Diagram

```
┌──────────────┐
│  Main Loop   │
└──────┬───────┘
       │ Every N minutes
       ▼
┌──────────────────────────────────────┐
│  Kubernetes Client                   │
│  - List all running pods             │
│  - Filter excluded namespaces        │
└──────┬───────────────────────────────┘
       │ Pod list
       ▼
┌──────────────────────────────────────┐
│  Metrics Collector                   │
│  - Query Prometheus per pod          │
│  - Fetch 14 days of data             │
└──────┬───────────────────────────────┘
       │ Raw time series
       ▼
┌──────────────────────────────────────┐
│  Data Pipeline                       │
│  - Preprocess                        │
│  - Create ML input JSON              │
└──────┬───────────────────────────────┘
       │ ML-ready JSON
       ▼
┌──────────────────────────────────────┐
│  HTTP Request to ML Service          │
└──────┬───────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│  ML Service (Python)                 │
│  ┌────────────────────────────────┐  │
│  │ Prophet: Trend analysis        │  │
│  └───────────┬────────────────────┘  │
│  ┌───────────▼────────────────────┐  │
│  │ LSTM: Pattern recognition      │  │
│  └───────────┬────────────────────┘  │
│  ┌───────────▼────────────────────┐  │
│  │ Ensemble: Combine predictions  │  │
│  └────────────────────────────────┘  │
└──────┬───────────────────────────────┘
       │ Predictions
       ▼
┌──────────────────────────────────────┐
│  Optimization Engine                 │
│  - Calculate recommendations         │
│  - Apply safety margins              │
│  - Generate YAML                     │
└──────┬───────────────────────────────┘
       │ Recommendations
       ▼
┌──────────────────────────────────────┐
│  API Server                          │
│  - Store recommendation              │
│  - Expose via REST                   │
│  - Export Prometheus metrics         │
└──────────────────────────────────────┘
```

## Deployment Architecture

```
┌────────────────────────────────────────────────┐
│          Kubernetes Cluster                     │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │  Namespace: default                      │  │
│  │                                          │  │
│  │  ┌────────────────────────────────────┐ │  │
│  │  │  Deployment: k8s-resource-optimizer│ │  │
│  │  │  ┌──────────────────────────────┐  │ │  │
│  │  │  │  Pod                         │  │ │  │
│  │  │  │  Container: optimizer        │  │ │  │
│  │  │  │  - Port: 8080                │  │ │  │
│  │  │  │  - CPU: 100m-500m            │  │ │  │
│  │  │  │  - Mem: 128Mi-512Mi          │  │ │  │
│  │  │  └──────────────────────────────┘  │ │  │
│  │  └────────────────────────────────────┘ │  │
│  │  Service: k8s-resource-optimizer        │  │
│  │  - ClusterIP: 8080                      │  │
│  │                                          │  │
│  │  ┌────────────────────────────────────┐ │  │
│  │  │  Deployment: ml-service            │ │  │
│  │  │  ┌──────────────────────────────┐  │ │  │
│  │  │  │  Pod                         │  │ │  │
│  │  │  │  Container: ml-service       │  │ │  │
│  │  │  │  - Port: 5000                │  │ │  │
│  │  │  │  - CPU: 500m-2000m          │  │ │  │
│  │  │  │  - Mem: 1Gi-4Gi             │  │ │  │
│  │  │  └──────────────────────────────┘  │ │  │
│  │  └────────────────────────────────────┘ │  │
│  │  Service: ml-service                    │  │
│  │  - ClusterIP: 5000                      │  │
│  │                                          │  │
│  │  ConfigMap: optimizer-config            │  │
│  │  ServiceAccount: k8s-resource-optimizer │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │  Namespace: monitoring                   │  │
│  │  - Prometheus                            │  │
│  │  - Grafana                               │  │
│  └──────────────────────────────────────────┘  │
└────────────────────────────────────────────────┘
```

## Resource Requirements

### Optimizer Service
- **Typical Load**: 100m CPU, 128Mi memory
- **Peak Load**: 500m CPU, 512Mi memory
- **Scales**: Horizontally (stateless)

### ML Service
- **Typical Load**: 500m CPU, 1Gi memory
- **Peak Load**: 2000m CPU, 4Gi memory (during training)
- **Scales**: Horizontally with session affinity

## Security Considerations

1. **RBAC**: Minimal permissions (read-only by default)
2. **Non-root containers**: Both services run as non-root users
3. **No write access**: Cannot modify cluster by default (dry-run mode)
4. **Secrets**: Prometheus credentials via Kubernetes Secrets
5. **Network policies**: Recommended to restrict egress

## Performance Characteristics

### Analysis Performance
- **Small cluster** (50 pods): ~2-5 minutes
- **Medium cluster** (500 pods): ~20-30 minutes
- **Large cluster** (5000 pods): ~3-4 hours

### Optimization
- Single pod analysis: 30-60 seconds
- ML prediction: 10-20 seconds per container
- Memory: Streaming processing, O(1) memory per pod

## Failure Modes and Handling

| Failure | Impact | Handling |
|---------|--------|----------|
| Prometheus unavailable | No metrics collection | Retry with exponential backoff, skip pod |
| ML service down | No predictions | Cache previous results, retry |
| Insufficient data | Low confidence | Use statistical fallback, flag in output |
| K8s API throttling | Slow analysis | Rate limiting, backoff |
| ML prediction error | No recommendation | Log error, continue with other pods |

## Future Enhancements

### Phase 2
- Persistent storage for historical recommendations
- Trend analysis (improving/worsening over time)
- Auto-rollback on pod failures post-change
- Web UI for visualization

### Phase 3
- Multi-cluster support
- Cost attribution (cloud provider billing integration)
- Policy engine (custom rules per namespace/label)
- GitOps integration (automatic PR generation)

### Phase 4
- Reinforcement learning (learn from applied changes)
- Pod dependency graphs (avoid cascading failures)
- Capacity planning forecasts
- What-if analysis tools

## Technology Decisions

### Why Go for Backend?
- Excellent Kubernetes ecosystem (client-go)
- Fast, efficient, low memory footprint
- Great for HTTP APIs and concurrent operations
- Native container support

### Why Python for ML?
- Rich ML ecosystem (Prophet, TensorFlow, scikit-learn)
- Mature libraries for data science
- Easier to experiment and iterate
- Can be swapped for specialized ML platforms later

### Why Microservices?
- Independent scaling (ML is compute-heavy)
- Technology flexibility (Go + Python)
- Easier testing and deployment
- Can replace ML component without touching optimizer

### Why Prometheus?
- Standard in Kubernetes ecosystem
- Already collecting the metrics we need
- Query language is powerful
- Wide adoption means good compatibility
