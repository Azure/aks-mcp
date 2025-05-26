// Package resourcehelpers provides helper functions for working with Azure resources in AKS MCP server.
package resourcehelpers

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
