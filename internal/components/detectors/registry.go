package detectors

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Detector-related tool registrations

// RegisterAksDetectorTool registers the unified aks_detector MCP tool
func RegisterAksDetectorTool() mcp.Tool {
	return mcp.NewTool(
		"aks_detector",
		mcp.WithDescription("Execute AKS detector operations (list detectors, run detector, or run detectors by category)"),
		mcp.WithTitleAnnotation("Call Detector"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("operation",
			mcp.Description("Operation to perform: 'list' (list all detectors), 'run' (run single detector), 'run_by_category' (run all detectors in category)"),
			mcp.Required(),
		),
		mcp.WithString("aks_resource_id",
			mcp.Description("AKS cluster resource ID"),
			mcp.Required(),
		),
		mcp.WithString("detector_name",
			mcp.Description("Name of the detector to run (required for 'run' operation)"),
		),
		mcp.WithString("category",
			mcp.Description("Detector category (required for 'run_by_category' operation). Valid values: Best Practices, Cluster and Control Plane Availability and Performance, Connectivity Issues, Create/Upgrade/Delete and Scale, Deprecations, Identity and Security, Node Health, Storage"),
		),
		mcp.WithString("start_time",
			mcp.Description("Start time in UTC ISO format (required for 'run' and 'run_by_category' operations, within last 30 days). Example: 2025-07-11T10:55:13Z"),
		),
		mcp.WithString("end_time",
			mcp.Description("End time in UTC ISO format (required for 'run' and 'run_by_category' operations, within last 30 days, max 24h from start). Example: 2025-07-11T14:55:13Z"),
		),
	)
}
