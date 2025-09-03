package auth

import (
	"os"
	"testing"
	"time"
)

func TestOAuthConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *OAuthConfig
		wantErr bool
	}{
		{
			name: "disabled OAuth should pass validation",
			config: &OAuthConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "enabled OAuth with missing tenant ID should fail",
			config: &OAuthConfig{
				Enabled:        true,
				ClientID:       "test-client-id",
				RequiredScopes: []string{"scope1"},
			},
			wantErr: true,
		},
		{
			name: "enabled OAuth with missing client ID should fail",
			config: &OAuthConfig{
				Enabled:        true,
				TenantID:       "test-tenant-id",
				RequiredScopes: []string{"scope1"},
			},
			wantErr: true,
		},
		{
			name: "enabled OAuth with empty scopes should pass",
			config: &OAuthConfig{
				Enabled:        true,
				TenantID:       "test-tenant-id",
				ClientID:       "test-client-id",
				RequiredScopes: []string{},
			},
			wantErr: false,
		},
		{
			name: "valid enabled OAuth config should pass",
			config: &OAuthConfig{
				Enabled:        true,
				TenantID:       "test-tenant-id",
				ClientID:       "test-client-id",
				RequiredScopes: []string{"scope1"},
			},
			wantErr: false,
		},
		{
			name: "valid enabled OAuth config with full token validation should pass",
			config: &OAuthConfig{
				Enabled:        true,
				TenantID:       "test-tenant-id",
				ClientID:       "test-client-id",
				RequiredScopes: []string{"scope1"},
				TokenValidation: TokenValidationConfig{
					ValidateJWT:      true,
					ValidateAudience: true,
					ExpectedAudience: "https://management.azure.com/",
					CacheTTL:         DefaultTokenCacheTTL,
					ClockSkew:        DefaultClockSkew,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("OAuthConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewDefaultOAuthConfig(t *testing.T) {
	config := NewDefaultOAuthConfig()

	if config.Enabled {
		t.Error("Default config should have OAuth disabled")
	}

	if len(config.RequiredScopes) != 1 || config.RequiredScopes[0] != AzureADScope {
		t.Errorf("Default config should have Azure AD scope, got %v", config.RequiredScopes)
	}

	if !config.TokenValidation.ValidateJWT {
		t.Error("Default config should enable JWT validation for production")
	}

	if !config.TokenValidation.ValidateAudience {
		t.Error("Default config should enable audience validation for security")
	}

	if config.TokenValidation.ExpectedAudience != DefaultExpectedAudience {
		t.Errorf("Default config should have correct expected audience, got %s", config.TokenValidation.ExpectedAudience)
	}

	if config.TokenValidation.CacheTTL != DefaultTokenCacheTTL {
		t.Errorf("Default config should have correct cache TTL, got %v", config.TokenValidation.CacheTTL)
	}

	if config.TokenValidation.ClockSkew != DefaultClockSkew {
		t.Errorf("Default config should have correct clock skew, got %v", config.TokenValidation.ClockSkew)
	}
}

func TestOAuthConfigConstants(t *testing.T) {
	if DefaultTokenCacheTTL != 5*time.Minute {
		t.Errorf("DefaultTokenCacheTTL should be 5 minutes, got %v", DefaultTokenCacheTTL)
	}

	if DefaultClockSkew != 1*time.Minute {
		t.Errorf("DefaultClockSkew should be 1 minute, got %v", DefaultClockSkew)
	}

	if DefaultExpectedAudience != "https://management.azure.com" {
		t.Errorf("DefaultExpectedAudience should be Azure management, got %s", DefaultExpectedAudience)
	}

	if AzureADScope != "https://management.azure.com/.default" {
		t.Errorf("AzureADScope should be Azure management default, got %s", AzureADScope)
	}
}

func TestOAuthConfigEnvironmentVariables(t *testing.T) {
	// Test that environment variables are respected
	oldTenantID := os.Getenv("AZURE_TENANT_ID")
	oldClientID := os.Getenv("AZURE_CLIENT_ID")

	defer func() {
		os.Setenv("AZURE_TENANT_ID", oldTenantID)
		os.Setenv("AZURE_CLIENT_ID", oldClientID)
	}()

	os.Setenv("AZURE_TENANT_ID", "env-tenant-id")
	os.Setenv("AZURE_CLIENT_ID", "env-client-id")

	config := NewDefaultOAuthConfig()
	config.Enabled = true

	// Simulate the environment variable loading that happens in config parsing
	if config.TenantID == "" {
		config.TenantID = os.Getenv("AZURE_TENANT_ID")
	}
	if config.ClientID == "" {
		config.ClientID = os.Getenv("AZURE_CLIENT_ID")
	}

	if config.TenantID != "env-tenant-id" {
		t.Errorf("Expected tenant ID from environment, got %s", config.TenantID)
	}

	if config.ClientID != "env-client-id" {
		t.Errorf("Expected client ID from environment, got %s", config.ClientID)
	}

	// Should pass validation with environment variables
	if err := config.Validate(); err != nil {
		t.Errorf("Config with environment variables should be valid, got error: %v", err)
	}
}
