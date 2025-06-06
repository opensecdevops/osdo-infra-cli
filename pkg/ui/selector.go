package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/osdo/osdo-infra-cli/pkg/config"
)

// CLISelector implements ComponentSelector for CLI interface
type CLISelector struct {
	prompter InteractivePrompter
}

// NewCLISelector creates a new CLI selector
func NewCLISelector(prompter InteractivePrompter) *CLISelector {
	return &CLISelector{
		prompter: prompter,
	}
}

// SelectComponents implements interactive component selection
func (s *CLISelector) SelectComponents(available []config.ComponentConfig) ([]config.ComponentConfig, error) {
	fmt.Println(color.CyanString("📦 Seleccionar Componentes DevSecOps"))
	fmt.Println("Selecciona los componentes que deseas desplegar:")
	fmt.Println()

	var selected []config.ComponentConfig
	
	for i, component := range available {
		fmt.Printf("%s %d. %s\n", 
			color.YellowString("🔧"), 
			i+1, 
			color.WhiteString("%s - %s", component.Name, component.Description))
	}
	fmt.Println()

	// Get user input for component selection
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Ingresa los números de los componentes separados por comas (ej: 1,3,5): ")
	
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			return selected, nil
		}

		parts := strings.Split(input, ",")
		for _, part := range parts {
			index, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || index < 1 || index > len(available) {
				fmt.Printf("⚠️  Índice inválido: %s\n", part)
				continue
			}
			selected = append(selected, available[index-1])
		}
	}

	return selected, nil
}

// SelectPreset implements preset selection
func (s *CLISelector) SelectPreset(presets []config.PresetConfig) (*config.PresetConfig, error) {
	if len(presets) == 0 {
		return nil, fmt.Errorf("no hay presets disponibles")
	}

	fmt.Println(color.CyanString("🎯 Seleccionar Preset"))
	fmt.Println("Presets disponibles:")
	fmt.Println()

	for i, preset := range presets {
		fmt.Printf("%s %d. %s\n", 
			color.GreenString("📋"), 
			i+1, 
			color.WhiteString("%s - %s", preset.Name, preset.Description))
		
		if len(preset.Components) > 0 {
			fmt.Printf("   Componentes: %s\n", 
				grayString(strings.Join(preset.Components, ", ")))
		}
		fmt.Println()
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Selecciona un preset (número): ")
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			index, err := strconv.Atoi(input)
			
			if err != nil || index < 1 || index > len(presets) {
				fmt.Printf("⚠️  Selección inválida. Debe ser un número entre 1 y %d\n", len(presets))
				continue
			}
			
			return &presets[index-1], nil
		}
		
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("error leyendo entrada: %w", err)
		}
	}
}

// SelectPlatform implements platform selection
func (s *CLISelector) SelectPlatform(platforms []config.Platform) (config.Platform, error) {
	if len(platforms) == 0 {
		return "", fmt.Errorf("no hay plataformas disponibles")
	}

	fmt.Println(color.CyanString("🚀 Seleccionar Plataforma de Despliegue"))
	fmt.Println("Plataformas disponibles:")
	fmt.Println()

	platformIcons := map[config.Platform]string{
		config.PlatformKubernetes:     "☸️",
		config.PlatformK3s:           "🐳",
		config.PlatformDockerCompose: "🐋",
		config.PlatformDockerSwarm:   "🐋",
		config.PlatformHelm:          "⎈",
	}

	platformNames := map[config.Platform]string{
		config.PlatformKubernetes:     "Kubernetes",
		config.PlatformK3s:           "K3s (Lightweight Kubernetes)",
		config.PlatformDockerCompose: "Docker Compose",
		config.PlatformDockerSwarm:   "Docker Swarm",
		config.PlatformHelm:          "Helm Charts",
	}

	availablePlatforms := make([]config.Platform, 0)
	for _, platform := range platforms {
		availablePlatforms = append(availablePlatforms, platform)
	}

	for i, platform := range availablePlatforms {
		icon := platformIcons[platform]
		name := platformNames[platform]
		fmt.Printf("%s %d. %s\n", 
			color.BlueString(icon), 
			i+1, 
			color.WhiteString(name))
	}
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Selecciona una plataforma (número): ")
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			index, err := strconv.Atoi(input)
			
			if err != nil || index < 1 || index > len(availablePlatforms) {
				fmt.Printf("⚠️  Selección inválida. Debe ser un número entre 1 y %d\n", len(availablePlatforms))
				continue
			}
			
			return availablePlatforms[index-1], nil
		}
		
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error leyendo entrada: %w", err)
		}
	}
}

// GetDeploymentOptions implements deployment options gathering
func (s *CLISelector) GetDeploymentOptions(platform config.Platform) (*config.DeploymentOptions, error) {
	fmt.Println(color.CyanString("⚙️  Configurar Opciones de Despliegue"))
	fmt.Printf("Plataforma: %s\n", color.YellowString(string(platform)))
	fmt.Println()

	options := &config.DeploymentOptions{
		Platform: platform,
		Config:   &config.ComponentSelectorConfig{},
	}

	scanner := bufio.NewScanner(os.Stdin)
	
	// Namespace (para Kubernetes/K3s/Helm)
	if platform == config.PlatformKubernetes || platform == config.PlatformK3s || platform == config.PlatformHelm {
		fmt.Print("Namespace (default: osdo-devsecops): ")
		if scanner.Scan() {
			namespace := strings.TrimSpace(scanner.Text())
			if namespace == "" {
				namespace = "osdo-devsecops"
			}
			options.Namespace = namespace
		}
	}

	// Dry run option
	fmt.Print("¿Ejecutar en modo dry-run? (y/N): ")
	if scanner.Scan() {
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		options.DryRun = response == "y" || response == "yes"
	}

	// Timeout
	fmt.Print("Timeout en segundos (default: 600): ")
	if scanner.Scan() {
		timeoutStr := strings.TrimSpace(scanner.Text())
		if timeoutStr != "" {
			if timeout, err := strconv.Atoi(timeoutStr); err == nil {
				options.Timeout = time.Duration(timeout) * time.Second
			}
		}
		if options.Timeout == 0 {
			options.Timeout = 600 * time.Second
		}
	}

	// Platform-specific options
	switch platform {
	case config.PlatformKubernetes, config.PlatformK3s:
		fmt.Print("Kubeconfig path (default: ~/.kube/config): ")
		if scanner.Scan() {
			kubeconfig := strings.TrimSpace(scanner.Text())
			if kubeconfig != "" {
				if options.Values == nil {
					options.Values = make(map[string]string)
				}
				options.Values["kubeconfig"] = kubeconfig
			}
		}

	case config.PlatformDockerCompose:
		fmt.Print("Docker Compose file path (default: docker-compose.yml): ")
		if scanner.Scan() {
			composePath := strings.TrimSpace(scanner.Text())
			if composePath == "" {
				composePath = "docker-compose.yml"
			}
			if options.Values == nil {
				options.Values = make(map[string]string)
			}
			options.Values["compose_file"] = composePath
		}

	case config.PlatformDockerSwarm:
		fmt.Print("Docker Swarm stack name (default: osdo-stack): ")
		if scanner.Scan() {
			stackName := strings.TrimSpace(scanner.Text())
			if stackName == "" {
				stackName = "osdo-stack"
			}
			if options.Values == nil {
				options.Values = make(map[string]string)
			}
			options.Values["stack_name"] = stackName
		}

	case config.PlatformHelm:
		fmt.Print("Helm chart path o repository (default: ./charts): ")
		if scanner.Scan() {
			chartPath := strings.TrimSpace(scanner.Text())
			if chartPath == "" {
				chartPath = "./charts"
			}
			if options.Values == nil {
				options.Values = make(map[string]string)
			}
			options.Values["chart_path"] = chartPath
		}
	}

	return options, nil
}
