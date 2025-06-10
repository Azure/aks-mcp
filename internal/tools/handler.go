package tools

import (
	"context"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandlerFunc is a function that processes tool requests
// Deprecated: Use CommandExecutorFunc instead
type ToolHandlerFunc func(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error)

// CreateToolHandler creates an adapter that converts CommandExecutor to the format expected by MCP server
func CreateToolHandler(executor CommandExecutor, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := executor.Execute(req.Params.Arguments, cfg)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
