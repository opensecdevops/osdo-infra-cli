package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultOSDOConfigDir  = ".osdo"
	DefaultConfigFileName = "config.yaml"
)

// Manager handles configuration management for the OSDO CLI
type Manager struct {
	configDir  string
	configFile string
	config     *CLIConfig
	verbose    bool
	dryRun     bool
	output     string
	kubeconfig string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, DefaultOSDOConfigDir)
	configFile := filepath.Join(configDir, DefaultConfigFileName)

	return &Manager{
		configDir:  configDir,
		configFile: configFile,
		config:     &CLIConfig{},
		verbose:    false,
		dryRun:     false,
		output:     "text",
	}
}

// Init initializes the configuration manager
func (m *Manager) Init() error {
	if err := m.ensureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := m.Load(); err != nil {
		// If config doesn't exist, create a default one
		if os.IsNotExist(err) {
			m.config = &CLIConfig{
				ConfigVersion:   "1.0",
				DefaultPlatform: "kubernetes",
				DefaultDomain:   "osdo.local",
				Kubeconfig:      "",
			}
			return m.Save()
		}
		return err
	}

	return nil
}

// Load loads the configuration from file
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, m.config)
}

// Save saves the configuration to file
func (m *Manager) Save() error {
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return err
	}

	return os.WriteFile(m.configFile, data, 0644)
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *CLIConfig {
	return m.config
}

// SetVerbose sets the verbose mode
func (m *Manager) SetVerbose(verbose bool) {
	m.verbose = verbose
}

// IsVerbose returns if verbose mode is enabled
func (m *Manager) IsVerbose() bool {
	return m.verbose
}

// SetDryRun sets the dry run mode
func (m *Manager) SetDryRun(dryRun bool) {
	m.dryRun = dryRun
}

// IsDryRun returns if dry run mode is enabled
func (m *Manager) IsDryRun() bool {
	return m.dryRun
}

// SetOutput sets the output format
func (m *Manager) SetOutput(output string) {
	m.output = output
}

// GetOutput returns the output format
func (m *Manager) GetOutput() string {
	return m.output
}

// SetKubeconfig sets the kubeconfig path
func (m *Manager) SetKubeconfig(kubeconfig string) {
	m.kubeconfig = kubeconfig
	m.config.Kubeconfig = kubeconfig
}

// GetKubeconfig returns the kubeconfig path
func (m *Manager) GetKubeconfig() string {
	if m.kubeconfig != "" {
		return m.kubeconfig
	}
	return m.config.Kubeconfig
}

// ensureConfigDir ensures the configuration directory exists
func (m *Manager) ensureConfigDir() error {
	if _, err := os.Stat(m.configDir); os.IsNotExist(err) {
		return os.MkdirAll(m.configDir, 0755)
	}
	return nil
}

// GetConfigDir returns the configuration directory path
func (m *Manager) GetConfigDir() string {
	return m.configDir
}

// GetConfigFile returns the configuration file path
func (m *Manager) GetConfigFile() string {
	return m.configFile
}

// SetPlatform sets the deployment platform
func (m *Manager) SetPlatform(platform string) {
	m.config.DefaultPlatform = platform
}

// GetPlatform returns the deployment platform
func (m *Manager) GetPlatform() string {
	return m.config.DefaultPlatform
}

// SetDomain sets the default domain
func (m *Manager) SetDomain(domain string) {
	m.config.DefaultDomain = domain
}

// GetDomain returns the default domain
func (m *Manager) GetDomain() string {
	return m.config.DefaultDomain
}

// LoadCLIConfig loads and returns the CLI configuration
func (m *Manager) LoadCLIConfig() (*CLIConfig, error) {
	if err := m.Load(); err != nil {
		return nil, fmt.Errorf("failed to load CLI config: %w", err)
	}
	return m.config, nil
}

// LoadComponentConfig loads and returns the component configuration
func (m *Manager) LoadComponentConfig() (*ComponentSelectorConfig, error) {
	// For now, return a basic component configuration
	// In a real implementation, this would load from a separate component config file
	return &ComponentSelectorConfig{
		Version: "1.0",
		Name:    "OSDO Component Configuration",
		DeploymentPlatforms: map[string]DeploymentPlatform{
			"kubernetes": {
				Name:            "Kubernetes",
				Description:     "Kubernetes cluster",
				SupportsScaling: true,
				RecommendedFor:  []string{"production", "staging"},
			},
			"docker": {
				Name:            "Docker Compose",
				Description:     "Docker Compose",
				SupportsScaling: false,
				RecommendedFor:  []string{"development", "testing"},
			},
			"docker-swarm": {
				Name:            "Docker Swarm",
				Description:     "Docker Swarm cluster",
				SupportsScaling: true,
				RecommendedFor:  []string{"production"},
			},
		},
		Categories: make(map[string]Category),
	}, nil
}

// ValidateComponents validates component selection
func (m *Manager) ValidateComponents(components []string) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	if len(components) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "No components selected")
		return result
	}

	// Basic validation - in a real implementation this would be more comprehensive
	supportedComponents := []string{"prometheus", "grafana", "jaeger", "vault", "sonarqube", "jenkins", "gitlab", "portainer", "traefik"}
	
	for _, component := range components {
		found := false
		for _, supported := range supportedComponents {
			if component == supported {
				found = true
				break
			}
		}
		if !found {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Component '%s' may not be fully supported", component))
		}
	}

	return result
}

// GetCategories returns component categories
func (m *Manager) GetCategories() ([]Category, error) {
	// Return basic categories - in a real implementation this would load from config
	return []Category{
		{
			Name:        "Monitoring",
			Description: "System monitoring and observability tools",
			Icon:        "📊",
		},
		{
			Name:        "Security",
			Description: "Security and compliance tools",
			Icon:        "🔒",
		},
		{
			Name:        "CI/CD",
			Description: "Continuous Integration and Deployment",
			Icon:        "🚀",
		},
	}, nil
}

// ValidationResult is already defined in types.go
