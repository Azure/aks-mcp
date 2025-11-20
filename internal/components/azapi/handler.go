package azapi

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/azure-api-mcp/pkg/azcli"
	"github.com/mark3labs/mcp-go/mcp"
)

func AzApiHandler(azClient azcli.Client, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("arguments must be a map[string]interface{}, got %T", req.Params.Arguments)), nil
		}

		cliCommand, ok := args["cli_command"].(string)
		if !ok {
			return mcp.NewToolResultError("missing or invalid 'cli_command' parameter"), nil
		}

		timeout := time.Duration(cfg.Timeout) * time.Second
		if timeoutParam, ok := args["timeout"].(float64); ok && timeoutParam > 0 {
			timeout = time.Duration(timeoutParam) * time.Second
		}

		cmdCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		result, err := azClient.ExecuteCommand(cmdCtx, cliCommand)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to execute Azure CLI command: %v", err)), nil
		}

		if result.Error != "" {
			return mcp.NewToolResultError(fmt.Sprintf("Azure CLI command failed: %s", result.Error)), nil
		}

		return mcp.NewToolResultText(string(result.Output)), nil
	}
}
