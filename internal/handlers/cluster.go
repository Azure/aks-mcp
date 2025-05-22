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

// GetClusterInfoHandler returns a handler for the get_cluster_info tool.
func GetClusterInfoHandler(resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Try to get from cache first
		cacheKey := fmt.Sprintf("cluster:%s", resourceID.FullID)
		if cachedData, found := cache.Get(cacheKey); found {
			if clusterInfo, ok := cachedData.(*models.ClusterInfo); ok {
				// We found it in cache
				jsonStr, err := formatJSON(clusterInfo)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal cluster info: %v", err)
				}
				return mcp.NewToolResultText(jsonStr), nil
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
		jsonStr, err := formatJSON(clusterInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster info: %v", err)
		}
		
		return mcp.NewToolResultText(jsonStr), nil
	}
}
