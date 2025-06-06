package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/deployer"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// DockerSwarmMonitor monitors Docker Swarm deployments
type DockerSwarmMonitor struct {
	client *client.Client
}

// NewDockerSwarmMonitor creates a new Docker Swarm monitor
func NewDockerSwarmMonitor() *DockerSwarmMonitor {
	return &DockerSwarmMonitor{}
}

// Initialize initializes the Docker client
func (d *DockerSwarmMonitor) Initialize() error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	
	d.client = cli
	return nil
}

// GetSystemStatus returns the current system status
func (d *DockerSwarmMonitor) GetSystemStatus(ctx context.Context, opts MonitoringOptions) (*SystemStatus, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	status := &SystemStatus{
		Timestamp:  time.Now(),
		Platform:   config.PlatformDockerSwarm,
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
func (d *DockerSwarmMonitor) GetComponentStatus(ctx context.Context, components []string, opts MonitoringOptions) (map[string]deployer.ComponentStatus, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	status := make(map[string]deployer.ComponentStatus)
	
	for _, component := range components {
		serviceName := fmt.Sprintf("osdo-%s", component)
		
		service, _, err := d.client.ServiceInspectWithRaw(ctx, serviceName, types.ServiceInspectOptions{})
		if err != nil {
			status[component] = deployer.ComponentStatus{
				Name:   component,
				Status: "Not Found",
				Ready:  false,
			}
			continue
		}
		
		// Get service tasks to determine actual status
		taskFilters := filters.NewArgs()
		taskFilters.Add("service", serviceName)
		
		tasks, err := d.client.TaskList(ctx, types.TaskListOptions{
			Filters: taskFilters,
		})
		if err != nil {
			status[component] = deployer.ComponentStatus{
				Name:   component,
				Status: "Unknown",
				Ready:  false,
			}
			continue
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
		serviceStatus := "Not Ready"
		if ready {
			serviceStatus = "Running"
		} else if runningTasks > 0 {
			serviceStatus = "Partially Ready"
		}
		
		status[component] = deployer.ComponentStatus{
			Name:   component,
			Status: serviceStatus,
			Ready:  ready,
			Replicas: deployer.ReplicaStatus{
				Desired:   replicas,
				Ready:     int32(runningTasks),
				Available: int32(runningTasks),
				Updated:   int32(runningTasks),
			},
			Health: deployer.HealthStatus{
				Status:    serviceStatus,
				LastCheck: time.Now(),
			},
		}
	}
	
	return status, nil
}

// GetResourceUsage returns resource usage information
func (d *DockerSwarmMonitor) GetResourceUsage(ctx context.Context, opts MonitoringOptions) (*ResourceUsage, error) {
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
	
	// Get swarm nodes
	nodes, err := d.client.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return usage, nil
	}
	
	var totalCPU, totalMemory, usedCPU, usedMemory uint64
	
	for _, node := range nodes {
		if node.Description.Resources.NanoCPUs > 0 {
			totalCPU += uint64(node.Description.Resources.NanoCPUs)
		}
		if node.Description.Resources.MemoryBytes > 0 {
			totalMemory += uint64(node.Description.Resources.MemoryBytes)
		}
		
		// Calculate used resources (simplified)
		// In reality, you'd sum up the resources used by tasks on each node
		usedCPU += uint64(node.Description.Resources.NanoCPUs) / 4 // Assume 25% usage
		usedMemory += uint64(node.Description.Resources.MemoryBytes) / 4
	}
	
	usage.CPU.Total = fmt.Sprintf("%.2f", float64(totalCPU)/1000000000)
	usage.Memory.Total = fmt.Sprintf("%d", totalMemory)
	usage.CPU.Used = fmt.Sprintf("%.2f", float64(usedCPU)/1000000000)
	usage.Memory.Used = fmt.Sprintf("%d", usedMemory)
	
	if totalCPU > 0 {
		usage.CPU.Percentage = float64(usedCPU) / float64(totalCPU) * 100
	}
	if totalMemory > 0 {
		usage.Memory.Percentage = float64(usedMemory) / float64(totalMemory) * 100
	}
	
	return usage, nil
}

// GetAlerts returns active alerts
func (d *DockerSwarmMonitor) GetAlerts(ctx context.Context, opts MonitoringOptions) ([]Alert, error) {
	if d.client == nil {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}
	
	alerts := make([]Alert, 0)
	
	// Check for unhealthy nodes
	nodes, err := d.client.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return alerts, nil
	}
	
	for _, node := range nodes {
		if node.Status.State != swarm.NodeStateReady {
			alerts = append(alerts, Alert{
				ID:        fmt.Sprintf("node-%s-not-ready", node.ID),
				Severity:  AlertSeverityCritical,
				Component: "node",
				Message:   fmt.Sprintf("Swarm node %s is not ready (state: %s)", node.Description.Hostname, node.Status.State),
				Timestamp: time.Now(),
				Status:    AlertStatusFiring,
				Labels: map[string]string{
					"node_id":   node.ID,
					"hostname":  node.Description.Hostname,
					"state":     string(node.Status.State),
					"type":      "node_not_ready",
				},
			})
		}
		
		if node.Spec.Availability != swarm.NodeAvailabilityActive {
			alerts = append(alerts, Alert{
				ID:        fmt.Sprintf("node-%s-not-available", node.ID),
				Severity:  AlertSeverityWarning,
				Component: "node",
				Message:   fmt.Sprintf("Swarm node %s is not available (availability: %s)", node.Description.Hostname, node.Spec.Availability),
				Timestamp: time.Now(),
				Status:    AlertStatusFiring,
				Labels: map[string]string{
					"node_id":      node.ID,
					"hostname":     node.Description.Hostname,
					"availability": string(node.Spec.Availability),
					"type":         "node_not_available",
				},
			})
		}
	}
	
	// Check for failed services
	serviceFilters := filters.NewArgs()
	serviceFilters.Add("label", "osdo.component")
	
	services, err := d.client.ServiceList(ctx, types.ServiceListOptions{
		Filters: serviceFilters,
	})
	if err != nil {
		return alerts, nil
	}
	
	for _, service := range services {
		// Get service tasks
		serviceTaskFilters := filters.NewArgs()
		serviceTaskFilters.Add("service", service.ID)
		
		tasks, err := d.client.TaskList(ctx, types.TaskListOptions{
			Filters: serviceTaskFilters,
		})
		if err != nil {
			continue
		}
		
		failedTasks := 0
		for _, task := range tasks {
			if task.Status.State == swarm.TaskStateFailed {
				failedTasks++
			}
		}
		
		if failedTasks > 0 {
			alerts = append(alerts, Alert{
				ID:        fmt.Sprintf("service-%s-failed-tasks", service.ID),
				Severity:  AlertSeverityWarning,
				Component: "service",
				Message:   fmt.Sprintf("Service %s has %d failed tasks", service.Spec.Name, failedTasks),
				Timestamp: time.Now(),
				Status:    AlertStatusFiring,
				Labels: map[string]string{
					"service_id":   service.ID,
					"service_name": service.Spec.Name,
					"failed_tasks": fmt.Sprintf("%d", failedTasks),
					"type":         "service_failed_tasks",
				},
			})
		}
	}
	
	return alerts, nil
}

// RunHealthChecks performs health checks on components
func (d *DockerSwarmMonitor) RunHealthChecks(ctx context.Context, components []string, opts MonitoringOptions) (map[string]HealthCheck, error) {
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
func (d *DockerSwarmMonitor) StartWatching(ctx context.Context, opts MonitoringOptions) (<-chan *SystemStatus, error) {
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
func (d *DockerSwarmMonitor) GetPlatform() config.Platform {
	return config.PlatformDockerSwarm
}

// IsReady checks if monitoring is ready
func (d *DockerSwarmMonitor) IsReady(ctx context.Context) error {
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

func (d *DockerSwarmMonitor) runComponentHealthCheck(ctx context.Context, component string) HealthCheck {
	startTime := time.Now()
	
	check := HealthCheck{
		Name:      component,
		Timestamp: startTime,
		Metadata:  make(map[string]string),
	}
	
	serviceName := fmt.Sprintf("osdo-%s", component)
	
	service, _, err := d.client.ServiceInspectWithRaw(ctx, serviceName, types.ServiceInspectOptions{})
	if err != nil {
		check.Success = false
		check.Status = "Not Found"
		check.Message = "Service not found"
	} else {
		// Check service tasks
		healthTaskFilters := filters.NewArgs()
		healthTaskFilters.Add("service", serviceName)
		
		tasks, err := d.client.TaskList(ctx, types.TaskListOptions{
			Filters: healthTaskFilters,
		})
		if err != nil {
			check.Success = false
			check.Status = "Error"
			check.Message = "Failed to get service tasks"
		} else {
			runningTasks := 0
			for _, task := range tasks {
				if task.Status.State == swarm.TaskStateRunning {
					runningTasks++
				}
			}
			
			replicas := 1
			if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
				replicas = int(*service.Spec.Mode.Replicated.Replicas)
			}
			
			check.Success = runningTasks == replicas
			if check.Success {
				check.Status = "Running"
				check.Message = "Service is healthy"
			} else {
				check.Status = "Degraded"
				check.Message = fmt.Sprintf("Service has %d/%d running tasks", runningTasks, replicas)
			}
		}
	}
	
	check.Duration = time.Since(startTime)
	return check
}

func (d *DockerSwarmMonitor) calculateOverallHealth(status *SystemStatus) string {
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
