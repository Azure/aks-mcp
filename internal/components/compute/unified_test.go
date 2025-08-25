package compute

import (
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

func TestRegisterAzComputeOperations(t *testing.T) {
	// Test tool registration for different access levels
	accessLevels := []string{"readonly", "readwrite", "admin"}

	for _, level := range accessLevels {
		t.Run("AccessLevel_"+level, func(t *testing.T) {
			cfg := &config.ConfigData{AccessLevel: level}
			tool := RegisterAzComputeOperations(cfg)

			if tool.Name != "az_compute_operations" {
				t.Errorf("Expected tool name 'az_compute_operations', got '%s'", tool.Name)
			}

			if tool.Description == "" {
				t.Error("Expected tool description to be set")
			}

			// Check that description varies by access level and contains safety warning
			if !strings.Contains(tool.Description, "IMPORTANT: VM/VMSS resources are managed by AKS") {
				t.Error("All access levels should include AKS safety warning")
			}
		})
	}
}

func TestValidateOperationAccess(t *testing.T) {
	testCases := []struct {
		operation   string
		accessLevel string
		shouldPass  bool
	}{
		// Read-only operations
		{"show", "readonly", true},
		{"list", "readonly", true},
		{"get-instance-view", "readonly", true},

		// Read-write operations
		{"start", "readonly", false},
		{"start", "readwrite", true},
		{"start", "admin", true},
		{"stop", "readonly", false},
		{"stop", "readwrite", true},
		{"stop", "admin", true},
		{"restart", "readonly", false},
		{"restart", "readwrite", true},
		{"restart", "admin", true},
		{"reimage", "readonly", false},
		{"reimage", "readwrite", true},
		{"reimage", "admin", true},

		// Unknown operations
		{"invalid-op", "admin", false},
	}

	for _, tc := range testCases {
		t.Run(tc.operation+"_"+tc.accessLevel, func(t *testing.T) {
			cfg := &config.ConfigData{AccessLevel: tc.accessLevel}
			err := ValidateOperationAccess(tc.operation, cfg)

			if tc.shouldPass && err != nil {
				t.Errorf("Expected operation '%s' to pass for access level '%s', but got error: %v", tc.operation, tc.accessLevel, err)
			}
			if !tc.shouldPass && err == nil {
				t.Errorf("Expected operation '%s' to fail for access level '%s', but it passed", tc.operation, tc.accessLevel)
			}
		})
	}
}

func TestMapOperationToCommand(t *testing.T) {
	testCases := []struct {
		operation    string
		resourceType string
		expected     string
		shouldPass   bool
	}{
		// VM operations
		{"show", "vm", "az vm show", true},
		{"start", "vm", "az vm start", true},

		// VMSS operations
		{"show", "vmss", "az vmss show", true},
		{"restart", "vmss", "az vmss restart", true},
		{"reimage", "vmss", "az vmss reimage", true},
		// Scale operation removed - not safe for AKS-managed VMSS

		// Invalid resource types
		{"show", "invalid", "", false},

		// Invalid operations
		{"invalid-op", "vm", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.operation+"_"+tc.resourceType, func(t *testing.T) {
			result, err := MapOperationToCommand(tc.operation, tc.resourceType)

			if tc.shouldPass {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if result != tc.expected {
					t.Errorf("Expected command '%s', got '%s'", tc.expected, result)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error but got success with result: %s", result)
				}
			}
		})
	}
}

func TestGetOperationAccessLevel(t *testing.T) {
	testCases := []struct {
		operation string
		expected  string
	}{
		// Read-only operations
		{"show", "readonly"},
		{"list", "readonly"},
		{"get-instance-view", "readonly"},

		// Read-write operations
		{"start", "readwrite"},
		{"stop", "readwrite"},
		{"restart", "readwrite"},
		{"reimage", "readwrite"},

		// Unknown operations
		{"invalid-op", "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.operation, func(t *testing.T) {
			result := GetOperationAccessLevel(tc.operation)
			if result != tc.expected {
				t.Errorf("Expected access level '%s' for operation '%s', got '%s'", tc.expected, tc.operation, result)
			}
		})
	}
}
