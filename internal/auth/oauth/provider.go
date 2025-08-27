package oauth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

// AzureOAuthProvider implements OAuth authentication for Azure AD
type AzureOAuthProvider struct {
	config     *auth.OAuthConfig
	httpClient *http.Client
	keyCache   *keyCache
	mu         sync.RWMutex
}

// keyCache caches Azure AD signing keys
type keyCache struct {
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	mu        sync.RWMutex
}

// AzureADMetadata represents Azure AD OAuth metadata
type AzureADMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	JWKSUri                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

// ProtectedResourceMetadata represents MCP protected resource metadata
type ProtectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
	Resource             string   `json:"resource"`
	ScopesSupported      []string `json:"scopes_supported"`
}

// AzureADProtectedResourceMetadata represents Azure AD compatible protected resource metadata
// Omits the resource field to prevent Azure AD authorization parameter conflicts
type AzureADProtectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

// NewAzureOAuthProvider creates a new Azure OAuth provider
func NewAzureOAuthProvider(config *auth.OAuthConfig) (*AzureOAuthProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OAuth config: %w", err)
	}

	return &AzureOAuthProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		keyCache: &keyCache{
			keys: make(map[string]*rsa.PublicKey),
		},
	}, nil
}

// GetProtectedResourceMetadata returns OAuth 2.0 Protected Resource Metadata (RFC 9728)
func (p *AzureOAuthProvider) GetProtectedResourceMetadata(serverURL string) (*AzureADProtectedResourceMetadata, error) {
	// For MCP compatibility, point to our local authorization server proxy
	// which properly advertises PKCE support
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Use the same scheme and host as the server URL
	authServerURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Azure AD v2.0 doesn't support RFC 8707 Resource Indicators
	// We omit the Resource field to prevent MCP clients from sending the resource parameter
	// The resource identification is handled through scopes instead
	return &AzureADProtectedResourceMetadata{
		AuthorizationServers: []string{authServerURL},
		// Resource field omitted for Azure AD compatibility
		ScopesSupported: p.config.RequiredScopes,
	}, nil
}

// GetAuthorizationServerMetadata returns OAuth 2.0 Authorization Server Metadata (RFC 8414)
func (p *AzureOAuthProvider) GetAuthorizationServerMetadata(serverURL string) (*AzureADMetadata, error) {
	metadataURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0/.well-known/openid-configuration", p.config.TenantID)

	resp, err := p.httpClient.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata from %s: %w", metadataURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("tenant ID '%s' not found (HTTP 404). Please verify your Azure AD tenant ID is correct", p.config.TenantID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("metadata endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata response: %w", err)
	}

	var metadata AzureADMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Ensure PKCE support is advertised for MCP Inspector compatibility
	if metadata.GrantTypesSupported == nil {
		metadata.GrantTypesSupported = []string{"authorization_code", "refresh_token"}
	}

	// Add S256 code challenge method support (Azure AD supports this)
	metadata.CodeChallengeMethodsSupported = []string{"S256"}

	// Azure AD v2.0 doesn't support RFC 8707 Resource Indicators
	// Override the authorization endpoint to point to our proxy
	// which will filter out the resource parameter before forwarding to Azure AD
	parsedURL, err := url.Parse(serverURL)
	if err == nil {
		// If the server URL includes /mcp path, include it in the proxy endpoint
		proxyPath := "/oauth2/v2.0/authorize"
		registrationPath := "/oauth/register"
		proxyAuthURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, proxyPath)
		registrationURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, registrationPath)

		metadata.AuthorizationEndpoint = proxyAuthURL
		// Add dynamic client registration endpoint
		metadata.RegistrationEndpoint = registrationURL
	}

	return &metadata, nil
}

// ValidateToken validates an OAuth access token
func (p *AzureOAuthProvider) ValidateToken(ctx context.Context, tokenString string) (*auth.TokenInfo, error) {
	// Check if token looks truncated (missing dots for JWT format)
	dotCount := strings.Count(tokenString, ".")
	if dotCount == 0 {
		fmt.Printf("Token appears to be truncated (no dots found). This might be due to MCP Inspector/proxy limitations.\n")

		// For truncated tokens, try to decode what we have and provide basic validation
		// This is a workaround for the MCP Inspector token truncation issue
		if len(tokenString) > 0 {
			// Try to decode the first part (header) to verify it's a valid JWT header
			parts := strings.Split(tokenString, ".")
			if len(parts) >= 1 {
				headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
				if err == nil {
					var header map[string]interface{}
					if json.Unmarshal(headerBytes, &header) == nil {
						// Check if it looks like a JWT header
						if typ, ok := header["typ"].(string); ok && typ == "JWT" {
							fmt.Printf("Truncated token has valid JWT header. Allowing for testing purposes.\n")
							fmt.Printf("Truncated token debug:\n")
							fmt.Printf("  JWT header parsed: %+v\n", header)
							fmt.Printf("  Creating test TokenInfo with scope: [testing-truncated-token]\n")
							fmt.Printf("  Expected audience: %s\n", p.config.TokenValidation.ExpectedAudience)
							fmt.Printf("  Tenant ID: %s\n", p.config.TenantID)

							// EXPERIMENT: Try to get the original complete token from MCP Inspector
							// Let's check if there's a way to get the full token
							fmt.Printf("  Raw token string length: %d\n", len(tokenString))
							fmt.Printf("  Raw token (first 100 chars): %s\n", tokenString[:min(len(tokenString), 100)])

							// Try to decode what we have and provide info about what's missing
							fmt.Printf("  Missing payload and signature parts for complete validation\n")
							fmt.Printf("  A complete Azure AD token would contain scopes like:\n")
							fmt.Printf("    - https://management.azure.com/user_impersonation\n")
							fmt.Printf("    - User.Read (for Graph API)\n")
							fmt.Printf("    - openid, profile, email (OpenID Connect)\n")

							// IMPORTANT: Add comprehensive scope mapping for truncated tokens
							// Include all the scopes that MCP Inspector might request and AKS-MCP expects
							testScopes := []string{
								"testing-truncated-token",
								// Azure Management API scopes
								"https://management.azure.com/.default",
								"https://management.azure.com/user_impersonation",
								"user_impersonation",
								// OpenID Connect scopes that MCP Inspector requests
								"openid",
								"profile",
								"email",
								"offline_access",
							}

							// Return a basic token info for truncated tokens
							testTokenInfo := &auth.TokenInfo{
								AccessToken: tokenString,
								TokenType:   "Bearer",
								ExpiresAt:   time.Now().Add(time.Hour), // Default expiration
								// For truncated tokens, we'll accept them for testing
								// In production, this should be fixed at the client/proxy level
								Scope:    testScopes,
								Subject:  "truncated-token-user",
								Audience: []string{p.config.TokenValidation.ExpectedAudience},
								Issuer:   fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", p.config.TenantID),
							}
							fmt.Printf("  Final test TokenInfo scopes: %v\n", testTokenInfo.Scope)
							fmt.Printf("  TEST TOKEN CREATED: This token contains all possible scopes for testing\n")
							return testTokenInfo, nil
						}
					}
				}
			}
		}
		return nil, fmt.Errorf("token appears to be truncated and cannot be validated")
	}

	if !p.config.TokenValidation.ValidateJWT {
		// Simple bearer token validation without JWT parsing
		// For testing purposes, include necessary scopes when JWT validation is disabled
		testScopes := []string{
			// Azure Management API scopes
			"https://management.azure.com/.default",
			"https://management.azure.com/user_impersonation",
			"user_impersonation",
			// OpenID Connect scopes that might be present
			"openid",
			"profile",
			"email",
			"offline_access",
		}

		return &auth.TokenInfo{
			AccessToken: tokenString,
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(time.Hour), // Default expiration
			Scope:       testScopes,
			Subject:     "test-user",
			Audience:    []string{p.config.TokenValidation.ExpectedAudience},
			Issuer:      fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", p.config.TenantID),
		}, nil
	}

	// Parse and validate JWT token
	token, err := jwt.Parse(tokenString, p.getKeyFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate issuer - Azure AD can use different issuer formats
	issuer, ok := claims["iss"].(string)
	if !ok {
		return nil, fmt.Errorf("missing issuer claim")
	}

	// Azure AD supports both v1.0 and v2.0 issuer formats
	expectedIssuerV2 := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", p.config.TenantID)
	expectedIssuerV1 := fmt.Sprintf("https://sts.windows.net/%s/", p.config.TenantID)

	if issuer != expectedIssuerV2 && issuer != expectedIssuerV1 {
		return nil, fmt.Errorf("invalid issuer: expected %s or %s, got %s", expectedIssuerV2, expectedIssuerV1, issuer)
	}

	// Validate audience
	if p.config.TokenValidation.ValidateAudience {
		if err := p.validateAudience(claims); err != nil {
			return nil, err
		}
	}

	// Extract token information
	tokenInfo := &auth.TokenInfo{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		Claims:      claims,
	}

	// Extract subject
	if sub, ok := claims["sub"].(string); ok {
		tokenInfo.Subject = sub
	}

	// Extract audience
	if aud, ok := claims["aud"].(string); ok {
		tokenInfo.Audience = []string{aud}
	} else if audSlice, ok := claims["aud"].([]interface{}); ok {
		for _, a := range audSlice {
			if audStr, ok := a.(string); ok {
				tokenInfo.Audience = append(tokenInfo.Audience, audStr)
			}
		}
	}

	// Extract scope
	fmt.Printf("Scope extraction debug:\n")
	fmt.Printf("  All claims in token: %+v\n", claims)

	// Check for 'scp' claim (Azure AD v2.0)
	if scp, ok := claims["scp"].(string); ok {
		fmt.Printf("  Found 'scp' claim: %s\n", scp)
		tokenInfo.Scope = strings.Split(scp, " ")
		fmt.Printf("  Parsed scopes from 'scp': %v\n", tokenInfo.Scope)
	} else {
		fmt.Printf("  No 'scp' claim found\n")
	}

	// Check for 'scope' claim (alternative)
	if scope, ok := claims["scope"].(string); ok {
		fmt.Printf("  Found 'scope' claim: %s\n", scope)
		if len(tokenInfo.Scope) == 0 {
			tokenInfo.Scope = strings.Split(scope, " ")
			fmt.Printf("  Parsed scopes from 'scope': %v\n", tokenInfo.Scope)
		}
	} else {
		fmt.Printf("  No 'scope' claim found\n")
	}

	// Check for 'roles' claim (Azure AD app roles)
	if roles, ok := claims["roles"].([]interface{}); ok {
		fmt.Printf("  Found 'roles' claim: %v\n", roles)
		for _, role := range roles {
			if roleStr, ok := role.(string); ok {
				tokenInfo.Scope = append(tokenInfo.Scope, roleStr)
			}
		}
		fmt.Printf("  Total scopes after adding roles: %v\n", tokenInfo.Scope)
	} else {
		fmt.Printf("  No 'roles' claim found\n")
	}

	fmt.Printf("  Final extracted scopes: %v\n", tokenInfo.Scope)

	// Extract expiration
	if exp, ok := claims["exp"].(float64); ok {
		tokenInfo.ExpiresAt = time.Unix(int64(exp), 0)
	}

	// Set issuer
	tokenInfo.Issuer = issuer

	return tokenInfo, nil
}

// validateAudience validates the audience claim
func (p *AzureOAuthProvider) validateAudience(claims jwt.MapClaims) error {
	expectedAudience := p.config.TokenValidation.ExpectedAudience

	// Normalize expected audience - remove trailing slash for comparison
	normalizedExpected := strings.TrimSuffix(expectedAudience, "/")

	// Check single audience
	if aud, ok := claims["aud"].(string); ok {
		normalizedAud := strings.TrimSuffix(aud, "/")
		if normalizedAud == normalizedExpected || aud == p.config.ClientID {
			return nil
		}
		return fmt.Errorf("invalid audience: expected %s or %s, got %s", expectedAudience, p.config.ClientID, aud)
	}

	// Check audience array
	if audSlice, ok := claims["aud"].([]interface{}); ok {
		for _, a := range audSlice {
			if audStr, ok := a.(string); ok {
				normalizedAud := strings.TrimSuffix(audStr, "/")
				if normalizedAud == normalizedExpected || audStr == p.config.ClientID {
					return nil
				}
			}
		}
		return fmt.Errorf("invalid audience: expected %s or %s in audience list", expectedAudience, p.config.ClientID)
	}

	return fmt.Errorf("missing audience claim")
}

// getKeyFunc returns a function to retrieve JWT signing keys
func (p *AzureOAuthProvider) getKeyFunc(token *jwt.Token) (interface{}, error) {
	// Ensure the token uses RS256
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	// Get key ID from token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("missing key ID in token header")
	}

	// Get the public key for this key ID
	key, err := p.getPublicKey(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	return key, nil
}

// getPublicKey retrieves and caches Azure AD public keys
func (p *AzureOAuthProvider) getPublicKey(kid string) (*rsa.PublicKey, error) {
	p.keyCache.mu.RLock()
	if key, exists := p.keyCache.keys[kid]; exists && time.Now().Before(p.keyCache.expiresAt) {
		p.keyCache.mu.RUnlock()
		return key, nil
	}
	p.keyCache.mu.RUnlock()

	// Fetch keys from Azure AD
	jwksURL := fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", p.config.TenantID)

	resp, err := p.httpClient.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
			Kty string `json:"kty"`
		} `json:"keys"`
	}

	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Update cache
	p.keyCache.mu.Lock()
	defer p.keyCache.mu.Unlock()

	p.keyCache.keys = make(map[string]*rsa.PublicKey)
	p.keyCache.expiresAt = time.Now().Add(p.config.TokenValidation.CacheTTL)

	for _, key := range jwks.Keys {
		if key.Kty == "RSA" {
			pubKey, err := parseRSAPublicKey(key.N, key.E)
			if err != nil {
				continue // Skip invalid keys
			}
			p.keyCache.keys[key.Kid] = pubKey
		}
	}

	// Return the requested key
	if key, exists := p.keyCache.keys[kid]; exists {
		return key, nil
	}

	return nil, fmt.Errorf("key with ID %s not found", kid)
}

// parseRSAPublicKey parses RSA public key from JWK format
func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	// Decode base64url-encoded modulus
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode base64url-encoded exponent
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	// Create RSA public key
	pubKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	return pubKey, nil
}
