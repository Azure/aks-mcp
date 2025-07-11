package fleet

import (
	"testing"
)

func TestGetReadOnlyFleetCommands_ContainsBasicCommands(t *testing.T) {
	commands := GetReadOnlyFleetCommands()

	// Check that basic fleet commands are included
	foundFleetList := false
	foundFleetShow := false

	for _, cmd := range commands {
		if cmd.Name == "az fleet list" {
			foundFleetList = true
			if cmd.Description == "" {
				t.Error("Expected fleet list command to have a description")
			}
			if cmd.ArgsExample == "" {
				t.Error("Expected fleet list command to have an args example")
			}
		}
		if cmd.Name == "az fleet show" {
			foundFleetShow = true
			if cmd.Description == "" {
				t.Error("Expected fleet show command to have a description")
			}
			if cmd.ArgsExample == "" {
				t.Error("Expected fleet show command to have an args example")
			}
		}
	}

	if !foundFleetList {
		t.Error("Expected to find 'az fleet list' command in read-only commands")
	}

	if !foundFleetShow {
		t.Error("Expected to find 'az fleet show' command in read-only commands")
	}
}

func TestRegisterFleetCommand_BasicCommands(t *testing.T) {
	// Test that fleet list command can be registered
	listCmd := FleetCommand{
		Name:        "az fleet list",
		Description: "List all fleets",
		ArgsExample: "--resource-group myResourceGroup",
	}

	tool := RegisterFleetCommand(listCmd)

	if tool.Name != "az_fleet_list" {
		t.Errorf("Expected tool name 'az_fleet_list', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}

	// Test that fleet member create command can be registered
	createCmd := FleetCommand{
		Name:        "az fleet member create",
		Description: "Add a member to a fleet",
		ArgsExample: "--name myMember --fleet-name myFleet --resource-group myResourceGroup",
	}

	tool2 := RegisterFleetCommand(createCmd)

	if tool2.Name != "az_fleet_member_create" {
		t.Errorf("Expected tool name 'az_fleet_member_create', got '%s'", tool2.Name)
	}

	if tool2.Description == "" {
		t.Error("Expected tool description to be set")
	}
}

func TestGetReadWriteFleetCommands_ContainsManagementCommands(t *testing.T) {
	commands := GetReadWriteFleetCommands()

	// Check that management commands are included
	foundFleetCreate := false
	foundMemberCreate := false

	for _, cmd := range commands {
		if cmd.Name == "az fleet create" {
			foundFleetCreate = true
		}
		if cmd.Name == "az fleet member create" {
			foundMemberCreate = true
		}
	}

	if !foundFleetCreate {
		t.Error("Expected to find 'az fleet create' command in read-write commands")
	}

	if !foundMemberCreate {
		t.Error("Expected to find 'az fleet member create' command in read-write commands")
	}
}
