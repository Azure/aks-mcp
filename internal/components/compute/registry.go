package compute

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Compute-related tool registrations

// RegisterAKSVMSSInfoTool registers the get_aks_vmss_info tool
func RegisterAKSVMSSInfoTool() mcp.Tool {
	return mcp.NewTool(
		"get_aks_vmss_info",
		mcp.WithDescription("Get detailed VMSS configuration for a specific node pool or all node pools in the AKS cluster (provides low-level VMSS settings not available in az aks nodepool show). Leave node_pool_name empty to get info for all node pools."),
		mcp.WithTitleAnnotation("Get AKS VMSS Info"),
		mcp.WithReadOnlyHintAnnotation(true),
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
		mcp.WithString("node_pool_name",
			mcp.Description("Name of the node pool to get VMSS information for. Leave empty to get info for all node pools."),
		),
	)
}

// RegisterCollectAKSNodeLogsTool registers the collect_aks_node_logs tool
func RegisterCollectAKSNodeLogsTool() mcp.Tool {
	return mcp.NewTool(
		"collect_aks_node_logs",
		mcp.WithDescription(
			"Collect kubelet, containerd, kernel (dmesg), or syslog logs from AKS node using VMSS run command. "+
				"Useful for debugging node-level issues. Supports filtering by time range and log level. "+
				"IMPORTANT: Only ONE run command can execute at a time per VMSS instance - wait for completion before running another command on the same instance. "+
				"To get vmss_name and instance_id for a specific node A, use 'kubectl get node A -o json' and parse .spec.providerID "+
				"(format: azure:///.../virtualMachineScaleSets/{vmss_name}/virtualMachines/{instance_id}).",
		),
		mcp.WithTitleAnnotation("Collect AKS Node Logs"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithString("aks_resource_id",
			mcp.Description("AKS cluster resource ID (/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{name})"),
			mcp.Required(),
		),
		mcp.WithString("vmss_name",
			mcp.Description("VMSS name (can be obtained from get_aks_vmss_info or kubectl get nodes)"),
			mcp.Required(),
		),
		mcp.WithString("instance_id",
			mcp.Description("VMSS instance ID (can be obtained from get_aks_vmss_info or kubectl get nodes)"),
			mcp.Required(),
		),
		mcp.WithString("log_type",
			mcp.Description("Type of logs to collect: kubelet, containerd, kernel, syslog"),
			mcp.Required(),
		),
		mcp.WithNumber("lines",
			mcp.Description("Number of most recent log lines to return (default: 500, max: 2000)"),
		),
		mcp.WithString("since",
			mcp.Description("Time range for logs, e.g., '1h', '30m', '2d' (takes precedence over lines)"),
		),
		mcp.WithString("level",
			mcp.Description("Log level filter: ERROR, WARN, INFO (default: INFO shows all logs)"),
		),
		mcp.WithString("filter",
			mcp.Description("Filter logs by keyword (case-insensitive text match, not regex)"),
		),
	)
}
