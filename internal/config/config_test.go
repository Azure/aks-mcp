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

func TestNewConfig_UseLegacyTools_Default(t *testing.T) {
	cfg := NewConfig()
	if cfg.UseLegacyTools {
		t.Error("Expected UseLegacyTools to be false by default")
	}
}

func TestNewConfig_UseLegacyTools_FromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "enabled via env",
			envValue: "true",
			expected: true,
		},
		{
			name:     "disabled via env",
			envValue: "false",
			expected: false,
		},
		{
			name:     "empty env",
			envValue: "",
			expected: false,
		},
		{
			name:     "invalid value",
			envValue: "invalid",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldEnv := getEnvOrEmpty("USE_LEGACY_TOOLS")
			defer func() {
				if oldEnv != "" {
					setEnv(t, "USE_LEGACY_TOOLS", oldEnv)
				} else {
					unsetEnv(t, "USE_LEGACY_TOOLS")
				}
			}()

			if tt.envValue != "" {
				setEnv(t, "USE_LEGACY_TOOLS", tt.envValue)
			} else {
				unsetEnv(t, "USE_LEGACY_TOOLS")
			}

			cfg := NewConfig()
			if cfg.UseLegacyTools != tt.expected {
				t.Errorf("Expected UseLegacyTools to be %v, got %v", tt.expected, cfg.UseLegacyTools)
			}
		})
	}
}

func getEnvOrEmpty(key string) string {
	value, exists := lookupEnv(key)
	if !exists {
		return ""
	}
	return value
}

func lookupEnv(key string) (string, bool) {
	for _, env := range getAllEnv() {
		pair := splitEnvPair(env)
		if len(pair) == 2 && pair[0] == key {
			return pair[1], true
		}
	}
	return "", false
}

func getAllEnv() []string {
	return []string{}
}

func splitEnvPair(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func setEnv(t *testing.T, key, value string) {
	t.Setenv(key, value)
}

func unsetEnv(t *testing.T, key string) {
	t.Setenv(key, "")
}

func TestValidateConfig_OAuthWithStdio(t *testing.T) {
	cfg := NewConfig()
	cfg.OAuthConfig.Enabled = true
	cfg.Transport = "stdio"

	err := cfg.ValidateConfig()
	if err == nil {
		t.Fatal("Expected error when OAuth is enabled with stdio transport, got nil")
	}

	expectedMsg := "OAuth authentication is not supported with stdio transport per MCP specification"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestValidateConfig_OAuthWithSSE(t *testing.T) {
	cfg := NewConfig()
	cfg.OAuthConfig.Enabled = true
	cfg.Transport = "sse"

	err := cfg.ValidateConfig()
	if err != nil {
		t.Errorf("Expected no error for OAuth with SSE transport, got: %v", err)
	}
}

func TestValidateConfig_OAuthWithStreamableHTTP(t *testing.T) {
	cfg := NewConfig()
	cfg.OAuthConfig.Enabled = true
	cfg.Transport = "streamable-http"

	err := cfg.ValidateConfig()
	if err != nil {
		t.Errorf("Expected no error for OAuth with streamable-http transport, got: %v", err)
	}
}

func TestValidateConfig_MultiClusterWithLegacyTools(t *testing.T) {
	cfg := NewConfig()
	cfg.EnableMultiCluster = true
	cfg.UseLegacyTools = true
	cfg.Transport = "sse"

	err := cfg.ValidateConfig()
	if err == nil {
		t.Fatal("Expected error when multi-cluster is enabled with legacy tools, got nil")
	}

	expectedMsg := "multi-cluster mode (--enable-multi-cluster) requires unified tools and is not compatible with legacy tools (USE_LEGACY_TOOLS=true)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestValidateConfig_MultiClusterWithStdio(t *testing.T) {
	cfg := NewConfig()
	cfg.EnableMultiCluster = true
	cfg.Transport = "stdio"

	err := cfg.ValidateConfig()
	if err == nil {
		t.Fatal("Expected error when multi-cluster is enabled with stdio transport, got nil")
	}

	expectedMsg := "multi-cluster mode (--enable-multi-cluster) is not supported with stdio transport, use sse or streamable-http instead"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestValidateConfig_MultiClusterWithSSE(t *testing.T) {
	cfg := NewConfig()
	cfg.EnableMultiCluster = true
	cfg.Transport = "sse"
	cfg.UseLegacyTools = false

	err := cfg.ValidateConfig()
	if err != nil {
		t.Errorf("Expected no error for multi-cluster with SSE transport, got: %v", err)
	}
}

func TestValidateConfig_MultiClusterWithStreamableHTTP(t *testing.T) {
	cfg := NewConfig()
	cfg.EnableMultiCluster = true
	cfg.Transport = "streamable-http"
	cfg.UseLegacyTools = false

	err := cfg.ValidateConfig()
	if err != nil {
		t.Errorf("Expected no error for multi-cluster with streamable-http transport, got: %v", err)
	}
}

func TestValidateConfig_MultiClusterWithUnifiedTools(t *testing.T) {
	cfg := NewConfig()
	cfg.EnableMultiCluster = true
	cfg.UseLegacyTools = false
	cfg.Transport = "sse"

	err := cfg.ValidateConfig()
	if err != nil {
		t.Errorf("Expected no error for multi-cluster with unified tools, got: %v", err)
	}
}

func TestValidateConfig_LegacyToolsWithoutMultiCluster(t *testing.T) {
	cfg := NewConfig()
	cfg.EnableMultiCluster = false
	cfg.UseLegacyTools = true

	err := cfg.ValidateConfig()
	if err != nil {
		t.Errorf("Expected no error for legacy tools without multi-cluster, got: %v", err)
	}
}

func TestValidateConfig_ValidCombinations(t *testing.T) {
	tests := []struct {
		name               string
		oauthEnabled       bool
		transport          string
		enableMultiCluster bool
		useLegacyTools     bool
		wantErr            bool
	}{
		{
			name:               "OAuth disabled with stdio",
			oauthEnabled:       false,
			transport:          "stdio",
			enableMultiCluster: false,
			useLegacyTools:     false,
			wantErr:            false,
		},
		{
			name:               "OAuth enabled with SSE",
			oauthEnabled:       true,
			transport:          "sse",
			enableMultiCluster: false,
			useLegacyTools:     false,
			wantErr:            false,
		},
		{
			name:               "OAuth enabled with streamable-http",
			oauthEnabled:       true,
			transport:          "streamable-http",
			enableMultiCluster: false,
			useLegacyTools:     false,
			wantErr:            false,
		},
		{
			name:               "Multi-cluster with unified tools",
			oauthEnabled:       false,
			transport:          "sse",
			enableMultiCluster: true,
			useLegacyTools:     false,
			wantErr:            false,
		},
		{
			name:               "Single cluster with legacy tools",
			oauthEnabled:       false,
			transport:          "stdio",
			enableMultiCluster: false,
			useLegacyTools:     true,
			wantErr:            false,
		},
		{
			name:               "All features compatible",
			oauthEnabled:       true,
			transport:          "sse",
			enableMultiCluster: true,
			useLegacyTools:     false,
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.OAuthConfig.Enabled = tt.oauthEnabled
			cfg.Transport = tt.transport
			cfg.EnableMultiCluster = tt.enableMultiCluster
			cfg.UseLegacyTools = tt.useLegacyTools

			err := cfg.ValidateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig_InvalidCombinations(t *testing.T) {
	tests := []struct {
		name               string
		oauthEnabled       bool
		transport          string
		enableMultiCluster bool
		useLegacyTools     bool
		expectedErrMsg     string
	}{
		{
			name:               "OAuth with stdio",
			oauthEnabled:       true,
			transport:          "stdio",
			enableMultiCluster: false,
			useLegacyTools:     false,
			expectedErrMsg:     "OAuth authentication is not supported with stdio transport",
		},
		{
			name:               "Multi-cluster with stdio",
			oauthEnabled:       false,
			transport:          "stdio",
			enableMultiCluster: true,
			useLegacyTools:     false,
			expectedErrMsg:     "multi-cluster mode (--enable-multi-cluster) is not supported with stdio transport",
		},
		{
			name:               "Multi-cluster with legacy tools",
			oauthEnabled:       false,
			transport:          "sse",
			enableMultiCluster: true,
			useLegacyTools:     true,
			expectedErrMsg:     "multi-cluster mode (--enable-multi-cluster) requires unified tools",
		},
		{
			name:               "All invalid combinations",
			oauthEnabled:       true,
			transport:          "stdio",
			enableMultiCluster: true,
			useLegacyTools:     true,
			expectedErrMsg:     "OAuth authentication is not supported with stdio transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.OAuthConfig.Enabled = tt.oauthEnabled
			cfg.Transport = tt.transport
			cfg.EnableMultiCluster = tt.enableMultiCluster
			cfg.UseLegacyTools = tt.useLegacyTools

			err := cfg.ValidateConfig()
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErrMsg, err.Error())
			}
		})
	}
}
