package config

import (
	"testing"
)

func TestBasicOAuthConfig(t *testing.T) {
	// Test basic OAuth configuration parsing
	cfg := NewConfig()
	cfg.OAuthConfig.Enabled = true
	cfg.OAuthConfig.TenantID = "test-tenant"
	cfg.OAuthConfig.ClientID = "test-client"

	// Parse OAuth configuration
	cfg.parseOAuthConfig()

	// Verify basic configuration is preserved
	if !cfg.OAuthConfig.Enabled {
		t.Error("Expected OAuth to be enabled")
	}
	if cfg.OAuthConfig.TenantID != "test-tenant" {
		t.Errorf("Expected tenant ID 'test-tenant', got %s", cfg.OAuthConfig.TenantID)
	}
	if cfg.OAuthConfig.ClientID != "test-client" {
		t.Errorf("Expected client ID 'test-client', got %s", cfg.OAuthConfig.ClientID)
	}
}
