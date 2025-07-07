package resourcehandlers

import (
	"testing"
	"time"
)

func TestExtractStringParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		key         string
		expected    string
		expectError bool
	}{
		{
			name: "valid string parameter",
			params: map[string]interface{}{
				"test_key": "test_value",
			},
			key:         "test_key",
			expected:    "test_value",
			expectError: false,
		},
		{
			name: "missing parameter",
			params: map[string]interface{}{
				"other_key": "other_value",
			},
			key:         "test_key",
			expectError: true,
		},
		{
			name: "non-string parameter",
			params: map[string]interface{}{
				"test_key": 123,
			},
			key:         "test_key",
			expectError: true,
		},
		{
			name: "empty string parameter",
			params: map[string]interface{}{
				"test_key": "",
			},
			key:         "test_key",
			expected:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractStringParam(tt.params, tt.key)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestParseTimeRange(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name         string
		timeRangeStr string
		expectError  bool
		expectedDiff time.Duration
	}{
		{
			name:         "1 hour",
			timeRangeStr: "1h",
			expectError:  false,
			expectedDiff: time.Hour,
		},
		{
			name:         "24 hours",
			timeRangeStr: "24h",
			expectError:  false,
			expectedDiff: 24 * time.Hour,
		},
		{
			name:         "1 day",
			timeRangeStr: "1d",
			expectError:  false,
			expectedDiff: 24 * time.Hour,
		},
		{
			name:         "7 days",
			timeRangeStr: "7d",
			expectError:  false,
			expectedDiff: 7 * 24 * time.Hour,
		},
		{
			name:         "30 days",
			timeRangeStr: "30d",
			expectError:  false,
			expectedDiff: 30 * 24 * time.Hour,
		},
		{
			name:         "Go duration format",
			timeRangeStr: "2h30m",
			expectError:  false,
			expectedDiff: 2*time.Hour + 30*time.Minute,
		},
		{
			name:         "invalid format",
			timeRangeStr: "invalid",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeRange(tt.timeRangeStr)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Check if the time range is approximately correct (within 1 second tolerance)
				actualDiff := result.End.Sub(result.Start)
				if abs(actualDiff-tt.expectedDiff) > time.Second {
					t.Errorf("Expected duration %v, got %v", tt.expectedDiff, actualDiff)
				}

				// Check if End is after Start
				if !result.End.After(result.Start) {
					t.Errorf("End time should be after Start time")
				}

				// Check if End is approximately now (within 1 second tolerance)
				if abs(result.End.Sub(now)) > time.Second {
					t.Errorf("End time should be approximately now")
				}
			}
		})
	}
}

func TestLogAnalyticsParameters(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"workspace_id":    "workspace-456",
				"kql_query":       "ContainerLog | limit 10",
				"time_range":      "1h",
			},
			expectError: false,
		},
		{
			name: "missing subscription_id",
			params: map[string]interface{}{
				"workspace_id": "workspace-456",
				"kql_query":    "ContainerLog | limit 10",
			},
			expectError: true,
		},
		{
			name: "missing workspace_id",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"kql_query":       "ContainerLog | limit 10",
			},
			expectError: true,
		},
		{
			name: "missing kql_query",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"workspace_id":    "workspace-456",
			},
			expectError: true,
		},
		{
			name: "optional time_range",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"workspace_id":    "workspace-456",
				"kql_query":       "ContainerLog | limit 10",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test subscription_id extraction
			_, err := extractStringParam(tt.params, "subscription_id")
			if tt.expectError && (tt.name == "missing subscription_id") {
				if err == nil {
					t.Errorf("Expected error for missing subscription_id but got none")
				}
				return
			}

			// Test workspace_id extraction
			_, err = extractStringParam(tt.params, "workspace_id")
			if tt.expectError && (tt.name == "missing workspace_id") {
				if err == nil {
					t.Errorf("Expected error for missing workspace_id but got none")
				}
				return
			}

			// Test kql_query extraction
			_, err = extractStringParam(tt.params, "kql_query")
			if tt.expectError && (tt.name == "missing kql_query") {
				if err == nil {
					t.Errorf("Expected error for missing kql_query but got none")
				}
				return
			}

			// Test optional time_range extraction
			timeRange, _ := extractStringParam(tt.params, "time_range")
			if timeRange == "" {
				timeRange = "24h" // default
			}

			// Validate time range
			_, err = parseTimeRange(timeRange)
			if err != nil {
				t.Errorf("Error parsing time range: %v", err)
			}
		})
	}
}

func TestPrometheusMetricsParameters(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"resource_group":  "rg-test",
				"cluster_name":    "cluster-test",
				"metric_names":    "node_cpu_usage_millicores,node_memory_working_set_bytes",
				"region":          "eastus",
				"time_range":      "1h",
			},
			expectError: false,
		},
		{
			name: "missing metric_names",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"resource_group":  "rg-test",
				"cluster_name":    "cluster-test",
			},
			expectError: true,
		},
		{
			name: "optional region and time_range",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"resource_group":  "rg-test",
				"cluster_name":    "cluster-test",
				"metric_names":    "node_cpu_usage_millicores",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test required parameters
			requiredParams := []string{"subscription_id", "resource_group", "cluster_name", "metric_names"}
			for _, param := range requiredParams {
				_, err := extractStringParam(tt.params, param)
				if tt.expectError && (tt.name == "missing "+param) {
					if err == nil {
						t.Errorf("Expected error for missing %s but got none", param)
					}
					return
				}
			}

			// Test optional parameters with defaults
			region, _ := extractStringParam(tt.params, "region")
			if region == "" {
				region = "eastus"
			}

			timeRange, _ := extractStringParam(tt.params, "time_range")
			if timeRange == "" {
				timeRange = "1h"
			}

			// Validate time range
			_, err := parseTimeRange(timeRange)
			if err != nil {
				t.Errorf("Error parsing time range: %v", err)
			}
		})
	}
}

func TestApplicationInsightsParameters(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"subscription_id":    "sub-123",
				"resource_group":     "rg-test",
				"app_insights_name":  "appinsights-test",
				"kql_query":          "requests | limit 10",
				"time_range":         "24h",
			},
			expectError: false,
		},
		{
			name: "missing app_insights_name",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"resource_group":  "rg-test",
				"kql_query":       "requests | limit 10",
			},
			expectError: true,
		},
		{
			name: "optional time_range",
			params: map[string]interface{}{
				"subscription_id":   "sub-123",
				"resource_group":    "rg-test",
				"app_insights_name": "appinsights-test",
				"kql_query":         "requests | limit 10",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test required parameters
			requiredParams := []string{"subscription_id", "resource_group", "app_insights_name", "kql_query"}
			for _, param := range requiredParams {
				_, err := extractStringParam(tt.params, param)
				if tt.expectError && (tt.name == "missing "+param) {
					if err == nil {
						t.Errorf("Expected error for missing %s but got none", param)
					}
					return
				}
			}

			// Test optional time_range with default
			timeRange, _ := extractStringParam(tt.params, "time_range")
			if timeRange == "" {
				timeRange = "24h"
			}

			// Validate time range
			_, err := parseTimeRange(timeRange)
			if err != nil {
				t.Errorf("Error parsing time range: %v", err)
			}
		})
	}
}

// Helper function to calculate absolute difference between durations
func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}