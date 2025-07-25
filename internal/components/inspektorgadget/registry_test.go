package inspektorgadget

import "testing"

func TestRegisterInspektorGadgetTool(t *testing.T) {
	tool := RegisterInspektorGadgetTool()
	if tool.Name != "inspektor_gadget" {
		t.Errorf("Expected tool name 'inspektor_gadget', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}

	_, ok := tool.InputSchema.Properties["action_params"].(map[string]any)
	if !ok {
		t.Error("action_params should be an object")
		return
	}
	_, ok = tool.InputSchema.Properties["filter_params"].(map[string]any)
	if !ok {
		t.Error("filter_params should be an object")
		return
	}
}

func TestRegisterInspektorGadgetDeployUndeployTool(t *testing.T) {
	tool := RegisterInspektorGadgetDeployUndeployTool()
	if tool.Name != "inspektor_gadget_deploy_undeploy" {
		t.Errorf("Expected tool name 'inspektor_gadget_deploy_undeploy', got '%s'", tool.Name)
	}
	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}
	if _, ok := tool.InputSchema.Properties["action"].(map[string]any); !ok {
		t.Error("action should be a string")
	}
}
