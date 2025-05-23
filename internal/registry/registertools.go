// Package registry provides a tool registry for AKS MCP server.
package registry

// RegisterAllTools registers all tools with the registry.
func (r *ToolRegistry) RegisterAllTools() {
	// Register cluster tools
	r.registerClusterTools()

	// Register network tools
	r.registerNetworkTools()

	// Register other tool categories as needed
}
