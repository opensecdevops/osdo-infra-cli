package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	
	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/monitor"
	"github.com/osdo/osdo-infra-cli/pkg/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Verifica el estado de los componentes OSDO desplegados",
	Long: `Verifica el estado de salud, disponibilidad y recursos de los componentes
DevSecOps desplegados en la plataforma especificada.

Proporciona información detallada sobre:
- Estado de los servicios
- Salud de los endpoints
- Uso de recursos
- Logs recientes
- Métricas de rendimiento

Ejemplos:
  osdo status
  osdo status --platform kubernetes --namespace osdo
  osdo status --component gitlab --detailed
  osdo status --format json --output status.json
  osdo status --watch --interval 30s`,
	RunE: runStatus,
}

var (
	statusPlatform   string
	statusComponent  string
	statusNamespace  string
	statusDetailed   bool
	statusWatch      bool
	statusInterval   time.Duration
	statusFormat     string
	statusOutput     string
	statusNoColor    bool
	statusQuiet      bool
)

func init() {
	rootCmd.AddCommand(statusCmd)

	// Flags principales
	statusCmd.Flags().StringVarP(&statusPlatform, "platform", "p", "", 
		"Plataforma de despliegue (kubernetes, k3s, docker, swarm)")
	statusCmd.Flags().StringVarP(&statusComponent, "component", "c", "", 
		"Verificar solo un componente específico")
	statusCmd.Flags().StringVarP(&statusNamespace, "namespace", "n", "", 
		"Namespace/contexto a verificar")
	
	// Flags de visualización
	statusCmd.Flags().BoolVarP(&statusDetailed, "detailed", "d", false, 
		"Mostrar información detallada")
	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, 
		"Monitorear continuamente el estado")
	statusCmd.Flags().DurationVar(&statusInterval, "interval", 30*time.Second, 
		"Intervalo para el modo watch")
	statusCmd.Flags().StringVarP(&statusFormat, "format", "f", "table", 
		"Formato de salida (table, json, yaml)")
	statusCmd.Flags().StringVarP(&statusOutput, "output", "o", "", 
		"Archivo de salida (por defecto: stdout)")
	statusCmd.Flags().BoolVar(&statusNoColor, "no-color", false, 
		"Deshabilitar colores en la salida")
	statusCmd.Flags().BoolVarP(&statusQuiet, "quiet", "q", false, 
		"Mostrar solo errores")
}

func runStatus(cmd *cobra.Command, args []string) error {
	if statusQuiet {
		logger.SetLevel("error")
	}

	// Cargar configuración
	configManager := config.NewManager(viper.GetString("config"), "")
	cliConfig, err := configManager.LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("error cargando configuración: %w", err)
	}

	// Determinar plataforma
	platform := statusPlatform
	if platform == "" {
		platform = cliConfig.DefaultPlatform
	}

	// Determinar namespace
	namespace := statusNamespace
	if namespace == "" {
		namespace = "osdo"
	}

	logger.Info("Verificando estado de componentes OSDO", 
		"platform", platform, "namespace", namespace)

	// Crear monitor
	mon, err := monitor.New(platform, namespace, cliConfig)
	if err != nil {
		return fmt.Errorf("error creando monitor: %w", err)
	}

	if statusWatch {
		return runStatusWatch(mon)
	}

	return runStatusOnce(mon)
}

func runStatusOnce(mon *monitor.Monitor) error {
	// Obtener estado
	status, err := getSystemStatus(mon)
	if err != nil {
		return fmt.Errorf("error obteniendo estado: %w", err)
	}

	// Mostrar resultados
	return displayStatus(status)
}

func runStatusWatch(mon *monitor.Monitor) error {
	logger.Info("Iniciando monitoreo continuo", "interval", statusInterval)
	
	ticker := time.NewTicker(statusInterval)
	defer ticker.Stop()

	// Mostrar estado inicial
	if err := runStatusOnce(mon); err != nil {
		logger.Error("Error obteniendo estado inicial", "error", err)
	}

	for {
		select {
		case <-ticker.C:
			// Limpiar pantalla (solo en modo interactivo)
			if statusFormat == "table" && statusOutput == "" {
				clearScreen()
			}
			
			if err := runStatusOnce(mon); err != nil {
				logger.Error("Error obteniendo estado", "error", err)
			}
		}
	}
}

func getSystemStatus(mon *monitor.Monitor) (*config.DeploymentStatus, error) {
	if statusComponent != "" {
		// Verificar componente específico
		return mon.GetComponentStatus(statusComponent)
	}
	
	// Verificar todos los componentes
	return mon.GetSystemStatus()
}

func displayStatus(status *config.DeploymentStatus) error {
	switch statusFormat {
	case "json":
		return displayStatusJSON(status)
	case "yaml":
		return displayStatusYAML(status)
	default:
		return displayStatusTable(status)
	}
}

func displayStatusTable(status *config.DeploymentStatus) error {
	renderer := ui.NewTableRenderer(!statusNoColor)
	
	// Resumen general
	renderer.AddSection("📊 Resumen del Sistema")
	renderer.AddRow("Plataforma", status.Platform)
	renderer.AddRow("Environment", status.Environment)
	renderer.AddRow("Última Actualización", status.LastUpdate.Format("2006-01-02 15:04:05"))
	renderer.AddRow("Estado General", getHealthStatusIcon(status.HealthStatus))
	
	// Tabla de componentes
	renderer.AddSection("🔧 Estado de Componentes")
	
	headers := []string{"Componente", "Estado", "Salud", "Última Verificación"}
	if statusDetailed {
		headers = append(headers, "CPU", "Memoria", "Endpoints")
	}
	
	rows := [][]string{}
	for _, comp := range status.Components {
		row := []string{
			comp.Name,
			getStatusIcon(comp.Status),
			getHealthStatusIcon(comp.Health),
			comp.LastCheck.Format("15:04:05"),
		}
		
		if statusDetailed {
			row = append(row, 
				comp.Resources.CPUUsage,
				comp.Resources.MemoryUsage,
				strings.Join(comp.Endpoints, ", "),
			)
		}
		
		rows = append(rows, row)
	}
	
	table := renderer.CreateTable(headers, rows)
	output := renderer.Render(table)

	// Escribir salida
	if statusOutput != "" {
		return writeToFile(statusOutput, output)
	}
	
	fmt.Print(output)
	
	// Mostrar detalles adicionales si se solicita
	if statusDetailed {
		return displayDetailedInfo(status)
	}
	
	return nil
}

func displayDetailedInfo(status *config.DeploymentStatus) error {
	for _, comp := range status.Components {
		if len(comp.Errors) > 0 {
			logger.Warning(fmt.Sprintf("❌ Errores en %s:", comp.Name))
			for _, err := range comp.Errors {
				logger.Warning(fmt.Sprintf("   • %s", err))
			}
		}
		
		if statusDetailed && len(comp.Metadata) > 0 {
			logger.Info(fmt.Sprintf("📋 Metadata de %s:", comp.Name))
			for key, value := range comp.Metadata {
				logger.Info(fmt.Sprintf("   %s: %s", key, value))
			}
		}
	}
	
	return nil
}

func displayStatusJSON(status *config.DeploymentStatus) error {
	output, err := ui.FormatJSON(status)
	if err != nil {
		return err
	}
	
	if statusOutput != "" {
		return writeToFile(statusOutput, output)
	}
	
	fmt.Print(output)
	return nil
}

func displayStatusYAML(status *config.DeploymentStatus) error {
	output, err := ui.FormatYAML(status)
	if err != nil {
		return err
	}
	
	if statusOutput != "" {
		return writeToFile(statusOutput, output)
	}
	
	fmt.Print(output)
	return nil
}

func writeToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creando archivo: %w", err)
	}
	defer file.Close()
	
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("error escribiendo archivo: %w", err)
	}
	
	logger.Info("Estado guardado", "file", filename)
	return nil
}

func getStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "deployed", "running":
		if statusNoColor {
			return "✓ " + status
		}
		return "🟢 " + status
	case "pending", "starting":
		if statusNoColor {
			return "⧗ " + status
		}
		return "🟡 " + status
	case "failed", "error", "crashed":
		if statusNoColor {
			return "✗ " + status
		}
		return "🔴 " + status
	case "unknown":
		if statusNoColor {
			return "? " + status
		}
		return "⚪ " + status
	default:
		return status
	}
}

func getHealthStatusIcon(health string) string {
	switch strings.ToLower(health) {
	case "healthy":
		if statusNoColor {
			return "✓ Saludable"
		}
		return "💚 Saludable"
	case "unhealthy":
		if statusNoColor {
			return "✗ No Saludable"
		}
		return "❤️ No Saludable"
	case "warning":
		if statusNoColor {
			return "⚠ Advertencia"
		}
		return "💛 Advertencia"
	case "unknown":
		if statusNoColor {
			return "? Desconocido"
		}
		return "🤔 Desconocido"
	default:
		return health
	}
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}
