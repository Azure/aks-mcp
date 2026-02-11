package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/aks-mcp/internal/auth"
	"github.com/Azure/aks-mcp/internal/logger"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const tokenInfoKey contextKey = "token_info"

// AuthMiddleware handles OAuth authentication for HTTP requests
type AuthMiddleware struct {
	provider  *AzureOAuthProvider
	serverURL string
}

// setCORSHeaders sets CORS headers for OAuth endpoints with origin whitelisting
func (m *AuthMiddleware) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	requestOrigin := r.Header.Get("Origin")

	// Check if the request origin is in the allowed list
	var allowedOrigin string
	for _, allowed := range m.provider.config.AllowedOrigins {
		if requestOrigin == allowed {
			allowedOrigin = requestOrigin
			break
		}
	}

	// Only set CORS headers if origin is allowed
	if allowedOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, mcp-protocol-version")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		w.Header().Set("Access-Control-Allow-Credentials", "false")
	} else if requestOrigin != "" {
		logger.Errorf("CORS ERROR: Origin %s is not in the allowed list - cross-origin requests will be blocked for security", requestOrigin)
	}
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(provider *AzureOAuthProvider, serverURL string) *AuthMiddleware {
	return &AuthMiddleware{
		provider:  provider,
		serverURL: serverURL,
	}
}

// Middleware returns an HTTP middleware function for OAuth authentication
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Skip authentication for specific endpoints
		if m.shouldSkipAuth(r) {
			logger.Debugf("Skipping auth for path: %s", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		// Perform authentication
		authResult := m.authenticateRequest(r)

		if !authResult.Authenticated {
			logger.Errorf("Authentication FAILED - handling error")
			m.handleAuthError(w, r, authResult)
			return
		}

		// Add token info to request context
		ctx := context.WithValue(r.Context(), tokenInfoKey, authResult.TokenInfo)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// shouldSkipAuth determines if authentication should be skipped for this request
func (m *AuthMiddleware) shouldSkipAuth(r *http.Request) bool {
	// Skip auth for OAuth metadata endpoints
	path := r.URL.Path

	skipPaths := []string{
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-authorization-server",
		"/.well-known/openid-configuration",
		"/oauth2/v2.0/authorize",
		"/oauth/register",
		"/oauth/callback",
		"/oauth2/v2.0/token",
		"/oauth/introspect",
		"/health",
		"/ping",
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}

	return false
}

// authenticateRequest performs OAuth authentication on the request
func (m *AuthMiddleware) authenticateRequest(r *http.Request) *auth.AuthResult {
	// Extract Bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		logger.Debugf("OAuth DEBUG - Missing authorization header for %s %s", r.Method, r.URL.Path)
		logger.Debugf("OAuth DEBUG - Request headers: %+v", r.Header)
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "missing authorization header",
			StatusCode:    http.StatusUnauthorized,
		}
	}

	// Check for Bearer token format
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		logger.Errorf("FAILED - Invalid authorization header format (missing Bearer prefix)")
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "invalid authorization header format",
			StatusCode:    http.StatusUnauthorized,
		}
	}

	token := strings.TrimPrefix(authHeader, bearerPrefix)
	if token == "" {
		logger.Errorf("FAILED - Empty bearer token")
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "empty bearer token",
			StatusCode:    http.StatusUnauthorized,
		}
	}

	// Basic JWT structure validation
	tokenParts := strings.Split(token, ".")
	if len(tokenParts) != 3 {
		logger.Errorf("FAILED - JWT structure validation (has %d parts, expected 3)", len(tokenParts))
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "invalid JWT structure",
			StatusCode:    http.StatusUnauthorized,
		}
	}

	// Validate the token
	tokenInfo, err := m.provider.ValidateToken(r.Context(), token)
	if err != nil {
		logger.Errorf("FAILED - Provider token validation failed: %v", err)
		return &auth.AuthResult{
			Authenticated: false,
			Error:         fmt.Sprintf("token validation failed: %v", err),
			StatusCode:    http.StatusUnauthorized,
		}
	}

	// Validate required scopes - supports both user tokens (scp) and MI/SPN tokens (roles or audience-based)
	if !m.validateScopesWithAudience(tokenInfo) {
		logger.Errorf("SCOPE ERROR: Token scopes %v don't match required scopes %v (audience: %v)", tokenInfo.Scope, m.provider.config.RequiredScopes, tokenInfo.Audience)
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "insufficient scope",
			StatusCode:    http.StatusForbidden,
		}
	}

	return &auth.AuthResult{
		Authenticated: true,
		TokenInfo:     tokenInfo,
		StatusCode:    http.StatusOK,
	}
}

// validateScopes checks if the token has required scopes
func (m *AuthMiddleware) validateScopes(tokenScopes []string) bool {
	requiredScopes := m.provider.config.RequiredScopes
	if len(requiredScopes) == 0 {
		return true // No scopes required
	}

	// Check if token has at least one required scope
	for _, required := range requiredScopes {
		if m.hasScopePermission(required, tokenScopes) {
			return true
		}
	}

	return false
}

// validateScopesWithAudience checks scopes considering the token's audience
// This handles Managed Identity tokens that may not have scp/roles but have correct audience
func (m *AuthMiddleware) validateScopesWithAudience(tokenInfo *auth.TokenInfo) bool {
	requiredScopes := m.provider.config.RequiredScopes
	if len(requiredScopes) == 0 {
		return true // No scopes required
	}

	// First try standard scope validation
	if m.validateScopes(tokenInfo.Scope) {
		return true
	}

	// For Managed Identity / Service Principal tokens:
	// If the audience matches the required resource and token is valid, allow access
	// MI tokens for https://management.azure.com often have empty scp/roles
	for _, required := range requiredScopes {
		// Extract resource from scope (e.g., "https://management.azure.com" from "https://management.azure.com/.default")
		resource := strings.TrimSuffix(required, "/.default")
		resource = strings.TrimSuffix(resource, "/")

		for _, aud := range tokenInfo.Audience {
			normalizedAud := strings.TrimSuffix(aud, "/")
			if normalizedAud == resource {
				logger.Debugf("Scope validation: accepting token with matching audience %s for resource %s (MI/SPN token)", aud, resource)
				return true
			}
		}
	}

	return false
}

// hasScopePermission checks if the token scopes satisfy the required scope
func (m *AuthMiddleware) hasScopePermission(requiredScope string, tokenScopes []string) bool {
	// Direct scope match
	for _, tokenScope := range tokenScopes {
		if tokenScope == requiredScope {
			return true
		}
	}

	// Azure resource scope mapping
	// Maps required scopes to acceptable token claims (both 'scp' and 'roles')
	azureResourceMappings := map[string][]string{
		"https://management.azure.com/.default": {
			// User delegated scopes (scp claim)
			"user_impersonation",
			"https://management.azure.com/user_impersonation",
			"https://management.azure.com/.default",
			"https://management.core.windows.net/",
			"https://management.azure.com/",
			// Application roles for Service Principals / Managed Identities (roles claim)
			// When an MI has RBAC roles on Azure, the token includes these
			"Reader",
			"Contributor",
			"Owner",
			// Azure AD may also return the resource URL as a role
			"https://management.azure.com",
		},
		"https://graph.microsoft.com/.default": {
			"User.Read",
			"https://graph.microsoft.com/User.Read",
		},
	}

	if allowedScopes, exists := azureResourceMappings[requiredScope]; exists {
		for _, allowedScope := range allowedScopes {
			for _, tokenScope := range tokenScopes {
				if tokenScope == allowedScope {
					return true
				}
			}
		}
	}

	return false
}

// handleAuthError handles authentication errors
func (m *AuthMiddleware) handleAuthError(w http.ResponseWriter, r *http.Request, authResult *auth.AuthResult) {
	// Set CORS headers
	m.setCORSHeaders(w, r)
	w.Header().Set("Content-Type", "application/json")

	// Add WWW-Authenticate header for 401 responses (RFC 9728 Section 5.1)
	if authResult.StatusCode == http.StatusUnauthorized {
		// Build the resource metadata URL
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		host := r.Host
		if host == "" {
			host = r.URL.Host
		}
		serverURL := fmt.Sprintf("%s://%s", scheme, host)
		resourceMetadataURL := fmt.Sprintf("%s/.well-known/oauth-protected-resource", serverURL)

		// RFC 9728 compliant WWW-Authenticate header
		wwwAuth := fmt.Sprintf(`Bearer realm="%s", resource_metadata="%s"`, serverURL, resourceMetadataURL)

		// Add error information if available
		if authResult.Error != "" {
			wwwAuth += fmt.Sprintf(`, error="invalid_token", error_description="%s"`, authResult.Error)
		}

		w.Header().Set("WWW-Authenticate", wwwAuth)
	}

	w.WriteHeader(authResult.StatusCode)

	errorResponse := map[string]interface{}{
		"error":             getOAuthErrorCode(authResult.StatusCode),
		"error_description": authResult.Error,
	}

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		logger.Errorf("MIDDLEWARE ERROR: Failed to encode error response: %v", err)
	} else {
		logger.Errorf("MIDDLEWARE ERROR: Error response sent")
	}
}

// getOAuthErrorCode returns appropriate OAuth error code for HTTP status
func getOAuthErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusUnauthorized:
		return "invalid_token"
	case http.StatusForbidden:
		return "insufficient_scope"
	case http.StatusBadRequest:
		return "invalid_request"
	default:
		return "server_error"
	}
}
