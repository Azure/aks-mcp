package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/aks-mcp/internal/config"
)

// PlacementOperations handles Fleet placement CRD operations
type PlacementOperations struct {
	client *Client
}

// NewPlacementOperations creates a new placement operations handler
func NewPlacementOperations(client *Client) *PlacementOperations {
	if client == nil {
		panic("placement operations client cannot be nil")
	}
	return &PlacementOperations{
		client: client,
	}
}

// CreatePlacement creates a new ClusterResourcePlacement using kubectl
func (p *PlacementOperations) CreatePlacement(ctx context.Context, name, selector, policy string, cfg *config.ConfigData) (string, error) {
	// Build resource selectors
	var resourceSelectors string
	if selector != "" {
		resourceSelectors = `
  resourceSelectors:
  - group: ""
    version: "v1"
    kind: "Namespace"
    labelSelector:
      matchLabels:`
		for _, pair := range strings.Split(selector, ",") {
			if parts := strings.Split(pair, "="); len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				resourceSelectors += fmt.Sprintf("\n        %s: \"%s\"", key, value)
			}
		}
	} else {
		resourceSelectors = `
  resourceSelectors:
  - group: ""
    version: "v1"
    kind: "Namespace"
    labelSelector:
      matchLabels:
        fleet.azure.com/name: "default"`
	}

	manifest := fmt.Sprintf(`apiVersion: placement.kubernetes-fleet.io/v1beta1
kind: ClusterResourcePlacement
metadata:
  name: %s
spec:%s
  policy:
    placementType: %s`, name, resourceSelectors, policy)

	tempFile, err := os.CreateTemp("", "placement-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()

	if _, err := tempFile.WriteString(manifest); err != nil {
		_ = tempFile.Close()
		return "", fmt.Errorf("failed to write manifest: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return p.client.ExecuteKubectl(ctx, fmt.Sprintf("apply -f %s", tempFile.Name()), cfg)
}

// GetPlacement retrieves a ClusterResourcePlacement by name using kubectl
func (p *PlacementOperations) GetPlacement(ctx context.Context, name string, cfg *config.ConfigData) (string, error) {
	return p.client.ExecuteKubectl(ctx, fmt.Sprintf("get clusterresourceplacement %s -o json", name), cfg)
}

// ListPlacements lists all ClusterResourcePlacements using kubectl
func (p *PlacementOperations) ListPlacements(ctx context.Context, cfg *config.ConfigData) (string, error) {
	if p == nil || p.client == nil {
		return "", fmt.Errorf("placement client is nil")
	}

	return p.client.ExecuteKubectl(ctx, "get clusterresourceplacement -o json", cfg)
}

// DeletePlacement deletes a ClusterResourcePlacement by name using kubectl
func (p *PlacementOperations) DeletePlacement(ctx context.Context, name string, cfg *config.ConfigData) (string, error) {
	return p.client.ExecuteKubectl(ctx, fmt.Sprintf("delete clusterresourceplacement %s", name), cfg)
}

// ParsePlacementArgs parses command arguments for placement operations
func ParsePlacementArgs(args string) (map[string]string, error) {
	result := make(map[string]string)
	parts := strings.Fields(args)
	for i := 0; i < len(parts); i++ {
		if key, found := strings.CutPrefix(parts[i], "--"); found {
			if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "--") {
				result[key] = parts[i+1]
				i++
			}
		}
	}
	return result, nil
}
