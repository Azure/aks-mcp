package config

import (
	"testing"
)

func TestBasicOAuthConfig(t *testing.T) {
	// Test basic OAuth configuration parsing with valid GUIDs
	cfg := NewConfig()
	cfg.OAuthConfig.Enabled = true
	cfg.OAuthConfig.TenantID = "12345678-1234-1234-1234-123456789abc"
	cfg.OAuthConfig.ClientID = "87654321-4321-4321-4321-cba987654321"

	// Parse OAuth configuration
	if err := cfg.parseOAuthConfig("", ""); err != nil {
		t.Fatalf("Unexpected error in parseOAuthConfig: %v", err)
	}

	// Verify basic configuration is preserved
	if !cfg.OAuthConfig.Enabled {
		t.Error("Expected OAuth to be enabled")
	}
	if cfg.OAuthConfig.TenantID != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("Expected tenant ID '12345678-1234-1234-1234-123456789abc', got %s", cfg.OAuthConfig.TenantID)
	}
	if cfg.OAuthConfig.ClientID != "87654321-4321-4321-4321-cba987654321" {
		t.Errorf("Expected client ID '87654321-4321-4321-4321-cba987654321', got %s", cfg.OAuthConfig.ClientID)
	}
}

func TestOAuthRedirectURIsConfig(t *testing.T) {
	// Test OAuth redirect URIs configuration with additional URIs
	cfg := NewConfig()
	cfg.OAuthConfig.Enabled = true
	cfg.Host = "127.0.0.1"
	cfg.Port = 8081

	// Test with additional redirect URIs
	additionalRedirectURIs := "http://localhost:6274/oauth/callback,http://localhost:8080/oauth/callback"
	if err := cfg.parseOAuthConfig(additionalRedirectURIs, ""); err != nil {
		t.Fatalf("Unexpected error in parseOAuthConfig: %v", err)
	}

	// Should have default URIs plus additional ones
	expectedURIs := []string{
		"http://127.0.0.1:8081/oauth/callback",
		"http://localhost:8081/oauth/callback",
		"http://localhost:6274/oauth/callback",
		"http://localhost:8080/oauth/callback",
	}

	if len(cfg.OAuthConfig.RedirectURIs) != len(expectedURIs) {
		t.Errorf("Expected %d redirect URIs, got %d", len(expectedURIs), len(cfg.OAuthConfig.RedirectURIs))
	}

	for i, expected := range expectedURIs {
		if i >= len(cfg.OAuthConfig.RedirectURIs) || cfg.OAuthConfig.RedirectURIs[i] != expected {
			t.Errorf("Expected redirect URI '%s' at index %d, got '%s'", expected, i,
				func() string {
					if i < len(cfg.OAuthConfig.RedirectURIs) {
						return cfg.OAuthConfig.RedirectURIs[i]
					}
					return "missing"
				}())
		}
	}
}

func TestOAuthRedirectURIsEmptyAdditional(t *testing.T) {
	// Test OAuth redirect URIs configuration without additional URIs
	cfg := NewConfig()
	cfg.OAuthConfig.Enabled = true
	cfg.Host = "127.0.0.1"
	cfg.Port = 8081

	// Test with empty additional redirect URIs
	if err := cfg.parseOAuthConfig("", ""); err != nil {
		t.Fatalf("Unexpected error in parseOAuthConfig: %v", err)
	}

	// Should have only default URIs
	expectedURIs := []string{
		"http://127.0.0.1:8081/oauth/callback",
		"http://localhost:8081/oauth/callback",
	}

	if len(cfg.OAuthConfig.RedirectURIs) != len(expectedURIs) {
		t.Errorf("Expected %d redirect URIs, got %d", len(expectedURIs), len(cfg.OAuthConfig.RedirectURIs))
	}

	for i, expected := range expectedURIs {
		if cfg.OAuthConfig.RedirectURIs[i] != expected {
			t.Errorf("Expected redirect URI '%s' at index %d, got '%s'", expected, i, cfg.OAuthConfig.RedirectURIs[i])
		}
	}
}

func TestValidateGUID(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		wantErr   bool
	}{
		{
			name:      "valid GUID",
			value:     "12345678-1234-1234-1234-123456789abc",
			fieldName: "test field",
			wantErr:   false,
		},
		{
			name:      "valid GUID uppercase",
			value:     "12345678-1234-1234-1234-123456789ABC",
			fieldName: "test field",
			wantErr:   false,
		},
		{
			name:      "empty value allowed",
			value:     "",
			fieldName: "test field",
			wantErr:   false,
		},
		{
			name:      "invalid format - missing hyphens",
			value:     "123456781234123412341234567890ab",
			fieldName: "test field",
			wantErr:   true,
		},
		{
			name:      "invalid format - wrong length",
			value:     "12345678-1234-1234-1234-123456789",
			fieldName: "test field",
			wantErr:   true,
		},
		{
			name:      "invalid format - non-hex characters",
			value:     "12345678-1234-1234-1234-123456789abg",
			fieldName: "test field",
			wantErr:   true,
		},
		{
			name:      "invalid format - extra hyphens",
			value:     "12345678-1234-1234-1234-1234-56789abc",
			fieldName: "test field",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGUID(tt.value, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				// Verify error message contains the field name and value
				errorMsg := err.Error()
				if !contains(errorMsg, tt.fieldName) {
					t.Errorf("Error message should contain field name '%s', got: %s", tt.fieldName, errorMsg)
				}
				if tt.value != "" && !contains(errorMsg, tt.value) {
					t.Errorf("Error message should contain value '%s', got: %s", tt.value, errorMsg)
				}
			}
		})
	}
}

func TestOAuthGUIDValidation(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		clientID string
		wantErr  bool
	}{
		{
			name:     "valid GUIDs",
			tenantID: "12345678-1234-1234-1234-123456789abc",
			clientID: "87654321-4321-4321-4321-cba987654321",
			wantErr:  false,
		},
		{
			name:     "empty values allowed",
			tenantID: "",
			clientID: "",
			wantErr:  false,
		},
		{
			name:     "invalid tenant ID",
			tenantID: "invalid-tenant-id",
			clientID: "87654321-4321-4321-4321-cba987654321",
			wantErr:  true,
		},
		{
			name:     "invalid client ID",
			tenantID: "12345678-1234-1234-1234-123456789abc",
			clientID: "invalid-client-id",
			wantErr:  true,
		},
		{
			name:     "both invalid",
			tenantID: "invalid-tenant",
			clientID: "invalid-client",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.OAuthConfig.Enabled = true
			cfg.OAuthConfig.TenantID = tt.tenantID
			cfg.OAuthConfig.ClientID = tt.clientID
			cfg.Host = "127.0.0.1"
			cfg.Port = 8081

			err := cfg.parseOAuthConfig("", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOAuthConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
