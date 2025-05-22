package main

import (
	"log"

	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/config"
	"github.com/azure/aks-mcp/internal/registry"
	"github.com/azure/aks-mcp/internal/server"
)

func main() {
	// Parse command line arguments
	cfg := config.ParseFlags()

	// Parse resource ID
	resourceID, err := azure.ParseAzureResourceID(cfg.AKSResourceID)
	if err != nil {
		log.Fatalf("Failed to parse resource ID: %v", err)
	}

	// Initialize Azure client
	client, err := azure.NewAzureClient()
	if err != nil {
		log.Fatalf("Failed to initialize Azure client: %v", err)
	}

	// Initialize cache
	cache := azure.NewAzureCache()
	
	// Create Azure provider
	azureProvider := azure.NewAzureResourceProvider(resourceID, client, cache)

	// Initialize tool registry
	toolRegistry := registry.NewToolRegistry(azureProvider)

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