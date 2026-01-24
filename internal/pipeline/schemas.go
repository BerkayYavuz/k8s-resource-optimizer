package pipeline

// MLInput represents the input format for the ML service
type MLInput struct {
	PodName   string                      `json:"pod_name"`
	Namespace string                      `json:"namespace"`
	Metrics   map[string]ContainerMetrics `json:"metrics"`
}

// ContainerMetrics contains processed metrics for a single container
type ContainerMetrics struct {
	Container   string            `json:"container"`
	CPU         []DataPoint       `json:"cpu"`
	Memory      []DataPoint       `json:"memory"`
	CPUStats    MetricStatistics  `json:"cpu_stats"`
	MemoryStats MetricStatistics  `json:"memory_stats"`
}

// DataPoint represents a single time-value pair
type DataPoint struct {
	Timestamp string  `json:"timestamp"` // ISO 8601 format
	Value     float64 `json:"value"`
}

// MetricStatistics contains statistical analysis of metrics
type MetricStatistics struct {
	Average    float64 `json:"average"`
	Peak       float64 `json:"peak"`
	Min        float64 `json:"min"`
	P50        float64 `json:"p50"`
	P95        float64 `json:"p95"`
	P99        float64 `json:"p99"`
	StdDev     float64 `json:"std_dev"`
	DataPoints int     `json:"data_points"`
}

// MLPrediction represents the ML service response
type MLPrediction struct {
	Container        string  `json:"container"`
	PredictedAvgCPU  float64 `json:"predicted_avg_cpu"`
	PredictedPeakCPU float64 `json:"predicted_peak_cpu"`
	PredictedAvgMem  float64 `json:"predicted_avg_memory"` // in MB
	PredictedPeakMem float64 `json:"predicted_peak_memory"` // in MB
	Confidence       float64 `json:"confidence"`
	Model            string  `json:"model"` // "prophet", "lstm", "ensemble"
}

// MLResponse represents the complete response from ML service
type MLResponse struct {
	PodName     string                  `json:"pod_name"`
	Namespace   string                  `json:"namespace"`
	Predictions map[string]MLPrediction `json:"predictions"` // keyed by container name
	Timestamp   string                  `json:"timestamp"`
	Success     bool                    `json:"success"`
	Error       string                  `json:"error,omitempty"`
}
