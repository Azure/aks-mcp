package applens

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// DetectorManager manages AppLens detector operations
type DetectorManager struct {
	client *AppLensClient
}

// NewDetectorManager creates a new detector manager
func NewDetectorManager(subscriptionID string, credential *azidentity.DefaultAzureCredential) (*DetectorManager, error) {
	client, err := NewAppLensClient(subscriptionID, credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create AppLens client: %w", err)
	}

	return &DetectorManager{
		client: client,
	}, nil
}

// ListDetectors returns a formatted list of available detectors
func (dm *DetectorManager) ListDetectors(ctx context.Context, clusterResourceID string, category string) (string, error) {
	detectors, err := dm.client.ListDetectors(ctx, clusterResourceID, category)
	if err != nil {
		return "", fmt.Errorf("failed to list detectors: %w", err)
	}

	result := map[string]interface{}{
		"clusterResourceId":  clusterResourceID,
		"category":          category,
		"detectorCount":     len(detectors),
		"availableDetectors": detectors,
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal detector list: %w", err)
	}

	return string(resultJSON), nil
}

// InvokeDetector executes a detector and returns formatted results
func (dm *DetectorManager) InvokeDetector(ctx context.Context, clusterResourceID, detectorName string, timeRange string) (string, error) {
	options := &AppLensOptions{}
	if timeRange != "" {
		options.TimeRange = timeRange
	}

	response, err := dm.client.InvokeDetector(ctx, clusterResourceID, detectorName, options)
	if err != nil {
		return "", fmt.Errorf("failed to invoke detector: %w", err)
	}

	// Format the response for better readability
	formattedResponse := formatDetectorResponse(response)

	resultJSON, err := json.MarshalIndent(formattedResponse, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal detector response: %w", err)
	}

	return string(resultJSON), nil
}

// formatDetectorResponse formats the detector response for better readability
func formatDetectorResponse(response *DetectorResponse) map[string]interface{} {
	result := map[string]interface{}{
		"detectorId":     response.ID,
		"detectorName":   response.Name,
		"executionTime": map[string]interface{}{
			"startTime": response.StartTime.Format(time.RFC3339),
			"endTime":   response.EndTime.Format(time.RFC3339),
			"duration":  response.EndTime.Sub(response.StartTime).String(),
		},
		"status":   response.Status,
		"metadata": response.Metadata,
	}

	// Add summary of findings
	if len(response.Insights) > 0 {
		insights := make(map[string]interface{})
		insights["count"] = len(response.Insights)
		
		// Categorize insights by severity
		severityCounts := make(map[string]int)
		var criticalIssues []string
		var warnings []string
		var recommendations []string

		for _, insight := range response.Insights {
			severityCounts[insight.Level]++
			
			switch insight.Level {
			case "high":
				criticalIssues = append(criticalIssues, insight.Message)
			case "medium":
				warnings = append(warnings, insight.Message)
			case "low":
				recommendations = append(recommendations, insight.Message)
			}
		}

		insights["severityCounts"] = severityCounts
		if len(criticalIssues) > 0 {
			insights["criticalIssues"] = criticalIssues
		}
		if len(warnings) > 0 {
			insights["warnings"] = warnings
		}
		if len(recommendations) > 0 {
			insights["recommendations"] = recommendations
		}

		result["insights"] = insights
	}

	// Add data summary
	if len(response.Data) > 0 {
		dataSummary := make(map[string]interface{})
		dataSummary["datasetCount"] = len(response.Data)
		
		var tables []map[string]interface{}
		for _, data := range response.Data {
			if data.Table.TableName != "" {
				tableInfo := map[string]interface{}{
					"tableName":   data.Table.TableName,
					"columnCount": len(data.Table.Columns),
					"rowCount":    len(data.Table.Rows),
					"columns":     extractColumnNames(data.Table.Columns),
				}
				
				// Add sample data if available
				if len(data.Table.Rows) > 0 && len(data.Table.Rows) <= 5 {
					tableInfo["sampleData"] = data.Table.Rows
				} else if len(data.Table.Rows) > 5 {
					tableInfo["sampleData"] = data.Table.Rows[:5]
					tableInfo["note"] = fmt.Sprintf("Showing first 5 rows of %d total rows", len(data.Table.Rows))
				}
				
				tables = append(tables, tableInfo)
			}
		}
		
		if len(tables) > 0 {
			dataSummary["tables"] = tables
		}
		
		result["dataSummary"] = dataSummary
	}

	// Add detailed insights if available
	if len(response.Insights) > 0 {
		result["detailedInsights"] = response.Insights
	}

	return result
}

// extractColumnNames extracts column names from detector columns
func extractColumnNames(columns []DetectorColumn) []string {
	names := make([]string, len(columns))
	for i, col := range columns {
		names[i] = col.ColumnName
	}
	return names
}