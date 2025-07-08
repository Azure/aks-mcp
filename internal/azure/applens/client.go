package applens

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
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
	// For now, return a mock list of common AKS detectors
	// In a real implementation, this would call the AppLens API
	detectors := []DetectorInfo{
		{
			ID:          "cluster-health",
			Name:        "Cluster Health Check",
			Description: "Analyzes overall cluster health status and identifies potential issues",
			Category:    "reliability",
			Metadata: map[string]string{
				"estimatedTime": "2-3 minutes",
				"complexity":    "medium",
			},
		},
		{
			ID:          "node-performance",
			Name:        "Node Performance Analysis",
			Description: "Evaluates node performance metrics and resource utilization",
			Category:    "performance",
			Metadata: map[string]string{
				"estimatedTime": "3-5 minutes",
				"complexity":    "high",
			},
		},
		{
			ID:          "network-connectivity",
			Name:        "Network Connectivity Check",
			Description: "Validates network connectivity and DNS resolution within the cluster",
			Category:    "reliability",
			Metadata: map[string]string{
				"estimatedTime": "1-2 minutes",
				"complexity":    "low",
			},
		},
		{
			ID:          "security-assessment",
			Name:        "Security Configuration Assessment",
			Description: "Reviews cluster security settings and identifies potential vulnerabilities",
			Category:    "security",
			Metadata: map[string]string{
				"estimatedTime": "4-6 minutes",
				"complexity":    "high",
			},
		},
		{
			ID:          "resource-utilization",
			Name:        "Resource Utilization Analysis",
			Description: "Analyzes CPU, memory, and storage utilization across the cluster",
			Category:    "performance",
			Metadata: map[string]string{
				"estimatedTime": "2-3 minutes",
				"complexity":    "medium",
			},
		},
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

	// For now, return mock detector results
	// In a real implementation, this would call the AppLens API
	response := &DetectorResponse{
		ID:        detectorName,
		Name:      detectorName,
		StartTime: startTime,
		EndTime:   endTime,
		Status:    "completed",
		Data:      generateMockDetectorData(detectorName),
		Insights:  generateMockInsights(detectorName),
		Metadata: map[string]interface{}{
			"clusterResourceId": clusterResourceID,
			"executionTime":     "2.3 seconds",
			"dataPoints":        42,
		},
	}

	return response, nil
}

// generateMockDetectorData creates sample detector data for testing
func generateMockDetectorData(detectorName string) []DetectorData {
	switch detectorName {
	case "cluster-health":
		return []DetectorData{
			{
				Table: DetectorTable{
					TableName: "HealthMetrics",
					Columns: []DetectorColumn{
						{ColumnName: "Component", DataType: "string", ColumnType: "string"},
						{ColumnName: "Status", DataType: "string", ColumnType: "string"},
						{ColumnName: "Score", DataType: "int", ColumnType: "int"},
					},
					Rows: [][]interface{}{
						{"API Server", "Healthy", 95},
						{"etcd", "Healthy", 98},
						{"Scheduler", "Warning", 85},
						{"Controller Manager", "Healthy", 92},
					},
				},
				RenderingProperties: map[string]interface{}{
					"type":  "table",
					"title": "Cluster Component Health",
				},
			},
		}
	case "node-performance":
		return []DetectorData{
			{
				Table: DetectorTable{
					TableName: "NodeMetrics",
					Columns: []DetectorColumn{
						{ColumnName: "NodeName", DataType: "string", ColumnType: "string"},
						{ColumnName: "CPUUsage", DataType: "double", ColumnType: "double"},
						{ColumnName: "MemoryUsage", DataType: "double", ColumnType: "double"},
						{ColumnName: "DiskUsage", DataType: "double", ColumnType: "double"},
					},
					Rows: [][]interface{}{
						{"aks-nodepool1-12345", 65.2, 78.5, 45.1},
						{"aks-nodepool1-67890", 72.8, 82.3, 51.7},
						{"aks-nodepool1-11111", 58.9, 69.4, 38.2},
					},
				},
				RenderingProperties: map[string]interface{}{
					"type":  "table",
					"title": "Node Performance Metrics",
				},
			},
		}
	default:
		return []DetectorData{
			{
				Table: DetectorTable{
					TableName: "GenericResults",
					Columns: []DetectorColumn{
						{ColumnName: "Metric", DataType: "string", ColumnType: "string"},
						{ColumnName: "Value", DataType: "string", ColumnType: "string"},
					},
					Rows: [][]interface{}{
						{"Status", "Analysis completed"},
						{"Issues Found", "0"},
					},
				},
				RenderingProperties: map[string]interface{}{
					"type":  "table",
					"title": "Detection Results",
				},
			},
		}
	}
}

// generateMockInsights creates sample insights for testing
func generateMockInsights(detectorName string) []DetectorInsight {
	switch detectorName {
	case "cluster-health":
		return []DetectorInsight{
			{
				Message: "Scheduler component showing warning status due to high latency",
				Status:  "warning",
				Level:   "medium",
				Metadata: map[string]interface{}{
					"component":    "scheduler",
					"latency":      "120ms",
					"recommended":  "Monitor and consider scaling if latency persists",
				},
			},
		}
	case "node-performance":
		return []DetectorInsight{
			{
				Message: "Node aks-nodepool1-67890 has high memory utilization",
				Status:  "warning",
				Level:   "medium",
				Metadata: map[string]interface{}{
					"node":         "aks-nodepool1-67890",
					"memoryUsage":  "82.3%",
					"threshold":    "80%",
					"recommended":  "Consider adding more nodes or optimizing workloads",
				},
			},
		}
	case "security-assessment":
		return []DetectorInsight{
			{
				Message: "Pod Security Standards not enforced on some namespaces",
				Status:  "warning",
				Level:   "high",
				Metadata: map[string]interface{}{
					"affectedNamespaces": []string{"default", "kube-public"},
					"recommended":        "Implement Pod Security Standards across all namespaces",
				},
			},
		}
	default:
		return []DetectorInsight{
			{
				Message: "Analysis completed successfully with no critical issues found",
				Status:  "info",
				Level:   "low",
			},
		}
	}
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