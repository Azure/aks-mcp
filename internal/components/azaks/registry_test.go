package azaks

import (
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

func TestRegisterAzAksOperations_Tool(t *testing.T) {
	// Test that the tool is registered correctly
	cfg := &config.ConfigData{AccessLevel: "readonly"}
	tool := RegisterAzAksOperations(cfg)

	if tool.Name != "az_aks_operations" {
		t.Errorf("Expected tool name 'az_aks_operations', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}
}

func TestGetSupportedOperations_ContainsExpectedOps(t *testing.T) {
	// Test that supported operations include expected operations
	operations := GetSupportedOperations()

	expectedOps := []string{
		"show", "list", "create", "delete", "scale", "update", "upgrade",
		"nodepool-list", "nodepool-show", "nodepool-add", "nodepool-delete",
		"account-list", "account-set", "login", "get-credentials",
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
			t.Errorf("Expected operation '%s' not found in supported operations", expectedOp)
		}
	}
}

func TestValidateOperationAccess_ChecksAccessLevels(t *testing.T) {
	// Test access level validation for different operations
	testCases := []struct {
		operation   string
		accessLevel string
		shouldPass  bool
	}{
		{"show", "readonly", true},
		{"list", "readonly", true},
		{"create", "readonly", false},
		{"create", "readwrite", true},
		{"create", "admin", true},
		{"get-credentials", "readonly", false},
		{"get-credentials", "readwrite", false},
		{"get-credentials", "admin", true},
	}

	for _, tc := range testCases {
		cfg := &config.ConfigData{AccessLevel: tc.accessLevel}
		err := ValidateOperationAccess(tc.operation, cfg)
		if tc.shouldPass && err != nil {
			t.Errorf("Expected operation '%s' with access level '%s' to pass, but got error: %v", tc.operation, tc.accessLevel, err)
		}
		if !tc.shouldPass && err == nil {
			t.Errorf("Expected operation '%s' with access level '%s' to fail, but no error returned", tc.operation, tc.accessLevel)
		}
	}
}
