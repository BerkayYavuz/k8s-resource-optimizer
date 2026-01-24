# Kubernetes Resource Optimizer AI

**Production-ready AI-based predictive resource recommendation system for Kubernetes clusters.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🎯 Overview

The Kubernetes Resource Optimizer AI addresses a critical problem in Kubernetes cluster management: **resource waste from over-provisioned pods**. Most teams guess at CPU and memory requests/limits, leading to massive inefficiency.

This system:
- ✅ Analyzes historical resource usage from Prometheus
- ✅ Uses AI (Prophet + LSTM) to predict future resource needs
- ✅ Recommends optimal resource requests and limits per pod
- ✅ Operates in safe dry-run mode by default
- ✅ Provides Grafana-compatible metrics
- ✅ Works cluster-wide or per-pod

### Why This Matters

- **Cost Savings**: Reduce cloud costs by 20-60% through right-sizing
- **Better Scheduling**: Kubernetes can schedule more efficiently with accurate requests
- **Prevent Waste**: No more "just to be safe" resource allocation
- **Data-Driven**: Decisions based on real metrics and ML predictions, not guesses

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│  ┌─────────┐        ┌─────────────┐      ┌──────────────┐  │
│  │  Pods   │───────▶│ Prometheus  │      │ K8s API      │  │
│  └─────────┘        └──────┬──────┘      └──────┬───────┘  │
└─────────────────────────────┼────────────────────┼──────────┘
                              │                    │
                              ▼                    ▼
                    ┌──────────────────────────────────┐
                    │   Go Backend (Optimizer)         │
                    │  ┌────────────┐  ┌─────────────┐ │
                    │  │ Collector  │  │ K8s Client  │ │
                    │  └─────┬──────┘  └──────┬──────┘ │
                    │  ┌─────▼────────────────▼──────┐ │
                    │  │   Data Pipeline              │ │
                    │  └─────┬────────────────────────┘ │
                    └────────┼────────────────────────── ┘
                             │ JSON/REST
                             ▼
                    ┌──────────────────────────────────┐
                    │  Python ML Service               │
                    │  ┌─────────┐    ┌─────────────┐  │
                    │  │ Prophet │    │    LSTM     │  │
                    │  └────┬────┘    └──────┬──────┘  │
                    │  ┌────▼────────────────▼──────┐  │
                    │  │    Ensemble Predictor       │  │
                    │  └─────────────┬───────────────┘  │
                    └────────────────┼──────────────────┘
                                     │ Predictions
                                     ▼
                    ┌──────────────────────────────────┐
                    │   Optimization Engine            │
                    │   • Calculate recommendations    │
                    │   • Apply safety margins         │
                    │   • Generate YAML patches        │
                    └────────────┬─────────────────────┘
                                 │
                                 ▼
                    ┌──────────────────────────────────┐
                    │      REST API & Output           │
                    │  • JSON recommendations          │
                    │  • YAML patches                  │
                    │  • Grafana metrics               │
                    └──────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (v1.24+)
- Prometheus installed and scraping pod metrics
- kubectl configured
- Docker (for building images)
- Go 1.21+ (for local development)
- Python 3.11+ (for ML service development)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-org/k8s-resource-optimizer.git
   cd k8s-resource-optimizer
   ```

2. **Build Docker images**
   ```bash
   # Build Go optimizer
   docker build -f deployments/docker/Dockerfile.optimizer -t k8s-resource-optimizer:latest .
   
   # Build Python ML service
   docker build -f deployments/docker/Dockerfile.ml -t k8s-resource-optimizer-ml:latest .
   ```

3. **Deploy to Kubernetes**
   ```bash
   # Create RBAC permissions
   kubectl apply -f deployments/kubernetes/rbac.yaml
   
   # Create ConfigMap
   kubectl apply -f deployments/kubernetes/configmap.yaml
   
   # Deploy ML service
   kubectl apply -f deployments/kubernetes/ml-deployment.yaml
   kubectl apply -f deployments/kubernetes/service.yaml
   
   # Deploy optimizer
   kubectl apply -f deployments/kubernetes/deployment.yaml
   ```

4. **Verify deployment**
   ```bash
   kubectl get pods -l app=k8s-resource-optimizer
   kubectl logs -l component=optimizer -f
   ```

## 📖 Usage

### Accessing Recommendations

**List all recommendations:**
```bash
kubectl port-forward svc/k8s-resource-optimizer 8080:8080
curl http://localhost:8080/api/v1/recommendations | jq
```

**Get specific pod recommendation:**
```bash
curl http://localhost:8080/api/v1/recommendations/default/my-pod | jq
```

**Get YAML patch:**
```bash
curl "http://localhost:8080/api/v1/recommendations/default/my-pod?format=yaml"
```

### Example Output

```json
{
  "pod_name": "web-app-7d4f8b9c",
  "namespace": "production",
  "timestamp": "2026-01-06T19:00:00Z",
  "containers": {
    "nginx": {
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
      "memory_request_change_percent": -71.3,
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
```

## ⚙️ Configuration

Edit `deployments/kubernetes/configmap.yaml` to customize:

```yaml
optimizer:
  # Safety margins prevent under-provisioning
  safety_margin_requests: 0.15  # 15% buffer on predictions
  safety_margin_limits: 0.25    # 25% buffer for limits
  
  # Minimum thresholds (never go below these)
  min_thresholds:
    cpu_cores: 0.01   # 10 millicores
    memory_mb: 32     # 32MB
  
  # Dry-run mode (set false to enable auto-apply - USE WITH CAUTION)
  dry_run: true

analysis:
  # How far back to analyze metrics
  window_days: 14
  
  # How often to run analysis
  interval_minutes: 60
```

## 🔒 Safety Features

This system is designed with **safety-first** principles:

1. **Dry-run by default**: Never applies changes automatically unless explicitly configured
2. **Safety margins**: Adds configurable buffers to prevent under-provisioning
3. **Minimum thresholds**: Never recommends resources below safe minimums
4. **Namespace filtering**: Skips critical system namespaces (kube-system, etc.)
5. **Confidence scores**: All recommendations include confidence levels
6. **Read-only RBAC**: Default permissions are read-only

## 📊 Grafana Integration

The system exposes Prometheus-format metrics at `/api/v1/metrics`:

```prometheus
# Total pods analyzed
k8s_optimizer_pods_analyzed

# Potential savings
k8s_optimizer_cpu_savings_cores
k8s_optimizer_memory_savings_mb

# Average confidence
k8s_optimizer_avg_confidence
```

See `examples/grafana-queries.json` for a complete dashboard.

## 🤖 ML Models

### Prophet
- **Purpose**: Trend detection and seasonality
- **Strengths**: Daily/weekly patterns, holidays, long-term trends
- **Use case**: Workloads with predictable patterns

### LSTM
- **Purpose**: Complex pattern recognition
- **Strengths**: Short-term predictions, non-linear patterns
- **Use case**: Dynamic workloads, sudden changes

### Ensemble
- **Strategy**: Weighted average based on confidence
- **Peak selection**: Maximum of both models for safety
- **Confidence**: Combined confidence score

## 📁 Project Structure

```
k8s-resource-optimizer/
├── cmd/
│   └── optimizer/          # Main application entry point
├── internal/
│   ├── api/                # REST API server
│   ├── collector/          # Prometheus metrics collector
│   ├── config/             # Configuration management
│   ├── k8s/                # Kubernetes client
│   ├── optimizer/          # Optimization engine
│   └── pipeline/           # Data preprocessing
├── ml-service/
│   ├── api/                # Flask REST API
│   ├── models/             # Prophet, LSTM, Ensemble
│   └── requirements.txt    # Python dependencies
├── deployments/
│   ├── docker/             # Dockerfiles
│   └── kubernetes/         # K8s manifests
├── configs/                # Example configurations
├── examples/               # Usage examples
└── docs/                   # Documentation
```

## 🛠️ Development

### Running Locally

**Go Backend:**
```bash
go mod download
go run cmd/optimizer/main.go -config configs/config.yaml
```

**Python ML Service:**
```bash
cd ml-service
pip install -r requirements.txt
python api/app.py
```

### Testing

```bash
# Go tests
go test ./internal/... -v -cover

# Python tests
cd ml-service
pytest tests/ -v
```

## 🚦 Roadmap

### Phase 1 (Current)
- [x] Prometheus metrics collection
- [x] Prophet + LSTM ensemble predictions
- [x] REST API endpoints
- [x] Kubernetes integration
- [x] Dry-run mode

### Phase 2
- [ ] Historical recommendation tracking
- [ ] A/B testing framework
- [ ] Auto-rollback on instability
- [ ] Slack/Teams notifications
- [ ] Web UI dashboard

### Phase 3
- [ ] Multi-cluster support
- [ ] Cost tracking integration (AWS/GCP/Azure)
- [ ] Advanced anomaly detection
- [ ] Canary deployments
- [ ] Policy-based rules engine

### Phase 4
- [ ] Reinforcement learning for optimization
- [ ] What-if scenario analysis
- [ ] Capacity planning predictions
- [ ] Integration with GitOps (ArgoCD/Flux)

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## 📜 License

MIT License - See LICENSE file for details

## 🙏 Acknowledgments

- Kubernetes SIG-Autoscaling for inspiration
- Facebook Prophet team
- TensorFlow/Keras community
- Prometheus project

## 📞 Support

- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Documentation**: `/docs` directory

## ⚠️ Disclaimer

This system is provided as-is. Always:
- Test in non-production environments first
- Review recommendations before applying
- Monitor cluster health after changes
- Keep safety margins appropriate for your workloads

---

**Built with ❤️ for the Kubernetes community**
