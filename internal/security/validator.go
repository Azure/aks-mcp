package security

import (
	"strings"
)

// Command type constants
const (
	CommandTypeAz = "az"
)

var (
	// AzReadOperations defines az operations that don't modify state
	AzReadOperations = []string{
		"az aks show",
		"az aks list",
		// "az aks get-credentials",
		"az aks get-versions",
		"az aks browse",
		"az aks nodepool list",
		"az aks nodepool show",
		"az account",
		"az find",
		// "login",
		"az version",
		"az help",
		"az config",
		"az group list",
		"az group show",
		"az resource list",
		"az resource show",
	}
)

// Validator handles validation of commands against security configuration
type Validator struct {
	secConfig *SecurityConfig
}

// NewValidator creates a new Validator instance with the given security configuration
func NewValidator(secConfig *SecurityConfig) *Validator {
	return &Validator{
		secConfig: secConfig,
	}
}

// ValidationError represents a security validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// getReadOperationsList returns the appropriate list of read operations based on command type
func (v *Validator) getReadOperationsList(commandType string) []string {
	switch commandType {
	case CommandTypeAz:
		return AzReadOperations
	default:
		return []string{}
	}
}

// ValidateCommand validates a command against all security settings
// The command parameter should be the full command string (e.g., "az aks show --name myCluster")
// AzReadOperations should now contain full command prefixes with "az" included
func (v *Validator) ValidateCommand(command, commandType string) error {
	readOperations := v.getReadOperationsList(commandType)

	// Check access level restrictions
	if err := v.validateAccessLevel(command, readOperations); err != nil {
		return err
	}

	return nil
}

// validateAccessLevel validates if a command is allowed based on the current access level
func (v *Validator) validateAccessLevel(command string, readOperations []string) error {
	// Check if this is a read operation
	isReadOperation := v.isReadOperation(command, readOperations)

	// Handle restrictions based on access level
	switch v.secConfig.AccessLevel {
	case "readonly":
		if !isReadOperation {
			return &ValidationError{Message: "Error: Cannot execute write operations in read-only mode"}
		}
	case "readwrite":
		// All read and write operations are allowed, but not admin operations
		// Admin operations are handled separately by not registering those commands
	case "admin":
		// All operations are allowed
	default:
		// Default to readwrite behavior for unknown access levels
		// This could alternatively return an error for invalid access levels
	}

	return nil
}

// isReadOperation checks if a command is a read operation
func (v *Validator) isReadOperation(command string, allowedOperations []string) bool {
	// Normalize command by removing any options/arguments
	// This extracts the base command like "az aks show" from "az aks show --name myCluster"
	cmdParts := strings.Fields(command)
	var baseCommand string

	if len(cmdParts) >= 3 && cmdParts[0] == CommandTypeAz {
		// Take the first three parts (e.g., "az aks show")
		baseCommand = strings.Join(cmdParts[:3], " ")
	} else if len(cmdParts) == 2 && cmdParts[0] == CommandTypeAz {
		// Handle commands with just two parts (e.g., "az version")
		baseCommand = strings.Join(cmdParts[:2], " ")
	}

	// Now check if the base command is in our list of allowed read operations
	for _, allowed := range allowedOperations {
		if baseCommand == allowed {
			return true
		}

		// Also check if it's a prefix match (for commands like "az account list")
		// where we may have just "az account" in our allowed operations
		if strings.HasPrefix(baseCommand, allowed) {
			return true
		}
	}

	return false
}
