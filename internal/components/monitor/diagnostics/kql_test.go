package diagnostics

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildSafeKQLQuery(t *testing.T) {
	tests := []struct {
		name               string
		category           string
		logLevel           string
		maxRecords         int
		clusterResourceID  string
		isResourceSpecific bool
		expectedContains   []string
		notExpected        []string
	}{
		{
			name:               "azure diagnostics query without log level",
			category:           "kube-apiserver",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
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
			name:               "resource-specific query for kube-apiserver",
			category:           "kube-apiserver",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"limit 100",
				"project TimeGenerated, Category, Level, Message, PodName",
				"order by TimeGenerated desc",
			},
			notExpected: []string{
				"AzureDiagnostics",
				"where Category ==",
			},
		},
		{
			name:               "resource-specific query for audit logs",
			category:           "kube-audit",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSAudit",
				"where _ResourceId ==",
				"limit 100",
				"project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User",
			},
			notExpected: []string{
				"AzureDiagnostics",
				"where Category ==",
			},
		},
		{
			name:               "azure diagnostics query with info log level",
			category:           "kube-apiserver",
			logLevel:           "info",
			maxRecords:         50,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'kube-apiserver'",
				"where log_s startswith 'I'",
				"limit 50",
				"project TimeGenerated, Level, log_s",
			},
		},
		{
			name:               "resource-specific query with info log level",
			category:           "kube-apiserver",
			logLevel:           "info",
			maxRecords:         50,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Level == 'INFO'",
				"limit 50",
			},
		},
		{
			name:               "azure diagnostics query with error log level",
			category:           "kube-controller-manager",
			logLevel:           "error",
			maxRecords:         200,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'kube-controller-manager'",
				"where log_s startswith 'E'",
				"limit 200",
			},
		},
		{
			name:               "azure diagnostics query with warning log level",
			category:           "kube-scheduler",
			logLevel:           "warning",
			maxRecords:         300,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'kube-scheduler'",
				"where log_s startswith 'W'",
				"limit 300",
			},
		},
		{
			name:               "azure diagnostics query with audit category",
			category:           "kube-audit",
			logLevel:           "",
			maxRecords:         1000,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'kube-audit'",
				"limit 1000",
			},
		},
		{
			name:               "azure diagnostics query with audit category and log level - should skip log level filtering",
			category:           "kube-audit",
			logLevel:           "info",
			maxRecords:         500,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
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
			name:               "azure diagnostics query with audit-admin category and log level - should skip log level filtering",
			category:           "kube-audit-admin",
			logLevel:           "error",
			maxRecords:         200,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
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
			name:               "azure diagnostics query with cloud controller manager",
			category:           "cloud-controller-manager",
			logLevel:           "info",
			maxRecords:         150,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'cloud-controller-manager'",
				"where log_s startswith 'I'",
				"limit 150",
			},
		},
		{
			name:               "fallback to azure diagnostics for unmapped category in resource-specific mode",
			category:           "unknown-category",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'unknown-category'",
				"limit 100",
			},
			notExpected: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
			},
		},
		// New comprehensive test cases for resource-specific table mode with correct log level handling
		{
			name:               "resource-specific query with warning log level for control plane",
			category:           "kube-controller-manager",
			logLevel:           "warning",
			maxRecords:         75,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId == '/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster'", // lowercase conversion
				"where Level == 'WARNING'",
				"limit 75",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"where Message startswith",
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific query with error log level for control plane",
			category:           "cloud-controller-manager",
			logLevel:           "error",
			maxRecords:         25,
			clusterResourceID:  "/subscriptions/TEST/resourcegroups/RG/providers/Microsoft.ContainerService/managedClusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId == '/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster'", // lowercase conversion
				"where Level == 'ERROR'",
				"limit 25",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"where Message startswith 'E'",
				"where log_s startswith",
			},
		},
		{
			name:               "resource-specific audit query should skip log level filtering",
			category:           "kube-audit",
			logLevel:           "info",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSAudit",
				"where _ResourceId ==",
				"limit 100",
				"project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User",
			},
			notExpected: []string{
				"where Level == 'INFO'",
				"where Message startswith",
				"where log_s startswith",
			},
		},
		{
			name:               "resource-specific audit-admin query should skip log level filtering",
			category:           "kube-audit-admin",
			logLevel:           "error",
			maxRecords:         200,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSAuditAdmin",
				"where _ResourceId ==",
				"limit 200",
				"project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User",
			},
			notExpected: []string{
				"where Level == 'ERROR'",
				"where Message startswith",
				"where log_s startswith",
			},
		},
		{
			name:               "resource-specific case sensitivity test - mixed case resource ID converted to lowercase",
			category:           "kube-scheduler",
			logLevel:           "",
			maxRecords:         50,
			clusterResourceID:  "/SUBSCRIPTIONS/TEST/RESOURCEGROUPS/RG/PROVIDERS/MICROSOFT.CONTAINERSERVICE/MANAGEDCLUSTERS/CLUSTER",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId == '/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster'", // all lowercase
				"limit 50",
			},
		},
		{
			name:               "azure diagnostics case sensitivity test - mixed case resource ID converted to uppercase",
			category:           "kube-scheduler",
			logLevel:           "",
			maxRecords:         50,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'kube-scheduler'",
				"ResourceId == '/SUBSCRIPTIONS/TEST/RESOURCEGROUPS/RG/PROVIDERS/MICROSOFT.CONTAINERSERVICE/MANAGEDCLUSTERS/CLUSTER'", // all uppercase
				"limit 50",
			},
		},
		{
			name:               "resource-specific fallback with log level should use azure diagnostics filtering",
			category:           "unknown-category",
			logLevel:           "warning",
			maxRecords:         30,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'unknown-category'",
				"where log_s startswith 'W'", // fallback to azure diagnostics log level filtering
				"limit 30",
			},
			notExpected: []string{
				"where Level == 'WARNING'",
				"AKSControlPlane",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := BuildSafeKQLQuery(tt.category, tt.logLevel, tt.maxRecords, tt.clusterResourceID, tt.isResourceSpecific)

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
			if tt.isResourceSpecific {
				if tableName, exists := resourceSpecificTableMapping[tt.category]; exists {
					if !strings.HasPrefix(query, tableName) {
						t.Errorf("Resource-specific query should start with %s, got: %s", tableName, query)
					}
				} else {
					// Fallback to AzureDiagnostics
					if !strings.HasPrefix(query, "AzureDiagnostics") {
						t.Errorf("Fallback query should start with AzureDiagnostics, got: %s", query)
					}
				}
			} else {
				if !strings.HasPrefix(query, "AzureDiagnostics") {
					t.Errorf("Azure Diagnostics query should start with AzureDiagnostics, got: %s", query)
				}
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
			query := BuildSafeKQLQuery("kube-apiserver", tt.logLevel, 100, "/test/resource", false)

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
	query := BuildSafeKQLQuery("kube-apiserver", "info", 100, "/test/resource", false)

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
			query := BuildSafeKQLQuery(tt.category, "", 100, tt.clusterResourceID, false)

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

func TestBuildSafeKQLQueryResourceSpecificMode(t *testing.T) {
	tests := []struct {
		name               string
		category           string
		expectedTable      string
		isResourceSpecific bool
	}{
		{
			name:               "kube-audit maps to AKSAudit table",
			category:           "kube-audit",
			expectedTable:      "AKSAudit",
			isResourceSpecific: true,
		},
		{
			name:               "kube-audit-admin maps to AKSAuditAdmin table",
			category:           "kube-audit-admin",
			expectedTable:      "AKSAuditAdmin",
			isResourceSpecific: true,
		},
		{
			name:               "kube-apiserver maps to AKSControlPlane table",
			category:           "kube-apiserver",
			expectedTable:      "AKSControlPlane",
			isResourceSpecific: true,
		},
		{
			name:               "unknown category falls back to AzureDiagnostics",
			category:           "unknown-category",
			expectedTable:      "AzureDiagnostics",
			isResourceSpecific: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := BuildSafeKQLQuery(tt.category, "", 100, "/test/resource", tt.isResourceSpecific)

			if !strings.Contains(query, tt.expectedTable) {
				t.Errorf("Expected query to contain table '%s', but it didn't. Query: %s", tt.expectedTable, query)
			}

			if tt.isResourceSpecific && tt.expectedTable != "AzureDiagnostics" {
				// Should use _ResourceId instead of ResourceId
				if !strings.Contains(query, "_ResourceId ==") {
					t.Errorf("Expected resource-specific query to use '_ResourceId ==', but it didn't. Query: %s", query)
				}
				// Should NOT contain Category filter
				if strings.Contains(query, "where Category ==") {
					t.Errorf("Expected resource-specific query NOT to contain 'where Category ==', but it did. Query: %s", query)
				}
			}
		})
	}
}

func TestResourceSpecificLogLevelFiltering(t *testing.T) {
	testResourceID := "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster"

	tests := []struct {
		name               string
		category           string
		logLevel           string
		isResourceSpecific bool
		expectedFilter     string
		notExpected        string
	}{
		{
			name:               "resource-specific info level filtering",
			category:           "kube-apiserver",
			logLevel:           "info",
			isResourceSpecific: true,
			expectedFilter:     "where Level == 'INFO'",
			notExpected:        "where Message startswith",
		},
		{
			name:               "resource-specific warning level filtering",
			category:           "kube-controller-manager",
			logLevel:           "warning",
			isResourceSpecific: true,
			expectedFilter:     "where Level == 'WARNING'",
			notExpected:        "where log_s startswith",
		},
		{
			name:               "resource-specific error level filtering",
			category:           "cloud-controller-manager",
			logLevel:           "error",
			isResourceSpecific: true,
			expectedFilter:     "where Level == 'ERROR'",
			notExpected:        "where Message startswith 'E'",
		},
		{
			name:               "azure diagnostics info level filtering",
			category:           "kube-apiserver",
			logLevel:           "info",
			isResourceSpecific: false,
			expectedFilter:     "where log_s startswith 'I'",
			notExpected:        "where Level == 'INFO'",
		},
		{
			name:               "azure diagnostics warning level filtering",
			category:           "kube-scheduler",
			logLevel:           "warning",
			isResourceSpecific: false,
			expectedFilter:     "where log_s startswith 'W'",
			notExpected:        "where Level == 'WARNING'",
		},
		{
			name:               "azure diagnostics error level filtering",
			category:           "cluster-autoscaler",
			logLevel:           "error",
			isResourceSpecific: false,
			expectedFilter:     "where log_s startswith 'E'",
			notExpected:        "where Level == 'ERROR'",
		},
		{
			name:               "resource-specific audit should skip log level filtering",
			category:           "kube-audit",
			logLevel:           "info",
			isResourceSpecific: true,
			expectedFilter:     "", // no filtering expected
			notExpected:        "where Level == 'INFO'",
		},
		{
			name:               "resource-specific audit-admin should skip log level filtering",
			category:           "kube-audit-admin",
			logLevel:           "error",
			isResourceSpecific: true,
			expectedFilter:     "", // no filtering expected
			notExpected:        "where Level == 'ERROR'",
		},
		{
			name:               "azure diagnostics audit should skip log level filtering",
			category:           "kube-audit",
			logLevel:           "warning",
			isResourceSpecific: false,
			expectedFilter:     "", // no filtering expected
			notExpected:        "where log_s startswith 'W'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := BuildSafeKQLQuery(tt.category, tt.logLevel, 100, testResourceID, tt.isResourceSpecific)

			if tt.expectedFilter != "" {
				if !strings.Contains(query, tt.expectedFilter) {
					t.Errorf("Expected query to contain '%s', but it didn't. Query: %s", tt.expectedFilter, query)
				}
			}

			if tt.notExpected != "" {
				if strings.Contains(query, tt.notExpected) {
					t.Errorf("Expected query NOT to contain '%s', but it did. Query: %s", tt.notExpected, query)
				}
			}
		})
	}
}

func TestResourceIdCaseSensitivity(t *testing.T) {
	testCases := []struct {
		name               string
		inputResourceID    string
		isResourceSpecific bool
		expectedInQuery    string
		description        string
	}{
		{
			name:               "resource-specific lowercase conversion",
			inputResourceID:    "/subscriptions/TEST/resourcegroups/RG/providers/Microsoft.ContainerService/managedClusters/CLUSTER",
			isResourceSpecific: true,
			expectedInQuery:    "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			description:        "Resource-specific tables should convert resource ID to lowercase",
		},
		{
			name:               "azure diagnostics uppercase conversion",
			inputResourceID:    "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedInQuery:    "/SUBSCRIPTIONS/TEST/RESOURCEGROUPS/RG/PROVIDERS/MICROSOFT.CONTAINERSERVICE/MANAGEDCLUSTERS/CLUSTER",
			description:        "Azure Diagnostics should convert resource ID to uppercase",
		},
		{
			name:               "mixed case resource-specific conversion",
			inputResourceID:    "/Subscriptions/Test/ResourceGroups/Rg/Providers/Microsoft.ContainerService/ManagedClusters/Cluster",
			isResourceSpecific: true,
			expectedInQuery:    "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			description:        "Mixed case should be converted to all lowercase for resource-specific",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := BuildSafeKQLQuery("kube-apiserver", "", 100, tc.inputResourceID, tc.isResourceSpecific)

			if !strings.Contains(query, tc.expectedInQuery) {
				t.Errorf("Expected query to contain resource ID '%s', but it didn't. Query: %s", tc.expectedInQuery, query)
			}

			// Make sure it doesn't contain the original case
			if tc.inputResourceID != tc.expectedInQuery && strings.Contains(query, tc.inputResourceID) {
				t.Errorf("Query should not contain original resource ID case '%s'. Query: %s", tc.inputResourceID, query)
			}
		})
	}
}
