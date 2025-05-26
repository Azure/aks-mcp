// Package registry provides a tool registry for AKS MCP server.
package registry

import (
	"github.com/azure/aks-mcp/internal/handlers"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerNetworkTools registers all tools related to networking.
func (r *ToolRegistry) registerNetworkTools() {
	cfg := r.GetConfig()

	var vnetTool mcp.Tool
	if cfg.SingleClusterMode {
		vnetTool = mcp.NewTool(
			"get_vnet_info",
			mcp.WithDescription("Get information about the VNet used by the AKS cluster"),
		)
	} else {
		vnetTool = mcp.NewTool(
			"get_vnet_info",
			mcp.WithDescription("Get information about the VNet used by the AKS cluster"),
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
		)
	}
	// Register get_vnet_info tool
	r.RegisterTool(
		"get_vnet_info",
		vnetTool,
		handlers.GetVNetInfoHandler(r.GetClient(), r.GetCache(), cfg),
		CategoryNetwork,
	)

	var routeTableTool mcp.Tool
	if cfg.SingleClusterMode {
		routeTableTool = mcp.NewTool(
			"get_route_table_info",
			mcp.WithDescription("Get information about the route tables used by the AKS cluster"),
		)
	} else {
		routeTableTool = mcp.NewTool(
			"get_route_table_info",
			mcp.WithDescription("Get information about the route tables used by the AKS cluster"),
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
		)
	}
	// Register get_route_table_info tool
	r.RegisterTool(
		"get_route_table_info",
		routeTableTool,
		handlers.GetRouteTableInfoHandler(r.GetClient(), r.GetCache(), cfg),
		CategoryNetwork,
	)

	var nsgTool mcp.Tool
	if cfg.SingleClusterMode {
		nsgTool = mcp.NewTool(
			"get_nsg_info",
			mcp.WithDescription("Get information about the network security groups used by the AKS cluster"),
		)
	} else {
		nsgTool = mcp.NewTool(
			"get_nsg_info",
			mcp.WithDescription("Get information about the network security groups used by the AKS cluster"),
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
		)
	}
	// Register get_nsg_info tool
	r.RegisterTool(
		"get_nsg_info",
		nsgTool,
		handlers.GetNSGInfoHandler(r.GetClient(), r.GetCache(), cfg),
		CategoryNetwork,
	)

	// Create Subnet tool with parameters if needed
	var subnetTool mcp.Tool
	if cfg.SingleClusterMode {
		subnetTool = mcp.NewTool(
			"get_subnet_info",
			mcp.WithDescription("Get information about the subnets used by the AKS cluster"),
		)
	} else {
		subnetTool = mcp.NewTool(
			"get_subnet_info",
			mcp.WithDescription("Get information about the subnets used by the AKS cluster"),
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
		)
	}
	
	// Register get_subnet_info tool
	r.RegisterTool(
		"get_subnet_info",
		subnetTool,
		handlers.GetSubnetInfoHandler(r.GetClient(), r.GetCache(), cfg),
		CategoryNetwork,
	)
}
