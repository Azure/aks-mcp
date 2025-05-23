// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/azure/aks-mcp/internal/azure"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetNSGInfoHandler returns a handler for the get_nsg_info tool.
func GetNSGInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Try to get cluster info first to extract network resources
		cluster, err := getClusterFromCacheOrFetch(ctx, resourceID, client, cache)
		if err != nil {
			return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
		}

		// Extract network resources from the cluster
		networkResources := azure.ExtractNetworkProfileFromAKS(cluster)

		// In a real-world scenario, we'd need to extract NSG information by looking up
		// the NSGs associated with the VNet subnet
		// For demonstration purposes, we'll try to use the VNet information
		// and check for an NSG in the subnet

		var nsgID string

		// Check if we have VNet or subnet info to find associated NSGs
		if subnetID, found := networkResources[azure.ResourceTypeSubnet]; found {
			// Parse subnet ID
			subnetResourceID, err := azure.ParseResourceID(subnetID)
			if err == nil {
				// Normally we would query the subnet to get its NSG ID
				// For now, we'll construct a plausible NSG ID
				nsgID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s-nsg",
					subnetResourceID.SubscriptionID, subnetResourceID.ResourceGroup, subnetResourceID.ResourceName)
			}
		}

		// If we didn't find an NSG ID, return an empty response with a log message
		if nsgID == "" {
			message := "No network security group found for this AKS cluster"
			fmt.Printf("WARNING: %s\n", message)
			return mcp.NewToolResultText(fmt.Sprintf(`{"message": "%s"}`, message)), nil
		}

		// Parse the NSG ID to get the subscription, resource group, and name
		nsgResourceID, err := azure.ParseResourceID(nsgID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse NSG ID: %v", err)
		}

		// Check if NSG is in cache
		cacheKey := fmt.Sprintf("nsg:%s", nsgID)

		if cachedData, found := cache.Get(cacheKey); found {
			if nsg, ok := cachedData.(*armnetwork.SecurityGroup); ok {
				// Return the cached NSG directly
				jsonStr, err := formatJSON(nsg)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal NSG info: %v", err)
				}

				return mcp.NewToolResultText(jsonStr), nil
			}
		}

		// Not in cache, so get the NSG from Azure
		nsg, err := client.GetNetworkSecurityGroup(ctx, nsgResourceID.SubscriptionID, nsgResourceID.ResourceGroup, nsgResourceID.ResourceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get NSG details: %v", err)
		}

		// Add to cache
		cache.Set(cacheKey, nsg)

		// Return the raw ARM response
		jsonStr, err := formatJSON(nsg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal NSG info: %v", err)
		}

		return mcp.NewToolResultText(jsonStr), nil
	}
}
