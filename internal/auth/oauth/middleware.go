package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/aks-mcp/internal/auth"
)

// AuthMiddleware handles OAuth authentication for HTTP requests
type AuthMiddleware struct {
	provider  *AzureOAuthProvider
	serverURL string
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
			next.ServeHTTP(w, r)
			return
		}

		// Perform authentication
		authResult := m.authenticateRequest(r)

		if !authResult.Authenticated {
			m.handleAuthError(w, r, authResult)
			return
		}

		// Add token info to request context
		ctx := context.WithValue(r.Context(), "token_info", authResult.TokenInfo)
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
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "missing authorization header",
			StatusCode:    http.StatusUnauthorized,
		}
	}

	// Check for Bearer token format
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "invalid authorization header format",
			StatusCode:    http.StatusUnauthorized,
		}
	}

	token := strings.TrimPrefix(authHeader, bearerPrefix)
	if token == "" {
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "empty bearer token",
			StatusCode:    http.StatusUnauthorized,
		}
	}

	// Validate the token
	tokenInfo, err := m.provider.ValidateToken(r.Context(), token)
	if err != nil {
		return &auth.AuthResult{
			Authenticated: false,
			Error:         fmt.Sprintf("token validation failed: %v", err),
			StatusCode:    http.StatusUnauthorized,
		}
	}

	// Validate required scopes
	if !m.validateScopes(tokenInfo.Scope) {
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "insufficient scopes",
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
		for _, tokenScope := range tokenScopes {
			if tokenScope == required {
				return true
			}
		}
	}

	return false
}

// handleAuthError handles authentication errors
func (m *AuthMiddleware) handleAuthError(w http.ResponseWriter, r *http.Request, authResult *auth.AuthResult) {
	w.Header().Set("Content-Type", "application/json")

	// Add WWW-Authenticate header for 401 responses (RFC 9728 Section 5.1)
	if authResult.StatusCode == http.StatusUnauthorized {
		resourceMetadataURL := fmt.Sprintf("%s/.well-known/oauth-protected-resource", m.serverURL)
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s", resource_metadata="%s"`, m.serverURL, resourceMetadataURL))
	}

	w.WriteHeader(authResult.StatusCode)

	errorResponse := map[string]interface{}{
		"error":             getOAuthErrorCode(authResult.StatusCode),
		"error_description": authResult.Error,
	}

	json.NewEncoder(w).Encode(errorResponse)
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

// ProtectedResourceMetadataHandler handles OAuth 2.0 Protected Resource Metadata requests
func (m *AuthMiddleware) ProtectedResourceMetadataHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metadata, err := m.provider.GetProtectedResourceMetadata(m.serverURL)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// AuthorizationServerMetadataHandler handles OAuth 2.0 Authorization Server Metadata requests
func (m *AuthMiddleware) AuthorizationServerMetadataHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metadata, err := m.provider.GetAuthorizationServerMetadata()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// GetTokenInfo extracts token information from request context
func GetTokenInfo(r *http.Request) (*auth.TokenInfo, bool) {
	tokenInfo, ok := r.Context().Value("token_info").(*auth.TokenInfo)
	return tokenInfo, ok
}
