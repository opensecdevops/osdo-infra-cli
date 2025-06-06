package monitor

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemMetricsCollector collects system-level metrics
type SystemMetricsCollector struct {
	interval time.Duration
}

// NewSystemMetricsCollector creates a new system metrics collector
func NewSystemMetricsCollector(interval time.Duration) *SystemMetricsCollector {
	return &SystemMetricsCollector{
		interval: interval,
	}
}

// Collect implements MetricsCollector interface for system metrics
func (c *SystemMetricsCollector) Collect(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
		CPU:       &CPUMetrics{},
		Memory:    &MemoryMetrics{},
		Storage:   &StorageMetrics{},
		Network:   &NetworkMetrics{},
		System:    &SystemInfo{},
	}

	// Collect CPU metrics
	if err := c.collectCPUMetrics(metrics.CPU); err != nil {
		return nil, fmt.Errorf("error collecting CPU metrics: %w", err)
	}

	// Collect memory metrics
	if err := c.collectMemoryMetrics(metrics.Memory); err != nil {
		return nil, fmt.Errorf("error collecting memory metrics: %w", err)
	}

	// Collect storage metrics
	if err := c.collectStorageMetrics(metrics.Storage); err != nil {
		return nil, fmt.Errorf("error collecting storage metrics: %w", err)
	}

	// Collect network metrics
	if err := c.collectNetworkMetrics(metrics.Network); err != nil {
		return nil, fmt.Errorf("error collecting network metrics: %w", err)
	}

	// Collect system info
	if err := c.collectSystemInfo(metrics.System); err != nil {
		return nil, fmt.Errorf("error collecting system info: %w", err)
	}

	return metrics, nil
}

// collectCPUMetrics collects CPU usage metrics
func (c *SystemMetricsCollector) collectCPUMetrics(cpuMetrics *CPUMetrics) error {
	// Get CPU percentage
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return err
	}
	
	if len(percentages) > 0 {
		cpuMetrics.UsagePercent = percentages[0]
	}

	// Get CPU count
	cpuMetrics.Cores = runtime.NumCPU()

	// Get load average
	loadAvg, err := load.Avg()
	if err == nil {
		cpuMetrics.LoadAverage1m = loadAvg.Load1
		cpuMetrics.LoadAverage5m = loadAvg.Load5
		cpuMetrics.LoadAverage15m = loadAvg.Load15
	}

	return nil
}

// collectMemoryMetrics collects memory usage metrics
func (c *SystemMetricsCollector) collectMemoryMetrics(memMetrics *MemoryMetrics) error {
	// Virtual memory
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	memMetrics.TotalBytes = vmStat.Total
	memMetrics.UsedBytes = vmStat.Used
	memMetrics.FreeBytes = vmStat.Free
	memMetrics.AvailableBytes = vmStat.Available
	memMetrics.UsagePercent = vmStat.UsedPercent

	// Swap memory
	swapStat, err := mem.SwapMemory()
	if err == nil {
		memMetrics.SwapTotalBytes = swapStat.Total
		memMetrics.SwapUsedBytes = swapStat.Used
		memMetrics.SwapFreeBytes = swapStat.Free
		memMetrics.SwapUsagePercent = swapStat.UsedPercent
	}

	return nil
}

// collectStorageMetrics collects storage usage metrics
func (c *SystemMetricsCollector) collectStorageMetrics(storageMetrics *StorageMetrics) error {
	// Get disk usage for root partition
	usage, err := disk.Usage("/")
	if err != nil {
		return err
	}

	storageMetrics.TotalBytes = usage.Total
	storageMetrics.UsedBytes = usage.Used
	storageMetrics.FreeBytes = usage.Free
	storageMetrics.UsagePercent = usage.UsedPercent

	// Get disk IO statistics
	ioCounters, err := disk.IOCounters()
	if err == nil {
		var totalReads, totalWrites uint64
		for _, counter := range ioCounters {
			totalReads += counter.ReadCount
			totalWrites += counter.WriteCount
		}
		storageMetrics.ReadOps = totalReads
		storageMetrics.WriteOps = totalWrites
	}

	return nil
}

// collectNetworkMetrics collects network usage metrics
func (c *SystemMetricsCollector) collectNetworkMetrics(netMetrics *NetworkMetrics) error {
	// Get network IO statistics
	ioCounters, err := net.IOCounters(false)
	if err != nil {
		return err
	}

	if len(ioCounters) > 0 {
		counter := ioCounters[0]
		netMetrics.BytesReceived = counter.BytesRecv
		netMetrics.BytesSent = counter.BytesSent
		netMetrics.PacketsReceived = counter.PacketsRecv
		netMetrics.PacketsSent = counter.PacketsSent
		netMetrics.ErrorsIn = counter.Errin
		netMetrics.ErrorsOut = counter.Errout
		netMetrics.DroppedIn = counter.Dropin
		netMetrics.DroppedOut = counter.Dropout
	}

	return nil
}

// collectSystemInfo collects general system information
func (c *SystemMetricsCollector) collectSystemInfo(sysInfo *SystemInfo) error {
	// Host information
	hostInfo, err := host.Info()
	if err != nil {
		return err
	}

	sysInfo.Hostname = hostInfo.Hostname
	sysInfo.OS = hostInfo.OS
	sysInfo.Platform = hostInfo.Platform
	sysInfo.PlatformFamily = hostInfo.PlatformFamily
	sysInfo.PlatformVersion = hostInfo.PlatformVersion
	sysInfo.KernelVersion = hostInfo.KernelVersion
	sysInfo.Architecture = hostInfo.KernelArch
	sysInfo.UptimeSeconds = hostInfo.Uptime

	return nil
}

// CollectMetrics implements MetricsCollector interface
func (c *SystemMetricsCollector) CollectMetrics(ctx context.Context, opts MonitoringOptions) (map[string]interface{}, error) {
	metrics, err := c.Collect(ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert SystemMetrics to map[string]interface{}
	result := map[string]interface{}{
		"cpu": map[string]interface{}{
			"usage_percent":    metrics.CPU.UsagePercent,
			"cores":           metrics.CPU.Cores,
			"load_average_1m": metrics.CPU.LoadAverage1m,
			"load_average_5m": metrics.CPU.LoadAverage5m,
			"load_average_15m": metrics.CPU.LoadAverage15m,
		},
		"memory": map[string]interface{}{
			"total_bytes":         metrics.Memory.TotalBytes,
			"used_bytes":          metrics.Memory.UsedBytes,
			"free_bytes":          metrics.Memory.FreeBytes,
			"available_bytes":     metrics.Memory.AvailableBytes,
			"usage_percent":       metrics.Memory.UsagePercent,
			"swap_total_bytes":    metrics.Memory.SwapTotalBytes,
			"swap_used_bytes":     metrics.Memory.SwapUsedBytes,
			"swap_free_bytes":     metrics.Memory.SwapFreeBytes,
			"swap_usage_percent":  metrics.Memory.SwapUsagePercent,
		},
		"storage": map[string]interface{}{
			"total_bytes":    metrics.Storage.TotalBytes,
			"used_bytes":     metrics.Storage.UsedBytes,
			"free_bytes":     metrics.Storage.FreeBytes,
			"usage_percent":  metrics.Storage.UsagePercent,
			"read_ops":       metrics.Storage.ReadOps,
			"write_ops":      metrics.Storage.WriteOps,
		},
		"network": map[string]interface{}{
			"bytes_received":    metrics.Network.BytesReceived,
			"bytes_sent":        metrics.Network.BytesSent,
			"packets_received":  metrics.Network.PacketsReceived,
			"packets_sent":      metrics.Network.PacketsSent,
			"errors_in":         metrics.Network.ErrorsIn,
			"errors_out":        metrics.Network.ErrorsOut,
			"dropped_in":        metrics.Network.DroppedIn,
			"dropped_out":       metrics.Network.DroppedOut,
		},
		"system": map[string]interface{}{
			"hostname":         metrics.System.Hostname,
			"os":               metrics.System.OS,
			"platform":         metrics.System.Platform,
			"platform_family":  metrics.System.PlatformFamily,
			"platform_version": metrics.System.PlatformVersion,
			"kernel_version":   metrics.System.KernelVersion,
			"architecture":     metrics.System.Architecture,
			"uptime_seconds":   metrics.System.UptimeSeconds,
		},
	}
	
	return result, nil
}

// GetMetricHistory implements MetricsCollector interface
func (c *SystemMetricsCollector) GetMetricHistory(ctx context.Context, metric string, duration time.Duration) ([]MetricPoint, error) {
	// For now, return empty history as we don't have persistent storage
	return []MetricPoint{}, nil
}

// GetSupportedMetrics implements MetricsCollector interface
func (c *SystemMetricsCollector) GetSupportedMetrics() []string {
	return []string{
		"cpu.total", "cpu.used", "cpu.percentage",
		"memory.total", "memory.used", "memory.percentage",
		"storage.total", "storage.used", "storage.percentage",
		"network.bytes_in", "network.bytes_out", "network.packets_in", "network.packets_out",
		"system.os", "system.arch", "system.hostname", "system.uptime", "system.load_avg", "system.processes",
	}
}

// GetInterval returns the collection interval
func (c *SystemMetricsCollector) GetInterval() time.Duration {
	return c.interval
}

// SetInterval sets the collection interval
func (c *SystemMetricsCollector) SetInterval(interval time.Duration) {
	c.interval = interval
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	Timestamp time.Time    `json:"timestamp"`
	CPU       *CPUMetrics  `json:"cpu"`
	Memory    *MemoryMetrics `json:"memory"`
	Storage   *StorageMetrics `json:"storage"`
	Network   *NetworkMetrics `json:"network"`
	System    *SystemInfo   `json:"system"`
}

// CPUMetrics represents CPU usage metrics
type CPUMetrics struct {
	UsagePercent    float64 `json:"usage_percent"`
	Cores           int     `json:"cores"`
	LoadAverage1m   float64 `json:"load_average_1m"`
	LoadAverage5m   float64 `json:"load_average_5m"`
	LoadAverage15m  float64 `json:"load_average_15m"`
}

// MemoryMetrics represents memory usage metrics
type MemoryMetrics struct {
	TotalBytes        uint64  `json:"total_bytes"`
	UsedBytes         uint64  `json:"used_bytes"`
	FreeBytes         uint64  `json:"free_bytes"`
	AvailableBytes    uint64  `json:"available_bytes"`
	UsagePercent      float64 `json:"usage_percent"`
	SwapTotalBytes    uint64  `json:"swap_total_bytes"`
	SwapUsedBytes     uint64  `json:"swap_used_bytes"`
	SwapFreeBytes     uint64  `json:"swap_free_bytes"`
	SwapUsagePercent  float64 `json:"swap_usage_percent"`
}

// StorageMetrics represents storage usage metrics
type StorageMetrics struct {
	TotalBytes    uint64  `json:"total_bytes"`
	UsedBytes     uint64  `json:"used_bytes"`
	FreeBytes     uint64  `json:"free_bytes"`
	UsagePercent  float64 `json:"usage_percent"`
	ReadOps       uint64  `json:"read_ops"`
	WriteOps      uint64  `json:"write_ops"`
}

// NetworkMetrics represents network usage metrics
type NetworkMetrics struct {
	BytesReceived    uint64 `json:"bytes_received"`
	BytesSent        uint64 `json:"bytes_sent"`
	PacketsReceived  uint64 `json:"packets_received"`
	PacketsSent      uint64 `json:"packets_sent"`
	ErrorsIn         uint64 `json:"errors_in"`
	ErrorsOut        uint64 `json:"errors_out"`
	DroppedIn        uint64 `json:"dropped_in"`
	DroppedOut       uint64 `json:"dropped_out"`
}

// SystemInfo represents general system information
type SystemInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformFamily  string `json:"platform_family"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	Architecture    string `json:"architecture"`
	UptimeSeconds   uint64 `json:"uptime_seconds"`
}
