package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// DockerMetricsCollector collects Docker-specific metrics
type DockerMetricsCollector struct {
	client   *client.Client
	interval time.Duration
}

// NewDockerMetricsCollector creates a new Docker metrics collector
func NewDockerMetricsCollector(interval time.Duration) (*DockerMetricsCollector, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerMetricsCollector{
		client:   cli,
		interval: interval,
	}, nil
}

// Collect implements MetricsCollector interface for Docker metrics
func (c *DockerMetricsCollector) Collect(ctx context.Context) (*DockerMetrics, error) {
	metrics := &DockerMetrics{
		Timestamp:  time.Now(),
		Containers: make(map[string]*ContainerMetrics),
		Images:     &ImageMetrics{},
		Network:    &DockerNetworkMetrics{},
		Volume:     &VolumeMetrics{},
	}

	// Collect container metrics
	if err := c.collectContainerMetrics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("error collecting container metrics: %w", err)
	}

	// Collect image metrics
	if err := c.collectImageMetrics(ctx, metrics.Images); err != nil {
		return nil, fmt.Errorf("error collecting image metrics: %w", err)
	}

	// Collect network metrics
	if err := c.collectNetworkMetrics(ctx, metrics.Network); err != nil {
		return nil, fmt.Errorf("error collecting network metrics: %w", err)
	}

	// Collect volume metrics
	if err := c.collectVolumeMetrics(ctx, metrics.Volume); err != nil {
		return nil, fmt.Errorf("error collecting volume metrics: %w", err)
	}

	return metrics, nil
}

// collectContainerMetrics collects metrics for all containers
func (c *DockerMetricsCollector) collectContainerMetrics(ctx context.Context, metrics *DockerMetrics) error {
	containers, err := c.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}

	metrics.TotalContainers = len(containers)
	runningCount := 0
	stoppedCount := 0
	errorCount := 0

	for _, container := range containers {
		containerMetrics := &ContainerMetrics{
			ID:      container.ID[:12], // Short ID
			Name:    container.Names[0][1:], // Remove leading slash
			Image:   container.Image,
			State:   container.State,
			Status:  container.Status,
			Created: time.Unix(container.Created, 0),
		}

		// Count containers by state
		switch container.State {
		case "running":
			runningCount++
		case "exited":
			stoppedCount++
		default:
			errorCount++
		}

		// Get container stats for running containers
		if container.State == "running" {
			stats, err := c.client.ContainerStats(ctx, container.ID, false)
			if err == nil {
				containerMetrics.Stats = &ContainerStats{}
				// Read stats would be implemented here
				// For now, we'll set basic info
				stats.Body.Close()
			}
		}

		metrics.Containers[container.ID[:12]] = containerMetrics
	}

	metrics.RunningContainers = runningCount
	metrics.StoppedContainers = stoppedCount
	metrics.ErrorContainers = errorCount

	return nil
}

// collectImageMetrics collects Docker image metrics
func (c *DockerMetricsCollector) collectImageMetrics(ctx context.Context, imageMetrics *ImageMetrics) error {
	images, err := c.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return err
	}

	imageMetrics.TotalImages = len(images)
	
	var totalSize int64
	danglingCount := 0

	for _, image := range images {
		totalSize += image.Size
		
		// Check for dangling images (no repository tags)
		if len(image.RepoTags) == 0 || (len(image.RepoTags) == 1 && image.RepoTags[0] == "<none>:<none>") {
			danglingCount++
		}
	}

	imageMetrics.TotalSizeBytes = totalSize
	imageMetrics.DanglingImages = danglingCount

	return nil
}

// collectNetworkMetrics collects Docker network metrics
func (c *DockerMetricsCollector) collectNetworkMetrics(ctx context.Context, netMetrics *DockerNetworkMetrics) error {
	networks, err := c.client.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return err
	}

	netMetrics.TotalNetworks = len(networks)
	
	customCount := 0
	for _, network := range networks {
		// Count custom networks (not default bridge, host, none)
		if network.Driver != "bridge" || network.Name != "bridge" {
			if network.Name != "host" && network.Name != "none" {
				customCount++
			}
		}
	}
	
	netMetrics.CustomNetworks = customCount

	return nil
}

// collectVolumeMetrics collects Docker volume metrics
func (c *DockerMetricsCollector) collectVolumeMetrics(ctx context.Context, volMetrics *VolumeMetrics) error {
	volumes, err := c.client.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return err
	}

	volMetrics.TotalVolumes = len(volumes.Volumes)
	
	// Count dangling volumes (not attached to any container)
	danglingCount := 0
	for _, volume := range volumes.Volumes {
		if volume.UsageData != nil && volume.UsageData.RefCount == 0 {
			danglingCount++
		}
	}
	
	volMetrics.DanglingVolumes = danglingCount

	return nil
}

// GetInterval returns the collection interval
func (c *DockerMetricsCollector) GetInterval() time.Duration {
	return c.interval
}

// SetInterval sets the collection interval
func (c *DockerMetricsCollector) SetInterval(interval time.Duration) {
	c.interval = interval
}

// CollectMetrics implements MetricsCollector interface
func (c *DockerMetricsCollector) CollectMetrics(ctx context.Context, opts MonitoringOptions) (map[string]interface{}, error) {
	metrics, err := c.Collect(ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert DockerMetrics to map[string]interface{}
	result := map[string]interface{}{
		"containers": map[string]interface{}{
			"total":   metrics.TotalContainers,
			"running": metrics.RunningContainers,
			"stopped": metrics.StoppedContainers,
			"error":   metrics.ErrorContainers,
		},
		"images": map[string]interface{}{
			"total":           metrics.Images.TotalImages,
			"total_size_bytes": metrics.Images.TotalSizeBytes,
			"dangling":        metrics.Images.DanglingImages,
		},
		"networks": map[string]interface{}{
			"total":  metrics.Network.TotalNetworks,
			"custom": metrics.Network.CustomNetworks,
		},
		"volumes": map[string]interface{}{
			"total":    metrics.Volume.TotalVolumes,
			"dangling": metrics.Volume.DanglingVolumes,
		},
	}
	
	return result, nil
}

// GetMetricHistory implements MetricsCollector interface
func (c *DockerMetricsCollector) GetMetricHistory(ctx context.Context, metric string, duration time.Duration) ([]MetricPoint, error) {
	// For now, return empty history as we don't have persistent storage
	return []MetricPoint{}, nil
}

// GetSupportedMetrics implements MetricsCollector interface
func (c *DockerMetricsCollector) GetSupportedMetrics() []string {
	return []string{
		"containers.total", "containers.running", "containers.stopped", "containers.error",
		"images.total", "images.total_size_bytes", "images.dangling",
		"networks.total", "networks.custom",
		"volumes.total", "volumes.dangling",
	}
}

// Close closes the Docker client
func (c *DockerMetricsCollector) Close() error {
	return c.client.Close()
}

// DockerMetrics represents Docker-specific metrics
type DockerMetrics struct {
	Timestamp         time.Time                    `json:"timestamp"`
	TotalContainers   int                          `json:"total_containers"`
	RunningContainers int                          `json:"running_containers"`
	StoppedContainers int                          `json:"stopped_containers"`
	ErrorContainers   int                          `json:"error_containers"`
	Containers        map[string]*ContainerMetrics `json:"containers"`
	Images            *ImageMetrics                `json:"images"`
	Network           *DockerNetworkMetrics        `json:"network"`
	Volume            *VolumeMetrics               `json:"volume"`
}

// ContainerMetrics represents metrics for a single container
type ContainerMetrics struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Image   string           `json:"image"`
	State   string           `json:"state"`
	Status  string           `json:"status"`
	Created time.Time        `json:"created"`
	Stats   *ContainerStats  `json:"stats,omitempty"`
}

// ContainerStats represents runtime statistics for a container
type ContainerStats struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsageBytes   uint64  `json:"memory_usage_bytes"`
	MemoryLimitBytes   uint64  `json:"memory_limit_bytes"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	NetworkRxBytes     uint64  `json:"network_rx_bytes"`
	NetworkTxBytes     uint64  `json:"network_tx_bytes"`
	BlockReadBytes     uint64  `json:"block_read_bytes"`
	BlockWriteBytes    uint64  `json:"block_write_bytes"`
}

// ImageMetrics represents Docker image metrics
type ImageMetrics struct {
	TotalImages    int   `json:"total_images"`
	TotalSizeBytes int64 `json:"total_size_bytes"`
	DanglingImages int   `json:"dangling_images"`
}

// DockerNetworkMetrics represents Docker network metrics
type DockerNetworkMetrics struct {
	TotalNetworks  int `json:"total_networks"`
	CustomNetworks int `json:"custom_networks"`
}

// VolumeMetrics represents Docker volume metrics
type VolumeMetrics struct {
	TotalVolumes    int `json:"total_volumes"`
	DanglingVolumes int `json:"dangling_volumes"`
}
