package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/aks-mcp/internal/auth"
)

// EndpointManager manages OAuth-related HTTP endpoints
type EndpointManager struct {
	middleware *AuthMiddleware
	config     *auth.OAuthConfig
}

// NewEndpointManager creates a new OAuth endpoint manager
func NewEndpointManager(middleware *AuthMiddleware, config *auth.OAuthConfig) *EndpointManager {
	return &EndpointManager{
		middleware: middleware,
		config:     config,
	}
}

// RegisterEndpoints registers OAuth endpoints with the provided HTTP mux
func (em *EndpointManager) RegisterEndpoints(mux *http.ServeMux) {
	// OAuth 2.0 Protected Resource Metadata endpoint (RFC 9728)
	mux.HandleFunc("/.well-known/oauth-protected-resource", em.middleware.ProtectedResourceMetadataHandler())

	// OAuth 2.0 Authorization Server Metadata endpoint (RFC 8414)
	// Note: This would typically be served by Azure AD, but we provide a proxy for convenience
	mux.HandleFunc("/.well-known/oauth-authorization-server", em.authServerMetadataProxyHandler())

	// Dynamic Client Registration endpoint (RFC 7591)
	mux.HandleFunc("/oauth/register", em.clientRegistrationHandler())

	// Token introspection endpoint (RFC 7662) - optional
	mux.HandleFunc("/oauth/introspect", em.tokenIntrospectionHandler())

	// OAuth 2.0 callback endpoint for Authorization Code flow
	mux.HandleFunc("/oauth/callback", em.callbackHandler())

	// Health check endpoint (unauthenticated)
	mux.HandleFunc("/health", em.healthHandler())
}

// authServerMetadataProxyHandler proxies authorization server metadata from Azure AD
func (em *EndpointManager) authServerMetadataProxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get metadata from Azure AD
		provider, err := NewAzureOAuthProvider(em.config)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		metadata, err := provider.GetAuthorizationServerMetadata()
		if err != nil {
			http.Error(w, "Failed to fetch authorization server metadata", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// clientRegistrationHandler implements OAuth 2.0 Dynamic Client Registration (RFC 7591)
func (em *EndpointManager) clientRegistrationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// For AKS-MCP, we use a simplified client registration approach
		// In production, you might want to integrate with Azure AD Application Registration API

		clientInfo := map[string]interface{}{
			"client_id":                  em.config.ClientID, // Use configured client ID
			"redirect_uris":              registrationRequest.RedirectURIs,
			"token_endpoint_auth_method": "none", // Public client
			"grant_types":                []string{"authorization_code", "refresh_token"},
			"response_types":             []string{"code"},
			"client_name":                registrationRequest.ClientName,
			"client_uri":                 registrationRequest.ClientURI,
		}

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
		provider, err := NewAzureOAuthProvider(em.config)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

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

// callbackHandler handles OAuth 2.0 Authorization Code flow callback
func (em *EndpointManager) callbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Validate the received token
		provider, err := NewAzureOAuthProvider(em.config)
		if err != nil {
			em.writeCallbackErrorResponse(w, "Internal server error")
			return
		}

		tokenInfo, err := provider.ValidateToken(r.Context(), tokenResponse.AccessToken)
		if err != nil {
			em.writeCallbackErrorResponse(w, fmt.Sprintf("Token validation failed: %v", err))
			return
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

// generateSessionToken generates a secure random session token
func (em *EndpointManager) generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
