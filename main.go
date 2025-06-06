package main

import (
	"fmt"
	"os"

	"github.com/osdo/osdo-infra-cli/cmd"
	"github.com/osdo/osdo-infra-cli/pkg/config"
	"github.com/osdo/osdo-infra-cli/pkg/logger"
)

func main() {
	// Inicializar logger
	logger.Init()

	// Inicializar configuración
	configManager := config.NewManager()
	if err := configManager.Init(); err != nil {
		fmt.Printf("Error inicializando configuración: %v\n", err)
		os.Exit(1)
	}

	// Ejecutar comando root
	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error ejecutando comando: %v\n", err)
		os.Exit(1)
	}
}
