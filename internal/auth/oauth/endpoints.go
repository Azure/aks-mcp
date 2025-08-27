package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
	"github.com/Azure/aks-mcp/internal/config"
)

// EndpointManager manages OAuth-related HTTP endpoints
type EndpointManager struct {
	provider *AzureOAuthProvider
	config   *auth.OAuthConfig
}

// NewEndpointManager creates a new OAuth endpoint manager
func NewEndpointManager(provider *AzureOAuthProvider, config *auth.OAuthConfig) *EndpointManager {
	return &EndpointManager{
		provider: provider,
		config:   config,
	}
}

// setCORSHeaders sets CORS headers for OAuth endpoints to allow MCP Inspector access
func (em *EndpointManager) setCORSHeaders(w http.ResponseWriter) {
	origin := "*" // TODO: Restrict to specific origins

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, mcp-protocol-version")
	w.Header().Set("Access-Control-Max-Age", "86400")           // 24 hours
	w.Header().Set("Access-Control-Allow-Credentials", "false") // Explicit false for wildcard origin
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

	// Health check endpoint (unauthenticated)
	mux.HandleFunc("/health", em.healthHandler())
}

// authServerMetadataProxyHandler proxies authorization server metadata from Azure AD
func (em *EndpointManager) authServerMetadataProxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("OAuth DEBUG: Received request for authorization server metadata: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			log.Printf("OAuth ERROR: Invalid method %s for metadata endpoint", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get metadata from Azure AD
		provider := em.provider

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

		serverURL := fmt.Sprintf("%s://%s", scheme, host)

		metadata, err := provider.GetAuthorizationServerMetadata(serverURL)
		if err != nil {
			log.Printf("Failed to fetch authorization server metadata: %v\n", err)
			http.Error(w, fmt.Sprintf("Failed to fetch authorization server metadata: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		em.setCacheHeaders(w)

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			log.Printf("Failed to encode response: %v\n", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// clientRegistrationHandler implements OAuth 2.0 Dynamic Client Registration (RFC 7591)
func (em *EndpointManager) clientRegistrationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("OAuth DEBUG: Received client registration request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			log.Printf("OAuth ERROR: Invalid method %s for client registration endpoint, only POST allowed", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse client registration request
		var registrationRequest ClientRegistrationRequest

		if err := json.NewDecoder(r.Body).Decode(&registrationRequest); err != nil {
			log.Printf("OAuth ERROR: Failed to parse client registration JSON: %v", err)
			em.writeErrorResponse(w, "invalid_request", "Invalid JSON in request body", http.StatusBadRequest)
			return
		}

		log.Printf("OAuth DEBUG: Client registration request parsed - client_name: %s, redirect_uris: %v", registrationRequest.ClientName, registrationRequest.RedirectURIs)

		// Validate registration request
		if err := em.validateClientRegistration(&registrationRequest); err != nil {
			log.Printf("OAuth ERROR: Client registration validation failed: %v", err)
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
		clientID := em.config.ClientID

		log.Printf("OAuth DEBUG: Client registration successful - returning client_id: %s", clientID)

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
			log.Printf("OAuth ERROR: Failed to encode client registration response: %v", err)
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

// tokenIntrospectionHandler implements RFC 7662 OAuth 2.0 Token Introspection
func (em *EndpointManager) tokenIntrospectionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		em.setCORSHeaders(w)

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
			json.NewEncoder(w).Encode(response)
			return
		}

		// Return active token response
		response := map[string]interface{}{
			"active":    true,
			"client_id": em.config.ClientID,
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

// healthHandler provides a simple health check endpoint
func (em *EndpointManager) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		response := map[string]interface{}{
			"status": "healthy",
			"oauth": map[string]interface{}{
				"enabled": em.config.Enabled,
			},
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
		log.Printf("OAuth DEBUG: Received request for protected resource metadata: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			log.Printf("OAuth ERROR: Invalid method %s for protected resource metadata endpoint", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

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

		// Build the resource URL
		resourceURL := fmt.Sprintf("%s://%s", scheme, host)
		log.Printf("OAuth DEBUG: Building protected resource metadata for URL: %s", resourceURL)

		provider := em.provider

		metadata, err := provider.GetProtectedResourceMetadata(resourceURL)
		if err != nil {
			log.Printf("OAuth ERROR: Failed to get protected resource metadata: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("OAuth DEBUG: Successfully generated protected resource metadata with %d authorization servers", len(metadata.AuthorizationServers))

		w.Header().Set("Content-Type", "application/json")
		em.setCacheHeaders(w)

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			log.Printf("OAuth ERROR: Failed to encode protected resource metadata response: %v", err)
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

	json.NewEncoder(w).Encode(response)
}

// authorizationProxyHandler proxies authorization requests to Azure AD with resource parameter filtering
func (em *EndpointManager) authorizationProxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("OAuth DEBUG: Received authorization proxy request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			log.Printf("OAuth ERROR: Invalid method %s for authorization endpoint, only GET allowed", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse query parameters
		query := r.URL.Query()

		// Enforce PKCE for OAuth 2.1 compliance (MCP requirement)
		codeChallenge := query.Get("code_challenge")
		codeChallengeMethod := query.Get("code_challenge_method")

		if codeChallenge == "" {
			log.Printf("OAuth ERROR: Missing PKCE code_challenge parameter (required for OAuth 2.1)")
			em.writeErrorResponse(w, "invalid_request", "PKCE code_challenge is required", http.StatusBadRequest)
			return
		}

		if codeChallengeMethod == "" {
			// Default to S256 if not specified
			query.Set("code_challenge_method", "S256")
			log.Printf("OAuth DEBUG: Setting default code_challenge_method to S256")
		} else if codeChallengeMethod != "S256" {
			log.Printf("OAuth ERROR: Unsupported code_challenge_method: %s (only S256 supported)", codeChallengeMethod)
			em.writeErrorResponse(w, "invalid_request", "Only S256 code_challenge_method is supported", http.StatusBadRequest)
			return
		}

		// Resource parameter handling for MCP compliance
		// requestedScopes := strings.Split(query.Get("scope"), " ")

		// Azure AD v2.0 doesn't support RFC 8707 Resource Indicators in authorization requests
		// Remove the resource parameter if present for Azure AD compatibility
		resourceParam := query.Get("resource")
		if resourceParam != "" {
			log.Printf("OAuth DEBUG: Removing resource parameter for Azure AD compatibility: %s", resourceParam)
			query.Del("resource")
		}

		// Use only server-required scopes for Azure AD compatibility
		// Azure AD .default scopes cannot be mixed with OpenID Connect scopes
		// We prioritize Azure Management API access over OpenID Connect user info
		finalScopes := em.config.RequiredScopes

		finalScopeString := strings.Join(finalScopes, " ")
		query.Set("scope", finalScopeString)
		log.Printf("OAuth DEBUG: Setting final scope for Azure AD: %s", finalScopeString)

		// Build the Azure AD authorization URL
		azureAuthURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", em.config.TenantID)

		// Create the redirect URL with filtered parameters
		redirectURL := fmt.Sprintf("%s?%s", azureAuthURL, query.Encode())
		log.Printf("OAuth DEBUG: Redirecting to Azure AD authorization endpoint: %s", azureAuthURL)

		// Redirect to Azure AD
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

// callbackHandler handles OAuth 2.0 Authorization Code flow callback
func (em *EndpointManager) callbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("OAuth DEBUG: Received callback request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			log.Printf("OAuth ERROR: Invalid method %s for callback endpoint, only GET allowed", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse query parameters
		query := r.URL.Query()

		// Check for error response from authorization server
		if authError := query.Get("error"); authError != "" {
			errorDesc := query.Get("error_description")
			log.Printf("OAuth ERROR: Authorization server returned error: %s - %s", authError, errorDesc)
			em.writeCallbackErrorResponse(w, fmt.Sprintf("Authorization failed: %s - %s", authError, errorDesc))
			return
		}

		// Get authorization code
		code := query.Get("code")
		if code == "" {
			log.Printf("OAuth ERROR: Missing authorization code in callback")
			em.writeCallbackErrorResponse(w, "Missing authorization code")
			return
		}

		// Get state parameter for CSRF protection
		state := query.Get("state")
		if state == "" {
			log.Printf("OAuth ERROR: Missing state parameter in callback")
			em.writeCallbackErrorResponse(w, "Missing state parameter")
			return
		}

		log.Printf("OAuth DEBUG: Callback parameters validated - has_code: true, state: %s", state)

		// Exchange authorization code for access token
		tokenResponse, err := em.exchangeCodeForToken(code, state)
		if err != nil {
			log.Printf("OAuth ERROR: Failed to exchange authorization code for token: %v", err)
			em.writeCallbackErrorResponse(w, fmt.Sprintf("Failed to exchange code for token: %v", err))
			return
		}

		log.Printf("OAuth DEBUG: Token exchange successful, preparing callback success response")

		// Skip token validation in callback - validation happens during MCP requests
		// Create minimal token info for callback success page
		tokenInfo := &auth.TokenInfo{
			AccessToken: tokenResponse.AccessToken,
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(time.Hour), // Default 1 hour expiration
			Scope:       em.config.RequiredScopes,  // Use configured scopes
			Subject:     "authenticated_user",      // Placeholder
			Audience:    []string{fmt.Sprintf("https://sts.windows.net/%s/", em.config.TenantID)},
			Issuer:      fmt.Sprintf("https://sts.windows.net/%s/", em.config.TenantID),
			Claims:      make(map[string]interface{}),
		}

		// Return success response with token information
		em.writeCallbackSuccessResponse(w, tokenResponse, tokenInfo)
	}
}

// TokenResponse represents the response from token exchange
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// exchangeCodeForToken exchanges authorization code for access token
func (em *EndpointManager) exchangeCodeForToken(code, state string) (*TokenResponse, error) {
	// Prepare token exchange request
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", em.config.TenantID)

	// Use default callback redirect URI for token exchange
	redirectURI := fmt.Sprintf("http://localhost:%d/oauth/callback", 8080) // Default for MCP servers

	// Prepare form data
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", em.config.ClientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("scope", strings.Join(em.config.RequiredScopes, " "))

	// Note: Azure AD v2.0 doesn't support the 'resource' parameter in token requests
	// It uses scope-based resource identification instead
	// For MCP compliance, we handle resource binding through audience validation

	// Make token exchange request
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResponse, nil
}

// writeCallbackErrorResponse writes an error response for callback
func (em *EndpointManager) writeCallbackErrorResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>OAuth Authentication Error</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .error { background-color: #fee; border: 1px solid #fcc; padding: 20px; border-radius: 5px; }
        .error h1 { color: #c33; margin-top: 0; }
    </style>
</head>
<body>
    <div class="error">
        <h1>Authentication Error</h1>
        <p>%s</p>
        <p>Please try again or contact your administrator.</p>
    </div>
</body>
</html>`, message)

	w.Write([]byte(html))
}

// writeCallbackSuccessResponse writes a success response for callback
func (em *EndpointManager) writeCallbackSuccessResponse(w http.ResponseWriter, tokenResponse *TokenResponse, tokenInfo *auth.TokenInfo) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Generate a secure session token for the client to use
	_, err := em.generateSessionToken()
	if err != nil {
		em.writeCallbackErrorResponse(w, "Failed to generate session token")
		return
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>OAuth Authentication Success</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .success { background-color: #efe; border: 1px solid #cfc; padding: 20px; border-radius: 5px; }
        .success h1 { color: #3c3; margin-top: 0; }
        .token-info { background-color: #f9f9f9; border: 1px solid #ddd; padding: 15px; margin: 15px 0; border-radius: 3px; }
        .token { font-family: monospace; word-break: break-all; background-color: #f5f5f5; padding: 10px; border-radius: 3px; }
        .copy-btn { background-color: #007cba; color: white; border: none; padding: 5px 10px; border-radius: 3px; cursor: pointer; }
    </style>
</head>
<body>
    <div class="success">
        <h1>Authentication Successful</h1>
        <p>You have been successfully authenticated with Azure AD.</p>
        
        <div class="token-info">
            <h3>Access Token (use as Bearer token):</h3>
            <div class="token" id="accessToken">%s</div>
            <button class="copy-btn" onclick="copyToClipboard('accessToken')">Copy Token</button>
        </div>
        
        <div class="token-info">
            <h3>Token Information:</h3>
            <ul>
                <li><strong>Subject:</strong> %s</li>
                <li><strong>Audience:</strong> %s</li>
                <li><strong>Scope:</strong> %s</li>
                <li><strong>Expires:</strong> %s</li>
            </ul>
        </div>
        
        <div class="token-info">
            <h3>For MCP Client Usage:</h3>
            <p>Use this token in the Authorization header:</p>
            <div class="token">Authorization: Bearer %s</div>
            <button class="copy-btn" onclick="copyToClipboard('bearerToken')">Copy Authorization Header</button>
        </div>
    </div>
    
    <script>
        function copyToClipboard(elementId) {
            const element = document.getElementById(elementId);
            const text = elementId === 'bearerToken' ? 'Bearer ' + element.textContent : element.textContent;
            navigator.clipboard.writeText(text).then(function() {
                alert('Copied to clipboard!');
            });
        }
        
        // Set hidden bearer token element
        const bearerTokenElement = document.createElement('div');
        bearerTokenElement.id = 'bearerToken';
        bearerTokenElement.style.display = 'none';
        bearerTokenElement.textContent = '%s';
        document.body.appendChild(bearerTokenElement);
    </script>
</body>
</html>`,
		tokenResponse.AccessToken,
		tokenInfo.Subject,
		strings.Join(tokenInfo.Audience, ", "),
		strings.Join(tokenInfo.Scope, ", "),
		tokenInfo.ExpiresAt.Format("2006-01-02 15:04:05 UTC"),
		tokenResponse.AccessToken,
		tokenResponse.AccessToken)

	w.Write([]byte(html))
}

// isValidClientID validates if a client ID is acceptable
func (em *EndpointManager) isValidClientID(clientID string) bool {
	// Accept configured client ID (primary method for Azure AD)
	if clientID == em.config.ClientID {
		return true
	}

	// For future extensibility, could accept other registered client IDs
	// But for Azure AD integration, we primarily use the configured client ID

	return false
}

// generateSessionToken generates a secure random session token
func (em *EndpointManager) generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// tokenHandler handles OAuth 2.0 token endpoint requests (Authorization Code exchange)
func (em *EndpointManager) tokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("OAuth DEBUG: Received token endpoint request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			log.Printf("OAuth ERROR: Invalid method %s for token endpoint, only POST allowed", r.Method)
			em.writeErrorResponse(w, "invalid_request", "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			log.Printf("OAuth ERROR: Failed to parse form data: %v", err)
			em.writeErrorResponse(w, "invalid_request", "Failed to parse form data", http.StatusBadRequest)
			return
		}

		// Validate grant type
		grantType := r.FormValue("grant_type")
		if grantType != "authorization_code" {
			log.Printf("OAuth ERROR: Unsupported grant type: %s", grantType)
			em.writeErrorResponse(w, "unsupported_grant_type", fmt.Sprintf("Unsupported grant type: %s", grantType), http.StatusBadRequest)
			return
		}

		// Extract required parameters
		code := r.FormValue("code")
		clientID := r.FormValue("client_id")
		redirectURI := r.FormValue("redirect_uri")
		codeVerifier := r.FormValue("code_verifier") // PKCE parameter

		if code == "" {
			log.Printf("OAuth ERROR: Missing authorization code in token request")
			em.writeErrorResponse(w, "invalid_request", "Missing authorization code", http.StatusBadRequest)
			return
		}

		if clientID == "" {
			log.Printf("OAuth ERROR: Missing client_id in token request")
			em.writeErrorResponse(w, "invalid_request", "Missing client_id", http.StatusBadRequest)
			return
		}

		if redirectURI == "" {
			log.Printf("OAuth ERROR: Missing redirect_uri in token request")
			em.writeErrorResponse(w, "invalid_request", "Missing redirect_uri", http.StatusBadRequest)
			return
		}

		// Enforce PKCE code_verifier for OAuth 2.1 compliance
		if codeVerifier == "" {
			log.Printf("OAuth ERROR: Missing PKCE code_verifier (required for OAuth 2.1)")
			em.writeErrorResponse(w, "invalid_request", "PKCE code_verifier is required", http.StatusBadRequest)
			return
		}

		log.Printf("OAuth DEBUG: Token request parameters validated - client_id: %s, redirect_uri: %s, has_code_verifier: %t", clientID, redirectURI, codeVerifier != "")

		// Validate client ID (accept both configured and dynamically registered clients)
		if !em.isValidClientID(clientID) {
			log.Printf("OAuth ERROR: Invalid client_id: %s", clientID)
			em.writeErrorResponse(w, "invalid_client", "Invalid client_id", http.StatusBadRequest)
			return
		}

		// Extract scope from the token request (MCP client should send the same scope)
		requestedScope := r.FormValue("scope")
		if requestedScope == "" {
			// Fallback to server required scopes if not provided
			requestedScope = strings.Join(em.config.RequiredScopes, " ")
		}

		log.Printf("OAuth DEBUG: Exchanging authorization code for access token with Azure AD, scope: %s", requestedScope)

		// Exchange authorization code for access token with Azure AD
		tokenResponse, err := em.exchangeCodeForTokenDirect(code, redirectURI, codeVerifier, requestedScope)
		if err != nil {
			log.Printf("OAuth ERROR: Token exchange with Azure AD failed: %v", err)
			em.writeErrorResponse(w, "invalid_grant", fmt.Sprintf("Authorization code exchange failed: %v", err), http.StatusBadRequest)
			return
		}

		log.Printf("OAuth DEBUG: Token exchange successful, returning token response to client")

		// Return token response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")

		if err := json.NewEncoder(w).Encode(tokenResponse); err != nil {
			log.Printf("OAuth ERROR: Failed to encode token response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// exchangeCodeForTokenDirect exchanges authorization code for access token directly with Azure AD
func (em *EndpointManager) exchangeCodeForTokenDirect(code, redirectURI, codeVerifier, scope string) (*TokenResponse, error) {
	// Prepare token exchange request to Azure AD
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", em.config.TenantID)

	// Prepare form data
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", em.config.ClientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("scope", scope) // Use the scope provided by the client

	// Add PKCE code_verifier if present
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
		log.Printf("Including PKCE code_verifier in Azure AD token request")
	} else {
		log.Printf("No PKCE code_verifier provided - this may cause PKCE verification to fail")
	}

	// Note: Azure AD v2.0 doesn't support the 'resource' parameter in token requests
	// It uses scope-based resource identification instead
	// For MCP compliance, we handle resource binding through audience validation
	log.Printf("Azure AD token request with scope: %s", scope)

	// Make token exchange request to Azure AD
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	log.Printf("Token exchange successful: access_token received (length: %d)", len(tokenResponse.AccessToken))

	return &tokenResponse, nil
}
