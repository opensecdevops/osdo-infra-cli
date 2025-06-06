package deployer

import (
	"context"
	"time"

	"github.com/osdo/osdo-infra-cli/pkg/config"
)

// DeploymentResult represents the result of a deployment operation
type DeploymentResult struct {
	Success       bool                   `json:"success"`
	Components    []ComponentResult      `json:"components"`
	Platform      config.Platform        `json:"platform"`
	Namespace     string                 `json:"namespace,omitempty"`
	Duration      time.Duration          `json:"duration"`
	AccessInfo    map[string]AccessInfo  `json:"access_info,omitempty"`
	Errors        []string               `json:"errors,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ComponentResult represents the deployment result for a single component
type ComponentResult struct {
	Name      string            `json:"name"`
	Success   bool              `json:"success"`
	Status    string            `json:"status"`
	Resources []ResourceInfo    `json:"resources,omitempty"`
	Endpoints []string          `json:"endpoints,omitempty"`
	Error     string            `json:"error,omitempty"`
	Duration  time.Duration     `json:"duration"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ResourceInfo represents information about a deployed resource
type ResourceInfo struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Status    string `json:"status"`
	Ready     bool   `json:"ready"`
}

// AccessInfo contains information about how to access a deployed component
type AccessInfo struct {
	URL         string            `json:"url,omitempty"`
	Ports       []int             `json:"ports,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
	Instructions string           `json:"instructions,omitempty"`
}

// DeploymentOptions contains options for deployment
type DeploymentOptions struct {
	DryRun          bool              `json:"dry_run"`
	Force           bool              `json:"force"`
	Timeout         time.Duration     `json:"timeout"`
	Values          map[string]string `json:"values,omitempty"`
	Namespace       string            `json:"namespace,omitempty"`
	CreateNamespace bool              `json:"create_namespace"`
	Wait            bool              `json:"wait"`
	Verbose         bool              `json:"verbose"`
}

// Deployer interface defines the contract for platform deployers
type Deployer interface {
	// Deploy deploys the specified components to the target platform
	Deploy(ctx context.Context, components []string, opts DeploymentOptions) (*DeploymentResult, error)
	
	// Undeploy removes the specified components from the target platform
	Undeploy(ctx context.Context, components []string, opts DeploymentOptions) error
	
	// GetStatus returns the status of deployed components
	GetStatus(ctx context.Context, components []string) (map[string]ComponentStatus, error)
	
	// ValidateComponents validates that components can be deployed on this platform
	ValidateComponents(components []string) error
	
	// GetPlatform returns the platform type this deployer handles
	GetPlatform() config.Platform
	
	// IsReady checks if the platform is ready for deployment
	IsReady(ctx context.Context) error
}

// ComponentStatus represents the current status of a deployed component
type ComponentStatus struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Ready     bool                   `json:"ready"`
	Replicas  ReplicaStatus          `json:"replicas,omitempty"`
	Resources []ResourceInfo         `json:"resources,omitempty"`
	Health    HealthStatus           `json:"health,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ReplicaStatus represents replica information
type ReplicaStatus struct {
	Desired   int32 `json:"desired"`
	Ready     int32 `json:"ready"`
	Available int32 `json:"available"`
	Updated   int32 `json:"updated"`
}

// HealthStatus represents health check information
type HealthStatus struct {
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	LastCheck  time.Time `json:"last_check"`
	Checks     []string  `json:"checks,omitempty"`
}

// DeploymentPlan represents a planned deployment
type DeploymentPlan struct {
	Components []PlannedComponent `json:"components"`
	Platform   config.Platform    `json:"platform"`
	Namespace  string             `json:"namespace,omitempty"`
	Resources  ResourceRequirements `json:"resources"`
	Dependencies []Dependency      `json:"dependencies"`
	Conflicts  []string           `json:"conflicts,omitempty"`
}

// PlannedComponent represents a component in a deployment plan
type PlannedComponent struct {
	Name         string               `json:"name"`
	Version      string               `json:"version,omitempty"`
	Resources    ResourceRequirements `json:"resources"`
	Dependencies []string             `json:"dependencies,omitempty"`
	Conflicts    []string             `json:"conflicts,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// ResourceRequirements represents resource requirements
type ResourceRequirements struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	Storage string `json:"storage,omitempty"`
}

// Dependency represents a component dependency
type Dependency struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Required bool   `json:"required"`
}
