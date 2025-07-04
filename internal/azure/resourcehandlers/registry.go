package resourcehandlers

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Network-related tool registrations

// RegisterVNetInfoTool registers the get_vnet_info tool
func RegisterVNetInfoTool() mcp.Tool {
	return mcp.NewTool(
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

// RegisterNSGInfoTool registers the get_nsg_info tool
func RegisterNSGInfoTool() mcp.Tool {
	return mcp.NewTool(
		"get_nsg_info",
		mcp.WithDescription("Get information about the Network Security Group used by the AKS cluster"),
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

// RegisterRouteTableInfoTool registers the get_route_table_info tool
func RegisterRouteTableInfoTool() mcp.Tool {
	return mcp.NewTool(
		"get_route_table_info",
		mcp.WithDescription("Get information about the Route Table used by the AKS cluster"),
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

// RegisterSubnetInfoTool registers the get_subnet_info tool
func RegisterSubnetInfoTool() mcp.Tool {
	return mcp.NewTool(
		"get_subnet_info",
		mcp.WithDescription("Get information about the Subnet used by the AKS cluster"),
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

// RegisterLoadBalancersInfoTool registers the get_load_balancers_info tool
func RegisterLoadBalancersInfoTool() mcp.Tool {
	return mcp.NewTool(
		"get_load_balancers_info",
		mcp.WithDescription("Get information about all Load Balancers used by the AKS cluster (external and internal)"),
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

// TODO: Future tool categories can be added here:
