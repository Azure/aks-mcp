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

// =============================================================================
// AppLens Diagnostic Tool Registrations
// =============================================================================

// RegisterListAppLensDetectorsTool registers the list_applens_detectors tool
func RegisterListAppLensDetectorsTool() mcp.Tool {
	return mcp.NewTool(
		"list_applens_detectors",
		mcp.WithDescription("List all available AppLens detectors for an AKS cluster"),
		mcp.WithString("cluster_resource_id",
			mcp.Description("Full Azure resource ID of the AKS cluster"),
			mcp.Required(),
		),
		mcp.WithString("category",
			mcp.Description("Filter by detector category (performance, security, reliability)"),
		),
	)
}

// RegisterInvokeAppLensDetectorTool registers the invoke_applens_detector tool
func RegisterInvokeAppLensDetectorTool() mcp.Tool {
	return mcp.NewTool(
		"invoke_applens_detector",
		mcp.WithDescription("Call and invoke AppLens detectors for AKS clusters"),
		mcp.WithString("cluster_resource_id",
			mcp.Description("Full Azure resource ID of the AKS cluster"),
			mcp.Required(),
		),
		mcp.WithString("detector_name",
			mcp.Description("Specific detector to run, if not provided, list available detectors"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range for analysis (e.g., '24h', '7d', '30d')"),
		),
	)
}
