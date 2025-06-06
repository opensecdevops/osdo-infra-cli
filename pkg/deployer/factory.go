package deployer

import (
	"context"
	"fmt"

	"github.com/osdo/osdo-infra-cli/pkg/config"
)

// Factory creates deployers for different platforms
type Factory struct {
	deployers map[config.Platform]Deployer
}

// NewFactory creates a new deployer factory
func NewFactory() *Factory {
	return &Factory{
		deployers: make(map[config.Platform]Deployer),
	}
}

// Register registers a deployer for a platform
func (f *Factory) Register(platform config.Platform, deployer Deployer) {
	f.deployers[platform] = deployer
}

// GetDeployer returns a deployer for the specified platform
func (f *Factory) GetDeployer(platform config.Platform) (Deployer, error) {
	deployer, exists := f.deployers[platform]
	if !exists {
		return nil, fmt.Errorf("no deployer registered for platform: %s", platform)
	}
	return deployer, nil
}

// GetSupportedPlatforms returns a list of supported platforms
func (f *Factory) GetSupportedPlatforms() []config.Platform {
	platforms := make([]config.Platform, 0, len(f.deployers))
	for platform := range f.deployers {
		platforms = append(platforms, platform)
	}
	return platforms
}

// InitializeDefaultDeployers initializes the factory with default deployers
func (f *Factory) InitializeDefaultDeployers() {
	// Register Kubernetes deployer
	f.Register(config.PlatformKubernetes, NewKubernetesDeployer())
	
	// Register K3s deployer (can reuse Kubernetes deployer)
	f.Register(config.PlatformK3s, NewKubernetesDeployer())
	
	// Register Docker Compose deployer
	f.Register(config.PlatformDockerCompose, NewDockerComposeDeployer())
	
	// Register Docker Swarm deployer
	f.Register(config.PlatformDockerSwarm, NewDockerSwarmDeployer())
	
	// Register Helm deployer
	f.Register(config.PlatformHelm, NewHelmDeployer())
}

// DefaultFactory is the default deployer factory instance
var DefaultFactory = NewFactory()

// GetDeployer is a convenience function to get a deployer from the default factory
func GetDeployer(platform config.Platform) (Deployer, error) {
	return DefaultFactory.GetDeployer(platform)
}

// DeployWithPlan creates and executes a deployment plan
func DeployWithPlan(ctx context.Context, cfg *config.Config, components []string, platform config.Platform, opts DeploymentOptions) (*DeploymentResult, error) {
	deployer, err := GetDeployer(platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployer: %w", err)
	}

	// Validate components before deployment
	if err := deployer.ValidateComponents(components); err != nil {
		return nil, fmt.Errorf("component validation failed: %w", err)
	}

	// Check if platform is ready
	if err := deployer.IsReady(ctx); err != nil {
		return nil, fmt.Errorf("platform not ready: %w", err)
	}

	// Execute deployment
	return deployer.Deploy(ctx, components, opts)
}

// GetComponentStatus gets the status of components across all platforms
func GetComponentStatus(ctx context.Context, components []string) (map[config.Platform]map[string]ComponentStatus, error) {
	result := make(map[config.Platform]map[string]ComponentStatus)
	
	for _, platform := range DefaultFactory.GetSupportedPlatforms() {
		deployer, err := GetDeployer(platform)
		if err != nil {
			continue
		}
		
		status, err := deployer.GetStatus(ctx, components)
		if err != nil {
			continue
		}
		
		result[platform] = status
	}
	
	return result, nil
}
