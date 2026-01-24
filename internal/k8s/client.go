package k8s

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/k8s-resource-optimizer/optimizer/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes client with helper methods
type Client struct {
	clientset         *kubernetes.Clientset
	excludeNamespaces map[string]bool
}

// NewClient creates a new Kubernetes client
func NewClient(cfg *config.KubernetesConfig) (*Client, error) {
	var k8sConfig *rest.Config
	var err error

	if cfg.InCluster {
		log.Println("Using in-cluster Kubernetes configuration")
		k8sConfig, err = rest.InClusterConfig()
	} else {
		log.Printf("Using kubeconfig from: %s", cfg.KubeconfigPath)
		kubeconfigPath := cfg.KubeconfigPath
		if kubeconfigPath == "" {
			// Default to ~/.kube/config
			home, _ := os.UserHomeDir()
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Build exclude namespaces map for fast lookup
	excludeMap := make(map[string]bool)
	for _, ns := range cfg.ExcludeNamespaces {
		excludeMap[ns] = true
	}

	return &Client{
		clientset:         clientset,
		excludeNamespaces: excludeMap,
	}, nil
}

// ListPods returns all pods across all non-excluded namespaces
func (c *Client) ListPods(ctx context.Context) ([]corev1.Pod, error) {
	// Get all namespaces
	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var allPods []corev1.Pod

	for _, ns := range namespaces.Items {
		// Skip excluded namespaces
		if c.excludeNamespaces[ns.Name] {
			log.Printf("Skipping excluded namespace: %s", ns.Name)
			continue
		}

		pods, err := c.clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Warning: failed to list pods in namespace %s: %v", ns.Name, err)
			continue
		}

		// Filter to only running pods
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				allPods = append(allPods, pod)
			}
		}
	}

	log.Printf("Found %d running pods across %d namespaces", len(allPods), len(namespaces.Items))
	return allPods, nil
}

// GetPod retrieves a specific pod
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
}

// GetPodResourceRequests extracts resource requests and limits from a pod
func GetPodResourceRequests(pod *corev1.Pod) map[string]ContainerResources {
	resources := make(map[string]ContainerResources)

	for _, container := range pod.Spec.Containers {
		cr := ContainerResources{
			Name: container.Name,
		}

		// CPU requests and limits
		if cpuReq := container.Resources.Requests.Cpu(); cpuReq != nil {
			cr.CPURequest = float64(cpuReq.MilliValue()) / 1000.0
		}
		if cpuLimit := container.Resources.Limits.Cpu(); cpuLimit != nil {
			cr.CPULimit = float64(cpuLimit.MilliValue()) / 1000.0
		}

		// Memory requests and limits (in MB)
		if memReq := container.Resources.Requests.Memory(); memReq != nil {
			cr.MemoryRequest = memReq.Value() / (1024 * 1024)
		}
		if memLimit := container.Resources.Limits.Memory(); memLimit != nil {
			cr.MemoryLimit = memLimit.Value() / (1024 * 1024)
		}

		resources[container.Name] = cr
	}

	return resources
}

// IsNamespaceExcluded checks if a namespace should be excluded
func (c *Client) IsNamespaceExcluded(namespace string) bool {
	return c.excludeNamespaces[namespace]
}
