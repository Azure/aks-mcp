package detectors

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Detector-related tool registrations

// RegisterListDetectorsTool registers the list_detectors MCP tool
func RegisterListDetectorsTool() mcp.Tool {
	return mcp.NewTool(
		"list_detectors",
		mcp.WithDescription("List all available AKS cluster detectors"),
		mcp.WithTitleAnnotation("List Detectors"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("cluster_resource_id",
			mcp.Description("AKS cluster resource ID"),
			mcp.Required(),
		),
	)
}

// RegisterRunDetectorTool registers the run_detector MCP tool
func RegisterRunDetectorTool() mcp.Tool {
	return mcp.NewTool(
		"run_detector",
		mcp.WithDescription("Run a specific AKS detector"),
		mcp.WithTitleAnnotation("Run Detector"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("cluster_resource_id",
			mcp.Description("AKS cluster resource ID"),
			mcp.Required(),
		),
		mcp.WithString("detector_name",
			mcp.Description("Name of the detector to run"),
			mcp.Required(),
		),
		mcp.WithString("start_time",
			mcp.Description("Start time in UTC ISO format (within last 30 days). Example: 2025-07-11T10:55:13Z"),
			mcp.Required(),
		),
		mcp.WithString("end_time",
			mcp.Description("End time in UTC ISO format (within last 30 days, max 24h from start). Example: 2025-07-11T14:55:13Z"),
			mcp.Required(),
		),
	)
}

// RegisterRunDetectorsByCategoryTool registers the run_detectors_by_category MCP tool
func RegisterRunDetectorsByCategoryTool() mcp.Tool {
	return mcp.NewTool(
		"run_detectors_by_category",
		mcp.WithDescription("Run all detectors in a specific category"),
		mcp.WithTitleAnnotation("Run Detectors By Category"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("cluster_resource_id",
			mcp.Description("AKS cluster resource ID"),
			mcp.Required(),
		),
		mcp.WithString("category",
			mcp.Description("Detector category to run (Best Practices, Cluster and Control Plane Availability and Performance, Connectivity Issues, Create/Upgrade/Delete and Scale, Deprecations, Identity and Security, Node Health, Storage)"),
			mcp.Required(),
		),
		mcp.WithString("start_time",
			mcp.Description("Start time in UTC ISO format (within last 30 days). Example: 2025-07-11T10:55:13Z"),
			mcp.Required(),
		),
		mcp.WithString("end_time",
			mcp.Description("End time in UTC ISO format (within last 30 days, max 24h from start). Example: 2025-07-11T14:55:13Z"),
			mcp.Required(),
		),
	)
}
