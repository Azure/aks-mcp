package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/mark3labs/mcp-go/mcp"
)

// extractToolContext extracts _tool_context from arguments and adds to context.
// It removes _tool_context from args to avoid passing it to the actual tool.
// This supports passing context via arguments for all transport modes (stdio/SSE/streamable-http).
func extractToolContext(ctx context.Context, args map[string]interface{}) context.Context {
	toolContextRaw, exists := args["_tool_context"]
	if !exists {
		return ctx
	}

	// Remove from arguments so tools don't see it
	delete(args, "_tool_context")

	// Extract context data
	toolContextMap, ok := toolContextRaw.(map[string]interface{})
	if !ok {
		return ctx
	}

	// Handle request_context as either string or map
	if requestContextMap, ok := toolContextMap["request_context"].(map[string]interface{}); ok {
		if jsonBytes, err := json.Marshal(requestContextMap); err == nil {
			ctx = context.WithValue(ctx, "request_context", string(jsonBytes))
		}
	} else if requestContextStr, ok := toolContextMap["request_context"].(string); ok && requestContextStr != "" {
		ctx = context.WithValue(ctx, "request_context", requestContextStr)
	}

	return ctx
}

// logToolCall logs the start of a tool call
func logToolCall(toolName string, arguments interface{}) {
	// Try to format as JSON for better readability
	if jsonBytes, err := json.Marshal(arguments); err == nil {
		logger.Debugf("\n>>> [%s] %s", toolName, string(jsonBytes))
	} else {
		logger.Debugf("\n>>> [%s] %v", toolName, arguments)
	}
}

// logToolResult logs the result or error of a tool call
func logToolResult(toolName string, result string, err error) {
	if err != nil {
		logger.Debugf("\n<<< [%s] ERROR: %v", toolName, err)
	} else if len(result) > 500 {
		logger.Debugf("\n<<< [%s] Result: %d bytes (truncated): %.500s...", toolName, len(result), result)
	} else {
		logger.Debugf("\n<<< [%s] Result: %s", toolName, result)
	}
}

// CreateToolHandler creates an adapter that converts CommandExecutor to the format expected by MCP server
func CreateToolHandler(executor CommandExecutor, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			err := fmt.Errorf("arguments must be a map[string]interface{}, got %T", req.Params.Arguments)
			// Track failed tool invocation
			if cfg.TelemetryService != nil {
				cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, "", false)
			}
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Extract and apply _tool_context from arguments
		ctx = extractToolContext(ctx, args)

		// Log tool call after extracting context to avoid logging sensitive information
		logToolCall(req.Params.Name, args)

		result, err := executor.Execute(ctx, args, cfg)
		if cfg.TelemetryService != nil {
			cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, getOperationValue(args), err == nil)
		}

		logToolResult(req.Params.Name, result, err)

		if err != nil {
			// Include command output (often stderr) in the error for context
			if result != "" {
				return mcp.NewToolResultError(fmt.Sprintf("%s\n%s", err.Error(), result)), nil
			}
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

// CreateResourceHandler creates an adapter that converts ResourceHandler to the format expected by MCP server
func CreateResourceHandler(handler ResourceHandler, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			err := fmt.Errorf("arguments must be a map[string]interface{}, got %T", req.Params.Arguments)
			// Track failed tool invocation
			if cfg.TelemetryService != nil {
				cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, "", false)
			}
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Extract and apply _tool_context from arguments
		ctx = extractToolContext(ctx, args)

		// Log tool call after extracting context to avoid logging sensitive information
		logToolCall(req.Params.Name, args)

		result, err := handler.Handle(ctx, args, cfg)

		// Track tool invocation with minimal data
		if cfg.TelemetryService != nil {
			cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, getOperationValue(args), err == nil)
		}

		logToolResult(req.Params.Name, result, err)

		if err != nil {
			// Include handler output in the error message for better diagnostics
			if result != "" {
				return mcp.NewToolResultError(fmt.Sprintf("%s\n%s", err.Error(), result)), nil
			}
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func getOperationValue(args map[string]interface{}) string {
	if op, _ := args["operation"].(string); op != "" {
		return op
	}
	if action, _ := args["action"].(string); action != "" {
		return action
	}
	return ""
}
