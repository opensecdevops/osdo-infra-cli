package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// presetCmd represents the preset command
var presetCmd = &cobra.Command{
	Use:   "preset",
	Short: "Gestionar presets de configuración",
	Long: `Gestiona presets de configuración predefinidos para diferentes escenarios de despliegue.

Los presets permiten configurar rápidamente conjuntos de componentes y opciones
para escenarios comunes como desarrollo, staging, producción, etc.

Ejemplos:
  osdo preset list                    # Listar presets disponibles
  osdo preset show basic             # Mostrar detalles de un preset
  osdo preset create my-preset       # Crear un nuevo preset`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
	},
}

// presetListCmd lists available presets
var presetListCmd = &cobra.Command{
	Use:   "list",
	Short: "Listar presets disponibles",
	Long:  "Lista todos los presets de configuración disponibles.",
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewManager()
		if err := configManager.Init(); err != nil {
			fmt.Printf("Error inicializando configuración: %v\n", err)
			os.Exit(1)
		}
		
		cfg, err := configManager.LoadCLIConfig()
		if err != nil {
			fmt.Printf("Error cargando configuración: %v\n", err)
			os.Exit(1)
		}

		listPresets(cfg)
	},
}

// presetShowCmd shows details of a specific preset
var presetShowCmd = &cobra.Command{
	Use:   "show <preset-name>",
	Short: "Mostrar detalles de un preset",
	Long:  "Muestra la configuración detallada de un preset específico.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewManager()
		if err := configManager.Init(); err != nil {
			fmt.Printf("Error inicializando configuración: %v\n", err)
			os.Exit(1)
		}
		
		cfg, err := configManager.LoadCLIConfig()
		if err != nil {
			fmt.Printf("Error cargando configuración: %v\n", err)
			os.Exit(1)
		}

		presetName := args[0]
		showPreset(cfg, presetName)
	},
}

// presetCreateCmd creates a new preset
var presetCreateCmd = &cobra.Command{
	Use:   "create <preset-name>",
	Short: "Crear un nuevo preset",
	Long:  "Crea un nuevo preset de configuración de forma interactiva.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewManager()
		if err := configManager.Init(); err != nil {
			fmt.Printf("Error inicializando configuración: %v\n", err)
			os.Exit(1)
		}
		
		cfg, err := configManager.LoadCLIConfig()
		if err != nil {
			fmt.Printf("Error cargando configuración: %v\n", err)
			os.Exit(1)
		}

		presetName := args[0]
		err = createPreset(cfg, presetName, configManager)
		if err != nil {
			fmt.Printf("Error creando preset: %v\n", err)
			os.Exit(1)
		}
	},
}

// listPresets lists all available presets
func listPresets(cfg *config.CLIConfig) {
	fmt.Println(color.CyanString("📋 Presets Disponibles"))
	fmt.Println(strings.Repeat("=", 50))

	if len(cfg.Presets) == 0 {
		fmt.Println(color.YellowString("No hay presets configurados."))
		fmt.Println("Usa 'osdo preset create <nombre>' para crear uno nuevo.")
		return
	}

	for _, preset := range cfg.Presets {
		fmt.Printf("\n%s %s\n", color.GreenString("📦"), color.WhiteString(preset.Name))
		fmt.Printf("  Descripción: %s\n", preset.Description)
		fmt.Printf("  Plataforma: %s\n", color.CyanString(string(preset.Platform)))
		fmt.Printf("  Componentes: %s\n", color.YellowString(strings.Join(preset.Components, ", ")))
		
		if preset.Namespace != "" {
			fmt.Printf("  Namespace: %s\n", preset.Namespace)
		}
	}

	fmt.Printf("\nTotal: %s presets\n", color.WhiteString(fmt.Sprintf("%d", len(cfg.Presets))))
}

// showPreset shows detailed information about a specific preset
func showPreset(cfg *config.CLIConfig, presetName string) {
	var preset *config.PresetConfig
	for _, p := range cfg.Presets {
		if p.Name == presetName {
			preset = &p
			break
		}
	}
	
	if preset == nil {
		fmt.Printf("Preset '%s' no encontrado.\n", presetName)
		fmt.Println("Usa 'osdo preset list' para ver presets disponibles.")
		return
	}

	fmt.Printf("%s %s\n", color.CyanString("📦"), color.WhiteString("Preset: %s", presetName))
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("Descripción: %s\n", preset.Description)
	fmt.Printf("Plataforma: %s\n", color.CyanString(string(preset.Platform)))
	fmt.Printf("Namespace: %s\n", color.YellowString(preset.Namespace))

	if len(preset.Components) > 0 {
		fmt.Println(color.YellowString("\n🔧 Componentes:"))
		for i, component := range preset.Components {
			fmt.Printf("  %d. %s\n", i+1, component)
		}
	}

	if len(preset.Config) > 0 {
		fmt.Println(color.YellowString("\n⚙️ Configuración:"))
		for key, value := range preset.Config {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	if preset.DryRun {
		fmt.Printf("\n%s %s\n", 
			color.YellowString("⚠️"), 
			color.YellowString("Modo dry-run habilitado"))
	}

	if preset.Timeout > 0 {
		fmt.Printf("Timeout: %s segundos\n", fmt.Sprintf("%d", preset.Timeout))
	}
}

// createPreset creates a new preset interactively
func createPreset(cfg *config.CLIConfig, presetName string, configManager *config.Manager) error {
	// Check if preset already exists
	for _, p := range cfg.Presets {
		if p.Name == presetName {
			fmt.Printf("El preset '%s' ya existe.\n", presetName)
			return fmt.Errorf("preset already exists")
		}
	}

	// Initialize UI manager
	uiConfig := &ui.UIConfig{
		Mode:     ui.UIModeCLI,
		NoPrompt: false,
		Verbose:  verbose,
	}
	
	uiManager, err := ui.NewUIManager(uiConfig, &config.Config{})
	if err != nil {
		return fmt.Errorf("error inicializando UI: %w", err)
	}

	prompter := uiManager.GetPrompter()

	fmt.Printf("%s Creando preset: %s\n", color.CyanString("📦"), color.WhiteString(presetName))
	fmt.Println()

	// Get description
	description, err := prompter.Input("Descripción del preset", "")
	if err != nil {
		return err
	}

	// Select platform (simplified)
	platformOptions := []string{"kubernetes", "k3s", "docker-compose", "docker-swarm", "helm"}
	platformStr, err := prompter.Select("Selecciona plataforma", platformOptions)
	if err != nil {
		return err
	}

	// Get components (simplified)
	components, err := prompter.MultiSelect("Selecciona componentes", []string{
		"sonarqube", "defectdojo", "grafana", "prometheus", "gitlab", "jenkins", "nexus",
	})
	if err != nil {
		return err
	}

	// Get namespace
	namespace, err := prompter.Input("Namespace", "osdo-devsecops")
	if err != nil {
		return err
	}

	// Get timeout
	timeoutStr, err := prompter.Input("Timeout (segundos)", "600")
	if err != nil {
		return err
	}

	timeout := 600
	if timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil {
			timeout = t
		}
	}

	// Get dry run option
	dryRun, err := prompter.Confirm("¿Habilitar modo dry-run por defecto?")
	if err != nil {
		return err
	}

	// Create preset
	preset := config.PresetConfig{
		Name:        presetName,
		Description: description,
		Platform:    config.Platform(platformStr),
		Components:  components,
		Namespace:   namespace,
		Timeout:     timeout,
		DryRun:      dryRun,
		Config:      make(map[string]interface{}),
	}

	// Add preset to configuration
	cfg.Presets = append(cfg.Presets, preset)

	// Save configuration
	if err := configManager.Save(); err != nil {
		return fmt.Errorf("error guardando configuración: %w", err)
	}

	fmt.Printf("\n%s Preset '%s' creado exitosamente!\n", 
		color.GreenString("✅"), 
		color.WhiteString(presetName))

	return nil
}

func init() {
	rootCmd.AddCommand(presetCmd)

	// Add subcommands
	presetCmd.AddCommand(presetListCmd)
	presetCmd.AddCommand(presetShowCmd)
	presetCmd.AddCommand(presetCreateCmd)
}
