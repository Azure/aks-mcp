package config

import (
	"fmt"
	"os/exec"
)

// Validator handles all validation logic for AKS MCP
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

	// kubectl is always required (core Kubernetes functionality)
	if !v.isCliInstalled("kubectl") {
		v.errors = append(v.errors, "kubectl is not installed or not found in PATH")
		valid = false
	}

	// helm is optional - only validate if explicitly enabled
	if v.config.AdditionalTools["helm"] && !v.isCliInstalled("helm") {
		v.errors = append(v.errors, "helm is not installed or not found in PATH (required when --additional-tools includes helm)")
		valid = false
	}

	// cilium is optional - only validate if explicitly enabled
	if v.config.AdditionalTools["cilium"] && !v.isCliInstalled("cilium") {
		v.errors = append(v.errors, "cilium is not installed or not found in PATH (required when --additional-tools includes cilium)")
		valid = false
	}

	// hubble is optional - only validate if explicitly enabled
	if v.config.AdditionalTools["hubble"] && !v.isCliInstalled("hubble") {
		v.errors = append(v.errors, "hubble is not installed or not found in PATH (required when --additional-tools includes hubble)")
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
