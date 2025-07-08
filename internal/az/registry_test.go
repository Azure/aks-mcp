package az

import (
	"testing"
)

func TestGetReadOnlyAzCommands_ContainsNodepoolCommands(t *testing.T) {
	commands := GetReadOnlyAzCommands()

	// Check that nodepool list command is included
	foundNodepoolList := false
	foundNodepoolShow := false

	for _, cmd := range commands {
		if cmd.Name == "az aks nodepool list" {
			foundNodepoolList = true
			if cmd.Description == "" {
				t.Error("Expected nodepool list command to have a description")
			}
			if cmd.ArgsExample == "" {
				t.Error("Expected nodepool list command to have an args example")
			}
		}
		if cmd.Name == "az aks nodepool show" {
			foundNodepoolShow = true
			if cmd.Description == "" {
				t.Error("Expected nodepool show command to have a description")
			}
			if cmd.ArgsExample == "" {
				t.Error("Expected nodepool show command to have an args example")
			}
		}
	}

	if !foundNodepoolList {
		t.Error("Expected to find 'az aks nodepool list' command in read-only commands")
	}

	if !foundNodepoolShow {
		t.Error("Expected to find 'az aks nodepool show' command in read-only commands")
	}
}

func TestRegisterAzCommand_NodepoolCommands(t *testing.T) {
	// Test that nodepool list command can be registered
	listCmd := AksCommand{
		Name:        "az aks nodepool list",
		Description: "List node pools in a managed Kubernetes cluster",
		ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup",
	}

	tool := RegisterAzCommand(listCmd)

	if tool.Name != "az_aks_nodepool_list" {
		t.Errorf("Expected tool name 'az_aks_nodepool_list', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}

	// Test that nodepool show command can be registered
	showCmd := AksCommand{
		Name:        "az aks nodepool show",
		Description: "Show the details for a node pool in the managed Kubernetes cluster",
		ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1",
	}

	tool2 := RegisterAzCommand(showCmd)

	if tool2.Name != "az_aks_nodepool_show" {
		t.Errorf("Expected tool name 'az_aks_nodepool_show', got '%s'", tool2.Name)
	}

	if tool2.Description == "" {
		t.Error("Expected tool description to be set")
	}
}
