package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/logger"
)

// validateAzureADURL validates that the URL is a legitimate Azure AD endpoint
func validateAzureADURL(tokenURL string) error {
	parsedURL, err := url.Parse(tokenURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow HTTPS for security
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	// Only allow Azure AD endpoints
	if parsedURL.Host != "login.microsoftonline.com" {
		return fmt.Errorf("only Azure AD endpoints are allowed")
	}

	// Validate path format for token endpoint (should be /{tenantId}/oauth2/v2.0/token)
	if !strings.Contains(parsedURL.Path, "/oauth2/v2.0/token") {
		return fmt.Errorf("invalid Azure AD token endpoint path")
	}

	return nil
}

// EndpointManager manages OAuth-related HTTP endpoints
type EndpointManager struct {
	provider      *AzureOAuthProvider
	cfg           *config.ConfigData
	mu            sync.Mutex
	pendingStates map[string]string // state → original client redirect_uri
}

// NewEndpointManager creates a new OAuth endpoint manager
func NewEndpointManager(provider *AzureOAuthProvider, cfg *config.ConfigData) *EndpointManager {
	return &EndpointManager{
		provider:      provider,
		cfg:           cfg,
		pendingStates: make(map[string]string),
	}
}

// setCORSHeaders sets CORS headers for OAuth endpoints with origin whitelisting
func (em *EndpointManager) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	requestOrigin := r.Header.Get("Origin")

	// Check if the request origin is in the allowed list
	var allowedOrigin string
	for _, allowed := range em.provider.config.AllowedOrigins {
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

// setCacheHeaders sets cache control headers based on EnableCache configuration
func (em *EndpointManager) setCacheHeaders(w http.ResponseWriter) {
	if config.EnableCache {
		// Enable caching for 1 hour when cache is enabled
		w.Header().Set("Cache-Control", "max-age=3600")
	} else {
		// Disable all caching when cache is disabled (for debugging)
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
}

// RegisterEndpoints registers OAuth endpoints with the provided HTTP mux
func (em *EndpointManager) RegisterEndpoints(mux *http.ServeMux) {
	// OAuth 2.0 Protected Resource Metadata endpoint (RFC 9728)
	mux.HandleFunc("/.well-known/oauth-protected-resource", em.protectedResourceMetadataHandler())

	// OAuth 2.0 Authorization Server Metadata endpoint (RFC 8414)
	// Note: This would typically be served by Azure AD, but we provide a proxy for convenience
	mux.HandleFunc("/.well-known/oauth-authorization-server", em.authServerMetadataProxyHandler())

	// OpenID Connect Discovery endpoint (compatibility with MCP Inspector)
	mux.HandleFunc("/.well-known/openid-configuration", em.authServerMetadataProxyHandler())

	// Authorization endpoint proxy to handle Azure AD compatibility
	mux.HandleFunc("/oauth2/v2.0/authorize", em.authorizationProxyHandler())

	// Dynamic Client Registration endpoint (RFC 7591)
	mux.HandleFunc("/oauth/register", em.clientRegistrationHandler())

	// Token introspection endpoint (RFC 7662) - optional
	mux.HandleFunc("/oauth/introspect", em.tokenIntrospectionHandler())

	// OAuth 2.0 callback endpoint for Authorization Code flow
	mux.HandleFunc("/oauth/callback", em.callbackHandler())

	// OAuth 2.0 token endpoint for Authorization Code exchange
	mux.HandleFunc("/oauth2/v2.0/token", em.tokenHandler())
}

// authServerMetadataProxyHandler proxies authorization server metadata from Azure AD
func (em *EndpointManager) authServerMetadataProxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("OAuth DEBUG: Received request for authorization server metadata: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w, r)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			logger.Errorf("OAuth ERROR: Invalid method %s for metadata endpoint", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get metadata from Azure AD
		provider := em.provider

		// Build server URL: prefer ExternalURL (needed behind TLS-terminating proxies
		// where r.TLS is always nil), otherwise derive from the request.
		var serverURL string
		if em.provider.config.ExternalURL != "" {
			serverURL = em.provider.config.ExternalURL
		} else {
			// Build server URL based on the request
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}

			// Use the Host header from the request
			host := r.Host
			if host == "" {
				host = r.URL.Host
			}
			serverURL = fmt.Sprintf("%s://%s", scheme, host)
		}

		metadata, err := provider.GetAuthorizationServerMetadata(serverURL)
		if err != nil {
			logger.Errorf("Failed to fetch authorization server metadata: %v", err)
			http.Error(w, fmt.Sprintf("Failed to fetch authorization server metadata: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		em.setCacheHeaders(w)

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			logger.Errorf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// clientRegistrationHandler implements OAuth 2.0 Dynamic Client Registration (RFC 7591)
func (em *EndpointManager) clientRegistrationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("OAuth DEBUG: Received client registration request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w, r)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			logger.Errorf("OAuth ERROR: Invalid method %s for client registration endpoint, only POST allowed", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse client registration request
		var registrationRequest ClientRegistrationRequest

		if err := json.NewDecoder(r.Body).Decode(&registrationRequest); err != nil {
			logger.Errorf("OAuth ERROR: Failed to parse client registration JSON: %v", err)
			em.writeErrorResponse(w, "invalid_request", "Invalid JSON in request body", http.StatusBadRequest)
			return
		}

		logger.Debugf("OAuth DEBUG: Client registration request parsed - client_name: %s, redirect_uris: %v", registrationRequest.ClientName, registrationRequest.RedirectURIs)

		// Validate registration request
		if err := em.validateClientRegistration(&registrationRequest); err != nil {
			logger.Errorf("OAuth ERROR: Client registration validation failed: %v", err)
			em.writeErrorResponse(w, "invalid_client_metadata", err.Error(), http.StatusBadRequest)
			return
		}

		// Use client-requested grant types if provided and valid, otherwise use defaults
		grantTypes := registrationRequest.GrantTypes
		if len(grantTypes) == 0 {
			grantTypes = []string{"authorization_code", "refresh_token"}
		}

		// Use client-requested response types if provided and valid, otherwise use defaults
		responseTypes := registrationRequest.ResponseTypes
		if len(responseTypes) == 0 {
			responseTypes = []string{"code"}
		}

		// For Azure AD compatibility, use the configured client ID
		// In a full RFC 7591 implementation, each registration would get a unique ID
		// But since Azure AD requires pre-registered client IDs, we return the configured one
		clientID := em.cfg.OAuthConfig.ClientID

		clientInfo := map[string]interface{}{
			"client_id":                  clientID,          // Use configured Azure AD client ID
			"client_id_issued_at":        time.Now().Unix(), // RFC 7591: timestamp of issuance
			"redirect_uris":              registrationRequest.RedirectURIs,
			"token_endpoint_auth_method": "none", // Public client (PKCE required)
			"grant_types":                grantTypes,
			"response_types":             responseTypes,
			"client_name":                registrationRequest.ClientName,
			"client_uri":                 registrationRequest.ClientURI,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		if err := json.NewEncoder(w).Encode(clientInfo); err != nil {
			logger.Errorf("OAuth ERROR: Failed to encode client registration response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// validateClientRegistration validates a client registration request
func (em *EndpointManager) validateClientRegistration(req *ClientRegistrationRequest) error {
	// Validate redirect URIs - require at least one
	if len(req.RedirectURIs) == 0 {
		return fmt.Errorf("at least one redirect_uri is required")
	}

	// Basic URL validation for redirect URIs
	for _, redirectURI := range req.RedirectURIs {
		if _, err := url.Parse(redirectURI); err != nil {
			return fmt.Errorf("invalid redirect_uri format: %s", redirectURI)
		}
	}

	// Validate grant types
	validGrantTypes := map[string]bool{
		"authorization_code": true,
		"refresh_token":      true,
	}

	for _, grantType := range req.GrantTypes {
		if !validGrantTypes[grantType] {
			return fmt.Errorf("unsupported grant_type: %s", grantType)
		}
	}

	// Validate response types
	validResponseTypes := map[string]bool{
		"code": true,
	}

	for _, responseType := range req.ResponseTypes {
		if !validResponseTypes[responseType] {
			return fmt.Errorf("unsupported response_type: %s", responseType)
		}
	}

	return nil
}

// validateRedirectURI validates that a redirect URI is registered and allowed
func (em *EndpointManager) validateRedirectURI(redirectURI string) error {
	if len(em.cfg.OAuthConfig.RedirectURIs) == 0 {
		return fmt.Errorf("no redirect URIs configured")
	}

	for _, allowed := range em.cfg.OAuthConfig.RedirectURIs {
		if redirectURI == allowed {
			return nil
		}
	}

	logger.Warnf("OAuth SECURITY WARNING: Invalid redirect URI attempted: %s, allowed: %v",
		redirectURI, em.cfg.OAuthConfig.RedirectURIs)
	return fmt.Errorf("redirect_uri not registered: %s", redirectURI)
}

// tokenIntrospectionHandler implements RFC 7662 OAuth 2.0 Token Introspection
func (em *EndpointManager) tokenIntrospectionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		em.setCORSHeaders(w, r)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// This endpoint should be protected with client authentication
		// For simplicity, we'll skip client auth in this implementation

		token := r.FormValue("token")
		if token == "" {
			em.writeErrorResponse(w, "invalid_request", "Missing token parameter", http.StatusBadRequest)
			return
		}

		// Validate the token
		provider := em.provider

		tokenInfo, err := provider.ValidateToken(r.Context(), token)
		if err != nil {
			// Return inactive token response
			response := map[string]interface{}{
				"active": false,
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				logger.Errorf("Failed to encode introspection response: %v", err)
			}
			return
		}

		// Return active token response
		response := map[string]interface{}{
			"active":    true,
			"client_id": em.cfg.OAuthConfig.ClientID,
			"scope":     strings.Join(tokenInfo.Scope, " "),
			"sub":       tokenInfo.Subject,
			"aud":       tokenInfo.Audience,
			"iss":       tokenInfo.Issuer,
			"exp":       tokenInfo.ExpiresAt.Unix(),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// protectedResourceMetadataHandler handles OAuth 2.0 Protected Resource Metadata requests
func (em *EndpointManager) protectedResourceMetadataHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("OAuth DEBUG: Received request for protected resource metadata: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w, r)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			logger.Errorf("OAuth ERROR: Invalid method %s for protected resource metadata endpoint", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Build the resource URL with correct MCP endpoint path based on transport
		var mcpPath string
		switch em.cfg.Transport {
		case "streamable-http":
			mcpPath = "/mcp"
		case "sse":
			mcpPath = "/sse"
		default:
			mcpPath = ""
		}

		// Build resource URL: prefer ExternalURL (needed behind TLS-terminating proxies
		// where r.TLS is always nil), otherwise derive from the request.
		var resourceURL string
		if em.provider.config.ExternalURL != "" {
			resourceURL = em.provider.config.ExternalURL + mcpPath
		} else {
			// Build resource URL based on the request
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}

			// Use the Host header from the request
			host := r.Host
			if host == "" {
				host = r.URL.Host
			}
			resourceURL = fmt.Sprintf("%s://%s%s", scheme, host, mcpPath)
		}
		logger.Debugf("OAuth DEBUG: Building protected resource metadata for URL: %s (transport: %s)", resourceURL, em.cfg.Transport)

		provider := em.provider

		metadata, err := provider.GetProtectedResourceMetadata(resourceURL)
		if err != nil {
			logger.Errorf("OAuth ERROR: Failed to get protected resource metadata: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		logger.Debugf("OAuth DEBUG: Successfully generated protected resource metadata with %d authorization servers", len(metadata.AuthorizationServers))

		w.Header().Set("Content-Type", "application/json")
		em.setCacheHeaders(w)

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			logger.Errorf("OAuth ERROR: Failed to encode protected resource metadata response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// writeErrorResponse writes an OAuth error response
func (em *EndpointManager) writeErrorResponse(w http.ResponseWriter, errorCode, description string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":             errorCode,
		"error_description": description,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode error response: %v", err)
	}
}

// authorizationProxyHandler proxies authorization requests to Azure AD with resource parameter filtering
func (em *EndpointManager) authorizationProxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("OAuth DEBUG: Received authorization proxy request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w, r)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			logger.Errorf("OAuth ERROR: Invalid method %s for authorization endpoint, only GET allowed", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse query parameters
		query := r.URL.Query()

		// Validate redirect_uri parameter for security and better user experience
		redirectURI := query.Get("redirect_uri")
		if redirectURI == "" {
			logger.Errorf("OAuth ERROR: Missing redirect_uri parameter in authorization request")
			logger.Infof("OAuth HELP: To fix this error, configure redirect URIs using --oauth-redirects flag")
			logger.Infof("OAuth HELP: For MCP Inspector, use: --oauth-redirects=\"http://localhost:8000/oauth/callback,http://localhost:6274/oauth/callback\"")
			em.writeErrorResponse(w, "invalid_request", "redirect_uri parameter is required", http.StatusBadRequest)
			return
		}

		// Validate that the redirect_uri is registered and allowed
		if err := em.validateRedirectURI(redirectURI); err != nil {
			logger.Errorf("OAuth ERROR: redirect_uri %s not registered - requests will be blocked for security", redirectURI)
			em.writeErrorResponse(w, "invalid_request", fmt.Sprintf("redirect_uri not registered: %s", redirectURI), http.StatusBadRequest)
			return
		}

		// Enforce PKCE for OAuth 2.1 compliance (MCP requirement)
		codeChallenge := query.Get("code_challenge")
		codeChallengeMethod := query.Get("code_challenge_method")

		if codeChallenge == "" {
			logger.Errorf("OAuth ERROR: Missing PKCE code_challenge parameter (required for OAuth 2.1)")
			em.writeErrorResponse(w, "invalid_request", "PKCE code_challenge is required", http.StatusBadRequest)
			return
		}

		if codeChallengeMethod == "" {
			// Default to S256 if not specified
			query.Set("code_challenge_method", "S256")
			logger.Debugf("OAuth DEBUG: Setting default code_challenge_method to S256")
		} else if codeChallengeMethod != "S256" {
			logger.Errorf("OAuth ERROR: Unsupported code_challenge_method: %s (only S256 supported)", codeChallengeMethod)
			em.writeErrorResponse(w, "invalid_request", "Only S256 code_challenge_method is supported", http.StatusBadRequest)
			return
		}

		// Resource parameter handling for MCP compliance
		// requestedScopes := strings.Split(query.Get("scope"), " ")

		// Azure AD v2.0 doesn't support RFC 8707 Resource Indicators in authorization requests
		// Remove the resource parameter if present for Azure AD compatibility
		resourceParam := query.Get("resource")
		if resourceParam != "" {
			logger.Debugf("OAuth DEBUG: Removing resource parameter for Azure AD compatibility: %s", resourceParam)
			query.Del("resource")
		}

		// Use only server-required scopes for Azure AD compatibility
		// Azure AD .default scopes cannot be mixed with OpenID Connect scopes
		// We prioritize Azure Management API access over OpenID Connect user info
		finalScopes := em.cfg.OAuthConfig.RequiredScopes

		finalScopeString := strings.Join(finalScopes, " ")
		query.Set("scope", finalScopeString)
		logger.Debugf("OAuth DEBUG: Setting final scope for Azure AD: %s", finalScopeString)

		// Store state → client redirect_uri so the callback handler can relay the
		// authorization code back to the MCP client after Azure AD redirects to our
		// /oauth/callback endpoint. This is client-agnostic: any OAuth 2.1 client
		// (Claude.ai, VS Code, MCP Inspector, etc.) works as long as its redirect_uri
		// is in the configured allowed list.
		state := query.Get("state")
		if state != "" {
			em.mu.Lock()
			em.pendingStates[state] = redirectURI
			em.mu.Unlock()
		}

		// Replace redirect_uri with the server's own callback URL. Azure AD must
		// redirect to a URI registered in the app registration (our server), not
		// directly to the MCP client.
		var serverCallbackURL string
		if em.provider.config.ExternalURL != "" {
			serverCallbackURL = em.provider.config.ExternalURL + "/oauth/callback"
		} else {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			host := r.Host
			if host == "" {
				host = r.URL.Host
			}
			serverCallbackURL = fmt.Sprintf("%s://%s/oauth/callback", scheme, host)
		}
		query.Set("redirect_uri", serverCallbackURL)
		logger.Debugf("OAuth DEBUG: Using server callback URL for Azure AD: %s", serverCallbackURL)

		// Build the Azure AD authorization URL
		azureAuthURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", em.cfg.OAuthConfig.TenantID)

		// Create the redirect URL with filtered parameters
		redirectURL := fmt.Sprintf("%s?%s", azureAuthURL, query.Encode())
		logger.Debugf("OAuth DEBUG: Redirecting to Azure AD authorization endpoint: %s", azureAuthURL)

		// Redirect to Azure AD
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

// callbackHandler handles OAuth 2.0 Authorization Code flow callback.
//
// The server acts as a proxy: Azure AD redirects here with the authorization
// code, and we relay the code back to the original MCP client redirect_uri
// that was stored during /authorize. The MCP client then exchanges the code
// directly with Azure AD (using its PKCE verifier) via /oauth2/v2.0/token.
func (em *EndpointManager) callbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("OAuth DEBUG: Received callback request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w, r)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			logger.Errorf("OAuth ERROR: Invalid method %s for callback endpoint, only GET allowed", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query()
		code := query.Get("code")
		state := query.Get("state")
		errParam := query.Get("error")
		errDesc := query.Get("error_description")

		// Look up and consume the client redirect_uri stored during /authorize.
		em.mu.Lock()
		redirectURI, ok := em.pendingStates[state]
		delete(em.pendingStates, state)
		em.mu.Unlock()

		if !ok || redirectURI == "" {
			logger.Errorf("OAuth ERROR: No pending state found for state=%q", state)
			http.Error(w, "invalid or expired state", http.StatusBadRequest)
			return
		}

		// Forward any upstream error from Azure AD back to the MCP client.
		if errParam != "" {
			logger.Errorf("OAuth ERROR: Authorization server returned error: %s - %s", errParam, errDesc)
			target := redirectURI +
				"?error=" + url.QueryEscape(errParam) +
				"&error_description=" + url.QueryEscape(errDesc) +
				"&state=" + url.QueryEscape(state)
			http.Redirect(w, r, target, http.StatusFound)
			return
		}

		if code == "" {
			logger.Errorf("OAuth ERROR: Missing authorization code in callback")
			http.Error(w, "missing code parameter", http.StatusBadRequest)
			return
		}

		logger.Debugf("OAuth DEBUG: Relaying authorization code to client redirect_uri, state: %s", state)

		// Relay the authorization code to the MCP client. The client holds the
		// PKCE code_verifier and will exchange the code directly with Azure AD.
		target := redirectURI +
			"?code=" + url.QueryEscape(code) +
			"&state=" + url.QueryEscape(state)
		http.Redirect(w, r, target, http.StatusFound)
	}
}

// TokenResponse represents the response from token exchange
type TokenResponse struct {
	AccessToken  string `json:"access_token"` // #nosec G117 -- Standard OAuth2 field name
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"` // #nosec G117 -- Standard OAuth2 field name
	Scope        string `json:"scope,omitempty"`
}

// isValidClientID validates if a client ID is acceptable
func (em *EndpointManager) isValidClientID(clientID string) bool {
	// Accept configured client ID (primary method for Azure AD)
	if clientID == em.cfg.OAuthConfig.ClientID {
		return true
	}

	// For future extensibility, could accept other registered client IDs
	// But for Azure AD integration, we primarily use the configured client ID

	return false
}

// tokenHandler handles OAuth 2.0 token endpoint requests (Authorization Code exchange)
func (em *EndpointManager) tokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("OAuth DEBUG: Received token endpoint request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w, r)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			logger.Errorf("OAuth ERROR: Invalid method %s for token endpoint, only POST allowed", r.Method)
			em.writeErrorResponse(w, "invalid_request", "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			logger.Errorf("OAuth ERROR: Failed to parse form data: %v", err)
			em.writeErrorResponse(w, "invalid_request", "Failed to parse form data", http.StatusBadRequest)
			return
		}

		// Validate grant type
		grantType := r.FormValue("grant_type")
		if grantType != "authorization_code" {
			logger.Errorf("OAuth ERROR: Unsupported grant type: %s", grantType)
			em.writeErrorResponse(w, "unsupported_grant_type", fmt.Sprintf("Unsupported grant type: %s", grantType), http.StatusBadRequest)
			return
		}

		// Extract required parameters
		code := r.FormValue("code")
		clientID := r.FormValue("client_id")
		redirectURI := r.FormValue("redirect_uri")
		codeVerifier := r.FormValue("code_verifier") // PKCE parameter

		if code == "" {
			logger.Errorf("OAuth ERROR: Missing authorization code in token request")
			em.writeErrorResponse(w, "invalid_request", "Missing authorization code", http.StatusBadRequest)
			return
		}

		if clientID == "" {
			logger.Errorf("OAuth ERROR: Missing client_id in token request")
			em.writeErrorResponse(w, "invalid_request", "Missing client_id", http.StatusBadRequest)
			return
		}

		if redirectURI == "" {
			logger.Errorf("OAuth ERROR: Missing redirect_uri in token request")
			em.writeErrorResponse(w, "invalid_request", "Missing redirect_uri", http.StatusBadRequest)
			return
		}

		// Enforce PKCE code_verifier for OAuth 2.1 compliance
		if codeVerifier == "" {
			logger.Errorf("OAuth ERROR: Missing PKCE code_verifier (required for OAuth 2.1)")
			em.writeErrorResponse(w, "invalid_request", "PKCE code_verifier is required", http.StatusBadRequest)
			return
		}

		// Validate client ID (accept both configured and dynamically registered clients)
		if !em.isValidClientID(clientID) {
			logger.Errorf("OAuth ERROR: Invalid client_id: %s", clientID)
			em.writeErrorResponse(w, "invalid_client", "Invalid client_id", http.StatusBadRequest)
			return
		}

		// Validate redirect URI for security
		if err := em.validateRedirectURI(redirectURI); err != nil {
			logger.Errorf("OAuth ERROR: Redirect URI validation failed in token endpoint: %v", err)
			em.writeErrorResponse(w, "invalid_request", "Invalid redirect_uri", http.StatusBadRequest)
			return
		}

		// Azure AD requires the redirect_uri in the token request to exactly match
		// the one used in the authorize request (RFC 6749 §4.1.3). The authorize
		// handler substitutes the server's own callback URL before forwarding to
		// Azure AD, so the token request must use the same URL. The client's
		// redirect_uri has already been validated above.
		var serverCallbackURL string
		if em.provider.config.ExternalURL != "" {
			serverCallbackURL = em.provider.config.ExternalURL + "/oauth/callback"
		} else {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			host := r.Host
			if host == "" {
				host = r.URL.Host
			}
			serverCallbackURL = fmt.Sprintf("%s://%s/oauth/callback", scheme, host)
		}
		logger.Debugf("OAuth DEBUG: Using server callback URL for token exchange: %s", serverCallbackURL)

		// Extract scope from the token request (MCP client should send the same scope)
		requestedScope := r.FormValue("scope")
		if requestedScope == "" {
			// Fallback to server required scopes if not provided
			requestedScope = strings.Join(em.cfg.OAuthConfig.RequiredScopes, " ")
		}

		logger.Debugf("OAuth DEBUG: Exchanging authorization code for access token with Azure AD, scope: %s", requestedScope)

		// Exchange authorization code for access token with Azure AD
		tokenResponse, err := em.exchangeCodeForTokenDirect(code, serverCallbackURL, codeVerifier, requestedScope)
		if err != nil {
			logger.Errorf("OAuth ERROR: Token exchange with Azure AD failed: %v", err)
			em.writeErrorResponse(w, "invalid_grant", fmt.Sprintf("Authorization code exchange failed: %v", err), http.StatusBadRequest)
			return
		}

		// Return token response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")

		if err := json.NewEncoder(w).Encode(tokenResponse); err != nil {
			logger.Errorf("OAuth ERROR: Failed to encode token response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// exchangeCodeForTokenDirect exchanges authorization code for access token directly with Azure AD
func (em *EndpointManager) exchangeCodeForTokenDirect(code, redirectURI, codeVerifier, scope string) (*TokenResponse, error) {
	// Prepare token exchange request to Azure AD
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", em.cfg.OAuthConfig.TenantID)

	// Validate URL for security
	if err := validateAzureADURL(tokenURL); err != nil {
		return nil, fmt.Errorf("invalid token URL: %w", err)
	}

	// Prepare form data
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", em.cfg.OAuthConfig.ClientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("scope", scope) // Use the scope provided by the client

	// Add PKCE code_verifier if present
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
		logger.Debugf("Including PKCE code_verifier in Azure AD token request")
	} else {
		logger.Warnf("No PKCE code_verifier provided - this may cause PKCE verification to fail")
	}

	// Note: Azure AD v2.0 doesn't support the 'resource' parameter in token requests
	// It uses scope-based resource identification instead
	// For MCP compliance, we handle resource binding through audience validation
	logger.Debugf("Azure AD token request with scope: %s", scope)

	// Make token exchange request to Azure AD
	resp, err := http.PostForm(tokenURL, data) // #nosec G107,G704 -- URL is validated above
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	logger.Infof("Token exchange successful: access_token received (length: %d)", len(tokenResponse.AccessToken))

	return &tokenResponse, nil
}
