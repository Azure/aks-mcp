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
	JWKSUri                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// ProtectedResourceMetadata represents MCP protected resource metadata
type ProtectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
	Resource             string   `json:"resource"`
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
func (p *AzureOAuthProvider) GetProtectedResourceMetadata(serverURL string) (*ProtectedResourceMetadata, error) {
	authServerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", p.config.TenantID)

	return &ProtectedResourceMetadata{
		AuthorizationServers: []string{authServerURL},
		Resource:             serverURL,
		ScopesSupported:      p.config.RequiredScopes,
	}, nil
}

// GetAuthorizationServerMetadata returns OAuth 2.0 Authorization Server Metadata (RFC 8414)
func (p *AzureOAuthProvider) GetAuthorizationServerMetadata() (*AzureADMetadata, error) {
	metadataURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0/.well-known/openid_configuration", p.config.TenantID)

	resp, err := p.httpClient.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata response: %w", err)
	}

	var metadata AzureADMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// ValidateToken validates an OAuth access token
func (p *AzureOAuthProvider) ValidateToken(ctx context.Context, tokenString string) (*auth.TokenInfo, error) {
	if !p.config.TokenValidation.ValidateJWT {
		// Simple bearer token validation without JWT parsing
		return &auth.TokenInfo{
			AccessToken: tokenString,
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(time.Hour), // Default expiration
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

	// Validate issuer
	issuer, ok := claims["iss"].(string)
	if !ok {
		return nil, fmt.Errorf("missing issuer claim")
	}

	expectedIssuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", p.config.TenantID)
	if issuer != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", expectedIssuer, issuer)
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
	if scp, ok := claims["scp"].(string); ok {
		tokenInfo.Scope = strings.Split(scp, " ")
	}

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

	// Check single audience
	if aud, ok := claims["aud"].(string); ok {
		if aud == expectedAudience || aud == p.config.ClientID {
			return nil
		}
		return fmt.Errorf("invalid audience: expected %s or %s, got %s", expectedAudience, p.config.ClientID, aud)
	}

	// Check audience array
	if audSlice, ok := claims["aud"].([]interface{}); ok {
		for _, a := range audSlice {
			if audStr, ok := a.(string); ok {
				if audStr == expectedAudience || audStr == p.config.ClientID {
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
