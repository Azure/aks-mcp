// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetClusterInfoHandler returns a handler for the get_cluster_info tool.
func GetClusterInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Try to get from cache first
		cacheKey := fmt.Sprintf("cluster:%s", resourceID.FullID)
		if cachedData, found := cache.Get(cacheKey); found {
			if clusterInfo, ok := cachedData.(*models.ClusterInfo); ok {
				// We found it in cache
				jsonBytes, err := json.MarshalIndent(clusterInfo, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to marshal cluster info: %v", err)
				}
				return mcp.NewToolResultText(string(jsonBytes)), nil
			}
		}
		
		// Not in cache, fetch from Azure
		cluster, err := client.GetAKSCluster(ctx, resourceID.SubscriptionID, resourceID.ResourceGroup, resourceID.ResourceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
		}
		
		// Convert to our model
		clusterInfo := &models.ClusterInfo{
			Name:                *cluster.Name,
			ResourceGroup:       resourceID.ResourceGroup,
			Location:            *cluster.Location,
			KubernetesVersion:   *cluster.Properties.KubernetesVersion,
			NodeResourceGroup:   *cluster.Properties.NodeResourceGroup,
			SubscriptionID:      resourceID.SubscriptionID,
			ResourceID:          resourceID.FullID,
		}
		
		// Set optional fields if they exist
		if cluster.Properties.NetworkProfile != nil {
			if cluster.Properties.NetworkProfile.NetworkPlugin != nil {
				clusterInfo.NetworkPlugin = string(*cluster.Properties.NetworkProfile.NetworkPlugin)
			}
			if cluster.Properties.NetworkProfile.NetworkPolicy != nil {
				clusterInfo.NetworkPolicy = string(*cluster.Properties.NetworkProfile.NetworkPolicy)
			}
		}
		
		if cluster.Properties.DNSPrefix != nil {
			clusterInfo.DNSPrefix = *cluster.Properties.DNSPrefix
		}
		
		if cluster.Properties.Fqdn != nil {
			clusterInfo.FQDN = *cluster.Properties.Fqdn
		}
		
		// Add agent pool profiles
		if cluster.Properties.AgentPoolProfiles != nil {
			for _, profile := range cluster.Properties.AgentPoolProfiles {
				if profile.Name != nil {
					clusterInfo.AgentPoolProfiles = append(clusterInfo.AgentPoolProfiles, *profile.Name)
				}
			}
		}
		
		// Cache the result
		cache.Set(cacheKey, clusterInfo)
		
		// Return the result
		jsonBytes, err := json.MarshalIndent(clusterInfo, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster info: %v", err)
		}
		
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

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
			jsonBytes, err := json.MarshalIndent(vnetInfo, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal VNet info: %v", err)
			}
			
			return mcp.NewToolResultText(string(jsonBytes)), nil
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
		jsonBytes, err := json.MarshalIndent(vnetInfo, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal VNet info: %v", err)
		}
		
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

// GetRouteTableInfoHandler returns a handler for the get_route_table_info tool.
func GetRouteTableInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		
		// Cache key for route table info
		cacheKey := fmt.Sprintf("routetable:%s", resourceID.FullID)
		
		// Check if route table info is in cache
		if cachedData, found := cache.Get(cacheKey); found {
			if routeTableInfo, ok := cachedData.(*models.RouteTableInfo); ok {
				jsonBytes, err := json.MarshalIndent(routeTableInfo, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to marshal route table info: %v", err)
				}
				return mcp.NewToolResultText(string(jsonBytes)), nil
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
		jsonBytes, err := json.MarshalIndent(routeTableInfo, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal route table info: %v", err)
		}
		
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

// GetNSGInfoHandler returns a handler for the get_nsg_info tool.
func GetNSGInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		
		// Cache key for NSG info
		cacheKey := fmt.Sprintf("nsg:%s", resourceID.FullID)
		
		// Check if NSG info is in cache
		if cachedData, found := cache.Get(cacheKey); found {
			if nsgInfo, ok := cachedData.(*models.NSGInfo); ok {
				jsonBytes, err := json.MarshalIndent(nsgInfo, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to marshal NSG info: %v", err)
				}
				return mcp.NewToolResultText(string(jsonBytes)), nil
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
		jsonBytes, err := json.MarshalIndent(nsgInfo, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal NSG info: %v", err)
		}
		
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}
