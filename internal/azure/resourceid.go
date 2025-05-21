// Package azure provides Azure SDK integration for AKS MCP server.
package azure

import (
	"errors"
	"fmt"
	"strings"
)

// AzureResourceID represents an Azure resource ID.
type AzureResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	ResourceName   string
	FullID         string
}

// ParseAzureResourceID parses an Azure resource ID into its components.
func ParseAzureResourceID(resourceID string) (*AzureResourceID, error) {
	return ParseResourceID(resourceID)
}

// ParseResourceID parses an Azure resource ID into its components.
func ParseResourceID(resourceID string) (*AzureResourceID, error) {
	if resourceID == "" {
		return nil, errors.New("resource ID cannot be empty")
	}

	// Normalize the resource ID
	resourceID = strings.TrimSpace(resourceID)

	// Azure resource IDs have the format:
	// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/{resourceProvider}/{resourceType}/{resourceName}
	segments := strings.Split(resourceID, "/")

	// A valid resourceID should have at least 9 segments (including empty segments at the start)
	if len(segments) < 9 {
		return nil, fmt.Errorf("invalid resource ID format: %s", resourceID)
	}

	// Check that the resource ID follows the expected pattern
	if segments[1] != "subscriptions" || segments[3] != "resourceGroups" || segments[5] != "providers" {
		return nil, fmt.Errorf("invalid resource ID format: %s", resourceID)
	}

	// For AKS resources, the pattern is:
	// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{clusterName}
	if segments[6] != "Microsoft.ContainerService" || segments[7] != "managedClusters" {
		return nil, fmt.Errorf("resource ID is not an AKS cluster: %s", resourceID)
	}

	return &AzureResourceID{
		SubscriptionID: segments[2],
		ResourceGroup:  segments[4],
		ResourceType:   segments[7],
		ResourceName:   segments[8],
		FullID:         resourceID,
	}, nil
}
