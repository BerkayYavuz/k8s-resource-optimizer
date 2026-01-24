package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/k8s-resource-optimizer/optimizer/internal/config"
)

// Collector handles metrics collection from Prometheus
type Collector struct {
	client     *http.Client
	config     *config.PrometheusConfig
	baseURL    string
}

// NewCollector creates a new Prometheus metrics collector
func NewCollector(cfg *config.PrometheusConfig) (*Collector, error) {
	timeout, err := cfg.GetTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}

	return &Collector{
		client: &http.Client{
			Timeout: timeout,
		},
		config:  cfg,
		baseURL: cfg.URL,
	}, nil
}

// CollectPodMetrics collects CPU and memory metrics for a specific pod
func (c *Collector) CollectPodMetrics(ctx context.Context, namespace, podName string, windowDays int) (*PodMetrics, error) {
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(windowDays) * 24 * time.Hour)

	// Collect CPU metrics
	cpuMetrics, err := c.queryMetric(ctx, namespace, podName, MetricTypeCPU, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to collect CPU metrics: %w", err)
	}

	// Collect memory metrics
	memMetrics, err := c.queryMetric(ctx, namespace, podName, MetricTypeMemory, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to collect memory metrics: %w", err)
	}

	// Extract unique container names
	containers := make(map[string]bool)
	for _, ts := range cpuMetrics {
		containers[ts.Container] = true
	}
	for _, ts := range memMetrics {
		containers[ts.Container] = true
	}

	containerList := make([]string, 0, len(containers))
	for c := range containers {
		containerList = append(containerList, c)
	}
	sort.Strings(containerList)

	return &PodMetrics{
		PodName:    podName,
		Namespace:  namespace,
		Containers: containerList,
		CPU:        cpuMetrics,
		Memory:     memMetrics,
	}, nil
}

// queryMetric queries Prometheus for a specific metric type
func (c *Collector) queryMetric(ctx context.Context, namespace, podName string, metricType MetricType, start, end time.Time) ([]TimeSeries, error) {
	query := c.buildQuery(namespace, podName, metricType)
	
	log.Printf("Querying Prometheus: %s (from %s to %s)", query, start.Format(time.RFC3339), end.Format(time.RFC3339))

	result, err := c.queryRange(ctx, query, start, end, 5*time.Minute)
	if err != nil {
		return nil, err
	}

	return c.parseResults(result, metricType)
}

// buildQuery constructs a Prometheus query for the specified metric
func (c *Collector) buildQuery(namespace, podName string, metricType MetricType) string {
	switch metricType {
	case MetricTypeCPU:
		// Rate of CPU usage in cores
		return fmt.Sprintf(
			`rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s",container!="",container!="POD"}[5m])`,
			namespace, podName,
		)
	case MetricTypeMemory:
		// Working set memory in bytes
		return fmt.Sprintf(
			`container_memory_working_set_bytes{namespace="%s",pod="%s",container!="",container!="POD"}`,
			namespace, podName,
		)
	default:
		return ""
	}
}

// queryRange executes a range query against Prometheus
func (c *Collector) queryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*PrometheusQueryResult, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", fmt.Sprintf("%d", start.Unix()))
	params.Add("end", fmt.Sprintf("%d", end.Unix()))
	params.Add("step", fmt.Sprintf("%ds", int(step.Seconds())))

	reqURL := fmt.Sprintf("%s/api/v1/query_range?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication if configured
	if c.config.Username != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result PrometheusQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed with status: %s", result.Status)
	}

	return &result, nil
}

// parseResults converts Prometheus query results into TimeSeries
func (c *Collector) parseResults(result *PrometheusQueryResult, metricType MetricType) ([]TimeSeries, error) {
	var series []TimeSeries

	for _, matrixResult := range result.Data.Result {
		ts := TimeSeries{
			MetricType: metricType,
			DataPoints: make([]TimeSeriesPoint, 0, len(matrixResult.Values)),
		}

		// Extract labels
		if pod, ok := matrixResult.Metric["pod"]; ok {
			ts.PodName = pod
		}
		if ns, ok := matrixResult.Metric["namespace"]; ok {
			ts.Namespace = ns
		}
		if container, ok := matrixResult.Metric["container"]; ok {
			ts.Container = container
		}

		// Parse data points
		for _, value := range matrixResult.Values {
			if len(value) != 2 {
				continue
			}

			timestamp, ok := value[0].(float64)
			if !ok {
				continue
			}

			valueStr, ok := value[1].(string)
			if !ok {
				continue
			}

			var floatValue float64
			if _, err := fmt.Sscanf(valueStr, "%f", &floatValue); err != nil {
				log.Printf("Warning: failed to parse value %s: %v", valueStr, err)
				continue
			}

			point := TimeSeriesPoint{
				Timestamp: time.Unix(int64(timestamp), 0),
				Value:     floatValue,
			}

			ts.DataPoints = append(ts.DataPoints, point)

			// Track time range
			if ts.StartTime.IsZero() || point.Timestamp.Before(ts.StartTime) {
				ts.StartTime = point.Timestamp
			}
			if ts.EndTime.IsZero() || point.Timestamp.After(ts.EndTime) {
				ts.EndTime = point.Timestamp
			}
		}

		if len(ts.DataPoints) > 0 {
			series = append(series, ts)
		}
	}

	return series, nil
}

// CollectAllPods collects metrics for all pods in specified namespaces
func (c *Collector) CollectAllPods(ctx context.Context, namespaces []string, windowDays int) ([]PodMetrics, error) {
	// This would be optimized to query all pods at once
	// For now, we return empty - the actual implementation would query:
	// rate(container_cpu_usage_seconds_total{namespace=~"ns1|ns2",container!=""}[5m])
	// This is a placeholder that shows the structure
	log.Printf("CollectAllPods called for namespaces: %v (window: %d days)", namespaces, windowDays)
	return []PodMetrics{}, nil
}
