package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k8s-resource-optimizer/optimizer/internal/api"
	"github.com/k8s-resource-optimizer/optimizer/internal/collector"
	"github.com/k8s-resource-optimizer/optimizer/internal/config"
	"github.com/k8s-resource-optimizer/optimizer/internal/k8s"
	"github.com/k8s-resource-optimizer/optimizer/internal/optimizer"
	"github.com/k8s-resource-optimizer/optimizer/internal/pipeline"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	flag.Parse()

	log.Println("Starting Kubernetes Resource Optimizer AI...")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded from %s", *configPath)
	log.Printf("Dry-run mode: %v", cfg.Optimizer.DryRun)

	// Initialize components
	promCollector, err := collector.NewCollector(&cfg.Prometheus)
	if err != nil {
		log.Fatalf("Failed to create Prometheus collector: %v", err)
	}
	log.Println("Prometheus collector initialized")

	k8sClient, err := k8s.NewClient(&cfg.Kubernetes)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	log.Println("Kubernetes client initialized")

	dataPipeline := pipeline.NewPipeline()
	log.Println("Data pipeline initialized")

	optimizationEngine, err := optimizer.NewEngine(cfg)
	if err != nil {
		log.Fatalf("Failed to create optimization engine: %v", err)
	}
	log.Println("Optimization engine initialized")

	// Start API server in a goroutine
	apiServer := api.NewServer(cfg.API.Port)
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run analysis loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(time.Duration(cfg.Analysis.IntervalMinutes) * time.Minute)
	defer ticker.Stop()

	// Run first analysis immediately
	log.Println("Running initial analysis...")
	runAnalysis(ctx, cfg, k8sClient, promCollector, dataPipeline, optimizationEngine, apiServer)

	// Main loop
	for {
		select {
		case <-ticker.C:
			log.Println("Running periodic analysis...")
			runAnalysis(ctx, cfg, k8sClient, promCollector, dataPipeline, optimizationEngine, apiServer)

		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down gracefully...", sig)
			cancel()
			return
		}
	}
}

// runAnalysis performs a complete analysis cycle
func runAnalysis(
	ctx context.Context,
	cfg *config.Config,
	k8sClient *k8s.Client,
	promCollector *collector.Collector,
	dataPipeline *pipeline.Pipeline,
	optimizationEngine *optimizer.Engine,
	apiServer *api.Server,
) {
	startTime := time.Now()

	// Get all pods from Kubernetes
	pods, err := k8sClient.ListPods(ctx)
	if err != nil {
		log.Printf("Error listing pods: %v", err)
		return
	}

	log.Printf("Found %d running pods to analyze", len(pods))

	successCount := 0
	errorCount := 0

	// Analyze each pod
	for _, pod := range pods {
		// Skip if namespace is excluded
		if k8sClient.IsNamespaceExcluded(pod.Namespace) {
			continue
		}

		log.Printf("Analyzing pod: %s/%s", pod.Namespace, pod.Name)

		// Collect metrics from Prometheus
		metrics, err := promCollector.CollectPodMetrics(
			ctx,
			pod.Namespace,
			pod.Name,
			cfg.Analysis.WindowDays,
		)
		if err != nil {
			log.Printf("Error collecting metrics for %s/%s: %v", pod.Namespace, pod.Name, err)
			errorCount++
			continue
		}

		// Check if we have enough data
		if len(metrics.CPU) == 0 && len(metrics.Memory) == 0 {
			log.Printf("No metrics available for %s/%s, skipping", pod.Namespace, pod.Name)
			continue
		}

		// Process metrics through pipeline
		mlInput, err := dataPipeline.ProcessMetrics(metrics)
		if err != nil {
			log.Printf("Error processing metrics for %s/%s: %v", pod.Namespace, pod.Name, err)
			errorCount++
			continue
		}

		// Get current resource configuration
		currentResources := k8s.GetPodResourceRequests(&pod)

		// Generate recommendations
		recommendation, err := optimizationEngine.GenerateRecommendations(ctx, mlInput, currentResources)
		if err != nil {
			log.Printf("Error generating recommendation for %s/%s: %v", pod.Namespace, pod.Name, err)
			errorCount++
			continue
		}

		// Store recommendation in API server
		apiServer.StoreRecommendation(recommendation)

		// Log summary
		log.Printf("Recommendation for %s/%s: CPU savings: %.3f cores (%.1f%%), Memory savings: %d MB (%.1f%%), Confidence: %.2f%%",
			pod.Namespace,
			pod.Name,
			recommendation.PotentialCPUSavingCores,
			recommendation.CPUWastePercentage,
			recommendation.PotentialMemorySavingMB,
			recommendation.MemoryWastePercentage,
			recommendation.OverallConfidence*100,
		)

		successCount++
	}

	elapsed := time.Since(startTime)
	log.Printf("Analysis complete: %d successful, %d errors, duration: %v", successCount, errorCount, elapsed)
}
