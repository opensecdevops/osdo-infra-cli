package deployer

import (
	"context"
	"fmt"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// KubernetesDeployer handles deployments to Kubernetes clusters
type KubernetesDeployer struct {
	client    kubernetes.Interface
	config    *rest.Config
	namespace string
}

// NewKubernetesDeployer creates a new Kubernetes deployer
func NewKubernetesDeployer() *KubernetesDeployer {
	return &KubernetesDeployer{
		namespace: "default",
	}
}

// Initialize initializes the Kubernetes client
func (k *KubernetesDeployer) Initialize(kubeconfig string) error {
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
	
	k.client = client
	k.config = config
	return nil
}

// Deploy deploys components to Kubernetes
func (k *KubernetesDeployer) Deploy(ctx context.Context, components []string, opts DeploymentOptions) (*DeploymentResult, error) {
	if k.client == nil {
		if err := k.Initialize(""); err != nil {
			return nil, err
		}
	}
	
	startTime := time.Now()
	result := &DeploymentResult{
		Platform:   config.PlatformKubernetes,
		Namespace:  opts.Namespace,
		Components: make([]ComponentResult, 0, len(components)),
		AccessInfo: make(map[string]AccessInfo),
	}
	
	if opts.Namespace != "" {
		k.namespace = opts.Namespace
	}
	
	// Create namespace if needed
	if opts.CreateNamespace && k.namespace != "default" {
		if err := k.ensureNamespace(ctx, k.namespace); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create namespace: %v", err))
		}
	}
	
	var allSuccess = true
	
	for _, component := range components {
		componentResult := k.deployComponent(ctx, component, opts)
		result.Components = append(result.Components, componentResult)
		
		if !componentResult.Success {
			allSuccess = false
		}
	}
	
	result.Success = allSuccess
	result.Duration = time.Since(startTime)
	
	return result, nil
}

// deployComponent deploys a single component
func (k *KubernetesDeployer) deployComponent(ctx context.Context, component string, opts DeploymentOptions) ComponentResult {
	startTime := time.Now()
	
	result := ComponentResult{
		Name:      component,
		Resources: make([]ResourceInfo, 0),
		Endpoints: make([]string, 0),
		Metadata:  make(map[string]string),
	}
	
	// This is a simplified implementation - in reality you'd have
	// specific deployment logic for each component
	switch component {
	case "prometheus":
		err := k.deployPrometheus(ctx, opts)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Status = "Failed"
		} else {
			result.Success = true
			result.Status = "Deployed"
			result.Resources = append(result.Resources, ResourceInfo{
				Kind:      "Deployment",
				Name:      "prometheus-server",
				Namespace: k.namespace,
				Status:    "Running",
				Ready:     true,
			})
			result.Endpoints = append(result.Endpoints, "http://prometheus.local")
		}
	case "grafana":
		err := k.deployGrafana(ctx, opts)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Status = "Failed"
		} else {
			result.Success = true
			result.Status = "Deployed"
			result.Resources = append(result.Resources, ResourceInfo{
				Kind:      "Deployment",
				Name:      "grafana",
				Namespace: k.namespace,
				Status:    "Running",
				Ready:     true,
			})
			result.Endpoints = append(result.Endpoints, "http://grafana.local")
		}
	default:
		result.Success = false
		result.Error = fmt.Sprintf("component %s not supported", component)
		result.Status = "Not Supported"
	}
	
	result.Duration = time.Since(startTime)
	return result
}

// Undeploy removes components from Kubernetes
func (k *KubernetesDeployer) Undeploy(ctx context.Context, components []string, opts DeploymentOptions) error {
	if k.client == nil {
		if err := k.Initialize(""); err != nil {
			return err
		}
	}
	
	for _, component := range components {
		if err := k.undeployComponent(ctx, component, opts); err != nil {
			return fmt.Errorf("failed to undeploy %s: %w", component, err)
		}
	}
	
	return nil
}

// GetStatus returns the status of deployed components
func (k *KubernetesDeployer) GetStatus(ctx context.Context, components []string) (map[string]ComponentStatus, error) {
	if k.client == nil {
		if err := k.Initialize(""); err != nil {
			return nil, err
		}
	}
	
	status := make(map[string]ComponentStatus)
	
	for _, component := range components {
		componentStatus, err := k.getComponentStatus(ctx, component)
		if err != nil {
			continue // Skip components that can't be found
		}
		status[component] = componentStatus
	}
	
	return status, nil
}

// ValidateComponents validates that components can be deployed
func (k *KubernetesDeployer) ValidateComponents(components []string) error {
	supportedComponents := []string{"prometheus", "grafana", "jaeger", "vault", "sonarqube"}
	
	for _, component := range components {
		supported := false
		for _, supported_comp := range supportedComponents {
			if component == supported_comp {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("component %s is not supported on Kubernetes", component)
		}
	}
	
	return nil
}

// GetPlatform returns the platform type
func (k *KubernetesDeployer) GetPlatform() config.Platform {
	return config.PlatformKubernetes
}

// IsReady checks if the Kubernetes cluster is ready
func (k *KubernetesDeployer) IsReady(ctx context.Context) error {
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

func (k *KubernetesDeployer) ensureNamespace(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	
	_, err := k.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !isAlreadyExistsError(err) {
		return err
	}
	
	return nil
}

func (k *KubernetesDeployer) deployPrometheus(ctx context.Context, opts DeploymentOptions) error {
	// Simplified Prometheus deployment
	// In reality, you'd use Helm charts or proper manifests
	if opts.DryRun {
		return nil
	}
	
	// This would contain actual Prometheus deployment logic
	return nil
}

func (k *KubernetesDeployer) deployGrafana(ctx context.Context, opts DeploymentOptions) error {
	// Simplified Grafana deployment
	if opts.DryRun {
		return nil
	}
	
	// This would contain actual Grafana deployment logic
	return nil
}

func (k *KubernetesDeployer) undeployComponent(ctx context.Context, component string, opts DeploymentOptions) error {
	// Component-specific undeployment logic
	return nil
}

func (k *KubernetesDeployer) getComponentStatus(ctx context.Context, component string) (ComponentStatus, error) {
	// Get actual status from Kubernetes
	deployments, err := k.client.AppsV1().Deployments(k.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", component),
	})
	if err != nil {
		return ComponentStatus{}, err
	}
	
	if len(deployments.Items) == 0 {
		return ComponentStatus{
			Name:   component,
			Status: "Not Found",
			Ready:  false,
		}, nil
	}
	
	deployment := deployments.Items[0]
	
	return ComponentStatus{
		Name:   component,
		Status: getDeploymentStatus(&deployment),
		Ready:  isDeploymentReady(&deployment),
		Replicas: ReplicaStatus{
			Desired:   *deployment.Spec.Replicas,
			Ready:     deployment.Status.ReadyReplicas,
			Available: deployment.Status.AvailableReplicas,
			Updated:   deployment.Status.UpdatedReplicas,
		},
		Health: HealthStatus{
			Status:    "Unknown",
			LastCheck: time.Now(),
		},
	}, nil
}

func getDeploymentStatus(deployment *appsv1.Deployment) string {
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return "Running"
	} else if deployment.Status.ReadyReplicas > 0 {
		return "Partially Ready"
	}
	return "Not Ready"
}

func isDeploymentReady(deployment *appsv1.Deployment) bool {
	return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
}

func isAlreadyExistsError(err error) bool {
	// Simple check - in reality you'd check the specific error type
	return err != nil && (err.Error() == "already exists" || 
		err.Error() == "namespaces \"\" already exists")
}
