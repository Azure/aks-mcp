// Package registry provides a tool registry for AKS MCP server.
package registry

import (
	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolCategory defines a category for tools.
type ToolCategory string

// ToolAccessLevel defines access level required for a tool.
type ToolAccessLevel string

const (
	// CategoryCluster defines tools related to AKS clusters.
	CategoryCluster ToolCategory = "cluster"
	// CategoryNetwork defines tools related to networking.
	CategoryNetwork ToolCategory = "network"
	// CategorySecurity defines tools related to security.
	CategorySecurity ToolCategory = "security"
	// CategoryGeneral defines general tools.
	CategoryGeneral ToolCategory = "general"

	// AccessRead represents read-only access level.
	AccessRead ToolAccessLevel = "read"
	// AccessReadWrite represents read-write access level.
	AccessReadWrite ToolAccessLevel = "readwrite"
	// AccessAdmin represents administrative access level.
	AccessAdmin ToolAccessLevel = "admin"
)

// ToolDefinition defines a tool and its handler.
type ToolDefinition struct {
	Tool       mcp.Tool
	Handler    server.ToolHandlerFunc
	Category   ToolCategory
	AccessLevel ToolAccessLevel
}

// ToolRegistry is a registry of tools for the AKS MCP server.
type ToolRegistry struct {
	tools         map[string]ToolDefinition
	azureProvider azure.AzureProvider
	config        *config.Config
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry(azureProvider azure.AzureProvider, cfg *config.Config) *ToolRegistry {
	return &ToolRegistry{
		tools:         make(map[string]ToolDefinition),
		azureProvider: azureProvider,
		config:        cfg,
	}
}

// RegisterTool registers a tool with the registry.
func (r *ToolRegistry) RegisterTool(name string, tool mcp.Tool, handler server.ToolHandlerFunc, category ToolCategory, accessLevel ToolAccessLevel) {
	r.tools[name] = ToolDefinition{
		Tool:       tool,
		Handler:    handler,
		Category:   category,
		AccessLevel: accessLevel,
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

// GetConfig returns the configuration.
func (r *ToolRegistry) GetConfig() *config.Config {
	return r.config
}

// ConfigureMCPServer registers all tools with the MCP server.
func (r *ToolRegistry) ConfigureMCPServer(mcpServer *server.MCPServer) {
	configAccessLevel := r.config.AccessLevel
	
	for _, def := range r.tools {
		// Filter tools based on access level
		if shouldRegisterTool(string(def.AccessLevel), configAccessLevel) {
			mcpServer.AddTool(def.Tool, def.Handler)
		}
	}
}

// shouldRegisterTool determines if a tool should be registered based on access level.
func shouldRegisterTool(toolAccessLevel, configAccessLevel string) bool {
	switch configAccessLevel {
	case "read":
		return toolAccessLevel == "read"
	case "readwrite":
		return toolAccessLevel == "read" || toolAccessLevel == "readwrite"
	case "admin":
		return true // Admin has access to all tools
	default:
		return toolAccessLevel == "read" // Default to read-only for unknown access levels
	}
}
