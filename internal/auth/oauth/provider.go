package oauth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
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

// ProtectedResourceMetadata represents MCP protected resource metadata (RFC 9728 compliant)
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

	// Azure AD v2.0 has limited support for RFC 8707 Resource Indicators
	// - Authorization endpoint: doesn't support resource parameter
	// - Token endpoint: doesn't support resource parameter
	// - Uses scope-based resource identification instead
	// Our proxy handles MCP resource parameter translation
	parsedURL, err := url.Parse(serverURL)
	if err == nil {
		// If the server URL includes /mcp path, include it in the proxy endpoint
		proxyPath := "/oauth2/v2.0/authorize"
		tokenPath := "/oauth2/v2.0/token"
		registrationPath := "/oauth/register"
		proxyAuthURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, proxyPath)
		tokenURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, tokenPath)
		registrationURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, registrationPath)

		metadata.AuthorizationEndpoint = proxyAuthURL
		metadata.TokenEndpoint = tokenURL
		// Add dynamic client registration endpoint
		metadata.RegistrationEndpoint = registrationURL
	}

	return &metadata, nil
}

// ValidateToken validates an OAuth access token
func (p *AzureOAuthProvider) ValidateToken(ctx context.Context, tokenString string) (*auth.TokenInfo, error) {
	// Check if token looks like a valid JWT (should have exactly 2 dots)
	dotCount := strings.Count(tokenString, ".")
	if dotCount != 2 {
		return nil, fmt.Errorf("invalid JWT token format: expected 3 parts separated by dots, got %d dots", dotCount)
	}

	// If JWT validation is disabled, return a minimal token info without full validation
	if !p.config.TokenValidation.ValidateJWT {
		fmt.Printf("JWT validation disabled, returning minimal token info\n")
		return &auth.TokenInfo{
			AccessToken: tokenString,
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(time.Hour), // Default 1 hour expiration
			Scope:       p.config.RequiredScopes,   // Use configured scopes
			Subject:     "unknown",                 // Cannot extract without parsing
			Audience:    []string{p.config.TokenValidation.ExpectedAudience},
			Issuer:      fmt.Sprintf("https://sts.windows.net/%s/", p.config.TenantID),
			Claims:      make(map[string]interface{}),
		}, nil
	}

	// Parse and validate JWT token
	fmt.Printf("Starting JWT token parsing and validation...\n")
	fmt.Println("xxxxxxxxxxxxxxxx")
	fmt.Println(tokenString)

	// STEP 1: First parse WITHOUT signature validation to check claims and expiration
	fmt.Printf("=== STEP 1: Parsing token WITHOUT signature validation ===\n")
	parserUnsafe := jwt.NewParser(jwt.WithoutClaimsValidation())
	tokenUnsafe, _, err := parserUnsafe.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		fmt.Printf("Failed to parse token structure: %v\n", err)
		return nil, fmt.Errorf("invalid token structure: %w", err)
	}

	// Check claims and expiration manually
	if claims, ok := tokenUnsafe.Claims.(jwt.MapClaims); ok {
		fmt.Printf("Token claims extracted successfully\n")

		// Check expiration
		if exp, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			fmt.Printf("Token exp claim: %v (timestamp: %.0f)\n", expTime, exp)
			fmt.Printf("Current time: %v\n", time.Now())
			fmt.Printf("Time until expiry: %v\n", time.Until(expTime))

			if time.Now().After(expTime) {
				fmt.Printf("*** TOKEN IS EXPIRED - This explains the signature validation failure! ***\n")
				return nil, fmt.Errorf("token expired at %v", expTime)
			} else {
				fmt.Printf("✓ Token is not expired\n")
			}
		} else {
			fmt.Printf("WARNING: No exp claim found in token\n")
		}

		// Check issuer
		if iss, ok := claims["iss"].(string); ok {
			fmt.Printf("Token issuer: %s\n", iss)
		}

		// Check key ID
		if tokenUnsafe.Header != nil {
			if kid, ok := tokenUnsafe.Header["kid"].(string); ok {
				fmt.Printf("Token key ID (kid): %s\n", kid)
			}
			if alg, ok := tokenUnsafe.Header["alg"].(string); ok {
				fmt.Printf("Token algorithm: %s\n", alg)
			}
		}
	}

	// STEP 2: Now try with signature validation
	fmt.Printf("=== STEP 2: Parsing token WITH signature validation ===\n")

	// Let's try a different approach - manually verify signature first
	fmt.Printf("=== MANUAL SIGNATURE VERIFICATION TEST ===\n")
	tokenParts := strings.Split(tokenString, ".")
	if len(tokenParts) == 3 {
		headerAndPayload := tokenParts[0] + "." + tokenParts[1]
		signature := tokenParts[2]

		fmt.Printf("Header+Payload length: %d\n", len(headerAndPayload))
		fmt.Printf("Signature (base64url): %s\n", signature)

		// Decode signature
		sigBytes, err := base64.RawURLEncoding.DecodeString(signature)
		if err != nil {
			fmt.Printf("Failed to decode signature: %v\n", err)
		} else {
			fmt.Printf("Signature decoded length: %d bytes\n", len(sigBytes))
			fmt.Printf("Signature first 10 bytes: %x\n", sigBytes[:10])

			// Try manual RSA verification
			fmt.Printf("=== TRYING MANUAL RSA VERIFICATION ===\n")

			// First, get the public key from JWKS (we'll use the same logic)
			if claims, ok := tokenUnsafe.Claims.(jwt.MapClaims); ok {
				if iss, ok := claims["iss"].(string); ok {
					if kid, ok := tokenUnsafe.Header["kid"].(string); ok {
						fmt.Printf("Getting public key for manual verification...\n")
						pubKey, keyErr := p.getPublicKey(kid, iss)
						if keyErr != nil {
							fmt.Printf("Failed to get public key for manual verification: %v\n", keyErr)
						} else {
							fmt.Printf("Got public key, attempting manual RSA verification...\n")

							// Create SHA256 hash of header+payload
							hasher := sha256.New()
							hasher.Write([]byte(headerAndPayload))
							hash := hasher.Sum(nil)

							fmt.Printf("SHA256 hash of header+payload: %x\n", hash)
							fmt.Printf("Hash length: %d bytes\n", len(hash))

							// Try to verify the signature manually
							verifyErr := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash, sigBytes)
							if verifyErr != nil {
								fmt.Printf("*** MANUAL RSA VERIFICATION FAILED: %v ***\n", verifyErr)
								fmt.Printf("This confirms the issue is with the RSA signature itself\n")

								// Let's try different padding schemes
								fmt.Printf("Trying PSS padding instead of PKCS1v15...\n")
								pssErr := rsa.VerifyPSS(pubKey, crypto.SHA256, hash, sigBytes, nil)
								if pssErr != nil {
									fmt.Printf("PSS verification also failed: %v\n", pssErr)
								} else {
									fmt.Printf("*** PSS VERIFICATION SUCCEEDED! ***\n")
									fmt.Printf("Azure AD might be using PSS padding instead of PKCS1v15!\n")
								}
							} else {
								fmt.Printf("*** MANUAL RSA VERIFICATION SUCCEEDED! ***\n")
								fmt.Printf("This suggests the JWT library has a bug or different expectations\n")
							}
						}
					}
				}
			}
		}
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(tokenString, jwt.MapClaims{}, p.getKeyFunc)
	if err != nil {
		fmt.Printf("JWT parsing failed with error: %v\n", err)
		fmt.Printf("Error type: %T\n", err)

		// Check if the error message contains signature-related keywords
		errStr := err.Error()
		if strings.Contains(errStr, "signature") {
			fmt.Printf("  *** SIGNATURE VALIDATION FAILED ***\n")
			fmt.Printf("  Since token is not expired, this indicates:\n")
			fmt.Printf("    1. Wrong public key being used\n")
			fmt.Printf("    2. Key parsing/decoding issue\n")
			fmt.Printf("    3. Algorithm mismatch\n")
			fmt.Printf("    4. Token corruption/modification\n")

			fmt.Printf("  Raw error details: %v\n", err)
			fmt.Printf("  Error type: %T\n", err)
		}
		if strings.Contains(errStr, "expired") {
			fmt.Printf("  *** TOKEN EXPIRED ***\n")
		}

		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Now manually validate expiration with better error handling
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			fmt.Printf("Token expiration time: %v\n", expTime)
			fmt.Printf("Current time: %v\n", time.Now())
			if time.Now().After(expTime) {
				fmt.Printf("  *** TOKEN IS EXPIRED ***\n")
				return nil, fmt.Errorf("token expired at %v", expTime)
			}
		}
	}

	fmt.Printf("JWT parsing completed successfully\n")
	fmt.Printf("Token valid: %t\n", token.Valid)

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

	// Additional validation: ensure token was issued for this specific MCP server
	// This implements RFC 8707 resource binding validation
	if err := p.validateResourceBinding(claims); err != nil {
		fmt.Printf("Resource binding validation warning: %v\n", err)
		// For now, log but don't fail - Azure AD may use different claim names
		// In production, you should enable this validation based on your Azure AD setup
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

	// Validate with Azure AD directly
	fmt.Printf("DEBUG: Validating token with Azure AD introspection endpoint...\n")
	if err := p.validateTokenWithAzureAD(tokenString); err != nil {
		fmt.Printf("DEBUG: Azure AD validation failed: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Azure AD validation successful - token is active\n")
	}

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

// validateResourceBinding validates that the token was issued for this specific MCP server
// This implements RFC 8707 resource binding validation
func (p *AzureOAuthProvider) validateResourceBinding(claims jwt.MapClaims) error {
	// Azure AD may include resource information in different claims
	// Check for common resource-related claims

	// Check for 'aud' claim that matches our expected resource
	if p.config.TokenValidation.ExpectedAudience != "" {
		expectedResource := strings.TrimSuffix(p.config.TokenValidation.ExpectedAudience, "/")

		// Check single audience
		if aud, ok := claims["aud"].(string); ok {
			normalizedAud := strings.TrimSuffix(aud, "/")
			if normalizedAud == expectedResource {
				return nil // Resource binding validated
			}
		}

		// Check audience array
		if audSlice, ok := claims["aud"].([]interface{}); ok {
			for _, a := range audSlice {
				if audStr, ok := a.(string); ok {
					normalizedAud := strings.TrimSuffix(audStr, "/")
					if normalizedAud == expectedResource {
						return nil // Resource binding validated
					}
				}
			}
		}
	}

	// If no specific resource validation is configured, accept the token
	// This maintains backward compatibility
	if p.config.TokenValidation.ExpectedAudience == "" {
		return nil
	}

	return fmt.Errorf("token was not issued for this MCP server resource: expected audience %s", p.config.TokenValidation.ExpectedAudience)
}

// getKeyFunc returns a function to retrieve JWT signing keys
func (p *AzureOAuthProvider) getKeyFunc(token *jwt.Token) (interface{}, error) {
	// Debug the signing method
	fmt.Printf("JWT signing method debug:\n")
	fmt.Printf("  Token method type: %T\n", token.Method)
	fmt.Printf("  Token method string: %s\n", token.Method.Alg())
	fmt.Printf("  Header alg: %v\n", token.Header["alg"])

	// Ensure the token uses RS256 specifically
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
	fmt.Printf("Token issuer: %s\n", issuer)
	fmt.Printf("Key ID (kid): %s\n", kid)

	// Get the public key for this key ID using the appropriate issuer
	key, err := p.getPublicKey(kid, issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	fmt.Printf("Returning RSA public key for signature verification (N bit length: %d)\n", key.N.BitLen())
	return key, nil
}

// getPublicKey retrieves and caches Azure AD public keys
func (p *AzureOAuthProvider) getPublicKey(kid string, issuer string) (*rsa.PublicKey, error) {
	// TODO: Re-enable caching after fixing JWT signature validation issues
	// Cache disabled for debugging JWT signature validation problems

	// Generate cache key based on both kid and issuer to avoid conflicts between v1.0 and v2.0 keys
	cacheKey := fmt.Sprintf("%s_%s", kid, issuer)

	// TODO: Re-enable cache checking logic
	// Cache logic commented out for debugging
	/*
		// Force cache miss for v1.0 tokens to ensure we get the right key
		if strings.Contains(issuer, "sts.windows.net") {
			fmt.Printf("v1.0 token detected, forcing cache refresh for key %s\n", cacheKey)
			p.keyCache.mu.Lock()
			if p.keyCache.keys != nil {
				// Clear ALL cached keys to force fresh fetch from correct endpoint
				p.keyCache.keys = make(map[string]*rsa.PublicKey)
				fmt.Printf("Cleared all cached keys for v1.0 token validation\n")
			}
			p.keyCache.mu.Unlock()
		} else {
			p.keyCache.mu.RLock()
			if key, exists := p.keyCache.keys[cacheKey]; exists && time.Now().Before(p.keyCache.expiresAt) {
				p.keyCache.mu.RUnlock()
				fmt.Printf("Using cached key for %s\n", cacheKey)
				return key, nil
			}
			p.keyCache.mu.RUnlock()
		}
	*/

	fmt.Printf("Cache disabled - fetching fresh key %s from JWKS\n", cacheKey)

	// Determine the correct JWKS URL based on issuer
	var jwksURL string
	if strings.Contains(issuer, "sts.windows.net") {
		// v1.0 endpoint
		jwksURL = fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/keys", p.config.TenantID)
		fmt.Printf("Using v1.0 JWKS endpoint: %s\n", jwksURL)
	} else {
		// v2.0 endpoint (default)
		jwksURL = fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", p.config.TenantID)
		fmt.Printf("Using v2.0 JWKS endpoint: %s\n", jwksURL)
	}

	resp, err := p.httpClient.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", jwksURL, err)
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

	fmt.Printf("JWKS response debug:\n")
	fmt.Printf("  Response status: %d\n", resp.StatusCode)
	fmt.Printf("  Response body length: %d bytes\n", len(body))
	fmt.Printf("  Found %d keys in JWKS\n", len(jwks.Keys))
	for i, key := range jwks.Keys {
		fmt.Printf("  Key %d: kid=%s, kty=%s, n_length=%d, e_length=%d\n",
			i+1, key.Kid, key.Kty, len(key.N), len(key.E))
	}

	// TODO: Re-enable cache update logic after fixing JWT signature validation
	// Cache update logic commented out for debugging
	/*
		// Update cache
		p.keyCache.mu.Lock()
		defer p.keyCache.mu.Unlock()

		// Don't clear all keys, just add new ones with cache key
		if p.keyCache.keys == nil {
			p.keyCache.keys = make(map[string]*rsa.PublicKey)
		}
		p.keyCache.expiresAt = time.Now().Add(p.config.TokenValidation.CacheTTL)
	*/

	fmt.Printf("Parsing JWKS response, found %d keys (cache disabled)\n", len(jwks.Keys))

	// Parse keys without caching for debugging
	var targetKey *rsa.PublicKey
	for _, key := range jwks.Keys {
		if key.Kty == "RSA" {
			fmt.Printf("Parsing RSA key %s...\n", key.Kid)
			pubKey, err := parseRSAPublicKey(key.N, key.E)
			if err != nil {
				fmt.Printf("Failed to parse key %s: %v\n", key.Kid, err)
				continue // Skip invalid keys
			}

			// TODO: Re-enable caching when cache is fixed
			// keyCache := fmt.Sprintf("%s_%s", key.Kid, issuer)
			// p.keyCache.keys[keyCache] = pubKey
			fmt.Printf("Successfully parsed key: %s (RSA modulus bits: %d) - not cached\n", key.Kid, pubKey.N.BitLen())

			// Special debug for the target key
			if key.Kid == kid {
				fmt.Printf("*** Found target key %s! ***\n", kid)
				nPreview := key.N
				if len(nPreview) > 50 {
					nPreview = nPreview[:50] + "..."
				}
				fmt.Printf("  RSA N (preview): %s\n", nPreview)
				fmt.Printf("  RSA E: %s\n", key.E)
				fmt.Printf("  Parsed N bit length: %d\n", pubKey.N.BitLen())
				fmt.Printf("  Parsed E value: %d\n", pubKey.E)

				// Debug: Let's test if our key parsing is working correctly
				// by trying to verify the JWT signature manually
				fmt.Printf("=== DEBUGGING RSA KEY CONSTRUCTION ===\n")
				fmt.Printf("  Raw N (base64url): %s\n", key.N)
				fmt.Printf("  Raw E (base64url): %s\n", key.E)

				// Test base64url decoding
				nBytes, nErr := base64.RawURLEncoding.DecodeString(key.N)
				eBytes, eErr := base64.RawURLEncoding.DecodeString(key.E)

				if nErr != nil {
					fmt.Printf("  ERROR: Failed to decode N: %v\n", nErr)
				} else {
					fmt.Printf("  N decoded length: %d bytes\n", len(nBytes))
					fmt.Printf("  N first 10 bytes: %x\n", nBytes[:10])
				}

				if eErr != nil {
					fmt.Printf("  ERROR: Failed to decode E: %v\n", eErr)
				} else {
					fmt.Printf("  E decoded length: %d bytes\n", len(eBytes))
					fmt.Printf("  E bytes: %x\n", eBytes)
					// E should typically be 65537 (0x010001)
					if len(eBytes) == 3 && eBytes[0] == 0x01 && eBytes[1] == 0x00 && eBytes[2] == 0x01 {
						fmt.Printf("  ✓ E value looks correct (65537)\n")
					} else {
						fmt.Printf("  ⚠ E value is unexpected\n")
					}
				}

				targetKey = pubKey
			}
		}
	}

	// Return the requested key without caching
	if targetKey != nil {
		fmt.Printf("Found requested key %s in fresh JWKS response (not cached)\n", kid)
		return targetKey, nil
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

// validateTokenWithAzureAD validates token directly with Azure AD token introspection endpoint
func (p *AzureOAuthProvider) validateTokenWithAzureAD(token string) error {
	// Use Azure AD token introspection endpoint to validate the token
	introspectURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/introspect", p.config.TenantID)
	
	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", p.config.ClientID)
	
	resp, err := p.httpClient.PostForm(introspectURL, data)
	if err != nil {
		return fmt.Errorf("introspection request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read introspection response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("introspection failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var introspectionResult struct {
		Active bool `json:"active"`
	}
	
	if err := json.Unmarshal(body, &introspectionResult); err != nil {
		return fmt.Errorf("failed to parse introspection response: %w", err)
	}
	
	if !introspectionResult.Active {
		return fmt.Errorf("token is not active according to Azure AD")
	}
	
	return nil
}
