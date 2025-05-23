// Package resourcehelpers provides helper functions for working with Azure resources in AKS MCP server.
package resourcehelpers

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/azure/aks-mcp/internal/azure"
)

// ResourceType represents the type of Azure resource.
type ResourceType string

const (
	// ResourceTypeVirtualNetwork represents a virtual network resource.
	ResourceTypeVirtualNetwork ResourceType = "VirtualNetwork"
	// ResourceTypeSubnet represents a subnet resource.
	ResourceTypeSubnet ResourceType = "Subnet"
	// ResourceTypeRouteTable represents a route table resource.
	ResourceTypeRouteTable ResourceType = "RouteTable"
	// ResourceTypeNetworkSecurityGroup represents a network security group resource.
	ResourceTypeNetworkSecurityGroup ResourceType = "NetworkSecurityGroup"
)

// GetAllNetworkResourcesFromAKS retrieves all network-related resources associated with an AKS cluster.
// This function consolidates the calls to individual resource helper functions.
// Returns a map of resource type to resource ID.
func GetAllNetworkResourcesFromAKS(
	ctx context.Context,
	cluster *armcontainerservice.ManagedCluster,
	client *azure.AzureClient,
	cache *azure.AzureCache,
) (map[ResourceType]string, error) {
	result := make(map[ResourceType]string)

	// Get VNet ID
	vnetID, err := GetVNetIDFromAKS(ctx, cluster, client, cache)
	if err == nil && vnetID != "" {
		result[ResourceTypeVirtualNetwork] = vnetID
	}

	// Get Subnet ID from agent pool profiles
	subnetID := ""
	if cluster.Properties != nil && cluster.Properties.AgentPoolProfiles != nil {
		for _, pool := range cluster.Properties.AgentPoolProfiles {
			if pool.VnetSubnetID != nil {
				subnetID = *pool.VnetSubnetID
				result[ResourceTypeSubnet] = subnetID
				break
			}
		}
	}

	// Get NSG ID
	nsgID, err := GetNSGIDFromAKS(ctx, cluster, client, cache)
	if err == nil && nsgID != "" {
		result[ResourceTypeNetworkSecurityGroup] = nsgID
	}

	// Get Route Table ID
	routeTableID, err := GetRouteTableIDFromAKS(ctx, cluster, client, cache)
	if err == nil && routeTableID != "" {
		result[ResourceTypeRouteTable] = routeTableID
	}

	return result, nil
}

// GetResourceByTypeFromAKS retrieves a specific resource from an AKS cluster based on the resource type.
// This is a convenience wrapper around the individual resource getter functions.
func GetResourceByTypeFromAKS(
	ctx context.Context,
	resourceType ResourceType,
	cluster *armcontainerservice.ManagedCluster,
	client *azure.AzureClient,
	cache *azure.AzureCache,
) (string, error) {
	switch resourceType {
	case ResourceTypeVirtualNetwork:
		return GetVNetIDFromAKS(ctx, cluster, client, cache)
	case ResourceTypeNetworkSecurityGroup:
		return GetNSGIDFromAKS(ctx, cluster, client, cache)
	case ResourceTypeRouteTable:
		return GetRouteTableIDFromAKS(ctx, cluster, client, cache)
	default:
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}
