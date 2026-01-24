package k8s

// ContainerResources represents the resource requests and limits for a container
type ContainerResources struct {
	Name          string  `json:"name"`
	CPURequest    float64 `json:"cpu_request"`    // in cores
	CPULimit      float64 `json:"cpu_limit"`      // in cores
	MemoryRequest int64   `json:"memory_request"` // in MB
	MemoryLimit   int64   `json:"memory_limit"`   // in MB
}

// PodInfo contains information about a pod and its current resources
type PodInfo struct {
	Name       string                        `json:"name"`
	Namespace  string                        `json:"namespace"`
	Containers map[string]ContainerResources `json:"containers"`
	Labels     map[string]string             `json:"labels"`
	Age        string                        `json:"age"`
}
