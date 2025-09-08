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
