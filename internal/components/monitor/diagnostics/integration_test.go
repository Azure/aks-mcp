package diagnostics

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/security"
)

// Integration tests for the monitor package interface to diagnostics

func TestDiagnosticSettings_Integration(t *testing.T) {
	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	// Test with invalid params to ensure delegation works
	params := map[string]interface{}{}
	_, err := HandleControlPlaneDiagnosticSettings(context.Background(), params, nil, cfg) // Pass nil Azure client for testing
	if err == nil {
		t.Error("Expected error for missing parameters, got nil")
	}

	if !strings.Contains(err.Error(), "missing or invalid subscription_id parameter") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestControlPlaneLogs_Integration(t *testing.T) {
	cfg := &config.ConfigData{
		SecurityConfig: &security.SecurityConfig{
			AccessLevel: "readonly",
		},
	}

	// Test with invalid params to ensure delegation works
	params := map[string]interface{}{}
	_, err := HandleControlPlaneLogs(context.Background(), params, nil, cfg) // Pass nil Azure client for testing
	if err == nil {
		t.Error("Expected error for missing parameters, got nil")
	}

	if !strings.Contains(err.Error(), "missing or invalid") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestDiagnosticSettingsHandler_Integration(t *testing.T) {
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

func TestControlPlaneLogsHandler_Integration(t *testing.T) {
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
