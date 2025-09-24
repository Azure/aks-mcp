package azaks

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/command"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/security"
	"github.com/google/shlex"
)

// AksOperationsExecutor handles execution of AKS operations
type AksOperationsExecutor struct{}

// NewAksOperationsExecutor creates a new AksOperationsExecutor
func NewAksOperationsExecutor() *AksOperationsExecutor {
	return &AksOperationsExecutor{}
}

// Execute handles the AKS operations
func (e *AksOperationsExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Parse operation parameter
	operation, ok := params["operation"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'operation' parameter")
	}

	// Parse args parameter
	args, ok := params["args"].(string)
	if !ok {
		args = ""
	}

	// Ensure az CLI returns JSON unless caller already specified an output format
	args, err := ensureOutputFormatArgs(args)
	if err != nil {
		return "", err
	}

	// Validate access for this operation
	if err := ValidateOperationAccess(operation, cfg); err != nil {
		return "", err
	}

	// Map operation to Azure CLI command
	baseCommand, err := MapOperationToCommand(operation)
	if err != nil {
		return "", err
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
	return process.Run(cmdArgs)
}

// ExecuteSpecificCommand executes a specific operation with the given arguments (for backward compatibility)
func (e *AksOperationsExecutor) ExecuteSpecificCommand(operation string, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Create new params with operation
	newParams := make(map[string]interface{})
	for k, v := range params {
		newParams[k] = v
	}
	newParams["operation"] = operation

	return e.Execute(newParams, cfg)
}

// ensureOutputFormatArgs adds --output json when no explicit output format is provided
func ensureOutputFormatArgs(args string) (string, error) {
	if strings.TrimSpace(args) == "" {
		return "--output json", nil
	}

	parsedArgs, err := shlex.Split(args)
	if err != nil {
		return "", fmt.Errorf("parsing args: %w", err)
	}

	for _, token := range parsedArgs {
		lowerToken := strings.ToLower(token)
		switch {
		case lowerToken == "--output":
			return args, nil
		case strings.HasPrefix(lowerToken, "--output="):
			return args, nil
		case lowerToken == "-o":
			return args, nil
		case strings.HasPrefix(lowerToken, "-o") && len(lowerToken) > 2:
			return args, nil
		}
	}

	return args + " --output json", nil
}
