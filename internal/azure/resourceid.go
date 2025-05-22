// Package azure provides Azure SDK integration for AKS MCP server.
package azure

import (
	"errors"
	"fmt"
	"strings"
)

// ResourceType represents an Azure resource type
type ResourceType string

// Known Azure resource types
const (
	ResourceTypeAKSCluster      ResourceType = "Microsoft.ContainerService/managedClusters"
	ResourceTypeVirtualNetwork  ResourceType = "Microsoft.Network/virtualNetworks"
	ResourceTypeRouteTable      ResourceType = "Microsoft.Network/routeTables"
	ResourceTypeSecurityGroup   ResourceType = "Microsoft.Network/networkSecurityGroups"
	ResourceTypeSubnet          ResourceType = "Microsoft.Network/virtualNetworks/subnets"
	ResourceTypeUnknown         ResourceType = "Unknown"
)

// AzureResourceID represents an Azure resource ID.
type AzureResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   ResourceType
	ResourceName   string
	SubResourceName string // Used for child resources like subnets
	FullID         string
}

// ParseAzureResourceID parses an Azure resource ID into its components.
func ParseAzureResourceID(resourceID string) (*AzureResourceID, error) {
	return ParseResourceID(resourceID)
}

// ParseResourceID parses an Azure resource ID into its components.
func ParseResourceID(resourceID string) (*AzureResourceID, error) {
	if resourceID == "" {
		return nil, errors.New("resource ID cannot be empty")
	}

	// Normalize the resource ID
	resourceID = strings.TrimSpace(resourceID)

	// Azure resource IDs have the format:
	// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/{resourceProvider}/{resourceType}/{resourceName}
	// Or for child resources:
	// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/{resourceProvider}/{resourceType}/{resourceName}/{childType}/{childName}
	segments := strings.Split(resourceID, "/")

	// A valid resourceID should have at least 9 segments (including empty segments at the start)
	if len(segments) < 9 {
		return nil, fmt.Errorf("invalid resource ID format: %s", resourceID)
	}

	// Check that the resource ID follows the expected pattern
	if segments[1] != "subscriptions" || segments[3] != "resourceGroups" || segments[5] != "providers" {
		return nil, fmt.Errorf("invalid resource ID format: %s", resourceID)
	}

	result := &AzureResourceID{
		SubscriptionID: segments[2],
		ResourceGroup:  segments[4],
		FullID:         resourceID,
	}

	// Determine the resource type and name based on the provider and resource type
	provider := segments[6]
	
	// Handle different resource types
	switch {
	case provider == "Microsoft.ContainerService" && segments[7] == "managedClusters" && len(segments) >= 9:
		result.ResourceType = ResourceTypeAKSCluster
		result.ResourceName = segments[8]
	
	case provider == "Microsoft.Network" && segments[7] == "virtualNetworks" && len(segments) >= 9:
		// Check if this is a subnet (child resource of VNet)
		if len(segments) >= 11 && segments[9] == "subnets" {
			result.ResourceType = ResourceTypeSubnet
			result.ResourceName = segments[8]
			result.SubResourceName = segments[10]
		} else {
			result.ResourceType = ResourceTypeVirtualNetwork
			result.ResourceName = segments[8]
		}
	
	case provider == "Microsoft.Network" && segments[7] == "routeTables" && len(segments) >= 9:
		result.ResourceType = ResourceTypeRouteTable
		result.ResourceName = segments[8]
	
	case provider == "Microsoft.Network" && segments[7] == "networkSecurityGroups" && len(segments) >= 9:
		result.ResourceType = ResourceTypeSecurityGroup
		result.ResourceName = segments[8]
	
	default:
		// For unsupported or unknown resource types, we'll still try to extract the basic info
		if len(segments) >= 9 {
			result.ResourceType = ResourceType(fmt.Sprintf("%s/%s", provider, segments[7]))
			result.ResourceName = segments[8]
			// If there's a sub-resource and it has a name
			if len(segments) >= 11 {
				result.SubResourceName = segments[10]
			}
		} else {
			result.ResourceType = ResourceTypeUnknown
		}
	}

	return result, nil
}

// IsAKSCluster returns true if the resource is an AKS cluster.
func (r *AzureResourceID) IsAKSCluster() bool {
	return r.ResourceType == ResourceTypeAKSCluster
}

// IsVirtualNetwork returns true if the resource is a virtual network.
func (r *AzureResourceID) IsVirtualNetwork() bool {
	return r.ResourceType == ResourceTypeVirtualNetwork
}

// IsRouteTable returns true if the resource is a route table.
func (r *AzureResourceID) IsRouteTable() bool {
	return r.ResourceType == ResourceTypeRouteTable
}

// IsSecurityGroup returns true if the resource is a network security group.
func (r *AzureResourceID) IsSecurityGroup() bool {
	return r.ResourceType == ResourceTypeSecurityGroup
}

// IsSubnet returns true if the resource is a subnet.
func (r *AzureResourceID) IsSubnet() bool {
	return r.ResourceType == ResourceTypeSubnet
}
