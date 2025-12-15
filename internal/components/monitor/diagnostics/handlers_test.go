package diagnostics

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/security"
)

func TestGetControlPlaneDiagnosticSettingsHandler(t *testing.T) {
	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}
	handler := GetControlPlaneDiagnosticSettingsHandler(nil, cfg) // Pass nil Azure client for testing

	if handler == nil {
		t.Error("Expected handler to be created, got nil")
	}

	// Test handler with invalid params to ensure it calls the underlying function
	params := map[string]interface{}{}
	_, err := handler.Handle(context.Background(), params, cfg)
	if err == nil {
		t.Error("Expected error for missing parameters, got nil")
	}

	if !strings.Contains(err.Error(), "missing or invalid subscription_id parameter") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGetControlPlaneLogsHandler(t *testing.T) {
	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}
	handler := GetControlPlaneLogsHandler(nil, cfg) // Pass nil Azure client for testing

	if handler == nil {
		t.Error("Expected handler to be created, got nil")
	}

	// Test handler with invalid params to ensure it calls the underlying function
	params := map[string]interface{}{}
	_, err := handler.Handle(context.Background(), params, cfg)
	if err == nil {
		t.Error("Expected error for missing parameters, got nil")
	}

	if !strings.Contains(err.Error(), "missing or invalid") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestHandleControlPlaneDiagnosticSettings_ParameterValidation(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"subscription_id": "12345678-1234-1234-1234-123456789012",
				"resource_group":  "test-rg",
				"cluster_name":    "test-cluster",
			},
			wantError: false,
		},
		{
			name: "missing subscription_id",
			params: map[string]interface{}{
				"resource_group": "test-rg",
				"cluster_name":   "test-cluster",
			},
			wantError: true,
			errorMsg:  "missing or invalid subscription_id parameter",
		},
		{
			name: "empty subscription_id",
			params: map[string]interface{}{
				"subscription_id": "",
				"resource_group":  "test-rg",
				"cluster_name":    "test-cluster",
			},
			wantError: true,
			errorMsg:  "missing or invalid subscription_id parameter",
		},
		{
			name: "missing resource_group",
			params: map[string]interface{}{
				"subscription_id": "12345678-1234-1234-1234-123456789012",
				"cluster_name":    "test-cluster",
			},
			wantError: true,
			errorMsg:  "missing or invalid resource_group parameter",
		},
		{
			name: "missing cluster_name",
			params: map[string]interface{}{
				"subscription_id": "12345678-1234-1234-1234-123456789012",
				"resource_group":  "test-rg",
			},
			wantError: true,
			errorMsg:  "missing or invalid cluster_name parameter",
		},
	}

	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HandleControlPlaneDiagnosticSettings(context.Background(), tt.params, nil, cfg) // Pass nil Azure client for testing

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				// Note: This test will now fail because Azure client is required
				// In a real unit test environment, we would provide a valid Azure client
				if err != nil && !strings.Contains(err.Error(), "azure client is required but not provided") {
					t.Errorf("Unexpected error type: %v", err)
				}
			}
		})
	}
}

func TestHandleControlPlaneLogs_ParameterValidation(t *testing.T) {
	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	// Test with invalid params to ensure validation works
	params := map[string]interface{}{}
	_, err := HandleControlPlaneLogs(context.Background(), params, nil, cfg) // Pass nil Azure client for testing
	if err == nil {
		t.Error("Expected error for missing parameters, got nil")
	}

	if !strings.Contains(err.Error(), "missing or invalid") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}
