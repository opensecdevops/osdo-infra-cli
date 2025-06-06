package monitor

import (
	"context"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/deployer"
)

// SystemStatus represents the overall system status
type SystemStatus struct {
	Timestamp     time.Time                              `json:"timestamp"`
	Platform      config.Platform                       `json:"platform"`
	Namespace     string                                `json:"namespace,omitempty"`
	OverallHealth string                                `json:"overall_health"`
	Components    map[string]deployer.ComponentStatus   `json:"components"`
	Resources     ResourceUsage                         `json:"resources"`
	Alerts        []Alert                               `json:"alerts,omitempty"`
	Metadata      map[string]interface{}                `json:"metadata,omitempty"`
}

// ResourceUsage represents resource usage information
type ResourceUsage struct {
	CPU     ResourceMetric `json:"cpu"`
	Memory  ResourceMetric `json:"memory"`
	Storage ResourceMetric `json:"storage"`
	Network NetworkMetric  `json:"network"`
}

// ResourceMetric represents a resource usage metric
type ResourceMetric struct {
	Used      string  `json:"used"`
	Total     string  `json:"total"`
	Percentage float64 `json:"percentage"`
	Unit      string  `json:"unit"`
}

// NetworkMetric represents network usage metrics
type NetworkMetric struct {
	BytesIn  string `json:"bytes_in"`
	BytesOut string `json:"bytes_out"`
	PacketsIn int64 `json:"packets_in"`
	PacketsOut int64 `json:"packets_out"`
}

// Alert represents a system alert
type Alert struct {
	ID          string                 `json:"id"`
	Severity    AlertSeverity          `json:"severity"`
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      AlertStatus            `json:"status"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Success   bool              `json:"success"`
	Message   string            `json:"message,omitempty"`
	Duration  time.Duration     `json:"duration"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// MonitoringOptions contains options for monitoring operations
type MonitoringOptions struct {
	Platform    config.Platform `json:"platform"`
	Namespace   string          `json:"namespace,omitempty"`
	Components  []string        `json:"components,omitempty"`
	WatchMode   bool            `json:"watch_mode"`
	Interval    time.Duration   `json:"interval"`
	Timeout     time.Duration   `json:"timeout"`
	HealthCheck bool            `json:"health_check"`
	Verbose     bool            `json:"verbose"`
}

// Monitor interface defines the contract for system monitoring
type Monitor interface {
	// GetSystemStatus returns the current system status
	GetSystemStatus(ctx context.Context, opts MonitoringOptions) (*SystemStatus, error)
	
	// GetComponentStatus returns the status of specific components
	GetComponentStatus(ctx context.Context, components []string, opts MonitoringOptions) (map[string]deployer.ComponentStatus, error)
	
	// GetResourceUsage returns resource usage information
	GetResourceUsage(ctx context.Context, opts MonitoringOptions) (*ResourceUsage, error)
	
	// GetAlerts returns active alerts
	GetAlerts(ctx context.Context, opts MonitoringOptions) ([]Alert, error)
	
	// RunHealthChecks performs health checks on components
	RunHealthChecks(ctx context.Context, components []string, opts MonitoringOptions) (map[string]HealthCheck, error)
	
	// StartWatching starts monitoring in watch mode
	StartWatching(ctx context.Context, opts MonitoringOptions) (<-chan *SystemStatus, error)
	
	// GetPlatform returns the platform this monitor supports
	GetPlatform() config.Platform
	
	// IsReady checks if monitoring is ready
	IsReady(ctx context.Context) error
}

// WatchEvent represents an event in watch mode
type WatchEvent struct {
	Type      WatchEventType `json:"type"`
	Component string         `json:"component,omitempty"`
	Status    *SystemStatus  `json:"status,omitempty"`
	Alert     *Alert         `json:"alert,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// WatchEventType represents the type of watch event
type WatchEventType string

const (
	WatchEventTypeStatusUpdate WatchEventType = "status_update"
	WatchEventTypeAlert        WatchEventType = "alert"
	WatchEventTypeHealthCheck  WatchEventType = "health_check"
)

// MetricsCollector interface for collecting metrics
type MetricsCollector interface {
	// CollectMetrics collects metrics from the system
	CollectMetrics(ctx context.Context, opts MonitoringOptions) (map[string]interface{}, error)
	
	// GetMetricHistory returns historical metric data
	GetMetricHistory(ctx context.Context, metric string, duration time.Duration) ([]MetricPoint, error)
	
	// GetSupportedMetrics returns list of supported metrics
	GetSupportedMetrics() []string
}

// MetricPoint represents a single metric point in time
type MetricPoint struct {
	Timestamp time.Time   `json:"timestamp"`
	Value     interface{} `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// AlertManager interface for managing alerts
type AlertManager interface {
	// GetAlerts returns current alerts
	GetAlerts(ctx context.Context, filters map[string]string) ([]Alert, error)
	
	// SilenceAlert silences an alert
	SilenceAlert(ctx context.Context, alertID string, duration time.Duration) error
	
	// ResolveAlert marks an alert as resolved
	ResolveAlert(ctx context.Context, alertID string) error
	
	// CreateAlert creates a new alert
	CreateAlert(ctx context.Context, alert Alert) error
}

// LogCollector interface for collecting logs
type LogCollector interface {
	// GetLogs returns logs for components
	GetLogs(ctx context.Context, components []string, opts LogOptions) (map[string][]LogEntry, error)
	
	// StreamLogs streams logs in real-time
	StreamLogs(ctx context.Context, components []string, opts LogOptions) (<-chan LogEntry, error)
}

// LogOptions contains options for log collection
type LogOptions struct {
	Since      time.Time `json:"since,omitempty"`
	Until      time.Time `json:"until,omitempty"`
	Lines      int       `json:"lines,omitempty"`
	Follow     bool      `json:"follow"`
	Timestamps bool      `json:"timestamps"`
	Level      string    `json:"level,omitempty"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Component string            `json:"component"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
