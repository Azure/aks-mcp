package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/aks-mcp/internal/ctx"
	"github.com/mark3labs/mcp-go/server"
)

func TestSSEContextFunc_ExtractsToken(t *testing.T) {
	cfg := createTestConfig("readonly", []string{})
	service := NewService(cfg)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	req := httptest.NewRequest("GET", "/sse", nil)
	req.Header.Set("X-Azure-Token", "test-token-abc")

	var capturedContext context.Context
	contextFunc := func(c context.Context, r *http.Request) context.Context {
		if token := r.Header.Get("X-Azure-Token"); token != "" {
			c = context.WithValue(c, ctx.AzureTokenKey, token)
		}
		capturedContext = c
		return c
	}

	c := contextFunc(context.Background(), req)

	token, ok := c.Value(ctx.AzureTokenKey).(string)
	if !ok {
		t.Fatal("Expected token in context")
	}

	if token != "test-token-abc" {
		t.Errorf("Expected token 'test-token-abc', got '%s'", token)
	}

	if capturedContext == nil {
		t.Fatal("Context should have been captured")
	}
}

func TestSSEContextFunc_NoToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/sse", nil)

	contextFunc := func(c context.Context, r *http.Request) context.Context {
		if token := r.Header.Get("X-Azure-Token"); token != "" {
			c = context.WithValue(c, ctx.AzureTokenKey, token)
		}
		return c
	}

	c := contextFunc(context.Background(), req)

	token, ok := c.Value(ctx.AzureTokenKey).(string)
	if ok {
		t.Errorf("Expected no token in context, got '%s'", token)
	}
}

func TestSSEContextFunc_EmptyToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/sse", nil)
	req.Header.Set("X-Azure-Token", "")

	contextFunc := func(c context.Context, r *http.Request) context.Context {
		if token := r.Header.Get("X-Azure-Token"); token != "" {
			c = context.WithValue(c, ctx.AzureTokenKey, token)
		}
		return c
	}

	c := contextFunc(context.Background(), req)

	token, ok := c.Value(ctx.AzureTokenKey).(string)
	if ok {
		t.Errorf("Expected no token in context for empty header, got '%s'", token)
	}
}

func TestStreamableHTTPContextFunc_ExtractsToken(t *testing.T) {
	cfg := createTestConfig("readonly", []string{})
	service := NewService(cfg)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("X-Azure-Token", "test-token-xyz")

	contextFunc := func(c context.Context, r *http.Request) context.Context {
		if token := r.Header.Get("X-Azure-Token"); token != "" {
			c = context.WithValue(c, ctx.AzureTokenKey, token)
		}
		return c
	}

	c := contextFunc(context.Background(), req)

	token, ok := c.Value(ctx.AzureTokenKey).(string)
	if !ok {
		t.Fatal("Expected token in context")
	}

	if token != "test-token-xyz" {
		t.Errorf("Expected token 'test-token-xyz', got '%s'", token)
	}
}

func TestStreamableHTTPContextFunc_NoToken(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)

	contextFunc := func(c context.Context, r *http.Request) context.Context {
		if token := r.Header.Get("X-Azure-Token"); token != "" {
			c = context.WithValue(c, ctx.AzureTokenKey, token)
		}
		return c
	}

	c := contextFunc(context.Background(), req)

	token, ok := c.Value(ctx.AzureTokenKey).(string)
	if ok {
		t.Errorf("Expected no token in context, got '%s'", token)
	}
}

func TestMultipleHeaders_OnlyTokenExtracted(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("X-Azure-Token", "correct-token")
	req.Header.Set("Authorization", "Bearer should-not-be-used")
	req.Header.Set("X-Custom-Header", "custom-value")

	contextFunc := func(c context.Context, r *http.Request) context.Context {
		if token := r.Header.Get("X-Azure-Token"); token != "" {
			c = context.WithValue(c, ctx.AzureTokenKey, token)
		}
		return c
	}

	c := contextFunc(context.Background(), req)

	token, ok := c.Value(ctx.AzureTokenKey).(string)
	if !ok {
		t.Fatal("Expected token in context")
	}

	if token != "correct-token" {
		t.Errorf("Expected token 'correct-token', got '%s'", token)
	}
}

func TestSSEServerCreation_WithContextFunc(t *testing.T) {
	cfg := createTestConfig("readonly", []string{})
	service := NewService(cfg)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	contextFunc := func(c context.Context, r *http.Request) context.Context {
		if token := r.Header.Get("X-Azure-Token"); token != "" {
			c = context.WithValue(c, ctx.AzureTokenKey, token)
		}
		return c
	}

	sseServer := server.NewSSEServer(
		service.mcpServer,
		server.WithSSEContextFunc(contextFunc),
	)

	if sseServer == nil {
		t.Fatal("SSE server should not be nil")
	}
}

func TestStreamableHTTPServerCreation_WithContextFunc(t *testing.T) {
	cfg := createTestConfig("readonly", []string{})
	service := NewService(cfg)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	contextFunc := func(c context.Context, r *http.Request) context.Context {
		if token := r.Header.Get("X-Azure-Token"); token != "" {
			c = context.WithValue(c, ctx.AzureTokenKey, token)
		}
		return c
	}

	addr := "localhost:8080"
	customServer := service.createCustomHTTPServerWithHelp404(addr)

	streamableServer := server.NewStreamableHTTPServer(
		service.mcpServer,
		server.WithStreamableHTTPServer(customServer),
		server.WithHTTPContextFunc(contextFunc),
	)

	if streamableServer == nil {
		t.Fatal("Streamable HTTP server should not be nil")
	}
}
