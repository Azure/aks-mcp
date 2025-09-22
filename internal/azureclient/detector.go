package azureclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// MakeDetectorAPICall makes an HTTP request to Azure Management API for detector operations
func (c *AzureClient) MakeDetectorAPICall(ctx context.Context, url string, subscriptionID string) (*http.Response, error) {
	// Create HTTP client with Azure authentication
	client := &http.Client{}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Get access token for the request
	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AKS-MCP")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}

	return resp, nil
}

// ParseResourceID extracts subscription, resource group, and cluster name from AKS resource ID
func ParseAKSResourceID(resourceID string) (subscriptionID, resourceGroup, clusterName string, err error) {
	// Expected format: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{clusterName}
	parts := strings.Split(strings.TrimPrefix(resourceID, "/"), "/")

	if len(parts) < 8 || !strings.EqualFold(parts[0], "subscriptions") || !strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") || !strings.EqualFold(parts[5], "Microsoft.ContainerService") || !strings.EqualFold(parts[6], "managedClusters") {
		return "", "", "", fmt.Errorf("invalid AKS resource ID format: %s", resourceID)
	}

	subscriptionID = parts[1]
	resourceGroup = parts[3]
	clusterName = parts[7]

	return subscriptionID, resourceGroup, clusterName, nil
}

// HandleDetectorAPIResponse reads and handles the response from detector API calls
func HandleDetectorAPIResponse(resp *http.Response) ([]byte, error) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warnf("Warning: failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorMsg map[string]interface{}
		if err := json.Unmarshal(body, &errorMsg); err == nil {
			if msg, ok := errorMsg["error"].(map[string]interface{}); ok {
				if message, ok := msg["message"].(string); ok {
					return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, message)
				}
			}
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}
