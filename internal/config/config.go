package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the complete system configuration
type Config struct {
	Prometheus PrometheusConfig `yaml:"prometheus"`
	MLService  MLServiceConfig  `yaml:"ml_service"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	Optimizer  OptimizerConfig  `yaml:"optimizer"`
	API        APIConfig        `yaml:"api"`
	Analysis   AnalysisConfig   `yaml:"analysis"`
}

// PrometheusConfig contains Prometheus connection settings
type PrometheusConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Timeout  string `yaml:"timeout"`
}

// MLServiceConfig contains ML service connection settings
type MLServiceConfig struct {
	URL     string `yaml:"url"`
	Timeout string `yaml:"timeout"`
}

// KubernetesConfig contains Kubernetes-specific settings
type KubernetesConfig struct {
	InCluster         bool     `yaml:"in_cluster"`
	KubeconfigPath    string   `yaml:"kubeconfig_path,omitempty"`
	ExcludeNamespaces []string `yaml:"exclude_namespaces"`
	ExcludeLabels     []string `yaml:"exclude_labels"`
}

// OptimizerConfig contains optimization engine settings
type OptimizerConfig struct {
	SafetyMarginRequests float64       `yaml:"safety_margin_requests"`
	SafetyMarginLimits   float64       `yaml:"safety_margin_limits"`
	MinCPUCores          float64       `yaml:"min_cpu_cores"`
	MinMemoryMB          int64         `yaml:"min_memory_mb"`
	DryRun               bool          `yaml:"dry_run"`
	MinThresholds        MinThresholds `yaml:"min_thresholds"`
}

// MinThresholds defines minimum resource thresholds
type MinThresholds struct {
	CPUCores float64 `yaml:"cpu_cores"`
	MemoryMB int64   `yaml:"memory_mb"`
}

// APIConfig contains API server settings
type APIConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// AnalysisConfig contains analysis behavior settings
type AnalysisConfig struct {
	WindowDays       int    `yaml:"window_days"`
	IntervalMinutes  int    `yaml:"interval_minutes"`
	MetricResolution string `yaml:"metric_resolution"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Apply defaults if not set
	applyDefaults(&config)

	return &config, nil
}

// applyDefaults sets default values for optional fields
func applyDefaults(cfg *Config) {
	if cfg.Prometheus.Timeout == "" {
		cfg.Prometheus.Timeout = "30s"
	}
	if cfg.MLService.Timeout == "" {
		cfg.MLService.Timeout = "60s"
	}
	if cfg.Optimizer.SafetyMarginRequests == 0 {
		cfg.Optimizer.SafetyMarginRequests = 0.15 // 15%
	}
	if cfg.Optimizer.SafetyMarginLimits == 0 {
		cfg.Optimizer.SafetyMarginLimits = 0.25 // 25%
	}
	if cfg.Optimizer.MinThresholds.CPUCores == 0 {
		cfg.Optimizer.MinThresholds.CPUCores = 0.01 // 10m
	}
	if cfg.Optimizer.MinThresholds.MemoryMB == 0 {
		cfg.Optimizer.MinThresholds.MemoryMB = 32 // 32MB
	}
	if cfg.Analysis.WindowDays == 0 {
		cfg.Analysis.WindowDays = 14
	}
	if cfg.Analysis.IntervalMinutes == 0 {
		cfg.Analysis.IntervalMinutes = 60
	}
	if cfg.Analysis.MetricResolution == "" {
		cfg.Analysis.MetricResolution = "5m"
	}
	if cfg.API.Port == 0 {
		cfg.API.Port = 8080
	}
	if cfg.API.Host == "" {
		cfg.API.Host = "0.0.0.0"
	}
	if len(cfg.Kubernetes.ExcludeNamespaces) == 0 {
		cfg.Kubernetes.ExcludeNamespaces = []string{
			"kube-system",
			"kube-public",
			"kube-node-lease",
		}
	}
}

// GetPrometheusDuration returns the Prometheus timeout as a time.Duration
func (c *PrometheusConfig) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}

// GetMLServiceDuration returns the ML service timeout as a time.Duration
func (c *MLServiceConfig) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}
