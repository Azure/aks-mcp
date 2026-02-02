package monitor

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// MonitoringOperationType defines the type of monitoring operation
type MonitoringOperationType string

const (
	OpMetrics          MonitoringOperationType = "metrics"
	OpResourceHealth   MonitoringOperationType = "resource_health"
	OpAppInsights      MonitoringOperationType = "app_insights"
	OpDiagnostics      MonitoringOperationType = "diagnostics"
	OpControlPlaneLogs MonitoringOperationType = "control_plane_logs"
)

// RegisterAksMonitoring registers the monitoring tool
func RegisterAksMonitoring() mcp.Tool {
	description := `Unified tool for Azure monitoring and diagnostics operations for AKS clusters.

Supported Operations:

1. Metrics - Query Azure Monitor metrics for AKS clusters and nodes
   - list: Get metric values for specific metrics
   - list-definitions: Get available metrics for a resource
   - list-namespaces: Get metric namespaces for a resource
   
   Use for: CPU usage, memory consumption, network traffic, pod counts, node health
   Required parameters: resource (Azure resource ID)
   Additional for 'list': metrics (metric names)
   Optional: aggregation, start-time, end-time, interval, filter

2. Resource Health - Get Azure Resource Health events for AKS clusters
   Use for: Cluster availability issues, platform problems, service health events
   Required parameters: subscription_id, resource_group, cluster_name, start_time
   Optional: end_time, status (Available, Unavailable, Degraded, Unknown)

3. Application Insights - Execute KQL queries against Application Insights telemetry
   Use for: Application performance monitoring, custom telemetry analysis, trace correlation
   Required parameters: subscription_id, resource_group, app_insights_name, query
   Optional: start_time + end_time OR timespan (not both)

4. Diagnostics - Check AKS cluster diagnostic settings configuration
   Use for: Verify logging is enabled, check log retention, validate diagnostic configuration
   Required parameters: subscription_id, resource_group, cluster_name

5. Control Plane Logs - Query AKS control plane logs
   Supported log categories:
   - kube-apiserver
   - kube-audit
   - kube-audit-admin
   - kube-controller-manager
   - kube-scheduler
   - cluster-autoscaler
   - cloud-controller-manager
   - guard (for authentication/authorization issues)
   - csi-azuredisk-controller
   - csi-azurefile-controller
   - csi-snapshot-controller
   - fleet-member-agent
   - fleet-member-net-controller-manager
   - fleet-mcs-controller-manager
   PLEASE NOTE: you need to check if the category is enabled in your cluster's diagnostic settings by using the diagnostics tool.

Use This Tool When You Need To:
- Monitor cluster or other azure resource performance and usage (use metrics)
- Check cluster availability and platform health (use resource_health)
- Analyze application telemetry and performance (use app_insights)
- Verify diagnostic logging configuration (use diagnostics)
- Debug Kubernetes API server issues (use control_plane_logs with kube-apiserver)
- Investigate authentication/authorization problems (use control_plane_logs with kube-audit, guard)
- Troubleshoot pod scheduling issues (use control_plane_logs with kube-scheduler)
- Check storage-related problems (use control_plane_logs with csi-azuredisk-controller, csi-azurefile-controller)
- Analyze cluster scaling behavior (use control_plane_logs with cluster-autoscaler)
- Review security audit events (use control_plane_logs with kube-audit, kube-audit-admin)

Examples:

metrics:
- Get CPU usage: operation="metrics", query_type="list", parameters="{\"resource\":\"/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster\", \"metrics\":\"node_cpu_usage_percentage\", \"aggregation\":\"Average\", \"start-time\":\"<start-time>\", \"end-time\":\"<end-time>\"}"
- List available metrics: operation="metrics", query_type="list-definitions", parameters="{\"resource\":\"/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster\"}"

resource_health:
- Check recent cluster health: operation="resource_health", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"start_time\":\"<start-time>\"}"

app_insights:
- Query request telemetry: operation="app_insights", subscription_id="<subscription-id>", resource_group="<resource-group>", parameters="{\"app_insights_name\":\"myapp-insights\", \"query\":\"requests | where timestamp > ago(1h) | summarize count() by bin(timestamp, 5m)\"}"
- Analyze exceptions: operation="app_insights", subscription_id="<subscription-id>", resource_group="<resource-group>", parameters="{\"app_insights_name\":\"myapp-insights\", \"query\":\"exceptions | where timestamp > ago(24h) | summarize count() by type, bin(timestamp, 1h)\"}"
- Performance with timespan: operation="app_insights", subscription_id="<subscription-id>", resource_group="<resource-group>", parameters="{\"app_insights_name\":\"myapp-insights\", \"query\":\"performanceCounters | where category == 'Processor' | summarize avg(value) by bin(timestamp, 5m)\", \"timespan\":\"PT1H\"}"

diagnostics:
- Verify diagnostic settings: operation="diagnostics", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{}"

control_plane_logs:
- Query API server logs: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"kube-apiserver\", \"start_time\":\"<start-time>\", \"end_time\":\"<end-time>\", \"max_records\":\"50\"}"
- Debug authentication issues: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"guard\", \"start_time\":\"<start-time>\", \"end_time\":\"<end-time>\", \"max_records\":\"100\"}"
- Analyze audit events: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"kube-audit\", \"log_level\":\"error\", \"start_time\":\"<start-time>\", \"end_time\":\"<end-time>\", \"max_records\":\"50\"}"
`

	return mcp.NewTool("aks_monitoring",
		mcp.WithDescription(description),
		mcp.WithTitleAnnotation("Azure Monitoring"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The monitoring operation to perform: 'metrics' (CPU/memory/network), 'resource_health' (cluster availability), 'app_insights' (telemetry analysis), 'diagnostics' (logging config), 'control_plane_logs' (Kubernetes logs like kube-apiserver, kube-audit, guard, etc.)"),
		),
		mcp.WithString("query_type",
			mcp.Description("For metrics operations only: 'list' (get metric values), 'list-definitions' (available metrics), 'list-namespaces' (metric categories)"),
		),
		mcp.WithString("parameters",
			mcp.Required(),
			mcp.Description("JSON string with operation parameters. metrics: resource (required), metrics (required for 'list' query_type), aggregation/start-time/end-time/interval/filter (optional). resource_health: start_time, end_time, status. app_insights: app_insights_name, query, start_time/end_time OR timespan (optional). diagnostics: none required. control_plane_logs: log_category (kube-apiserver/kube-audit/guard/etc), start_time, end_time, max_records, log_level"),
		),
		mcp.WithString("subscription_id",
			mcp.Description("Azure subscription ID (required for resource_health, app_insights, diagnostics, control_plane_logs)"),
		),
		mcp.WithString("resource_group",
			mcp.Description("Resource group name (required for resource_health, app_insights, diagnostics, control_plane_logs)"),
		),
		mcp.WithString("cluster_name",
			mcp.Description("AKS cluster name (required for resource_health, diagnostics, control_plane_logs)"),
		),
	)
}
