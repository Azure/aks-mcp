package tools

import (
	"context"

	"github.com/Azure/aks-mcp/internal/config"
)

// CommandExecutor defines the interface for executing CLI commands
// This ensures all command executors follow the same pattern and signature
type CommandExecutor interface {
	Execute(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error)
}

// CommandExecutorFunc is a function type that implements CommandExecutor
// This allows regular functions to be used as CommandExecutors without having to create a struct
type CommandExecutorFunc func(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error)

var _ CommandExecutor = CommandExecutorFunc(nil)

// Execute implements the CommandExecutor interface for CommandExecutorFunc
func (f CommandExecutorFunc) Execute(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	return f(ctx, params, cfg)
}

// ResourceHandler defines the interface for handling Azure SDK-based resource operations
// This interface is semantically different from CommandExecutor as it handles API calls rather than CLI commands
type ResourceHandler interface {
	Handle(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error)
}

// ResourceHandlerFunc is a function type that implements ResourceHandler
// This allows regular functions to be used as ResourceHandlers without having to create a struct
type ResourceHandlerFunc func(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error)

var _ ResourceHandler = ResourceHandlerFunc(nil)

// Handle implements the ResourceHandler interface for ResourceHandlerFunc
func (f ResourceHandlerFunc) Handle(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	return f(ctx, params, cfg)
}
