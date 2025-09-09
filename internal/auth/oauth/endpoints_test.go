package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/auth"
	"github.com/Azure/aks-mcp/internal/config"
)

// createTestConfig creates a test ConfigData with OAuth configuration
func createTestConfig() *config.ConfigData {
	cfg := config.NewConfig()
	cfg.Host = "127.0.0.1"
	cfg.Port = 8000
	cfg.OAuthConfig = &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		RedirectURIs:   []string{"http://127.0.0.1:8000/oauth/callback", "http://localhost:8000/oauth/callback"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false,
			ValidateAudience: false,
			ExpectedAudience: "https://management.azure.com/",
		},
	}
	return cfg
}

func TestEndpointManager_RegisterEndpoints(t *testing.T) {
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
		t.Errorf("Expected scopes %v, got %v", cfg.OAuthConfig.RequiredScopes, metadata.ScopesSupported)
	}
}

func TestClientRegistrationEndpoint(t *testing.T) {
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

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
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

	// Test callback with POST method (should only accept GET)
	req := httptest.NewRequest("POST", "/oauth/callback", nil)
	w := httptest.NewRecorder()

	handler := manager.callbackHandler()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for POST method, got %d", w.Code)
	}
}

func TestValidateRedirectURI(t *testing.T) {
	cfg := createTestConfig()

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

	tests := []struct {
		name        string
		redirectURI string
		wantErr     bool
	}{
		{
			name:        "valid redirect URI - 127.0.0.1",
			redirectURI: "http://127.0.0.1:8000/oauth/callback",
			wantErr:     false,
		},
		{
			name:        "valid redirect URI - localhost",
			redirectURI: "http://localhost:8000/oauth/callback",
			wantErr:     false,
		},
		{
			name:        "invalid redirect URI - wrong port",
			redirectURI: "http://127.0.0.1:9000/oauth/callback",
			wantErr:     true,
		},
		{
			name:        "invalid redirect URI - wrong path",
			redirectURI: "http://127.0.0.1:8000/oauth/malicious",
			wantErr:     true,
		},
		{
			name:        "invalid redirect URI - external domain",
			redirectURI: "http://malicious.com:8000/oauth/callback",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateRedirectURI(tt.redirectURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRedirectURI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test with empty redirect URIs configuration
	cfgEmpty := createTestConfig()
	cfgEmpty.OAuthConfig.RedirectURIs = []string{}
	managerEmpty := NewEndpointManager(provider, cfgEmpty)

	err := managerEmpty.validateRedirectURI("http://127.0.0.1:8000/oauth/callback")
	if err == nil {
		t.Error("Expected error when no redirect URIs are configured")
	}
}

// TestAuthorizationProxyRedirectURIValidation tests the authorization endpoint redirect URI validation
func TestCORSHeaders(t *testing.T) {
	cfg := createTestConfig()
	cfg.OAuthConfig.AllowedOrigins = []string{"http://localhost:6274"}

	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

	tests := []struct {
		name          string
		origin        string
		expectCORSSet bool
		expectOrigin  string
	}{
		{
			name:          "allowed origin",
			origin:        "http://localhost:6274",
			expectCORSSet: true,
			expectOrigin:  "http://localhost:6274",
		},
		{
			name:          "disallowed origin",
			origin:        "http://malicious.com",
			expectCORSSet: false,
			expectOrigin:  "",
		},
		{
			name:          "no origin header",
			origin:        "",
			expectCORSSet: false,
			expectOrigin:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			handler := manager.healthHandler()
			handler(w, req)

			corsOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORSSet {
				if corsOrigin != tt.expectOrigin {
					t.Errorf("Expected CORS origin %s, got %s", tt.expectOrigin, corsOrigin)
				}
			} else {
				if corsOrigin != "" {
					t.Errorf("Expected no CORS headers, but got Access-Control-Allow-Origin: %s", corsOrigin)
				}
			}
		})
	}
}

func TestAuthorizationProxyRedirectURIValidation(t *testing.T) {
	cfg := createTestConfig()
	provider, _ := NewAzureOAuthProvider(cfg.OAuthConfig)
	manager := NewEndpointManager(provider, cfg)

	tests := []struct {
		name        string
		redirectURI string
		expectError bool
		expectCode  int
	}{
		{
			name:        "missing redirect_uri",
			redirectURI: "",
			expectError: true,
			expectCode:  http.StatusBadRequest,
		},
		{
			name:        "valid redirect_uri - 127.0.0.1",
			redirectURI: "http://127.0.0.1:8000/oauth/callback",
			expectError: false,
			expectCode:  http.StatusFound, // Should redirect to Azure AD
		},
		{
			name:        "valid redirect_uri - localhost",
			redirectURI: "http://localhost:8000/oauth/callback",
			expectError: false,
			expectCode:  http.StatusFound, // Should redirect to Azure AD
		},
		{
			name:        "invalid redirect_uri - wrong port",
			redirectURI: "http://127.0.0.1:9000/oauth/callback",
			expectError: true,
			expectCode:  http.StatusBadRequest,
		},
		{
			name:        "invalid redirect_uri - wrong path",
			redirectURI: "http://127.0.0.1:8000/oauth/malicious",
			expectError: true,
			expectCode:  http.StatusBadRequest,
		},
		{
			name:        "invalid redirect_uri - external domain",
			redirectURI: "http://malicious.com:8000/oauth/callback",
			expectError: true,
			expectCode:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request URL with redirect_uri parameter if provided
			requestURL := "/oauth2/v2.0/authorize?response_type=code&client_id=test-client&code_challenge=test&code_challenge_method=S256&state=test"
			if tt.redirectURI != "" {
				requestURL += "&redirect_uri=" + tt.redirectURI
			}

			req := httptest.NewRequest("GET", requestURL, nil)
			w := httptest.NewRecorder()

			handler := manager.authorizationProxyHandler()
			handler(w, req)

			if tt.expectError {
				if w.Code != tt.expectCode {
					t.Errorf("Expected status code %d, got %d", tt.expectCode, w.Code)
				}

				// Check that error response contains helpful information
				body := w.Body.String()
				if !strings.Contains(body, "redirect_uri") {
					t.Errorf("Error response should mention redirect_uri, got: %s", body)
				}
			} else {
				if w.Code != tt.expectCode {
					t.Errorf("Expected status code %d, got %d", tt.expectCode, w.Code)
				}

				// For successful cases, check redirect location contains expected parameters
				location := w.Header().Get("Location")
				if location == "" {
					t.Errorf("Expected redirect location header, got empty")
				}
				if !strings.Contains(location, "login.microsoftonline.com") {
					t.Errorf("Expected redirect to Azure AD, got: %s", location)
				}
			}
		})
	}
}
