package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/auth"
)

func TestEndpointManager_RegisterEndpoints(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false,
			ValidateAudience: false,
			ExpectedAudience: "https://management.azure.com/",
		},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	mux := http.NewServeMux()
	manager.RegisterEndpoints(mux)

	// Test that endpoints are registered by making requests
	testCases := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/.well-known/oauth-protected-resource", http.StatusOK},
		{"GET", "/.well-known/oauth-authorization-server", http.StatusInternalServerError}, // Will fail without real Azure AD
		{"POST", "/oauth/register", http.StatusBadRequest},                                 // Missing required data
		{"POST", "/oauth/introspect", http.StatusBadRequest},                               // Missing token param
		{"GET", "/oauth/callback", http.StatusBadRequest},                                  // Missing required params
		{"GET", "/health", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tc.status {
				t.Errorf("Expected status %d for %s %s, got %d", tc.status, tc.method, tc.path, w.Code)
			}
		})
	}
}

func TestProtectedResourceMetadataEndpoint(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	req := httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)
	w := httptest.NewRecorder()

	handler := manager.protectedResourceMetadataHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var metadata ProtectedResourceMetadata
	if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	expectedAuthServer := "http://example.com"
	if len(metadata.AuthorizationServers) != 1 || metadata.AuthorizationServers[0] != expectedAuthServer {
		t.Errorf("Expected auth server %s, got %v", expectedAuthServer, metadata.AuthorizationServers)
	}

	if len(metadata.ScopesSupported) != 1 || metadata.ScopesSupported[0] != "https://management.azure.com/.default" {
		t.Errorf("Expected scopes %v, got %v", config.RequiredScopes, metadata.ScopesSupported)
	}
}

func TestClientRegistrationEndpoint(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	// Test valid registration request
	registrationRequest := map[string]interface{}{
		"redirect_uris":              []string{"http://localhost:3000/callback"},
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
		"scope":                      "https://management.azure.com/.default",
		"client_name":                "Test Client",
	}

	reqBody, _ := json.Marshal(registrationRequest)
	req := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler := manager.clientRegistrationHandler()
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["client_id"] == "" {
		t.Error("Expected client_id in response")
	}

	redirectURIs, ok := response["redirect_uris"].([]interface{})
	if !ok || len(redirectURIs) != 1 {
		t.Errorf("Expected redirect URIs in response")
	}
}

func TestTokenIntrospectionEndpoint(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false,
			ValidateAudience: false,
		},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	// Test with valid token (since JWT validation is disabled, any token works)
	// Note: Must use a token that looks like a JWT (has dots) to pass initial format checks
	req := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader("token=header.payload.signature"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	handler := manager.tokenIntrospectionHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if active, ok := response["active"].(bool); !ok || !active {
		t.Error("Expected active token")
	}
}

func TestTokenIntrospectionEndpointMissingToken(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	// Test without token parameter
	req := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	handler := manager.tokenIntrospectionHandler()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing token, got %d", w.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := manager.healthHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %v", response["status"])
	}

	oauth, ok := response["oauth"].(map[string]interface{})
	if !ok {
		t.Error("Expected oauth object in response")
	}

	if oauth["enabled"] != true {
		t.Errorf("Expected oauth enabled true, got %v", oauth["enabled"])
	}
}

func TestValidateClientRegistration(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	tests := []struct {
		name    string
		request map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid request",
			request: map[string]interface{}{
				"redirect_uris":  []string{"http://localhost:3000/callback"},
				"grant_types":    []string{"authorization_code"},
				"response_types": []string{"code"},
			},
			wantErr: false,
		},
		{
			name: "missing redirect URIs",
			request: map[string]interface{}{
				"grant_types":    []string{"authorization_code"},
				"response_types": []string{"code"},
			},
			wantErr: true,
		},
		{
			name: "invalid grant type",
			request: map[string]interface{}{
				"redirect_uris":  []string{"http://localhost:3000/callback"},
				"grant_types":    []string{"client_credentials"},
				"response_types": []string{"code"},
			},
			wantErr: true,
		},
		{
			name: "invalid response type",
			request: map[string]interface{}{
				"redirect_uris":  []string{"http://localhost:3000/callback"},
				"grant_types":    []string{"authorization_code"},
				"response_types": []string{"token"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert test request to the expected struct format
			req := &ClientRegistrationRequest{}

			if redirectURIs, ok := tt.request["redirect_uris"].([]string); ok {
				req.RedirectURIs = redirectURIs
			}
			if grantTypes, ok := tt.request["grant_types"].([]string); ok {
				req.GrantTypes = grantTypes
			}
			if responseTypes, ok := tt.request["response_types"].([]string); ok {
				req.ResponseTypes = responseTypes
			}

			err := manager.validateClientRegistration(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateClientRegistration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallbackEndpointMissingCode(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	// Test callback without authorization code
	req := httptest.NewRequest("GET", "/oauth/callback?state=test-state", nil)
	w := httptest.NewRecorder()

	handler := manager.callbackHandler()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing code, got %d", w.Code)
	}

	// Check that response contains HTML error page
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected HTML content type, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Missing authorization code") {
		t.Error("Expected error message about missing authorization code")
	}
}

func TestCallbackEndpointMissingState(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	// Test callback without state parameter
	req := httptest.NewRequest("GET", "/oauth/callback?code=test-code", nil)
	w := httptest.NewRecorder()

	handler := manager.callbackHandler()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing state, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Missing state parameter") {
		t.Error("Expected error message about missing state parameter")
	}
}

func TestCallbackEndpointAuthError(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	// Test callback with authorization error
	req := httptest.NewRequest("GET", "/oauth/callback?error=access_denied&error_description=User%20denied%20access", nil)
	w := httptest.NewRecorder()

	handler := manager.callbackHandler()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for auth error, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Authorization failed") {
		t.Error("Expected error message about authorization failure")
	}
	if !strings.Contains(body, "access_denied") {
		t.Error("Expected specific error code in response")
	}
}

func TestCallbackEndpointMethodNotAllowed(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
	}

	provider, _ := NewAzureOAuthProvider(config)
	manager := NewEndpointManager(provider, config)

	// Test callback with POST method (should only accept GET)
	req := httptest.NewRequest("POST", "/oauth/callback", nil)
	w := httptest.NewRecorder()

	handler := manager.callbackHandler()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for POST method, got %d", w.Code)
	}
}
