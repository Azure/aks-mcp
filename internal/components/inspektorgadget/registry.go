package inspektorgadget

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// =============================================================================
// Inspektor Gadget related Tool Registrations
// =============================================================================

// RegisterInspektorGadgetTool registers the inspektor-gadget tool to manage gadgets
func RegisterInspektorGadgetTool() mcp.Tool {
	return mcp.NewTool(
		"inspektor_gadget_observability",
		mcp.WithDescription("Real-time observability tool for Azure Kubernetes Service (AKS) clusters, allowing users to manage gadgets for monitoring and debugging"),
		mcp.WithTitleAnnotation("Inspektor Gadget Observability"),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action to perform"),
			mcp.Description("Action to perform on the gadget: "+
				runAction+" to run a gadget for a specific duration, "+
				startAction+" to start a gadget for continuous (background) observation, "+
				stopAction+" to stop a running gadget using gadget_id, "+
				getResultsAction+" to retrieve results of a gadget run using gadget_id (only available before stopping the gadget), "+
				listGadgetsAction+" to list all running (not available) gadgets, "+
				deployAction+" to deploy Inspektor Gadget via the AKS cluster extension, "+
				undeployAction+" to undeploy the Inspektor Gadget cluster extension, "+
				isDeployedAction+" to check if Inspektor Gadget is deployed",
			),
			mcp.Enum(getActions()...),
		),
		mcp.WithString("gadget_name",
			mcp.Description("Gadget to run/start"),
			mcp.Enum(getGadgetNames()...),
		),
		mcp.WithNumber("duration",
			mcp.Description("Run duration (seconds)"),
			mcp.DefaultNumber(10),
		),
		mcp.WithString("gadget_id",
			mcp.Description("Gadget ID for stop/get_results"),
		),
		// Cluster extension lifecycle parameters
		mcp.WithString("release_train",
			mcp.Description("Release train for the Inspektor Gadget cluster extension (deploy). Defaults to 'preview'"),
		),
		mcp.WithString("version",
			mcp.Description("Specific version of the Inspektor Gadget cluster extension (deploy). Only set if the user explicitly requests a version"),
		),
		mcp.WithBoolean("auto_upgrade_minor_version",
			mcp.Description("Whether the extension auto-upgrades its minor version (deploy). Defaults to true"),
		),
		// AKS cluster identity for lifecycle actions (deploy/undeploy).
		// Falls back to the server's default AKS resource ID when not provided.
		mcp.WithString("aks_resource_id",
			mcp.Description("Full AKS cluster resource ID (/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{name}) used for extension lifecycle actions"),
		),
		mcp.WithString("subscription_id",
			mcp.Description("Azure subscription ID of the AKS cluster (used with resource_group and cluster_name)"),
		),
		mcp.WithString("resource_group",
			mcp.Description("Resource group of the AKS cluster (used with subscription_id and cluster_name)"),
		),
		mcp.WithString("cluster_name",
			mcp.Description("Name of the AKS cluster (used with subscription_id and resource_group)"),
		),
		mcp.WithBoolean("confirm",
			mcp.Description("Confirm deploy/undeploy in readonly mode"),
		),
		// Common filter parameters (flattened from filter_params)
		mcp.WithString("namespace",
			mcp.Description("Kubernetes namespace, empty to use all namespaces"),
		),
		mcp.WithString("pod",
			mcp.Description("Pod name"),
		),
		mcp.WithString("container",
			mcp.Description("Container name"),
		),
		mcp.WithString("selector",
			mcp.Description("Label selector (e.g. app=myapp,stage=prod)"),
		),
		mcp.WithString("node",
			mcp.Description("Kubernetes node name to filter on"),
		),
		withGadgetFilterParams(),
	)
}

// withGadgetFilterParams returns MCP options for all gadget-specific filter parameters
func withGadgetFilterParams() mcp.ToolOption {
	return func(t *mcp.Tool) {
		for key, value := range getGadgetParams() {
			if paramDef, ok := value.(map[string]any); ok {
				desc := ""
				if d, ok := paramDef["description"].(string); ok {
					desc = d
				}
				typ := "string"
				if tp, ok := paramDef["type"].(string); ok {
					typ = tp
				}
				t.InputSchema.Properties[key] = map[string]any{
					"type":        typ,
					"description": desc,
				}
			}
		}
	}
}
