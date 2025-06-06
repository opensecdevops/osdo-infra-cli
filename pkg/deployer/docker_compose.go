package deployer

import (
	"context"
	"fmt"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// DockerComposeDeployer handles deployments using Docker Compose
type DockerComposeDeployer struct {
	client     *client.Client
	composeDir string
}

// NewDockerComposeDeployer creates a new Docker Compose deployer
func NewDockerComposeDeployer() *DockerComposeDeployer {
	return &DockerComposeDeployer{
		composeDir: "./docker-compose",
	}
}

// Initialize initializes the Docker client
func (d *DockerComposeDeployer) Initialize() error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	
	d.client = cli
	return nil
}

// Deploy deploys components using Docker Compose
func (d *DockerComposeDeployer) Deploy(ctx context.Context, components []string, opts DeploymentOptions) (*DeploymentResult, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	startTime := time.Now()
	result := &DeploymentResult{
		Platform:   config.PlatformDockerCompose,
		Components: make([]ComponentResult, 0, len(components)),
		AccessInfo: make(map[string]AccessInfo),
	}
	
	var allSuccess = true
	
	for _, component := range components {
		componentResult := d.deployComponent(ctx, component, opts)
		result.Components = append(result.Components, componentResult)
		
		if !componentResult.Success {
			allSuccess = false
		}
	}
	
	result.Success = allSuccess
	result.Duration = time.Since(startTime)
	
	return result, nil
}

// deployComponent deploys a single component via Docker Compose
func (d *DockerComposeDeployer) deployComponent(ctx context.Context, component string, opts DeploymentOptions) ComponentResult {
	startTime := time.Now()
	
	result := ComponentResult{
		Name:      component,
		Resources: make([]ResourceInfo, 0),
		Endpoints: make([]string, 0),
		Metadata:  make(map[string]string),
	}
	
	// Component-specific deployment logic
	switch component {
	case "prometheus":
		err := d.deployPrometheus(ctx, opts)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Status = "Failed"
		} else {
			result.Success = true
			result.Status = "Running"
			result.Endpoints = append(result.Endpoints, "http://localhost:9090")
		}
	case "grafana":
		err := d.deployGrafana(ctx, opts)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Status = "Failed"
		} else {
			result.Success = true
			result.Status = "Running"
			result.Endpoints = append(result.Endpoints, "http://localhost:3000")
		}
	case "jaeger":
		err := d.deployJaeger(ctx, opts)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Status = "Failed"
		} else {
			result.Success = true
			result.Status = "Running"
			result.Endpoints = append(result.Endpoints, "http://localhost:16686")
		}
	default:
		result.Success = false
		result.Error = fmt.Sprintf("component %s not supported", component)
		result.Status = "Not Supported"
	}
	
	result.Duration = time.Since(startTime)
	return result
}

// Undeploy removes components
func (d *DockerComposeDeployer) Undeploy(ctx context.Context, components []string, opts DeploymentOptions) error {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return err
		}
	}
	
	for _, component := range components {
		if err := d.undeployComponent(ctx, component, opts); err != nil {
			return fmt.Errorf("failed to undeploy %s: %w", component, err)
		}
	}
	
	return nil
}

// GetStatus returns the status of deployed components
func (d *DockerComposeDeployer) GetStatus(ctx context.Context, components []string) (map[string]ComponentStatus, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	status := make(map[string]ComponentStatus)
	
	for _, component := range components {
		componentStatus, err := d.getComponentStatus(ctx, component)
		if err != nil {
			continue
		}
		status[component] = componentStatus
	}
	
	return status, nil
}

// ValidateComponents validates that components can be deployed
func (d *DockerComposeDeployer) ValidateComponents(components []string) error {
	supportedComponents := []string{"prometheus", "grafana", "jaeger", "vault", "sonarqube", "jenkins"}
	
	for _, component := range components {
		supported := false
		for _, supported_comp := range supportedComponents {
			if component == supported_comp {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("component %s is not supported on Docker Compose", component)
		}
	}
	
	return nil
}

// GetPlatform returns the platform type
func (d *DockerComposeDeployer) GetPlatform() config.Platform {
	return config.PlatformDockerCompose
}

// IsReady checks if Docker is ready
func (d *DockerComposeDeployer) IsReady(ctx context.Context) error {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return err
		}
	}
	
	// Check if Docker daemon is running
	_, err := d.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker daemon not ready: %w", err)
	}
	
	return nil
}

// Helper methods

func (d *DockerComposeDeployer) deployPrometheus(ctx context.Context, opts DeploymentOptions) error {
	if opts.DryRun {
		return nil
	}
	
	// Create Prometheus container
	containerConfig := &container.Config{
		Image: "prom/prometheus:latest",
		ExposedPorts: nat.PortSet{
			"9090/tcp": struct{}{},
		},
	}
	
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"9090/tcp": []nat.PortBinding{{HostPort: "9090"}},
		},
	}
	
	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "osdo-prometheus")
	if err != nil {
		return fmt.Errorf("failed to create prometheus container: %w", err)
	}
	
	if err := d.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start prometheus container: %w", err)
	}
	
	return nil
}

func (d *DockerComposeDeployer) deployGrafana(ctx context.Context, opts DeploymentOptions) error {
	if opts.DryRun {
		return nil
	}
	
	containerConfig := &container.Config{
		Image: "grafana/grafana:latest",
		ExposedPorts: nat.PortSet{
			"3000/tcp": struct{}{},
		},
		Env: []string{
			"GF_SECURITY_ADMIN_PASSWORD=admin",
		},
	}
	
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"3000/tcp": []nat.PortBinding{{HostPort: "3000"}},
		},
	}
	
	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "osdo-grafana")
	if err != nil {
		return fmt.Errorf("failed to create grafana container: %w", err)
	}
	
	if err := d.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start grafana container: %w", err)
	}
	
	return nil
}

func (d *DockerComposeDeployer) deployJaeger(ctx context.Context, opts DeploymentOptions) error {
	if opts.DryRun {
		return nil
	}
	
	containerConfig := &container.Config{
		Image: "jaegertracing/all-in-one:latest",
		ExposedPorts: nat.PortSet{
			"16686/tcp": struct{}{},
			"14268/tcp": struct{}{},
		},
	}
	
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"16686/tcp": []nat.PortBinding{{HostPort: "16686"}},
			"14268/tcp": []nat.PortBinding{{HostPort: "14268"}},
		},
	}
	
	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "osdo-jaeger")
	if err != nil {
		return fmt.Errorf("failed to create jaeger container: %w", err)
	}
	
	if err := d.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start jaeger container: %w", err)
	}
	
	return nil
}

func (d *DockerComposeDeployer) undeployComponent(ctx context.Context, component string, opts DeploymentOptions) error {
	containerName := fmt.Sprintf("osdo-%s", component)
	
	// Stop and remove container
	if err := d.client.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerName, err)
	}
	
	if err := d.client.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerName, err)
	}
	
	return nil
}

func (d *DockerComposeDeployer) getComponentStatus(ctx context.Context, component string) (ComponentStatus, error) {
	containerName := fmt.Sprintf("osdo-%s", component)
	
	inspect, err := d.client.ContainerInspect(ctx, containerName)
	if err != nil {
		return ComponentStatus{
			Name:   component,
			Status: "Not Found",
			Ready:  false,
		}, nil
	}
	
	status := "Unknown"
	ready := false
	
	if inspect.State.Running {
		status = "Running"
		ready = true
	} else if inspect.State.Restarting {
		status = "Restarting"
	} else if inspect.State.Paused {
		status = "Paused"
	} else {
		status = "Stopped"
	}
	
	return ComponentStatus{
		Name:   component,
		Status: status,
		Ready:  ready,
		Health: HealthStatus{
			Status:    status,
			LastCheck: time.Now(),
		},
	}, nil
}
