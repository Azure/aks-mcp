package main

import (
	"log"

	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/config"
	"github.com/azure/aks-mcp/internal/registry"
	"github.com/azure/aks-mcp/internal/server"
)

func main() {
	// Parse command line arguments and validate configuration
	// This will also parse and validate resource ID if provided
	cfg := config.ParseFlagsAndValidate()

	// If we're here, the config is valid
	if cfg.ResourceIDString == "" {
		// If no resource ID provided, it's null and will be handled by the handlers
		log.Printf("No AKS Resource ID provided, tools will require parameters")
	}

	// Initialize Azure client
	client, err := azure.NewAzureClient()
	if err != nil {
		log.Fatalf("Failed to initialize Azure client: %v", err)
	}

	// Initialize cache
	cache := azure.NewAzureCache()

	// Create Azure provider
	azureProvider := azure.NewAzureResourceProvider(cfg.ParsedResourceID, client, cache)

	// Initialize tool registry with the config
	toolRegistry := registry.NewToolRegistry(azureProvider, cfg)

	// Register all tools
	toolRegistry.RegisterAllTools()

	// Create MCP server
	s := server.NewAKSMCPServer(toolRegistry)

	// Start the server with the specified transport
	switch cfg.Transport {
	case "stdio":
		log.Printf("Starting AKS MCP server with stdio transport")
		if err := s.ServeStdio(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case "sse":
		log.Printf("Starting AKS MCP server with SSE transport on %s", cfg.Address)
		sseServer := s.ServeSSE(cfg.Address)
		if err := sseServer.Start(cfg.Address); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	default:
		log.Fatalf(
			"Invalid transport type: %s. Must be 'stdio' or 'sse'",
			cfg.Transport,
		)
	}
}
