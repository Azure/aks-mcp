// Package common provides shared utility functions for AKS MCP components.
package common

import (
	"context"
	"fmt"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
)

// ExtractAKSParameters extracts and validates the common AKS parameters from the params map
func ExtractAKSParameters(params map[string]interface{}) (subscriptionID, resourceGroup, clusterName string, err error) {
	subID, ok := params["subscription_id"].(string)
	if !ok || subID == "" {
		return "", "", "", fmt.Errorf("missing or invalid subscription_id parameter")
	}

	rg, ok := params["resource_group"].(string)
	if !ok || rg == "" {
		return "", "", "", fmt.Errorf("missing or invalid resource_group parameter")
	}

	clusterNameParam, ok := params["cluster_name"].(string)
	if !ok || clusterNameParam == "" {
		return "", "", "", fmt.Errorf("missing or invalid cluster_name parameter")
	}

	return subID, rg, clusterNameParam, nil
}

// GetClusterDetails gets the details of an AKS cluster
func GetClusterDetails(ctx context.Context, client *azureclient.AzureClient, subscriptionID, resourceGroup, clusterName string) (*armcontainerservice.ManagedCluster, error) {
	// Get the cluster from Azure client (which now handles caching internally)
	return client.GetAKSCluster(ctx, subscriptionID, resourceGroup, clusterName)
}
