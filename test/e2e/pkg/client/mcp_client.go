package client

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClient wraps the mcp-go client for E2E testing
type MCPClient struct {
	client    *client.Client
	serverURL string
}

// NewMCPClient creates a new MCP client connected to the server
// No Azure token needed - the MCP server uses Workload Identity for authentication
func NewMCPClient(serverURL string) (*MCPClient, error) {
	// Create streamable HTTP client using the correct API
	mcpClient, err := client.NewStreamableHttpClient(serverURL + "/mcp")
	if err != nil {
		return nil, fmt.Errorf("failed to create streamable HTTP client: %w", err)
	}

	return &MCPClient{
		client:    mcpClient,
		serverURL: serverURL,
	}, nil
}

// Initialize performs the MCP handshake
func (c *MCPClient) Initialize(ctx context.Context) (*mcp.InitializeResult, error) {
	result, err := c.client.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "aks-mcp-e2e-test",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{
				Roots: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{
					ListChanged: true,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	return result, nil
}

// ListTools retrieves available tools from the server
func (c *MCPClient) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	result, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return result, nil
}

// CallTool invokes a tool on the MCP server
func (c *MCPClient) CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := c.client.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	return result, nil
}

// Close closes the MCP client connection
func (c *MCPClient) Close() error {
	return c.client.Close()
}

// GetInternalClient returns the internal mcp-go client for direct access
func (c *MCPClient) GetInternalClient() *client.Client {
	return c.client
}
