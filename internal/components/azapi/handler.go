package azapi

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/azure-api-mcp/pkg/azcli"
	"github.com/mark3labs/mcp-go/mcp"
)

func AzApiHandler(azClient azcli.Client, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			errMsg := fmt.Sprintf("arguments must be a map[string]interface{}, got %T", req.Params.Arguments)
			logger.Errorf("AzApiHandler: %s", errMsg)
			return mcp.NewToolResultError(errMsg), nil
		}

		cliCommand, ok := args["cli_command"].(string)
		if !ok {
			errMsg := "missing or invalid 'cli_command' parameter"
			logger.Errorf("AzApiHandler: %s", errMsg)
			return mcp.NewToolResultError(errMsg), nil
		}

		timeout := time.Duration(cfg.Timeout) * time.Second
		if timeoutParam, ok := args["timeout"].(float64); ok && timeoutParam > 0 {
			timeout = time.Duration(timeoutParam) * time.Second
		}

		cmdCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		logger.Debugf("AzApiHandler: Executing Azure CLI command: %s", cliCommand)

		result, err := azClient.ExecuteCommand(cmdCtx, cliCommand)
		if err != nil {
			errMsg := fmt.Sprintf("failed to execute Azure CLI command: %v", err)
			logger.Errorf("AzApiHandler: %s", errMsg)
			return mcp.NewToolResultError(errMsg), nil
		}

		if result.Error != "" {
			errMsg := fmt.Sprintf("Azure CLI command failed: %s", result.Error)
			logger.Errorf("AzApiHandler: %s", errMsg)
			return mcp.NewToolResultError(errMsg), nil
		}

		logger.Debugf("AzApiHandler: Command completed successfully")

		return mcp.NewToolResultText(string(result.Output)), nil
	}
}
