package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/components/common"
	"github.com/Azure/aks-mcp/internal/components/monitor/diagnostics"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
)

// mergeMonitoringParams merges top-level parameters with nested "parameters" JSON string
func mergeMonitoringParams(params map[string]interface{}) (map[string]interface{}, error) {
	merged := make(map[string]interface{})

	// Copy all top-level parameters first
	for key, value := range params {
		merged[key] = value
	}

	// Parse and merge parameters from JSON string if present
	if parametersStr, ok := params["parameters"].(string); ok && parametersStr != "" {
		var nestedParams map[string]interface{}
		if err := json.Unmarshal([]byte(parametersStr), &nestedParams); err == nil {
			// Merge nested parameters, giving precedence to top-level params
			for key, value := range nestedParams {
				if _, exists := merged[key]; !exists {
					merged[key] = value
				}
			}
		}
	}

	return merged, nil
}

// HandleResourceHealthQuery handles the resource health query for AKS clusters
func HandleResourceHealthQuery(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Extract and validate parameters
	subscriptionID, ok := params["subscription_id"].(string)
	if !ok || subscriptionID == "" {
		var err error
		subscriptionID, err = common.GetDefaultSubscriptionID(cfg)
		if err != nil {
			return "", err
		}
	}

	resourceGroup, ok := params["resource_group"].(string)
	if !ok || resourceGroup == "" {
		return "", fmt.Errorf("missing or invalid resource_group parameter")
	}

	clusterName, ok := params["cluster_name"].(string)
	if !ok || clusterName == "" {
		return "", fmt.Errorf("missing or invalid cluster_name parameter")
	}

	startTime, ok := params["start_time"].(string)
	if !ok || startTime == "" {
		return "", fmt.Errorf("missing or invalid start_time parameter")
	}

	// Validate parameters
	if err := validateResourceHealthParams(params); err != nil {
		return "", err
	}

	// Build resource ID
	resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
		subscriptionID, resourceGroup, clusterName)

	// Build Azure CLI command
	executor := azcli.NewExecutor()
	args := []string{
		"monitor", "activity-log", "list",
		"--resource-id", resourceID,
		"--start-time", startTime,
		"--query", "[?category.value=='ResourceHealth']",
		"--output", "json",
	}

	// Add end time if provided
	if endTime, ok := params["end_time"].(string); ok && endTime != "" {
		args = append(args, "--end-time", endTime)
	}

	// Add status filter if provided
	if status, ok := params["status"].(string); ok && status != "" {
		// Apply status filter in the query
		statusFilter := fmt.Sprintf("[?category.value=='ResourceHealth' && properties.currentHealthStatus=='%s']", status)
		args[len(args)-3] = statusFilter // Replace the query parameter
	}

	// Execute command
	cmdParams := map[string]interface{}{
		"command": "az " + strings.Join(args, " "),
	}

	result, err := executor.Execute(ctx, cmdParams, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to execute resource health query: %w", err)
	}

	// Return the raw JSON result from Azure CLI
	return result, nil
}

// validateResourceHealthParams validates the parameters for resource health queries
func validateResourceHealthParams(params map[string]interface{}) error {
	// Validate required parameters
	required := []string{"subscription_id", "resource_group", "cluster_name", "start_time"}
	for _, param := range required {
		if value, ok := params[param].(string); !ok || value == "" {
			return fmt.Errorf("missing or invalid %s parameter", param)
		}
	}

	// Validate time format
	startTime := params["start_time"].(string)
	if _, err := time.Parse(time.RFC3339, startTime); err != nil {
		return fmt.Errorf("invalid start_time format, expected RFC3339 (ISO 8601): %w", err)
	}

	// Validate end_time if provided
	if endTime, ok := params["end_time"].(string); ok && endTime != "" {
		if _, err := time.Parse(time.RFC3339, endTime); err != nil {
			return fmt.Errorf("invalid end_time format, expected RFC3339 (ISO 8601): %w", err)
		}
	}

	// Validate status if provided
	if status, ok := params["status"].(string); ok && status != "" {
		validStatuses := []string{"Available", "Unavailable", "Degraded", "Unknown"}
		valid := false
		for _, validStatus := range validStatuses {
			if status == validStatus {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid status parameter, must be one of: %s", strings.Join(validStatuses, ", "))
		}
	}

	return nil
}

// GetResourceHealthHandler returns a ResourceHandler for the resource health tool
func GetResourceHealthHandler(cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return HandleResourceHealthQuery(ctx, params, cfg)
	})
}

// HandleAppInsightsQuery handles Application Insights telemetry queries for AKS clusters
func HandleAppInsightsQuery(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Extract and validate parameters
	subscriptionID, ok := params["subscription_id"].(string)
	if !ok || subscriptionID == "" {
		var err error
		subscriptionID, err = common.GetDefaultSubscriptionID(cfg)
		if err != nil {
			return "", err
		}
	}

	resourceGroup, ok := params["resource_group"].(string)
	if !ok || resourceGroup == "" {
		return "", fmt.Errorf("missing or invalid resource_group parameter")
	}

	appInsightsName, ok := params["app_insights_name"].(string)
	if !ok || appInsightsName == "" {
		return "", fmt.Errorf("missing or invalid app_insights_name parameter")
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing or invalid query parameter")
	}

	// Validate parameters
	if err := validateAppInsightsParams(params); err != nil {
		return "", err
	}

	// Build Application Insights resource ID
	appResourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Insights/components/%s",
		subscriptionID, resourceGroup, appInsightsName)

	// Build Azure CLI command
	executor := azcli.NewExecutor()
	args := []string{
		"monitor", "app-insights", "query",
		"--app", appResourceID,
		"--analytics-query", query,
		"--output", "json",
	}

	// Add start time if provided
	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		args = append(args, "--start-time", startTime)
	}

	// Add end time if provided
	if endTime, ok := params["end_time"].(string); ok && endTime != "" {
		args = append(args, "--end-time", endTime)
	}

	// Add timespan if provided
	if timespan, ok := params["timespan"].(string); ok && timespan != "" {
		args = append(args, "--timespan", timespan)
	}

	// Execute command
	cmdParams := map[string]interface{}{
		"command": "az " + strings.Join(args, " "),
	}

	result, err := executor.Execute(ctx, cmdParams, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to execute Application Insights query: %w", err)
	}

	// Return the raw JSON result from Azure CLI
	return result, nil
}

// validateAppInsightsParams validates the parameters for Application Insights queries
func validateAppInsightsParams(params map[string]interface{}) error {
	// Validate required parameters
	required := []string{"subscription_id", "resource_group", "app_insights_name", "query"}
	for _, param := range required {
		if value, ok := params[param].(string); !ok || value == "" {
			return fmt.Errorf("missing or invalid %s parameter", param)
		}
	}

	// Validate time format for start_time if provided
	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		if _, err := time.Parse(time.RFC3339, startTime); err != nil {
			return fmt.Errorf("invalid start_time format, expected RFC3339 (ISO 8601): %w", err)
		}
	}

	// Validate time format for end_time if provided
	if endTime, ok := params["end_time"].(string); ok && endTime != "" {
		if _, err := time.Parse(time.RFC3339, endTime); err != nil {
			return fmt.Errorf("invalid end_time format, expected RFC3339 (ISO 8601): %w", err)
		}
	}

	// Validate timespan format if provided (basic validation for ISO 8601 duration)
	if timespan, ok := params["timespan"].(string); ok && timespan != "" {
		if !strings.HasPrefix(timespan, "P") && !strings.HasPrefix(timespan, "PT") {
			return fmt.Errorf("invalid timespan format, expected ISO 8601 duration (e.g., PT1H, P1D)")
		}
	}

	return nil
}

// GetAppInsightsHandler returns a ResourceHandler for the Application Insights tool
func GetAppInsightsHandler(cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return HandleAppInsightsQuery(ctx, params, cfg)
	})
}

// GetAzMonitoringHandler returns a ResourceHandler for the monitoring tool
func GetAzMonitoringHandler(azClient *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract operation parameter
		operation, ok := params["operation"].(string)
		if !ok {
			return "", fmt.Errorf("missing or invalid 'operation' parameter")
		}

		// Validate operation
		if !ValidateMonitoringOperation(operation) {
			supportedOps := GetSupportedMonitoringOperations()
			return "", fmt.Errorf("unsupported operation: %s. Supported operations: %v", operation, supportedOps)
		}

		// Handle different operations
		switch operation {
		case string(OpMetrics):
			return handleMetricsOperation(ctx, params, cfg)
		case string(OpResourceHealth):
			return handleResourceHealthOperation(ctx, params, cfg)
		case string(OpAppInsights):
			return handleAppInsightsOperation(ctx, params, cfg)
		case string(OpDiagnostics):
			return handleDiagnosticsOperation(ctx, params, azClient, cfg)
		case string(OpControlPlaneLogs):
			return handleLogsOperation(ctx, params, azClient, cfg)
		default:
			return "", fmt.Errorf("operation '%s' not implemented", operation)
		}
	})
}

// Helper functions for different monitoring operations

func handleMetricsOperation(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	queryType, ok := params["query_type"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'query_type' parameter for metrics operation")
	}

	if !ValidateMetricsQueryType(queryType) {
		return "", fmt.Errorf("invalid query_type: %s. Supported types: list, list-definitions, list-namespaces", queryType)
	}

	// Extract parameters from JSON string
	parametersStr, ok := params["parameters"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'parameters' parameter")
	}

	// Parse JSON parameters and convert to command-line arguments
	var jsonParams map[string]interface{}
	if err := json.Unmarshal([]byte(parametersStr), &jsonParams); err != nil {
		return "", fmt.Errorf("failed to parse parameters JSON: %w", err)
	}

	// Convert JSON parameters to command-line argument format [--key1 value1 --key2 value2]
	var args []string
	for key, value := range jsonParams {
		args = append(args, "--"+key)
		args = append(args, fmt.Sprintf("%v", value))
	}

	args = ensureOutputArg(args)

	// Map query type to command
	baseCommand, err := MapMetricsQueryTypeToCommand(queryType)
	if err != nil {
		return "", err
	}

	// Execute the command
	executor := azcli.NewExecutor()
	cmdParams := map[string]interface{}{
		"command": baseCommand + " " + strings.Join(args, " "),
	}

	return executor.Execute(ctx, cmdParams, cfg)
}

func handleResourceHealthOperation(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Merge parameters from top-level and nested JSON
	mergedParams, err := mergeMonitoringParams(params)
	if err != nil {
		return "", fmt.Errorf("failed to merge parameters: %w", err)
	}

	// Use existing resource health handler
	return GetResourceHealthHandler(cfg).Handle(ctx, mergedParams, cfg)
}

func handleAppInsightsOperation(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Merge parameters from top-level and nested JSON
	mergedParams, err := mergeMonitoringParams(params)
	if err != nil {
		return "", fmt.Errorf("failed to merge parameters: %w", err)
	}

	// Use existing app insights handler
	return GetAppInsightsHandler(cfg).Handle(ctx, mergedParams, cfg)
}

func handleDiagnosticsOperation(ctx context.Context, params map[string]interface{}, azClient *azureclient.AzureClient, cfg *config.ConfigData) (string, error) {
	// Merge parameters from top-level and nested JSON
	mergedParams, err := mergeMonitoringParams(params)
	if err != nil {
		return "", fmt.Errorf("failed to merge parameters: %w", err)
	}

	// Use existing control plane diagnostics handler
	return diagnostics.GetControlPlaneDiagnosticSettingsHandler(azClient, cfg).Handle(ctx, mergedParams, cfg)
}

func handleLogsOperation(ctx context.Context, params map[string]interface{}, azClient *azureclient.AzureClient, cfg *config.ConfigData) (string, error) {
	// Merge parameters from top-level and nested JSON
	mergedParams, err := mergeMonitoringParams(params)
	if err != nil {
		return "", fmt.Errorf("failed to merge parameters: %w", err)
	}

	// Use existing control plane logs handler
	return diagnostics.GetControlPlaneLogsHandler(azClient, cfg).Handle(ctx, mergedParams, cfg)
}

// ensureOutputArg appends --output json when no explicit output flag is present
func ensureOutputArg(args []string) []string {
	for _, arg := range args {
		lower := strings.ToLower(arg)
		switch {
		case lower == "--output":
			return args
		case strings.HasPrefix(lower, "--output="):
			return args
		case lower == "-o":
			return args
		case strings.HasPrefix(lower, "-o") && len(lower) > 2:
			return args
		}
	}

	return append(args, "--output", "json")
}
