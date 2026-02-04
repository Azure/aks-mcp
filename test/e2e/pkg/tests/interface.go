package tests

import (
	"context"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolTest defines the interface for all E2E tool tests
type ToolTest interface {
	// Name returns the name of the test
	Name() string

	// Run executes the test and returns the result
	Run(ctx context.Context, mcpClient *client.Client) (*mcp.CallToolResult, error)

	// Validate verifies the tool call result
	Validate(result *mcp.CallToolResult) error
}

// TestConfig holds configuration for test execution
type TestConfig struct {
	// MCP Server connection
	ServerURL string

	// Azure resource identifiers (for test parameters)
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
}
