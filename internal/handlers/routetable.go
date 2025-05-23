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

// GetRouteTableInfoHandler returns a handler for the get_route_table_info tool.
func GetRouteTableInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Try to get cluster info first to extract network resources
		cluster, err := getClusterFromCacheOrFetch(ctx, resourceID, client, cache)
		if err != nil {
			return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
		}

		// Extract network resources from the cluster
		networkResources := azure.ExtractNetworkProfileFromAKS(cluster)

		// In a real-world scenario, we'd need to extract route table information by looking up
		// the route tables associated with the VNet subnet
		// For demonstration purposes, we'll try to use the VNet information
		// and check for a route table in the subnet

		var routeTableID string

		// Check if we have VNet or subnet info to find associated route tables
		if subnetID, found := networkResources[azure.ResourceTypeSubnet]; found {
			// Parse subnet ID
			subnetResourceID, err := azure.ParseResourceID(subnetID)
			if err == nil {
				// Normally we would query the subnet to get its route table ID
				// For now, we'll construct a plausible route table ID
				routeTableID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/routeTables/%s-rt",
					subnetResourceID.SubscriptionID, subnetResourceID.ResourceGroup, subnetResourceID.ResourceName)
			}
		}

		// If we didn't find a route table ID, return an empty response with a log message
		if routeTableID == "" {
			message := "No route table found for this AKS cluster"
			fmt.Printf("WARNING: %s\n", message)
			return mcp.NewToolResultText(fmt.Sprintf(`{"message": "%s"}`, message)), nil
		}

		// Check if route table is in cache
		cacheKey := fmt.Sprintf("routetable:%s", routeTableID)

		if cachedData, found := cache.Get(cacheKey); found {
			if rt, ok := cachedData.(*armnetwork.RouteTable); ok {
				// Return the cached route table directly
				jsonStr, err := formatJSON(rt)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal route table info: %v", err)
				}

				return mcp.NewToolResultText(jsonStr), nil
			}
		}

		// Not in cache, so try to fetch the route table
		// Parse the route table ID
		rtResourceID, err := azure.ParseResourceID(routeTableID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse route table ID: %v", err)
		}

		// Get route table from Azure using the correct subscription ID
		routeTable, err := client.GetRouteTable(ctx, rtResourceID.SubscriptionID, rtResourceID.ResourceGroup, rtResourceID.ResourceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get route table details: %v", err)
		}

		// Add to cache
		cache.Set(cacheKey, routeTable)

		// Return the raw ARM response
		jsonStr, err := formatJSON(routeTable)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal route table info: %v", err)
		}

		return mcp.NewToolResultText(jsonStr), nil
	}
}
