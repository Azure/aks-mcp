package config

import (
	"fmt"
	"os/exec"
)

// Validator handles all validation logic for MCP Kubernetes
type Validator struct {
	// Configuration to validate
	config *ConfigData
	// Errors discovered during validation
	errors []string
}

// NewValidator creates a new validator instance
func NewValidator(cfg *ConfigData) *Validator {
	return &Validator{
		config: cfg,
		errors: make([]string, 0),
	}
}

// isCliInstalled checks if a CLI tool is installed and available in the system PATH
func (v *Validator) isCliInstalled(cliName string) bool {
	_, err := exec.LookPath(cliName)
	return err == nil
}

// validateCli checks if the required CLI tools are installed
func (v *Validator) validateCli() bool {
	valid := true

	// az is always required
	if !v.isCliInstalled("az") {
		v.errors = append(v.errors, "az is not installed or not found in PATH")
		valid = false
	}

	return valid
}

// Validate runs all validation checks
func (v *Validator) Validate() bool {
	// Run all validation checks
	validCli := v.validateCli()

	return validCli
}

// GetErrors returns all errors found during validation
func (v *Validator) GetErrors() []string {
	return v.errors
}

// PrintErrors prints all validation errors to stdout
func (v *Validator) PrintErrors() {
	for _, err := range v.errors {
		fmt.Println(err)
	}
}
