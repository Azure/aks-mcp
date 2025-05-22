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

// GetNSGInfoHandler returns a handler for the get_nsg_info tool.
func GetNSGInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		
		// Cache key for NSG info
		cacheKey := fmt.Sprintf("nsg:%s", resourceID.FullID)
		
		// Check if NSG info is in cache
		if cachedData, found := cache.Get(cacheKey); found {
			if nsgInfo, ok := cachedData.(*models.NSGInfo); ok {
				jsonStr, err := formatJSON(nsgInfo)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal NSG info: %v", err)
				}
				return mcp.NewToolResultText(jsonStr), nil
			}
		}
		
		// Not in cache, try to fetch data from Azure
		// We'll first get the AKS cluster to find network information
		_, err := client.GetAKSCluster(ctx, resourceID.SubscriptionID, resourceID.ResourceGroup, resourceID.ResourceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
		}
		
		// For now, we'll return a placeholder response
		nsgInfo := &models.NSGInfo{
			Name:          "aks-nsg",
			ResourceGroup: resourceID.ResourceGroup,
			Location:      "eastus", // Would come from actual cluster data
			ID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/aks-nsg", 
				           resourceID.SubscriptionID, resourceID.ResourceGroup),
			SecurityRules: []models.NSGRule{
				{
					Name:                     "allow-ssh",
					Protocol:                 "Tcp",
					SourceAddressPrefix:      "*",
					SourcePortRange:          "*",
					DestinationAddressPrefix: "*",
					DestinationPortRange:     "22",
					Access:                   "Allow",
					Priority:                 100,
					Direction:                "Inbound",
					ProvisioningState:        "Succeeded",
				},
				{
					Name:                     "allow-apiserver",
					Protocol:                 "Tcp",
					SourceAddressPrefix:      "*",
					SourcePortRange:          "*",
					DestinationAddressPrefix: "*",
					DestinationPortRange:     "443",
					Access:                   "Allow",
					Priority:                 110,
					Direction:                "Inbound",
					ProvisioningState:        "Succeeded",
				},
			},
			DefaultSecurityRules: []models.NSGRule{
				{
					Name:                     "AllowVnetInBound",
					Protocol:                 "*",
					SourceAddressPrefix:      "VirtualNetwork",
					SourcePortRange:          "*",
					DestinationAddressPrefix: "VirtualNetwork",
					DestinationPortRange:     "*",
					Access:                   "Allow",
					Priority:                 65000,
					Direction:                "Inbound",
					ProvisioningState:        "Succeeded",
				},
				{
					Name:                     "DenyAllInBound",
					Protocol:                 "*",
					SourceAddressPrefix:      "*",
					SourcePortRange:          "*",
					DestinationAddressPrefix: "*",
					DestinationPortRange:     "*",
					Access:                   "Deny",
					Priority:                 65500,
					Direction:                "Inbound",
					ProvisioningState:        "Succeeded",
				},
			},
			ProvisioningState: "Succeeded",
		}
		
		// Cache the result
		cache.Set(cacheKey, nsgInfo)
		
		// Return the result
		jsonStr, err := formatJSON(nsgInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal NSG info: %v", err)
		}
		
		return mcp.NewToolResultText(jsonStr), nil
	}
}
