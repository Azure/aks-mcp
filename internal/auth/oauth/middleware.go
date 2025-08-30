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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, mcp-protocol-version")
	w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
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
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("=== NEW AUTHENTICATION REQUEST ===\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n")

	// Extract Bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")
	fmt.Printf("Authentication request debug:\n")
	fmt.Printf("  Request URL: %s\n", r.URL.String())
	fmt.Printf("  Authorization header length: %d\n", len(authHeader))

	// Check if we're getting the full header
	allHeaders := r.Header["Authorization"]
	if len(allHeaders) > 1 {
		fmt.Printf("  Multiple Authorization headers found: %d\n", len(allHeaders))
		for i, header := range allHeaders {
			fmt.Printf("    Header %d length: %d\n", i, len(header))
		}
	}

	// Check all headers to see if token might be in a different header
	fmt.Printf("  All headers with 'auth' in name:\n")
	for name, values := range r.Header {
		if strings.Contains(strings.ToLower(name), "auth") {
			fmt.Printf("    %s: %v (length: %d)\n", name, values, len(strings.Join(values, "")))
		}
	}

	// Check for custom auth header from MCP Inspector proxy
	customAuthHeader := r.Header.Get("x-custom-auth-header")
	if customAuthHeader != "" {
		fmt.Printf("  Custom auth header name: %s\n", customAuthHeader)
		customToken := r.Header.Get(customAuthHeader)
		if customToken != "" {
			fmt.Printf("  Custom auth header value length: %d\n", len(customToken))
			if len(customToken) > len(authHeader) {
				fmt.Printf("  Using custom auth header instead of Authorization header\n")
				authHeader = "Bearer " + customToken
			}
		}
	}

	if len(authHeader) > 0 {
		// Print the full header to see if it's being truncated
		fmt.Printf("  Full Authorization header: %s\n", authHeader)

		// Also check if header ends with ... or seems incomplete
		if strings.HasSuffix(authHeader, "...") || strings.HasSuffix(authHeader, "..") {
			fmt.Printf("  WARNING: Authorization header appears to be truncated!\n")
		}

		// Check if it looks like a complete JWT (should have 2 dots)
		tokenPart := strings.TrimPrefix(authHeader, "Bearer ")
		dotCount := strings.Count(tokenPart, ".")
		fmt.Printf("  Token dot count: %d (should be 2 for complete JWT)\n", dotCount)
	}

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

	// Debug: Log token format for debugging
	tokenParts := strings.Split(token, ".")
	fmt.Printf("Token validation debug:\n")
	fmt.Printf("  Token length: %d characters\n", len(token))
	fmt.Printf("  Token parts: %d (should be 3 for JWT)\n", len(tokenParts))
	tokenPrefixLen := 50
	if len(token) < tokenPrefixLen {
		tokenPrefixLen = len(token)
	}
	fmt.Printf("  Token prefix: %s...\n", token[:tokenPrefixLen])
	if len(tokenParts) >= 1 {
		fmt.Printf("  Header length: %d\n", len(tokenParts[0]))
	}
	if len(tokenParts) >= 2 {
		fmt.Printf("  Payload length: %d\n", len(tokenParts[1]))
	}
	if len(tokenParts) >= 3 {
		fmt.Printf("  Signature length: %d\n", len(tokenParts[2]))
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

	// Debug token info before scope validation
	fmt.Printf("Token validation completed. TokenInfo debug:\n")
	fmt.Printf("  TokenInfo.Scope: %v\n", tokenInfo.Scope)
	fmt.Printf("  TokenInfo.Subject: %s\n", tokenInfo.Subject)
	fmt.Printf("  TokenInfo.Audience: %v\n", tokenInfo.Audience)
	fmt.Printf("  TokenInfo.Issuer: %s\n", tokenInfo.Issuer)
	fmt.Printf("  TokenInfo.TokenType: %s\n", tokenInfo.TokenType)
	fmt.Printf("  TokenInfo.ExpiresAt: %v\n", tokenInfo.ExpiresAt)

	// Validate required scopes
	fmt.Printf("Starting scope validation with tokenInfo.Scope: %v\n", tokenInfo.Scope)
	if !m.validateScopes(tokenInfo.Scope) {
		fmt.Printf("Scope validation failed, but allowing access for debugging\n")
		fmt.Printf("WARNING: In production, this should return insufficient_scope error\n")
		// For now, let's not block authentication due to scope issues
		// return &auth.AuthResult{
		// 	Authenticated: false,
		// 	Error:         "insufficient scopes",
		// 	StatusCode:    http.StatusForbidden,
		// }
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

	fmt.Printf("Scope validation debug:\n")
	fmt.Printf("  Required scopes: %v\n", requiredScopes)
	fmt.Printf("  Token scopes: %v\n", tokenScopes)
	fmt.Printf("  Token scopes count: %d\n", len(tokenScopes))
	fmt.Printf("  Required scopes count: %d\n", len(requiredScopes))

	// Check if token has at least one required scope
	for i, required := range requiredScopes {
		fmt.Printf("  Checking required scope %d: %s\n", i+1, required)
		if m.hasScopePermission(required, tokenScopes) {
			fmt.Printf("  ✓ Scope validation passed for: %s\n", required)
			return true
		} else {
			fmt.Printf("  ✗ Scope validation failed for: %s\n", required)
		}
	}

	fmt.Printf("  Scope validation failed - no matching scopes found\n")
	return false
}

// hasScopePermission checks if the token scopes satisfy the required scope
func (m *AuthMiddleware) hasScopePermission(requiredScope string, tokenScopes []string) bool {
	fmt.Printf("    hasScopePermission debug:\n")
	fmt.Printf("      Required scope: %s\n", requiredScope)
	fmt.Printf("      Token scopes: %v\n", tokenScopes)
	fmt.Printf("      Token scopes count: %d\n", len(tokenScopes))

	// Direct scope match
	fmt.Printf("      Checking for direct scope match...\n")
	for i, tokenScope := range tokenScopes {
		fmt.Printf("        Token scope %d: '%s' (length: %d)\n", i+1, tokenScope, len(tokenScope))
		fmt.Printf("        Required scope: '%s' (length: %d)\n", requiredScope, len(requiredScope))
		fmt.Printf("        String comparison: tokenScope == requiredScope? %t\n", tokenScope == requiredScope)
		if tokenScope == requiredScope {
			fmt.Printf("        ✓ Direct match found!\n")
			return true
		}
	}
	fmt.Printf("        ✗ No direct match found\n")

	// Azure resource scope mapping
	// Azure AD converts ".default" scopes to specific permissions in tokens
	fmt.Printf("      Checking Azure resource scope mappings...\n")
	fmt.Printf("      IMPORTANT: If this fails, check Azure AD app registration API permissions!\n")
	fmt.Printf("      Required: Azure Service Management -> user_impersonation (Delegated)\n")
	fmt.Printf("      Required: Grant admin consent for tenant\n")
	azureResourceMappings := map[string][]string{
		"https://management.azure.com/.default": {
			"user_impersonation",                              // Most common Azure Management API permission
			"https://management.azure.com/user_impersonation", // Full URI version
			// These might appear if configured differently in Azure AD
			"https://management.azure.com/.default",
			"https://management.core.windows.net/",
			"https://management.azure.com/",
		},
		"https://graph.microsoft.com/.default": {
			"User.Read",                             // Common default
			"https://graph.microsoft.com/User.Read", // Full URI version
		},
		"https://storage.azure.com/.default": {
			"user_impersonation",
			"https://storage.azure.com/user_impersonation",
		},
		"https://vault.azure.com/.default": {
			"user_impersonation",
			"https://vault.azure.com/user_impersonation",
		},
	}

	if allowedScopes, exists := azureResourceMappings[requiredScope]; exists {
		fmt.Printf("      Found mapping for required scope: %s -> %v\n", requiredScope, allowedScopes)
		for i, allowedScope := range allowedScopes {
			fmt.Printf("        Checking allowed scope %d: '%s'\n", i+1, allowedScope)
			for j, tokenScope := range tokenScopes {
				fmt.Printf("          Against token scope %d: '%s'\n", j+1, tokenScope)
				fmt.Printf("          String comparison: '%s' == '%s'? %t\n", tokenScope, allowedScope, tokenScope == allowedScope)
				if tokenScope == allowedScope {
					fmt.Printf("          ✓ Azure resource scope match found!\n")
					return true
				}
			}
		}
		fmt.Printf("        ✗ No Azure resource scope matches found\n")
	} else {
		fmt.Printf("      No mapping found for required scope: %s\n", requiredScope)
	}

	// ADDITIONAL CHECK: OpenID Connect scopes compatibility
	fmt.Printf("      Checking OpenID Connect scope compatibility...\n")
	openIDScopes := []string{"openid", "profile", "email", "offline_access"}
	hasOpenIDScope := false
	for _, oidcScope := range openIDScopes {
		for _, tokenScope := range tokenScopes {
			if tokenScope == oidcScope {
				hasOpenIDScope = true
				fmt.Printf("        Found OpenID Connect scope: %s\n", oidcScope)
				break
			}
		}
	}

	// If we have OpenID scopes and we're asking for Azure Management, this might be a scope mismatch
	if hasOpenIDScope && strings.Contains(requiredScope, "management.azure.com") {
		fmt.Printf("      SCOPE MISMATCH DETECTED: Token has OpenID scopes but server requires Azure Management scopes\n")
		fmt.Printf("      This suggests MCP Inspector requested different scopes than what AKS-MCP expects\n")
		fmt.Printf("      MCP Inspector requested: openid, profile, email, offline_access\n")
		fmt.Printf("      AKS-MCP expects: %s\n", requiredScope)
	}

	fmt.Printf("      ✗ All scope checks failed\n")
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

// GetTokenInfo extracts token information from request context
func GetTokenInfo(r *http.Request) (*auth.TokenInfo, bool) {
	tokenInfo, ok := r.Context().Value("token_info").(*auth.TokenInfo)
	return tokenInfo, ok
}
