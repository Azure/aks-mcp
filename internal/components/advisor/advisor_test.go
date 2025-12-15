package advisor

import (
	"context"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

// Test data
var mockCLIRecommendations = []CLIRecommendation{
	{
		ID:            "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerService/managedClusters/aks-cluster-1",
		Name:          "rec1",
		Category:      "Cost",
		Impact:        "High",
		ImpactedValue: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerService/managedClusters/aks-cluster-1",
		LastUpdated:   "2024-01-15T10:30:00Z",
		ShortDescription: struct {
			Problem  string `json:"problem"`
			Solution string `json:"solution"`
		}{
			Problem:  "Underutilized AKS cluster nodes",
			Solution: "Consider reducing node count or using smaller VM sizes for your AKS cluster",
		},
	},
	{
		ID:            "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerService/managedClusters/aks-cluster-1/agentPools/nodepool1",
		Name:          "rec2",
		Category:      "Security",
		Impact:        "Medium",
		ImpactedValue: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerService/managedClusters/aks-cluster-1/agentPools/nodepool1",
		LastUpdated:   "2024-01-15T09:15:00Z",
		ShortDescription: struct {
			Problem  string `json:"problem"`
			Solution string `json:"solution"`
		}{
			Problem:  "AKS node pool missing security configurations",
			Solution: "Enable Azure Policy and security monitoring for AKS node pools",
		},
	},
	{
		ID:            "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Storage/storageAccounts/mystorage",
		Name:          "rec3",
		Category:      "Performance",
		Impact:        "Low",
		ImpactedValue: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Storage/storageAccounts/mystorage",
		LastUpdated:   "2024-01-15T08:00:00Z",
		ShortDescription: struct {
			Problem  string `json:"problem"`
			Solution string `json:"solution"`
		}{
			Problem:  "Storage account performance issue",
			Solution: "Upgrade storage account tier",
		},
	},
}

func TestFilterAKSRecommendationsFromCLI(t *testing.T) {
	aksRecommendations := filterAKSRecommendationsFromCLI(mockCLIRecommendations)

	// Should filter out the storage account recommendation and keep only AKS-related ones
	expectedCount := 2
	if len(aksRecommendations) != expectedCount {
		t.Errorf("Expected %d AKS recommendations, got %d", expectedCount, len(aksRecommendations))
	}

	// Verify the filtered recommendations are AKS-related
	for _, rec := range aksRecommendations {
		if !isAKSRelatedCLI(rec.ImpactedValue) {
			t.Errorf("Non-AKS recommendation found in filtered results: %s", rec.ImpactedValue)
		}
	}
}

func TestIsAKSRelatedCLI(t *testing.T) {
	testCases := []struct {
		resourceID string
		expected   bool
	}{
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerService/managedClusters/aks-cluster-1", true},
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerService/managedClusters/aks-cluster-1/agentPools/nodepool1", true},
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/loadBalancers/kubernetes-lb", true},
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/publicIPAddresses/kubernetes-ip", true},
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Storage/storageAccounts/mystorage", false},
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isAKSRelatedCLI(tc.resourceID)
		if result != tc.expected {
			t.Errorf("For resourceID %s, expected %v, got %v", tc.resourceID, tc.expected, result)
		}
	}
}

func TestExtractAKSClusterNameFromCLI(t *testing.T) {
	testCases := []struct {
		resourceID   string
		expectedName string
	}{
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerService/managedClusters/aks-cluster-1", "aks-cluster-1"},
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ContainerService/managedClusters/my-test-cluster/agentPools/nodepool1", "my-test-cluster"},
		{"/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Storage/storageAccounts/mystorage", ""},
		{"", ""},
	}

	for _, tc := range testCases {
		result := extractAKSClusterNameFromCLI(tc.resourceID)
		if result != tc.expectedName {
			t.Errorf("For resourceID %s, expected cluster name %s, got %s", tc.resourceID, tc.expectedName, result)
		}
	}
}

func TestExtractResourceGroupFromResourceID(t *testing.T) {
	testCases := []struct {
		resourceID string
		expectedRG string
	}{
		{"/subscriptions/sub1/resourceGroups/my-rg/providers/Microsoft.ContainerService/managedClusters/aks-cluster-1", "my-rg"},
		{"/subscriptions/sub1/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/mystorage", "test-rg"},
		{"/subscriptions/sub1/providers/Microsoft.Advisor/recommendations/rec1", ""},
		{"", ""},
	}

	for _, tc := range testCases {
		result := extractResourceGroupFromResourceID(tc.resourceID)
		if result != tc.expectedRG {
			t.Errorf("For resourceID %s, expected resource group %s, got %s", tc.resourceID, tc.expectedRG, result)
		}
	}
}

func TestConvertToAKSRecommendationSummary(t *testing.T) {
	rec := mockCLIRecommendations[0] // Cost recommendation for AKS cluster
	summary := convertToAKSRecommendationSummary(rec)

	if summary.ID != rec.ID {
		t.Errorf("Expected ID %s, got %s", rec.ID, summary.ID)
	}

	if summary.Category != rec.Category {
		t.Errorf("Expected category %s, got %s", rec.Category, summary.Category)
	}

	if summary.ClusterName != "aks-cluster-1" {
		t.Errorf("Expected cluster name aks-cluster-1, got %s", summary.ClusterName)
	}

	if summary.ResourceGroup != "rg1" {
		t.Errorf("Expected resource group rg1, got %s", summary.ResourceGroup)
	}

	if summary.ResourceID != rec.ID {
		t.Errorf("Expected resource ID %s, got %s", rec.ID, summary.ResourceID)
	}

	if summary.AKSSpecific.ConfigurationArea != "compute" {
		t.Errorf("Expected configuration area compute, got %s", summary.AKSSpecific.ConfigurationArea)
	}
}

func TestFilterBySeverity(t *testing.T) {
	// Filter for High severity
	highSeverity := filterBySeverity(mockCLIRecommendations, "High")
	if len(highSeverity) != 1 {
		t.Errorf("Expected 1 high severity recommendation, got %d", len(highSeverity))
	}

	// Filter for Medium severity
	mediumSeverity := filterBySeverity(mockCLIRecommendations, "Medium")
	if len(mediumSeverity) != 1 {
		t.Errorf("Expected 1 medium severity recommendation, got %d", len(mediumSeverity))
	}

	// Filter for Low severity
	lowSeverity := filterBySeverity(mockCLIRecommendations, "Low")
	if len(lowSeverity) != 1 {
		t.Errorf("Expected 1 low severity recommendation, got %d", len(lowSeverity))
	}
}

func TestGenerateAKSAdvisorReport(t *testing.T) {
	// Convert mock data to AKS recommendations
	aksRecommendations := filterAKSRecommendationsFromCLI(mockCLIRecommendations)
	summaries := convertToAKSRecommendationSummaries(aksRecommendations)

	// Generate report
	report := generateAKSAdvisorReport("test-subscription", summaries, "summary")

	if report.SubscriptionID != "test-subscription" {
		t.Errorf("Expected subscription ID test-subscription, got %s", report.SubscriptionID)
	}

	if len(report.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations in report, got %d", len(report.Recommendations))
	}

	if report.Summary.TotalRecommendations != 2 {
		t.Errorf("Expected total recommendations 2, got %d", report.Summary.TotalRecommendations)
	}

	if report.Summary.ClustersAffected != 1 {
		t.Errorf("Expected 1 cluster affected, got %d", report.Summary.ClustersAffected)
	}

	// Check category breakdown
	if report.Summary.ByCategory["Cost"] != 1 {
		t.Errorf("Expected 1 cost recommendation, got %d", report.Summary.ByCategory["Cost"])
	}

	if report.Summary.ByCategory["Security"] != 1 {
		t.Errorf("Expected 1 security recommendation, got %d", report.Summary.ByCategory["Security"])
	}
}

func TestMapCategoryToConfigArea(t *testing.T) {
	testCases := []struct {
		category     string
		expectedArea string
	}{
		{"Cost", "compute"},
		{"Security", "security"},
		{"Performance", "compute"},
		{"HighAvailability", "networking"},
		{"Unknown", "general"},
	}

	for _, tc := range testCases {
		result := mapCategoryToConfigArea(tc.category)
		if result != tc.expectedArea {
			t.Errorf("For category %s, expected config area %s, got %s", tc.category, tc.expectedArea, result)
		}
	}
}

func TestHandleAdvisorRecommendationInvalidOperation(t *testing.T) {
	cfg := &config.ConfigData{}
	params := map[string]interface{}{
		"operation": "invalid_operation",
	}

	_, err := HandleAdvisorRecommendation(context.Background(), params, cfg)
	if err == nil {
		t.Error("Expected error for invalid operation, got nil")
	}

	expectedError := "invalid operation: invalid_operation"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
	}
}

func TestHandleAdvisorRecommendationMissingOperation(t *testing.T) {
	cfg := &config.ConfigData{}
	params := map[string]interface{}{}

	_, err := HandleAdvisorRecommendation(context.Background(), params, cfg)
	if err == nil {
		t.Error("Expected error for missing operation, got nil")
	}

	expectedError := "operation parameter is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error %s, got %s", expectedError, err.Error())
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || s[len(s)-len(substr):] == substr || s[:len(substr)] == substr || containsInMiddle(s, substr))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
