package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

// Color helpers are now in colors.go

// CLIMonitoring implements MonitoringUI for CLI interface
type CLIMonitoring struct {
	outputFormat string
	verbose      bool
}

// NewCLIMonitoring creates a new CLI monitoring UI
func NewCLIMonitoring(outputFormat string, verbose bool) *CLIMonitoring {
	return &CLIMonitoring{
		outputFormat: outputFormat,
		verbose:      verbose,
	}
}

// DisplayStatus implements MonitoringUI interface
func (m *CLIMonitoring) DisplayStatus(status map[string]interface{}) error {
	switch m.outputFormat {
	case "json":
		return m.displayStatusJSON(status)
	case "yaml":
		return m.displayStatusYAML(status)
	default:
		return m.displayStatusTable(status)
	}
}

// displayStatusTable shows status in a formatted table
func (m *CLIMonitoring) displayStatusTable(status map[string]interface{}) error {
	fmt.Println(color.CyanString("📊 Estado del Sistema OSDO"))
	fmt.Println(strings.Repeat("=", 60))

	// System Overview
	if overview, ok := status["overview"].(map[string]interface{}); ok {
		fmt.Println(color.YellowString("📋 Resumen General"))
		if platform, ok := overview["platform"].(string); ok {
			fmt.Printf("  Plataforma: %s\n", color.WhiteString(platform))
		}
		if totalComponents, ok := overview["total_components"].(int); ok {
			fmt.Printf("  Componentes Totales: %s\n", color.WhiteString("%d", totalComponents))
		}
		if healthyComponents, ok := overview["healthy_components"].(int); ok {
			fmt.Printf("  Componentes Saludables: %s\n", color.GreenString("%d", healthyComponents))
		}
		if unhealthyComponents, ok := overview["unhealthy_components"].(int); ok {
			fmt.Printf("  Componentes con Problemas: %s\n", color.RedString("%d", unhealthyComponents))
		}
		fmt.Println()
	}

	// Components Status
	if components, ok := status["components"].(map[string]interface{}); ok {
		fmt.Println(color.YellowString("🔧 Estado de Componentes"))
		
		// Sort components by name for consistent display
		var names []string
		for name := range components {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			if component, ok := components[name].(map[string]interface{}); ok {
				statusStr := "❓"
				statusColor := color.WhiteString
				
				if compStatus, ok := component["status"].(string); ok {
					switch compStatus {
					case "healthy":
						statusStr = "✅"
						statusColor = color.GreenString
					case "unhealthy":
						statusStr = "❌"
						statusColor = color.RedString
					case "warning":
						statusStr = "⚠️"
						statusColor = color.YellowString
					case "unknown":
						statusStr = "❓"
						statusColor = grayStringf
					}
				}

				fmt.Printf("  %s %s", statusStr, statusColor(name))
				
				if version, ok := component["version"].(string); ok {
					fmt.Printf(" (v%s)", grayString(version))
				}
				
				if message, ok := component["message"].(string); ok && message != "" {
					fmt.Printf(" - %s", grayString(message))
				}
				
				fmt.Println()

				// Show detailed info in verbose mode
				if m.verbose {
					if namespace, ok := component["namespace"].(string); ok {
						fmt.Printf("    Namespace: %s\n", grayString(namespace))
					}
					if replicas, ok := component["replicas"].(map[string]interface{}); ok {
						if ready, ok := replicas["ready"].(int); ok {
							if desired, ok := replicas["desired"].(int); ok {
								fmt.Printf("    Replicas: %s/%s\n", 
									grayString("%d", ready), 
									grayString("%d", desired))
							}
						}
					}
					if lastUpdate, ok := component["last_update"].(string); ok {
						fmt.Printf("    Última Actualización: %s\n", grayString(lastUpdate))
					}
				}
			}
		}
		fmt.Println()
	}

	// Resource Usage
	if resources, ok := status["resources"].(map[string]interface{}); ok {
		fmt.Println(color.YellowString("💾 Uso de Recursos"))
		
		if cpu, ok := resources["cpu"].(map[string]interface{}); ok {
			if usage, ok := cpu["usage_percent"].(float64); ok {
				fmt.Printf("  CPU: %s%%\n", m.formatPercentage(usage))
			}
		}
		
		if memory, ok := resources["memory"].(map[string]interface{}); ok {
			if usage, ok := memory["usage_percent"].(float64); ok {
				fmt.Printf("  Memoria: %s%%\n", m.formatPercentage(usage))
			}
		}
		
		if storage, ok := resources["storage"].(map[string]interface{}); ok {
			if usage, ok := storage["usage_percent"].(float64); ok {
				fmt.Printf("  Almacenamiento: %s%%\n", m.formatPercentage(usage))
			}
		}
		
		fmt.Println()
	}

	// Alerts
	if alerts, ok := status["alerts"].([]interface{}); ok && len(alerts) > 0 {
		fmt.Println(color.YellowString("🚨 Alertas Activas"))
		for _, alert := range alerts {
			if alertMap, ok := alert.(map[string]interface{}); ok {
				severity := "info"
				if s, ok := alertMap["severity"].(string); ok {
					severity = s
				}
				
				message := "Sin mensaje"
				if m, ok := alertMap["message"].(string); ok {
					message = m
				}
				
				icon := "ℹ️"
				colorFunc := color.WhiteString
				switch severity {
				case "critical":
					icon = "🔴"
					colorFunc = color.RedString
				case "warning":
					icon = "⚠️"
					colorFunc = color.YellowString
				case "info":
					icon = "ℹ️"
					colorFunc = color.CyanString
				}
				
				fmt.Printf("  %s %s\n", icon, colorFunc(message))
			}
		}
		fmt.Println()
	}

	// Timestamp
	fmt.Printf("Última actualización: %s\n", grayString(time.Now().Format("2006-01-02 15:04:05")))
	
	return nil
}

// displayStatusJSON shows status in JSON format
func (m *CLIMonitoring) displayStatusJSON(status map[string]interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}

// displayStatusYAML shows status in YAML format
func (m *CLIMonitoring) displayStatusYAML(status map[string]interface{}) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(status)
}

// formatPercentage formats percentage with appropriate color
func (m *CLIMonitoring) formatPercentage(usage float64) string {
	switch {
	case usage < 70:
		return color.GreenString("%.1f", usage)
	case usage < 85:
		return color.YellowString("%.1f", usage)
	default:
		return color.RedString("%.1f", usage)
	}
}

// DisplayMetrics implements MonitoringUI interface
func (m *CLIMonitoring) DisplayMetrics(metrics map[string]interface{}) error {
	switch m.outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(metrics)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer encoder.Close()
		return encoder.Encode(metrics)
	default:
		return m.displayMetricsTable(metrics)
	}
}

// displayMetricsTable shows metrics in a formatted table
func (m *CLIMonitoring) displayMetricsTable(metrics map[string]interface{}) error {
	fmt.Println(color.CyanString("📈 Métricas del Sistema"))
	fmt.Println(strings.Repeat("=", 60))

	// Performance Metrics
	if perf, ok := metrics["performance"].(map[string]interface{}); ok {
		fmt.Println(color.YellowString("🚀 Rendimiento"))
		
		if responseTime, ok := perf["avg_response_time"].(float64); ok {
			fmt.Printf("  Tiempo de Respuesta Promedio: %s ms\n", 
				color.WhiteString("%.2f", responseTime))
		}
		
		if throughput, ok := perf["requests_per_second"].(float64); ok {
			fmt.Printf("  Requests por Segundo: %s\n", 
				color.WhiteString("%.2f", throughput))
		}
		
		if errorRate, ok := perf["error_rate"].(float64); ok {
			colorFunc := color.GreenString
			if errorRate > 5 {
				colorFunc = color.RedString
			} else if errorRate > 1 {
				colorFunc = color.YellowString
			}
			fmt.Printf("  Tasa de Error: %s%%\n", colorFunc("%.2f", errorRate))
		}
		
		fmt.Println()
	}

	// Security Metrics
	if security, ok := metrics["security"].(map[string]interface{}); ok {
		fmt.Println(color.YellowString("🔒 Seguridad"))
		
		if violations, ok := security["policy_violations"].(int); ok {
			colorFunc := color.GreenString
			if violations > 0 {
				colorFunc = color.RedString
			}
			fmt.Printf("  Violaciones de Políticas: %s\n", colorFunc("%d", violations))
		}
		
		if scans, ok := security["security_scans"].(int); ok {
			fmt.Printf("  Escaneos de Seguridad: %s\n", color.WhiteString("%d", scans))
		}
		
		if vulnerabilities, ok := security["vulnerabilities"].(map[string]interface{}); ok {
			if critical, ok := vulnerabilities["critical"].(int); ok {
				colorFunc := color.GreenString
				if critical > 0 {
					colorFunc = color.RedString
				}
				fmt.Printf("  Vulnerabilidades Críticas: %s\n", colorFunc("%d", critical))
			}
			
			if high, ok := vulnerabilities["high"].(int); ok {
				colorFunc := color.GreenString
				if high > 0 {
					colorFunc = color.YellowString
				}
				fmt.Printf("  Vulnerabilidades Altas: %s\n", colorFunc("%d", high))
			}
		}
		
		fmt.Println()
	}

	return nil
}

// StartWatch implements MonitoringUI interface
func (m *CLIMonitoring) StartWatch(ctx context.Context, interval int) error {
	fmt.Println(color.CyanString("👀 Iniciando modo watch (Ctrl+C para salir)"))
	fmt.Printf("Intervalo de actualización: %d segundos\n", interval)
	fmt.Println()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println(color.YellowString("\n🛑 Deteniendo monitoreo..."))
			return nil
		case <-ticker.C:
			// Clear screen
			fmt.Print("\033[H\033[2J")
			fmt.Printf("🔄 Actualizando... %s\n\n", 
				grayString(time.Now().Format("15:04:05")))
			
			// Note: In a real implementation, this would fetch fresh status
			// For now, just show that we're watching
			fmt.Println(grayString("(Los datos se actualizarían aquí)"))
		}
	}
}

// ShowAlerts implements MonitoringUI interface
func (m *CLIMonitoring) ShowAlerts(alerts []interface{}) error {
	if len(alerts) == 0 {
		fmt.Println(color.GreenString("✅ No hay alertas activas"))
		return nil
	}

	fmt.Println(color.CyanString("🚨 Alertas del Sistema"))
	fmt.Println(strings.Repeat("=", 60))

	for i, alert := range alerts {
		if alertMap, ok := alert.(map[string]interface{}); ok {
			severity := "info"
			if s, ok := alertMap["severity"].(string); ok {
				severity = s
			}
			
			message := "Sin mensaje"
			if m, ok := alertMap["message"].(string); ok {
				message = m
			}
			
			timestamp := time.Now().Format("15:04:05")
			if t, ok := alertMap["timestamp"].(string); ok {
				timestamp = t
			}
			
			icon := "ℹ️"
			colorFunc := color.WhiteString
			switch severity {
			case "critical":
				icon = "🔴"
				colorFunc = color.RedString
			case "warning":
				icon = "⚠️"
				colorFunc = color.YellowString
			case "info":
				icon = "ℹ️"
				colorFunc = color.CyanString
			}
			
			fmt.Printf("%d. %s %s\n", i+1, icon, colorFunc(message))
			fmt.Printf("   %s - %s\n", 
				grayString("Severidad: %s", severity),
				grayString("Hora: %s", timestamp))
			
			if component, ok := alertMap["component"].(string); ok {
				fmt.Printf("   %s\n", grayString("Componente: %s", component))
			}
			
			fmt.Println()
		}
	}

	return nil
}
