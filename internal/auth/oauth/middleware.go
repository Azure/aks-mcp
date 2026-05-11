package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
	appctx "github.com/Azure/aks-mcp/internal/ctx"
	"github.com/Azure/aks-mcp/internal/logger"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const tokenInfoKey contextKey = "token_info"

// aksServerAppID is the well-known Azure AD application ID for the AKS server,
// used as the token audience when obtaining cluster tokens for Kubernetes RBAC clusters.
const aksServerAppID = "6dae42f8-4368-4678-94ff-3960e28e3630"

// sessionEntry caches OBO tokens for an established MCP session so that
// follow-up requests (which carry Mcp-Session-Id but no Authorization header)
// can still reach downstream tools with the correct Azure tokens.
type sessionEntry struct {
	tokenInfo    *auth.TokenInfo
	bearerToken  string // original bearer token, used to proactively refresh OBO tokens
	azureToken   string
	clusterToken string
	oboExpiresAt time.Time // when the OBO-derived tokens expire (~1h from issue)
	expiresAt    time.Time // when the session itself expires (bearer token lifetime)
}

// AuthMiddleware handles OAuth authentication for HTTP requests
type AuthMiddleware struct {
	provider  *AzureOAuthProvider
	serverURL string
	sessions  sync.Map // Mcp-Session-Id → *sessionEntry
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

		// Claude Web (and other MCP clients) authenticate once on the initial request,
		// then send subsequent requests with only Mcp-Session-Id and no Authorization header.
		// Re-use the cached auth result for these session continuation requests.
		sessionHandled := false
		if sessionID := r.Header.Get("Mcp-Session-Id"); sessionID != "" {
			if val, ok := m.sessions.Load(sessionID); ok {
				se := val.(*sessionEntry)
				if time.Now().Before(se.expiresAt) {
					// Proactively refresh OBO tokens 5 minutes before expiry so the
					// session stays alive without the user needing to reconnect.
					if m.provider.config.OBOEnabled && time.Now().After(se.oboExpiresAt.Add(-5*time.Minute)) {
						logger.Debugf("Session %s: proactively refreshing OBO tokens", sessionID)
						if !m.refreshSessionOBO(r.Context(), se, sessionID) {
							// Bearer token has also expired — fall through to full re-auth
							m.sessions.Delete(sessionID)
							logger.Debugf("Session %s: OBO refresh failed, re-authentication required", sessionID)
						}
					}
					if _, stillValid := m.sessions.Load(sessionID); stillValid {
						ctx := context.WithValue(r.Context(), tokenInfoKey, se.tokenInfo)
						if se.azureToken != "" {
							ctx = context.WithValue(ctx, appctx.AzureTokenKey, se.azureToken)
						}
						if se.clusterToken != "" {
							ctx = context.WithValue(ctx, appctx.AzureClusterTokenKey, se.clusterToken)
						}
						r = r.WithContext(ctx)
						next.ServeHTTP(w, r)
						sessionHandled = true
					}
				} else {
					m.sessions.Delete(sessionID)
					logger.Debugf("Session %s expired, requiring re-authentication", sessionID)
				}
			}
		}

		if sessionHandled {
			return
		}

		// Full authentication for new or expired sessions
		authResult := m.authenticateRequest(r)

		if !authResult.Authenticated {
			logger.Errorf("Authentication FAILED - handling error")
			m.handleAuthError(w, r, authResult)
			return
		}

		// Add token info and OBO tokens to request context
		ctx := context.WithValue(r.Context(), tokenInfoKey, authResult.TokenInfo)
		if authResult.AzureToken != "" {
			ctx = context.WithValue(ctx, appctx.AzureTokenKey, authResult.AzureToken)
		}
		if authResult.AzureClusterToken != "" {
			ctx = context.WithValue(ctx, appctx.AzureClusterTokenKey, authResult.AzureClusterToken)
		}
		r = r.WithContext(ctx)

		// Wrap the ResponseWriter to capture the Mcp-Session-Id mcp-go assigns on the
		// response, then cache the full auth result (including bearer token for OBO refresh).
		oboExpiry := time.Now().Add(55 * time.Minute) // slightly under the 1h ARM token lifetime
		sessionExpiry := time.Now().Add(24 * time.Hour)
		if authResult.TokenInfo != nil && authResult.TokenInfo.ExpiresAt.After(time.Now()) && authResult.TokenInfo.ExpiresAt.Before(sessionExpiry) {
			sessionExpiry = authResult.TokenInfo.ExpiresAt
		}
		bearerToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		scw := &sessionCapturingWriter{
			ResponseWriter: w,
			onSession: func(sessionID string) {
				m.sessions.Store(sessionID, &sessionEntry{
					tokenInfo:    authResult.TokenInfo,
					bearerToken:  bearerToken,
					azureToken:   authResult.AzureToken,
					clusterToken: authResult.AzureClusterToken,
					oboExpiresAt: oboExpiry,
					expiresAt:    sessionExpiry,
				})
				logger.Debugf("Cached auth for session %s (OBO expires: %v, session expires: %v)", sessionID, oboExpiry, sessionExpiry)
			},
		}
		next.ServeHTTP(scw, r)
	})
}

// sessionCapturingWriter wraps http.ResponseWriter to capture the Mcp-Session-Id header
// that mcp-go sets on the response when it creates a new session.
type sessionCapturingWriter struct {
	http.ResponseWriter
	onSession func(string)
	once      sync.Once
}

func (w *sessionCapturingWriter) capture() {
	if sessionID := w.ResponseWriter.Header().Get("Mcp-Session-Id"); sessionID != "" {
		w.onSession(sessionID)
	}
}

func (w *sessionCapturingWriter) WriteHeader(code int) {
	w.once.Do(w.capture)
	w.ResponseWriter.WriteHeader(code)
}

func (w *sessionCapturingWriter) Write(b []byte) (int, error) {
	w.once.Do(w.capture)
	return w.ResponseWriter.Write(b)
}

func (w *sessionCapturingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// refreshSessionOBO uses the cached bearer token to get fresh OBO tokens and updates
// the session entry in-place. Returns false if the bearer token has expired.
func (m *AuthMiddleware) refreshSessionOBO(ctx context.Context, se *sessionEntry, sessionID string) bool {
	if se.bearerToken == "" {
		return false
	}

	armToken, err := m.provider.ExchangeOBO(ctx, se.bearerToken, "https://management.azure.com/user_impersonation")
	if err != nil {
		logger.Warnf("Session %s: ARM OBO refresh failed: %v", sessionID, err)
		return false
	}

	se.azureToken = armToken
	se.oboExpiresAt = time.Now().Add(55 * time.Minute)

	if clusterToken, err := m.provider.ExchangeOBO(ctx, se.bearerToken, aksServerAppID+"/.default"); err != nil {
		logger.Warnf("Session %s: cluster OBO refresh failed: %v", sessionID, err)
	} else {
		se.clusterToken = clusterToken
	}

	m.sessions.Store(sessionID, se)
	logger.Debugf("Session %s: OBO tokens refreshed successfully", sessionID)
	return true
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

	result := &auth.AuthResult{
		Authenticated: true,
		TokenInfo:     tokenInfo,
		StatusCode:    http.StatusOK,
	}

	// OBO exchange: trade the user's MCP bearer token for tokens needed by downstream tools.
	if m.provider.config.OBOEnabled {
		// ARM token — authenticates the RunCommand API call against Azure management plane.
		armToken, err := m.provider.ExchangeOBO(r.Context(), token, "https://management.azure.com/user_impersonation")
		if err != nil {
			logger.Errorf("OBO ARM exchange failed for user %s: %v", tokenInfo.Subject, err)
			return &auth.AuthResult{
				Authenticated: false,
				Error:         fmt.Sprintf("OBO token exchange failed: %v", err),
				StatusCode:    http.StatusUnauthorized,
			}
		}
		result.AzureToken = armToken

		// AKS cluster token — passed as clusterToken in RunCommand requests for AAD-enabled clusters
		// that use Kubernetes RBAC. Audience: aksServerAppID (AKS server app).
		clusterToken, err := m.provider.ExchangeOBO(r.Context(), token, aksServerAppID+"/.default")
		if err != nil {
			logger.Warnf("OBO cluster token exchange failed for user %s: %v — kubectl on AAD clusters may fail", tokenInfo.Subject, err)
		} else {
			result.AzureClusterToken = clusterToken
		}
	}

	return result
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

	// For custom App ID URI scopes (e.g., api://my-app/.default):
	// Accept tokens with app roles defined in the Enterprise Application
	// Also accept common delegated scope patterns
	if strings.HasPrefix(requiredScope, "api://") {
		// Extract app ID from scope for logging
		appResource := strings.TrimSuffix(requiredScope, "/.default")

		// Accept common app roles that may be assigned to users/SPNs in Enterprise Application
		customAppRoles := []string{
			"access_as_user", // Common delegated scope
			"user_impersonation",
			"Reader",
			"Contributor",
			"Owner",
			"Admin",
			"User",
		}
		for _, role := range customAppRoles {
			for _, tokenScope := range tokenScopes {
				if tokenScope == role {
					logger.Debugf("Scope validation: accepting custom app scope %s with role %s for resource %s", tokenScope, role, appResource)
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
		// Build the resource metadata URL: prefer ExternalURL (needed behind TLS-terminating
		// proxies where r.TLS is always nil), otherwise derive from the request.
		var serverURL string
		if m.provider.config.ExternalURL != "" {
			serverURL = m.provider.config.ExternalURL
		} else {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			host := r.Host
			if host == "" {
				host = r.URL.Host
			}
			serverURL = fmt.Sprintf("%s://%s", scheme, host)
		}
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
