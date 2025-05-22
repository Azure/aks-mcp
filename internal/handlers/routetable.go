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

// GetRouteTableInfoHandler returns a handler for the get_route_table_info tool.
func GetRouteTableInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		
		// Cache key for route table info
		cacheKey := fmt.Sprintf("routetable:%s", resourceID.FullID)
		
		// Check if route table info is in cache
		if cachedData, found := cache.Get(cacheKey); found {
			if routeTableInfo, ok := cachedData.(*models.RouteTableInfo); ok {
				jsonStr, err := formatJSON(routeTableInfo)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal route table info: %v", err)
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
		routeTableInfo := &models.RouteTableInfo{
			Name:          "aks-route-table",
			ResourceGroup: resourceID.ResourceGroup,
			Location:      "eastus", // Would come from actual cluster data
			ID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/routeTables/aks-route-table", 
				            resourceID.SubscriptionID, resourceID.ResourceGroup),
			Routes: []models.RouteInfo{
				{
					Name:              "default-route",
					AddressPrefix:     "0.0.0.0/0",
					NextHopType:       "VirtualAppliance",
					NextHopIPAddress:  "10.0.0.4",
					ProvisioningState: "Succeeded",
				},
				{
					Name:              "kubernetes-route",
					AddressPrefix:     "10.244.0.0/16",
					NextHopType:       "VirtualNetwork",
					ProvisioningState: "Succeeded",
				},
			},
			ProvisioningState: "Succeeded",
		}
		
		// Cache the result
		cache.Set(cacheKey, routeTableInfo)
		
		// Return the result
		jsonStr, err := formatJSON(routeTableInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal route table info: %v", err)
		}
		
		return mcp.NewToolResultText(jsonStr), nil
	}
}
