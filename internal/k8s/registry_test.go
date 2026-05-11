package k8s

import (
	"encoding/json"
	"testing"
)

func TestCreateCallKubectlTool_ResourceIDRequired_WhenNoDefault(t *testing.T) {
	tool := createCallKubectlTool("readonly", "")

	schemaBytes, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("failed to marshal input schema: %v", err)
	}

	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	found := false
	for _, r := range schema.Required {
		if r == "aks_resource_id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected aks_resource_id to be required when no default provided, required fields: %v", schema.Required)
	}
}

func TestCreateCallKubectlTool_ResourceIDOptional_WhenDefaultSet(t *testing.T) {
	tool := createCallKubectlTool("readonly", "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster")

	schemaBytes, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("failed to marshal input schema: %v", err)
	}

	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	for _, r := range schema.Required {
		if r == "aks_resource_id" {
			t.Errorf("aks_resource_id should be optional when default is configured, but found in required: %v", schema.Required)
		}
	}
}

func TestCreateCallKubectlTool_CommandAlwaysRequired(t *testing.T) {
	tests := []struct {
		name              string
		defaultResourceID string
	}{
		{"no default", ""},
		{"with default", "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := createCallKubectlTool("readonly", tt.defaultResourceID)

			schemaBytes, _ := json.Marshal(tool.InputSchema)
			var schema struct {
				Required []string `json:"required"`
			}
			_ = json.Unmarshal(schemaBytes, &schema)

			found := false
			for _, r := range schema.Required {
				if r == "command" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("command should always be required, required fields: %v", schema.Required)
			}
		})
	}
}

func TestCreateCallKubectlTool_DescriptionContainsDefault(t *testing.T) {
	defaultID := "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/mycluster"
	tool := createCallKubectlTool("readonly", defaultID)

	schemaBytes, _ := json.Marshal(tool.InputSchema)
	var schema struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	_ = json.Unmarshal(schemaBytes, &schema)

	prop, ok := schema.Properties["aks_resource_id"]
	if !ok {
		t.Fatal("aks_resource_id property not found in schema")
	}
	if prop.Description == "" {
		t.Error("expected non-empty description for aks_resource_id")
	}
}

func TestCreateCallKubectlTool_Name(t *testing.T) {
	tool := createCallKubectlTool("readonly", "")
	if tool.Name != "call_kubectl" {
		t.Errorf("expected tool name call_kubectl, got %q", tool.Name)
	}
}

func TestRegisterKubectlTools_TokenAuthOnly(t *testing.T) {
	tools := RegisterKubectlTools("readonly", true, true, "")
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool in tokenAuthOnly mode, got %d", len(tools))
	}
	if tools[0].Name != "call_kubectl" {
		t.Errorf("expected call_kubectl, got %q", tools[0].Name)
	}
}

func TestRegisterKubectlTools_TokenAuthOnly_WithDefault(t *testing.T) {
	defaultID := "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster"
	tools := RegisterKubectlTools("readonly", true, true, defaultID)

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	schemaBytes, _ := json.Marshal(tools[0].InputSchema)
	var schema struct {
		Required []string `json:"required"`
	}
	_ = json.Unmarshal(schemaBytes, &schema)

	for _, r := range schema.Required {
		if r == "aks_resource_id" {
			t.Error("aks_resource_id should be optional when default resource ID is provided")
		}
	}
}
