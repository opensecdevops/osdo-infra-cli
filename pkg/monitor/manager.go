package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
)

// Manager manages monitoring operations across different platforms
type Manager struct {
	monitors   map[config.Platform]Monitor
	collectors map[string]MetricsCollector
	alerts     AlertManager
	logs       LogCollector
}

// NewManager creates a new monitoring manager
func NewManager() *Manager {
	return &Manager{
		monitors:   make(map[config.Platform]Monitor),
		collectors: make(map[string]MetricsCollector),
	}
}

// RegisterMonitor registers a monitor for a platform
func (m *Manager) RegisterMonitor(platform config.Platform, monitor Monitor) {
	m.monitors[platform] = monitor
}

// RegisterMetricsCollector registers a metrics collector
func (m *Manager) RegisterMetricsCollector(name string, collector MetricsCollector) {
	m.collectors[name] = collector
}

// SetAlertManager sets the alert manager
func (m *Manager) SetAlertManager(alertManager AlertManager) {
	m.alerts = alertManager
}

// SetLogCollector sets the log collector
func (m *Manager) SetLogCollector(logCollector LogCollector) {
	m.logs = logCollector
}

// GetSystemStatus returns comprehensive system status
func (m *Manager) GetSystemStatus(ctx context.Context, opts MonitoringOptions) (*SystemStatus, error) {
	monitor, exists := m.monitors[opts.Platform]
	if !exists {
		return nil, fmt.Errorf("no monitor registered for platform: %s", opts.Platform)
	}
	
	return monitor.GetSystemStatus(ctx, opts)
}

// GetMultiPlatformStatus returns status across multiple platforms
func (m *Manager) GetMultiPlatformStatus(ctx context.Context, platforms []config.Platform, opts MonitoringOptions) (map[config.Platform]*SystemStatus, error) {
	result := make(map[config.Platform]*SystemStatus)
	
	for _, platform := range platforms {
		opts.Platform = platform
		status, err := m.GetSystemStatus(ctx, opts)
		if err != nil {
			continue // Skip platforms that error out
		}
		result[platform] = status
	}
	
	return result, nil
}

// StartWatching starts monitoring in watch mode
func (m *Manager) StartWatching(ctx context.Context, opts MonitoringOptions) (<-chan *SystemStatus, error) {
	monitor, exists := m.monitors[opts.Platform]
	if !exists {
		return nil, fmt.Errorf("no monitor registered for platform: %s", opts.Platform)
	}
	
	return monitor.StartWatching(ctx, opts)
}

// GetComponentHealth performs health checks on components
func (m *Manager) GetComponentHealth(ctx context.Context, components []string, opts MonitoringOptions) (map[string]HealthCheck, error) {
	monitor, exists := m.monitors[opts.Platform]
	if !exists {
		return nil, fmt.Errorf("no monitor registered for platform: %s", opts.Platform)
	}
	
	return monitor.RunHealthChecks(ctx, components, opts)
}

// GetAlerts returns alerts from the alert manager
func (m *Manager) GetAlerts(ctx context.Context, filters map[string]string) ([]Alert, error) {
	if m.alerts == nil {
		return []Alert{}, nil
	}
	
	return m.alerts.GetAlerts(ctx, filters)
}

// GetLogs returns logs for components
func (m *Manager) GetLogs(ctx context.Context, components []string, opts LogOptions) (map[string][]LogEntry, error) {
	if m.logs == nil {
		return nil, fmt.Errorf("no log collector configured")
	}
	
	return m.logs.GetLogs(ctx, components, opts)
}

// CollectMetrics collects metrics from registered collectors
func (m *Manager) CollectMetrics(ctx context.Context, opts MonitoringOptions) (map[string]map[string]interface{}, error) {
	result := make(map[string]map[string]interface{})
	
	for name, collector := range m.collectors {
		metrics, err := collector.CollectMetrics(ctx, opts)
		if err != nil {
			continue // Skip collectors that error out
		}
		result[name] = metrics
	}
	
	return result, nil
}

// GetResourceUsage returns resource usage across platforms
func (m *Manager) GetResourceUsage(ctx context.Context, opts MonitoringOptions) (*ResourceUsage, error) {
	monitor, exists := m.monitors[opts.Platform]
	if !exists {
		return nil, fmt.Errorf("no monitor registered for platform: %s", opts.Platform)
	}
	
	return monitor.GetResourceUsage(ctx, opts)
}

// IsReady checks if monitoring is ready for the specified platform
func (m *Manager) IsReady(ctx context.Context, platform config.Platform) error {
	monitor, exists := m.monitors[platform]
	if !exists {
		return fmt.Errorf("no monitor registered for platform: %s", platform)
	}
	
	return monitor.IsReady(ctx)
}

// GetSupportedPlatforms returns platforms with registered monitors
func (m *Manager) GetSupportedPlatforms() []config.Platform {
	platforms := make([]config.Platform, 0, len(m.monitors))
	for platform := range m.monitors {
		platforms = append(platforms, platform)
	}
	return platforms
}

// InitializeDefaultMonitors initializes monitors for all supported platforms
func (m *Manager) InitializeDefaultMonitors() {
	// Register Kubernetes monitor
	m.RegisterMonitor(config.PlatformKubernetes, NewKubernetesMonitor())
	
	// Register K3s monitor (reuse Kubernetes monitor)
	m.RegisterMonitor(config.PlatformK3s, NewKubernetesMonitor())
	
	// Register Docker Compose monitor
	m.RegisterMonitor(config.PlatformDockerCompose, NewDockerComposeMonitor())
	
	// Register Docker Swarm monitor
	m.RegisterMonitor(config.PlatformDockerSwarm, NewDockerSwarmMonitor())
	
	// Register Helm monitor (reuse Kubernetes monitor)
	m.RegisterMonitor(config.PlatformHelm, NewKubernetesMonitor())
	
	// Register default metrics collectors
	systemCollector := NewSystemMetricsCollector(30 * time.Second)
	m.RegisterMetricsCollector("system", systemCollector)
	
	if dockerCollector, err := NewDockerMetricsCollector(30 * time.Second); err == nil {
		m.RegisterMetricsCollector("docker", dockerCollector)
	}
}

// DefaultManager is the default monitoring manager instance
var DefaultManager = NewManager()

// Convenience functions

// GetSystemStatus is a convenience function to get system status
func GetSystemStatus(ctx context.Context, opts MonitoringOptions) (*SystemStatus, error) {
	return DefaultManager.GetSystemStatus(ctx, opts)
}

// StartWatching is a convenience function to start watching
func StartWatching(ctx context.Context, opts MonitoringOptions) (<-chan *SystemStatus, error) {
	return DefaultManager.StartWatching(ctx, opts)
}

// GetComponentHealth is a convenience function to get component health
func GetComponentHealth(ctx context.Context, components []string, opts MonitoringOptions) (map[string]HealthCheck, error) {
	return DefaultManager.GetComponentHealth(ctx, components, opts)
}

// FormatStatus formats system status for display
func FormatStatus(status *SystemStatus, format string) (string, error) {
	switch format {
	case "table":
		return formatStatusAsTable(status), nil
	case "json":
		return formatStatusAsJSON(status)
	case "yaml":
		return formatStatusAsYAML(status)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// Helper functions for formatting

func formatStatusAsTable(status *SystemStatus) string {
	// Implementation for table format
	result := fmt.Sprintf("Platform: %s\n", status.Platform)
	result += fmt.Sprintf("Overall Health: %s\n", status.OverallHealth)
	result += fmt.Sprintf("Timestamp: %s\n\n", status.Timestamp.Format(time.RFC3339))
	
	result += "Components:\n"
	for name, component := range status.Components {
		result += fmt.Sprintf("  %s: %s (Ready: %t)\n", name, component.Status, component.Ready)
	}
	
	result += "\nResource Usage:\n"
	result += fmt.Sprintf("  CPU: %s/%s (%.1f%%)\n", 
		status.Resources.CPU.Used, 
		status.Resources.CPU.Total, 
		status.Resources.CPU.Percentage)
	result += fmt.Sprintf("  Memory: %s/%s (%.1f%%)\n", 
		status.Resources.Memory.Used, 
		status.Resources.Memory.Total, 
		status.Resources.Memory.Percentage)
	
	if len(status.Alerts) > 0 {
		result += "\nAlerts:\n"
		for _, alert := range status.Alerts {
			result += fmt.Sprintf("  [%s] %s: %s\n", alert.Severity, alert.Component, alert.Message)
		}
	}
	
	return result
}

func formatStatusAsJSON(status *SystemStatus) (string, error) {
	// Implementation would use json.Marshal
	return "{}", nil // Placeholder
}

func formatStatusAsYAML(status *SystemStatus) (string, error) {
	// Implementation would use yaml.Marshal
	return "---", nil // Placeholder
}
