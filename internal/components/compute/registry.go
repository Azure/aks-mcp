package compute

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Compute-related tool registrations

// RegisterVMSSInfoByNodePoolTool registers the get_vmss_info_by_node_pool tool
func RegisterVMSSInfoByNodePoolTool() mcp.Tool {
	return mcp.NewTool(
		"get_vmss_info_by_node_pool",
		mcp.WithDescription("Get detailed VMSS configuration for a specific node pool (provides low-level VMSS settings not available in az aks nodepool show)"),
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
			mcp.Description("Name of the node pool to get VMSS information for"),
			mcp.Required(),
		),
	)
}

// RegisterAllVMSSByClusterTool registers the get_all_vmss_by_cluster tool
func RegisterAllVMSSByClusterTool() mcp.Tool {
	return mcp.NewTool(
		"get_all_vmss_by_cluster",
		mcp.WithDescription("Get detailed VMSS configuration and properties for all node pools in the AKS cluster (complements az aks nodepool commands with low-level VMSS details)"),
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

// VMSS Command Registration Functions

// RegisterAzVmssCommand registers a specific az vmss command as an MCP tool
func RegisterAzVmssCommand(cmd ComputeCommand) mcp.Tool {
	return RegisterAzComputeCommand(cmd)
}

// GetReadOnlyVmssAzCommands returns all read-only az vmss commands
func GetReadOnlyVmssAzCommands() []ComputeCommand {
	return GetReadOnlyVmssCommands()
}

// GetReadWriteVmssAzCommands returns all read-write az vmss commands
func GetReadWriteVmssAzCommands() []ComputeCommand {
	return GetReadWriteVmssCommands()
}

// GetAdminVmssAzCommands returns all admin az vmss commands
func GetAdminVmssAzCommands() []ComputeCommand {
	return GetAdminVmssCommands()
}
