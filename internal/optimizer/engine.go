package optimizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/k8s-resource-optimizer/optimizer/internal/config"
	"github.com/k8s-resource-optimizer/optimizer/internal/k8s"
	"github.com/k8s-resource-optimizer/optimizer/internal/pipeline"
)

// Engine handles the optimization logic
type Engine struct {
	config    *config.OptimizerConfig
	mlClient  *http.Client
	mlBaseURL string
}

// NewEngine creates a new optimization engine
func NewEngine(cfg *config.Config) (*Engine, error) {
	timeout, err := cfg.MLService.GetTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid ML service timeout: %w", err)
	}

	return &Engine{
		config: &cfg.Optimizer,
		mlClient: &http.Client{
			Timeout: timeout,
		},
		mlBaseURL: cfg.MLService.URL,
	}, nil
}

// GenerateRecommendations creates resource recommendations for a pod
func (e *Engine) GenerateRecommendations(ctx context.Context, mlInput *pipeline.MLInput, currentResources map[string]k8s.ContainerResources) (*PodRecommendation, error) {
	// Get ML predictions
	predictions, err := e.getPredictions(ctx, mlInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get ML predictions: %w", err)
	}

	recommendation := &PodRecommendation{
		PodName:     mlInput.PodName,
		Namespace:   mlInput.Namespace,
		Timestamp:   time.Now(),
		Containers:  make(map[string]ContainerRecommendation),
		DryRun:      e.config.DryRun,
	}

	totalCurrentCPU := 0.0
	totalRecommendedCPU := 0.0
	totalCurrentMemory := int64(0)
	totalRecommendedMemory := int64(0)

	// Process each container
	for containerName, pred := range predictions.Predictions {
		current, exists := currentResources[containerName]
		if !exists {
			log.Printf("Warning: no current resources found for container %s", containerName)
			current = k8s.ContainerResources{Name: containerName}
		}

		// Calculate recommended resources
		rec := e.calculateRecommendation(pred, current)
		recommendation.Containers[containerName] = rec

		// Accumulate totals for pod-level metrics
		totalCurrentCPU += current.CPURequest
		totalRecommendedCPU += rec.RecommendedCPURequest
		totalCurrentMemory += current.MemoryRequest
		totalRecommendedMemory += rec.RecommendedMemoryRequest
	}

	// Calculate savings
	recommendation.PotentialCPUSavingCores = totalCurrentCPU - totalRecommendedCPU
	recommendation.PotentialMemorySavingMB = totalCurrentMemory - totalRecommendedMemory

	// Calculate waste percentage
	if totalCurrentCPU > 0 {
		recommendation.CPUWastePercentage = (recommendation.PotentialCPUSavingCores / totalCurrentCPU) * 100
	}
	if totalCurrentMemory > 0 {
		recommendation.MemoryWastePercentage = float64(recommendation.PotentialMemorySavingMB) / float64(totalCurrentMemory) * 100
	}

	// Overall confidence (average of container confidences)
	totalConfidence := 0.0
	for _, pred := range predictions.Predictions {
		totalConfidence += pred.Confidence
	}
	if len(predictions.Predictions) > 0 {
		recommendation.OverallConfidence = totalConfidence / float64(len(predictions.Predictions))
	}

	return recommendation, nil
}

// calculateRecommendation calculates recommended resources for a single container
func (e *Engine) calculateRecommendation(pred pipeline.MLPrediction, current k8s.ContainerResources) ContainerRecommendation {
	rec := ContainerRecommendation{
		Container: pred.Container,
		Current: CurrentResources{
			CPURequest:    current.CPURequest,
			CPULimit:      current.CPULimit,
			MemoryRequest: current.MemoryRequest,
			MemoryLimit:   current.MemoryLimit,
		},
		Predicted: PredictedUsage{
			AvgCPU:  pred.PredictedAvgCPU,
			PeakCPU: pred.PredictedPeakCPU,
			AvgMem:  int64(pred.PredictedAvgMem),
			PeakMem: int64(pred.PredictedPeakMem),
		},
		Confidence: pred.Confidence,
	}

	// Calculate recommended requests (based on average + safety margin)
	recCPURequest := pred.PredictedAvgCPU * (1 + e.config.SafetyMarginRequests)
	recMemRequest := int64(pred.PredictedAvgMem * (1 + e.config.SafetyMarginRequests))

	// Calculate recommended limits (based on peak + safety margin)
	recCPULimit := pred.PredictedPeakCPU * (1 + e.config.SafetyMarginLimits)
	recMemLimit := int64(pred.PredictedPeakMem * (1 + e.config.SafetyMarginLimits))

	// Apply minimum thresholds
	recCPURequest = math.Max(recCPURequest, e.config.MinThresholds.CPUCores)
	recCPULimit = math.Max(recCPULimit, e.config.MinThresholds.CPUCores)
	recMemRequest = maxInt64(recMemRequest, e.config.MinThresholds.MemoryMB)
	recMemLimit = maxInt64(recMemLimit, e.config.MinThresholds.MemoryMB)

	// Ensure limits >= requests
	recCPULimit = math.Max(recCPULimit, recCPURequest)
	recMemLimit = maxInt64(recMemLimit, recMemRequest)

	rec.RecommendedCPURequest = recCPURequest
	rec.RecommendedCPULimit = recCPULimit
	rec.RecommendedMemoryRequest = recMemRequest
	rec.RecommendedMemoryLimit = recMemLimit

	// Calculate change percentages
	if current.CPURequest > 0 {
		rec.CPURequestChange = ((recCPURequest - current.CPURequest) / current.CPURequest) * 100
	}
	if current.CPULimit > 0 {
		rec.CPULimitChange = ((recCPULimit - current.CPULimit) / current.CPULimit) * 100
	}
	if current.MemoryRequest > 0 {
		rec.MemoryRequestChange = (float64(recMemRequest-current.MemoryRequest) / float64(current.MemoryRequest)) * 100
	}
	if current.MemoryLimit > 0 {
		rec.MemoryLimitChange = (float64(recMemLimit-current.MemoryLimit) / float64(current.MemoryLimit)) * 100
	}

	return rec
}

// getPredictions calls the ML service to get predictions
func (e *Engine) getPredictions(ctx context.Context, mlInput *pipeline.MLInput) (*pipeline.MLResponse, error) {
	jsonData, err := mlInput.ToJSON()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/predict/all", e.mlBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.mlClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ML service request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ML service response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned status %d: %s", resp.StatusCode, string(body))
	}

	var mlResponse pipeline.MLResponse
	if err := json.Unmarshal(body, &mlResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ML response: %w", err)
	}

	if !mlResponse.Success {
		return nil, fmt.Errorf("ML service returned error: %s", mlResponse.Error)
	}

	return &mlResponse, nil
}

// GenerateYAMLPatch creates a Kubernetes patch YAML for the recommendation
func (rec *PodRecommendation) GenerateYAMLPatch() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("# Resource recommendation for pod: %s/%s\n", rec.Namespace, rec.PodName))
	buf.WriteString(fmt.Sprintf("# Generated: %s\n", rec.Timestamp.Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("# Confidence: %.2f%%\n", rec.OverallConfidence*100))
	if rec.DryRun {
		buf.WriteString("# DRY RUN MODE - Review before applying\n")
	}
	buf.WriteString("\n")

	buf.WriteString("spec:\n")
	buf.WriteString("  containers:\n")

	for _, container := range rec.Containers {
		buf.WriteString(fmt.Sprintf("  - name: %s\n", container.Container))
		buf.WriteString("    resources:\n")
		buf.WriteString("      requests:\n")
		buf.WriteString(fmt.Sprintf("        cpu: \"%.0fm\"  # was: %.0fm (change: %+.1f%%)\n",
			container.RecommendedCPURequest*1000,
			container.Current.CPURequest*1000,
			container.CPURequestChange))
		buf.WriteString(fmt.Sprintf("        memory: \"%dMi\"  # was: %dMi (change: %+.1f%%)\n",
			container.RecommendedMemoryRequest,
			container.Current.MemoryRequest,
			container.MemoryRequestChange))
		buf.WriteString("      limits:\n")
		buf.WriteString(fmt.Sprintf("        cpu: \"%.0fm\"  # was: %.0fm (change: %+.1f%%)\n",
			container.RecommendedCPULimit*1000,
			container.Current.CPULimit*1000,
			container.CPULimitChange))
		buf.WriteString(fmt.Sprintf("        memory: \"%dMi\"  # was: %dMi (change: %+.1f%%)\n",
			container.RecommendedMemoryLimit,
			container.Current.MemoryLimit,
			container.MemoryLimitChange))
		buf.WriteString("\n")
	}

	return buf.String()
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
