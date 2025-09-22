package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/components/common"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/aks-mcp/internal/tools"
)

// buildClusterResourceID constructs the Azure resource ID for an AKS cluster
func buildClusterResourceID(subscriptionID, resourceGroup, clusterName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
		subscriptionID, resourceGroup, clusterName)
}

// HandleControlPlaneDiagnosticSettings checks diagnostic settings for AKS cluster
func HandleControlPlaneDiagnosticSettings(params map[string]interface{}, azClient *azureclient.AzureClient, cfg *config.ConfigData) (string, error) {
	// Extract and validate parameters using common helper
	subscriptionID, resourceGroup, clusterName, err := common.ExtractAKSParameters(params)
	if err != nil {
		return "", err
	}

	// Build cluster resource ID using utility function
	clusterResourceID := buildClusterResourceID(subscriptionID, resourceGroup, clusterName)

	// Azure client is required
	if azClient == nil {
		return "", fmt.Errorf("azure client is required but not provided")
	}

	// Get diagnostic settings using Azure SDK
	ctx := context.Background()
	diagnosticSettings, err := azClient.GetDiagnosticSettings(ctx, subscriptionID, clusterResourceID)
	if err != nil {
		return "", fmt.Errorf("failed to get diagnostic settings for cluster %s in resource group %s: %w", clusterName, resourceGroup, err)
	}

	// Convert to JSON for backward compatibility
	result, err := json.Marshal(diagnosticSettings)
	if err != nil {
		return "", fmt.Errorf("failed to marshal diagnostic settings to JSON: %w", err)
	}

	return string(result), nil
}

// HandleControlPlaneLogs queries specific control plane logs
func HandleControlPlaneLogs(params map[string]interface{}, azClient *azureclient.AzureClient, cfg *config.ConfigData) (string, error) {
	// Extract and validate AKS parameters using common helper
	subscriptionID, resourceGroup, clusterName, err := common.ExtractAKSParameters(params)
	if err != nil {
		return "", err
	}

	// Extract remaining parameters
	logCategory, _ := params["log_category"].(string)
	startTime, _ := params["start_time"].(string)
	endTime, _ := params["end_time"].(string)
	maxRecords := GetMaxRecords(params)
	logLevel, _ := params["log_level"].(string)

	// Validate parameters
	if err := ValidateControlPlaneLogsParams(params); err != nil {
		return "", err
	}

	// Find the diagnostic setting that has the requested log category enabled
	// This handles cases where multiple diagnostic settings exist for the same cluster
	workspaceResourceID, isResourceSpecific, err := FindDiagnosticSettingForCategory(subscriptionID, resourceGroup, clusterName, logCategory, azClient, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to find diagnostic setting for log category %s in cluster %s: %w", logCategory, clusterName, err)
	}

	// Get workspace GUID from the workspace resource ID
	workspaceGUID, err := getWorkspaceGUID(workspaceResourceID, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace GUID for cluster %s: %w", clusterName, err)
	}

	// Build cluster resource ID for scoping using utility function
	clusterResourceID := buildClusterResourceID(subscriptionID, resourceGroup, clusterName)

	// Build safe KQL query scoped to this specific AKS cluster with appropriate table mode
	kqlQuery, err := BuildSafeKQLQuery(logCategory, logLevel, maxRecords, clusterResourceID, isResourceSpecific)
	if err != nil {
		return "", fmt.Errorf("failed to build KQL query for cluster %s: %w", clusterName, err)
	}

	// Calculate timespan for the query
	timespan, err := CalculateTimespan(startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("failed to calculate timespan: %w", err)
	}

	// Execute log query with properly quoted KQL
	executor := azcli.NewExecutor()

	// Build command string with proper quoting for the KQL query
	cmd := fmt.Sprintf("az monitor log-analytics query --workspace %s --analytics-query \"%s\" --timespan %s --output json",
		workspaceGUID, kqlQuery, timespan)

	// Log the query command for debugging
	logger.Debugf("Executing KQL query command: %s", cmd)

	cmdParams := map[string]interface{}{
		"command": cmd,
	}

	result, err := executor.Execute(cmdParams, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to query control plane logs for category %s in cluster %s: %w", logCategory, clusterName, err)
	}

	// Return raw JSON result from Azure CLI
	return result, nil
}

// Resource handler functions for control plane diagnostics tools

// GetControlPlaneDiagnosticSettingsHandler returns handler for diagnostic settings tool
func GetControlPlaneDiagnosticSettingsHandler(azClient *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return HandleControlPlaneDiagnosticSettings(params, azClient, cfg)
	})
}

// GetControlPlaneLogsHandler returns handler for logs querying tool
func GetControlPlaneLogsHandler(azClient *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return HandleControlPlaneLogs(params, azClient, cfg)
	})
}
