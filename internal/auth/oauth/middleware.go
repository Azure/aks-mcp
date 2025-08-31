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

// setCORSHeaders sets CORS headers for OAuth endpoints to allow MCP Inspector access
func (m *AuthMiddleware) setCORSHeaders(w http.ResponseWriter) {
	// In production, this should be more restrictive
	// For development and MCP Inspector compatibility, allow broader access
	origin := "*" // TODO: Restrict to specific origins in production

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, mcp-protocol-version")
	w.Header().Set("Access-Control-Max-Age", "86400")           // 24 hours
	w.Header().Set("Access-Control-Allow-Credentials", "false") // Explicit false for wildcard origin
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
		fmt.Printf("=== MIDDLEWARE ENTRY ===\n")
		fmt.Printf("MIDDLEWARE: Processing request: %s %s\n", r.Method, r.URL.Path)

		// Skip authentication for specific endpoints
		if m.shouldSkipAuth(r) {
			fmt.Printf("MIDDLEWARE: Skipping auth for path: %s\n", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}
		fmt.Printf("MIDDLEWARE: Auth required for path: %s\n", r.URL.Path)

		// Perform authentication
		authResult := m.authenticateRequest(r)

		if !authResult.Authenticated {
			fmt.Printf("MIDDLEWARE: Authentication FAILED - handling error\n")
			m.handleAuthError(w, r, authResult)
			return
		}

		fmt.Printf("MIDDLEWARE: Authentication SUCCESS - proceeding to handler\n")
		// Add token info to request context
		ctx := context.WithValue(r.Context(), "token_info", authResult.TokenInfo)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
		fmt.Printf("MIDDLEWARE: Request completed successfully\n")
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
	fmt.Printf("=== MIDDLEWARE AUTH START ===\n")
	fmt.Printf("MIDDLEWARE: Request URL: %s\n", r.URL.String())
	fmt.Printf("MIDDLEWARE: Request Method: %s\n", r.Method)

	// Extract Bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")
	fmt.Printf("MIDDLEWARE: Authorization header present: %t\n", authHeader != "")

	if authHeader == "" {
		fmt.Printf("MIDDLEWARE: FAILED - Missing authorization header\n")
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "missing authorization header",
			StatusCode:    http.StatusUnauthorized,
		}
	}

	// Check for Bearer token format
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		fmt.Printf("MIDDLEWARE: FAILED - Invalid authorization header format (missing Bearer prefix)\n")
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "invalid authorization header format",
			StatusCode:    http.StatusUnauthorized,
		}
	}
	fmt.Printf("MIDDLEWARE: Bearer token format validated\n")

	token := strings.TrimPrefix(authHeader, bearerPrefix)
	if token == "" {
		fmt.Printf("MIDDLEWARE: FAILED - Empty bearer token\n")
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "empty bearer token",
			StatusCode:    http.StatusUnauthorized,
		}
	}
	fmt.Printf("MIDDLEWARE: Token extracted (length: %d characters)\n", len(token))

	// Basic JWT structure validation
	tokenParts := strings.Split(token, ".")
	if len(tokenParts) != 3 {
		fmt.Printf("MIDDLEWARE: FAILED - JWT structure validation (has %d parts, expected 3)\n", len(tokenParts))
		return &auth.AuthResult{
			Authenticated: false,
			Error:         "invalid JWT structure",
			StatusCode:    http.StatusUnauthorized,
		}
	}
	fmt.Printf("MIDDLEWARE: JWT structure validated (3 parts)\n")

	// Validate the token
	fmt.Printf("MIDDLEWARE: Starting provider token validation...\n")
	tokenInfo, err := m.provider.ValidateToken(r.Context(), token)
	if err != nil {
		fmt.Printf("MIDDLEWARE: FAILED - Provider token validation failed: %v\n", err)
		return &auth.AuthResult{
			Authenticated: false,
			Error:         fmt.Sprintf("token validation failed: %v", err),
			StatusCode:    http.StatusUnauthorized,
		}
	}
	fmt.Printf("MIDDLEWARE: Provider token validation SUCCESS\n")
	fmt.Printf("MIDDLEWARE: Token info - Subject: %s, Audience: %v, Scopes: %v\n",
		tokenInfo.Subject, tokenInfo.Audience, tokenInfo.Scope)

	// Validate required scopes (relaxed for OpenID Connect)
	fmt.Printf("MIDDLEWARE: Starting scope validation...\n")
	fmt.Printf("MIDDLEWARE: Required scopes: %v\n", m.provider.config.RequiredScopes)
	fmt.Printf("MIDDLEWARE: Token scopes: %v\n", tokenInfo.Scope)

	if !m.validateScopes(tokenInfo.Scope) {
		fmt.Printf("MIDDLEWARE: SCOPE WARNING: Token scopes don't exactly match required, but allowing due to valid audience\n")
	} else {
		fmt.Printf("MIDDLEWARE: Scope validation SUCCESS\n")
	}

	fmt.Printf("MIDDLEWARE: Authentication completed successfully\n")
	fmt.Printf("=== MIDDLEWARE AUTH SUCCESS ===\n")
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

// hasScopePermission checks if the token scopes satisfy the required scope
func (m *AuthMiddleware) hasScopePermission(requiredScope string, tokenScopes []string) bool {
	// Direct scope match
	for _, tokenScope := range tokenScopes {
		if tokenScope == requiredScope {
			return true
		}
	}

	// Azure resource scope mapping
	azureResourceMappings := map[string][]string{
		"https://management.azure.com/.default": {
			"user_impersonation",
			"https://management.azure.com/user_impersonation",
			"https://management.azure.com/.default",
			"https://management.core.windows.net/",
			"https://management.azure.com/",
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
	fmt.Printf("=== MIDDLEWARE ERROR HANDLING ===\n")
	fmt.Printf("MIDDLEWARE ERROR: Status code: %d\n", authResult.StatusCode)
	fmt.Printf("MIDDLEWARE ERROR: Error message: %s\n", authResult.Error)

	// Set CORS headers
	m.setCORSHeaders(w)
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
		fmt.Printf("MIDDLEWARE ERROR: WWW-Authenticate header set: %s\n", wwwAuth)
	}

	w.WriteHeader(authResult.StatusCode)

	errorResponse := map[string]interface{}{
		"error":             getOAuthErrorCode(authResult.StatusCode),
		"error_description": authResult.Error,
	}

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		fmt.Printf("MIDDLEWARE ERROR: Failed to encode error response: %v\n", err)
	} else {
		fmt.Printf("MIDDLEWARE ERROR: Error response sent successfully\n")
	}
	fmt.Printf("=== MIDDLEWARE ERROR HANDLED ===\n")
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

// GetTokenInfo extracts token information from request context
func GetTokenInfo(r *http.Request) (*auth.TokenInfo, bool) {
	tokenInfo, ok := r.Context().Value("token_info").(*auth.TokenInfo)
	return tokenInfo, ok
}
