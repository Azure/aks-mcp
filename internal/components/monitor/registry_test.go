package monitor

import (
	"testing"
)

func TestGetReadOnlyMonitorCommands_ContainsMetricsCommands(t *testing.T) {
	commands := GetReadOnlyMonitorCommands()

	// Verify we have the expected number of commands
	expectedCommands := 3
	if len(commands) != expectedCommands {
		t.Errorf("Expected %d read-only commands, got %d", expectedCommands, len(commands))
	}

	// Check that specific metric commands are included
	foundMetricsList := false
	foundMetricsDefinitions := false
	foundMetricsNamespaces := false

	for _, cmd := range commands {
		switch cmd.Name {
		case "az monitor metrics list":
			foundMetricsList = true
			if cmd.Description == "" {
				t.Error("Expected metrics list command to have a description")
			}
			if cmd.ArgsExample == "" {
				t.Error("Expected metrics list command to have an args example")
			}
			if cmd.Category != "metrics" {
				t.Errorf("Expected metrics list command to have category 'metrics', got '%s'", cmd.Category)
			}
		case "az monitor metrics list-definitions":
			foundMetricsDefinitions = true
			if cmd.Description == "" {
				t.Error("Expected metrics list-definitions command to have a description")
			}
			if cmd.ArgsExample == "" {
				t.Error("Expected metrics list-definitions command to have an args example")
			}
			if cmd.Category != "metrics" {
				t.Errorf("Expected metrics list-definitions command to have category 'metrics', got '%s'", cmd.Category)
			}
		case "az monitor metrics list-namespaces":
			foundMetricsNamespaces = true
			if cmd.Description == "" {
				t.Error("Expected metrics list-namespaces command to have a description")
			}
			if cmd.ArgsExample == "" {
				t.Error("Expected metrics list-namespaces command to have an args example")
			}
			if cmd.Category != "metrics" {
				t.Errorf("Expected metrics list-namespaces command to have category 'metrics', got '%s'", cmd.Category)
			}
		}
	}

	if !foundMetricsList {
		t.Error("Expected to find 'az monitor metrics list' command in read-only commands")
	}

	if !foundMetricsDefinitions {
		t.Error("Expected to find 'az monitor metrics list-definitions' command in read-only commands")
	}

	if !foundMetricsNamespaces {
		t.Error("Expected to find 'az monitor metrics list-namespaces' command in read-only commands")
	}
}

func TestRegisterMonitorCommand_MetricsCommands(t *testing.T) {
	// Test that metrics list command can be registered
	listCmd := MonitorCommand{
		Name:        "az monitor metrics list",
		Description: "List the metric values for a resource",
		ArgsExample: "--resource /subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/virtualMachines/{vmName} --metric \"Percentage CPU\"",
		Category:    "metrics",
	}

	tool := RegisterMonitorCommand(listCmd)

	if tool.Name != "az_monitor_metrics_list" {
		t.Errorf("Expected tool name 'az_monitor_metrics_list', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}

	expectedDescription := "Run az monitor metrics list command: List the metric values for a resource.\nExample: `--resource /subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/virtualMachines/{vmName} --metric \"Percentage CPU\"`"
	if tool.Description != expectedDescription {
		t.Errorf("Expected tool description to contain example, got: %s", tool.Description)
	}

	// Test that metrics list-definitions command can be registered
	definitionsCmd := MonitorCommand{
		Name:        "az monitor metrics list-definitions",
		Description: "List the metric definitions for a resource",
		ArgsExample: "--resource /subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{clusterName}",
		Category:    "metrics",
	}

	tool2 := RegisterMonitorCommand(definitionsCmd)

	if tool2.Name != "az_monitor_metrics_list-definitions" {
		t.Errorf("Expected tool name 'az_monitor_metrics_list-definitions', got '%s'", tool2.Name)
	}

	if tool2.Description == "" {
		t.Error("Expected tool description to be set")
	}
}

func TestRegisterMonitorCommand_WithoutArgsExample(t *testing.T) {
	// Test command registration without args example
	cmd := MonitorCommand{
		Name:        "az monitor test",
		Description: "Test command",
		ArgsExample: "",
		Category:    "test",
	}

	tool := RegisterMonitorCommand(cmd)

	if tool.Name != "az_monitor_test" {
		t.Errorf("Expected tool name 'az_monitor_test', got '%s'", tool.Name)
	}

	expectedDescription := "Run az monitor test command: Test command."
	if tool.Description != expectedDescription {
		t.Errorf("Expected tool description '%s', got '%s'", expectedDescription, tool.Description)
	}
}

func TestGetReadWriteMonitorCommands_IsEmpty(t *testing.T) {
	commands := GetReadWriteMonitorCommands()

	if len(commands) != 0 {
		t.Errorf("Expected read-write commands to be empty, got %d commands", len(commands))
	}
}

func TestGetAdminMonitorCommands_IsEmpty(t *testing.T) {
	commands := GetAdminMonitorCommands()

	if len(commands) != 0 {
		t.Errorf("Expected admin commands to be empty, got %d commands", len(commands))
	}
}

func TestMonitorCommand_StructFields(t *testing.T) {
	cmd := MonitorCommand{
		Name:        "test name",
		Description: "test description",
		ArgsExample: "test args",
		Category:    "test category",
	}

	if cmd.Name != "test name" {
		t.Errorf("Expected Name to be 'test name', got '%s'", cmd.Name)
	}

	if cmd.Description != "test description" {
		t.Errorf("Expected Description to be 'test description', got '%s'", cmd.Description)
	}

	if cmd.ArgsExample != "test args" {
		t.Errorf("Expected ArgsExample to be 'test args', got '%s'", cmd.ArgsExample)
	}

	if cmd.Category != "test category" {
		t.Errorf("Expected Category to be 'test category', got '%s'", cmd.Category)
	}
}
