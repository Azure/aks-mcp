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

// =============================================================================
// Monitoring-related tool registrations
// =============================================================================

// RegisterLogAnalyticsQueryTool registers the query_log_analytics tool
func RegisterLogAnalyticsQueryTool() mcp.Tool {
	return mcp.NewTool(
		"query_log_analytics",
		mcp.WithDescription("Query Azure Log Analytics workspace for AKS cluster logs (control plane, audit, node/pod logs)"),
		mcp.WithString("subscription_id",
			mcp.Description("Azure Subscription ID"),
			mcp.Required(),
		),
		mcp.WithString("workspace_id",
			mcp.Description("Log Analytics Workspace ID"),
			mcp.Required(),
		),
		mcp.WithString("kql_query",
			mcp.Description("KQL query to execute against the workspace"),
			mcp.Required(),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range for the query (e.g., '1h', '24h', '7d'). Default is '24h'"),
		),
	)
}

// RegisterPrometheusMetricsQueryTool registers the query_prometheus_metrics tool
func RegisterPrometheusMetricsQueryTool() mcp.Tool {
	return mcp.NewTool(
		"query_prometheus_metrics",
		mcp.WithDescription("Query Prometheus metrics from Azure Monitor for AKS cluster (CPU, memory, network)"),
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
		mcp.WithString("metric_names",
			mcp.Description("Comma-separated list of metric names (e.g., 'node_cpu_usage_millicores,node_memory_working_set_bytes')"),
			mcp.Required(),
		),
		mcp.WithString("region",
			mcp.Description("Azure region for the metrics endpoint (e.g., 'eastus'). Default is 'eastus'"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range for the query (e.g., '1h', '24h', '7d'). Default is '1h'"),
		),
	)
}

// RegisterApplicationInsightsQueryTool registers the query_application_insights tool
func RegisterApplicationInsightsQueryTool() mcp.Tool {
	return mcp.NewTool(
		"query_application_insights",
		mcp.WithDescription("Query Application Insights for distributed tracing data with filtering capabilities"),
		mcp.WithString("subscription_id",
			mcp.Description("Azure Subscription ID"),
			mcp.Required(),
		),
		mcp.WithString("resource_group",
			mcp.Description("Azure Resource Group containing the Application Insights component"),
			mcp.Required(),
		),
		mcp.WithString("app_insights_name",
			mcp.Description("Name of the Application Insights component"),
			mcp.Required(),
		),
		mcp.WithString("kql_query",
			mcp.Description("KQL query to execute against Application Insights (e.g., traces, requests, dependencies)"),
			mcp.Required(),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range for the query (e.g., '1h', '24h', '7d'). Default is '24h'"),
		),
	)
}

// TODO: Future tool categories can be added here:
// - Storage-related tools
// - Compute-related tools
// - etc.
