package detectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/aks-mcp/internal/azureclient"
)

// DetectorClient wraps Azure API calls with caching
type DetectorClient struct {
	azClient *azureclient.AzureClient
	cache    *azureclient.AzureCache
}

// NewDetectorClient creates a new detector client
func NewDetectorClient(azClient *azureclient.AzureClient) *DetectorClient {
	return &DetectorClient{
		azClient: azClient,
		cache:    azClient.GetCache(),
	}
}

// ListDetectors lists all detectors for a cluster with caching
func (c *DetectorClient) ListDetectors(ctx context.Context, subscriptionID, resourceGroup, clusterName string) (*DetectorListResponse, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("detectors:list:%s:%s:%s", subscriptionID, resourceGroup, clusterName)

	// Check cache first
	if cached, found := c.cache.Get(cacheKey); found {
		if detectors, ok := cached.(*DetectorListResponse); ok {
			return detectors, nil
		}
	}

	// Build API URL
	apiURL := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s/detectors?api-version=2024-08-01",
		url.PathEscape(subscriptionID),
		url.PathEscape(resourceGroup),
		url.PathEscape(clusterName))

	// Make API call
	resp, err := c.azClient.MakeDetectorAPICall(ctx, apiURL, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to call detector list API: %v", err)
	}

	// Handle response
	body, err := azureclient.HandleDetectorAPIResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to handle detector list response: %v", err)
	}

	// Parse response
	var detectorList DetectorListResponse
	if err := json.Unmarshal(body, &detectorList); err != nil {
		return nil, fmt.Errorf("failed to parse detector list response: %v", err)
	}

	// Cache the result
	c.cache.Set(cacheKey, &detectorList)

	return &detectorList, nil
}

// RunDetector executes a specific detector
func (c *DetectorClient) RunDetector(ctx context.Context, subscriptionID, resourceGroup, clusterName, detectorName, startTime, endTime string) (*DetectorRunResponse, error) {
	// Build API URL with query parameters
	apiURL := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourcegroups/%s/providers/microsoft.containerservice/managedclusters/%s/detectors/%s?startTime=%s&endTime=%s&api-version=2024-08-01",
		url.PathEscape(subscriptionID),
		url.PathEscape(resourceGroup),
		url.PathEscape(clusterName),
		url.PathEscape(detectorName),
		url.QueryEscape(startTime),
		url.QueryEscape(endTime))

	// Make API call
	resp, err := c.azClient.MakeDetectorAPICall(ctx, apiURL, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to call detector run API: %v", err)
	}

	// Handle response
	body, err := azureclient.HandleDetectorAPIResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to handle detector run response: %v", err)
	}

	// Parse response
	var detectorRun DetectorRunResponse
	if err := json.Unmarshal(body, &detectorRun); err != nil {
		return nil, fmt.Errorf("failed to parse detector run response: %v", err)
	}

	return &detectorRun, nil
}

// GetDetectorsByCategory filters detectors by category from cached list
func (c *DetectorClient) GetDetectorsByCategory(ctx context.Context, subscriptionID, resourceGroup, clusterName, category string) ([]Detector, error) {
	// Get full detector list (will use cache if available)
	detectorList, err := c.ListDetectors(ctx, subscriptionID, resourceGroup, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get detector list: %v", err)
	}

	// Filter by category
	var filteredDetectors []Detector
	for _, detector := range detectorList.Value {
		if strings.EqualFold(detector.Properties.Metadata.Category, category) {
			filteredDetectors = append(filteredDetectors, detector)
		}
	}

	return filteredDetectors, nil
}

// RunDetectorsByCategory executes all detectors in a specific category
func (c *DetectorClient) RunDetectorsByCategory(ctx context.Context, subscriptionID, resourceGroup, clusterName, category, startTime, endTime string) ([]DetectorRunResponse, error) {
	// Get detectors by category
	detectors, err := c.GetDetectorsByCategory(ctx, subscriptionID, resourceGroup, clusterName, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get detectors by category: %v", err)
	}

	// Run each detector
	var results []DetectorRunResponse
	for _, detector := range detectors {
		result, err := c.RunDetector(ctx, subscriptionID, resourceGroup, clusterName, detector.Properties.Metadata.ID, startTime, endTime)
		if err != nil {
			// Log error but continue with other detectors
			fmt.Printf("Failed to run detector %s: %v\n", detector.Properties.Metadata.Name, err)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}
