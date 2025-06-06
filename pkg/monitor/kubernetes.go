package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/deployer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

// KubernetesMonitor monitors Kubernetes clusters
type KubernetesMonitor struct {
	client        kubernetes.Interface
	metricsClient versioned.Interface
	config        *rest.Config
	namespace     string
}

// NewKubernetesMonitor creates a new Kubernetes monitor
func NewKubernetesMonitor() *KubernetesMonitor {
	return &KubernetesMonitor{
		namespace: "default",
	}
}

// Initialize initializes the Kubernetes client
func (k *KubernetesMonitor) Initialize(kubeconfig string) error {
	var config *rest.Config
	var err error
	
	if kubeconfig == "" {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to kubeconfig file
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}
	}
	
	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	}
	
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	
	metricsClient, err := versioned.NewForConfig(config)
	if err != nil {
		// Metrics client is optional
		metricsClient = nil
	}
	
	k.client = client
	k.metricsClient = metricsClient
	k.config = config
	return nil
}

// GetSystemStatus returns the current system status
func (k *KubernetesMonitor) GetSystemStatus(ctx context.Context, opts MonitoringOptions) (*SystemStatus, error) {
	if k.client == nil {
		if err := k.Initialize(""); err != nil {
			return nil, err
		}
	}
	
	if opts.Namespace != "" {
		k.namespace = opts.Namespace
	}
	
	status := &SystemStatus{
		Timestamp: time.Now(),
		Platform:  config.PlatformKubernetes,
		Namespace: k.namespace,
		Components: make(map[string]deployer.ComponentStatus),
		Alerts:    make([]Alert, 0),
		Metadata:  make(map[string]interface{}),
	}
	
	// Get component status
	if len(opts.Components) > 0 {
		componentStatus, err := k.GetComponentStatus(ctx, opts.Components, opts)
		if err == nil {
			status.Components = componentStatus
		}
	}
	
	// Get resource usage
	resourceUsage, err := k.GetResourceUsage(ctx, opts)
	if err == nil {
		status.Resources = *resourceUsage
	}
	
	// Get alerts
	alerts, err := k.GetAlerts(ctx, opts)
	if err == nil {
		status.Alerts = alerts
	}
	
	// Determine overall health
	status.OverallHealth = k.calculateOverallHealth(status)
	
	return status, nil
}

// GetComponentStatus returns the status of specific components
func (k *KubernetesMonitor) GetComponentStatus(ctx context.Context, components []string, opts MonitoringOptions) (map[string]deployer.ComponentStatus, error) {
	if k.client == nil {
		if err := k.Initialize(""); err != nil {
			return nil, err
		}
	}
	
	status := make(map[string]deployer.ComponentStatus)
	
	for _, component := range components {
		componentStatus, err := k.getComponentStatus(ctx, component)
		if err != nil {
			continue
		}
		status[component] = componentStatus
	}
	
	return status, nil
}

// GetResourceUsage returns resource usage information
func (k *KubernetesMonitor) GetResourceUsage(ctx context.Context, opts MonitoringOptions) (*ResourceUsage, error) {
	if k.client == nil {
		if err := k.Initialize(""); err != nil {
			return nil, err
		}
	}
	
	// Get node information
	nodes, err := k.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}
	
	usage := &ResourceUsage{
		CPU: ResourceMetric{
			Used:       "0",
			Total:      "0",
			Percentage: 0,
			Unit:       "cores",
		},
		Memory: ResourceMetric{
			Used:       "0",
			Total:      "0",
			Percentage: 0,
			Unit:       "bytes",
		},
		Storage: ResourceMetric{
			Used:       "0",
			Total:      "0",
			Percentage: 0,
			Unit:       "bytes",
		},
		Network: NetworkMetric{
			BytesIn:    "0",
			BytesOut:   "0",
			PacketsIn:  0,
			PacketsOut: 0,
		},
	}
	
	// Calculate totals from node capacity
	var totalCPU, totalMemory int64
	for _, node := range nodes.Items {
		if cpu, ok := node.Status.Capacity["cpu"]; ok {
			totalCPU += cpu.MilliValue()
		}
		if memory, ok := node.Status.Capacity["memory"]; ok {
			totalMemory += memory.Value()
		}
	}
	
	usage.CPU.Total = fmt.Sprintf("%.2f", float64(totalCPU)/1000)
	usage.Memory.Total = fmt.Sprintf("%d", totalMemory)
	
	// Get metrics if available
	if k.metricsClient != nil {
		nodeMetrics, err := k.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
		if err == nil {
			var usedCPU, usedMemory int64
			for _, metric := range nodeMetrics.Items {
				if cpu, ok := metric.Usage["cpu"]; ok {
					usedCPU += cpu.MilliValue()
				}
				if memory, ok := metric.Usage["memory"]; ok {
					usedMemory += memory.Value()
				}
			}
			
			usage.CPU.Used = fmt.Sprintf("%.2f", float64(usedCPU)/1000)
			usage.Memory.Used = fmt.Sprintf("%d", usedMemory)
			
			if totalCPU > 0 {
				usage.CPU.Percentage = float64(usedCPU) / float64(totalCPU) * 100
			}
			if totalMemory > 0 {
				usage.Memory.Percentage = float64(usedMemory) / float64(totalMemory) * 100
			}
		}
	}
	
	return usage, nil
}

// GetAlerts returns active alerts
func (k *KubernetesMonitor) GetAlerts(ctx context.Context, opts MonitoringOptions) ([]Alert, error) {
	alerts := make([]Alert, 0)
	
	// Check for unhealthy nodes
	nodes, err := k.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return alerts, nil
	}
	
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status != "True" {
				alerts = append(alerts, Alert{
					ID:        fmt.Sprintf("node-%s-not-ready", node.Name),
					Severity:  AlertSeverityCritical,
					Component: "node",
					Message:   fmt.Sprintf("Node %s is not ready: %s", node.Name, condition.Message),
					Timestamp: time.Now(),
					Status:    AlertStatusFiring,
					Labels: map[string]string{
						"node": node.Name,
						"type": "node_not_ready",
					},
				})
			}
		}
	}
	
	// Check for failed pods
	pods, err := k.client.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return alerts, nil
	}
	
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Failed" {
			alerts = append(alerts, Alert{
				ID:        fmt.Sprintf("pod-%s-failed", pod.Name),
				Severity:  AlertSeverityWarning,
				Component: "pod",
				Message:   fmt.Sprintf("Pod %s failed: %s", pod.Name, pod.Status.Message),
				Timestamp: time.Now(),
				Status:    AlertStatusFiring,
				Labels: map[string]string{
					"pod":       pod.Name,
					"namespace": pod.Namespace,
					"type":      "pod_failed",
				},
			})
		}
	}
	
	return alerts, nil
}

// RunHealthChecks performs health checks on components
func (k *KubernetesMonitor) RunHealthChecks(ctx context.Context, components []string, opts MonitoringOptions) (map[string]HealthCheck, error) {
	if k.client == nil {
		if err := k.Initialize(""); err != nil {
			return nil, err
		}
	}
	
	checks := make(map[string]HealthCheck)
	
	for _, component := range components {
		check := k.runComponentHealthCheck(ctx, component)
		checks[component] = check
	}
	
	return checks, nil
}

// StartWatching starts monitoring in watch mode
func (k *KubernetesMonitor) StartWatching(ctx context.Context, opts MonitoringOptions) (<-chan *SystemStatus, error) {
	statusChan := make(chan *SystemStatus)
	
	go func() {
		defer close(statusChan)
		
		ticker := time.NewTicker(opts.Interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				status, err := k.GetSystemStatus(ctx, opts)
				if err != nil {
					continue
				}
				
				select {
				case statusChan <- status:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return statusChan, nil
}

// GetPlatform returns the platform type
func (k *KubernetesMonitor) GetPlatform() config.Platform {
	return config.PlatformKubernetes
}

// IsReady checks if monitoring is ready
func (k *KubernetesMonitor) IsReady(ctx context.Context) error {
	if k.client == nil {
		if err := k.Initialize(""); err != nil {
			return err
		}
	}
	
	// Check if we can list nodes
	_, err := k.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("kubernetes cluster not ready: %w", err)
	}
	
	return nil
}

// Helper methods

func (k *KubernetesMonitor) getComponentStatus(ctx context.Context, component string) (deployer.ComponentStatus, error) {
	// Look for deployments with the component label
	deployments, err := k.client.AppsV1().Deployments(k.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", component),
	})
	if err != nil {
		return deployer.ComponentStatus{}, err
	}
	
	if len(deployments.Items) == 0 {
		return deployer.ComponentStatus{
			Name:   component,
			Status: "Not Found",
			Ready:  false,
		}, nil
	}
	
	deployment := deployments.Items[0]
	
	return deployer.ComponentStatus{
		Name:   component,
		Status: getDeploymentStatus(&deployment),
		Ready:  isDeploymentReady(&deployment),
		Replicas: deployer.ReplicaStatus{
			Desired:   *deployment.Spec.Replicas,
			Ready:     deployment.Status.ReadyReplicas,
			Available: deployment.Status.AvailableReplicas,
			Updated:   deployment.Status.UpdatedReplicas,
		},
		Health: deployer.HealthStatus{
			Status:    "Unknown",
			LastCheck: time.Now(),
		},
	}, nil
}

func (k *KubernetesMonitor) runComponentHealthCheck(ctx context.Context, component string) HealthCheck {
	startTime := time.Now()
	
	check := HealthCheck{
		Name:      component,
		Timestamp: startTime,
		Metadata:  make(map[string]string),
	}
	
	// Get component status to determine health
	status, err := k.getComponentStatus(ctx, component)
	if err != nil {
		check.Success = false
		check.Status = "Error"
		check.Message = err.Error()
	} else {
		check.Success = status.Ready
		check.Status = status.Status
		if status.Ready {
			check.Message = "Component is healthy"
		} else {
			check.Message = "Component is not ready"
		}
	}
	
	check.Duration = time.Since(startTime)
	return check
}

func (k *KubernetesMonitor) calculateOverallHealth(status *SystemStatus) string {
	if len(status.Components) == 0 {
		return "Unknown"
	}
	
	readyCount := 0
	for _, component := range status.Components {
		if component.Ready {
			readyCount++
		}
	}
	
	if readyCount == len(status.Components) {
		return "Healthy"
	} else if readyCount > 0 {
		return "Degraded"
	}
	
	return "Unhealthy"
}

// Helper functions (these would be moved to a shared utils package)

func getDeploymentStatus(deployment interface{}) string {
	// Simplified implementation
	return "Running"
}

func isDeploymentReady(deployment interface{}) bool {
	// Simplified implementation
	return true
}
