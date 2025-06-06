package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// CLIPrompter implements InteractivePrompter for CLI interface
type CLIPrompter struct {
	noPrompt bool
	scanner  *bufio.Scanner
}

// NewCLIPrompter creates a new CLI prompter
func NewCLIPrompter(noPrompt bool) *CLIPrompter {
	return &CLIPrompter{
		noPrompt: noPrompt,
		scanner:  bufio.NewScanner(os.Stdin),
	}
}

// Confirm implements InteractivePrompter interface
func (p *CLIPrompter) Confirm(message string) (bool, error) {
	if p.noPrompt {
		return true, nil
	}

	fmt.Printf("%s %s (y/N): ", color.YellowString("❓"), message)
	
	if p.scanner.Scan() {
		response := strings.TrimSpace(strings.ToLower(p.scanner.Text()))
		return response == "y" || response == "yes" || response == "sí" || response == "si", nil
	}
	
	if err := p.scanner.Err(); err != nil {
		return false, fmt.Errorf("error leyendo entrada: %w", err)
	}
	
	return false, nil
}

// Input implements InteractivePrompter interface
func (p *CLIPrompter) Input(prompt string, defaultValue string) (string, error) {
	if p.noPrompt && defaultValue != "" {
		return defaultValue, nil
	}

	if defaultValue != "" {
		fmt.Printf("%s %s (default: %s): ", 
			color.CyanString("💬"), 
			prompt, 
			grayString(defaultValue))
	} else {
		fmt.Printf("%s %s: ", color.CyanString("💬"), prompt)
	}
	
	if p.scanner.Scan() {
		response := strings.TrimSpace(p.scanner.Text())
		if response == "" && defaultValue != "" {
			return defaultValue, nil
		}
		return response, nil
	}
	
	if err := p.scanner.Err(); err != nil {
		return "", fmt.Errorf("error leyendo entrada: %w", err)
	}
	
	return defaultValue, nil
}

// Select implements InteractivePrompter interface
func (p *CLIPrompter) Select(prompt string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no hay opciones disponibles")
	}

	if p.noPrompt {
		return options[0], nil
	}

	fmt.Println(color.CyanString("🔍 " + prompt))
	for i, option := range options {
		fmt.Printf("  %d. %s\n", i+1, option)
	}
	fmt.Println()

	for {
		fmt.Print("Selecciona una opción (número): ")
		if p.scanner.Scan() {
			input := strings.TrimSpace(p.scanner.Text())
			index, err := strconv.Atoi(input)
			
			if err != nil || index < 1 || index > len(options) {
				fmt.Printf("⚠️  Selección inválida. Debe ser un número entre 1 y %d\n", len(options))
				continue
			}
			
			return options[index-1], nil
		}
		
		if err := p.scanner.Err(); err != nil {
			return "", fmt.Errorf("error leyendo entrada: %w", err)
		}
	}
}

// MultiSelect implements InteractivePrompter interface
func (p *CLIPrompter) MultiSelect(prompt string, options []string) ([]string, error) {
	if len(options) == 0 {
		return []string{}, nil
	}

	if p.noPrompt {
		return options, nil
	}

	fmt.Println(color.CyanString("🔍 " + prompt))
	fmt.Println("Selecciona múltiples opciones separadas por comas:")
	
	for i, option := range options {
		fmt.Printf("  %d. %s\n", i+1, option)
	}
	fmt.Println()

	fmt.Print("Selecciona opciones (números separados por comas): ")
	
	if p.scanner.Scan() {
		input := strings.TrimSpace(p.scanner.Text())
		if input == "" {
			return []string{}, nil
		}

		var selected []string
		parts := strings.Split(input, ",")
		
		for _, part := range parts {
			index, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || index < 1 || index > len(options) {
				fmt.Printf("⚠️  Índice inválido: %s\n", part)
				continue
			}
			selected = append(selected, options[index-1])
		}
		
		return selected, nil
	}
	
	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("error leyendo entrada: %w", err)
	}
	
	return []string{}, nil
}

// NoPrompter implements InteractivePrompter with no user interaction
type NoPrompter struct{}

// NewNoPrompter creates a prompter that doesn't prompt for user input
func NewNoPrompter() *NoPrompter {
	return &NoPrompter{}
}

// Confirm always returns true
func (p *NoPrompter) Confirm(message string) (bool, error) {
	return true, nil
}

// Input returns the default value
func (p *NoPrompter) Input(prompt string, defaultValue string) (string, error) {
	return defaultValue, nil
}

// Select returns the first option
func (p *NoPrompter) Select(prompt string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no hay opciones disponibles")
	}
	return options[0], nil
}

// MultiSelect returns all options
func (p *NoPrompter) MultiSelect(prompt string, options []string) ([]string, error) {
	return options, nil
}
