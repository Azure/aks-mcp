package config

import (
	"testing"

	"github.com/Azure/aks-mcp/internal/auth"
)

func TestDynamicOAuthRedirectPort(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		expectedURI string
	}{
		{
			name:        "default port 8000",
			port:        8000,
			expectedURI: "http://localhost:8000/oauth/callback",
		},
		{
			name:        "custom port 3000",
			port:        3000,
			expectedURI: "http://localhost:3000/oauth/callback",
		},
		{
			name:        "custom port 9090",
			port:        9090,
			expectedURI: "http://localhost:9090/oauth/callback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config with custom port
			cfg := NewConfig()
			cfg.Port = tt.port
			cfg.OAuthConfig.Enabled = true
			cfg.OAuthConfig.TenantID = "test-tenant"
			cfg.OAuthConfig.ClientID = "test-client"
			cfg.OAuthConfig.RequiredScopes = []string{auth.AzureADScope}

			// Simulate the oauth config parsing which sets dynamic redirect
			cfg.parseOAuthConfig("")

			// Verify the redirect URI uses the configured port
			if len(cfg.OAuthConfig.AllowedRedirects) != 1 {
				t.Fatalf("Expected 1 redirect URI, got %d", len(cfg.OAuthConfig.AllowedRedirects))
			}

			actualURI := cfg.OAuthConfig.AllowedRedirects[0]
			if actualURI != tt.expectedURI {
				t.Errorf("Expected redirect URI %s, got %s", tt.expectedURI, actualURI)
			}
		})
	}
}

func TestDynamicOAuthRedirectWithCustomRedirects(t *testing.T) {
	// Test that when custom redirects are provided via CLI, they are used instead
	cfg := NewConfig()
	cfg.Port = 9000
	cfg.OAuthConfig.Enabled = true
	cfg.OAuthConfig.TenantID = "test-tenant"
	cfg.OAuthConfig.ClientID = "test-client"
	cfg.OAuthConfig.RequiredScopes = []string{auth.AzureADScope}

	// Provide custom redirects via CLI simulation
	customRedirects := "http://example.com/callback,http://localhost:3000/custom"
	cfg.parseOAuthConfig(customRedirects)

	// Verify custom redirects are used, not the dynamic default
	expectedRedirects := []string{
		"http://example.com/callback",
		"http://localhost:3000/custom",
	}

	if len(cfg.OAuthConfig.AllowedRedirects) != 2 {
		t.Fatalf("Expected 2 redirect URIs, got %d", len(cfg.OAuthConfig.AllowedRedirects))
	}

	for i, expected := range expectedRedirects {
		if cfg.OAuthConfig.AllowedRedirects[i] != expected {
			t.Errorf("Expected redirect URI %s, got %s", expected, cfg.OAuthConfig.AllowedRedirects[i])
		}
	}
}
