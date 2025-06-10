package tools

import (
	"github.com/Azure/aks-mcp/internal/config"
)

// CommandExecutor defines the interface for executing commands
// This ensures all command executors follow the same pattern and signature
type CommandExecutor interface {
	Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error)
}

// CommandExecutorFunc is a function type that implements CommandExecutor
// This allows regular functions to be used as CommandExecutors without having to create a struct
type CommandExecutorFunc func(params map[string]interface{}, cfg *config.ConfigData) (string, error)

var _ CommandExecutor = CommandExecutorFunc(nil)

// Execute implements the CommandExecutor interface for CommandExecutorFunc
func (f CommandExecutorFunc) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	return f(params, cfg)
}
