package diagnostics

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/config"
)

// ExtractWorkspaceGUIDFromDiagnosticSettings extracts workspace GUID from diagnostic settings
func ExtractWorkspaceGUIDFromDiagnosticSettings(subscriptionID, resourceGroup, clusterName string, cfg *config.ConfigData) (string, error) {
	// Get diagnostic settings using common parameter structure
	params := map[string]interface{}{
		"subscription_id": subscriptionID,
		"resource_group":  resourceGroup,
		"cluster_name":    clusterName,
	}

	diagnosticResult, err := HandleControlPlaneDiagnosticSettings(params, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to get diagnostic settings: %w", err)
	}

	// Parse to extract workspace ID
	var parsed interface{}
	if err := json.Unmarshal([]byte(diagnosticResult), &parsed); err != nil {
		return "", fmt.Errorf("failed to parse diagnostic settings JSON: %w", err)
	}

	// Handle both array and object formats
	var settings []interface{}

	// Check if it's an array (direct diagnostic settings response)
	if settingsArray, ok := parsed.([]interface{}); ok {
		settings = settingsArray
	} else if parsedObj, ok := parsed.(map[string]interface{}); ok {
		// Check if it's wrapped in a "value" property
		if value, ok := parsedObj["value"].([]interface{}); ok {
			settings = value
		}
	}

	// Extract workspace resource ID from the first diagnostic setting
	if len(settings) > 0 {
		if setting, ok := settings[0].(map[string]interface{}); ok {
			if workspaceResourceID, ok := setting["workspaceId"].(string); ok && workspaceResourceID != "" {
				// Extract workspace GUID from the workspace resource ID
				return getWorkspaceGUID(workspaceResourceID, cfg)
			}
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
func FindDiagnosticSettingForCategory(subscriptionID, resourceGroup, clusterName, logCategory string, cfg *config.ConfigData) (string, bool, error) {
	// Get diagnostic settings using common parameter structure
	params := map[string]interface{}{
		"subscription_id": subscriptionID,
		"resource_group":  resourceGroup,
		"cluster_name":    clusterName,
	}

	diagnosticResult, err := HandleControlPlaneDiagnosticSettings(params, cfg)
	if err != nil {
		return "", false, fmt.Errorf("failed to get diagnostic settings: %w", err)
	}

	// Parse to extract diagnostic settings
	var parsed interface{}
	if err := json.Unmarshal([]byte(diagnosticResult), &parsed); err != nil {
		return "", false, fmt.Errorf("failed to parse diagnostic settings JSON: %w", err)
	}

	// Handle both array and object formats
	var settings []interface{}

	// Check if it's an array (direct diagnostic settings response)
	if settingsArray, ok := parsed.([]interface{}); ok {
		settings = settingsArray
	} else if parsedObj, ok := parsed.(map[string]interface{}); ok {
		// Check if it's wrapped in a "value" property
		if value, ok := parsedObj["value"].([]interface{}); ok {
			settings = value
		}
	}

	// Find the first diagnostic setting that has the requested log category enabled
	for _, settingInterface := range settings {
		if setting, ok := settingInterface.(map[string]interface{}); ok {
			// Check if this setting has logs configuration
			if logs, ok := setting["logs"].([]interface{}); ok {
				// Check each log category in this setting
				for _, logInterface := range logs {
					if logConfig, ok := logInterface.(map[string]interface{}); ok {
						if category, ok := logConfig["category"].(string); ok && category == logCategory {
							if enabled, ok := logConfig["enabled"].(bool); ok && enabled {
								// Found the category and it's enabled, now get workspace and table mode
								workspaceResourceID, ok := setting["workspaceId"].(string)
								if !ok || workspaceResourceID == "" {
									continue // Skip if no workspace configured
								}

								// Determine table mode from logAnalyticsDestinationType
								isResourceSpecific := false
								if destinationType, ok := setting["logAnalyticsDestinationType"].(string); ok {
									isResourceSpecific = strings.ToLower(destinationType) == "dedicated"
								}

								// Get diagnostic setting name for debugging
								settingName := "unknown"
								if name, ok := setting["name"].(string); ok {
									settingName = name
								}

								// Debug log which setting and workspace is being used
								log.Printf("Using diagnostic setting '%s' for log category '%s' in cluster '%s': workspaceId=%s, destinationType=%s, isResourceSpecific=%t",
									settingName, logCategory, clusterName, workspaceResourceID,
									setting["logAnalyticsDestinationType"], isResourceSpecific)

								return workspaceResourceID, isResourceSpecific, nil
							}
						}
					}
				}
			}
		}
	}

	return "", false, fmt.Errorf("no diagnostic setting found with log category '%s' enabled", logCategory)
}
