package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/version"
)

var (
	cfgFile    string
	verbose    bool
	dryRun     bool
	output     string
	kubeconfig string
)

// rootCmd representa el comando base cuando se llama sin subcomandos
var rootCmd = &cobra.Command{
	Use:   "osdo",
	Short: "OSDO CLI - Herramienta de línea de comandos para desplegar el stack DevSecOps",
	Long: `OSDO CLI es una herramienta multiplataforma para desplegar y gestionar 
el stack completo de herramientas DevSecOps de OSDO.

Soporta despliegue en:
- Kubernetes (K8s)
- K3s (Lightweight Kubernetes)  
- Docker Swarm
- Docker Compose

Ejemplos de uso:
  # Desplegar stack completo en K3s
  osdo deploy --platform k3s --preset production

  # Modo interactivo para seleccionar componentes
  osdo deploy --interactive

  # Verificar estado del stack
  osdo status

  # Listar presets disponibles
  osdo preset list`,
	Version: version.Version,
}

// Execute añade todos los comandos hijo al comando root y establece las flags apropiadamente.
// Es llamado por main.main(). Solo necesita ocurrir una vez al rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Flags globales
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "archivo de configuración (por defecto: $HOME/.osdo.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "salida detallada")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "mostrar lo que se haría sin ejecutar")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "formato de salida (table, json, yaml)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "ruta al archivo kubeconfig")

	// Flags locales
	rootCmd.Flags().BoolP("toggle", "t", false, "mensaje de ayuda para toggle")
}

// initConfig lee el archivo de configuración y variables de entorno si están establecidas.
func initConfig() {
	if err := config.Load(cfgFile); err != nil {
		fmt.Printf("Error cargando configuración: %v\n", err)
		os.Exit(1)
	}

	// Establecer variables globales
	config.SetVerbose(verbose)
	config.SetDryRun(dryRun)
	config.SetOutput(output)
	config.SetKubeconfig(kubeconfig)
}
