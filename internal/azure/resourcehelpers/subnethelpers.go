// Package resourcehelpers provides helper functions for working with Azure resources.
package resourcehelpers

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/azure/aks-mcp/internal/azure"
)

// GetSubnetIDFromAKS extracts subnet ID from an AKS cluster.
// It tries to get the subnet ID from the agent pool profiles first.
// If not found, it will try to find the VNet in the node resource group, and then
// look for a subnet with the name 'aks-subnet' or use the first subnet if not found.
func GetSubnetIDFromAKS(ctx context.Context, cluster *armcontainerservice.ManagedCluster, client *azure.AzureClient, cache *azure.AzureCache) (string, error) {
	// First, try to get subnet ID directly from agent pool profiles
	if cluster.Properties != nil && cluster.Properties.AgentPoolProfiles != nil {
		for _, pool := range cluster.Properties.AgentPoolProfiles {
			if pool.VnetSubnetID != nil && *pool.VnetSubnetID != "" {
				return *pool.VnetSubnetID, nil
			}
		}
	}

	// If we couldn't find a subnet ID in the agent pool profiles, try to find the VNet first
	vnetID, err := GetVNetIDFromAKS(ctx, cluster, client, cache)
	if err != nil || vnetID == "" {
		return "", fmt.Errorf("could not find VNet for AKS cluster: %v", err)
	}

	// Parse VNet ID to extract subscription, resource group, and name
	vnetResourceID, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return "", fmt.Errorf("could not parse VNet ID: %v", err)
	}

	// Get the VNet details to list subnets
	vnet, err := client.GetVirtualNetwork(ctx, 
		vnetResourceID.SubscriptionID, 
		vnetResourceID.ResourceGroup, 
		vnetResourceID.ResourceName)
	if err != nil {
		return "", fmt.Errorf("could not get VNet details: %v", err)
	}

	// If VNet has no subnets, return error
	if vnet.Properties == nil || vnet.Properties.Subnets == nil || len(vnet.Properties.Subnets) == 0 {
		return "", fmt.Errorf("VNet has no subnets")
	}

	// First try to find a subnet with name "aks-subnet"
	for _, subnet := range vnet.Properties.Subnets {
		if subnet.Name != nil && *subnet.Name == "aks-subnet" {
			if subnet.ID != nil {
				return *subnet.ID, nil
			}
		}
	}

	// If no "aks-subnet" found, use the first subnet
	if vnet.Properties.Subnets[0].ID != nil {
		return *vnet.Properties.Subnets[0].ID, nil
	}

	return "", fmt.Errorf("could not find a valid subnet in the VNet")
}
