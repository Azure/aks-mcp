package applens

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// AppLensClient represents a client for interacting with AppLens APIs
type AppLensClient struct {
	subscriptionID string
	credential     azcore.TokenCredential
	httpClient     *http.Client
	baseURL        string
}

// NewAppLensClient creates a new AppLens client
func NewAppLensClient(subscriptionID string, credential *azidentity.DefaultAzureCredential) (*AppLensClient, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription ID cannot be empty")
	}

	return &AppLensClient{
		subscriptionID: subscriptionID,
		credential:     credential,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		baseURL:        "https://management.azure.com",
	}, nil
}

// ListDetectors retrieves a list of available detectors for an AKS cluster
func (c *AppLensClient) ListDetectors(ctx context.Context, clusterResourceID string, category string) ([]DetectorInfo, error) {
	// Extract cluster information from resource ID
	subscriptionID, resourceGroup, clusterName, err := ExtractClusterInfo(clusterResourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster resource ID: %w", err)
	}

	// Build the API URL to list all detectors
	apiURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s/detectors",
		c.baseURL, subscriptionID, resourceGroup, clusterName)

	// Add query parameters
	params := url.Values{}
	params.Add("api-version", "2023-05-01-preview")
	apiURL += "?" + params.Encode()

	// Make the API call
	resp, err := c.makeAuthenticatedRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call AKS detectors API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AKS detectors API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var detectorResponse struct {
		Value []struct {
			Name       string                 `json:"name"`
			ID         string                 `json:"id"`
			Properties map[string]interface{} `json:"properties"`
		} `json:"value"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&detectorResponse); err != nil {
		return nil, fmt.Errorf("failed to parse detectors response: %w", err)
	}

	// Convert to our detector format
	var detectors []DetectorInfo
	for _, det := range detectorResponse.Value {
		detector := DetectorInfo{
			ID:          det.Name,
			Name:        det.Name,
			Description: fmt.Sprintf("AKS detector: %s", det.Name),
			Category:    "aks", // Default category since the API doesn't provide detailed categories
			Metadata: map[string]string{
				"resourceId": det.ID,
			},
		}

		// Add any additional properties from the API response
		if det.Properties != nil {
			if desc, ok := det.Properties["description"].(string); ok && desc != "" {
				detector.Description = desc
			}
			if cat, ok := det.Properties["category"].(string); ok && cat != "" {
				detector.Category = cat
			}
		}

		detectors = append(detectors, detector)
	}

	// Filter by category if specified
	if category != "" {
		var filtered []DetectorInfo
		for _, detector := range detectors {
			if strings.EqualFold(detector.Category, category) {
				filtered = append(filtered, detector)
			}
		}
		return filtered, nil
	}

	return detectors, nil
}

// InvokeDetector executes a specific detector for an AKS cluster
func (c *AppLensClient) InvokeDetector(ctx context.Context, clusterResourceID, detectorName string, options *AppLensOptions) (*DetectorResponse, error) {
	if clusterResourceID == "" {
		return nil, fmt.Errorf("cluster resource ID cannot be empty")
	}
	if detectorName == "" {
		return nil, fmt.Errorf("detector name cannot be empty")
	}

	// Extract cluster information from resource ID
	subscriptionID, resourceGroup, clusterName, err := ExtractClusterInfo(clusterResourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster resource ID: %w", err)
	}

	// Build the API URL for the specific detector
	apiURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s/detectors/%s",
		c.baseURL, subscriptionID, resourceGroup, clusterName, detectorName)

	// Add query parameters
	params := url.Values{}
	params.Add("api-version", "2023-05-01-preview")

	// Set default time range if not provided
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour) // Default to last 24 hours

	if options != nil {
		if options.EndTime != nil {
			endTime = *options.EndTime
		}
		if options.StartTime != nil {
			startTime = *options.StartTime
		} else if options.TimeRange != "" {
			duration, err := parseTimeRange(options.TimeRange)
			if err == nil {
				startTime = endTime.Add(-duration)
			}
		}
	}

	// Add time range parameters
	params.Add("startTime", startTime.Format(time.RFC3339))
	params.Add("endTime", endTime.Format(time.RFC3339))

	apiURL += "?" + params.Encode()

	// Make the API call
	resp, err := c.makeAuthenticatedRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call AKS detector API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AKS detector API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var apiResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Properties struct {
			Dataset  interface{} `json:"dataset"`
			Metadata interface{} `json:"metadata"`
			Status   string      `json:"status"`
		} `json:"properties"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse detector response: %w", err)
	}

	// Convert API response to our format
	response := &DetectorResponse{
		ID:        apiResponse.Name,
		Name:      apiResponse.Name,
		StartTime: startTime,
		EndTime:   endTime,
		Status:    apiResponse.Properties.Status,
		Data:      convertDetectorDataset(apiResponse.Properties.Dataset),
		Insights:  extractInsightsFromResponse(&apiResponse),
		Metadata: map[string]interface{}{
			"clusterResourceId": clusterResourceID,
			"executionTime":     fmt.Sprintf("%.2f seconds", time.Since(startTime).Seconds()),
			"apiResponse":       apiResponse.Properties.Metadata,
		},
	}

	// If status is empty, default to completed
	if response.Status == "" {
		response.Status = "completed"
	}

	return response, nil
}

// makeAuthenticatedRequest makes an authenticated HTTP request to Azure Management API
func (c *AppLensClient) makeAuthenticatedRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Get access token
	tokenRequestOptions := policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	}

	token, err := c.credential.GetToken(ctx, tokenRequestOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	return c.httpClient.Do(req)
}

// convertDetectorDataset converts the API dataset to our internal format
func convertDetectorDataset(dataset interface{}) []DetectorData {
	if dataset == nil {
		return []DetectorData{}
	}

	// Try to convert the dataset to our expected format
	// The actual format depends on the specific detector API response
	dataList := []DetectorData{}

	// This is a simplified conversion - in practice, you'd need to handle
	// the specific structure returned by the AKS detector API
	if dataMap, ok := dataset.(map[string]interface{}); ok {
		data := DetectorData{
			Table: DetectorTable{
				TableName: "DetectorResults",
				Columns: []DetectorColumn{
					{ColumnName: "Property", DataType: "string", ColumnType: "string"},
					{ColumnName: "Value", DataType: "string", ColumnType: "string"},
				},
				Rows: [][]interface{}{},
			},
			RenderingProperties: map[string]interface{}{
				"type":  "table",
				"title": "Detector Results",
			},
		}

		// Convert map entries to table rows
		for key, value := range dataMap {
			data.Table.Rows = append(data.Table.Rows, []interface{}{key, fmt.Sprintf("%v", value)})
		}

		dataList = append(dataList, data)
	}

	return dataList
}

// extractInsightsFromResponse extracts insights from the API response
func extractInsightsFromResponse(apiResponse *struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Properties struct {
		Dataset  interface{} `json:"dataset"`
		Metadata interface{} `json:"metadata"`
		Status   string      `json:"status"`
	} `json:"properties"`
}) []DetectorInsight {
	insights := []DetectorInsight{}

	// Extract insights based on status
	if apiResponse.Properties.Status == "failed" {
		insights = append(insights, DetectorInsight{
			Message: "Detector execution failed",
			Status:  "error",
			Level:   "high",
			Metadata: map[string]interface{}{
				"detector": apiResponse.Name,
			},
		})
	} else if apiResponse.Properties.Status == "completed" {
		insights = append(insights, DetectorInsight{
			Message: "Detector execution completed successfully",
			Status:  "info",
			Level:   "low",
			Metadata: map[string]interface{}{
				"detector": apiResponse.Name,
			},
		})
	}

	// You can add more sophisticated insight extraction logic here
	// based on the actual structure of the dataset

	return insights
}

// ExtractClusterInfo extracts subscription ID, resource group, and cluster name from resource ID
func ExtractClusterInfo(resourceID string) (subscriptionID, resourceGroup, clusterName string, err error) {
	if resourceID == "" {
		return "", "", "", fmt.Errorf("cluster resource ID cannot be empty")
	}

	// Basic validation for AKS cluster resource ID format
	// Expected format: /subscriptions/{subscription-id}/resourceGroups/{resource-group}/providers/Microsoft.ContainerService/managedClusters/{cluster-name}
	parts := strings.Split(resourceID, "/")
	if len(parts) < 9 {
		return "", "", "", fmt.Errorf("invalid resource ID format: expected AKS cluster resource ID")
	}

	if parts[1] != "subscriptions" {
		return "", "", "", fmt.Errorf("invalid resource ID: must start with /subscriptions/")
	}

	if parts[3] != "resourceGroups" {
		return "", "", "", fmt.Errorf("invalid resource ID: missing resourceGroups segment")
	}

	if parts[5] != "providers" {
		return "", "", "", fmt.Errorf("invalid resource ID: missing providers segment")
	}

	if parts[6] != "Microsoft.ContainerService" {
		return "", "", "", fmt.Errorf("invalid resource ID: must be Microsoft.ContainerService provider")
	}

	if parts[7] != "managedClusters" {
		return "", "", "", fmt.Errorf("invalid resource ID: must be managedClusters resource type")
	}

	subscriptionID = parts[2]
	resourceGroup = parts[4]
	clusterName = parts[8]

	return subscriptionID, resourceGroup, clusterName, nil
}

// parseTimeRange converts a time range string to a duration
func parseTimeRange(timeRange string) (time.Duration, error) {
	switch strings.ToLower(timeRange) {
	case "1h", "1hour":
		return time.Hour, nil
	case "6h", "6hours":
		return 6 * time.Hour, nil
	case "12h", "12hours":
		return 12 * time.Hour, nil
	case "24h", "1d", "1day":
		return 24 * time.Hour, nil
	case "7d", "7days", "1w", "1week":
		return 7 * 24 * time.Hour, nil
	case "30d", "30days", "1m", "1month":
		return 30 * 24 * time.Hour, nil
	default:
		// Try to parse as a standard duration
		return time.ParseDuration(timeRange)
	}
}
