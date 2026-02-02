package monitor

import (
	"testing"
)

func TestRegisterAksMonitoring_Tool(t *testing.T) {
	// Test that the monitoring tool is registered correctly
	tool := RegisterAksMonitoring()

	if tool.Name != "aks_monitoring" {
		t.Errorf("Expected tool name 'aks_monitoring', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}
}

func TestGetSupportedMonitoringOperations_ContainsExpectedOps(t *testing.T) {
	// Test that supported operations include expected operations
	operations := GetSupportedMonitoringOperations()

	expectedOps := []string{
		"metrics", "resource_health", "app_insights", "diagnostics", "control_plane_logs",
	}

	for _, expectedOp := range expectedOps {
		found := false
		for _, op := range operations {
			if op == expectedOp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected operation '%s' not found in supported monitoring operations", expectedOp)
		}
	}
}

func TestValidateMonitoringOperation_ChecksValidOperations(t *testing.T) {
	// Test that validation works for supported operations
	validOps := []string{"metrics", "resource_health", "app_insights", "diagnostics", "control_plane_logs"}
	for _, op := range validOps {
		if !ValidateMonitoringOperation(op) {
			t.Errorf("Expected operation '%s' to be valid", op)
		}
	}

	// Test that validation fails for invalid operations
	invalidOps := []string{"invalid", "unknown", ""}
	for _, op := range invalidOps {
		if ValidateMonitoringOperation(op) {
			t.Errorf("Expected operation '%s' to be invalid", op)
		}
	}
}

func TestValidateMetricsQueryType_ChecksValidTypes(t *testing.T) {
	// Test that validation works for supported query types
	validTypes := []string{"list", "list-definitions", "list-namespaces"}
	for _, queryType := range validTypes {
		if !ValidateMetricsQueryType(queryType) {
			t.Errorf("Expected query type '%s' to be valid", queryType)
		}
	}

	// Test that validation fails for invalid query types
	invalidTypes := []string{"invalid", "unknown", ""}
	for _, queryType := range invalidTypes {
		if ValidateMetricsQueryType(queryType) {
			t.Errorf("Expected query type '%s' to be invalid", queryType)
		}
	}
}
