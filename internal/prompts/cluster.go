package prompts

import (
	"context"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterQueryAKSMetadataFromKubeconfigPrompt registers the prompts to query aks cluster from active kubeconfig.
func RegisterQueryAKSMetadataFromKubeconfigPrompt(s *server.MCPServer, cfg *config.ConfigData) {
	s.AddPrompt(mcp.NewPrompt("query_aks_cluster_metadata_from_kubeconfig",
		mcp.WithPromptDescription("Query AKS cluster (subscriptionID, resourceGroup and name) from current kubeconfig"),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {

		promptContent := `# Query AKS cluster metadata from current kubeconfig

This guide will help you query AKS cluster metadata (subscriptionID, resourceGroup and name) from current kubeconfig.

## Steps

### 1. Retrieve the control plane FQDN:

Invoke the call_kubectl MCP tool with inputs:
	{
		"command": "kubectl cluster-info"
	}


This will show the Kubernetes control plane endpoint URL (FQDN).

### 2. List available AKS clusters in your subscription

Invoke the call_az MCP tool with inputs:
	{
		"cli_command": "az aks list --query \"[].{id:id, fqdn:fqdn}\" -o json"
	}

This will show all the AKS clusters (in JSON format) in user pre-configured subscription.

### 3. Find the AKS cluster matching FQDN

Compare the control plane FQDN from Step 1 with the FQDNs of the AKS clusters from Step 2,
figure out the matched AKS cluster, and then respond the AKS cluster's subscriptionID, resourceGroup and name.
`

		return &mcp.GetPromptResult{
			Description: "Query AKS cluster (subscriptionID, resourceGroup and name) from current kubeconfig",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: promptContent,
					},
				},
			},
		}, nil
	})
}
