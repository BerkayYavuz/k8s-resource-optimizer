# Quick Reference - Local Development

## 🚀 Starting the Application

### Option 1: Automatic (Recommended)
```powershell
# From project root, run:
.\scripts\start-local-dev.ps1
```
This starts both services in separate windows.

### Option 2: Manual

**Terminal 1 - ML Service:**
```powershell
cd d:\k8s-resource-optimizer
.\scripts\start-ml-service.ps1
```

**Terminal 2 - Optimizer:**
```powershell
cd d:\k8s-resource-optimizer
.\scripts\start-optimizer.ps1
```

---

## 🧪 Testing

### Test ML Service
```powershell
# Health check
curl http://localhost:5000/health

# Run full test suite
python scripts\test-ml-service.py
```

### Test Optimizer API
```powershell
# Health check
curl http://localhost:8080/api/v1/health

# List analyzed pods (requires K8s connection)
curl http://localhost:8080/api/v1/pods

# Get recommendations
curl http://localhost:8080/api/v1/recommendations
```

---

## 📝 Common Tasks

### Install/Update Python Dependencies
```powershell
cd ml-service
.\venv\Scripts\Activate.ps1
pip install -r requirements.txt
```

### Update Go Dependencies
```powershell
go mod tidy
go mod download
```

### Fix Python Syntax Error (if still seeing it)
```powershell
# Close the file in your editor first!
# The error is due to encoding issues on line 66 of ensemble.py
# Just close and reopen the file to fix
```

### Run with Different Config
```powershell
go run cmd/optimizer/main.go -config configs/your-config.yaml
```

---

## 🔧 Configuration Files

- **configs/config.yaml** - Base configuration (for Kubernetes deployment)
- **configs/config-local.yaml** - Local development configuration (use this!)

### Key Settings for Local Dev

```yaml
# In config-local.yaml:

kubernetes:
  in_cluster: false              # Use local kubeconfig
  kubeconfig_path: "~/.kube/config"

prometheus:
  url: "http://localhost:9090"   # Local Prometheus

ml_service:
  url: "http://localhost:5000"   # Local ML service

optimizer:
  dry_run: true                  # IMPORTANT: Keep true for testing!

analysis:
  interval_minutes: 1440         # Run once per day (or on startup)
```

---

## 📊 Architecture

```
┌─────────────┐
│   Browser   │──┐
└─────────────┘  │
                 │ http://localhost:8080/api/v1/*
                 ▼
┌──────────────────────────────┐
│   Go Optimizer Service       │
│   Port: 8080                 │
│   ┌────────────────────────┐ │
│   │ • Metrics Collector    │ │───► Prometheus (:9090)
│   │ • Data Pipeline        │ │
│   │ • K8s Client           │ │───► Kubernetes API
│   └────────────────────────┘ │
└───────────┬──────────────────┘
            │ http://localhost:5000/predict/*
            ▼
┌──────────────────────────────┐
│   Python ML Service          │
│   Port: 5000                 │
│   ┌────────────────────────┐ │
│   │ • Prophet Model        │ │
│   │ • LSTM Model           │ │
│   │ • Ensemble             │ │
│   └────────────────────────┘ │
└──────────────────────────────┘
```

---

## 🐛 Troubleshooting

### ML Service won't start
```powershell
# Check Python version
python --version  # Should be 3.11+

# Recreate virtual environment
cd ml-service
Remove-Item -Recurse -Force venv
python -m venv venv
.\venv\Scripts\Activate.ps1
pip install -r requirements.txt
```

### Optimizer won't start
```powershell
# Check Go version
go version  # Should be 1.21+

# Clear Go cache
go clean -cache
go mod tidy
```

### Can't connect to Kubernetes
```powershell
# Test kubectl
kubectl cluster-info

# Check config path in config-local.yaml
# Make sure in_cluster: false
```

### Can't connect to Prometheus
```powershell
# If Prometheus is in Kubernetes, port-forward it:
kubectl port-forward -n monitoring svc/prometheus-server 9090:9090

# Test connection
curl http://localhost:9090/-/healthy
```

### Syntax Error in ensemble.py
```powershell
# This is an encoding issue
# Solution: Close the file in your editor completely, then reopen it
# The file has invisible characters that need to be cleaned
```

---

## 📚 Documentation

- **[LOCAL_SETUP.md](LOCAL_SETUP.md)** - Detailed setup guide
- **[README.md](README.md)** - Project overview
- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System architecture

---

## 🎯 Next Steps

1. ✅ Start both services
2. ✅ Test ML service with `python scripts\test-ml-service.py`
3. ✅ Configure Prometheus/K8s access in `config-local.yaml`
4. ✅ Test optimizer health endpoint
5. ✅ Review first recommendations (if K8s connected)

---

## 💡 Pro Tips

- Keep the ML service running - TensorFlow takes time to load
- Use `dry_run: true` until you're confident in recommendations
- Start with a single namespace for testing
- Check logs in both terminal windows for errors
- The first ML prediction will be slow (model training)

---

**Happy developing! 🎉**

For questions, check the main documentation or examine the code in:
- `cmd/optimizer/main.go` - Optimizer entry point  
- `ml-service/api/app.py` - ML service entry point
- `internal/` - Go backend packages
- `ml-service/models/` - ML models
