package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Color helpers are now in colors.go

// CLIProgress implements ProgressReporter for CLI interface
type CLIProgress struct {
	total    int
	current  int
	started  time.Time
	spinner  []string
	spinIdx  int
}

// NewCLIProgress creates a new CLI progress reporter
func NewCLIProgress() *CLIProgress {
	return &CLIProgress{
		spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Start implements ProgressReporter interface
func (p *CLIProgress) Start(total int) {
	p.total = total
	p.current = 0
	p.started = time.Now()
	
	fmt.Println(color.CyanString("🚀 Iniciando despliegue..."))
	fmt.Printf("Total de tareas: %d\n", total)
	fmt.Println()
}

// Update implements ProgressReporter interface
func (p *CLIProgress) Update(current int, message string) {
	p.current = current
	
	// Calculate progress percentage
	percentage := float64(current) / float64(p.total) * 100
	
	// Create progress bar
	barWidth := 40
	filled := int(percentage / 100 * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	
	// Show spinner
	spinner := p.spinner[p.spinIdx%len(p.spinner)]
	p.spinIdx++
	
	// Calculate elapsed time
	elapsed := time.Since(p.started)
	
	// Print progress line
	fmt.Printf("\r%s [%s] %.1f%% (%d/%d) - %s - %v",
		color.YellowString(spinner),
		color.GreenString(bar),
		percentage,
		current,
		p.total,
		color.WhiteString(message),
		grayString(elapsed.Truncate(time.Second).String()),
	)
}

// Finish implements ProgressReporter interface
func (p *CLIProgress) Finish(message string) {
	elapsed := time.Since(p.started)
	
	fmt.Printf("\r%s [%s] 100%% (%d/%d) - %s - %v\n",
		color.GreenString("✓"),
		color.GreenString(strings.Repeat("█", 40)),
		p.total,
		p.total,
		color.GreenString(message),
		grayString(elapsed.Truncate(time.Second).String()),
	)
	
	fmt.Println()
	fmt.Println(color.GreenString("🎉 ¡Despliegue completado exitosamente!"))
	fmt.Printf("Tiempo total: %v\n", color.CyanString(elapsed.Truncate(time.Second).String()))
}

// Error implements ProgressReporter interface
func (p *CLIProgress) Error(err error) {
	elapsed := time.Since(p.started)
	
	fmt.Printf("\r%s [%s] Error - %v - %v\n",
		color.RedString("✗"),
		color.RedString(strings.Repeat("█", int(float64(p.current)/float64(p.total)*40))+strings.Repeat("░", 40-int(float64(p.current)/float64(p.total)*40))),
		color.RedString(err.Error()),
		grayString(elapsed.Truncate(time.Second).String()),
	)
	
	fmt.Println()
	fmt.Println(color.RedString("❌ Error durante el despliegue"))
	fmt.Printf("Error: %s\n", color.RedString(err.Error()))
}

// SimpleProgress provides a simple progress implementation
type SimpleProgress struct {
	message string
}

// NewSimpleProgress creates a simple progress reporter
func NewSimpleProgress() *SimpleProgress {
	return &SimpleProgress{}
}

// Start implements ProgressReporter interface
func (p *SimpleProgress) Start(total int) {
	fmt.Printf("Iniciando proceso con %d tareas...\n", total)
}

// Update implements ProgressReporter interface
func (p *SimpleProgress) Update(current int, message string) {
	p.message = message
	fmt.Printf("Progreso: %s\n", message)
}

// Finish implements ProgressReporter interface
func (p *SimpleProgress) Finish(message string) {
	fmt.Printf("✓ Completado: %s\n", message)
}

// Error implements ProgressReporter interface
func (p *SimpleProgress) Error(err error) {
	fmt.Printf("✗ Error: %v\n", err)
}
