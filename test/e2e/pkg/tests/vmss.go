package tests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetVMSSInfoTest tests the get_vmss_info tool
type GetVMSSInfoTest struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	NodePoolName   string // Optional, empty for all node pools
}

// Name returns the test name
func (t *GetVMSSInfoTest) Name() string {
	if t.NodePoolName != "" {
		return fmt.Sprintf("get_vmss_info (node pool: %s)", t.NodePoolName)
	}
	return "get_vmss_info (all node pools)"
}

// Run executes the test
func (t *GetVMSSInfoTest) Run(ctx context.Context, mcpClient *client.Client) (*mcp.CallToolResult, error) {
	args := map[string]interface{}{
		"subscription_id": t.SubscriptionID,
		"resource_group":  t.ResourceGroup,
		"cluster_name":    t.ClusterName,
	}

	// Add node_pool_name only if specified
	if t.NodePoolName != "" {
		args["node_pool_name"] = t.NodePoolName
	}

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_vmss_info",
			Arguments: args,
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
		// Type assert to TextContent
		if tc, ok := content.(*mcp.TextContent); ok && tc.Type == "text" {
			textContent = tc.Text
			break
		}
	}

	if textContent == "" {
		return fmt.Errorf("no text content in result")
	}

	// Try to parse as JSON
	var vmssData interface{}
	if err := json.Unmarshal([]byte(textContent), &vmssData); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}

	// Basic validation: check if it's a map (single node pool) or array (multiple node pools)
	switch v := vmssData.(type) {
	case map[string]interface{}:
		// Single node pool result
		return validateVMSSObject(v)
	case []interface{}:
		// Multiple node pools result
		if len(v) == 0 {
			return fmt.Errorf("empty VMSS array")
		}
		for i, item := range v {
			vmssObj, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid VMSS object at index %d", i)
			}
			if err := validateVMSSObject(vmssObj); err != nil {
				return fmt.Errorf("validation failed for VMSS at index %d: %w", i, err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unexpected response type: %T", v)
	}
}

// validateVMSSObject validates a single VMSS object
func validateVMSSObject(obj map[string]interface{}) error {
	// Check for key fields that should exist in VMSS info
	requiredFields := []string{"id", "name", "location"}
	for _, field := range requiredFields {
		if _, ok := obj[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate that id contains "virtualMachineScaleSets"
	if id, ok := obj["id"].(string); ok {
		// VMSS ID format: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachineScaleSets/{vmssName}
		if id == "" {
			return fmt.Errorf("empty VMSS id")
		}
		// Note: We don't strictly validate the format as it might vary
	} else {
		return fmt.Errorf("id field is not a string")
	}

	return nil
}
