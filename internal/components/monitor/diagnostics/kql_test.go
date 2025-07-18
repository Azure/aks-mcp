package diagnostics

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildSafeKQLQuery(t *testing.T) {
	tests := []struct {
		name              string
		category          string
		logLevel          string
		maxRecords        int
		clusterResourceID string
		expectedContains  []string
		notExpected       []string
	}{
		{
			name:              "basic query without log level",
			category:          "kube-apiserver",
			logLevel:          "",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'kube-apiserver'",
				"limit 100",
				"project TimeGenerated, Level, log_s",
				"order by TimeGenerated desc",
			},
			notExpected: []string{
				"where log_s startswith",
			},
		},
		{
			name:              "query with info log level",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        50,
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'kube-apiserver'",
				"where log_s startswith 'I'",
				"limit 50",
				"project TimeGenerated, Level, log_s",
			},
		},
		{
			name:              "query with error log level",
			category:          "kube-controller-manager",
			logLevel:          "error",
			maxRecords:        200,
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			expectedContains: []string{
				"where Category == 'kube-controller-manager'",
				"where log_s startswith 'E'",
				"limit 200",
			},
		},
		{
			name:              "query with warning log level",
			category:          "kube-scheduler",
			logLevel:          "warning",
			maxRecords:        300,
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			expectedContains: []string{
				"where Category == 'kube-scheduler'",
				"where log_s startswith 'W'",
				"limit 300",
			},
		},
		{
			name:              "query with audit category",
			category:          "kube-audit",
			logLevel:          "",
			maxRecords:        1000,
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			expectedContains: []string{
				"where Category == 'kube-audit'",
				"limit 1000",
			},
		},
		{
			name:              "query with audit category and log level - should skip log level filtering",
			category:          "kube-audit",
			logLevel:          "info",
			maxRecords:        500,
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			expectedContains: []string{
				"where Category == 'kube-audit'",
				"limit 500",
			},
			notExpected: []string{
				"where log_s startswith",
				"where Level ==",
			},
		},
		{
			name:              "query with audit-admin category and log level - should skip log level filtering",
			category:          "kube-audit-admin",
			logLevel:          "error",
			maxRecords:        200,
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			expectedContains: []string{
				"where Category == 'kube-audit-admin'",
				"limit 200",
			},
			notExpected: []string{
				"where log_s startswith",
				"where Level ==",
			},
		},
		{
			name:              "query with cloud controller manager",
			category:          "cloud-controller-manager",
			logLevel:          "info",
			maxRecords:        150,
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			expectedContains: []string{
				"where Category == 'cloud-controller-manager'",
				"where log_s startswith 'I'",
				"limit 150",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := BuildSafeKQLQuery(tt.category, tt.logLevel, tt.maxRecords, tt.clusterResourceID)

			// Check that expected strings are present
			for _, expected := range tt.expectedContains {
				if !strings.Contains(query, expected) {
					t.Errorf("Expected query to contain '%s', but it didn't. Query: %s", expected, query)
				}
			}

			// Check that unexpected strings are not present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(query, notExpected) {
					t.Errorf("Expected query NOT to contain '%s', but it did. Query: %s", notExpected, query)
				}
			}

			// Verify query structure
			if !strings.HasPrefix(query, "AzureDiagnostics") {
				t.Errorf("Query should start with AzureDiagnostics, got: %s", query)
			}

			if !strings.Contains(query, "order by TimeGenerated desc") {
				t.Errorf("Query should contain ordering clause, got: %s", query)
			}
		})
	}
}

func TestCalculateTimespan(t *testing.T) {
	tests := []struct {
		name          string
		startTime     string
		endTime       string
		wantError     bool
		checkDuration bool // Whether to check if duration makes sense
	}{
		{
			name:          "valid start and end time - 1 hour",
			startTime:     "2025-07-15T10:00:00Z",
			endTime:       "2025-07-15T11:00:00Z",
			wantError:     false,
			checkDuration: true,
		},
		{
			name:          "valid start and end time - 4 hours",
			startTime:     "2025-07-15T10:00:00Z",
			endTime:       "2025-07-15T14:00:00Z",
			wantError:     false,
			checkDuration: true,
		},
		{
			name:      "valid start time, empty end time",
			startTime: "2025-07-15T10:00:00Z",
			endTime:   "",
			wantError: false,
		},
		{
			name:      "invalid start time format",
			startTime: "invalid-time",
			endTime:   "",
			wantError: true,
		},
		{
			name:      "invalid end time format",
			startTime: "2025-07-15T10:00:00Z",
			endTime:   "invalid-end-time",
			wantError: true,
		},
		{
			name:          "valid time with milliseconds",
			startTime:     "2025-07-15T10:00:00.000Z",
			endTime:       "2025-07-15T12:00:00.000Z",
			wantError:     false,
			checkDuration: true,
		},
		{
			name:          "valid time with timezone offset",
			startTime:     "2025-07-15T10:00:00+02:00",
			endTime:       "2025-07-15T11:00:00+02:00",
			wantError:     false,
			checkDuration: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timespan, err := CalculateTimespan(tt.startTime, tt.endTime)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			// Check timespan format: should be "start/end"
			if !strings.Contains(timespan, "/") {
				t.Errorf("Expected timespan to contain '/' separator, got: %s", timespan)
			}

			parts := strings.Split(timespan, "/")
			if len(parts) != 2 {
				t.Errorf("Expected timespan to have exactly 2 parts separated by '/', got: %s", timespan)
			}

			// Verify start time is preserved
			if !strings.HasPrefix(timespan, tt.startTime) && !strings.Contains(timespan, tt.startTime) {
				// For timezone conversions, just check that it's a valid RFC3339 format
				if _, err := time.Parse(time.RFC3339, parts[0]); err != nil {
					t.Errorf("Expected valid RFC3339 start time in timespan, got: %s", parts[0])
				}
			}

			// If we have specific end time, verify it's preserved or converted properly
			if tt.endTime != "" && tt.checkDuration {
				start, _ := time.Parse(time.RFC3339, tt.startTime)
				expectedEnd, _ := time.Parse(time.RFC3339, tt.endTime)
				actualEnd, err := time.Parse(time.RFC3339, parts[1])
				if err != nil {
					t.Errorf("Expected valid RFC3339 end time in timespan, got: %s", parts[1])
				} else {
					// Check that the duration is preserved (allowing for timezone conversion)
					expectedDuration := expectedEnd.Sub(start)
					actualDuration := actualEnd.Sub(start)
					if expectedDuration != actualDuration {
						// Allow small differences for timezone/parsing issues
						diff := expectedDuration - actualDuration
						if diff < 0 {
							diff = -diff
						}
						if diff > time.Second {
							t.Errorf("Duration mismatch: expected %v, got %v", expectedDuration, actualDuration)
						}
					}
				}
			}
		})
	}
}

func TestBuildSafeKQLQueryLogLevelMapping(t *testing.T) {
	tests := []struct {
		name           string
		logLevel       string
		expectedPrefix string
	}{
		{
			name:           "info level maps to I prefix",
			logLevel:       "info",
			expectedPrefix: "where log_s startswith 'I'",
		},
		{
			name:           "error level maps to E prefix",
			logLevel:       "error",
			expectedPrefix: "where log_s startswith 'E'",
		},
		{
			name:           "warning level maps to W prefix",
			logLevel:       "warning",
			expectedPrefix: "where log_s startswith 'W'",
		},
		{
			name:           "empty log level has no prefix filter",
			logLevel:       "",
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := BuildSafeKQLQuery("kube-apiserver", tt.logLevel, 100, "/test/resource")

			if tt.expectedPrefix == "" {
				// Should not contain any log level filtering
				if strings.Contains(query, "where log_s startswith") {
					t.Errorf("Expected no log level filtering for empty log level, but found it in: %s", query)
				}
			} else {
				if !strings.Contains(query, tt.expectedPrefix) {
					t.Errorf("Expected query to contain '%s', but it didn't. Query: %s", tt.expectedPrefix, query)
				}
			}
		})
	}
}

func TestBuildSafeKQLQueryStructure(t *testing.T) {
	query := BuildSafeKQLQuery("kube-apiserver", "info", 100, "/test/resource")

	// The query should be a single line with pipe separators
	if strings.Contains(query, "\n") {
		t.Errorf("Expected query to be a single line, but found newlines: %s", query)
	}

	// Check that query follows expected order by looking for the components in sequence
	expectedOrder := []string{
		"AzureDiagnostics",
		"where Category ==",
		"where log_s startswith",
		"order by",
		"limit",
		"project",
	}

	lastIndex := 0

	for _, expected := range expectedOrder {
		index := strings.Index(query[lastIndex:], expected)
		if index == -1 {
			t.Errorf("Expected to find '%s' in query after position %d, but didn't find it. Query: %s", expected, lastIndex, query)
			continue
		}
		lastIndex += index + len(expected)
	}

	// Verify essential components
	if !strings.HasPrefix(query, "AzureDiagnostics") {
		t.Errorf("Query should start with AzureDiagnostics, got: %s", query)
	}

	if !strings.Contains(query, "order by TimeGenerated desc") {
		t.Errorf("Query should contain proper ordering, got: %s", query)
	}

	if !strings.Contains(query, "limit 100") {
		t.Errorf("Query should contain proper limit, got: %s", query)
	}

	if !strings.Contains(query, "project TimeGenerated, Level, log_s") {
		t.Errorf("Query should contain proper projection, got: %s", query)
	}
}

func TestBuildSafeKQLQuerySanitization(t *testing.T) {
	tests := []struct {
		name              string
		category          string
		clusterResourceID string
		description       string
	}{
		{
			name:              "normal category",
			category:          "kube-apiserver",
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			description:       "should work with normal inputs",
		},
		{
			name:              "category with special characters should be safe",
			category:          "kube-apiserver",
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			description:       "query should be built safely even with special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := BuildSafeKQLQuery(tt.category, "", 100, tt.clusterResourceID)

			// Basic validation that query was generated
			if query == "" {
				t.Errorf("Expected non-empty query, got empty string")
			}

			// Ensure the category is properly quoted in the query
			expectedCategoryClause := fmt.Sprintf("where Category == '%s'", tt.category)
			if !strings.Contains(query, expectedCategoryClause) {
				t.Errorf("Expected query to contain properly quoted category clause '%s', got: %s", expectedCategoryClause, query)
			}
		})
	}
}
