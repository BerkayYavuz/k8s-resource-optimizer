package pipeline

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/k8s-resource-optimizer/optimizer/internal/collector"
)

// Pipeline handles data preprocessing and transformation
type Pipeline struct {
	smoothingWindow int
}

// NewPipeline creates a new data pipeline
func NewPipeline() *Pipeline {
	return &Pipeline{
		smoothingWindow: 5, // 5-point rolling average
	}
}

// ProcessMetrics processes raw metrics into ML-ready format
func (p *Pipeline) ProcessMetrics(metrics *collector.PodMetrics) (*MLInput, error) {
	mlInput := &MLInput{
		PodName:   metrics.PodName,
		Namespace: metrics.Namespace,
		Metrics:   make(map[string]ContainerMetrics),
	}

	// Process each container
	for _, container := range metrics.Containers {
		containerMetrics := ContainerMetrics{
			Container: container,
		}

		// Find CPU time series for this container
		for _, cpuTS := range metrics.CPU {
			if cpuTS.Container == container {
				processed, stats := p.processTimeSeries(cpuTS.DataPoints, collector.MetricTypeCPU)
				containerMetrics.CPU = processed
				containerMetrics.CPUStats = stats
			}
		}

		// Find memory time series for this container
		for _, memTS := range metrics.Memory {
			if memTS.Container == container {
				processed, stats := p.processTimeSeries(memTS.DataPoints, collector.MetricTypeMemory)
				containerMetrics.Memory = processed
				containerMetrics.MemoryStats = stats
			}
		}

		mlInput.Metrics[container] = containerMetrics
	}

	return mlInput, nil
}

// processTimeSeries applies preprocessing to a time series
func (p *Pipeline) processTimeSeries(points []collector.TimeSeriesPoint, metricType collector.MetricType) ([]DataPoint, MetricStatistics) {
	if len(points) == 0 {
		return []DataPoint{}, MetricStatistics{}
	}

	// Sort by timestamp
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	// Fill gaps (if there are large time gaps, interpolate)
	filled := p.fillGaps(points)

	// Apply smoothing
	smoothed := p.applySmoothing(filled)

	// Remove outliers
	cleaned := p.removeOutliers(smoothed)

	// Calculate statistics
	stats := p.calculateStatistics(cleaned, metricType)

	// Convert to output format
	output := make([]DataPoint, len(cleaned))
	for i, point := range cleaned {
		output[i] = DataPoint{
			Timestamp: point.Timestamp.Format(time.RFC3339),
			Value:     point.Value,
		}
	}

	return output, stats
}

// fillGaps fills missing data points using linear interpolation
func (p *Pipeline) fillGaps(points []collector.TimeSeriesPoint) []collector.TimeSeriesPoint {
	if len(points) < 2 {
		return points
	}

	var filled []collector.TimeSeriesPoint
	filled = append(filled, points[0])

	for i := 1; i < len(points); i++ {
		prev := points[i-1]
		curr := points[i]

		// If gap is more than 15 minutes, interpolate
		gap := curr.Timestamp.Sub(prev.Timestamp)
		if gap > 15*time.Minute {
			// Add intermediate points
			steps := int(gap.Minutes() / 5) // Every 5 minutes
			if steps > 100 {
				steps = 100 // Limit interpolation
			}

			for s := 1; s < steps; s++ {
				fraction := float64(s) / float64(steps)
				interpTime := prev.Timestamp.Add(time.Duration(fraction * float64(gap)))
				interpValue := prev.Value + fraction*(curr.Value-prev.Value)

				filled = append(filled, collector.TimeSeriesPoint{
					Timestamp: interpTime,
					Value:     interpValue,
				})
			}
		}

		filled = append(filled, curr)
	}

	return filled
}

// applySmoothing applies a rolling average to smooth the data
func (p *Pipeline) applySmoothing(points []collector.TimeSeriesPoint) []collector.TimeSeriesPoint {
	if len(points) < p.smoothingWindow {
		return points
	}

	smoothed := make([]collector.TimeSeriesPoint, len(points))

	for i := range points {
		start := i - p.smoothingWindow/2
		end := i + p.smoothingWindow/2 + 1

		if start < 0 {
			start = 0
		}
		if end > len(points) {
			end = len(points)
		}

		sum := 0.0
		count := 0
		for j := start; j < end; j++ {
			sum += points[j].Value
			count++
		}

		smoothed[i] = collector.TimeSeriesPoint{
			Timestamp: points[i].Timestamp,
			Value:     sum / float64(count),
		}
	}

	return smoothed
}

// removeOutliers removes extreme outliers using IQR method
func (p *Pipeline) removeOutliers(points []collector.TimeSeriesPoint) []collector.TimeSeriesPoint {
	if len(points) < 4 {
		return points
	}

	// Calculate IQR
	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = p.Value
	}
	sort.Float64s(values)

	q1 := values[len(values)/4]
	q3 := values[3*len(values)/4]
	iqr := q3 - q1

	lowerBound := q1 - 3*iqr // 3x IQR for extreme outliers only
	upperBound := q3 + 3*iqr

	// Filter outliers
	var filtered []collector.TimeSeriesPoint
	for _, point := range points {
		if point.Value >= lowerBound && point.Value <= upperBound {
			filtered = append(filtered, point)
		}
	}

	// If we removed too many points, return original
	if len(filtered) < len(points)/2 {
		return points
	}

	return filtered
}

// calculateStatistics computes statistical metrics
func (p *Pipeline) calculateStatistics(points []collector.TimeSeriesPoint, metricType collector.MetricType) MetricStatistics {
	if len(points) == 0 {
		return MetricStatistics{}
	}

	values := make([]float64, len(points))
	sum := 0.0
	min := math.MaxFloat64
	max := -math.MaxFloat64

	for i, point := range points {
		val := point.Value
		values[i] = val
		sum += val

		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	avg := sum / float64(len(values))

	// Calculate standard deviation
	variance := 0.0
	for _, val := range values {
		diff := val - avg
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(values)))

	// Calculate percentiles
	sort.Float64s(values)
	p50 := values[len(values)/2]
	p95 := values[int(float64(len(values))*0.95)]
	p99 := values[int(float64(len(values))*0.99)]

	stats := MetricStatistics{
		Average:    avg,
		Peak:       max,
		Min:        min,
		P50:        p50,
		P95:        p95,
		P99:        p99,
		StdDev:     stdDev,
		DataPoints: len(values),
	}

	// Convert memory from bytes to MB if needed
	if metricType == collector.MetricTypeMemory {
		stats.Average = stats.Average / (1024 * 1024)
		stats.Peak = stats.Peak / (1024 * 1024)
		stats.Min = stats.Min / (1024 * 1024)
		stats.P50 = stats.P50 / (1024 * 1024)
		stats.P95 = stats.P95 / (1024 * 1024)
		stats.P99 = stats.P99 / (1024 * 1024)
		stats.StdDev = stats.StdDev / (1024 * 1024)
	}

	return stats
}

// ToJSON converts MLInput to JSON for the ML service
func (m *MLInput) ToJSON() ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ML input: %w", err)
	}
	return data, nil
}
