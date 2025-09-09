package oauth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
	internalConfig "github.com/Azure/aks-mcp/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// AzureOAuthProvider implements OAuth authentication for Azure AD
type AzureOAuthProvider struct {
	config      *auth.OAuthConfig
	httpClient  *http.Client
	keyCache    *keyCache
	enableCache bool
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

// ProtectedResourceMetadata represents MCP protected resource metadata (RFC 9728 compliant)
type ProtectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
	Resource             string   `json:"resource"`
	ScopesSupported      []string `json:"scopes_supported"`
}

// ClientRegistrationRequest represents OAuth 2.0 Dynamic Client Registration request (RFC 7591)
type ClientRegistrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	ClientName              string   `json:"client_name"`
	ClientURI               string   `json:"client_uri"`
	Scope                   string   `json:"scope"`
}

// NewAzureOAuthProvider creates a new Azure OAuth provider
func NewAzureOAuthProvider(config *auth.OAuthConfig) (*AzureOAuthProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OAuth config: %w", err)
	}

	return &AzureOAuthProvider{
		config:      config,
		enableCache: internalConfig.EnableCache, // Use config constant for cache control
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		keyCache: &keyCache{
			keys: make(map[string]*rsa.PublicKey),
		},
	}, nil
}

// GetProtectedResourceMetadata returns OAuth 2.0 Protected Resource Metadata (RFC 9728)
func (p *AzureOAuthProvider) GetProtectedResourceMetadata(serverURL string) (*ProtectedResourceMetadata, error) {
	// For MCP compliance, point to our local authorization server proxy
	// which properly advertises PKCE support
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	// Use the same scheme and host as the server URL
	authServerURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// RFC 9728 requires the resource field to identify this MCP server
	return &ProtectedResourceMetadata{
		AuthorizationServers: []string{authServerURL},
		Resource:             serverURL, // Required by MCP spec
		ScopesSupported:      p.config.RequiredScopes,
	}, nil
}

// GetAuthorizationServerMetadata returns OAuth 2.0 Authorization Server Metadata (RFC 8414)
func (p *AzureOAuthProvider) GetAuthorizationServerMetadata(serverURL string) (*AzureADMetadata, error) {
	metadataURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0/.well-known/openid-configuration", p.config.TenantID)
	log.Printf("OAuth DEBUG: Fetching Azure AD metadata from: %s", metadataURL)

	resp, err := p.httpClient.Get(metadataURL)
	if err != nil {
		log.Printf("OAuth ERROR: Failed to fetch metadata from %s: %v", metadataURL, err)
		return nil, fmt.Errorf("failed to fetch metadata from %s: %w", metadataURL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("OAuth ERROR: Tenant ID '%s' not found (HTTP 404)", p.config.TenantID)
		return nil, fmt.Errorf("tenant ID '%s' not found (HTTP 404). Please verify your Azure AD tenant ID is correct", p.config.TenantID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("OAuth ERROR: Metadata endpoint returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("metadata endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("OAuth ERROR: Failed to read metadata response: %v", err)
		return nil, fmt.Errorf("failed to read metadata response: %w", err)
	}

	var metadata AzureADMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		log.Printf("OAuth ERROR: Failed to parse metadata JSON: %v", err)
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	log.Printf("OAuth DEBUG: Successfully parsed Azure AD metadata, original grant_types_supported: %v", metadata.GrantTypesSupported)

	// Ensure grant_types_supported is populated for MCP Inspector compatibility
	if len(metadata.GrantTypesSupported) == 0 {
		log.Printf("OAuth DEBUG: Setting default grant_types_supported (was empty/nil)")
		metadata.GrantTypesSupported = []string{"authorization_code", "refresh_token"}
	}

	// Ensure response_types_supported is populated for MCP Inspector compatibility
	if len(metadata.ResponseTypesSupported) == 0 {
		log.Printf("OAuth DEBUG: Setting default response_types_supported (was empty/nil)")
		metadata.ResponseTypesSupported = []string{"code"}
	}

	// Ensure subject_types_supported is populated for MCP Inspector compatibility
	if len(metadata.SubjectTypesSupported) == 0 {
		log.Printf("OAuth DEBUG: Setting default subject_types_supported (was empty/nil)")
		metadata.SubjectTypesSupported = []string{"public"}
	}

	// Ensure token_endpoint_auth_methods_supported is populated for MCP Inspector compatibility
	if len(metadata.TokenEndpointAuthMethodsSupported) == 0 {
		log.Printf("OAuth DEBUG: Setting default token_endpoint_auth_methods_supported (was empty/nil)")
		metadata.TokenEndpointAuthMethodsSupported = []string{"none"}
	}

	// Add S256 code challenge method support (Azure AD supports this but may not advertise it)
	// MCP specification requires S256 support, so we always ensure it's present
	log.Printf("OAuth DEBUG: Enforcing S256 code challenge method support (MCP requirement)")
	metadata.CodeChallengeMethodsSupported = []string{"S256"}

	// Azure AD v2.0 has limited support for RFC 8707 Resource Indicators
	// - Authorization endpoint: doesn't support resource parameter
	// - Token endpoint: doesn't support resource parameter
	// - Uses scope-based resource identification instead
	// Our proxy handles MCP resource parameter translation
	parsedURL, err := url.Parse(serverURL)
	if err == nil {
		// If the server URL includes /mcp path, include it in the proxy endpoint
		proxyPath := "/oauth2/v2.0/authorize"
		tokenPath := "/oauth2/v2.0/token" // #nosec G101 -- This is an OAuth endpoint path, not credentials
		registrationPath := "/oauth/register"
		proxyAuthURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, proxyPath)
		tokenURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, tokenPath)
		registrationURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, registrationPath)

		metadata.AuthorizationEndpoint = proxyAuthURL
		metadata.TokenEndpoint = tokenURL
		// Add dynamic client registration endpoint
		metadata.RegistrationEndpoint = registrationURL
	}

	log.Printf("OAuth DEBUG: Final metadata prepared - grant_types_supported: %v, response_types_supported: %v, code_challenge_methods_supported: %v",
		metadata.GrantTypesSupported, metadata.ResponseTypesSupported, metadata.CodeChallengeMethodsSupported)

	return &metadata, nil
}

// ValidateToken validates an OAuth access token
func (p *AzureOAuthProvider) ValidateToken(ctx context.Context, tokenString string) (*auth.TokenInfo, error) {
	// JWTs have three parts (header.payload.signature) separated by two dots.
	const jwtExpectedDotCount = 2

	dotCount := strings.Count(tokenString, ".")
	if dotCount != jwtExpectedDotCount {
		return nil, fmt.Errorf("invalid JWT token format: expected 3 parts separated by dots, got %d dots", dotCount)
	}

	// SECURITY WARNING: JWT validation bypass - for development and testing ONLY
	// ValidateJWT should ALWAYS be true in production environments
	// This bypass creates a significant security vulnerability if enabled in production
	if !p.config.TokenValidation.ValidateJWT {
		log.Printf("WARNING: JWT validation is DISABLED - this should ONLY be used in development/testing")
		return &auth.TokenInfo{
			AccessToken: tokenString,
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(time.Hour), // Default 1 hour expiration
			Scope:       p.config.RequiredScopes,   // Use configured scopes
			Subject:     "unknown",                 // Cannot extract without parsing
			Audience:    []string{p.config.TokenValidation.ExpectedAudience},
			Issuer:      fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", p.config.TenantID),
			Claims:      make(map[string]interface{}),
		}, nil
	}

	// Parse and validate JWT token

	// Parse token structure and check expiration
	parserUnsafe := jwt.NewParser(jwt.WithoutClaimsValidation())
	tokenUnsafe, _, err := parserUnsafe.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("invalid token structure: %w", err)
	}

	// Check claims and expiration
	if claims, ok := tokenUnsafe.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			if time.Now().After(expTime) {
				return nil, fmt.Errorf("token expired at %v", expTime)
			}
		}
	}

	// JWT signature validation
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(tokenString, jwt.MapClaims{}, p.getKeyFunc)
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

	// Validate issuer - with Azure Management API scope, we should get v2.0 format
	issuer, ok := claims["iss"].(string)
	if !ok {
		return nil, fmt.Errorf("missing issuer claim")
	}

	expectedIssuerV2 := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", p.config.TenantID)
	expectedIssuerV1 := fmt.Sprintf("https://sts.windows.net/%s/", p.config.TenantID)

	if issuer != expectedIssuerV2 && issuer != expectedIssuerV1 {
		return nil, fmt.Errorf("invalid issuer: expected %s (preferred) or %s (fallback), got %s", expectedIssuerV2, expectedIssuerV1, issuer)
	}

	// Azure AD may return v1.0 or v2.0 issuer format depending on token scope

	// Validate audience and resource binding
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

	// Extract scope from Azure AD token
	// Check for 'scp' claim (Azure AD v2.0)
	if scp, ok := claims["scp"].(string); ok {
		tokenInfo.Scope = strings.Split(scp, " ")
	} else if scope, ok := claims["scope"].(string); ok {
		// Check for 'scope' claim (alternative)
		tokenInfo.Scope = strings.Split(scope, " ")
	}

	// Check for 'roles' claim (Azure AD app roles)
	if roles, ok := claims["roles"].([]interface{}); ok {
		for _, role := range roles {
			if roleStr, ok := role.(string); ok {
				tokenInfo.Scope = append(tokenInfo.Scope, roleStr)
			}
		}
	}

	// Extract expiration
	if exp, ok := claims["exp"].(float64); ok {
		tokenInfo.ExpiresAt = time.Unix(int64(exp), 0)
	}

	// Set issuer
	tokenInfo.Issuer = issuer

	return tokenInfo, nil
}

// validateAudience validates the audience claim and resource binding (RFC 8707)
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
	// Validate signing method
	if token.Method.Alg() != "RS256" {
		return nil, fmt.Errorf("unexpected signing method: expected RS256, got %v", token.Method.Alg())
	}

	// Also verify it's an RSA method
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("signing method is not RSA: %T", token.Method)
	}

	// Get key ID from token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("missing key ID in token header")
	}

	// Extract issuer from token to determine the correct JWKS endpoint
	var issuer string
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if iss, ok := claims["iss"].(string); ok {
			issuer = iss
		}
	}

	// Get the public key for this key ID using the appropriate issuer
	key, err := p.getPublicKey(kid, issuer)
	if err != nil {
		log.Printf("PUBLIC KEY RETRIEVAL FAILED: %s\n", err)
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	return key, nil
}

// getPublicKey retrieves and caches Azure AD public keys
func (p *AzureOAuthProvider) getPublicKey(kid string, issuer string) (*rsa.PublicKey, error) {
	// Generate cache key based on both kid and issuer to avoid conflicts between v1.0 and v2.0 keys
	cacheKey := fmt.Sprintf("%s_%s", kid, issuer)

	// Check cache first if caching is enabled
	if p.enableCache {
		p.keyCache.mu.RLock()
		if key, exists := p.keyCache.keys[cacheKey]; exists && time.Now().Before(p.keyCache.expiresAt) {
			p.keyCache.mu.RUnlock()
			return key, nil
		}
		p.keyCache.mu.RUnlock()
	}

	// With Azure Management API scope, we should always get v2.0 format tokens
	// Force using v2.0 JWKS endpoint for consistency
	jwksURL := fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", p.config.TenantID)

	resp, err := p.httpClient.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", jwksURL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

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

	log.Printf("JWKS Contains %d keys, searching for kid=%s\n", len(jwks.Keys), kid)

	// Parse keys and find the target key
	var targetKey *rsa.PublicKey
	var foundKeyIds []string

	for _, key := range jwks.Keys {
		foundKeyIds = append(foundKeyIds, key.Kid)

		if key.Kty == "RSA" && key.Kid == kid {
			pubKey, err := parseRSAPublicKey(key.N, key.E)
			if err != nil {
				log.Printf("JWKS Failed to parse RSA key %s: %v\n", key.Kid, err)
				continue
			}
			targetKey = pubKey
			break
		}
	}

	// Cache the retrieved key and return it (only if caching is enabled)
	if targetKey != nil {
		if p.enableCache {
			p.keyCache.mu.Lock()
			if p.keyCache.keys == nil {
				p.keyCache.keys = make(map[string]*rsa.PublicKey)
			}
			p.keyCache.keys[cacheKey] = targetKey
			p.keyCache.expiresAt = time.Now().Add(24 * time.Hour) // Cache for 24 hours
			p.keyCache.mu.Unlock()
		}
		return targetKey, nil
	}

	return nil, fmt.Errorf("key with ID %s not found in JWKS (available: %v)", kid, foundKeyIds)
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
