package auth

// OAuthConfig is retained temporarily so command-line compatibility checks can
// reject the removed HTTP-only OAuth options before server initialization.
type OAuthConfig struct {
	Enabled                 bool
	TenantID                string
	ClientID                string
	ClientSecret            string
	ExternalURL             string
	OBOEnabled              bool
	RequiredScopes          []string
	RedirectURIs            []string
	AllowedOrigins          []string
	TokenValidation         TokenValidationConfig
}

type TokenValidationConfig struct {
	ExpectedAudience string
}

func NewDefaultOAuthConfig() *OAuthConfig { return &OAuthConfig{} }
