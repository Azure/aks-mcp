// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/azure/resourcehelpers"
	"github.com/azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetSubnetInfoHandler returns a handler for the get_subnet_info tool.
// It can handle both single-cluster and multi-cluster cases based on the configuration.
func GetSubnetInfoHandler(client *azure.AzureClient, cache *azure.AzureCache, cfg *config.Config) server.ToolHandlerFunc {
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
				FullID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
					subscriptionID, resourceGroup, clusterName),
			}
		}

		// Try to get cluster info first to extract subnet resources
		cluster, err := getClusterFromCacheOrFetch(ctx, clusterResourceID, client, cache)
		if err != nil {
			return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
		}
		
		// Use the resourcehelpers to get the subnet ID from the AKS cluster
		subnetID, err := resourcehelpers.GetSubnetIDFromAKS(ctx, cluster, client, cache)
		
		// If subnet information wasn't found, return an empty response with a log message
		if err != nil || subnetID == "" {
			message := "No subnet found for this AKS cluster"
			fmt.Printf("WARNING: %s: %v\n", message, err)
			return mcp.NewToolResultText(fmt.Sprintf(`{"message": "%s"}`, message)), nil
		}

		// Validate the subnet ID by trying to parse it
		_, err = azure.ParseResourceID(subnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse subnet ID: %v", err)
		}

		// Get the subnet from cache or fetch from Azure
		resource, err := getResourceByIDFromCacheOrFetch(ctx, subnetID, client, cache)
		if err != nil {
			return nil, fmt.Errorf("failed to get subnet details: %v", err)
		}
		
		subnet, ok := resource.(*armnetwork.Subnet)
		if !ok {
			return nil, fmt.Errorf("resource is not a Subnet")
		}
		
		// Return the raw ARM response
		jsonStr, err := formatJSON(subnet)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal subnet info: %v", err)
		}
		
		return mcp.NewToolResultText(jsonStr), nil
	}
}
