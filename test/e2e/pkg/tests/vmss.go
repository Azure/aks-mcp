package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetVMSSInfoTest tests the get_aks_vmss_info tool
type GetVMSSInfoTest struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	NodePoolName   string // Optional, empty for all node pools
}

// Name returns the test name
func (t *GetVMSSInfoTest) Name() string {
	if t.NodePoolName != "" {
		return fmt.Sprintf("get_aks_vmss_info (node pool: %s)", t.NodePoolName)
	}
	return "get_aks_vmss_info (all node pools)"
}

// GetParams returns the parameters for verbose output
func (t *GetVMSSInfoTest) GetParams() map[string]interface{} {
	params := map[string]interface{}{
		"subscription_id": t.SubscriptionID,
		"resource_group":  t.ResourceGroup,
		"cluster_name":    t.ClusterName,
	}
	if t.NodePoolName != "" {
		params["node_pool_name"] = t.NodePoolName
	}
	return params
}

// Run executes the test
func (t *GetVMSSInfoTest) Run(ctx context.Context, mcpClient *client.Client) (*mcp.CallToolResult, error) {
	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_aks_vmss_info",
			Arguments: t.GetParams(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return result, nil
}

// Validate verifies the tool result contains expected VMSS information
func (t *GetVMSSInfoTest) Validate(result *mcp.CallToolResult) error {
	if result == nil {
		return fmt.Errorf("result is nil")
	}

	if len(result.Content) == 0 {
		return fmt.Errorf("empty result content")
	}

	// Get the text content
	var textContent string
	for _, content := range result.Content {
		// Type assert to TextContent (value type, not pointer)
		if tc, ok := content.(mcp.TextContent); ok {
			textContent = tc.Text
			break
		}
	}

	if textContent == "" {
		return fmt.Errorf("no text content in result")
	}

	// Simple validation: just check that the response contains expected keywords
	// The content is for AI consumption, not strict programmatic parsing
	expectedKeywords := []string{
		"vmss",
		"cluster_name",
		t.ClusterName,
		t.ResourceGroup,
	}

	for _, keyword := range expectedKeywords {
		if !strings.Contains(textContent, keyword) {
			return fmt.Errorf("response missing expected keyword: %s", keyword)
		}
	}

	// Check that it looks like valid JSON (but don't parse it strictly)
	if !json.Valid([]byte(textContent)) {
		return fmt.Errorf("response is not valid JSON")
	}

	return nil
}
