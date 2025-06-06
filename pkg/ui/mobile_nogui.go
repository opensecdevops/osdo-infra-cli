//go:build !gui
// +build !gui

package ui

import (
	"fmt"
	"log"

	"github.com/osdo/osdo-infra-cli/pkg/config"
)

// NoOpMobileUI implements MobileUI interface without GUI dependencies
type NoOpMobileUI struct {
	config *config.Config
}

// NewFyneMobileUI creates a new mobile UI without GUI dependencies
func NewFyneMobileUI(cfg *config.Config) *NoOpMobileUI {
	return &NoOpMobileUI{
		config: cfg,
	}
}

// ShowMainMenu implements MobileUI interface
func (m *NoOpMobileUI) ShowMainMenu() error {
	log.Println("GUI no disponible - usa el modo CLI en su lugar")
	fmt.Println("=== OSDO DevSecOps CLI ===")
	fmt.Println("La interfaz gráfica no está disponible en esta versión.")
	fmt.Println("Usa los comandos CLI para interactuar con la aplicación:")
	fmt.Println("")
	fmt.Println("Comandos disponibles:")
	fmt.Println("  osdo deploy    - Desplegar componentes")
	fmt.Println("  osdo monitor   - Monitorear sistema")
	fmt.Println("  osdo --help    - Mostrar ayuda")
	fmt.Println("")
	return nil
}

// ShowDeploymentWizard implements MobileUI interface
func (m *NoOpMobileUI) ShowDeploymentWizard() error {
	fmt.Println("Asistente de despliegue no disponible en modo CLI")
	fmt.Println("Usa: osdo deploy --help para ver opciones de despliegue")
	return nil
}

// ShowMonitoringDashboard implements MobileUI interface
func (m *NoOpMobileUI) ShowMonitoringDashboard() error {
	fmt.Println("Dashboard de monitoreo no disponible en modo CLI")
	fmt.Println("Usa: osdo monitor --help para ver opciones de monitoreo")
	return nil
}

// ShowSettings implements MobileUI interface
func (m *NoOpMobileUI) ShowSettings() error {
	fmt.Println("Configuración gráfica no disponible en modo CLI")
	fmt.Println("Edita los archivos de configuración directamente")
	return nil
}
