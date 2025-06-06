package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/deployer"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// DockerComposeMonitor monitors Docker Compose deployments
type DockerComposeMonitor struct {
	client     *client.Client
	composeDir string
}

// NewDockerComposeMonitor creates a new Docker Compose monitor
func NewDockerComposeMonitor() *DockerComposeMonitor {
	return &DockerComposeMonitor{
		composeDir: "./docker-compose",
	}
}

// Initialize initializes the Docker client
func (d *DockerComposeMonitor) Initialize() error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	
	d.client = cli
	return nil
}

// GetSystemStatus returns the current system status
func (d *DockerComposeMonitor) GetSystemStatus(ctx context.Context, opts MonitoringOptions) (*SystemStatus, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	status := &SystemStatus{
		Timestamp:  time.Now(),
		Platform:   config.PlatformDockerCompose,
		Components: make(map[string]deployer.ComponentStatus),
		Alerts:     make([]Alert, 0),
		Metadata:   make(map[string]interface{}),
	}
	
	// Get component status
	if len(opts.Components) > 0 {
		componentStatus, err := d.GetComponentStatus(ctx, opts.Components, opts)
		if err == nil {
			status.Components = componentStatus
		}
	}
	
	// Get resource usage
	resourceUsage, err := d.GetResourceUsage(ctx, opts)
	if err == nil {
		status.Resources = *resourceUsage
	}
	
	// Get alerts
	alerts, err := d.GetAlerts(ctx, opts)
	if err == nil {
		status.Alerts = alerts
	}
	
	// Determine overall health
	status.OverallHealth = d.calculateOverallHealth(status)
	
	return status, nil
}

// GetComponentStatus returns the status of specific components
func (d *DockerComposeMonitor) GetComponentStatus(ctx context.Context, components []string, opts MonitoringOptions) (map[string]deployer.ComponentStatus, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	status := make(map[string]deployer.ComponentStatus)
	
	for _, component := range components {
		containerName := fmt.Sprintf("osdo-%s", component)
		
		inspect, err := d.client.ContainerInspect(ctx, containerName)
		if err != nil {
			status[component] = deployer.ComponentStatus{
				Name:   component,
				Status: "Not Found",
				Ready:  false,
			}
			continue
		}
		
		containerStatus := "Unknown"
		ready := false
		
		if inspect.State.Running {
			containerStatus = "Running"
			ready = true
		} else if inspect.State.Restarting {
			containerStatus = "Restarting"
		} else if inspect.State.Paused {
			containerStatus = "Paused"
		} else {
			containerStatus = "Stopped"
		}
		
		status[component] = deployer.ComponentStatus{
			Name:   component,
			Status: containerStatus,
			Ready:  ready,
			Health: deployer.HealthStatus{
				Status:    containerStatus,
				LastCheck: time.Now(),
			},
		}
	}
	
	return status, nil
}

// GetResourceUsage returns resource usage information
func (d *DockerComposeMonitor) GetResourceUsage(ctx context.Context, opts MonitoringOptions) (*ResourceUsage, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
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
	
	// Get system info
	info, err := d.client.Info(ctx)
	if err != nil {
		return usage, nil
	}
	
	// Set basic system information
	usage.CPU.Total = fmt.Sprintf("%d", info.NCPU)
	usage.Memory.Total = fmt.Sprintf("%d", info.MemTotal)
	
	// Get container stats
	filters := filters.NewArgs()
	filters.Add("label", "osdo.component")
	
	containers, err := d.client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters,
	})
	if err != nil {
		return usage, nil
	}
	
	var totalCPU, totalMemory uint64
	for _, container := range containers {
		stats, err := d.client.ContainerStats(ctx, container.ID, false)
		if err != nil {
			continue
		}
		
		// This is a simplified calculation
		// In reality, you'd parse the stats JSON response
		stats.Body.Close()
	}
	
	usage.CPU.Used = fmt.Sprintf("%.2f", float64(totalCPU)/1000000000)
	usage.Memory.Used = fmt.Sprintf("%d", totalMemory)
	
	if info.NCPU > 0 {
		usage.CPU.Percentage = float64(totalCPU) / float64(info.NCPU) * 100
	}
	if info.MemTotal > 0 {
		usage.Memory.Percentage = float64(totalMemory) / float64(info.MemTotal) * 100
	}
	
	return usage, nil
}

// GetAlerts returns active alerts
func (d *DockerComposeMonitor) GetAlerts(ctx context.Context, opts MonitoringOptions) ([]Alert, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	alerts := make([]Alert, 0)
	
	// Get OSDO containers
	nameFilters := filters.NewArgs()
	nameFilters.Add("name", "osdo-")
	
	containers, err := d.client.ContainerList(ctx, types.ContainerListOptions{
		All: true,
		Filters: nameFilters,
	})
	if err != nil {
		return alerts, nil
	}
	
	for _, container := range containers {
		if container.State != "running" {
			alerts = append(alerts, Alert{
				ID:        fmt.Sprintf("container-%s-not-running", container.ID[:12]),
				Severity:  AlertSeverityWarning,
				Component: "container",
				Message:   fmt.Sprintf("Container %s is not running (state: %s)", container.Names[0], container.State),
				Timestamp: time.Now(),
				Status:    AlertStatusFiring,
				Labels: map[string]string{
					"container_id":   container.ID,
					"container_name": container.Names[0],
					"state":          container.State,
					"type":           "container_not_running",
				},
			})
		}
	}
	
	return alerts, nil
}

// RunHealthChecks performs health checks on components
func (d *DockerComposeMonitor) RunHealthChecks(ctx context.Context, components []string, opts MonitoringOptions) (map[string]HealthCheck, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	checks := make(map[string]HealthCheck)
	
	for _, component := range components {
		check := d.runComponentHealthCheck(ctx, component)
		checks[component] = check
	}
	
	return checks, nil
}

// StartWatching starts monitoring in watch mode
func (d *DockerComposeMonitor) StartWatching(ctx context.Context, opts MonitoringOptions) (<-chan *SystemStatus, error) {
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
				status, err := d.GetSystemStatus(ctx, opts)
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
func (d *DockerComposeMonitor) GetPlatform() config.Platform {
	return config.PlatformDockerCompose
}

// IsReady checks if monitoring is ready
func (d *DockerComposeMonitor) IsReady(ctx context.Context) error {
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

func (d *DockerComposeMonitor) runComponentHealthCheck(ctx context.Context, component string) HealthCheck {
	startTime := time.Now()
	
	check := HealthCheck{
		Name:      component,
		Timestamp: startTime,
		Metadata:  make(map[string]string),
	}
	
	containerName := fmt.Sprintf("osdo-%s", component)
	
	inspect, err := d.client.ContainerInspect(ctx, containerName)
	if err != nil {
		check.Success = false
		check.Status = "Not Found"
		check.Message = "Container not found"
	} else {
		check.Success = inspect.State.Running
		if inspect.State.Running {
			check.Status = "Running"
			check.Message = "Container is healthy"
		} else {
			check.Status = "Stopped"
			check.Message = "Container is not running"
		}
	}
	
	check.Duration = time.Since(startTime)
	return check
}

func (d *DockerComposeMonitor) calculateOverallHealth(status *SystemStatus) string {
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
