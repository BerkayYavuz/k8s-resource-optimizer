package collector

import (
	"time"
)

// MetricType represents the type of metric (CPU or Memory)
type MetricType string

const (
	MetricTypeCPU    MetricType = "cpu"
	MetricTypeMemory MetricType = "memory"
)

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// TimeSeries represents a complete time series for a metric
type TimeSeries struct {
	PodName     string            `json:"pod_name"`
	Namespace   string            `json:"namespace"`
	Container   string            `json:"container"`
	MetricType  MetricType        `json:"metric_type"`
	DataPoints  []TimeSeriesPoint `json:"data_points"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
}

// PodMetrics contains all metrics for a specific pod
type PodMetrics struct {
	PodName    string       `json:"pod_name"`
	Namespace  string       `json:"namespace"`
	Containers []string     `json:"containers"`
	CPU        []TimeSeries `json:"cpu"`
	Memory     []TimeSeries `json:"memory"`
}

// PrometheusQueryResult represents a raw Prometheus query response
type PrometheusQueryResult struct {
	Status string             `json:"status"`
	Data   PrometheusQueryData `json:"data"`
}

// PrometheusQueryData contains the actual query data
type PrometheusQueryData struct {
	ResultType string                   `json:"resultType"`
	Result     []PrometheusMatrixResult `json:"result"`
}

// PrometheusMatrixResult represents a matrix result from Prometheus
type PrometheusMatrixResult struct {
	Metric map[string]string    `json:"metric"`
	Values [][]interface{}      `json:"values"`
}

// MetricStats contains statistical information about a metric
type MetricStats struct {
	Average    float64 `json:"average"`
	Peak       float64 `json:"peak"`
	Min        float64 `json:"min"`
	P50        float64 `json:"p50"`
	P95        float64 `json:"p95"`
	P99        float64 `json:"p99"`
	StdDev     float64 `json:"std_dev"`
	DataPoints int     `json:"data_points"`
}
