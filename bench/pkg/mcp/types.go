package mcp

import "context"

// MCP Protocol types based on Model Context Protocol specification

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      interface{}    `json:"id"`
	Method  string         `json:"method"`
	Params  interface{}    `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      interface{}    `json:"id"`
	Result  interface{}    `json:"result,omitempty"`
	Error   *JSONRPCError  `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeRequest represents the initialize request params
type InitializeRequest struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ClientInfo      ClientInfo   `json:"clientInfo"`
}

// Capabilities represents client/server capabilities
type Capabilities struct {
	Tools *ToolCapabilities `json:"tools,omitempty"`
}

// ToolCapabilities represents tool-related capabilities
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ClientInfo represents client information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult represents the initialize response result
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ListToolsResult represents the tools/list response result
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// Tool represents a tool definition from MCP server
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema ToolInputSchema        `json:"inputSchema"`
}

// ToolInputSchema represents the JSON schema for tool input
type ToolInputSchema struct {
	Type       string                            `json:"type"`
	Properties map[string]PropertySchema         `json:"properties"`
	Required   []string                          `json:"required,omitempty"`
}

// PropertySchema represents a property schema
type PropertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// CallToolRequest represents the tools/call request params
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult represents the tools/call response result
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in tool result
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type RequestContext struct {
	AzureToken     string `json:"azure_token"`
	SubscriptionID string `json:"subscription_id"`
	ResourceGroup  string `json:"resource_group"`
	ClusterName    string `json:"cluster_name"`
}

type MCPClient interface {
	Initialize(ctx context.Context) (*InitializeResult, error)
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallToolResult, error)
	Close() error
}
