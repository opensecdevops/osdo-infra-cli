package deployer

import (
	"context"
	"fmt"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

// DockerSwarmDeployer handles deployments to Docker Swarm
type DockerSwarmDeployer struct {
	client *client.Client
}

// NewDockerSwarmDeployer creates a new Docker Swarm deployer
func NewDockerSwarmDeployer() *DockerSwarmDeployer {
	return &DockerSwarmDeployer{}
}

// Initialize initializes the Docker client
func (d *DockerSwarmDeployer) Initialize() error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	
	d.client = cli
	return nil
}

// Deploy deploys components to Docker Swarm
func (d *DockerSwarmDeployer) Deploy(ctx context.Context, components []string, opts DeploymentOptions) (*DeploymentResult, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	startTime := time.Now()
	result := &DeploymentResult{
		Platform:   config.PlatformDockerSwarm,
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

// deployComponent deploys a single component to Docker Swarm
func (d *DockerSwarmDeployer) deployComponent(ctx context.Context, component string, opts DeploymentOptions) ComponentResult {
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
			result.Endpoints = append(result.Endpoints, "http://prometheus.swarm:9090")
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
			result.Endpoints = append(result.Endpoints, "http://grafana.swarm:3000")
		}
	case "traefik":
		err := d.deployTraefik(ctx, opts)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Status = "Failed"
		} else {
			result.Success = true
			result.Status = "Running"
			result.Endpoints = append(result.Endpoints, "http://traefik.swarm:8080")
		}
	default:
		result.Success = false
		result.Error = fmt.Sprintf("component %s not supported", component)
		result.Status = "Not Supported"
	}
	
	result.Duration = time.Since(startTime)
	return result
}

// Undeploy removes components from Docker Swarm
func (d *DockerSwarmDeployer) Undeploy(ctx context.Context, components []string, opts DeploymentOptions) error {
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
func (d *DockerSwarmDeployer) GetStatus(ctx context.Context, components []string) (map[string]ComponentStatus, error) {
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
func (d *DockerSwarmDeployer) ValidateComponents(components []string) error {
	supportedComponents := []string{"prometheus", "grafana", "traefik", "portainer", "vault"}
	
	for _, component := range components {
		supported := false
		for _, supported_comp := range supportedComponents {
			if component == supported_comp {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("component %s is not supported on Docker Swarm", component)
		}
	}
	
	return nil
}

// GetPlatform returns the platform type
func (d *DockerSwarmDeployer) GetPlatform() config.Platform {
	return config.PlatformDockerSwarm
}

// IsReady checks if Docker Swarm is ready
func (d *DockerSwarmDeployer) IsReady(ctx context.Context) error {
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
	
	// Check if Swarm mode is active
	info, err := d.client.Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to get docker info: %w", err)
	}
	
	if info.Swarm.LocalNodeState != swarm.LocalNodeStateActive {
		return fmt.Errorf("docker swarm not active")
	}
	
	return nil
}

// Helper methods

func (d *DockerSwarmDeployer) deployPrometheus(ctx context.Context, opts DeploymentOptions) error {
	if opts.DryRun {
		return nil
	}
	
	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "osdo-prometheus",
			Labels: map[string]string{
				"osdo.component": "prometheus",
			},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "prom/prometheus:latest",
			},
		},
		EndpointSpec: &swarm.EndpointSpec{
			Ports: []swarm.PortConfig{
				{
					Protocol:      swarm.PortConfigProtocolTCP,
					TargetPort:    9090,
					PublishedPort: 9090,
				},
			},
		},
	}
	
	_, err := d.client.ServiceCreate(ctx, serviceSpec, types.ServiceCreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create prometheus service: %w", err)
	}
	
	return nil
}

func (d *DockerSwarmDeployer) deployGrafana(ctx context.Context, opts DeploymentOptions) error {
	if opts.DryRun {
		return nil
	}
	
	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "osdo-grafana",
			Labels: map[string]string{
				"osdo.component": "grafana",
			},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "grafana/grafana:latest",
				Env: []string{
					"GF_SECURITY_ADMIN_PASSWORD=admin",
				},
			},
		},
		EndpointSpec: &swarm.EndpointSpec{
			Ports: []swarm.PortConfig{
				{
					Protocol:      swarm.PortConfigProtocolTCP,
					TargetPort:    3000,
					PublishedPort: 3000,
				},
			},
		},
	}
	
	_, err := d.client.ServiceCreate(ctx, serviceSpec, types.ServiceCreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create grafana service: %w", err)
	}
	
	return nil
}

func (d *DockerSwarmDeployer) deployTraefik(ctx context.Context, opts DeploymentOptions) error {
	if opts.DryRun {
		return nil
	}
	
	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "osdo-traefik",
			Labels: map[string]string{
				"osdo.component": "traefik",
			},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "traefik:latest",
				Args: []string{
					"--api.insecure=true",
					"--providers.docker.swarmMode=true",
					"--providers.docker.exposedbydefault=false",
					"--entrypoints.web.address=:80",
				},
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeBind,
						Source: "/var/run/docker.sock",
						Target: "/var/run/docker.sock",
					},
				},
			},
		},
		EndpointSpec: &swarm.EndpointSpec{
			Ports: []swarm.PortConfig{
				{
					Protocol:      swarm.PortConfigProtocolTCP,
					TargetPort:    80,
					PublishedPort: 80,
				},
				{
					Protocol:      swarm.PortConfigProtocolTCP,
					TargetPort:    8080,
					PublishedPort: 8080,
				},
			},
		},
	}
	
	_, err := d.client.ServiceCreate(ctx, serviceSpec, types.ServiceCreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create traefik service: %w", err)
	}
	
	return nil
}

func (d *DockerSwarmDeployer) undeployComponent(ctx context.Context, component string, opts DeploymentOptions) error {
	serviceName := fmt.Sprintf("osdo-%s", component)
	
	if err := d.client.ServiceRemove(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to remove service %s: %w", serviceName, err)
	}
	
	return nil
}

func (d *DockerSwarmDeployer) getComponentStatus(ctx context.Context, component string) (ComponentStatus, error) {
	serviceName := fmt.Sprintf("osdo-%s", component)
	
	service, _, err := d.client.ServiceInspectWithRaw(ctx, serviceName, types.ServiceInspectOptions{})
	if err != nil {
		return ComponentStatus{
			Name:   component,
			Status: "Not Found",
			Ready:  false,
		}, nil
	}
	
	// Get service tasks to determine actual status
	tasks, err := d.client.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", serviceName)),
	})
	if err != nil {
		return ComponentStatus{
			Name:   component,
			Status: "Unknown",
			Ready:  false,
		}, nil
	}
	
	runningTasks := 0
	for _, task := range tasks {
		if task.Status.State == swarm.TaskStateRunning {
			runningTasks++
		}
	}
	
	replicas := int32(1)
	if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
		replicas = int32(*service.Spec.Mode.Replicated.Replicas)
	}
	
	ready := int32(runningTasks) == replicas
	status := "Not Ready"
	if ready {
		status = "Running"
	} else if runningTasks > 0 {
		status = "Partially Ready"
	}
	
	return ComponentStatus{
		Name:   component,
		Status: status,
		Ready:  ready,
		Replicas: ReplicaStatus{
			Desired:   replicas,
			Ready:     int32(runningTasks),
			Available: int32(runningTasks),
			Updated:   int32(runningTasks),
		},
		Health: HealthStatus{
			Status:    status,
			LastCheck: time.Now(),
		},
	}, nil
}
