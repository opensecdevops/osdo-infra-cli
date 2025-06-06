package config

import "time"

// Platform representa un tipo de plataforma de despliegue
type Platform string

const (
	PlatformKubernetes    Platform = "kubernetes"
	PlatformK3s          Platform = "k3s"
	PlatformDockerSwarm  Platform = "docker-swarm"
	PlatformDockerCompose Platform = "docker-compose"
	PlatformHelm         Platform = "helm"
)

// Config es un alias para ComponentSelectorConfig para mantener compatibilidad
type Config = ComponentSelectorConfig

// ComponentConfig es un alias para Component
type ComponentConfig = Component

// DeploymentOptions representa las opciones de despliegue
type DeploymentOptions struct {
	Platform      Platform          `json:"platform"`
	Components    []string          `json:"components"`
	Values        map[string]string `json:"values"`
	DryRun        bool              `json:"dry_run"`
	Namespace     string            `json:"namespace"`
	ConfigPath    string            `json:"config_path"`
	Interactive   bool              `json:"interactive"`
	Config        *ComponentSelectorConfig `json:"config"`
	Timeout       time.Duration     `json:"timeout"`
}

// ComponentSelectorConfig representa la configuración principal del selector de componentes
type ComponentSelectorConfig struct {
	Version           string                         `yaml:"version"`
	Name              string                         `yaml:"name"`
	MutualExclusions  map[string][]string           `yaml:"mutual_exclusions"`
	DeploymentPlatforms map[string]DeploymentPlatform `yaml:"deployment_platforms"`
	Categories        map[string]Category           `yaml:"categories"`
}

// DeploymentPlatform representa una plataforma de despliegue
type DeploymentPlatform struct {
	Name             string   `yaml:"name"`
	Description      string   `yaml:"description"`
	Icon             string   `yaml:"icon"`
	SupportsScaling  bool     `yaml:"supports_scaling"`
	RecommendedFor   []string `yaml:"recommended_for"`
	TemplatesDir     string   `yaml:"templates_dir"`
}

// Category representa una categoría de componentes
type Category struct {
	Name            string                `yaml:"name"`
	Description     string                `yaml:"description"`
	Icon            string                `yaml:"icon"`
	MutualExclusive bool                  `yaml:"mutual_exclusive"`
	Components      map[string]Component  `yaml:"components"`
}

// Component representa un componente individual
type Component struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	Enabled       bool              `yaml:"enabled"`
	Ports         []string          `yaml:"ports"`
	Resources     Resources         `yaml:"resources"`
	Dependencies  []string          `yaml:"dependencies"`
	HealthCheck   string            `yaml:"health_check"`
	DockerImage   string            `yaml:"docker_image"`
	RequiresToken bool              `yaml:"requires_token"`
	Tags          []string          `yaml:"tags"`
	Environment   map[string]string `yaml:"environment"`
}

// Resources representa los recursos requeridos por un componente
type Resources struct {
	CPU     string `yaml:"cpu"`
	Memory  string `yaml:"memory"`
	Storage string `yaml:"storage"`
}

// DeploymentTarget representa el objetivo de despliegue
type DeploymentTarget struct {
	Platform    string            `yaml:"platform"`
	Environment string            `yaml:"environment"`
	Domain      string            `yaml:"domain"`
	Registry    string            `yaml:"registry"`
	Namespace   string            `yaml:"namespace"`
	Config      map[string]string `yaml:"config"`
}

// CLIConfig representa la configuración del CLI
type CLIConfig struct {
	ConfigVersion   string            `yaml:"config_version"`
	DefaultPlatform string            `yaml:"default_platform"`
	DefaultDomain   string            `yaml:"default_domain"`
	Kubeconfig      string            `yaml:"kubeconfig"`
	Registry        RegistryConfig    `yaml:"registry"`
	Monitoring      MonitoringConfig  `yaml:"monitoring"`
	Security        SecurityConfig    `yaml:"security"`
	Presets         []PresetConfig    `yaml:"presets"`
	Templates       TemplatesConfig   `yaml:"templates"`
}

// RegistryConfig configuración del registry
type RegistryConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Insecure bool   `yaml:"insecure"`
}

// MonitoringConfig configuración de monitoreo
type MonitoringConfig struct {
	Enabled         bool              `yaml:"enabled"`
	Namespace       string            `yaml:"namespace"`
	Retention       string            `yaml:"retention"`
	StorageSize     string            `yaml:"storage_size"`
	AlertManager    AlertManagerConfig `yaml:"alertmanager"`
	Grafana         GrafanaConfig     `yaml:"grafana"`
	Prometheus      PrometheusConfig  `yaml:"prometheus"`
}

// AlertManagerConfig configuración de AlertManager
type AlertManagerConfig struct {
	Enabled     bool              `yaml:"enabled"`
	SlackURL    string            `yaml:"slack_url"`
	EmailConfig EmailConfig       `yaml:"email"`
	Webhooks    []WebhookConfig   `yaml:"webhooks"`
}

// EmailConfig configuración de email
type EmailConfig struct {
	SMTPHost     string   `yaml:"smtp_host"`
	SMTPPort     int      `yaml:"smtp_port"`
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
	FromAddress  string   `yaml:"from_address"`
	ToAddresses  []string `yaml:"to_addresses"`
}

// WebhookConfig configuración de webhook
type WebhookConfig struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
}

// GrafanaConfig configuración de Grafana
type GrafanaConfig struct {
	AdminPassword string            `yaml:"admin_password"`
	Datasources   []DataSourceConfig `yaml:"datasources"`
	Dashboards    []DashboardConfig `yaml:"dashboards"`
}

// DataSourceConfig configuración de fuente de datos
type DataSourceConfig struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

// DashboardConfig configuración de dashboard
type DashboardConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// PrometheusConfig configuración de Prometheus
type PrometheusConfig struct {
	StorageSize     string            `yaml:"storage_size"`
	RetentionPeriod string            `yaml:"retention_period"`
	ScrapeInterval  string            `yaml:"scrape_interval"`
	Rules           []PrometheusRule  `yaml:"rules"`
}

// PrometheusRule regla de Prometheus
type PrometheusRule struct {
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
	Alert string `yaml:"alert"`
}

// SecurityConfig configuración de seguridad
type SecurityConfig struct {
	PolicyEngine    PolicyEngineConfig    `yaml:"policy_engine"`
	VulnScanning    VulnScanningConfig    `yaml:"vuln_scanning"`
	RuntimeSecurity RuntimeSecurityConfig `yaml:"runtime_security"`
}

// PolicyEngineConfig configuración del motor de políticas
type PolicyEngineConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Engine    string   `yaml:"engine"` // kyverno, opa, falco
	Policies  []string `yaml:"policies"`
	Enforce   bool     `yaml:"enforce"`
}

// VulnScanningConfig configuración de escaneo de vulnerabilidades
type VulnScanningConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Tools     []string `yaml:"tools"` // trivy, grype, snyk
	Schedule  string   `yaml:"schedule"`
	Threshold string   `yaml:"threshold"` // low, medium, high, critical
}

// RuntimeSecurityConfig configuración de seguridad en tiempo de ejecución
type RuntimeSecurityConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Tools        []string `yaml:"tools"` // falco, sysdig
	AlertWebhook string   `yaml:"alert_webhook"`
}

// PresetConfig configuración de preset
type PresetConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Platform    string   `yaml:"platform"`
	Components  []string `yaml:"components"`
	Config      map[string]interface{} `yaml:"config"`
}

// TemplatesConfig configuración de plantillas
type TemplatesConfig struct {
	DefaultPath string                 `yaml:"default_path"`
	Custom      map[string]TemplateRef `yaml:"custom"`
}

// TemplateRef referencia a plantilla
type TemplateRef struct {
	Path        string `yaml:"path"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// DeploymentStatus estado del despliegue
type DeploymentStatus struct {
	Platform     string            `json:"platform"`
	Environment  string            `json:"environment"`
	Components   []ComponentStatus `json:"components"`
	LastUpdate   time.Time         `json:"last_update"`
	HealthStatus string            `json:"health_status"`
}

// ComponentStatus estado de un componente
type ComponentStatus struct {
	Name        string            `json:"name"`
	Status      string            `json:"status"` // deployed, failed, pending, unknown
	Health      string            `json:"health"` // healthy, unhealthy, unknown
	LastCheck   time.Time         `json:"last_check"`
	Endpoints   []string          `json:"endpoints"`
	Resources   ResourceUsage     `json:"resources"`
	Errors      []string          `json:"errors"`
	Metadata    map[string]string `json:"metadata"`
}

// ResourceUsage uso de recursos
type ResourceUsage struct {
	CPUUsage    string `json:"cpu_usage"`
	MemoryUsage string `json:"memory_usage"`
	DiskUsage   string `json:"disk_usage"`
	NetworkIn   string `json:"network_in"`
	NetworkOut  string `json:"network_out"`
}

// ValidationResult resultado de validación
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// BackupConfig configuración de backup
type BackupConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Schedule    string            `yaml:"schedule"`
	Retention   string            `yaml:"retention"`
	Storage     BackupStorage     `yaml:"storage"`
	Components  []string          `yaml:"components"`
	Encryption  EncryptionConfig  `yaml:"encryption"`
}

// BackupStorage configuración de almacenamiento de backup
type BackupStorage struct {
	Type     string            `yaml:"type"` // s3, gcs, azure, local
	Config   map[string]string `yaml:"config"`
	Path     string            `yaml:"path"`
}

// EncryptionConfig configuración de encriptación
type EncryptionConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Algorithm string `yaml:"algorithm"`
	KeyID     string `yaml:"key_id"`
}
