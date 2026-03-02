package inspektorgadget

import (
	"strings"
	"testing"
)

func TestRegisterInspektorGadgetTool(t *testing.T) {
	tool := RegisterInspektorGadgetTool()

	t.Run("name", func(t *testing.T) {
		if tool.Name != "inspektor_gadget_observability" {
			t.Errorf("expected tool name 'inspektor_gadget_observability', got %q", tool.Name)
		}
	})

	t.Run("description", func(t *testing.T) {
		if tool.Description == "" {
			t.Error("tool description should not be empty")
		}
		if !strings.Contains(tool.Description, "Real-time observability tool") {
			t.Error("description should contain 'Real-time observability tool'")
		}
	})

	t.Run("schema properties", func(t *testing.T) {
		expectedProperties := []string{
			"action",
			"gadget_name",
			"duration",
			"gadget_id",
			"chart_version",
			"confirm",
			"namespace",
			"pod",
			"container",
			"selector",
			"node",
		}

		for _, prop := range expectedProperties {
			t.Run(prop, func(t *testing.T) {
				if _, exists := tool.InputSchema.Properties[prop]; !exists {
					t.Errorf("expected property %q to exist in schema", prop)
				}
			})
		}
	})

	t.Run("action enum values", func(t *testing.T) {
		actionProp, exists := tool.InputSchema.Properties["action"]
		if !exists {
			t.Fatal("action property should exist")
		}

		actionMap, ok := actionProp.(map[string]interface{})
		if !ok {
			t.Fatal("action property should be a map")
		}

		enumValues, hasEnum := actionMap["enum"]
		if !hasEnum {
			t.Fatal("action should have enum values")
		}

		enumSlice, ok := enumValues.([]string)
		if !ok {
			t.Fatal("enum values should be a string slice")
		}

		expectedActions := getActions()
		for _, action := range expectedActions {
			found := false
			for _, e := range enumSlice {
				if e == action {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("action enum should contain %q", action)
			}
		}
	})

	t.Run("gadget_name enum values", func(t *testing.T) {
		gadgetNameProp, exists := tool.InputSchema.Properties["gadget_name"]
		if !exists {
			t.Fatal("gadget_name property should exist")
		}

		gadgetNameMap, ok := gadgetNameProp.(map[string]interface{})
		if !ok {
			t.Fatal("gadget_name property should be a map")
		}

		enumValues, hasEnum := gadgetNameMap["enum"]
		if !hasEnum {
			t.Fatal("gadget_name should have enum values")
		}

		enumSlice, ok := enumValues.([]string)
		if !ok {
			t.Fatal("enum values should be a string slice")
		}
		if len(enumSlice) == 0 {
			t.Error("gadget_name enum should have values")
		}
	})

	t.Run("gadget specific params registered", func(t *testing.T) {
		gadgetParams := getGadgetParams()
		for paramKey := range gadgetParams {
			if _, exists := tool.InputSchema.Properties[paramKey]; !exists {
				t.Errorf("gadget param %q should be registered in schema", paramKey)
			}
		}
	})
}
