# Local Development Setup Guide

This guide will help you run the Kubernetes Resource Optimizer on your **Windows local machine** for development and testing.

## 📋 Prerequisites

Before starting, ensure you have the following installed:

### Required Software

1. **Go 1.21+**
   ```powershell
   # Verify installation
   go version
   ```
   Download from: https://go.dev/dl/

2. **Python 3.11+**
   ```powershell
   # Verify installation
   python --version
   # or
   python3 --version
   ```
   Download from: https://www.python.org/downloads/

3. **Git** (for cloning repositories)
   ```powershell
   git --version
   ```

### Optional (for full functionality)

4. **Kubernetes Cluster Access**
   - Local: Minikube, Docker Desktop with Kubernetes, or Kind
   - Remote: kubeconfig with cluster access

5. **Prometheus**
   - Running and scraping your Kubernetes cluster
   - Accessible from your local machine

---

## 🚀 Quick Start (Local Development Mode)

### Step 1: Set Up Python ML Service

1. **Navigate to ML service directory:**
   ```powershell
   cd d:\k8s-resource-optimizer\ml-service
   ```

2. **Create a virtual environment:**
   ```powershell
   python -m venv venv
   ```

3. **Activate the virtual environment:**
   ```powershell
   .\venv\Scripts\Activate.ps1
   ```
   
   > **Note:** If you get an execution policy error, run:
   > ```powershell
   > Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
   > ```

4. **Install dependencies:**
   ```powershell
   pip install --upgrade pip
   pip install -r requirements.txt
   ```

5. **Start the ML service:**
   ```powershell
   python api/app.py
   ```
   
   The ML service should now be running on `http://localhost:5000`
   
   **Test it:**
   ```powershell
   # In a new terminal
   curl http://localhost:5000/health
   ```

---

### Step 2: Set Up Go Backend (Optimizer)

1. **Open a new terminal** and navigate to project root:
   ```powershell
   cd d:\k8s-resource-optimizer
   ```

2. **Download Go dependencies:**
   ```powershell
   go mod download
   go mod tidy
   ```

3. **Create a local configuration file:**
   ```powershell
   # Copy the example config
   Copy-Item configs\config.yaml configs\config-local.yaml
   ```

4. **Edit `configs\config-local.yaml`** for local development:
   ```yaml
   # Kubernetes Resource Optimizer Configuration - LOCAL DEV

   prometheus:
     # Update to your local Prometheus or use a mock
     url: "http://localhost:9090"
     timeout: "30s"

   ml_service:
     # Point to local ML service
     url: "http://localhost:5000"
     timeout: "60s"

   kubernetes:
     # Use local kubeconfig instead of in-cluster
     in_cluster: false
     kubeconfig_path: "~/.kube/config"
     
     exclude_namespaces:
       - kube-system
       - kube-public
       - kube-node-lease

   optimizer:
     safety_margin_requests: 0.15
     safety_margin_limits: 0.25
     
     min_thresholds:
       cpu_cores: 0.01
       memory_mb: 32
     
     # Keep dry-run true for local testing
     dry_run: true

   api:
     port: 8080
     host: "0.0.0.0"

   analysis:
     window_days: 14
     # For local dev, you might want to increase this to avoid frequent runs
     interval_minutes: 60
     metric_resolution: "5m"
   ```

5. **Run the Go backend:**
   ```powershell
   go run cmd/optimizer/main.go -config configs/config-local.yaml
   ```

   The optimizer should now be running on `http://localhost:8080`
   
   **Test it:**
   ```powershell
   # In a new terminal
   curl http://localhost:8080/api/v1/health
   ```

---

## 🧪 Testing Without Kubernetes/Prometheus

If you don't have a Kubernetes cluster or Prometheus available, you can still test the ML service independently:

### Test ML Service Directly

1. **Create a test script** `test_ml_service.py`:
   ```python
   import requests
   import json
   from datetime import datetime, timedelta
   
   # Generate sample data
   base_time = datetime.now() - timedelta(days=7)
   sample_data = []
   
   for i in range(100):
       timestamp = (base_time + timedelta(hours=i)).isoformat()
       value = 0.3 + (0.1 * (i % 24) / 24)  # Simulated daily pattern
       sample_data.append({"timestamp": timestamp, "value": value})
   
   # Test CPU prediction
   payload = {
       "container": "test-container",
       "cpu": sample_data
   }
   
   response = requests.post(
       "http://localhost:5000/predict/cpu",
       json=payload
   )
   
   print("CPU Prediction Response:")
   print(json.dumps(response.json(), indent=2))
   ```

2. **Run the test:**
   ```powershell
   python test_ml_service.py
   ```

---

## 🐛 Troubleshooting

### Python Issues

**Problem:** `SyntaxError` in Python files
- **Solution:** There might be encoding issues. Close the file in your editor and reopen it, or use the fix from the previous conversation.

**Problem:** Module not found errors
- **Solution:** Make sure your virtual environment is activated:
  ```powershell
  .\venv\Scripts\Activate.ps1
  ```

**Problem:** TensorFlow installation fails
- **Solution:** On Windows, you might need Visual C++ redistributables:
  - Download from: https://aka.ms/vs/17/release/vc_redist.x64.exe

### Go Issues

**Problem:** Package import errors
- **Solution:** Run `go mod tidy` to fix module dependencies

**Problem:** Cannot connect to Kubernetes
- **Solution:** 
  - Verify your kubeconfig: `kubectl cluster-info`
  - Make sure `in_cluster: false` in config-local.yaml
  - Check kubeconfig path is correct

**Problem:** Cannot connect to Prometheus
- **Solution:** 
  - Verify Prometheus is running: `curl http://localhost:9090/-/healthy`
  - If using Kubernetes Prometheus, port-forward it:
    ```powershell
    kubectl port-forward -n monitoring svc/prometheus-server 9090:9090
    ```

### ML Service Issues

**Problem:** Predictions fail with low confidence
- **Solution:** This is normal with limited or synthetic data. Real historical metrics will improve predictions.

**Problem:** LSTM training is slow
- **Solution:** 
  - This is expected for the first run
  - Consider using CPU-only TensorFlow for development
  - Reduce data size for testing

---

## 📁 Project Structure (Quick Reference)

```
d:\k8s-resource-optimizer\
├── cmd/
│   └── optimizer/          # Go main entry point
├── internal/               # Go backend packages
│   ├── api/                # REST API server
│   ├── collector/          # Prometheus collector
│   ├── config/             # Configuration management
│   ├── k8s/                # Kubernetes client
│   ├── optimizer/          # Optimization engine
│   └── pipeline/           # Data preprocessing
├── ml-service/
│   ├── api/
│   │   └── app.py          # Flask REST API
│   ├── models/
│   │   ├── ensemble.py     # Ensemble predictor
│   │   ├── prophet_model.py
│   │   └── lstm_model.py
│   ├── requirements.txt    # Python dependencies
│   └── venv/               # Virtual environment (created)
├── configs/
│   ├── config.yaml         # Base config
│   └── config-local.yaml   # Local dev config (create this)
└── LOCAL_SETUP.md          # This file
```

---

## 🔄 Development Workflow

### Making Changes

1. **Python changes (ML models):**
   - Edit files in `ml-service/models/`
   - Restart the Flask server (Ctrl+C and run again)

2. **Go changes (backend):**
   - Edit files in `internal/` or `cmd/`
   - Stop and restart the Go application

### Hot Reload Development

For faster development:

**Python (Flask):**
```powershell
# Enable debug mode in app.py - change last line to:
# app.run(host='0.0.0.0', port=5000, debug=True)
```

**Go (Air - hot reload):**
```powershell
# Install Air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

---

## 📊 Viewing Results

### API Endpoints

Once both services are running:

1. **Health Check:**
   ```powershell
   curl http://localhost:8080/api/v1/health
   curl http://localhost:5000/health
   ```

2. **List Pods (requires K8s connection):**
   ```powershell
   curl http://localhost:8080/api/v1/pods
   ```

3. **Get Recommendations:**
   ```powershell
   curl http://localhost:8080/api/v1/recommendations
   ```

4. **Specific Pod Recommendation:**
   ```powershell
   curl http://localhost:8080/api/v1/recommendations/default/my-pod
   ```

### Using a REST Client

For easier testing, use:
- **Postman**: Import endpoints and test with UI
- **VS Code REST Client**: Create `.http` files with requests
- **Insomnia**: Another great API testing tool

---

## 🚀 Next Steps

1. **Test with Real Data**: Connect to a real Kubernetes cluster with Prometheus
2. **Experiment with Models**: Modify prediction models in `ml-service/models/`
3. **Add Features**: Extend the API or add new optimization strategies
4. **Deploy**: Once tested, deploy to Kubernetes using the deployment manifests

---

## 💡 Tips

- **Keep ML service running**: It takes time to load TensorFlow, so leave it running
- **Use dry-run mode**: Always test recommendations before applying (dry_run: true)
- **Check logs**: Both services output detailed logs for debugging
- **Start small**: Test with a single namespace or pod first
- **Monitor resources**: ML service can use significant RAM during predictions

---

## 📞 Getting Help

If you encounter issues:
1. Check the logs from both services
2. Verify network connectivity between components
3. Ensure all prerequisites are met
4. Review the main [README.md](README.md) and [ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

**Happy coding! 🎉**
