// Package registry provides a tool registry for AKS MCP server.
package registry

import (
	"github.com/azure/aks-mcp/internal/handlers"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterAllTools registers all tools with the registry.
func (r *ToolRegistry) RegisterAllTools() {
	// Register cluster tools
	r.registerClusterTools()
	
	// Register network tools
	r.registerNetworkTools()
	
	// Register other tool categories as needed
}

// registerClusterTools registers all tools related to AKS clusters.
func (r *ToolRegistry) registerClusterTools() {
	// Register get_cluster_info tool
	r.RegisterTool(
		"get_cluster_info",
		mcp.NewTool(
			"get_cluster_info",
			mcp.WithDescription("Get information about the AKS cluster"),
		),
		handlers.GetClusterInfoHandler(r.GetResourceID(), r.GetClient(), r.GetCache()),
		CategoryCluster,
	)
}

// registerNetworkTools registers all tools related to networking.
func (r *ToolRegistry) registerNetworkTools() {
	// Register get_vnet_info tool
	r.RegisterTool(
		"get_vnet_info",
		mcp.NewTool(
			"get_vnet_info",
			mcp.WithDescription("Get information about the VNet used by the AKS cluster"),
		),
		handlers.GetVNetInfoHandler(r.GetResourceID(), r.GetClient(), r.GetCache()),
		CategoryNetwork,
	)
	
	// Register get_route_table_info tool
	r.RegisterTool(
		"get_route_table_info",
		mcp.NewTool(
			"get_route_table_info",
			mcp.WithDescription("Get information about the route tables used by the AKS cluster"),
		),
		handlers.GetRouteTableInfoHandler(r.GetResourceID(), r.GetClient(), r.GetCache()),
		CategoryNetwork,
	)
	
	// Register get_nsg_info tool
	r.RegisterTool(
		"get_nsg_info",
		mcp.NewTool(
			"get_nsg_info",
			mcp.WithDescription("Get information about the network security groups used by the AKS cluster"),
		),
		handlers.GetNSGInfoHandler(r.GetResourceID(), r.GetClient(), r.GetCache()),
		CategoryNetwork,
	)
}
