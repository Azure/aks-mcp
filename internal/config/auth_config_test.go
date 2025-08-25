package config

import (
	"strings"
	"testing"
)

func TestNewAuthConfig(t *testing.T) {
	config := NewAuthConfig()

	// Test default values
	if config.Enabled {
		t.Error("Expected Enabled to be false by default")
	}

	if config.EntraAuthority != "https://login.microsoftonline.com" {
		t.Errorf("Expected EntraAuthority to be 'https://login.microsoftonline.com', got '%s'", config.EntraAuthority)
	}

	if config.JWKSCacheTimeout != 3600 {
		t.Errorf("Expected JWKSCacheTimeout to be 3600, got %d", config.JWKSCacheTimeout)
	}

	if !config.RequireAuthForHTTP {
		t.Error("Expected RequireAuthForHTTP to be true by default")
	}

	// Test that optional fields are empty by default
	if config.EntraClientID != "" {
		t.Errorf("Expected EntraClientID to be empty by default, got '%s'", config.EntraClientID)
	}

	if config.EntraTenantID != "" {
		t.Errorf("Expected EntraTenantID to be empty by default, got '%s'", config.EntraTenantID)
	}
}

func TestAuthConfig_ShouldAuthenticate(t *testing.T) {
	tests := []struct {
		name               string
		enabled            bool
		requireAuthForHTTP bool
		transport          string
		expectedResult     bool
	}{
		{
			name:               "disabled auth - stdio transport",
			enabled:            false,
			requireAuthForHTTP: true,
			transport:          "stdio",
			expectedResult:     false,
		},
		{
			name:               "disabled auth - http transport",
			enabled:            false,
			requireAuthForHTTP: true,
			transport:          "streamable-http",
			expectedResult:     false,
		},
		{
			name:               "enabled auth - stdio transport",
			enabled:            true,
			requireAuthForHTTP: true,
			transport:          "stdio",
			expectedResult:     false,
		},
		{
			name:               "enabled auth - http transport with RequireAuthForHTTP true",
			enabled:            true,
			requireAuthForHTTP: true,
			transport:          "streamable-http",
			expectedResult:     true,
		},
		{
			name:               "enabled auth - http transport with RequireAuthForHTTP false",
			enabled:            true,
			requireAuthForHTTP: false,
			transport:          "streamable-http",
			expectedResult:     false,
		},
		{
			name:               "enabled auth - sse transport with RequireAuthForHTTP true",
			enabled:            true,
			requireAuthForHTTP: true,
			transport:          "sse",
			expectedResult:     true,
		},
		{
			name:               "enabled auth - sse transport with RequireAuthForHTTP false",
			enabled:            true,
			requireAuthForHTTP: false,
			transport:          "sse",
			expectedResult:     false,
		},
		{
			name:               "enabled auth - custom transport",
			enabled:            true,
			requireAuthForHTTP: true,
			transport:          "custom-transport",
			expectedResult:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &AuthConfig{
				Enabled:            test.enabled,
				RequireAuthForHTTP: test.requireAuthForHTTP,
			}

			result := config.ShouldAuthenticate(test.transport)
			if result != test.expectedResult {
				t.Errorf("Expected ShouldAuthenticate to return %v, got %v", test.expectedResult, result)
			}
		})
	}
}

func TestAuthConfig_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *AuthConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "disabled config should pass validation",
			config: &AuthConfig{
				Enabled: false,
				// All other fields can be empty when disabled
			},
			expectError: false,
		},
		{
			name: "enabled config with all required fields",
			config: &AuthConfig{
				Enabled:          true,
				EntraClientID:    "test-client-id",
				EntraTenantID:    "test-tenant-id",
				JWKSCacheTimeout: 3600,
			},
			expectError: false,
		},
		{
			name: "enabled config missing client ID",
			config: &AuthConfig{
				Enabled:          true,
				EntraTenantID:    "test-tenant-id",
				JWKSCacheTimeout: 3600,
			},
			expectError: true,
			errorMsg:    "entra_client_id is required",
		},
		{
			name: "enabled config missing tenant ID",
			config: &AuthConfig{
				Enabled:          true,
				EntraClientID:    "test-client-id",
				JWKSCacheTimeout: 3600,
			},
			expectError: true,
			errorMsg:    "entra_tenant_id is required",
		},
		{
			name: "enabled config with zero cache timeout",
			config: &AuthConfig{
				Enabled:          true,
				EntraClientID:    "test-client-id",
				EntraTenantID:    "test-tenant-id",
				JWKSCacheTimeout: 0,
			},
			expectError: true,
			errorMsg:    "jwks_cache_timeout must be positive",
		},
		{
			name: "enabled config with negative cache timeout",
			config: &AuthConfig{
				Enabled:          true,
				EntraClientID:    "test-client-id",
				EntraTenantID:    "test-tenant-id",
				JWKSCacheTimeout: -100,
			},
			expectError: true,
			errorMsg:    "jwks_cache_timeout must be positive",
		},
		{
			name: "enabled config with empty client ID",
			config: &AuthConfig{
				Enabled:          true,
				EntraClientID:    "",
				EntraTenantID:    "test-tenant-id",
				JWKSCacheTimeout: 3600,
			},
			expectError: true,
			errorMsg:    "entra_client_id is required",
		},
		{
			name: "enabled config with empty tenant ID",
			config: &AuthConfig{
				Enabled:          true,
				EntraClientID:    "test-client-id",
				EntraTenantID:    "",
				JWKSCacheTimeout: 3600,
			},
			expectError: true,
			errorMsg:    "entra_tenant_id is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.ValidateConfig()

			if test.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if test.errorMsg != "" && !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestAuthConfig_GetIssuer(t *testing.T) {
	tests := []struct {
		name        string
		authority   string
		tenantID    string
		expectedURL string
	}{
		{
			name:        "standard authority without trailing slash",
			authority:   "https://login.microsoftonline.com",
			tenantID:    "12345678-1234-1234-1234-123456789012",
			expectedURL: "https://login.microsoftonline.com/12345678-1234-1234-1234-123456789012/v2.0",
		},
		{
			name:        "authority with trailing slash",
			authority:   "https://login.microsoftonline.com/",
			tenantID:    "test-tenant-id",
			expectedURL: "https://login.microsoftonline.com/test-tenant-id/v2.0",
		},
		{
			name:        "custom authority",
			authority:   "https://custom.authority.com",
			tenantID:    "tenant123",
			expectedURL: "https://custom.authority.com/tenant123/v2.0",
		},
		{
			name:        "authority with multiple trailing slashes",
			authority:   "https://login.microsoftonline.com///",
			tenantID:    "tenant-id",
			expectedURL: "https://login.microsoftonline.com///tenant-id/v2.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &AuthConfig{
				EntraAuthority: test.authority,
				EntraTenantID:  test.tenantID,
			}

			issuer := config.GetIssuer()
			if issuer != test.expectedURL {
				t.Errorf("Expected issuer URL '%s', got '%s'", test.expectedURL, issuer)
			}
		})
	}
}

func TestAuthConfig_GetJWKSURL(t *testing.T) {
	tests := []struct {
		name        string
		authority   string
		tenantID    string
		expectedURL string
	}{
		{
			name:        "standard authority without trailing slash",
			authority:   "https://login.microsoftonline.com",
			tenantID:    "12345678-1234-1234-1234-123456789012",
			expectedURL: "https://login.microsoftonline.com/12345678-1234-1234-1234-123456789012/discovery/v2.0/keys",
		},
		{
			name:        "authority with trailing slash",
			authority:   "https://login.microsoftonline.com/",
			tenantID:    "test-tenant-id",
			expectedURL: "https://login.microsoftonline.com/test-tenant-id/discovery/v2.0/keys",
		},
		{
			name:        "custom authority",
			authority:   "https://custom.authority.com",
			tenantID:    "tenant123",
			expectedURL: "https://custom.authority.com/tenant123/discovery/v2.0/keys",
		},
		{
			name:        "authority with multiple trailing slashes",
			authority:   "https://login.microsoftonline.com///",
			tenantID:    "tenant-id",
			expectedURL: "https://login.microsoftonline.com///tenant-id/discovery/v2.0/keys",
		},
		{
			name:        "empty tenant ID",
			authority:   "https://login.microsoftonline.com",
			tenantID:    "",
			expectedURL: "https://login.microsoftonline.com//discovery/v2.0/keys",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &AuthConfig{
				EntraAuthority: test.authority,
				EntraTenantID:  test.tenantID,
			}

			jwksURL := config.GetJWKSURL()
			if jwksURL != test.expectedURL {
				t.Errorf("Expected JWKS URL '%s', got '%s'", test.expectedURL, jwksURL)
			}
		})
	}
}

func TestAuthConfig_AllFieldsAndMethods(t *testing.T) {
	// Test with a fully populated config
	config := &AuthConfig{
		Enabled:            true,
		EntraClientID:      "client-12345",
		EntraTenantID:      "tenant-67890",
		EntraAuthority:     "https://login.microsoftonline.us",
		JWKSCacheTimeout:   7200, // 2 hours
		RequireAuthForHTTP: false,
	}

	// Test field access
	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if config.EntraClientID != "client-12345" {
		t.Errorf("Expected EntraClientID to be 'client-12345', got '%s'", config.EntraClientID)
	}

	if config.EntraTenantID != "tenant-67890" {
		t.Errorf("Expected EntraTenantID to be 'tenant-67890', got '%s'", config.EntraTenantID)
	}

	if config.EntraAuthority != "https://login.microsoftonline.us" {
		t.Errorf("Expected EntraAuthority to be 'https://login.microsoftonline.us', got '%s'", config.EntraAuthority)
	}

	if config.JWKSCacheTimeout != 7200 {
		t.Errorf("Expected JWKSCacheTimeout to be 7200, got %d", config.JWKSCacheTimeout)
	}

	if config.RequireAuthForHTTP {
		t.Error("Expected RequireAuthForHTTP to be false")
	}

	// Test validation passes with all required fields
	if err := config.ValidateConfig(); err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}

	// Test URL generation
	expectedIssuer := "https://login.microsoftonline.us/tenant-67890/v2.0"
	if issuer := config.GetIssuer(); issuer != expectedIssuer {
		t.Errorf("Expected issuer '%s', got '%s'", expectedIssuer, issuer)
	}

	expectedJWKS := "https://login.microsoftonline.us/tenant-67890/discovery/v2.0/keys"
	if jwksURL := config.GetJWKSURL(); jwksURL != expectedJWKS {
		t.Errorf("Expected JWKS URL '%s', got '%s'", expectedJWKS, jwksURL)
	}

	// Test ShouldAuthenticate behavior
	if config.ShouldAuthenticate("stdio") {
		t.Error("Expected ShouldAuthenticate to return false for stdio transport")
	}

	if config.ShouldAuthenticate("streamable-http") {
		t.Error("Expected ShouldAuthenticate to return false when RequireAuthForHTTP is false")
	}
}

func TestAuthConfig_EdgeCases(t *testing.T) {
	t.Run("empty strings in URLs", func(t *testing.T) {
		config := &AuthConfig{
			EntraAuthority: "",
			EntraTenantID:  "",
		}

		issuer := config.GetIssuer()
		expectedIssuer := "//v2.0"
		if issuer != expectedIssuer {
			t.Errorf("Expected issuer '%s' with empty authority, got '%s'", expectedIssuer, issuer)
		}

		jwksURL := config.GetJWKSURL()
		expectedJWKS := "//discovery/v2.0/keys"
		if jwksURL != expectedJWKS {
			t.Errorf("Expected JWKS URL '%s' with empty authority, got '%s'", expectedJWKS, jwksURL)
		}
	})

	t.Run("whitespace in configuration", func(t *testing.T) {
		config := &AuthConfig{
			Enabled:          true,
			EntraClientID:    "  client-id  ",
			EntraTenantID:    "  tenant-id  ",
			EntraAuthority:   "  https://login.microsoftonline.com  ",
			JWKSCacheTimeout: 3600,
		}

		// Validation should pass (no trimming is done in the current implementation)
		if err := config.ValidateConfig(); err != nil {
			t.Errorf("Expected validation to pass with whitespace, got error: %v", err)
		}

		// URLs should include the whitespace as-is
		issuer := config.GetIssuer()
		if !strings.Contains(issuer, "  tenant-id  ") {
			t.Errorf("Expected issuer to contain whitespace in tenant ID, got '%s'", issuer)
		}
	})
}
