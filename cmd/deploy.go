package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	
	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/deployer"
	"github.com/osdo/osdo-infra-cli/pkg/logger"
	"github.com/osdo/osdo-infra-cli/pkg/ui"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Despliega componentes DevSecOps en la plataforma especificada",
	Long: `Despliega componentes seleccionados de DevSecOps en la plataforma de destino.
	
Permite seleccionar componentes interactivamente o mediante flags, validar configuraciones
y desplegar en diferentes plataformas (Kubernetes, K3s, Docker Swarm, Docker Compose).

Ejemplos:
  osdo deploy --platform kubernetes --components gitlab,vault,sonarqube
  osdo deploy --interactive --platform k3s
  osdo deploy --preset development --platform docker
  osdo deploy --config myconfig.yaml`,
	RunE: runDeploy,
}

var (
	deployPlatform    string
	deployComponents  []string
	deployPreset      string
	deployInteractive bool
	deployDryRun      bool
	deployForce       bool
	deployNamespace   string
	deployDomain      string
	deployRegistry    string
	deployNoMonitoring bool
	deployNoSecurity   bool
)

func init() {
	rootCmd.AddCommand(deployCmd)

	// Flags principales
	deployCmd.Flags().StringVarP(&deployPlatform, "platform", "p", "", 
		"Plataforma de despliegue (kubernetes, k3s, docker, swarm, helm)")
	deployCmd.Flags().StringSliceVarP(&deployComponents, "components", "c", []string{}, 
		"Lista de componentes a desplegar (separados por comas)")
	deployCmd.Flags().StringVar(&deployPreset, "preset", "", 
		"Preset predefinido para desplegar")
	deployCmd.Flags().BoolVarP(&deployInteractive, "interactive", "i", false, 
		"Modo interactivo para seleccionar componentes")
	
	// Flags de control
	deployCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, 
		"Simular el despliegue sin realizar cambios")
	deployCmd.Flags().BoolVar(&deployForce, "force", false, 
		"Forzar el despliegue incluso si hay advertencias")
	
	// Flags de configuración
	deployCmd.Flags().StringVarP(&deployNamespace, "namespace", "n", "", 
		"Namespace/contexto para el despliegue")
	deployCmd.Flags().StringVarP(&deployDomain, "domain", "d", "", 
		"Dominio base para los servicios")
	deployCmd.Flags().StringVarP(&deployRegistry, "registry", "r", "", 
		"Registry de contenedores a usar")
	
	// Flags de características
	deployCmd.Flags().BoolVar(&deployNoMonitoring, "no-monitoring", false, 
		"Deshabilitar el stack de monitoreo")
	deployCmd.Flags().BoolVar(&deployNoSecurity, "no-security", false, 
		"Deshabilitar las herramientas de seguridad adicionales")

	// Marcar flags mutuamente exclusivos
	deployCmd.MarkFlagsMutuallyExclusive("components", "preset", "interactive")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	logger.Info("Iniciando proceso de despliegue OSDO")

	// Cargar configuración
	configManager := config.NewManager()
	cliConfig, err := configManager.LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("error cargando configuración: %w", err)
	}

	componentConfig, err := configManager.LoadComponentConfig()
	if err != nil {
		return fmt.Errorf("error cargando configuración de componentes: %w", err)
	}

	// Determinar plataforma
	platform := deployPlatform
	if platform == "" {
		platform = cliConfig.DefaultPlatform
	}

	// Validar plataforma
	if _, exists := componentConfig.DeploymentPlatforms[platform]; !exists {
		return fmt.Errorf("plataforma no soportada: %s", platform)
	}

	logger.Info("Plataforma seleccionada", "platform", platform)

	// Determinar componentes a desplegar
	var selectedComponents []string
	
	if deployPreset != "" {
		selectedComponents, err = getComponentsFromPreset(cliConfig, deployPreset)
		if err != nil {
			return fmt.Errorf("error obteniendo componentes del preset: %w", err)
		}
	} else if deployInteractive {
		selectedComponents, err = selectComponentsInteractively(configManager)
		if err != nil {
			return fmt.Errorf("error en selección interactiva: %w", err)
		}
	} else if len(deployComponents) > 0 {
		selectedComponents = deployComponents
	} else {
		return fmt.Errorf("debe especificar componentes mediante --components, --preset o --interactive")
	}

	logger.Info("Componentes seleccionados", "components", selectedComponents)

	// Validar componentes
	validation := configManager.ValidateComponents(selectedComponents)
	if !validation.Valid && !deployForce {
		logger.Error("Errores de validación encontrados:")
		for _, err := range validation.Errors {
			logger.Error("  - " + err)
		}
		return fmt.Errorf("validación fallida (use --force para ignorar)")
	}

	if len(validation.Warnings) > 0 {
		logger.Warning("Advertencias de validación:")
		for _, warn := range validation.Warnings {
			logger.Warning("  - " + warn)
		}
		if !deployForce {
			// Crear UI manager para prompts
			uiConfig := &ui.UIConfig{
				Mode:     ui.UIModeCLI,
				NoPrompt: false,
				Verbose:  false,
			}
			uiManager, err := ui.NewUIManager(uiConfig, &config.Config{})
			if err != nil {
				return fmt.Errorf("error creando UI manager: %w", err)
			}
			
			confirm, err := uiManager.GetPrompter().Confirm("¿Continuar con el despliegue?")
			if err != nil {
				return fmt.Errorf("error en prompt: %w", err)
			}
			if !confirm {
				return fmt.Errorf("despliegue cancelado por el usuario")
			}
		}
	}

	// Crear opciones de despliegue
	deploymentOpts := deployer.DeploymentOptions{
		DryRun:          deployDryRun,
		Force:           deployForce,
		Timeout:         30 * time.Minute, // timeout por defecto
		Namespace:       getNamespace(),
		CreateNamespace: true,
		Wait:            true,
		Verbose:         false,
		Values: map[string]string{
			"domain":   getDomain(cliConfig),
			"registry": getRegistry(cliConfig),
		},
	}

	// Convertir plataforma a tipo correcto
	var platformType config.Platform
	switch platform {
	case "kubernetes":
		platformType = config.PlatformKubernetes
	case "k3s":
		platformType = config.PlatformK3s
	case "docker":
		platformType = config.PlatformDockerCompose
	case "swarm":
		platformType = config.PlatformDockerSwarm
	case "helm":
		platformType = config.PlatformHelm
	default:
		return fmt.Errorf("plataforma no soportada: %s", platform)
	}

	// Ejecutar despliegue
	if deployDryRun {
		logger.Info("🧪 Ejecutando simulación de despliegue (dry-run)")
	} else {
		logger.Info("🚀 Ejecutando despliegue real")
	}

	ctx := context.Background()
	result, err := deployer.DeployWithPlan(ctx, &config.Config{}, selectedComponents, platformType, deploymentOpts)
	if err != nil {
		return fmt.Errorf("error durante el despliegue: %w", err)
	}

	// Mostrar resultados
	displayDeploymentResult(result)

	if !deployDryRun && result.Success {
		logger.Success("✅ Despliegue completado exitosamente")
		
		// Mostrar información de acceso
		if err := displayAccessInfo(result); err != nil {
			logger.Warning("Error mostrando información de acceso", "error", err)
		}
		
		// Sugerir próximos pasos
		suggestNextSteps(platform, selectedComponents)
	}

	return nil
}

func getComponentsFromPreset(cliConfig *config.CLIConfig, presetName string) ([]string, error) {
	for _, preset := range cliConfig.Presets {
		if preset.Name == presetName {
			return preset.Components, nil
		}
	}
	return nil, fmt.Errorf("preset '%s' no encontrado", presetName)
}

func selectComponentsInteractively(configManager *config.Manager) ([]string, error) {
	categories, err := configManager.GetCategories()
	if err != nil {
		return nil, err
	}

	// Crear UI manager para selección interactiva
	uiConfig := &ui.UIConfig{
		Mode:     ui.UIModeCLI,
		NoPrompt: false,
		Verbose:  false,
	}
	uiManager, err := ui.NewUIManager(uiConfig, &config.Config{})
	if err != nil {
		return nil, fmt.Errorf("error creando UI manager: %w", err)
	}

	// Convertir categorías a componentes individuales
	var allComponents []config.ComponentConfig
	for _, category := range categories {
		for _, component := range category.Components {
			allComponents = append(allComponents, component)
		}
	}

	selector := uiManager.GetSelector()
	selectedConfigs, err := selector.SelectComponents(allComponents)
	if err != nil {
		return nil, err
	}

	// Extraer nombres de componentes de las configuraciones
	var componentNames []string
	for _, cfg := range selectedConfigs {
		componentNames = append(componentNames, cfg.Name)
	}

	return componentNames, nil
}

func getNamespace() string {
	if deployNamespace != "" {
		return deployNamespace
	}
	return "osdo" // namespace por defecto
}

func getDomain(cliConfig *config.CLIConfig) string {
	if deployDomain != "" {
		return deployDomain
	}
	return cliConfig.DefaultDomain
}

func getRegistry(cliConfig *config.CLIConfig) string {
	if deployRegistry != "" {
		return deployRegistry
	}
	return cliConfig.Registry.URL
}

func displayDeploymentResult(result *deployer.DeploymentResult) {
	logger.Info("📊 Resumen del Despliegue")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	logger.Info("Plataforma", "platform", result.Platform)
	logger.Info("Namespace", "namespace", result.Namespace)
	logger.Info("Componentes", "total", len(result.Components))
	
	successful := 0
	failed := 0
	
	for _, compResult := range result.Components {
		if compResult.Success {
			successful++
			logger.Success(fmt.Sprintf("✅ %s - %s", compResult.Name, compResult.Status))
		} else {
			failed++
			logger.Error(fmt.Sprintf("❌ %s - %s", compResult.Name, compResult.Status))
			if compResult.Error != "" {
				logger.Error(fmt.Sprintf("   Error: %s", compResult.Error))
			}
		}
	}
	
	logger.Info("Exitosos", "count", successful)
	if failed > 0 {
		logger.Info("Fallidos", "count", failed)
	}
	
	logger.Info("Duración", "duration", result.Duration.String())
}

func displayAccessInfo(result *deployer.DeploymentResult) error {
	if len(result.AccessInfo) == 0 {
		return nil
	}

	logger.Info("🌐 Información de Acceso")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	for serviceName, accessInfo := range result.AccessInfo {
		if accessInfo.URL != "" {
			logger.Info(fmt.Sprintf("%-15s: %s", serviceName, accessInfo.URL))
		}
		if len(accessInfo.Ports) > 0 {
			logger.Info(fmt.Sprintf("%-15s Puertos: %v", serviceName, accessInfo.Ports))
		}
		if accessInfo.Instructions != "" {
			logger.Info(fmt.Sprintf("%-15s Instrucciones: %s", serviceName, accessInfo.Instructions))
		}
	}
	
	return nil
}

func suggestNextSteps(platform string, components []string) {
	logger.Info("📋 Próximos Pasos Recomendados")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	suggestions := []string{
		"Verificar el estado de los servicios: osdo status",
		"Configurar alertas de monitoreo: osdo monitoring setup",
		"Ejecutar auditoría de seguridad: osdo security audit",
	}
	
	// Sugerencias específicas por componente
	if contains(components, "vault") {
		suggestions = append(suggestions, "Configurar Vault: osdo vault init")
	}
	
	if contains(components, "gitlab") {
		suggestions = append(suggestions, "Configurar GitLab CI/CD: osdo gitlab setup")
	}
	
	if platform == "kubernetes" || platform == "k3s" {
		suggestions = append(suggestions, "Configurar ingress: osdo ingress setup")
		suggestions = append(suggestions, "Configurar certificados SSL: osdo ssl setup")
	}
	
	for i, suggestion := range suggestions {
		logger.Info(fmt.Sprintf("%d. %s", i+1, suggestion))
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
