// Package resourcehandlers provides handler functions for Azure monitoring tools.
package resourcehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/aks-mcp/internal/azure"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"
)

// =============================================================================
// Monitoring-related Handlers
// =============================================================================

// GetLogAnalyticsHandler returns a handler for the query_log_analytics command
func GetLogAnalyticsHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, err := extractStringParam(params, "subscription_id")
		if err != nil {
			return "", err
		}
		
		workspaceID, err := extractStringParam(params, "workspace_id")
		if err != nil {
			return "", err
		}
		
		query, err := extractStringParam(params, "kql_query")
		if err != nil {
			return "", err
		}
		
		// Optional time range parameters (default to last 24 hours)
		timeRangeStr, _ := extractStringParam(params, "time_range")
		if timeRangeStr == "" {
			timeRangeStr = "24h"
		}
		
		// Get clients
		clients, err := client.GetOrCreateClientsForSubscription(subID)
		if err != nil {
			return "", fmt.Errorf("failed to get clients: %v", err)
		}
		
		// Parse time range
		timeRange, err := parseTimeRange(timeRangeStr)
		if err != nil {
			return "", fmt.Errorf("invalid time range: %v", err)
		}
		
		// Execute the KQL query
		ctx := context.Background()
		
		timespan := azlogs.NewTimeInterval(timeRange.Start, timeRange.End)
		
		body := azlogs.QueryBody{
			Query:     &query,
			Timespan:  &timespan,
		}
		
		response, err := clients.LogsQueryClient.QueryWorkspace(ctx, workspaceID, body, nil)
		if err != nil {
			return "", fmt.Errorf("failed to execute query: %v", err)
		}
		
		// Format the response
		result := map[string]interface{}{
			"query":     query,
			"timespan":  fmt.Sprintf("%s to %s", timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339)),
			"tables":    response.Tables,
		}
		
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal response to JSON: %v", err)
		}
		
		return string(resultJSON), nil
	})
}

// GetPrometheusMetricsHandler returns a handler for the query_prometheus_metrics command
func GetPrometheusMetricsHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, err := extractStringParam(params, "subscription_id")
		if err != nil {
			return "", err
		}
		
		resourceGroup, err := extractStringParam(params, "resource_group")
		if err != nil {
			return "", err
		}
		
		clusterName, err := extractStringParam(params, "cluster_name")
		if err != nil {
			return "", err
		}
		
		metricNames, err := extractStringParam(params, "metric_names")
		if err != nil {
			return "", err
		}
		
		// Optional region parameter
		region, _ := extractStringParam(params, "region")
		if region == "" {
			region = "eastus" // Default region
		}
		
		// Optional time range parameters (default to last 1 hour)
		timeRangeStr, _ := extractStringParam(params, "time_range")
		if timeRangeStr == "" {
			timeRangeStr = "1h"
		}
		
		// Parse time range
		timeRange, err := parseTimeRange(timeRangeStr)
		if err != nil {
			return "", fmt.Errorf("invalid time range: %v", err)
		}
		
		// Create metrics query client for the region
		metricsClient, err := client.CreateMetricsQueryClient(region)
		if err != nil {
			return "", fmt.Errorf("failed to create metrics client: %v", err)
		}
		
		// Build resource URI for the AKS cluster
		resourceURI := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s", subID, resourceGroup, clusterName)
		
		// Query metrics
		ctx := context.Background()
		
		resourceIDList := azmetrics.ResourceIDList{
			ResourceIDs: []string{resourceURI},
		}
		
		options := &azmetrics.QueryResourcesOptions{
			StartTime: toPtr(timeRange.Start.Format(time.RFC3339)),
			EndTime:   toPtr(timeRange.End.Format(time.RFC3339)),
			Interval:  toPtr("PT1M"), // 1 minute intervals
		}
		
		response, err := metricsClient.QueryResources(ctx, subID, "Microsoft.ContainerService/managedClusters", []string{metricNames}, resourceIDList, options)
		if err != nil {
			return "", fmt.Errorf("failed to query metrics: %v", err)
		}
		
		// Format the response
		result := map[string]interface{}{
			"resource_uri": resourceURI,
			"metrics":      metricNames,
			"timespan":     fmt.Sprintf("%s to %s", timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339)),
			"results":      response.Values,
		}
		
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal response to JSON: %v", err)
		}
		
		return string(resultJSON), nil
	})
}

// GetApplicationInsightsHandler returns a handler for the query_application_insights command
func GetApplicationInsightsHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, err := extractStringParam(params, "subscription_id")
		if err != nil {
			return "", err
		}
		
		resourceGroup, err := extractStringParam(params, "resource_group")
		if err != nil {
			return "", err
		}
		
		appInsightsName, err := extractStringParam(params, "app_insights_name")
		if err != nil {
			return "", err
		}
		
		query, err := extractStringParam(params, "kql_query")
		if err != nil {
			return "", err
		}
		
		// Optional time range parameters (default to last 24 hours)
		timeRangeStr, _ := extractStringParam(params, "time_range")
		if timeRangeStr == "" {
			timeRangeStr = "24h"
		}
		
		// Parse time range
		timeRange, err := parseTimeRange(timeRangeStr)
		if err != nil {
			return "", fmt.Errorf("invalid time range: %v", err)
		}
		
		// Get clients
		clients, err := client.GetOrCreateClientsForSubscription(subID)
		if err != nil {
			return "", fmt.Errorf("failed to get clients: %v", err)
		}
		
		// Get Application Insights component to get the app ID
		ctx := context.Background()
		component, err := clients.ApplicationInsightsClient.Get(ctx, resourceGroup, appInsightsName, nil)
		if err != nil {
			return "", fmt.Errorf("failed to get Application Insights component: %v", err)
		}
		
		if component.Properties == nil || component.Properties.AppID == nil {
			return "", fmt.Errorf("Application Insights component does not have a valid App ID")
		}
		
		appID := *component.Properties.AppID
		
		// Execute the KQL query against Application Insights
		timespan := azlogs.NewTimeInterval(timeRange.Start, timeRange.End)
		
		body := azlogs.QueryBody{
			Query:    &query,
			Timespan: &timespan,
		}
		
		response, err := clients.LogsQueryClient.QueryWorkspace(ctx, appID, body, nil)
		if err != nil {
			return "", fmt.Errorf("failed to execute Application Insights query: %v", err)
		}
		
		// Format the response
		result := map[string]interface{}{
			"app_insights_name": appInsightsName,
			"app_id":           appID,
			"query":            query,
			"timespan":         fmt.Sprintf("%s to %s", timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339)),
			"tables":           response.Tables,
		}
		
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal response to JSON: %v", err)
		}
		
		return string(resultJSON), nil
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// extractStringParam extracts a string parameter from the params map
func extractStringParam(params map[string]interface{}, key string) (string, error) {
	value, exists := params[key]
	if !exists {
		return "", fmt.Errorf("parameter '%s' is required", key)
	}
	
	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("parameter '%s' must be a string", key)
	}
	
	return strValue, nil
}

// TimeRange represents a time range with start and end times
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// parseTimeRange parses a time range string like "1h", "24h", "7d" into a TimeRange
func parseTimeRange(timeRangeStr string) (*TimeRange, error) {
	now := time.Now()
	
	var duration time.Duration
	var err error
	
	switch timeRangeStr {
	case "1h":
		duration = time.Hour
	case "24h", "1d":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		// Try to parse as a Go duration
		duration, err = time.ParseDuration(timeRangeStr)
		if err != nil {
			return nil, fmt.Errorf("unsupported time range format: %s (use formats like '1h', '24h', '7d', or Go duration format)", timeRangeStr)
		}
	}
	
	return &TimeRange{
		Start: now.Add(-duration),
		End:   now,
	}, nil
}

// toPtr returns a pointer to the given value
func toPtr[T any](v T) *T {
	return &v
}