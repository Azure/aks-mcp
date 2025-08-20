package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Azure/aks-mcp/internal/config"
)

func TestAuthConfig_ShouldAuthenticate(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		transport string
		expected  bool
	}{
		{"disabled auth", false, "streamable-http", false},
		{"stdio transport", true, "stdio", false},
		{"http transport", true, "streamable-http", true},
		{"sse transport", true, "sse", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authConfig := &config.AuthConfig{
				Enabled:            test.enabled,
				RequireAuthForHTTP: true,
			}

			result := authConfig.ShouldAuthenticate(test.transport)
			if result != test.expected {
				t.Errorf("ShouldAuthenticate(%s) with enabled=%v = %v, expected %v",
					test.transport, test.enabled, result, test.expected)
			}
		})
	}
}

func TestUserContext_Creation(t *testing.T) {
	user := &UserContext{
		UserID:   "test-user-id",
		Email:    "test@example.com",
		Name:     "Test User",
		TenantID: "test-tenant",
		Scopes:   []string{"aks.read", "aks.write"},
		IsAdmin:  true,
	}

	if user.UserID != "test-user-id" {
		t.Errorf("Expected UserID 'test-user-id', got '%s'", user.UserID)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got '%s'", user.Email)
	}

	if !user.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}

	if len(user.Scopes) != 2 {
		t.Errorf("Expected 2 scopes, got %d", len(user.Scopes))
	}
}

// Test NewEntraValidator with invalid config
func TestNewEntraValidator_InvalidConfig(t *testing.T) {
	// Test with enabled but empty config
	emptyConfig := &config.AuthConfig{
		Enabled: true, // Enable auth to trigger validation
	}
	_, err := NewEntraValidator(emptyConfig)
	if err == nil {
		t.Error("Expected error with empty config, got nil")
	}

	// Test with missing client ID
	incompleteConfig := &config.AuthConfig{
		Enabled:       true,
		EntraTenantID: "test-tenant",
		// Missing EntraClientID
	}
	_, err = NewEntraValidator(incompleteConfig)
	if err == nil {
		t.Error("Expected error with missing client ID, got nil")
	}

	// Test with missing tenant ID
	incompleteConfig2 := &config.AuthConfig{
		Enabled:       true,
		EntraClientID: "test-client",
		// Missing EntraTenantID
	}
	_, err = NewEntraValidator(incompleteConfig2)
	if err == nil {
		t.Error("Expected error with missing tenant ID, got nil")
	}

	// Test with invalid cache timeout
	invalidCacheConfig := &config.AuthConfig{
		Enabled:          true,
		EntraClientID:    "test-client",
		EntraTenantID:    "test-tenant",
		JWKSCacheTimeout: -1, // Invalid timeout
	}
	_, err = NewEntraValidator(invalidCacheConfig)
	if err == nil {
		t.Error("Expected error with invalid cache timeout, got nil")
	}
}

// Test NewEntraValidator with valid config
func TestNewEntraValidator_ValidConfig(t *testing.T) {
	validConfig := &config.AuthConfig{
		Enabled:          true,
		EntraClientID:    "test-client-id",
		EntraTenantID:    "test-tenant-id",
		EntraAuthority:   "https://login.microsoftonline.com",
		JWKSCacheTimeout: 3600,
	}

	validator, err := NewEntraValidator(validConfig)
	if err != nil {
		t.Errorf("Expected no error with valid config, got: %v", err)
		return
	}

	if validator == nil {
		t.Error("Expected validator to be created, got nil")
		return
	}

	if validator.clientID != "test-client-id" {
		t.Errorf("Expected clientID 'test-client-id', got '%s'", validator.clientID)
	}

	if validator.tenantID != "test-tenant-id" {
		t.Errorf("Expected tenantID 'test-tenant-id', got '%s'", validator.tenantID)
	}
}

// Test Claims structure
func TestClaims_Structure(t *testing.T) {
	claims := &Claims{
		TenantID:          "test-tenant",
		Scope:             "aks.read aks.write",
		Roles:             []string{"admin", "user"},
		AppID:             "test-app-id",
		PreferredUsername: "test@example.com",
		Name:              "Test User",
		Email:             "test@example.com",
		ObjectID:          "test-object-id",
		UPN:               "test@example.com",
	}

	claims.Issuer = "https://login.microsoftonline.com/test-tenant/v2.0"
	claims.Subject = "test-subject"
	claims.Audience = []string{"test-audience"}
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())

	if claims.TenantID != "test-tenant" {
		t.Errorf("Expected TenantID 'test-tenant', got '%s'", claims.TenantID)
	}

	if claims.Scope != "aks.read aks.write" {
		t.Errorf("Expected Scope 'aks.read aks.write', got '%s'", claims.Scope)
	}

	if len(claims.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(claims.Roles))
	}

	if claims.AppID != "test-app-id" {
		t.Errorf("Expected AppID 'test-app-id', got '%s'", claims.AppID)
	}
}

// Test HTTP Middleware creation
func TestNewHTTPAuthMiddleware(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled:          true,
		EntraClientID:    "test-client-id",
		EntraTenantID:    "test-tenant-id",
		EntraAuthority:   "https://login.microsoftonline.com",
		JWKSCacheTimeout: 3600,
	}

	middleware := NewHTTPAuthMiddleware(authConfig)
	if middleware == nil {
		t.Error("Expected middleware to be created, got nil")
		return
	}

	if middleware.authConfig != authConfig {
		t.Error("Expected authConfig to be set correctly")
	}

	if middleware.validator == nil {
		t.Error("Expected validator to be created")
	}
}

// Test HTTP Middleware with disabled auth
func TestHTTPAuthMiddleware_DisabledAuth(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled: false,
	}

	middleware := NewHTTPAuthMiddleware(authConfig)

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("success"))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	// Create test request
	req := httptest.NewRequest("POST", "/mcp", nil)
	recorder := httptest.NewRecorder()

	// Call middleware
	handler := middleware.Middleware(nextHandler)
	handler.ServeHTTP(recorder, req)

	// Should pass through without authentication
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Body.String() != "success" {
		t.Errorf("Expected 'success', got '%s'", recorder.Body.String())
	}
}

// Test HTTP Middleware with missing Authorization header
func TestHTTPAuthMiddleware_MissingAuthHeader(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled:            true,
		RequireAuthForHTTP: true,
		EntraClientID:      "test-client-id",
		EntraTenantID:      "test-tenant-id",
		EntraAuthority:     "https://login.microsoftonline.com",
	}

	middleware := NewHTTPAuthMiddleware(authConfig)

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create test request without Authorization header
	req := httptest.NewRequest("POST", "/mcp", nil)
	recorder := httptest.NewRecorder()

	// Call middleware
	handler := middleware.Middleware(nextHandler)
	handler.ServeHTTP(recorder, req)

	// Should return 401
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}

	// Check response format
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response JSON: %v", err)
	}

	if response["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got '%v'", response["jsonrpc"])
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Error("Expected error object in response")
	}

	if errorObj["code"] != float64(-32600) {
		t.Errorf("Expected error code -32600, got %v", errorObj["code"])
	}
}

// Test HTTP Middleware with invalid Authorization header format
func TestHTTPAuthMiddleware_InvalidAuthHeaderFormat(t *testing.T) {
	authConfig := &config.AuthConfig{
		Enabled:            true,
		RequireAuthForHTTP: true,
		EntraClientID:      "test-client-id",
		EntraTenantID:      "test-tenant-id",
		EntraAuthority:     "https://login.microsoftonline.com",
	}

	middleware := NewHTTPAuthMiddleware(authConfig)

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		authHeader string
	}{
		{"Basic auth", "Basic dGVzdDp0ZXN0"},
		{"Invalid format", "InvalidFormat token"},
		{"Missing token", "Bearer"},
		{"Empty Bearer", "Bearer "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			req.Header.Set("Authorization", test.authHeader)
			recorder := httptest.NewRecorder()

			handler := middleware.Middleware(nextHandler)
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401 for %s, got %d", test.name, recorder.Code)
			}
		})
	}
}

// Test HTTP Middleware sendUnauthorizedResponse
func TestHTTPAuthMiddleware_SendUnauthorizedResponse(t *testing.T) {
	authConfig := &config.AuthConfig{}
	middleware := NewHTTPAuthMiddleware(authConfig)

	recorder := httptest.NewRecorder()
	message := "Test error message"

	middleware.sendUnauthorizedResponse(recorder, message)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response JSON: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Error("Expected error object in response")
	}

	if errorObj["data"] != message {
		t.Errorf("Expected error data '%s', got '%v'", message, errorObj["data"])
	}
}

// Test getUserID helper function
func TestEntraValidator_GetUserID(t *testing.T) {
	validator := &EntraValidator{}

	tests := []struct {
		name     string
		claims   *Claims
		expected string
	}{
		{
			name: "ObjectID present",
			claims: &Claims{
				ObjectID: "object-id-123",
			},
			expected: "object-id-123",
		},
		{
			name: "ObjectID empty, use Subject",
			claims: &Claims{
				ObjectID: "",
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "subject-456",
				},
			},
			expected: "subject-456",
		},
		{
			name: "Both empty",
			claims: &Claims{
				ObjectID: "",
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "",
				},
			},
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.getUserID(test.claims)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

// Test getEmail helper function
func TestEntraValidator_GetEmail(t *testing.T) {
	validator := &EntraValidator{}

	tests := []struct {
		name     string
		claims   *Claims
		expected string
	}{
		{
			name: "Email present",
			claims: &Claims{
				Email: "user@example.com",
			},
			expected: "user@example.com",
		},
		{
			name: "Email empty, use UPN",
			claims: &Claims{
				Email: "",
				UPN:   "user@upn.com",
			},
			expected: "user@upn.com",
		},
		{
			name: "Email and UPN empty, use PreferredUsername",
			claims: &Claims{
				Email:             "",
				UPN:               "",
				PreferredUsername: "preferred@username.com",
			},
			expected: "preferred@username.com",
		},
		{
			name: "All empty",
			claims: &Claims{
				Email:             "",
				UPN:               "",
				PreferredUsername: "",
			},
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.getEmail(test.claims)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

// Test validateBasicClaims function
func TestEntraValidator_ValidateBasicClaims(t *testing.T) {
	validator := &EntraValidator{
		clientID: "test-client-id",
		tenantID: "test-tenant-id",
		issuer:   "https://login.microsoftonline.com/test-tenant-id/v2.0",
	}

	tests := []struct {
		name        string
		claims      *Claims
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid claims",
			claims: &Claims{
				TenantID: "test-tenant-id",
				RegisteredClaims: jwt.RegisteredClaims{
					Audience: []string{"test-client-id"},
					Issuer:   "https://login.microsoftonline.com/test-tenant-id/v2.0",
				},
			},
			expectError: false,
		},
		{
			name: "valid claims with api:// audience",
			claims: &Claims{
				TenantID: "test-tenant-id",
				RegisteredClaims: jwt.RegisteredClaims{
					Audience: []string{"api://test-client-id"},
					Issuer:   "https://login.microsoftonline.com/test-tenant-id/v2.0",
				},
			},
			expectError: false,
		},
		{
			name: "valid claims with v1.0 issuer",
			claims: &Claims{
				TenantID: "test-tenant-id",
				RegisteredClaims: jwt.RegisteredClaims{
					Audience: []string{"test-client-id"},
					Issuer:   "https://sts.windows.net/test-tenant-id/",
				},
			},
			expectError: false,
		},
		{
			name: "invalid tenant ID",
			claims: &Claims{
				TenantID: "wrong-tenant-id",
				RegisteredClaims: jwt.RegisteredClaims{
					Audience: []string{"test-client-id"},
					Issuer:   "https://login.microsoftonline.com/test-tenant-id/v2.0",
				},
			},
			expectError: true,
			errorMsg:    "invalid tenant ID",
		},
		{
			name: "missing audience",
			claims: &Claims{
				TenantID: "test-tenant-id",
				RegisteredClaims: jwt.RegisteredClaims{
					Audience: []string{},
					Issuer:   "https://login.microsoftonline.com/test-tenant-id/v2.0",
				},
			},
			expectError: true,
			errorMsg:    "missing audience claim",
		},
		{
			name: "invalid audience",
			claims: &Claims{
				TenantID: "test-tenant-id",
				RegisteredClaims: jwt.RegisteredClaims{
					Audience: []string{"wrong-audience"},
					Issuer:   "https://login.microsoftonline.com/test-tenant-id/v2.0",
				},
			},
			expectError: true,
			errorMsg:    "invalid audience",
		},
		{
			name: "invalid issuer",
			claims: &Claims{
				TenantID: "test-tenant-id",
				RegisteredClaims: jwt.RegisteredClaims{
					Audience: []string{"test-client-id"},
					Issuer:   "https://invalid-issuer.com/test-tenant-id/v2.0",
				},
			},
			expectError: true,
			errorMsg:    "invalid issuer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.validateBasicClaims(test.claims)

			if test.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if test.errorMsg != "" && !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Test ContextKey constants
func TestContextKey_Constants(t *testing.T) {
	if JWTClaimsKey != "jwt_claims" {
		t.Errorf("Expected JWTClaimsKey to be 'jwt_claims', got '%s'", JWTClaimsKey)
	}
}

// Test config validation edge cases
func TestAuthConfig_ValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.AuthConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "disabled config should pass",
			config: &config.AuthConfig{
				Enabled: false,
				// All other fields empty
			},
			expectError: false,
		},
		{
			name: "enabled with all required fields",
			config: &config.AuthConfig{
				Enabled:          true,
				EntraClientID:    "client-id",
				EntraTenantID:    "tenant-id",
				JWKSCacheTimeout: 3600,
			},
			expectError: false,
		},
		{
			name: "zero cache timeout",
			config: &config.AuthConfig{
				Enabled:          true,
				EntraClientID:    "client-id",
				EntraTenantID:    "tenant-id",
				JWKSCacheTimeout: 0,
			},
			expectError: true,
			errorMsg:    "jwks_cache_timeout must be positive",
		},
		{
			name: "negative cache timeout",
			config: &config.AuthConfig{
				Enabled:          true,
				EntraClientID:    "client-id",
				EntraTenantID:    "tenant-id",
				JWKSCacheTimeout: -100,
			},
			expectError: true,
			errorMsg:    "jwks_cache_timeout must be positive",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.ValidateConfig()

			if test.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if test.errorMsg != "" && !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Test JWT token validation error cases (mock test since we can't easily mock JWKS)
func TestValidateToken_ErrorCases(t *testing.T) {
	validConfig := &config.AuthConfig{
		Enabled:          true,
		EntraClientID:    "test-client-id",
		EntraTenantID:    "test-tenant-id",
		EntraAuthority:   "https://login.microsoftonline.com",
		JWKSCacheTimeout: 3600,
	}

	validator, err := NewEntraValidator(validConfig)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "invalid.token"},
		{"malformed JWT", "header.payload"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validator.ValidateToken(context.Background(), test.token)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", test.name)
			}
		})
	}
}
