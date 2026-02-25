package tests

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// CollectNodeLogsTest tests the collect_aks_node_logs tool
type CollectNodeLogsTest struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	LogType        string
	Lines          int
}

// Name returns the test name
func (t *CollectNodeLogsTest) Name() string {
	return fmt.Sprintf("collect_aks_node_logs (%s logs)", t.LogType)
}

// GetParams returns the test parameters for verbose output
func (t *CollectNodeLogsTest) GetParams() map[string]interface{} {
	return map[string]interface{}{
		"subscription_id": t.SubscriptionID,
		"resource_group":  t.ResourceGroup,
		"cluster_name":    t.ClusterName,
		"log_type":        t.LogType,
		"lines":           t.Lines,
	}
}

// Run executes the test
func (t *CollectNodeLogsTest) Run(ctx context.Context, mcpClient *client.Client) (*mcp.CallToolResult, error) {
	// Get VMSS instance information using Azure SDK
	vmssInstance, err := GetFirstVMSSInstance(ctx, t.SubscriptionID, t.ResourceGroup, t.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VMSS instance: %w", err)
	}

	// Build AKS resource ID
	aksResourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
		t.SubscriptionID, t.ResourceGroup, t.ClusterName)

	// Call collect_aks_node_logs tool
	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "collect_aks_node_logs",
			Arguments: map[string]interface{}{
				"aks_resource_id": aksResourceID,
				"vmss_name":       vmssInstance.VMSSName,
				"instance_id":     vmssInstance.InstanceID,
				"log_type":        t.LogType,
				"lines":           t.Lines,
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return result, nil
}

// Validate checks if the tool response is valid
func (t *CollectNodeLogsTest) Validate(result *mcp.CallToolResult) error {
	if len(result.Content) == 0 {
		return fmt.Errorf("empty response content")
	}

	// Get the text content
	var textContent string
	for _, content := range result.Content {
		if text, ok := content.(mcp.TextContent); ok {
			textContent = text.Text
			break
		}
	}

	if textContent == "" {
		return fmt.Errorf("no text content in response")
	}

	// Check if the response is an error message
	// Error messages typically start with "failed to" or "error:"
	lowerContent := strings.ToLower(textContent)
	if strings.HasPrefix(lowerContent, "failed to") ||
		strings.HasPrefix(lowerContent, "error:") ||
		strings.Contains(lowerContent, "authorizationfailed") {
		return fmt.Errorf("tool returned error: %s", textContent)
	}

	// Validate response contains the expected header format
	if !strings.Contains(textContent, "=== AKS Node Logs ===") {
		return fmt.Errorf("response missing expected log header format")
	}

	// Validate response contains expected metadata
	expectedMetadata := []string{
		"Cluster:",     // metadata field
		"VMSS:",        // metadata field
		"Instance ID:", // metadata field
		"Log Type:",    // metadata field
	}

	for _, keyword := range expectedMetadata {
		if !strings.Contains(textContent, keyword) {
			return fmt.Errorf("response missing expected metadata field: %s", keyword)
		}
	}

	// Validate response contains actual log content (should have multiple lines)
	// The header is 7 lines, so we expect at least 10 total lines for meaningful logs
	lines := strings.Split(textContent, "\n")
	if len(lines) < 10 {
		return fmt.Errorf("response too short, expected log content but got only %d lines", len(lines))
	}

	return nil
}
