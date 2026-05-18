package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
	appctx "github.com/Azure/aks-mcp/internal/ctx"
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

func TestHandleAuthErrorWWWAuthenticateExternalURL(t *testing.T) {
	tests := []struct {
		name            string
		externalURL     string
		expectedURLBase string
		// r.TLS is nil and Host is set — without ExternalURL this would produce http://
	}{
		{
			name:            "externalURL overrides http scheme from proxy request",
			externalURL:     "https://aks-mcp.platform.example.com",
			expectedURLBase: "https://aks-mcp.platform.example.com",
		},
		{
			name:            "no externalURL falls back to request host",
			externalURL:     "",
			expectedURLBase: "http://aks-mcp.platform.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &auth.OAuthConfig{
				Enabled:        true,
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				ExternalURL:    tt.externalURL,
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

			wrappedHandler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Simulate a request arriving at the pod with no TLS (proxy-terminated)
			req := httptest.NewRequest("GET", "/mcp", nil)
			req.Host = "aks-mcp.platform.example.com"
			// r.TLS is nil — as it always is behind a TLS-terminating proxy

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("Expected 401, got %d", w.Code)
			}

			wwwAuth := w.Header().Get("WWW-Authenticate")
			if wwwAuth == "" {
				t.Fatal("Expected WWW-Authenticate header for 401 response")
			}

			expectedResourceMetadata := tt.expectedURLBase + "/.well-known/oauth-protected-resource"
			if !strings.Contains(wwwAuth, expectedResourceMetadata) {
				t.Errorf("Expected resource_metadata=%q in WWW-Authenticate, got: %s", expectedResourceMetadata, wwwAuth)
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

func newOBOMiddleware(t *testing.T, oboEnabled bool, oboHandler http.Handler) *AuthMiddleware {
	t.Helper()
	cfg := &auth.OAuthConfig{
		Enabled:      true,
		TenantID:     "test-tenant",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		OBOEnabled:   oboEnabled,
		TokenValidation: auth.TokenValidationConfig{
			ValidateJWT:      false,
			ValidateAudience: false,
			ExpectedAudience: "https://management.azure.com/",
		},
	}
	p, err := NewAzureOAuthProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	if oboHandler != nil {
		p.httpClient = &http.Client{
			Transport: &roundTripperFunc{
				fn: func(req *http.Request) (*http.Response, error) {
					w := httptest.NewRecorder()
					oboHandler.ServeHTTP(w, req)
					return w.Result(), nil
				},
			},
		}
	}
	return NewAuthMiddleware(p, "http://localhost:8000")
}

func TestSessionCapturingWriter(t *testing.T) {
	t.Run("captures session ID set in response header", func(t *testing.T) {
		var captured string
		scw := &sessionCapturingWriter{
			ResponseWriter: httptest.NewRecorder(),
			onSession:      func(id string) { captured = id },
		}
		scw.ResponseWriter.Header().Set("Mcp-Session-Id", "session-abc")
		scw.WriteHeader(http.StatusOK)

		if captured != "session-abc" {
			t.Errorf("expected session-abc, got %q", captured)
		}
	})

	t.Run("captures on Write when WriteHeader not called", func(t *testing.T) {
		var captured string
		rec := httptest.NewRecorder()
		scw := &sessionCapturingWriter{
			ResponseWriter: rec,
			onSession:      func(id string) { captured = id },
		}
		rec.Header().Set("Mcp-Session-Id", "session-xyz")
		_, _ = scw.Write([]byte("body"))

		if captured != "session-xyz" {
			t.Errorf("expected session-xyz, got %q", captured)
		}
	})

	t.Run("does not capture when header absent", func(t *testing.T) {
		called := false
		scw := &sessionCapturingWriter{
			ResponseWriter: httptest.NewRecorder(),
			onSession:      func(_ string) { called = true },
		}
		scw.WriteHeader(http.StatusOK)

		if called {
			t.Error("onSession should not be called when Mcp-Session-Id is absent")
		}
	})

	t.Run("only fires once despite multiple writes", func(t *testing.T) {
		count := 0
		rec := httptest.NewRecorder()
		scw := &sessionCapturingWriter{
			ResponseWriter: rec,
			onSession:      func(_ string) { count++ },
		}
		rec.Header().Set("Mcp-Session-Id", "session-once")
		scw.WriteHeader(http.StatusOK)
		_, _ = scw.Write([]byte("a"))
		_, _ = scw.Write([]byte("b"))

		if count != 1 {
			t.Errorf("expected onSession called once, got %d", count)
		}
	})
}

func TestSessionContinuation(t *testing.T) {
	m := newOBOMiddleware(t, false, nil)

	tokenInfo := &auth.TokenInfo{
		AccessToken: "original-token",
		Subject:     "user123",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	sessionID := "mcp-session-test-123"
	m.sessions.Store(sessionID, &sessionEntry{
		tokenInfo:    tokenInfo,
		azureToken:   "arm-token",
		clusterToken: "cluster-token",
		oboExpiresAt: time.Now().Add(55 * time.Minute),
		expiresAt:    time.Now().Add(time.Hour),
	})

	var gotAzureToken, gotClusterToken string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAzureToken, _ = r.Context().Value(appctx.AzureTokenKey).(string)
		gotClusterToken, _ = r.Context().Value(appctx.AzureClusterTokenKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Mcp-Session-Id", sessionID)
	w := httptest.NewRecorder()
	m.Middleware(handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gotAzureToken != "arm-token" {
		t.Errorf("expected arm-token in context, got %q", gotAzureToken)
	}
	if gotClusterToken != "cluster-token" {
		t.Errorf("expected cluster-token in context, got %q", gotClusterToken)
	}
}

func TestSessionContinuation_ExpiredSession(t *testing.T) {
	m := newOBOMiddleware(t, false, nil)

	sessionID := "mcp-session-expired"
	m.sessions.Store(sessionID, &sessionEntry{
		tokenInfo: &auth.TokenInfo{Subject: "user"},
		expiresAt: time.Now().Add(-time.Minute), // already expired
	})

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Mcp-Session-Id", sessionID)
	w := httptest.NewRecorder()
	m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	// Expired session has no auth header → should be 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired session, got %d", w.Code)
	}
	if _, ok := m.sessions.Load(sessionID); ok {
		t.Error("expired session should have been deleted")
	}
}

func TestSessionContinuation_UnknownSessionFallsThrough(t *testing.T) {
	m := newOBOMiddleware(t, false, nil)

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Mcp-Session-Id", "unknown-session")
	w := httptest.NewRecorder()
	m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	// No auth header and unknown session → 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown session, got %d", w.Code)
	}
}

func TestOBOTokensInjectedInContext(t *testing.T) {
	callCount := 0
	oboHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"access_token":"obo-token-%d"}`, callCount)
	})

	m := newOBOMiddleware(t, true, oboHandler)

	var gotAzure, gotCluster string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAzure, _ = r.Context().Value(appctx.AzureTokenKey).(string)
		gotCluster, _ = r.Context().Value(appctx.AzureClusterTokenKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer header.payload.signature")
	w := httptest.NewRecorder()
	m.Middleware(handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gotAzure == "" {
		t.Error("expected ARM token in context, got empty string")
	}
	if gotCluster == "" {
		t.Error("expected cluster token in context, got empty string")
	}
	if callCount != 2 {
		t.Errorf("expected 2 OBO calls (ARM + cluster), got %d", callCount)
	}
}

func TestOBOARMFailureCauses401(t *testing.T) {
	oboHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":"invalid_grant","error_description":"expired"}`)
	})

	m := newOBOMiddleware(t, true, oboHandler)

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer header.payload.signature")
	w := httptest.NewRecorder()
	m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when OBO fails, got %d", w.Code)
	}
}

func TestRefreshSessionOBO(t *testing.T) {
	t.Run("success refreshes tokens and updates session", func(t *testing.T) {
		callCount := 0
		oboHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"access_token":"refreshed-%d"}`, callCount)
		})

		m := newOBOMiddleware(t, true, oboHandler)
		se := &sessionEntry{
			bearerToken:  "header.payload.signature",
			azureToken:   "old-arm",
			clusterToken: "old-cluster",
			oboExpiresAt: time.Now().Add(-time.Minute),
			expiresAt:    time.Now().Add(time.Hour),
		}

		ok := m.refreshSessionOBO(context.Background(), se, "session-1")

		if !ok {
			t.Error("expected refresh to succeed")
		}
		if se.azureToken == "old-arm" {
			t.Error("ARM token should have been updated")
		}
		if se.oboExpiresAt.Before(time.Now()) {
			t.Error("oboExpiresAt should be in the future after refresh")
		}
		if callCount != 2 {
			t.Errorf("expected 2 OBO calls, got %d", callCount)
		}
	})

	t.Run("no bearer token returns false", func(t *testing.T) {
		m := newOBOMiddleware(t, true, nil)
		se := &sessionEntry{bearerToken: ""}

		ok := m.refreshSessionOBO(context.Background(), se, "session-2")
		if ok {
			t.Error("expected false when no bearer token stored")
		}
	})

	t.Run("ARM OBO failure returns false", func(t *testing.T) {
		oboHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprint(w, `{"error":"invalid_grant"}`)
		})

		m := newOBOMiddleware(t, true, oboHandler)
		se := &sessionEntry{
			bearerToken:  "header.payload.signature",
			azureToken:   "old",
			oboExpiresAt: time.Now().Add(-time.Minute),
		}

		ok := m.refreshSessionOBO(context.Background(), se, "session-3")
		if ok {
			t.Error("expected false when ARM OBO fails")
		}
		if se.azureToken != "old" {
			t.Error("token should not be modified on failure")
		}
	})
}
