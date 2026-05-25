package optimizer

import (
	"time"
)

// PodRecommendation contains resource recommendations for a pod
type PodRecommendation struct {
	PodName                 string                             `json:"pod_name"`
	Namespace               string                             `json:"namespace"`
	NodeName                string                             `json:"node_name"`
	WorkloadName            string                             `json:"workload_name"`
	WorkloadType            string                             `json:"workload_type"`
	Timestamp               time.Time                          `json:"timestamp"`
	Containers              map[string]ContainerRecommendation `json:"containers"`
	Status                  string                             `json:"status"`
	Severity                string                             `json:"severity"`
	Reason                  string                             `json:"reason"`
	Impact                  RecommendationImpact               `json:"impact"`
	PotentialCPUSavingCores float64                            `json:"potential_cpu_saving_cores"`
	PotentialMemorySavingMB int64                              `json:"potential_memory_saving_mb"`
	CPUWastePercentage      float64                            `json:"cpu_waste_percentage"`
	MemoryWastePercentage   float64                            `json:"memory_waste_percentage"`
	OverallConfidence       float64                            `json:"overall_confidence"`
	DryRun                  bool                               `json:"dry_run"`
}

// RecommendationImpact gives UI-friendly positive/negative resource impact.
type RecommendationImpact struct {
	CPUSavingsCores          float64 `json:"cpu_savings_cores"`
	CPUAdditionalCores       float64 `json:"cpu_additional_cores"`
	MemorySavingsMB          int64   `json:"memory_savings_mb"`
	MemoryAdditionalMB       int64   `json:"memory_additional_mb"`
	HasSavings               bool    `json:"has_savings"`
	RequiresAdditionalMemory bool    `json:"requires_additional_memory"`
	RequiresAdditionalCPU    bool    `json:"requires_additional_cpu"`
}

// ContainerRecommendation contains recommendations for a single container
type ContainerRecommendation struct {
	Container                string           `json:"container"`
	Current                  CurrentResources `json:"current"`
	Predicted                PredictedUsage   `json:"predicted"`
	RecommendedCPURequest    float64          `json:"recommended_cpu_request"`
	RecommendedCPULimit      float64          `json:"recommended_cpu_limit"`
	RecommendedMemoryRequest int64            `json:"recommended_memory_request"`
	RecommendedMemoryLimit   int64            `json:"recommended_memory_limit"`
	CPURequestChange         float64          `json:"cpu_request_change_percent"`
	CPULimitChange           float64          `json:"cpu_limit_change_percent"`
	MemoryRequestChange      float64          `json:"memory_request_change_percent"`
	MemoryLimitChange        float64          `json:"memory_limit_change_percent"`
	Confidence               float64          `json:"confidence"`
}

// CurrentResources represents the currently configured resources
type CurrentResources struct {
	CPURequest    float64 `json:"cpu_request"`
	CPULimit      float64 `json:"cpu_limit"`
	MemoryRequest int64   `json:"memory_request"`
	MemoryLimit   int64   `json:"memory_limit"`
}

// PredictedUsage represents ML-predicted resource usage
type PredictedUsage struct {
	AvgCPU  float64 `json:"avg_cpu"`
	PeakCPU float64 `json:"peak_cpu"`
	AvgMem  int64   `json:"avg_memory"`
	PeakMem int64   `json:"peak_memory"`
}
