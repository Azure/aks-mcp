package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
)

func TestNewAzureOAuthProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *auth.OAuthConfig
		wantErr bool
	}{
		{
			name: "valid config should create provider",
			config: &auth.OAuthConfig{
				Enabled:        true,
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				RequiredScopes: []string{"https://management.azure.com/.default"},
				TokenValidation: auth.TokenValidationConfig{
					ValidateJWT:      true,
					ValidateAudience: true,
					ExpectedAudience: "https://management.azure.com/",
					CacheTTL:         5 * time.Minute,
					ClockSkew:        1 * time.Minute,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config should fail",
			config: &auth.OAuthConfig{
				Enabled: true,
				// Missing required fields
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAzureOAuthProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAzureOAuthProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewAzureOAuthProvider() returned nil provider")
			}
		})
	}
}

func TestGetProtectedResourceMetadata(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant-id",
		ClientID:       "test-client-id",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      true,
			ValidateAudience: true,
			ExpectedAudience: "https://management.azure.com/",
			CacheTTL:         5 * time.Minute,
			ClockSkew:        1 * time.Minute,
		},
	}

	provider, err := NewAzureOAuthProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	serverURL := "http://localhost:8000"
	metadata, err := provider.GetProtectedResourceMetadata(serverURL)
	if err != nil {
		t.Fatalf("GetProtectedResourceMetadata() error = %v", err)
	}

	expectedAuthServer := "http://localhost:8000"
	if len(metadata.AuthorizationServers) != 1 || metadata.AuthorizationServers[0] != expectedAuthServer {
		t.Errorf("Expected authorization server %s, got %v", expectedAuthServer, metadata.AuthorizationServers)
	}

	// Note: AzureADProtectedResourceMetadata doesn't include a Resource field.
	// The resource URL is implied by the context of the request endpoint.

	if len(metadata.ScopesSupported) != 1 || metadata.ScopesSupported[0] != "https://management.azure.com/.default" {
		t.Errorf("Expected scopes %v, got %v", config.RequiredScopes, metadata.ScopesSupported)
	}
}

func TestGetAuthorizationServerMetadataWithDefaults(t *testing.T) {
	// Create a mock Azure AD metadata endpoint that's missing some fields
	// This simulates the case where Azure AD doesn't provide all required fields
	mockMetadata := AzureADMetadata{
		Issuer:                "https://login.microsoftonline.com/test-tenant/v2.0",
		AuthorizationEndpoint: "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/authorize",
		TokenEndpoint:         "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/token",
		JWKSUri:               "https://login.microsoftonline.com/test-tenant/discovery/v2.0/keys",
		ScopesSupported:       []string{"openid", "profile", "email"},
		// Intentionally omit GrantTypesSupported, ResponseTypesSupported, etc.
		// to test our default value logic
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockMetadata); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      true,
			ValidateAudience: true,
			ExpectedAudience: "https://management.azure.com/",
			CacheTTL:         5 * time.Minute,
			ClockSkew:        1 * time.Minute,
		},
	}

	provider, err := NewAzureOAuthProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Override the HTTP client to use our test server
	provider.httpClient = &http.Client{
		Transport: &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				// Redirect all requests to our test server
				req.URL.Scheme = "http"
				req.URL.Host = server.URL[7:] // Remove "http://"
				req.URL.Path = "/"
				return http.DefaultTransport.RoundTrip(req)
			},
		},
	}

	metadata, err := provider.GetAuthorizationServerMetadata(server.URL)
	if err != nil {
		t.Fatalf("GetAuthorizationServerMetadata() error = %v", err)
	}

	// Verify that default values were populated for missing fields
	expectedGrantTypes := []string{"authorization_code", "refresh_token"}
	if len(metadata.GrantTypesSupported) != len(expectedGrantTypes) {
		t.Errorf("Expected %d grant types, got %d", len(expectedGrantTypes), len(metadata.GrantTypesSupported))
	}
	for i, expected := range expectedGrantTypes {
		if i >= len(metadata.GrantTypesSupported) || metadata.GrantTypesSupported[i] != expected {
			t.Errorf("Expected grant type %s at index %d, got %v", expected, i, metadata.GrantTypesSupported)
		}
	}

	expectedResponseTypes := []string{"code"}
	if len(metadata.ResponseTypesSupported) != len(expectedResponseTypes) {
		t.Errorf("Expected %d response types, got %d", len(expectedResponseTypes), len(metadata.ResponseTypesSupported))
	}
	if len(metadata.ResponseTypesSupported) > 0 && metadata.ResponseTypesSupported[0] != "code" {
		t.Errorf("Expected response type 'code', got %s", metadata.ResponseTypesSupported[0])
	}

	expectedSubjectTypes := []string{"public"}
	if len(metadata.SubjectTypesSupported) != len(expectedSubjectTypes) {
		t.Errorf("Expected %d subject types, got %d", len(expectedSubjectTypes), len(metadata.SubjectTypesSupported))
	}
	if len(metadata.SubjectTypesSupported) > 0 && metadata.SubjectTypesSupported[0] != "public" {
		t.Errorf("Expected subject type 'public', got %s", metadata.SubjectTypesSupported[0])
	}

	expectedTokenEndpointAuthMethods := []string{"none"}
	if len(metadata.TokenEndpointAuthMethodsSupported) != len(expectedTokenEndpointAuthMethods) {
		t.Errorf("Expected %d auth methods, got %d", len(expectedTokenEndpointAuthMethods), len(metadata.TokenEndpointAuthMethodsSupported))
	}
	if len(metadata.TokenEndpointAuthMethodsSupported) > 0 && metadata.TokenEndpointAuthMethodsSupported[0] != "none" {
		t.Errorf("Expected auth method 'none', got %s", metadata.TokenEndpointAuthMethodsSupported[0])
	}

	// Verify that PKCE is properly configured
	expectedCodeChallengeMethods := []string{"S256"}
	if len(metadata.CodeChallengeMethodsSupported) != len(expectedCodeChallengeMethods) {
		t.Errorf("Expected %d code challenge methods, got %d", len(expectedCodeChallengeMethods), len(metadata.CodeChallengeMethodsSupported))
	}
	if len(metadata.CodeChallengeMethodsSupported) > 0 && metadata.CodeChallengeMethodsSupported[0] != "S256" {
		t.Errorf("Expected code challenge method 'S256', got %s", metadata.CodeChallengeMethodsSupported[0])
	}
}

func TestGetAuthorizationServerMetadata(t *testing.T) {
	// Create a mock Azure AD metadata endpoint
	mockMetadata := AzureADMetadata{
		Issuer:                "https://login.microsoftonline.com/test-tenant/v2.0",
		AuthorizationEndpoint: "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/authorize",
		TokenEndpoint:         "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/token",
		JWKSUri:               "https://login.microsoftonline.com/test-tenant/discovery/v2.0/keys",
		ScopesSupported:       []string{"openid", "profile", "email"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockMetadata); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      true,
			ValidateAudience: true,
			ExpectedAudience: "https://management.azure.com/",
			CacheTTL:         5 * time.Minute,
			ClockSkew:        1 * time.Minute,
		},
	}

	provider, err := NewAzureOAuthProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Override the HTTP client to use our test server
	provider.httpClient = &http.Client{
		Transport: &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				// Redirect all requests to our test server
				req.URL.Scheme = "http"
				req.URL.Host = server.URL[7:] // Remove "http://"
				req.URL.Path = "/"
				return http.DefaultTransport.RoundTrip(req)
			},
		},
	}

	metadata, err := provider.GetAuthorizationServerMetadata(server.URL)
	if err != nil {
		t.Fatalf("GetAuthorizationServerMetadata() error = %v", err)
	}

	if metadata.Issuer != mockMetadata.Issuer {
		t.Errorf("Expected issuer %s, got %s", mockMetadata.Issuer, metadata.Issuer)
	}

	expectedAuthEndpoint := fmt.Sprintf("%s/oauth2/v2.0/authorize", server.URL)
	if metadata.AuthorizationEndpoint != expectedAuthEndpoint {
		t.Errorf("Expected auth endpoint %s, got %s", expectedAuthEndpoint, metadata.AuthorizationEndpoint)
	}
}

func TestValidateTokenWithoutJWT(t *testing.T) {
	// SECURITY WARNING: This test verifies the JWT validation bypass functionality
	// ValidateJWT=false should ONLY be used in development/testing environments
	// This functionality should NEVER be enabled in production
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false, // Disable JWT validation
			ValidateAudience: false,
			ExpectedAudience: "https://management.azure.com/",
			CacheTTL:         5 * time.Minute,
			ClockSkew:        1 * time.Minute,
		},
	}

	provider, err := NewAzureOAuthProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	// Use a token that looks like a JWT to pass initial format checks
	testToken := "header.payload.signature"
	tokenInfo, err := provider.ValidateToken(ctx, testToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if tokenInfo.AccessToken != testToken {
		t.Errorf("Expected access token %s, got %s", testToken, tokenInfo.AccessToken)
	}

	if tokenInfo.TokenType != "Bearer" {
		t.Errorf("Expected token type Bearer, got %s", tokenInfo.TokenType)
	}
}

func TestValidateAudience(t *testing.T) {
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client-id",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      true,
			ValidateAudience: true,
			ExpectedAudience: "https://management.azure.com/",
			CacheTTL:         5 * time.Minute,
			ClockSkew:        1 * time.Minute,
		},
	}

	provider, err := NewAzureOAuthProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name    string
		claims  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid audience string",
			claims: map[string]interface{}{
				"aud": "https://management.azure.com/",
			},
			wantErr: false,
		},
		{
			name: "valid client ID audience",
			claims: map[string]interface{}{
				"aud": "test-client-id",
			},
			wantErr: false,
		},
		{
			name: "valid audience array",
			claims: map[string]interface{}{
				"aud": []interface{}{"https://management.azure.com/", "other-aud"},
			},
			wantErr: false,
		},
		{
			name: "invalid audience",
			claims: map[string]interface{}{
				"aud": "invalid-audience",
			},
			wantErr: true,
		},
		{
			name: "missing audience",
			claims: map[string]interface{}{
				"sub": "user123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.validateAudience(tt.claims)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAudience() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// roundTripperFunc is a helper type for creating custom HTTP transports in tests
type roundTripperFunc struct {
	fn func(*http.Request) (*http.Response, error)
}

func (f *roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f.fn(req)
}

func oboConfig() *auth.OAuthConfig {
	return &auth.OAuthConfig{
		Enabled:      true,
		TenantID:     "test-tenant",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false,
			ValidateAudience: false,
			ExpectedAudience: "https://management.azure.com/",
			CacheTTL:         5 * time.Minute,
			ClockSkew:        1 * time.Minute,
		},
	}
}

func newProviderWithTransport(t *testing.T, cfg *auth.OAuthConfig, handler http.Handler) *AzureOAuthProvider {
	t.Helper()
	p, err := NewAzureOAuthProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	p.httpClient = &http.Client{
		Transport: &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				return w.Result(), nil
			},
		},
	}
	return p
}

func TestExchangeOBO(t *testing.T) {
	tests := []struct {
		name          string
		clientSecret  string
		scope         string
		handlerStatus int
		handlerBody   string
		wantToken     string
		wantErrSubstr string
	}{
		{
			name:          "success returns access token",
			clientSecret:  "secret",
			scope:         "https://management.azure.com/user_impersonation",
			handlerStatus: http.StatusOK,
			handlerBody:   `{"access_token":"arm-token-xyz","token_type":"Bearer"}`,
			wantToken:     "arm-token-xyz",
		},
		{
			name:          "success with cluster scope",
			clientSecret:  "secret",
			scope:         "6dae42f8-4368-4678-94ff-3960e28e3630/.default",
			handlerStatus: http.StatusOK,
			handlerBody:   `{"access_token":"cluster-token-abc","token_type":"Bearer"}`,
			wantToken:     "cluster-token-abc",
		},
		{
			name:          "missing client secret errors immediately",
			clientSecret:  "",
			scope:         "https://management.azure.com/user_impersonation",
			wantErrSubstr: "client secret",
		},
		{
			name:          "azure ad error response",
			clientSecret:  "secret",
			scope:         "https://management.azure.com/user_impersonation",
			handlerStatus: http.StatusBadRequest,
			handlerBody:   `{"error":"invalid_grant","error_description":"Token is expired"}`,
			wantErrSubstr: "invalid_grant",
		},
		{
			name:          "non-json error response",
			clientSecret:  "secret",
			scope:         "https://management.azure.com/user_impersonation",
			handlerStatus: http.StatusInternalServerError,
			handlerBody:   `internal server error`,
			wantErrSubstr: "status 500",
		},
		{
			name:          "empty access token in response",
			clientSecret:  "secret",
			scope:         "https://management.azure.com/user_impersonation",
			handlerStatus: http.StatusOK,
			handlerBody:   `{"token_type":"Bearer"}`,
			wantErrSubstr: "empty access token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := oboConfig()
			cfg.ClientSecret = tt.clientSecret

			var p *AzureOAuthProvider
			if tt.clientSecret != "" {
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if err := r.ParseForm(); err != nil {
						t.Errorf("Failed to parse form: %v", err)
					}
					if r.FormValue("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
						t.Errorf("Expected OBO grant type, got %s", r.FormValue("grant_type"))
					}
					if r.FormValue("scope") != tt.scope {
						t.Errorf("Expected scope %q, got %q", tt.scope, r.FormValue("scope"))
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.handlerStatus)
					_, _ = fmt.Fprint(w, tt.handlerBody)
				})
				p = newProviderWithTransport(t, cfg, handler)
			} else {
				var err error
				p, err = NewAzureOAuthProvider(cfg)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			got, err := p.ExchangeOBO(context.Background(), "user-bearer-token", tt.scope)

			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSubstr)
				}
				if !contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrSubstr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantToken {
				t.Errorf("expected token %q, got %q", tt.wantToken, got)
			}
		})
	}
}

func TestExchangeOBO_RequestFields(t *testing.T) {
	cfg := oboConfig()

	var capturedForm map[string]string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		capturedForm = map[string]string{
			"grant_type":          r.FormValue("grant_type"),
			"assertion":           r.FormValue("assertion"),
			"client_id":           r.FormValue("client_id"),
			"client_secret":       r.FormValue("client_secret"),
			"scope":               r.FormValue("scope"),
			"requested_token_use": r.FormValue("requested_token_use"),
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"access_token":"tok"}`)
	})
	p := newProviderWithTransport(t, cfg, handler)

	_, err := p.ExchangeOBO(context.Background(), "my-bearer", "https://management.azure.com/user_impersonation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedForm["grant_type"] != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
		t.Errorf("wrong grant_type: %s", capturedForm["grant_type"])
	}
	if capturedForm["assertion"] != "my-bearer" {
		t.Errorf("wrong assertion: %s", capturedForm["assertion"])
	}
	if capturedForm["client_id"] != "test-client" {
		t.Errorf("wrong client_id: %s", capturedForm["client_id"])
	}
	if capturedForm["requested_token_use"] != "on_behalf_of" {
		t.Errorf("wrong requested_token_use: %s", capturedForm["requested_token_use"])
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
