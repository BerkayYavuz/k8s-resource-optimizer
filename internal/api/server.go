package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/k8s-resource-optimizer/optimizer/internal/optimizer"
)

// Server handles HTTP API requests
type Server struct {
	router          *mux.Router
	port            int
	recommendations map[string]*optimizer.PodRecommendation // keyed by namespace/podname
}

// NewServer creates a new API server
func NewServer(port int) *Server {
	s := &Server{
		router:          mux.NewRouter(),
		port:            port,
		recommendations: make(map[string]*optimizer.PodRecommendation),
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/pods", s.handleListPods).Methods("GET")
	api.HandleFunc("/recommendations", s.handleListRecommendations).Methods("GET")
	api.HandleFunc("/recommendations/{namespace}/{pod}", s.handleGetRecommendation).Methods("GET")
	api.HandleFunc("/workloads", s.handleListWorkloads).Methods("GET")
	api.HandleFunc("/metrics", s.handleMetrics).Methods("GET")

	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Starting API server on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

// StoreRecommendation stores a recommendation
func (s *Server) StoreRecommendation(rec *optimizer.PodRecommendation) {
	key := fmt.Sprintf("%s/%s", rec.Namespace, rec.PodName)
	s.recommendations[key] = rec
	log.Printf("Stored recommendation for %s", key)
}

// Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleListPods(w http.ResponseWriter, r *http.Request) {
	pods := make([]string, 0, len(s.recommendations))
	for key := range s.recommendations {
		pods = append(pods, key)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pods":  pods,
		"count": len(pods),
	})
}

func (s *Server) handleListRecommendations(w http.ResponseWriter, r *http.Request) {
	recs := make([]*optimizer.PodRecommendation, 0, len(s.recommendations))
	for _, rec := range s.recommendations {
		recs = append(recs, rec)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"recommendations": recs,
		"count":           len(recs),
	})
}

func (s *Server) handleGetRecommendation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	pod := vars["pod"]

	key := fmt.Sprintf("%s/%s", namespace, pod)
	rec, exists := s.recommendations[key]

	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("no recommendation found for %s", key),
		})
		return
	}

	// Check if YAML format requested
	if r.URL.Query().Get("format") == "yaml" {
		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rec.GenerateYAMLPatch()))
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleListWorkloads(w http.ResponseWriter, r *http.Request) {
	workloads := make(map[string]*WorkloadSummary)

	for _, rec := range s.recommendations {
		workloadName := rec.WorkloadName
		if workloadName == "" {
			workloadName = rec.PodName
		}
		workloadType := rec.WorkloadType
		if workloadType == "" {
			workloadType = "Pod"
		}

		key := fmt.Sprintf("%s/%s/%s", rec.Namespace, workloadType, workloadName)
		summary, exists := workloads[key]
		if !exists {
			summary = &WorkloadSummary{
				Namespace:    rec.Namespace,
				WorkloadName: workloadName,
				WorkloadType: workloadType,
				StatusCounts: make(map[string]int),
				Pods:         []WorkloadPod{},
			}
			workloads[key] = summary
		}

		summary.PodCount++
		summary.CPUSavingsCores += rec.Impact.CPUSavingsCores
		summary.CPUAdditionalCores += rec.Impact.CPUAdditionalCores
		summary.MemorySavingsMB += rec.Impact.MemorySavingsMB
		summary.MemoryAdditionalMB += rec.Impact.MemoryAdditionalMB
		summary.AvgConfidence += rec.OverallConfidence
		summary.StatusCounts[rec.Status]++
		summary.Pods = append(summary.Pods, WorkloadPod{
			PodName:    rec.PodName,
			NodeName:   rec.NodeName,
			Status:     rec.Status,
			Severity:   rec.Severity,
			Confidence: rec.OverallConfidence,
		})
	}

	list := make([]*WorkloadSummary, 0, len(workloads))
	for _, summary := range workloads {
		if summary.PodCount > 0 {
			summary.AvgConfidence /= float64(summary.PodCount)
		}
		summary.Status = dominantStatus(summary.StatusCounts)
		summary.Severity = dominantSeverity(summary.Pods)
		list = append(list, summary)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"workloads": list,
		"count":     len(list),
	})
}

type WorkloadSummary struct {
	Namespace          string         `json:"namespace"`
	WorkloadName       string         `json:"workload_name"`
	WorkloadType       string         `json:"workload_type"`
	Status             string         `json:"status"`
	Severity           string         `json:"severity"`
	PodCount           int            `json:"pod_count"`
	StatusCounts       map[string]int `json:"status_counts"`
	CPUSavingsCores    float64        `json:"cpu_savings_cores"`
	CPUAdditionalCores float64        `json:"cpu_additional_cores"`
	MemorySavingsMB    int64          `json:"memory_savings_mb"`
	MemoryAdditionalMB int64          `json:"memory_additional_mb"`
	AvgConfidence      float64        `json:"avg_confidence"`
	Pods               []WorkloadPod  `json:"pods"`
}

type WorkloadPod struct {
	PodName    string  `json:"pod_name"`
	NodeName   string  `json:"node_name"`
	Status     string  `json:"status"`
	Severity   string  `json:"severity"`
	Confidence float64 `json:"confidence"`
}

func dominantStatus(counts map[string]int) string {
	priority := []string{"under_provisioned", "low_confidence", "over_provisioned", "balanced"}
	for _, status := range priority {
		if counts[status] > 0 {
			return status
		}
	}
	return "unknown"
}

func dominantSeverity(pods []WorkloadPod) string {
	rank := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}
	highest := "low"
	for _, pod := range pods {
		if rank[pod.Severity] > rank[highest] {
			highest = pod.Severity
		}
	}
	return highest
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Prometheus-format metrics
	totalPods := len(s.recommendations)
	totalCPUSavings := 0.0
	totalMemSavings := int64(0)
	avgConfidence := 0.0

	for _, rec := range s.recommendations {
		totalCPUSavings += rec.PotentialCPUSavingCores
		totalMemSavings += rec.PotentialMemorySavingMB
		avgConfidence += rec.OverallConfidence
	}

	if totalPods > 0 {
		avgConfidence /= float64(totalPods)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "# HELP k8s_optimizer_pods_analyzed Total number of pods analyzed\n")
	fmt.Fprintf(w, "# TYPE k8s_optimizer_pods_analyzed gauge\n")
	fmt.Fprintf(w, "k8s_optimizer_pods_analyzed %d\n", totalPods)

	fmt.Fprintf(w, "# HELP k8s_optimizer_cpu_savings_cores Potential CPU savings in cores\n")
	fmt.Fprintf(w, "# TYPE k8s_optimizer_cpu_savings_cores gauge\n")
	fmt.Fprintf(w, "k8s_optimizer_cpu_savings_cores %.3f\n", totalCPUSavings)

	fmt.Fprintf(w, "# HELP k8s_optimizer_memory_savings_mb Potential memory savings in MB\n")
	fmt.Fprintf(w, "# TYPE k8s_optimizer_memory_savings_mb gauge\n")
	fmt.Fprintf(w, "k8s_optimizer_memory_savings_mb %d\n", totalMemSavings)

	fmt.Fprintf(w, "# HELP k8s_optimizer_avg_confidence Average prediction confidence\n")
	fmt.Fprintf(w, "# TYPE k8s_optimizer_avg_confidence gauge\n")
	fmt.Fprintf(w, "k8s_optimizer_avg_confidence %.3f\n", avgConfidence)
}

// Middleware

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("%s %s completed in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}
