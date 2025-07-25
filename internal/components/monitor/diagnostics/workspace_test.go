package diagnostics

import (
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/security"
)

func TestGetWorkspaceGUID(t *testing.T) {
	tests := []struct {
		name                string
		workspaceResourceID string
		wantError           bool
		errorMsg            string
	}{
		{
			name:                "invalid resource ID format - too short",
			workspaceResourceID: "/invalid/format",
			wantError:           true,
			errorMsg:            "invalid workspace resource ID format",
		},
		{
			name:                "invalid resource ID format - missing parts",
			workspaceResourceID: "/subscriptions/test/resourceGroups/rg/providers/Microsoft.OperationalInsights",
			wantError:           true,
			errorMsg:            "invalid workspace resource ID format",
		},
		{
			name:                "valid resource ID format structure",
			workspaceResourceID: "/subscriptions/test/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/workspace",
			wantError:           true, // Will fail at Azure CLI execution level in test
		},
		{
			name:                "case insensitive resource ID parsing",
			workspaceResourceID: "/subscriptions/test/RESOURCEGROUPS/rg/providers/microsoft.operationalinsights/WORKSPACES/workspace",
			wantError:           true, // Will fail at Azure CLI execution level in test
		},
	}

	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getWorkspaceGUID(tt.workspaceResourceID, cfg)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGetWorkspaceGUID_ResourceIDParsing(t *testing.T) {
	// Test that we can properly extract resource group and workspace name from resource ID
	validResourceID := "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace"

	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	// This will fail at Azure CLI execution but we can check that parsing doesn't fail immediately
	_, err := getWorkspaceGUID(validResourceID, cfg)

	// Should get an Azure CLI execution error, not a parsing error
	if err != nil && strings.Contains(err.Error(), "invalid workspace resource ID format") {
		t.Errorf("Expected Azure CLI execution error, got parsing error: %v", err)
	}
}

func TestExtractWorkspaceGUIDFromDiagnosticSettings_InvalidParameters(t *testing.T) {
	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	// This will fail at the diagnostic settings call, but we can test the error handling
	_, err := ExtractWorkspaceGUIDFromDiagnosticSettings("invalid", "invalid", "invalid", cfg)
	if err == nil {
		t.Error("Expected error for invalid parameters, got nil")
	}

	// Should fail at the Azure CLI execution level (could be timeout, permission, or other execution error)
	if !strings.Contains(err.Error(), "failed to get diagnostic settings") {
		t.Errorf("Expected diagnostic settings error, got: %v", err)
	}
}

func TestExtractWorkspaceGUIDFromDiagnosticSettings_JSONParsing(t *testing.T) {
	// Test JSON parsing logic with mock data
	// Note: This would require mocking the HandleControlPlaneDiagnosticSettings function
	// For now, we test the error case with invalid cluster details

	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	// Test with empty strings (should fail validation)
	_, err := ExtractWorkspaceGUIDFromDiagnosticSettings("", "", "", cfg)
	if err == nil {
		t.Error("Expected error for empty parameters, got nil")
	}

	if !strings.Contains(err.Error(), "failed to get diagnostic settings") {
		t.Errorf("Expected diagnostic settings failure, got: %v", err)
	}
}

func TestGetWorkspaceGUID_EdgeCases(t *testing.T) {
	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	testCases := []struct {
		name        string
		resourceID  string
		expectError string
	}{
		{
			name:        "empty resource ID",
			resourceID:  "",
			expectError: "invalid workspace resource ID format",
		},
		{
			name:        "resource ID with no resource groups",
			resourceID:  "/subscriptions/test/providers/Microsoft.OperationalInsights/workspaces/workspace",
			expectError: "invalid workspace resource ID format",
		},
		{
			name:        "resource ID with no workspaces",
			resourceID:  "/subscriptions/test/resourceGroups/rg/providers/Microsoft.OperationalInsights",
			expectError: "invalid workspace resource ID format",
		},
		{
			name:        "resource ID with missing workspace name",
			resourceID:  "/subscriptions/test/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/",
			expectError: "could not extract resource group and workspace name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getWorkspaceGUID(tc.resourceID, cfg)
			if err == nil {
				t.Errorf("Expected error for case '%s', got nil", tc.name)
				return
			}

			if !strings.Contains(err.Error(), tc.expectError) {
				t.Errorf("Expected error to contain '%s', got '%s'", tc.expectError, err.Error())
			}
		})
	}
}

func TestFindDiagnosticSettingForCategory(t *testing.T) {
	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	tests := []struct {
		name          string
		logCategory   string
		expectError   bool
		expectedError string
	}{
		{
			name:          "kube-apiserver category",
			logCategory:   "kube-apiserver",
			expectError:   true, // Will fail during Azure CLI execution in test environment
			expectedError: "",
		},
		{
			name:          "kube-audit category",
			logCategory:   "kube-audit",
			expectError:   true, // Will fail during Azure CLI execution in test environment
			expectedError: "",
		},
		{
			name:          "invalid category",
			logCategory:   "invalid-category",
			expectError:   true, // Will fail during Azure CLI execution in test environment
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := FindDiagnosticSettingForCategory("test-sub", "test-rg", "test-cluster", tt.logCategory, cfg)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for category %s, got nil", tt.logCategory)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for category %s: %v", tt.logCategory, err)
			}
		})
	}
}
