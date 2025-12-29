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
