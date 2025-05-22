// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"fmt"

	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetVNetInfoHandler returns a handler for the get_vnet_info tool.
func GetVNetInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		
		// Try to get cluster info first
		cacheKey := fmt.Sprintf("cluster:%s", resourceID.FullID)
		var cluster *models.ClusterInfo
		
		// Check if cluster info is in cache
		if cachedData, found := cache.Get(cacheKey); found {
			if clusterInfo, ok := cachedData.(*models.ClusterInfo); ok {
				cluster = clusterInfo
			}
		}
		
		// If not in cache, get cluster info first
		if cluster == nil {
			// Get cluster info to extract network properties
			aksCluster, err := client.GetAKSCluster(ctx, resourceID.SubscriptionID, resourceID.ResourceGroup, resourceID.ResourceName)
			if err != nil {
				return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
			}
			
			// For demonstration, we'll create a simple response with VNet information
			vnetInfo := &models.VNetInfo{
				Name:          "aks-vnet",
				ResourceGroup: resourceID.ResourceGroup,
				Location:      *aksCluster.Location,
				AddressSpace:  []string{"10.0.0.0/16"},
				Subnets: []models.SubnetInfo{
					{
						Name:           "aks-subnet",
						AddressPrefix:  "10.0.0.0/24",
						ProvisioningState: "Succeeded",
					},
				},
				ProvisioningState: "Succeeded",
			}
			
			// In a real implementation, you would use aksCluster.Properties.NetworkProfile 
			// to find the VNet ID and then query details with client.GetVirtualNetwork
			
			// Return the result
			jsonStr, err := formatJSON(vnetInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal VNet info: %v", err)
			}
			
			return mcp.NewToolResultText(jsonStr), nil
		}
		
		// Just a placeholder response since we don't have actual VNet data
		vnetInfo := &models.VNetInfo{
			Name:          "aks-vnet",
			ResourceGroup: cluster.ResourceGroup,
			Location:      cluster.Location,
			ID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/aks-vnet", 
				            resourceID.SubscriptionID, resourceID.ResourceGroup),
			AddressSpace:  []string{"10.0.0.0/16"},
			Subnets: []models.SubnetInfo{
				{
					Name:              "aks-subnet",
					AddressPrefix:     "10.0.0.0/24",
					ProvisioningState: "Succeeded",
				},
			},
			ProvisioningState: "Succeeded",
		}
		
		// Return the result
		jsonStr, err := formatJSON(vnetInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal VNet info: %v", err)
		}
		
		return mcp.NewToolResultText(jsonStr), nil
	}
}
