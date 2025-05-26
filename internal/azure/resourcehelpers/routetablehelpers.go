// Package resourcehelpers provides helper functions for working with Azure resources in AKS MCP server.
package resourcehelpers

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/azure/aks-mcp/internal/azure"
)

// GetRouteTableIDFromAKS attempts to find a route table associated with an AKS cluster.
// It first checks if a subnet is associated with the AKS cluster, then looks for a route table attached to that subnet.
// If no route table is found, it returns an empty string.
func GetRouteTableIDFromAKS(
	ctx context.Context,
	cluster *armcontainerservice.ManagedCluster,
	client *azure.AzureClient,
	cache *azure.AzureCache,
) (string, error) {
	// Ensure the cluster is valid
	if cluster == nil || cluster.Properties == nil {
		return "", fmt.Errorf("invalid cluster or cluster properties")
	}

	// Get subnet ID using the helper function which handles cases when VnetSubnetID is not set
	subnetID, err := GetSubnetIDFromAKS(ctx, cluster, client, cache)
	if err != nil || subnetID == "" {
		return "", fmt.Errorf("no subnet found for AKS cluster: %v", err)
	}

	// Check cache first
	cacheKey := fmt.Sprintf("subnet-routetable:%s", subnetID)
	if cachedID, found := cache.Get(cacheKey); found {
		if routeTableID, ok := cachedID.(string); ok {
			return routeTableID, nil
		}
	}

	// Parse subnet ID to get subscription, resource group, vnet name and subnet name
	parsedSubnetID, err := azure.ParseResourceID(subnetID)
	if err != nil {
		return "", fmt.Errorf("failed to parse subnet ID: %v", err)
	}

	if !parsedSubnetID.IsSubnet() {
		return "", fmt.Errorf("invalid subnet ID format: %s", subnetID)
	}

	// Get the subscription ID from the subnet ID
	subscriptionID := parsedSubnetID.SubscriptionID
	resourceGroup := parsedSubnetID.ResourceGroup
	vnetName := parsedSubnetID.ResourceName
	subnetName := parsedSubnetID.SubResourceName

	// Get subnet details to find attached route table
	clients, err := client.GetOrCreateClientsForSubscription(subscriptionID)
	if err != nil {
		return "", fmt.Errorf("failed to get clients for subscription %s: %v", subscriptionID, err)
	}

	subnet, err := clients.SubnetsClient.Get(ctx, resourceGroup, vnetName, subnetName, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get subnet details: %v", err)
	}

	// Check if the subnet has a route table attached
	if subnet.Properties == nil || subnet.Properties.RouteTable == nil || subnet.Properties.RouteTable.ID == nil {
		return "", fmt.Errorf("no route table attached to subnet %s", subnetName)
	}

	routeTableID := *subnet.Properties.RouteTable.ID

	// Store in cache
	cache.Set(cacheKey, routeTableID)

	return routeTableID, nil
}
