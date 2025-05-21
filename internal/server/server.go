// Package server provides MCP server implementation for AKS.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/azure/aks-mcp/internal/registry"
	"github.com/mark3labs/mcp-go/server"
)

// AKSMCPServer represents the MCP server for AKS.
type AKSMCPServer struct {
	server  *server.MCPServer
	registry *registry.ToolRegistry
}

// NewAKSMCPServer creates a new MCP server for AKS.
func NewAKSMCPServer(registry *registry.ToolRegistry) *AKSMCPServer {
	mcpServer := server.NewMCPServer(
		"aks-mcp-server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
	)
	
	// Register all tools with the MCP server
	registry.ConfigureMCPServer(mcpServer)
	
	return &AKSMCPServer{
		server:   mcpServer,
		registry: registry,
	}
}

// authKey is a custom context key for storing the auth token.
type authKey struct{}

// withAuthKey adds an auth key to the context.
func withAuthKey(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, authKey{}, auth)
}

// authFromRequest extracts the auth token from the request headers.
func authFromRequest(ctx context.Context, r *http.Request) context.Context {
	return withAuthKey(ctx, r.Header.Get("Authorization"))
}

// authFromEnv extracts the auth token from the environment
func authFromEnv(ctx context.Context) context.Context {
	return withAuthKey(ctx, os.Getenv("API_KEY"))
}

// ServeSSE serves the MCP server over SSE.
func (s *AKSMCPServer) ServeSSE(addr string) *server.SSEServer {
	return server.NewSSEServer(s.server,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
		server.WithHTTPContextFunc(authFromRequest),
	)
}

// ServeStdio serves the MCP server over stdio.
func (s *AKSMCPServer) ServeStdio() error {
	return server.ServeStdio(s.server, server.WithStdioContextFunc(authFromEnv))
}
