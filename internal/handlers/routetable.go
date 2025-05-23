// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/azure/resourcehelpers"
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

		// Use the resourcehelpers to get the route table ID from the AKS cluster
		routeTableID, err := resourcehelpers.GetRouteTableIDFromAKS(ctx, cluster, client, cache)

		// If we didn't find a route table ID, return an empty response with a log message
		if err != nil || routeTableID == "" {
			message := "No route table found for this AKS cluster"
			fmt.Printf("WARNING: %s: %v\n", message, err)
			return mcp.NewToolResultText(fmt.Sprintf(`{"message": "%s"}`, message)), nil
		}

		// Parse the route table ID to get the subscription, resource group, and name
		rtResourceID, err := azure.ParseResourceID(routeTableID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse route table ID: %v", err)
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
