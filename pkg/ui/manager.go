package ui

import (
	"fmt"

	"github.com/osdo/osdo-infra-cli/pkg/config"
)

// DefaultUIManager implements UIManager interface
type DefaultUIManager struct {
	config      *UIConfig
	selector    ComponentSelector
	progress    ProgressReporter
	prompter    InteractivePrompter
	monitoring  MonitoringUI
	mobile      MobileUI
}

// NewUIManager creates a new UI manager
func NewUIManager(cfg *UIConfig, appConfig *config.Config) (*DefaultUIManager, error) {
	if cfg == nil {
		cfg = &UIConfig{
			Mode:     UIModeCLI,
			NoPrompt: false,
			Verbose:  false,
			Theme:    "auto",
		}
	}

	manager := &DefaultUIManager{
		config: cfg,
	}

	// Initialize UI components based on mode
	if err := manager.initializeComponents(appConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize UI components: %w", err)
	}

	return manager, nil
}

// initializeComponents initializes UI components based on the current mode
func (m *DefaultUIManager) initializeComponents(appConfig *config.Config) error {
	switch m.config.Mode {
	case UIModeCLI:
		return m.initializeCLIComponents()
	case UIModeGUI, UIModeMobile:
		return m.initializeMobileComponents(appConfig)
	default:
		return fmt.Errorf("unsupported UI mode: %s", m.config.Mode)
	}
}

// initializeCLIComponents initializes CLI-based components
func (m *DefaultUIManager) initializeCLIComponents() error {
	// Initialize prompter
	if m.config.NoPrompt {
		m.prompter = NewNoPrompter()
	} else {
		m.prompter = NewCLIPrompter(m.config.NoPrompt)
	}

	// Initialize selector
	m.selector = NewCLISelector(m.prompter)

	// Initialize progress reporter
	if m.config.Verbose {
		m.progress = NewCLIProgress()
	} else {
		m.progress = NewSimpleProgress()
	}

	// Initialize monitoring UI
	outputFormat := "table"
	m.monitoring = NewCLIMonitoring(outputFormat, m.config.Verbose)

	// Mobile UI is not available in CLI mode
	m.mobile = nil

	return nil
}

// initializeMobileComponents initializes mobile/GUI components
func (m *DefaultUIManager) initializeMobileComponents(appConfig *config.Config) error {
	// Initialize mobile UI (will use NoOp version if GUI build tag not present)
	m.mobile = NewFyneMobileUI(appConfig)

	// For mobile mode, we still need CLI components as fallback
	m.prompter = NewNoPrompter() // Non-interactive for mobile
	m.selector = NewCLISelector(m.prompter)
	m.progress = NewSimpleProgress()
	m.monitoring = NewCLIMonitoring("json", false)

	return nil
}

// GetSelector implements UIManager interface
func (m *DefaultUIManager) GetSelector() ComponentSelector {
	return m.selector
}

// GetProgress implements UIManager interface
func (m *DefaultUIManager) GetProgress() ProgressReporter {
	return m.progress
}

// GetPrompter implements UIManager interface
func (m *DefaultUIManager) GetPrompter() InteractivePrompter {
	return m.prompter
}

// GetMonitoring implements UIManager interface
func (m *DefaultUIManager) GetMonitoring() MonitoringUI {
	return m.monitoring
}

// GetMobile implements UIManager interface
func (m *DefaultUIManager) GetMobile() MobileUI {
	return m.mobile
}

// SetMode implements UIManager interface
func (m *DefaultUIManager) SetMode(mode UIMode) {
	m.config.Mode = mode
}

// GetMode returns the current UI mode
func (m *DefaultUIManager) GetMode() UIMode {
	return m.config.Mode
}

// SetVerbose sets verbose mode
func (m *DefaultUIManager) SetVerbose(verbose bool) {
	m.config.Verbose = verbose
}

// IsVerbose returns whether verbose mode is enabled
func (m *DefaultUIManager) IsVerbose() bool {
	return m.config.Verbose
}

// SetNoPrompt sets no-prompt mode
func (m *DefaultUIManager) SetNoPrompt(noPrompt bool) {
	m.config.NoPrompt = noPrompt
}

// IsNoPrompt returns whether no-prompt mode is enabled
func (m *DefaultUIManager) IsNoPrompt() bool {
	return m.config.NoPrompt
}

// SetTheme sets the UI theme
func (m *DefaultUIManager) SetTheme(theme string) {
	m.config.Theme = theme
}

// GetTheme returns the current UI theme
func (m *DefaultUIManager) GetTheme() string {
	return m.config.Theme
}

// SetOutputFormat sets the output format for monitoring
func (m *DefaultUIManager) SetOutputFormat(format string) {
	if cliMonitoring, ok := m.monitoring.(*CLIMonitoring); ok {
		cliMonitoring.outputFormat = format
	}
}

// GetConfig returns the UI configuration
func (m *DefaultUIManager) GetConfig() *UIConfig {
	return m.config
}

// UpdateConfig updates the UI configuration
func (m *DefaultUIManager) UpdateConfig(cfg *UIConfig) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	m.config = cfg
	return nil
}

// IsMobileSupported returns whether mobile UI is supported
func (m *DefaultUIManager) IsMobileSupported() bool {
	return m.mobile != nil
}

// StartMobileApp starts the mobile application
func (m *DefaultUIManager) StartMobileApp() error {
	if m.mobile == nil {
		return fmt.Errorf("mobile UI is not available")
	}

	return m.mobile.ShowMainMenu()
}

// ValidateMode validates if the UI mode is supported
func ValidateMode(mode UIMode) error {
	switch mode {
	case UIModeCLI, UIModeGUI, UIModeMobile:
		return nil
	default:
		return fmt.Errorf("unsupported UI mode: %s", mode)
	}
}

// GetAvailableModes returns a list of available UI modes
func GetAvailableModes() []UIMode {
	return []UIMode{UIModeCLI, UIModeGUI, UIModeMobile}
}

// GetModeDescription returns a description for a UI mode
func GetModeDescription(mode UIMode) string {
	switch mode {
	case UIModeCLI:
		return "Interfaz de línea de comandos tradicional"
	case UIModeGUI:
		return "Interfaz gráfica de usuario de escritorio"
	case UIModeMobile:
		return "Interfaz optimizada para dispositivos móviles"
	default:
		return "Modo desconocido"
	}
}
