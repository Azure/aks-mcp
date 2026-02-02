package network

import (
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
)

// NetworkResourceType defines the type of network resource
type NetworkResourceType string

const (
	ResourceTypeAll             NetworkResourceType = "all"
	ResourceTypeVNet            NetworkResourceType = "vnet"
	ResourceTypeNSG             NetworkResourceType = "nsg"
	ResourceTypeRouteTable      NetworkResourceType = "route_table"
	ResourceTypeSubnet          NetworkResourceType = "subnet"
	ResourceTypeLoadBalancer    NetworkResourceType = "load_balancer"
	ResourceTypePrivateEndpoint NetworkResourceType = "private_endpoint"
)

// RegisterAksNetworkResources registers the network resources tool
func RegisterAksNetworkResources() mcp.Tool {
	description := `Unified tool for getting Azure network resource information used by AKS clusters.

Supported resource types:
- all: Get information about all network resources
- vnet: Get Virtual Network information
- nsg: Get Network Security Group information
- route_table: Get Route Table information
- subnet: Get Subnet information
- load_balancer: Get Load Balancer information
- private_endpoint: Get Private Endpoint information (private clusters only)

Examples:
- Get all network resources: resource_type="all"
- Get VNet info: resource_type="vnet"
- Get NSG info: resource_type="nsg"`

	return mcp.NewTool("aks_network_resources",
		mcp.WithDescription(description),
		mcp.WithTitleAnnotation("Azure Network Resources"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("resource_type",
			mcp.Required(),
			mcp.Description("The type of network resource to query"),
		),
		mcp.WithString("subscription_id",
			mcp.Description("Azure Subscription ID"),
			mcp.Required(),
		),
		mcp.WithString("resource_group",
			mcp.Description("Azure Resource Group containing the AKS cluster"),
			mcp.Required(),
		),
		mcp.WithString("cluster_name",
			mcp.Description("Name of the AKS cluster"),
			mcp.Required(),
		),
		mcp.WithString("filters",
			mcp.Description("Optional filters for the query"),
		),
	)
}

// ValidateNetworkResourceType checks if the resource type is supported
func ValidateNetworkResourceType(resourceType string) bool {
	supportedTypes := []string{
		string(ResourceTypeAll), string(ResourceTypeVNet), string(ResourceTypeNSG),
		string(ResourceTypeRouteTable), string(ResourceTypeSubnet),
		string(ResourceTypeLoadBalancer), string(ResourceTypePrivateEndpoint),
	}

	return slices.Contains(supportedTypes, resourceType)
}

// GetSupportedNetworkResourceTypes returns all supported resource types
func GetSupportedNetworkResourceTypes() []string {
	return []string{
		string(ResourceTypeAll), string(ResourceTypeVNet), string(ResourceTypeNSG),
		string(ResourceTypeRouteTable), string(ResourceTypeSubnet),
		string(ResourceTypeLoadBalancer), string(ResourceTypePrivateEndpoint),
	}
}
