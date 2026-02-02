package prompts

import (
	"context"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterHealthPrompts registers comprehensive AKS cluster health assessment prompts.
func RegisterHealthPrompts(s *server.MCPServer, cfg *config.ConfigData) {
	// Prompt: check_cluster_health
	s.AddPrompt(mcp.NewPrompt("check_cluster_health",
		mcp.WithPromptDescription("Comprehensive AKS cluster health assessment including platform health, diagnostics, cluster detectors, node health, and connectivity analysis"),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		promptContent := `# Comprehensive AKS Cluster Health Assessment

This guide performs a thorough health evaluation of your AKS cluster across multiple dimensions: cluster metadata, platform health, diagnostics configuration, detector analysis, and provides actionable recommendations.

## Steps

### 1. Retrieve Control Plane FQDN
Invoke kubectl_cluster tool:
{
  "operation": "cluster-info",
  "resource": "",
  "args": ""
}
Extract the Kubernetes control plane endpoint URL (FQDN) for cluster identification.

### 2. Identify AKS Cluster Metadata
Invoke az_aks_operations tool:
{
  "operation": "list",
  "args": "--query \"[].{id:id, fqdn:fqdn, resourceGroup:resourceGroup, name:name}\" -o json"
}
Match the control plane FQDN from Step 1 with the cluster list to determine subscription ID, resource group, and cluster name. Extract the full AKS resource ID for subsequent steps.

### 3. Check Azure Resource Health Status
Invoke aks_monitoring tool:
{
  "operation": "resource_health",
  "subscription_id": "<SUBSCRIPTION_ID>",
  "resource_group": "<RESOURCE_GROUP>",
  "cluster_name": "<CLUSTER_NAME>",
  "parameters": "{\"start_time\":\"<ISO8601_START>\",\"end_time\":\"<ISO8601_END>\"}"
}
Analyze: Identify any Azure platform incidents, service health issues, or resource degradation events that may impact cluster availability.

### 4. Run Cluster and Control Plane Availability Detectors
Invoke run_detectors_by_category tool:
{
  "cluster_resource_id": "<AKS_RESOURCE_ID>",
  "category": "Cluster and Control Plane Availability and Performance",
  "start_time": "<ISO8601_START>",
  "end_time": "<ISO8601_END>"
}
Analyze: Review API server responsiveness, control plane scaling issues, etcd health, and cluster networking performance problems.

### 5. Run Node Health Detectors
Invoke run_detectors_by_category tool:
{
  "cluster_resource_id": "<AKS_RESOURCE_ID>",
  "category": "Node Health",
  "start_time": "<ISO8601_START>",
  "end_time": "<ISO8601_END>"
}
Analyze: Examine node readiness issues, kubelet problems, container runtime health, disk pressure, memory pressure, and node pool scaling issues.

### 6. Run Connectivity Issue Detectors
Invoke run_detectors_by_category tool:
{
  "cluster_resource_id": "<AKS_RESOURCE_ID>",
  "category": "Connectivity Issues",
  "start_time": "<ISO8601_START>",
  "end_time": "<ISO8601_END>"
}
Analyze: Investigate DNS resolution problems, network policy conflicts, ingress/egress connectivity, load balancer issues, and service mesh problems.

### 7. Generate Comprehensive Health Report

Generate a comprehensive health report and recommendations based on the findings from the previous steps.

Provide specific commands, configurations, or Azure portal links where applicable for implementing recommendations.
`
		return &mcp.GetPromptResult{Description: "Comprehensive AKS cluster health assessment including platform health, diagnostics, availability detectors, node health, and connectivity analysis", Messages: []mcp.PromptMessage{{Role: mcp.RoleAssistant, Content: mcp.TextContent{Type: "text", Text: promptContent}}}}, nil
	})

}
