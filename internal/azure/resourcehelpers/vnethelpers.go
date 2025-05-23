// Package resourcehelpers provides helper functions for working with Azure resources in AKS MCP server.
package resourcehelpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/azure/aks-mcp/internal/azure"
)

// GetVNetIDFromAKS extracts the virtual network ID from an AKS cluster.
// It first checks the agent pool profiles for subnet IDs.
// If no subnet ID is found, it attempts to look up the VNet in the node resource group.
func GetVNetIDFromAKS(
	ctx context.Context,
	cluster *armcontainerservice.ManagedCluster,
	client *azure.AzureClient,
	cache *azure.AzureCache,
) (string, error) {
	// Ensure the cluster is valid
	if cluster == nil || cluster.Properties == nil {
		return "", fmt.Errorf("invalid cluster or cluster properties")
	}

	// First check: Look for subnet ID in agent pool profiles
	if cluster.Properties.AgentPoolProfiles != nil {
		for _, pool := range cluster.Properties.AgentPoolProfiles {
			if pool.VnetSubnetID != nil {
				// The subnet ID contains the VNet ID as its parent resource
				subnetID := *pool.VnetSubnetID
				// Parse the subnet ID to extract the VNet ID
				if parsed, err := azure.ParseResourceID(subnetID); err == nil && parsed.IsSubnet() {
					// Construct the VNet ID from the subnet ID
					vnetIDParts := strings.Split(subnetID, "/subnets/")
					if len(vnetIDParts) > 0 {
						return vnetIDParts[0], nil
					}
				}
				break
			}
		}
	}

	// Second check: Look for VNet in node resource group
	if cluster.Properties.NodeResourceGroup != nil {
		return findVNetInNodeResourceGroup(ctx, cluster, client, cache)
	}

	return "", fmt.Errorf("no virtual network found for AKS cluster")
}

// findVNetInNodeResourceGroup looks for a VNet in the node resource group that has
// a name prefix of "aks-vnet-". This is the naming convention used by AKS.
func findVNetInNodeResourceGroup(
	ctx context.Context,
	cluster *armcontainerservice.ManagedCluster,
	client *azure.AzureClient,
	cache *azure.AzureCache,
) (string, error) {
	// Get subscription ID and node resource group
	subscriptionID := getSubscriptionFromCluster(cluster)
	nodeResourceGroup := *cluster.Properties.NodeResourceGroup

	// Check cache first
	cacheKey := fmt.Sprintf("noderesourcegroup-vnet:%s:%s", subscriptionID, nodeResourceGroup)
	if cachedID, found := cache.Get(cacheKey); found {
		if vnetID, ok := cachedID.(string); ok && vnetID != "" {
			return vnetID, nil
		}
	}

	// List virtual networks in the node resource group
	clients, err := client.GetOrCreateClientsForSubscription(subscriptionID)
	if err != nil {
		return "", fmt.Errorf("failed to get clients for subscription %s: %v", subscriptionID, err)
	}

	pager := clients.VNetClient.NewListPager(nodeResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to list virtual networks in resource group %s: %v", nodeResourceGroup, err)
		}

		for _, vnet := range page.Value {
			// Check for VNet with prefix "aks-vnet-"
			if vnet.Name != nil && strings.HasPrefix(*vnet.Name, "aks-vnet-") {
				vnetID := *vnet.ID
				// Store in cache
				cache.Set(cacheKey, vnetID)
				return vnetID, nil
			}
		}
	}

	return "", fmt.Errorf("no suitable virtual network found in node resource group %s", nodeResourceGroup)
}

// getSubscriptionFromCluster extracts the subscription ID from the cluster's ID.
func getSubscriptionFromCluster(cluster *armcontainerservice.ManagedCluster) string {
	if cluster.ID == nil {
		return ""
	}

	parts := strings.Split(*cluster.ID, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "subscriptions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
