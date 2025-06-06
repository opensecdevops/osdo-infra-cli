package deployer

import (
	"context"
	"fmt"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

// HelmDeployer handles deployments using Helm charts
type HelmDeployer struct {
	settings   *cli.EnvSettings
	namespace  string
	kubeconfig string
}

// NewHelmDeployer creates a new Helm deployer
func NewHelmDeployer() *HelmDeployer {
	return &HelmDeployer{
		settings:  cli.New(),
		namespace: "default",
	}
}

// Initialize initializes the Helm client
func (h *HelmDeployer) Initialize(kubeconfig string) error {
	h.kubeconfig = kubeconfig
	if kubeconfig != "" {
		h.settings.KubeConfig = kubeconfig
	}
	return nil
}

// Deploy deploys components using Helm charts
func (h *HelmDeployer) Deploy(ctx context.Context, components []string, opts DeploymentOptions) (*DeploymentResult, error) {
	startTime := time.Now()
	result := &DeploymentResult{
		Platform:   config.PlatformHelm,
		Namespace:  opts.Namespace,
		Components: make([]ComponentResult, 0, len(components)),
		AccessInfo: make(map[string]AccessInfo),
	}
	
	if opts.Namespace != "" {
		h.namespace = opts.Namespace
	}
	
	var allSuccess = true
	
	for _, component := range components {
		componentResult := h.deployComponent(ctx, component, opts)
		result.Components = append(result.Components, componentResult)
		
		if !componentResult.Success {
			allSuccess = false
		}
	}
	
	result.Success = allSuccess
	result.Duration = time.Since(startTime)
	
	return result, nil
}

// deployComponent deploys a single component using Helm
func (h *HelmDeployer) deployComponent(ctx context.Context, component string, opts DeploymentOptions) ComponentResult {
	startTime := time.Now()
	
	result := ComponentResult{
		Name:      component,
		Resources: make([]ResourceInfo, 0),
		Endpoints: make([]string, 0),
		Metadata:  make(map[string]string),
	}
	
	// Get chart information for the component
	chartInfo, err := h.getChartInfo(component)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Status = "Chart Not Found"
		result.Duration = time.Since(startTime)
		return result
	}
	
	// Deploy using Helm
	err = h.deployChart(ctx, component, chartInfo, opts)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Status = "Failed"
	} else {
		result.Success = true
		result.Status = "Deployed"
		result.Endpoints = h.getComponentEndpoints(component)
	}
	
	result.Duration = time.Since(startTime)
	return result
}

// ChartInfo contains information about a Helm chart
type ChartInfo struct {
	Repository string
	Chart      string
	Version    string
	Values     map[string]interface{}
}

// getChartInfo returns chart information for a component
func (h *HelmDeployer) getChartInfo(component string) (*ChartInfo, error) {
	charts := map[string]*ChartInfo{
		"prometheus": {
			Repository: "https://prometheus-community.github.io/helm-charts",
			Chart:      "prometheus",
			Version:    "latest",
			Values: map[string]interface{}{
				"server": map[string]interface{}{
					"service": map[string]interface{}{
						"type": "NodePort",
					},
				},
			},
		},
		"grafana": {
			Repository: "https://grafana.github.io/helm-charts",
			Chart:      "grafana",
			Version:    "latest",
			Values: map[string]interface{}{
				"service": map[string]interface{}{
					"type": "NodePort",
				},
				"adminPassword": "admin",
			},
		},
		"jaeger": {
			Repository: "https://jaegertracing.github.io/helm-charts",
			Chart:      "jaeger",
			Version:    "latest",
			Values: map[string]interface{}{
				"query": map[string]interface{}{
					"service": map[string]interface{}{
						"type": "NodePort",
					},
				},
			},
		},
		"vault": {
			Repository: "https://helm.releases.hashicorp.com",
			Chart:      "vault",
			Version:    "latest",
			Values: map[string]interface{}{
				"server": map[string]interface{}{
					"dev": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
		"sonarqube": {
			Repository: "https://SonarSource.github.io/helm-chart-sonarqube",
			Chart:      "sonarqube",
			Version:    "latest",
			Values: map[string]interface{}{
				"service": map[string]interface{}{
					"type": "NodePort",
				},
			},
		},
	}
	
	chartInfo, exists := charts[component]
	if !exists {
		return nil, fmt.Errorf("no chart available for component: %s", component)
	}
	
	return chartInfo, nil
}

// deployChart deploys a Helm chart
func (h *HelmDeployer) deployChart(ctx context.Context, component string, chartInfo *ChartInfo, opts DeploymentOptions) error {
	if opts.DryRun {
		return nil
	}
	
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), h.namespace, "secret", func(format string, v ...interface{}) {}); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}
	
	// Check if release already exists
	listAction := action.NewList(actionConfig)
	releases, err := listAction.Run()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}
	
	var existingRelease *release.Release
	for _, rel := range releases {
		if rel.Name == component {
			existingRelease = rel
			break
		}
	}
	
	if existingRelease != nil {
		if opts.Force {
			// Upgrade existing release
			upgradeAction := action.NewUpgrade(actionConfig)
			upgradeAction.Namespace = h.namespace
			upgradeAction.Wait = opts.Wait
			upgradeAction.Timeout = opts.Timeout
			
			chart, err := loader.LoadDir(chartInfo.Chart)
			if err != nil {
				return fmt.Errorf("failed to load chart: %w", err)
			}
			
			_, err = upgradeAction.Run(component, chart, chartInfo.Values)
			if err != nil {
				return fmt.Errorf("failed to upgrade release: %w", err)
			}
		} else {
			return fmt.Errorf("release %s already exists (use --force to upgrade)", component)
		}
	} else {
		// Install new release
		installAction := action.NewInstall(actionConfig)
		installAction.ReleaseName = component
		installAction.Namespace = h.namespace
		installAction.CreateNamespace = opts.CreateNamespace
		installAction.Wait = opts.Wait
		installAction.Timeout = opts.Timeout
		
		chart, err := loader.LoadDir(chartInfo.Chart)
		if err != nil {
			return fmt.Errorf("failed to load chart: %w", err)
		}
		
		_, err = installAction.Run(chart, chartInfo.Values)
		if err != nil {
			return fmt.Errorf("failed to install release: %w", err)
		}
	}
	
	return nil
}

// Undeploy removes components using Helm
func (h *HelmDeployer) Undeploy(ctx context.Context, components []string, opts DeploymentOptions) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), h.namespace, "secret", func(format string, v ...interface{}) {}); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}
	
	for _, component := range components {
		uninstallAction := action.NewUninstall(actionConfig)
		uninstallAction.Timeout = opts.Timeout
		
		_, err := uninstallAction.Run(component)
		if err != nil {
			return fmt.Errorf("failed to uninstall %s: %w", component, err)
		}
	}
	
	return nil
}

// GetStatus returns the status of deployed components
func (h *HelmDeployer) GetStatus(ctx context.Context, components []string) (map[string]ComponentStatus, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), h.namespace, "secret", func(format string, v ...interface{}) {}); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}
	
	status := make(map[string]ComponentStatus)
	
	for _, component := range components {
		componentStatus, err := h.getComponentStatus(ctx, component, actionConfig)
		if err != nil {
			continue
		}
		status[component] = componentStatus
	}
	
	return status, nil
}

// ValidateComponents validates that components can be deployed
func (h *HelmDeployer) ValidateComponents(components []string) error {
	for _, component := range components {
		_, err := h.getChartInfo(component)
		if err != nil {
			return fmt.Errorf("component %s validation failed: %w", component, err)
		}
	}
	
	return nil
}

// GetPlatform returns the platform type
func (h *HelmDeployer) GetPlatform() config.Platform {
	return config.PlatformHelm
}

// IsReady checks if Helm can connect to Kubernetes
func (h *HelmDeployer) IsReady(ctx context.Context) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), h.namespace, "secret", func(format string, v ...interface{}) {}); err != nil {
		return fmt.Errorf("helm not ready: %w", err)
	}
	
	// Try to list releases to test connectivity
	listAction := action.NewList(actionConfig)
	_, err := listAction.Run()
	if err != nil {
		return fmt.Errorf("kubernetes cluster not accessible via helm: %w", err)
	}
	
	return nil
}

// Helper methods

func (h *HelmDeployer) getComponentStatus(ctx context.Context, component string, actionConfig *action.Configuration) (ComponentStatus, error) {
	statusAction := action.NewStatus(actionConfig)
	
	release, err := statusAction.Run(component)
	if err != nil {
		return ComponentStatus{
			Name:   component,
			Status: "Not Found",
			Ready:  false,
		}, nil
	}
	
	status := string(release.Info.Status)
	ready := release.Info.Status == "deployed"
	
	return ComponentStatus{
		Name:   component,
		Status: status,
		Ready:  ready,
		Health: HealthStatus{
			Status:    status,
			LastCheck: time.Now(),
		},
		Metadata: map[string]interface{}{
			"chart":     release.Chart.Metadata.Name,
			"version":   release.Chart.Metadata.Version,
			"revision":  release.Version,
			"namespace": release.Namespace,
		},
	}, nil
}

func (h *HelmDeployer) getComponentEndpoints(component string) []string {
	endpoints := map[string][]string{
		"prometheus": {"http://prometheus.local:9090"},
		"grafana":    {"http://grafana.local:3000"},
		"jaeger":     {"http://jaeger.local:16686"},
		"vault":      {"http://vault.local:8200"},
		"sonarqube":  {"http://sonarqube.local:9000"},
	}
	
	if eps, exists := endpoints[component]; exists {
		return eps
	}
	
	return []string{}
}
