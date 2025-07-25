package monitor

import (
	"fmt"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
)

// MonitoringOperationType defines the type of monitoring operation
type MonitoringOperationType string

const (
	OpMetrics        MonitoringOperationType = "metrics"
	OpResourceHealth MonitoringOperationType = "resource_health"
	OpAppInsights    MonitoringOperationType = "app_insights"
	OpDiagnostics    MonitoringOperationType = "diagnostics"
	OpLogs           MonitoringOperationType = "logs"
)

// RegisterAzMonitoring registers the monitoring tool
func RegisterAzMonitoring() mcp.Tool {
	description := `Unified tool for Azure monitoring and diagnostics operations for AKS clusters.

Supported operations:
- metrics: Query metrics for Azure resources (list, list-definitions, list-namespaces)
- resource_health: Get resource health events for AKS clusters
- app_insights: Execute KQL queries against Application Insights data
- diagnostics: Check AKS cluster diagnostic settings configuration
- logs: Query AKS control plane logs with safety constraints

Examples:
- Get metrics: operation="metrics", query_type="list", parameters="{\"resource\":\"...\", \"metric\":\"...\"}"
- Resource health: operation="resource_health", parameters="{\"start_time\":\"2025-01-01T00:00:00Z\"}"
- App Insights query: operation="app_insights", parameters="{\"app_insights_name\":\"...\", \"query\":\"...\"}"
- Check diagnostics: operation="diagnostics"
- Query logs: operation="logs", parameters="{\"log_category\":\"kube-apiserver\", \"start_time\":\"...\"}"
`

	return mcp.NewTool("az_monitoring",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The monitoring operation to perform"),
		),
		mcp.WithString("query_type",
			mcp.Description("Specific type of query for metrics operations (list, list-definitions, list-namespaces)"),
		),
		mcp.WithString("parameters",
			mcp.Required(),
			mcp.Description("JSON string containing operation-specific parameters"),
		),
		mcp.WithString("subscription_id",
			mcp.Description("Azure subscription ID (can be included in parameters)"),
		),
		mcp.WithString("resource_group",
			mcp.Description("Resource group name (can be included in parameters)"),
		),
		mcp.WithString("cluster_name",
			mcp.Description("AKS cluster name (can be included in parameters)"),
		),
	)
}

// ValidateMonitoringOperation checks if the monitoring operation is supported
func ValidateMonitoringOperation(operation string) bool {
	supportedOps := []string{
		string(OpMetrics), string(OpResourceHealth), string(OpAppInsights),
		string(OpDiagnostics), string(OpLogs),
	}
	return slices.Contains(supportedOps, operation)
}

// GetSupportedMonitoringOperations returns all supported monitoring operations
func GetSupportedMonitoringOperations() []string {
	return []string{
		string(OpMetrics), string(OpResourceHealth), string(OpAppInsights),
		string(OpDiagnostics), string(OpLogs),
	}
}

// ValidateMetricsQueryType checks if the metrics query type is supported
func ValidateMetricsQueryType(queryType string) bool {
	supportedTypes := []string{"list", "list-definitions", "list-namespaces"}
	return slices.Contains(supportedTypes, queryType)
}

// MapMetricsQueryTypeToCommand maps a metrics query type to its corresponding az command
func MapMetricsQueryTypeToCommand(queryType string) (string, error) {
	commandMap := map[string]string{
		"list":             "az monitor metrics list",
		"list-definitions": "az monitor metrics list-definitions",
		"list-namespaces":  "az monitor metrics list-namespaces",
	}

	cmd, exists := commandMap[queryType]
	if !exists {
		return "", fmt.Errorf("no command mapping for metrics query type: %s", queryType)
	}

	return cmd, nil
}
