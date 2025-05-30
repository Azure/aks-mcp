// Package registry provides a tool registry for AKS MCP server.
package registry

import (
	"github.com/azure/aks-mcp/internal/handlers"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerClusterTools registers all tools related to AKS clusters.
func (r *ToolRegistry) registerClusterTools() {
	cfg := r.GetConfig()

	// Register get_cluster_info tool
	var clusterTool mcp.Tool
	if cfg.SingleClusterMode {
		clusterTool = mcp.NewTool(
			"get_cluster_info",
			mcp.WithDescription("Get information about the AKS cluster"),
		)
	} else {
		clusterTool = mcp.NewTool(
			"get_cluster_info",
			mcp.WithDescription("Get information about the AKS cluster"),
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
	// Register the tool with the unified handler
	r.RegisterTool(
		"get_cluster_info",
		clusterTool,
		handlers.GetClusterInfoHandler(r.GetClient(), r.GetCache(), cfg),
		CategoryCluster,
		AccessRead,
	)

	// Register create_or_update_cluster tool - requires write access and is only available in multi-cluster mode
	if !cfg.SingleClusterMode {
		createClusterTool := mcp.NewTool(
			"create_or_update_cluster",
			mcp.WithDescription("Create or update an AKS cluster using an ARM template"),
			mcp.WithString("subscription_id",
				mcp.Description("Azure Subscription ID"),
				mcp.Required(),
			),
			mcp.WithString("resource_group",
				mcp.Description("Azure Resource Group for the AKS cluster"),
				mcp.Required(),
			),
			mcp.WithString("cluster_name",
				mcp.Description("Name of the AKS cluster to create or update"),
				mcp.Required(),
			),
			mcp.WithString("arm_template",
				mcp.Description("ARM template JSON for the AKS cluster"),
				mcp.Required(),
			),
		)

		// Register the create_or_update_cluster tool
		r.RegisterTool(
			"create_or_update_cluster",
			createClusterTool,
			handlers.CreateOrUpdateClusterHandler(r.GetClient(), r.GetCache(), cfg),
			CategoryCluster,
			AccessReadWrite, // This tool requires write access
		)
	}

	// Only register list_aks_clusters tool when not in SingleClusterMode
	if !cfg.SingleClusterMode {
		// Register list_aks_clusters tool
		listClustersTool := mcp.NewTool(
			"list_aks_clusters",
			mcp.WithDescription("List AKS clusters in a subscription and optional resource group"),
			mcp.WithString("subscription_id",
				mcp.Description("Azure Subscription ID"),
				mcp.Required(),
			),
			mcp.WithString("resource_group",
				mcp.Description("Optional: Azure Resource Group to filter clusters by"),
			),
		)

		// Register the list clusters tool
		r.RegisterTool(
			"list_aks_clusters",
			listClustersTool,
			handlers.ListClustersHandler(r.GetClient(), r.GetCache(), cfg),
			CategoryCluster,
			AccessRead,
		)
	}
}
