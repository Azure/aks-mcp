package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/aks-mcp/internal/config"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// JWTClaimsKey is the context key for JWT claims
	JWTClaimsKey ContextKey = "jwt_claims"
)

// HTTPAuthMiddleware provides HTTP-level authentication middleware
type HTTPAuthMiddleware struct {
	authConfig *config.AuthConfig
	validator  *EntraValidator
}

// NewHTTPAuthMiddleware creates a new HTTP authentication middleware
func NewHTTPAuthMiddleware(authConfig *config.AuthConfig) (*HTTPAuthMiddleware, error) {
	var validator *EntraValidator
	if authConfig != nil {
		var err error
		validator, err = NewEntraValidator(authConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize authentication validator: %w", err)
		}
	}

	return &HTTPAuthMiddleware{
		authConfig: authConfig,
		validator:  validator,
	}, nil
}

// Middleware returns an HTTP middleware function that enforces authentication
func (m *HTTPAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for non-streamable-http transports
		if !m.authConfig.ShouldAuthenticate("streamable-http") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.sendUnauthorizedResponse(w, "Missing Authorization header")
			return
		}

		// Validate Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			m.sendUnauthorizedResponse(w, "Invalid Authorization header format. Expected 'Bearer <token>'")
			return
		}

		// Extract and validate token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			m.sendUnauthorizedResponse(w, "Empty token")
			return
		}

		// Validate JWT token
		claims, err := m.validator.ValidateToken(r.Context(), token)
		if err != nil {
			m.sendUnauthorizedResponse(w, "Invalid token: "+err.Error())
			return
		}

		// Add claims to request context for potential use by downstream handlers
		ctx := context.WithValue(r.Context(), JWTClaimsKey, claims)
		r = r.WithContext(ctx)

		// Token is valid, proceed to next handler
		next.ServeHTTP(w, r)
	})
}

// sendUnauthorizedResponse sends a standardized 401 response
func (m *HTTPAuthMiddleware) sendUnauthorizedResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    -32600,
			"message": "Authentication required",
			"data":    message,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
	}
}
