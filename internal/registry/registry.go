// Package registry provides a tool registry for AKS MCP server.
package registry

import (
	"github.com/azure/aks-mcp/internal/azure"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolCategory defines a category for tools.
type ToolCategory string

const (
	// CategoryCluster defines tools related to AKS clusters.
	CategoryCluster ToolCategory = "cluster"
	// CategoryNetwork defines tools related to networking.
	CategoryNetwork ToolCategory = "network"
	// CategorySecurity defines tools related to security.
	CategorySecurity ToolCategory = "security"
	// CategoryGeneral defines general tools.
	CategoryGeneral ToolCategory = "general"
)

// ToolDefinition defines a tool and its handler.
type ToolDefinition struct {
	Tool     mcp.Tool
	Handler  server.ToolHandlerFunc
	Category ToolCategory
}

// ToolRegistry is a registry of tools for the AKS MCP server.
type ToolRegistry struct {
	tools         map[string]ToolDefinition
	azureProvider azure.AzureProvider
}


// NewToolRegistry creates a new tool registry.
func NewToolRegistry(azureProvider azure.AzureProvider) *ToolRegistry {
	return &ToolRegistry{
		tools:         make(map[string]ToolDefinition),
		azureProvider: azureProvider,
	}
}

// RegisterTool registers a tool with the registry.
func (r *ToolRegistry) RegisterTool(name string, tool mcp.Tool, handler server.ToolHandlerFunc, category ToolCategory) {
	r.tools[name] = ToolDefinition{
		Tool:     tool,
		Handler:  handler,
		Category: category,
	}
}

// GetAllTools returns all registered tools.
func (r *ToolRegistry) GetAllTools() map[string]ToolDefinition {
	return r.tools
}

// GetAzureProvider returns the Azure provider.
func (r *ToolRegistry) GetAzureProvider() azure.AzureProvider {
	return r.azureProvider
}

// GetCache returns the cache.
func (r *ToolRegistry) GetCache() *azure.AzureCache {
	return r.azureProvider.GetCache()
}

// GetClient returns the Azure client.
func (r *ToolRegistry) GetClient() *azure.AzureClient {
	return r.azureProvider.GetClient()
}

// GetResourceID returns the parsed resource ID.
func (r *ToolRegistry) GetResourceID() *azure.AzureResourceID {
	return r.azureProvider.GetResourceID()
}

// ConfigureMCPServer registers all tools with the MCP server.
func (r *ToolRegistry) ConfigureMCPServer(mcpServer *server.MCPServer) {
	for _, def := range r.tools {
		mcpServer.AddTool(def.Tool, def.Handler)
	}
}
