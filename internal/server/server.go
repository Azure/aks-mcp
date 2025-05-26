// Package server provides MCP server implementation for AKS.
package server

import (
	"fmt"

	"github.com/azure/aks-mcp/internal/registry"
	"github.com/mark3labs/mcp-go/server"
)

// AKSMCPServer represents the MCP server for AKS.
type AKSMCPServer struct {
	server   *server.MCPServer
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

// ServeSSE serves the MCP server over SSE.
func (s *AKSMCPServer) ServeSSE(addr string) *server.SSEServer {
	return server.NewSSEServer(s.server,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)
}

// ServeStdio serves the MCP server over stdio.
func (s *AKSMCPServer) ServeStdio() error {
	return server.ServeStdio(s.server)
}
