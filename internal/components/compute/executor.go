package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/command"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/security"
)

// ComputeOperationsExecutor handles execution of compute operations
type ComputeOperationsExecutor struct{}

// NewComputeOperationsExecutor creates a new ComputeOperationsExecutor
func NewComputeOperationsExecutor() *ComputeOperationsExecutor {
	return &ComputeOperationsExecutor{}
}

// Execute handles the compute operations
func (e *ComputeOperationsExecutor) Execute(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Parse operation parameter
	operation, ok := params["operation"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'operation' parameter. Common operations: list, show, start, stop, restart, reimage. Example: operation=\"list\"")
	}

	// Parse resource_type parameter
	resourceType, ok := params["resource_type"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'resource_type' parameter. Must be 'vm' (Virtual Machine) or 'vmss' (Virtual Machine Scale Set). Example: resource_type=\"vm\"")
	}

	// Parse args parameter
	args, ok := params["args"].(string)
	if !ok {
		args = ""
	}

	// Validate access for this operation
	if err := ValidateOperationAccess(operation, cfg); err != nil {
		// Enhance access error with suggestions
		requiredLevel := GetOperationAccessLevel(operation)
		return "", fmt.Errorf("%v. Your current access level is '%s', but this operation requires '%s' access. Contact your administrator to request higher access", err, cfg.AccessLevel, requiredLevel)
	}

	// Map operation to Azure CLI command
	baseCommand, err := MapOperationToCommand(operation, resourceType)
	if err != nil {
		// Provide helpful suggestions for invalid operations
		validOps := getSuggestedOperations(resourceType, cfg.AccessLevel)
		return "", fmt.Errorf("%v. Valid operations for %s with %s access: %s", err, resourceType, cfg.AccessLevel, validOps)
	}

	// Build full command
	fullCommand := baseCommand
	if args != "" {
		fullCommand += " " + args
	}

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err = validator.ValidateCommand(fullCommand, security.CommandTypeAz)
	if err != nil {
		return "", err
	}

	// Extract binary name and arguments from command
	cmdParts := strings.Fields(fullCommand)
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Use the first part as the binary name
	binaryName := cmdParts[0]

	// The rest of the command becomes the arguments
	cmdArgs := ""
	if len(cmdParts) > 1 {
		cmdArgs = strings.Join(cmdParts[1:], " ")
	}

	// If the command is not an az command, return an error
	if binaryName != "az" {
		return "", fmt.Errorf("command must start with 'az'")
	}

	// Execute the command
	process := command.NewShellProcess(binaryName, cfg.Timeout)
	result, err := process.Run(cmdArgs)
	if err != nil {
		// Provide helpful error messages for common issues
		errorMsg := fmt.Sprintf("Azure CLI command failed: %v", err)

		// Add contextual help based on the operation
		switch operation {
		case "list":
			errorMsg += "\nTip: For listing resources, try without --name parameter, or verify --resource-group exists"
		case "show":
			errorMsg += "\nTip: Verify the resource name and resource group are correct and the resource exists"
		case "start", "stop", "restart":
			errorMsg += "\nTip: Verify the resource exists and check if it's already in the desired state"
		case "reimage":
			errorMsg += "\nTip: Verify the VMSS name is correct and the instances are ready for reimaging"
		}

		return "", fmt.Errorf("%s\nExecuted command: %s", errorMsg, fullCommand)
	}

	return strings.TrimSpace(result), nil
}

// getSuggestedOperations returns a helpful list of valid operations for the given resource type and access level
func getSuggestedOperations(resourceType, accessLevel string) string {
	var operations []string

	// Read-only operations (available to all access levels)
	operations = append(operations, "list", "show", "get-instance-view")

	// Read-write operations
	if accessLevel == "readwrite" || accessLevel == "admin" {
		switch resourceType {
		case "vm":
			// Only safe VM operations
			operations = append(operations, "start", "stop", "restart")
		case "vmss":
			// Only safe operations for AKS-managed VMSS
			operations = append(operations, "restart", "reimage")
		}
	}

	// No admin operations - all unsafe operations removed for AKS safety

	return strings.Join(operations, ", ")
}
