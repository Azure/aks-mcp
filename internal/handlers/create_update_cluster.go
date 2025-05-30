// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CreateOrUpdateClusterHandler returns a handler for the create_or_update_cluster tool.
// It allows creating or updating an AKS cluster using an ARM template.
func CreateOrUpdateClusterHandler(client *azure.AzureClient, cache *azure.AzureCache, cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters from the request
		subscriptionID, _ := request.GetArguments()["subscription_id"].(string)
		resourceGroup, _ := request.GetArguments()["resource_group"].(string)
		clusterName, _ := request.GetArguments()["cluster_name"].(string)
		armTemplate, _ := request.GetArguments()["arm_template"].(string)

		// Validate required parameters
		if subscriptionID == "" || resourceGroup == "" || clusterName == "" || armTemplate == "" {
			return nil, fmt.Errorf("missing required parameters: subscription_id, resource_group, cluster_name, and arm_template")
		}

		// Parse the ARM template
		var clusterProperties map[string]interface{}
		if err := json.Unmarshal([]byte(armTemplate), &clusterProperties); err != nil {
			return nil, fmt.Errorf("failed to parse ARM template: %v", err)
		}

		// Create the cluster model from ARM template
		cluster, err := convertToClusterModel(clusterProperties, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ARM template to cluster model: %v", err)
		}

		// Create or update the cluster
		operation, err := client.CreateOrUpdateAKSCluster(ctx, subscriptionID, resourceGroup, clusterName, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to create or update AKS cluster: %v", err)
		}

		// Check if this is a create or update operation
		operationType := "update"
		if operation.Properties != nil && operation.Properties.ProvisioningState != nil {
			if strings.EqualFold(*operation.Properties.ProvisioningState, "Creating") {
				operationType = "create"
			}
		}

		// Clear the cache entry for this cluster
		cacheKey := fmt.Sprintf("akscluster:/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
			subscriptionID, resourceGroup, clusterName)
		cache.Delete(cacheKey)

		// Return the result
		result := map[string]interface{}{
			"operation": operationType,
			"status":    "initiated",
			"message":   fmt.Sprintf("AKS cluster %s operation initiated", operationType),
			"cluster": map[string]string{
				"name":           clusterName,
				"resourceGroup":  resourceGroup,
				"subscriptionId": subscriptionID,
			},
		}

		jsonResult, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %v", err)
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	}
}

// convertToClusterModel converts an ARM template to a cluster model
func convertToClusterModel(clusterProperties map[string]interface{}, clusterName string) (*armcontainerservice.ManagedCluster, error) {
	// Convert the ARM template to JSON
	clusterJSON, err := json.Marshal(clusterProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster properties: %v", err)
	}

	// Create a new ManagedCluster model
	cluster := &armcontainerservice.ManagedCluster{}

	// Unmarshal the JSON into the cluster model
	if err := json.Unmarshal(clusterJSON, cluster); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster properties: %v", err)
	}

	return cluster, nil
}
