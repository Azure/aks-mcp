// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/azure/aks-mcp/internal/azure"
)

// formatJSON formats the given object as JSON with indentation.
func formatJSON(obj interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %v", err)
	}
	return string(jsonBytes), nil
}

// getClusterFromCacheOrFetch retrieves an AKS cluster from cache or fetches it from Azure.
func getClusterFromCacheOrFetch(ctx context.Context, resourceID *azure.AzureResourceID, client *azure.AzureClient, cache *azure.AzureCache) (*armcontainerservice.ManagedCluster, error) {
	// Generate cache key for the cluster
	cacheKey := fmt.Sprintf("akscluster:%s", resourceID.FullID)
	
	// Try to get from cache first
	if cachedData, found := cache.Get(cacheKey); found {
		if cluster, ok := cachedData.(*armcontainerservice.ManagedCluster); ok {
			return cluster, nil
		}
	}
	
	// Not in cache, so fetch from Azure
	cluster, err := client.GetAKSCluster(ctx, resourceID.SubscriptionID, resourceID.ResourceGroup, resourceID.ResourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
	}
	
	// Add to cache
	cache.Set(cacheKey, cluster)
	
	return cluster, nil
}

// getResourceByIDFromCacheOrFetch retrieves any Azure resource by its ID from cache or fetches it from Azure.
func getResourceByIDFromCacheOrFetch(ctx context.Context, resourceID string, client *azure.AzureClient, cache *azure.AzureCache) (interface{}, error) {
	// Generate cache key for the resource
	cacheKey := fmt.Sprintf("resource:%s", resourceID)
	
	// Try to get from cache first
	if cachedData, found := cache.Get(cacheKey); found {
		return cachedData, nil
	}
	
	// Not in cache, so fetch from Azure
	resource, err := client.GetResourceByID(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %v", err)
	}
	
	// Add to cache
	cache.Set(cacheKey, resource)
	
	return resource, nil
}
