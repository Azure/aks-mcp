package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"

	"github.com/Azure/aks-mcp/internal/config"
)

// EntraValidator validates Microsoft Entra ID tokens
type EntraValidator struct {
	clientID   string
	tenantID   string
	jwksCache  *jwk.Cache
	issuer     string
	jwksURL    string
	httpClient *http.Client
}

// Claims represents the JWT claims structure for Entra ID tokens
type Claims struct {
	jwt.RegisteredClaims
	TenantID          string   `json:"tid"`
	Scope             string   `json:"scp"`
	Roles             []string `json:"roles"`
	AppID             string   `json:"appid"`
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Email             string   `json:"email"`
	ObjectID          string   `json:"oid"`
	UPN               string   `json:"upn"`
}

// NewEntraValidator creates a new Entra ID token validator
func NewEntraValidator(config *config.AuthConfig) (*EntraValidator, error) {
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid auth config: %w", err)
	}

	jwksURL := config.GetJWKSURL()
	issuer := config.GetIssuer()

	// Create JWKS cache with auto-refresh
	cache := jwk.NewCache(context.Background())

	// Register the JWKS URL with refresh interval
	refreshInterval := time.Duration(config.JWKSCacheTimeout/2) * time.Second
	if refreshInterval < 15*time.Minute {
		refreshInterval = 15 * time.Minute // Minimum refresh interval
	}

	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(refreshInterval)); err != nil {
		return nil, fmt.Errorf("failed to register JWKS cache: %w", err)
	}

	return &EntraValidator{
		clientID:   config.EntraClientID,
		tenantID:   config.EntraTenantID,
		jwksCache:  cache,
		issuer:     issuer,
		jwksURL:    jwksURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// ValidateToken validates an Entra ID JWT token and returns user context
func (v *EntraValidator) ValidateToken(ctx context.Context, tokenString string) (*UserContext, error) {
	// Parse and validate the JWT token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Get the key set from cache
		keySet, err := v.jwksCache.Get(ctx, v.jwksURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get JWKS: %w", err)
		}

		// Find the key with the matching kid
		key, found := keySet.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key %s not found in JWKS", kid)
		}

		// Extract the raw key
		var rawKey interface{}
		if err := key.Raw(&rawKey); err != nil {
			return nil, fmt.Errorf("failed to get raw key: %w", err)
		}

		return rawKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate basic claims
	if err := v.validateBasicClaims(claims); err != nil {
		return nil, err
	}

	// Extract user scopes (no specific scope validation)
	scopes := strings.Fields(claims.Scope)

	// Create user context with simplified permissions
	userContext := &UserContext{
		UserID:   v.getUserID(claims),
		Email:    v.getEmail(claims),
		Name:     claims.Name,
		TenantID: claims.TenantID,
		Scopes:   scopes,
		IsAdmin:  true, // Simplified: all authenticated users have access
	}

	return userContext, nil
}

// validateBasicClaims validates the basic JWT claims
func (v *EntraValidator) validateBasicClaims(claims *Claims) error {
	// Validate tenant ID
	if claims.TenantID != v.tenantID {
		return fmt.Errorf("invalid tenant ID: expected %s, got %s", v.tenantID, claims.TenantID)
	}

	// Validate audience
	if len(claims.Audience) == 0 {
		return fmt.Errorf("missing audience claim")
	}

	// Determine expected audience based on application configuration
	// Check if this is a custom API (with identifier URI) or minimal setup
	expectedAudiences := []string{
		v.clientID,                          // Minimal setup: clientID directly
		fmt.Sprintf("api://%s", v.clientID), // Custom API setup: api://clientID
	}

	validAudience := false
	var tokenAudience string
	for _, aud := range claims.Audience {
		tokenAudience = aud
		for _, expectedAud := range expectedAudiences {
			if aud == expectedAud {
				validAudience = true
				break
			}
		}
		if validAudience {
			break
		}
	}
	if !validAudience {
		return fmt.Errorf("invalid audience: expected %v, got %s", expectedAudiences, tokenAudience)
	}

	// Validate issuer - accept both v1.0 and v2.0 endpoints
	expectedIssuers := []string{
		v.issuer, // v2.0 endpoint
		fmt.Sprintf("https://sts.windows.net/%s/", v.tenantID), // v1.0 endpoint
	}

	validIssuer := false
	for _, expectedIssuer := range expectedIssuers {
		if claims.Issuer == expectedIssuer {
			validIssuer = true
			break
		}
	}
	if !validIssuer {
		return fmt.Errorf("invalid issuer: expected one of %v, got %s", expectedIssuers, claims.Issuer)
	}

	return nil
}

// getUserID extracts user ID from claims (prefers oid, falls back to sub)
func (v *EntraValidator) getUserID(claims *Claims) string {
	if claims.ObjectID != "" {
		return claims.ObjectID
	}
	return claims.Subject
}

// getEmail extracts email from claims (prefers email, falls back to upn)
func (v *EntraValidator) getEmail(claims *Claims) string {
	if claims.Email != "" {
		return claims.Email
	}
	if claims.UPN != "" {
		return claims.UPN
	}
	return claims.PreferredUsername
}
