package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
)

// GetTokenInfo extracts token information from request context (test helper)
func GetTokenInfo(r *http.Request) (*auth.TokenInfo, bool) {
	tokenInfo, ok := r.Context().Value(tokenInfoKey).(*auth.TokenInfo)
	return tokenInfo, ok
}

func TestAuthMiddleware(t *testing.T) {
	// Create test config with minimal required scopes for testing
	// Note: We cannot test with empty RequiredScopes because the OAuth configuration
	// validation now requires at least one scope to be specified. This is intentional
	// to prevent security misconfigurations in production environments.
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false,
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
	middleware := NewAuthMiddleware(provider, "http://localhost:8000")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("Failed to write test response: %v", err)
		}
	})

	wrappedHandler := middleware.Middleware(testHandler)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		path           string
	}{
		{
			name:           "valid bearer token",
			authHeader:     "Bearer header.payload.signature",
			expectedStatus: http.StatusOK,
			path:           "/test",
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			path:           "/test",
		},
		{
			name:           "invalid token format",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
			path:           "/test",
		},
		{
			name:           "non-bearer token",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectedStatus: http.StatusUnauthorized,
			path:           "/test",
		},
		{
			name:           "skip auth for metadata endpoint",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			path:           "/.well-known/oauth-protected-resource",
		},
		{
			name:           "skip auth for health endpoint",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			path:           "/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check WWW-Authenticate header for 401 responses
			if w.Code == http.StatusUnauthorized {
				wwwAuth := w.Header().Get("WWW-Authenticate")
				if wwwAuth == "" {
					t.Error("Expected WWW-Authenticate header for 401 response")
				}
			}
		})
	}
}

func TestAuthMiddlewareContextPropagation(t *testing.T) {
	// Note: We cannot test with empty RequiredScopes because the OAuth configuration
	// validation now requires at least one scope to be specified.
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"},
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false,
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
	middleware := NewAuthMiddleware(provider, "http://localhost:8000")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if token info is available in context
		tokenInfo, ok := GetTokenInfo(r)
		if !ok {
			t.Error("Token info not found in context")
			return
		}

		if tokenInfo.AccessToken != "header.payload.signature" {
			t.Errorf("Expected token header.payload.signature, got %s", tokenInfo.AccessToken)
		}

		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer header.payload.signature")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestShouldSkipAuth(t *testing.T) {
	// Note: We cannot test with empty RequiredScopes because the OAuth configuration
	// validation now requires at least one scope to be specified.
	config := &auth.OAuthConfig{
		Enabled:        true,
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		RequiredScopes: []string{"https://management.azure.com/.default"}, // Minimal scope for testing
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false,
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
	middleware := NewAuthMiddleware(provider, "http://localhost:8000")

	tests := []struct {
		path     string
		expected bool
	}{
		{"/.well-known/oauth-protected-resource", true},
		{"/.well-known/oauth-authorization-server", true},
		{"/.well-known/openid-configuration", true},
		{"/oauth2/v2.0/authorize", true},
		{"/oauth/register", true},
		{"/oauth/callback", true},
		{"/oauth2/v2.0/token", true},
		{"/oauth/introspect", true},
		{"/health", true},
		{"/ping", true},
		{"/test", false},
		{"/mcp", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			result := middleware.shouldSkipAuth(req)
			if result != tt.expected {
				t.Errorf("Expected %v for path %s, got %v", tt.expected, tt.path, result)
			}
		})
	}
}

func TestGetTokenInfo(t *testing.T) {
	// Test with valid token info
	tokenInfo := &auth.TokenInfo{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		Subject:     "user123",
	}

	ctx := context.WithValue(context.Background(), tokenInfoKey, tokenInfo)
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(ctx)

	retrievedTokenInfo, ok := GetTokenInfo(req)
	if !ok {
		t.Error("Expected to find token info in context")
	}

	if retrievedTokenInfo.AccessToken != "test-token" {
		t.Errorf("Expected access token test-token, got %s", retrievedTokenInfo.AccessToken)
	}

	// Test without token info
	req = httptest.NewRequest("GET", "/test", nil)
	_, ok = GetTokenInfo(req)
	if ok {
		t.Error("Expected not to find token info in empty context")
	}
}
