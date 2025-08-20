package config

import (
	"fmt"
	"strings"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	// Whether authentication is enabled
	Enabled bool `json:"enabled"`

	// Microsoft Entra ID configuration
	EntraClientID  string `json:"entra_client_id"`
	EntraTenantID  string `json:"entra_tenant_id"`
	EntraAuthority string `json:"entra_authority"`

	// Cache configuration
	JWKSCacheTimeout int `json:"jwks_cache_timeout"` // seconds

	// Transport policy - which transports require authentication
	RequireAuthForHTTP bool `json:"require_auth_for_http"`
}

// NewAuthConfig creates a new AuthConfig with default values
func NewAuthConfig() *AuthConfig {
	return &AuthConfig{
		Enabled:            false,
		EntraAuthority:     "https://login.microsoftonline.com",
		JWKSCacheTimeout:   3600, // 1 hour
		RequireAuthForHTTP: true,
	}
}

// ShouldAuthenticate determines if authentication should be enabled for the given transport
func (c *AuthConfig) ShouldAuthenticate(transport string) bool {
	if !c.Enabled {
		return false
	}

	// stdio transport typically doesn't need authentication (local usage)
	if transport == "stdio" {
		return false
	}

	// HTTP transports need authentication if configured
	return c.RequireAuthForHTTP
}

// ValidateConfig validates the authentication configuration
func (c *AuthConfig) ValidateConfig() error {
	if !c.Enabled {
		return nil
	}

	if c.EntraClientID == "" {
		return fmt.Errorf("auth: entra_client_id is required when authentication is enabled")
	}

	if c.EntraTenantID == "" {
		return fmt.Errorf("auth: entra_tenant_id is required when authentication is enabled")
	}

	if c.JWKSCacheTimeout <= 0 {
		return fmt.Errorf("auth: jwks_cache_timeout must be positive")
	}

	return nil
}

// GetIssuer returns the OAuth issuer URL for the tenant
func (c *AuthConfig) GetIssuer() string {
	authority := strings.TrimSuffix(c.EntraAuthority, "/")
	return fmt.Sprintf("%s/%s/v2.0", authority, c.EntraTenantID)
}

// GetJWKSURL returns the JWKS endpoint URL for the tenant
func (c *AuthConfig) GetJWKSURL() string {
	authority := strings.TrimSuffix(c.EntraAuthority, "/")
	return fmt.Sprintf("%s/%s/discovery/v2.0/keys", authority, c.EntraTenantID)
}
