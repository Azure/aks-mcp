// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"fmt"

	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetClusterInfoHandler returns a handler for the get_cluster_info tool.
// It can handle both single-cluster and multi-cluster cases based on the configuration.
func GetClusterInfoHandler(client *azure.AzureClient, cache *azure.AzureCache, cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var clusterResourceID *azure.AzureResourceID
		var err error

		// Determine which resource ID to use based on the configuration
		if cfg.SingleClusterMode {
			// Use the pre-configured resource ID for single-cluster mode
			clusterResourceID = cfg.ResourceID
		} else {
			// For multi-cluster mode, extract parameters from the request
			subscriptionID, _ := request.GetArguments()["subscription_id"].(string)
			resourceGroup, _ := request.GetArguments()["resource_group"].(string)
			clusterName, _ := request.GetArguments()["cluster_name"].(string)

			// Validate required parameters
			if subscriptionID == "" || resourceGroup == "" || clusterName == "" {
				return nil, fmt.Errorf("missing required parameters: subscription_id, resource_group, and cluster_name")
			}

			// Create a temporary resource ID for this request
			clusterResourceID = &azure.AzureResourceID{
				SubscriptionID: subscriptionID,
				ResourceGroup:  resourceGroup,
				ResourceName:   clusterName,
				ResourceType:   azure.ResourceTypeAKSCluster,
				FullID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
					subscriptionID, resourceGroup, clusterName),
			}
		}

		// Get the cluster from Azure using the appropriate resource ID
		cluster, err := getClusterFromCacheOrFetch(ctx, clusterResourceID, client, cache)
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
