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

// GetVNetInfoHandler returns a handler for the get_vnet_info tool.
func GetVNetInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Try to get cluster info first to extract VNet info
		cluster, err := getClusterFromCacheOrFetch(ctx, resourceID, client, cache)
		if err != nil {
			return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
		}

		// Extract network resources from the cluster
		networkResources := azure.ExtractNetworkProfileFromAKS(cluster)

		// Get VNet ID from network resources
		vnetID, vnetFound := networkResources[azure.ResourceTypeVirtualNetwork]

		// If VNet information wasn't found, return an empty response with a log message
		if !vnetFound || vnetID == "" {
			message := "No virtual network found for this AKS cluster"
			fmt.Printf("WARNING: %s\n", message)
			return mcp.NewToolResultText(fmt.Sprintf(`{"message": "%s"}`, message)), nil
		}

		// Parse the VNet ID to get the subscription, resource group, and name
		vnetResourceID, err := azure.ParseResourceID(vnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse VNet ID: %v", err)
		}

		// Check if VNet is in cache
		cacheKey := fmt.Sprintf("vnet:%s", vnetID)

		if cachedData, found := cache.Get(cacheKey); found {
			if vnet, ok := cachedData.(*armnetwork.VirtualNetwork); ok {
				// Return the cached VNet directly
				jsonStr, err := formatJSON(vnet)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal VNet info: %v", err)
				}

				return mcp.NewToolResultText(jsonStr), nil
			}
		}

		// Not in cache, so get the VNet from Azure
		vnet, err := client.GetVirtualNetwork(ctx, vnetResourceID.SubscriptionID, vnetResourceID.ResourceGroup, vnetResourceID.ResourceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get VNet details: %v", err)
		}

		// Add to cache
		cache.Set(cacheKey, vnet)

		// Return the raw ARM response
		jsonStr, err := formatJSON(vnet)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal VNet info: %v", err)
		}

		return mcp.NewToolResultText(jsonStr), nil
	}
}
