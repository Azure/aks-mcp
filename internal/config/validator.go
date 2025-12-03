package config

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Azure/aks-mcp/internal/components"
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

// isComponentEnabled checks if a component is enabled in the configuration
func (v *Validator) isComponentEnabled(componentName string) bool {
	if len(v.config.EnabledComponents) == 0 {
		return true
	}

	componentName = strings.ToLower(strings.TrimSpace(componentName))
	for _, enabled := range v.config.EnabledComponents {
		if strings.ToLower(strings.TrimSpace(enabled)) == componentName {
			return true
		}
	}
	return false
}

// validateCli checks if the required CLI tools are installed
func (v *Validator) validateCli() bool {
	valid := true

	// az is only required if enabled
	if v.isComponentEnabled("az_cli") && !v.isCliInstalled("az") {
		v.errors = append(v.errors, "az is not installed or not found in PATH (required when --enabled-components includes az_cli)")
		valid = false
	}

	// kubectl is only required if enabled
	if v.isComponentEnabled("kubectl") && !v.isCliInstalled("kubectl") {
		v.errors = append(v.errors, "kubectl is not installed or not found in PATH (required when --enabled-components includes kubectl)")
		valid = false
	}

	// helm is optional - only validate if explicitly enabled
	if v.isComponentEnabled("helm") && !v.isCliInstalled("helm") {
		v.errors = append(v.errors, "helm is not installed or not found in PATH (required when --enabled-components includes helm)")
		valid = false
	}

	// cilium is optional - only validate if explicitly enabled
	if v.isComponentEnabled("cilium") && !v.isCliInstalled("cilium") {
		v.errors = append(v.errors, "cilium is not installed or not found in PATH (required when --enabled-components includes cilium)")
		valid = false
	}

	// hubble is optional - only validate if explicitly enabled
	if v.isComponentEnabled("hubble") && !v.isCliInstalled("hubble") {
		v.errors = append(v.errors, "hubble is not installed or not found in PATH (required when --enabled-components includes hubble)")
		valid = false
	}

	return valid
}

// validateComponents validates the enabled components
func (v *Validator) validateComponents() bool {
	_, invalid, err := components.ValidateComponents(v.config.EnabledComponents)
	if err != nil {
		v.errors = append(v.errors, err.Error())
		return false
	}
	if len(invalid) > 0 {
		v.errors = append(v.errors, fmt.Sprintf("invalid components: %s", strings.Join(invalid, ", ")))
		return false
	}
	return true
}

// validateConfig checks configuration compatibility
func (v *Validator) validateConfig() bool {
	if err := v.config.ValidateConfig(); err != nil {
		v.errors = append(v.errors, err.Error())
		return false
	}
	return true
}

// Validate runs all validation checks
func (v *Validator) Validate() bool {
	// Run all validation checks
	validComponents := v.validateComponents()
	validCli := v.validateCli()
	validConfig := v.validateConfig()

	return validComponents && validCli && validConfig
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
