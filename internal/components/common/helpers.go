// Package common provides shared utility functions for AKS MCP components.
package common

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/config"
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

// GetDefaultSubscriptionID attempts to get the subscription ID from environment variable or Azure CLI
func GetDefaultSubscriptionID(cfg *config.ConfigData) (string, error) {
	if subID := os.Getenv("AZURE_SUBSCRIPTION_ID"); subID != "" {
		return subID, nil
	}

	executor := azcli.NewExecutor()
	cmdParams := map[string]interface{}{
		"command": "az account show --query id -o tsv",
	}

	out, err := executor.Execute(context.Background(), cmdParams, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to get default subscription ID from Azure CLI: %w. Please set AZURE_SUBSCRIPTION_ID environment variable or provide subscription_id parameter", err)
	}

	subID := strings.TrimSpace(out)
	if subID == "" {
		return "", fmt.Errorf("no default subscription found. Please set AZURE_SUBSCRIPTION_ID environment variable or provide subscription_id parameter")
	}

	return subID, nil
}
