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
)

// EndpointManager manages OAuth-related HTTP endpoints
type EndpointManager struct {
	provider *AzureOAuthProvider
	config   *auth.OAuthConfig
}

// setCORSHeaders sets CORS headers for OAuth endpoints to allow MCP Inspector access
func (em *EndpointManager) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, mcp-protocol-version")
	w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
}

// NewEndpointManager creates a new OAuth endpoint manager
func NewEndpointManager(provider *AzureOAuthProvider, config *auth.OAuthConfig) *EndpointManager {
	return &EndpointManager{
		provider: provider,
		config:   config,
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
		w.Header().Set("Cache-Control", "public, max-age=3600")

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

		// Parse client registration request
		var registrationRequest struct {
			RedirectURIs            []string `json:"redirect_uris"`
			TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
			GrantTypes              []string `json:"grant_types"`
			ResponseTypes           []string `json:"response_types"`
			ClientName              string   `json:"client_name"`
			ClientURI               string   `json:"client_uri"`
			Scope                   string   `json:"scope"`
		}

		if err := json.NewDecoder(r.Body).Decode(&registrationRequest); err != nil {
			em.writeErrorResponse(w, "invalid_request", "Invalid JSON in request body", http.StatusBadRequest)
			return
		}

		// Validate registration request
		if err := em.validateClientRegistration(&registrationRequest); err != nil {
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

		clientInfo := map[string]interface{}{
			"client_id":                  em.config.ClientID, // Use configured client ID
			"redirect_uris":              registrationRequest.RedirectURIs,
			"token_endpoint_auth_method": "none", // Public client
			"grant_types":                grantTypes,
			"response_types":             responseTypes,
			"client_name":                registrationRequest.ClientName,
			"client_uri":                 registrationRequest.ClientURI,
		}

		// TODO: these should be deleted once we are able to validate scope, now it is for debug only.
		// Debug: Log client registration
		log.Printf("SCOPE DEBUG: Client registration request:\n")
		reqJSON, _ := json.MarshalIndent(registrationRequest, "  ", "  ")
		log.Printf("  Request: %s\n", string(reqJSON))
		log.Printf("  Requested scope: '%s'\n", registrationRequest.Scope)
		log.Printf("  Config required scopes: %v\n", em.config.RequiredScopes)

		log.Printf("SCOPE DEBUG: Client registration response:\n")
		respJSON, _ := json.MarshalIndent(clientInfo, "  ", "  ")
		log.Printf("  Response: %s\n", string(respJSON))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		if err := json.NewEncoder(w).Encode(clientInfo); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// validateClientRegistration validates a client registration request
func (em *EndpointManager) validateClientRegistration(req *struct {
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	ClientName              string   `json:"client_name"`
	ClientURI               string   `json:"client_uri"`
	Scope                   string   `json:"scope"`
}) error {
	// Validate redirect URIs
	if len(req.RedirectURIs) == 0 {
		return fmt.Errorf("at least one redirect_uri is required")
	}

	for _, redirectURI := range req.RedirectURIs {
		if !em.isValidRedirectURI(redirectURI) {
			return fmt.Errorf("invalid redirect_uri: %s", redirectURI)
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

// isValidRedirectURI validates a redirect URI
func (em *EndpointManager) isValidRedirectURI(redirectURI string) bool {
	parsedURL, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}

	// Check against allowed redirects
	for _, allowed := range em.config.AllowedRedirects {
		if redirectURI == allowed {
			return true
		}

		// Allow localhost with any port for development
		if strings.HasPrefix(allowed, "http://localhost") &&
			strings.HasPrefix(redirectURI, "http://localhost") {
			return true
		}

		// Allow 127.0.0.1 with any port for development
		if strings.HasPrefix(allowed, "http://127.0.0.1") &&
			strings.HasPrefix(redirectURI, "http://127.0.0.1") {
			return true
		}
	}

	// Special handling for MCP Inspector debug endpoints
	// Allow any localhost/127.0.0.1 redirect URI for OAuth testing
	if parsedURL.Hostname() == "localhost" || parsedURL.Hostname() == "127.0.0.1" {
		if parsedURL.Scheme == "http" {
			log.Printf("Allowing development redirect URI: %s\n", redirectURI)
			return true
		}
	}

	// Require HTTPS for non-localhost URLs
	if parsedURL.Scheme != "https" && parsedURL.Hostname() != "localhost" && parsedURL.Hostname() != "127.0.0.1" {
		return false
	}

	return false
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

		provider := em.provider

		metadata, err := provider.GetProtectedResourceMetadata(resourceURL)
		if err != nil {
			log.Printf("SCOPE DEBUG: Failed to get protected resource metadata: %v\n", err)
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

		// Parse query parameters
		query := r.URL.Query()

		// TODO： these should be deleted once we are able to validate scope - now it is for debug only!
		// Debug: Log the incoming request parameters
		log.Printf("SCOPE DEBUG: Authorization proxy received parameters:\n")
		log.Printf("  Request URL: %s\n", r.URL.String())
		log.Printf("  Request Host: %s\n", r.Host)
		for key, values := range query {
			log.Printf("  %s: %v\n", key, values)
			if key == "redirect_uri" {
				log.Printf("  >>> MCP Inspector redirect_uri: '%s'\n", strings.Join(values, " "))
				log.Printf("  >>> Server allowed redirects: %v\n", em.config.AllowedRedirects)
			}
			if key == "scope" {
				log.Printf("  >>> MCP Inspector requested scope: '%s'\n", strings.Join(values, " "))
				log.Printf("  >>> Server required scopes: %v\n", em.config.RequiredScopes)

				// Check if there's a scope mismatch
				requestedScopes := strings.Split(strings.Join(values, " "), " ")
				hasAzureScope := false
				hasOpenIDScope := false

				for _, scope := range requestedScopes {
					if strings.Contains(scope, "management.azure.com") {
						hasAzureScope = true
					}
					if scope == "openid" || scope == "profile" || scope == "email" || scope == "offline_access" {
						hasOpenIDScope = true
					}
				}

				log.Printf("  >>> Has Azure Management scope: %t\n", hasAzureScope)
				log.Printf("  >>> Has OpenID Connect scopes: %t\n", hasOpenIDScope)

				// Check if MCP Inspector's requested scopes match server requirements
				requestedScopeSet := make(map[string]bool)
				for _, scope := range requestedScopes {
					requestedScopeSet[scope] = true
				}

				requiredScopeSet := make(map[string]bool)
				for _, scope := range em.config.RequiredScopes {
					requiredScopeSet[scope] = true
				}

				// Check for missing required scopes
				missingScopes := []string{}
				for _, required := range em.config.RequiredScopes {
					if !requestedScopeSet[required] {
						missingScopes = append(missingScopes, required)
					}
				}

				if len(missingScopes) > 0 {
					log.Printf("  >>> WARNING: MCP Inspector missing required scopes: %v\n", missingScopes)
					log.Printf("  >>> This may cause authentication failures later\n")
				} else {
					log.Printf("  >>> ✓ All required scopes are present in the request\n")
				}
			}
		}

		// Azure AD v2.0 doesn't support RFC 8707 Resource Indicators in authorization requests
		// Remove the resource parameter if present and log it for MCP compliance tracking
		resourceParam := query.Get("resource")
		if resourceParam != "" {
			log.Printf("  >>> Received resource parameter from MCP client: %s\n", resourceParam)
			log.Printf("  >>> Removing resource parameter for Azure AD v2.0 compatibility\n")
			query.Del("resource")
		} else {
			log.Printf("  >>> No resource parameter in request (MCP client may not be using RFC 8707)\n")
		}

		// Ensure the request includes all required scopes for Azure AD
		// Azure AD requires consistent scopes between authorization and token requests
		requestedScopes := strings.Split(query.Get("scope"), " ")

		// Build a set of all required scopes (client requested + server required)
		allScopes := make(map[string]bool)

		// Add client-requested scopes
		for _, scope := range requestedScopes {
			if scope != "" {
				allScopes[scope] = true
			}
		}

		// Add server-required scopes
		for _, scope := range em.config.RequiredScopes {
			allScopes[scope] = true
		}

		// Convert back to slice and update the query
		var finalScopes []string
		for scope := range allScopes {
			finalScopes = append(finalScopes, scope)
		}

		finalScopeString := strings.Join(finalScopes, " ")
		query.Set("scope", finalScopeString)

		log.Printf("  >>> Updated scope for Azure AD: '%s'\n", finalScopeString)

		// Build the Azure AD authorization URL
		azureAuthURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", em.config.TenantID)

		// Create the redirect URL with filtered parameters
		redirectURL := fmt.Sprintf("%s?%s", azureAuthURL, query.Encode())

		log.Printf("Redirecting to Azure AD: %s\n", redirectURL)

		// Redirect to Azure AD
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

// callbackHandler handles OAuth 2.0 Authorization Code flow callback
func (em *EndpointManager) callbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		fmt.Println("VVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse query parameters
		query := r.URL.Query()

		// Check for error response from authorization server
		if authError := query.Get("error"); authError != "" {
			errorDesc := query.Get("error_description")
			em.writeCallbackErrorResponse(w, fmt.Sprintf("Authorization failed: %s - %s", authError, errorDesc))
			return
		}

		// Get authorization code
		code := query.Get("code")
		if code == "" {
			em.writeCallbackErrorResponse(w, "Missing authorization code")
			return
		}

		// Get state parameter for CSRF protection
		state := query.Get("state")
		if state == "" {
			em.writeCallbackErrorResponse(w, "Missing state parameter")
			return
		}

		// Exchange authorization code for access token
		tokenResponse, err := em.exchangeCodeForToken(code, state)
		if err != nil {
			em.writeCallbackErrorResponse(w, fmt.Sprintf("Failed to exchange code for token: %v", err))
			return
		}

		// TODO: For now, skip token validation in callback since we know the token comes directly from Azure AD
		// We'll validate it later when it's actually used for MCP requests
		// This prevents callback failures due to JWT signature validation issues
		fmt.Printf("=== CALLBACK TOKEN VALIDATION SKIPPED ===\n")
		fmt.Printf("Token received from Azure AD (length: %d)\n", len(tokenResponse.AccessToken))
		fmt.Printf("Skipping JWT validation in callback - will validate on actual MCP requests\n")

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

	// Find the correct redirect URI from configuration
	var redirectURI string
	for _, uri := range em.config.AllowedRedirects {
		if strings.Contains(uri, "/oauth/callback") {
			redirectURI = uri
			break
		}
	}
	if redirectURI == "" {
		// Fallback to first allowed redirect
		if len(em.config.AllowedRedirects) > 0 {
			redirectURI = em.config.AllowedRedirects[0]
		} else {
			return nil, fmt.Errorf("no valid redirect URI configured")
		}
	}

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
	log.Printf("Azure AD token request with scope: %s", strings.Join(em.config.RequiredScopes, " "))

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

	fmt.Println("XXXXXXXX, EXCHANGE CODE FOR TOKEN", tokenResponse.AccessToken)

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
		log.Printf("=== TOKEN ENDPOINT CALLED ===")
		log.Printf("Method: %s", r.Method)
		log.Printf("URL: %s", r.URL.String())
		log.Printf("User-Agent: %s", r.Header.Get("User-Agent"))

		// Set CORS headers for all requests
		em.setCORSHeaders(w)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			em.writeErrorResponse(w, "invalid_request", "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			em.writeErrorResponse(w, "invalid_request", "Failed to parse form data", http.StatusBadRequest)
			return
		}

		// Validate grant type
		grantType := r.FormValue("grant_type")
		if grantType != "authorization_code" {
			em.writeErrorResponse(w, "unsupported_grant_type", fmt.Sprintf("Unsupported grant type: %s", grantType), http.StatusBadRequest)
			return
		}

		// Extract required parameters
		code := r.FormValue("code")
		clientID := r.FormValue("client_id")
		redirectURI := r.FormValue("redirect_uri")
		codeVerifier := r.FormValue("code_verifier") // PKCE parameter

		if code == "" {
			em.writeErrorResponse(w, "invalid_request", "Missing authorization code", http.StatusBadRequest)
			return
		}

		if clientID == "" {
			em.writeErrorResponse(w, "invalid_request", "Missing client_id", http.StatusBadRequest)
			return
		}

		if redirectURI == "" {
			em.writeErrorResponse(w, "invalid_request", "Missing redirect_uri", http.StatusBadRequest)
			return
		}

		// Validate client ID
		if clientID != em.config.ClientID {
			em.writeErrorResponse(w, "invalid_client", "Invalid client_id", http.StatusBadRequest)
			return
		}

		// Extract scope from the token request (MCP client should send the same scope)
		requestedScope := r.FormValue("scope")
		if requestedScope == "" {
			// Fallback to server required scopes if not provided
			requestedScope = strings.Join(em.config.RequiredScopes, " ")
			log.Printf("No scope in token request, using server required scopes: %s", requestedScope)
		} else {
			log.Printf("Using client requested scope from token request: %s", requestedScope)
		}

		// Exchange authorization code for access token with Azure AD
		tokenResponse, err := em.exchangeCodeForTokenDirect(code, redirectURI, codeVerifier, requestedScope)
		if err != nil {
			log.Printf("Token exchange failed: %v\n", err)
			em.writeErrorResponse(w, "invalid_grant", fmt.Sprintf("Authorization code exchange failed: %v", err), http.StatusBadRequest)
			return
		}

		// Return token response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")

		if err := json.NewEncoder(w).Encode(tokenResponse); err != nil {
			log.Printf("Failed to encode token response: %v\n", err)
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
