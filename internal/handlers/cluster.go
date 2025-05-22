// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"fmt"

	"github.com/azure/aks-mcp/internal/azure"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetClusterInfoHandler returns a handler for the get_cluster_info tool.
func GetClusterInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get the cluster from Azure
		cluster, err := getClusterFromCacheOrFetch(ctx, resourceID, client, cache)
		if err != nil {
			return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
		}
		
		// Return the ARM response directly as JSON
		jsonStr, err := formatJSON(cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster info: %v", err)
		}
		
		return mcp.NewToolResultText(jsonStr), nil
	}
}
