package ui

import (
	"context"

	"github.com/osdo/osdo-infra-cli/pkg/config"
)

// ComponentSelector defines the interface for component selection UI
type ComponentSelector interface {
	SelectComponents(available []config.ComponentConfig) ([]config.ComponentConfig, error)
	SelectPreset(presets []config.PresetConfig) (*config.PresetConfig, error)
	SelectPlatform(platforms []config.Platform) (config.Platform, error)
	GetDeploymentOptions(platform config.Platform) (*config.DeploymentOptions, error)
}

// ProgressReporter defines the interface for progress reporting
type ProgressReporter interface {
	Start(total int)
	Update(current int, message string)
	Finish(message string)
	Error(err error)
}

// InteractivePrompter defines the interface for interactive prompts
type InteractivePrompter interface {
	Confirm(message string) (bool, error)
	Input(prompt string, defaultValue string) (string, error)
	Select(prompt string, options []string) (string, error)
	MultiSelect(prompt string, options []string) ([]string, error)
}

// MonitoringUI defines the interface for monitoring display
type MonitoringUI interface {
	DisplayStatus(status map[string]interface{}) error
	DisplayMetrics(metrics map[string]interface{}) error
	StartWatch(ctx context.Context, interval int) error
	ShowAlerts(alerts []interface{}) error
}

// MobileUI defines the interface for mobile-compatible UI
type MobileUI interface {
	ShowMainMenu() error
	ShowDeploymentWizard() error
	ShowMonitoringDashboard() error
	ShowSettings() error
}

// UIMode represents the UI mode
type UIMode string

const (
	UIModeCLI    UIMode = "cli"
	UIModeGUI    UIMode = "gui"
	UIModeMobile UIMode = "mobile"
)

// UIConfig holds UI configuration
type UIConfig struct {
	Mode     UIMode
	NoPrompt bool
	Verbose  bool
	Theme    string
}

// UIManager manages different UI implementations
type UIManager interface {
	GetSelector() ComponentSelector
	GetProgress() ProgressReporter
	GetPrompter() InteractivePrompter
	GetMonitoring() MonitoringUI
	GetMobile() MobileUI
	SetMode(mode UIMode)
}
