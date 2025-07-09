package az

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/command"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/security"
	"github.com/Azure/aks-mcp/internal/tools"
)

// AzExecutor implements the CommandExecutor interface for az commands
type AzExecutor struct{}

// This line ensures AzExecutor implements the CommandExecutor interface
var _ tools.CommandExecutor = (*AzExecutor)(nil)

// NewExecutor creates a new AzExecutor instance
func NewExecutor() *AzExecutor {
	return &AzExecutor{}
}

// Execute handles general az command execution
func (e *AzExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	azCmd, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("invalid command parameter")
	}

	// Determine command type based on the command prefix
	commandType := security.CommandTypeAz
	if strings.HasPrefix(azCmd, "kubectl") {
		commandType = security.CommandTypeKubectl
	}

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(azCmd, commandType)
	if err != nil {
		return "", err
	}

	// Extract binary name and arguments from command
	cmdParts := strings.Fields(azCmd)
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

	// Validate binary name
	if binaryName != "az" && binaryName != "kubectl" {
		return "", fmt.Errorf("command must start with 'az' or 'kubectl'")
	}

	// Execute the command
	process := command.NewShellProcess(binaryName, cfg.Timeout)
	return process.Run(cmdArgs)
}

// ExecuteSpecificCommand executes a specific az command with the given arguments
func (e *AzExecutor) ExecuteSpecificCommand(cmd string, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	args, ok := params["args"].(string)
	if !ok {
		args = ""
	}

	fullCmd := cmd
	if args != "" {
		fullCmd += " " + args
	}

	// Determine command type based on the command prefix
	commandType := security.CommandTypeAz
	if strings.HasPrefix(fullCmd, "kubectl") {
		commandType = security.CommandTypeKubectl
	}

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(fullCmd, commandType)
	if err != nil {
		return "", err
	}

	// Extract binary name from command (should be "az")
	cmdParts := strings.Fields(fullCmd)
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

	// Validate binary name
	if binaryName != "az" && binaryName != "kubectl" {
		return "", fmt.Errorf("command must start with 'az' or 'kubectl'")
	}

	// Execute the command
	process := command.NewShellProcess(binaryName, cfg.Timeout)
	return process.Run(cmdArgs)
}

// CreateCommandExecutorFunc creates a CommandExecutor for a specific az command
func CreateCommandExecutorFunc(cmd string) tools.CommandExecutorFunc {
	f := func(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
		executor := NewExecutor()
		return executor.ExecuteSpecificCommand(cmd, params, cfg)
	}
	return tools.CommandExecutorFunc(f)
}
