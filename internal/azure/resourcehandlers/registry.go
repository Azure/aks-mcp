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

// Advisory-related tool registrations

// RegisterAdvisorRecommendationTool registers the az_advisor_recommendation tool
func RegisterAdvisorRecommendationTool() mcp.Tool {
	return mcp.NewTool(
		"az_advisor_recommendation",
		mcp.WithDescription("Retrieve and manage Azure Advisor recommendations for AKS clusters"),
		mcp.WithString("operation",
			mcp.Description("Operation to perform: list, details, or report"),
			mcp.Required(),
		),
		mcp.WithString("subscription_id",
			mcp.Description("Azure subscription ID to query recommendations"),
			mcp.Required(),
		),
		mcp.WithString("resource_group",
			mcp.Description("Filter by specific resource group containing AKS clusters"),
		),
		mcp.WithString("cluster_names",
			mcp.Description("Comma-separated list of specific AKS cluster names to filter recommendations"),
		),
		mcp.WithString("category",
			mcp.Description("Filter by recommendation category: Cost, HighAvailability, Performance, Security"),
		),
		mcp.WithString("severity",
			mcp.Description("Filter by severity level: High, Medium, Low"),
		),
		mcp.WithString("recommendation_id",
			mcp.Description("Unique identifier for specific recommendation (required for details operation)"),
		),
		mcp.WithString("format",
			mcp.Description("Output format for reports: summary, detailed, actionable"),
		),
	)
}
