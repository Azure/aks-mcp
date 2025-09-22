package diagnostics

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/logger"
)

// ExtractWorkspaceGUIDFromDiagnosticSettings extracts workspace GUID from diagnostic settings
func ExtractWorkspaceGUIDFromDiagnosticSettings(subscriptionID, resourceGroup, clusterName string, azClient *azureclient.AzureClient, cfg *config.ConfigData) (string, error) {
	// Build cluster resource ID
	clusterResourceID := buildClusterResourceID(subscriptionID, resourceGroup, clusterName)

	// Azure client is required
	if azClient == nil {
		return "", fmt.Errorf("azure client is required but not provided")
	}

	// Get diagnostic settings using Azure SDK
	ctx := context.Background()
	diagnosticSettings, err := azClient.GetDiagnosticSettings(ctx, subscriptionID, clusterResourceID)
	if err != nil {
		return "", fmt.Errorf("failed to get diagnostic settings: %w", err)
	}

	// Extract workspace resource ID from the first diagnostic setting
	if len(diagnosticSettings) > 0 {
		setting := diagnosticSettings[0]
		if setting.Properties != nil && setting.Properties.WorkspaceID != nil && *setting.Properties.WorkspaceID != "" {
			// Extract workspace GUID from the workspace resource ID
			return getWorkspaceGUID(*setting.Properties.WorkspaceID, cfg)
		}
	}

	return "", fmt.Errorf("no Log Analytics workspace found in diagnostic settings")
}

// getWorkspaceGUID extracts the workspace GUID from a workspace resource ID
func getWorkspaceGUID(workspaceResourceID string, cfg *config.ConfigData) (string, error) {
	// Parse the workspace resource ID to extract resource group and workspace name
	// Format: /subscriptions/{sub}/resourcegroups/{rg}/providers/microsoft.operationalinsights/workspaces/{workspace-name}
	parts := strings.Split(workspaceResourceID, "/")
	if len(parts) < 8 {
		return "", fmt.Errorf("invalid workspace resource ID format: %s", workspaceResourceID)
	}

	var resourceGroup, workspaceName string
	for i, part := range parts {
		if strings.ToLower(part) == "resourcegroups" && i+1 < len(parts) {
			resourceGroup = parts[i+1]
		}
		if strings.ToLower(part) == "workspaces" && i+1 < len(parts) {
			workspaceName = parts[i+1]
		}
	}

	if resourceGroup == "" || workspaceName == "" {
		return "", fmt.Errorf("could not extract resource group and workspace name from: %s", workspaceResourceID)
	}

	// Query the workspace to get its GUID (customerId)
	executor := azcli.NewExecutor()
	cmd := fmt.Sprintf("az monitor log-analytics workspace show --resource-group %s --workspace-name %s --query customerId --output tsv", resourceGroup, workspaceName)

	cmdParams := map[string]interface{}{
		"command": cmd,
	}

	result, err := executor.Execute(cmdParams, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace GUID: %w", err)
	}

	// The result should be the workspace GUID, trim any whitespace
	workspaceGUID := strings.TrimSpace(result)
	if workspaceGUID == "" {
		return "", fmt.Errorf("empty workspace GUID returned for workspace: %s", workspaceName)
	}

	return workspaceGUID, nil
}

// FindDiagnosticSettingForCategory finds the first diagnostic setting that has the specified log category enabled
// Returns the workspace ID and whether it uses resource-specific tables
func FindDiagnosticSettingForCategory(subscriptionID, resourceGroup, clusterName, logCategory string, azClient *azureclient.AzureClient, cfg *config.ConfigData) (string, bool, error) {
	// Build cluster resource ID
	clusterResourceID := buildClusterResourceID(subscriptionID, resourceGroup, clusterName)

	// Azure client is required
	if azClient == nil {
		return "", false, fmt.Errorf("azure client is required but not provided")
	}

	// Get diagnostic settings using Azure SDK
	ctx := context.Background()
	diagnosticSettings, err := azClient.GetDiagnosticSettings(ctx, subscriptionID, clusterResourceID)
	if err != nil {
		return "", false, fmt.Errorf("failed to get diagnostic settings: %w", err)
	}

	// Find the first diagnostic setting that has the requested log category enabled
	for _, setting := range diagnosticSettings {
		if setting.Properties == nil || setting.Properties.Logs == nil {
			continue
		}

		// Check each log category in this setting
		for _, logConfig := range setting.Properties.Logs {
			if logConfig.Category != nil && *logConfig.Category == logCategory {
				if logConfig.Enabled != nil && *logConfig.Enabled {
					// Found the category and it's enabled, now get workspace and table mode
					if setting.Properties.WorkspaceID == nil || *setting.Properties.WorkspaceID == "" {
						continue // Skip if no workspace configured
					}

					workspaceResourceID := *setting.Properties.WorkspaceID

					// Determine table mode from logAnalyticsDestinationType
					isResourceSpecific := false
					if setting.Properties.LogAnalyticsDestinationType != nil {
						isResourceSpecific = strings.ToLower(string(*setting.Properties.LogAnalyticsDestinationType)) == "dedicated"
					}

					// Get diagnostic setting name for debugging
					settingName := "unknown"
					if setting.Name != nil {
						settingName = *setting.Name
					}

					// Debug log which setting and workspace is being used
					destinationType := "AzureDiagnostics"
					if setting.Properties.LogAnalyticsDestinationType != nil {
						destinationType = string(*setting.Properties.LogAnalyticsDestinationType)
					}

					logger.Debugf("Using diagnostic setting '%s' for log category '%s' in cluster '%s': workspaceId=%s, destinationType=%s, isResourceSpecific=%t",
						settingName, logCategory, clusterName, workspaceResourceID, destinationType, isResourceSpecific)

					return workspaceResourceID, isResourceSpecific, nil
				}
			}
		}
	}

	return "", false, fmt.Errorf("no diagnostic setting found with log category '%s' enabled", logCategory)
}
