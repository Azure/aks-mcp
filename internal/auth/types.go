package auth

import (
	"fmt"
	"time"
)

// OAuthConfig represents OAuth configuration for AKS-MCP
type OAuthConfig struct {
	// Enable OAuth authentication
	Enabled bool `json:"enabled"`

	// Azure AD tenant ID
	TenantID string `json:"tenant_id"`

	// Azure AD application (client) ID
	ClientID string `json:"client_id"`

	// Required OAuth scopes for accessing AKS-MCP
	RequiredScopes []string `json:"required_scopes"`

	// Allowed redirect URIs for OAuth callback
	RedirectURIs []string `json:"redirect_uris"`

	// Allowed CORS origins for OAuth endpoints (for security, wildcard "*" should be avoided)
	AllowedOrigins []string `json:"allowed_origins"`

	// Token validation settings
	TokenValidation TokenValidationConfig `json:"token_validation"`
}

// TokenValidationConfig represents token validation configuration
type TokenValidationConfig struct {
	// SECURITY CRITICAL: Enable JWT token validation
	// Setting this to false creates a security vulnerability - for development/testing ONLY
	// MUST be true in production environments
	ValidateJWT bool `json:"validate_jwt"`

	// Enable audience validation
	ValidateAudience bool `json:"validate_audience"`

	// Expected audience for tokens
	ExpectedAudience string `json:"expected_audience"`

	// Token cache TTL
	CacheTTL time.Duration `json:"cache_ttl"`

	// Clock skew tolerance for token validation
	ClockSkew time.Duration `json:"clock_skew"`
}

// TokenInfo represents validated token information
type TokenInfo struct {
	// Access token
	AccessToken string `json:"access_token"` // #nosec G117 -- Standard OAuth2 field name

	// Token type (usually "Bearer")
	TokenType string `json:"token_type"`

	// Token expiration time
	ExpiresAt time.Time `json:"expires_at"`

	// Token scope
	Scope []string `json:"scope"`

	// Subject (user ID)
	Subject string `json:"subject"`

	// Audience
	Audience []string `json:"audience"`

	// Issuer
	Issuer string `json:"issuer"`

	// Additional claims
	Claims map[string]interface{} `json:"claims"`
}

// AuthResult represents the result of authentication
type AuthResult struct {
	// Whether authentication was successful
	Authenticated bool `json:"authenticated"`

	// Token information (if authenticated)
	TokenInfo *TokenInfo `json:"token_info,omitempty"`

	// Error message (if authentication failed)
	Error string `json:"error,omitempty"`

	// HTTP status code to return
	StatusCode int `json:"status_code"`
}

// Default OAuth configuration values
const (
	DefaultTokenCacheTTL    = 5 * time.Minute
	DefaultClockSkew        = 1 * time.Minute
	DefaultExpectedAudience = "https://management.azure.com"
	AzureADScope            = "https://management.azure.com/.default"
)

// NewDefaultOAuthConfig creates a default OAuth configuration
func NewDefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		Enabled: false,
		// Use Azure Management API scope to get v2.0 format tokens
		// This ensures we get v2.0 issuer format which works with v2.0 JWKS endpoints
		RequiredScopes: []string{AzureADScope}, // "https://management.azure.com/.default"
		// RedirectURIs will be populated dynamically based on host/port configuration
		RedirectURIs: []string{},
		TokenValidation: TokenValidationConfig{
			ValidateJWT:      true,                    // SECURITY CRITICAL: Always true in production
			ValidateAudience: true,                    // Re-enabled with correct audience
			ExpectedAudience: DefaultExpectedAudience, // "https://management.azure.com"
			CacheTTL:         DefaultTokenCacheTTL,
			ClockSkew:        DefaultClockSkew,
		},
	}
}

// Validate validates the OAuth configuration
func (cfg *OAuthConfig) Validate() error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.TenantID == "" {
		return fmt.Errorf("tenant_id is required when OAuth is enabled")
	}

	if cfg.ClientID == "" {
		return fmt.Errorf("client_id is required when OAuth is enabled")
	}

	// if len(cfg.RequiredScopes) == 0 {
	// 	return fmt.Errorf("at least one required scope must be specified")
	// }

	return nil
}
