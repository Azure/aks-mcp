package applens

import (
	"context"
	"testing"
	"time"
)

func TestValidateClusterResourceID(t *testing.T) {
	tests := []struct {
		name        string
		resourceID  string
		shouldError bool
	}{
		{
			name:        "valid AKS cluster resource ID",
			resourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test",
			shouldError: false,
		},
		{
			name:        "empty resource ID",
			resourceID:  "",
			shouldError: true,
		},
		{
			name:        "invalid format - too short",
			resourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012",
			shouldError: true,
		},
		{
			name:        "invalid provider",
			resourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.Compute/virtualMachines/vm-test",
			shouldError: true,
		},
		{
			name:        "invalid resource type",
			resourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/containerGroups/aci-test",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ExtractClusterInfo(tt.resourceID)
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestExtractClusterInfo(t *testing.T) {
	resourceID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test"

	subscriptionID, resourceGroup, clusterName, err := ExtractClusterInfo(resourceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSubscriptionID := "12345678-1234-1234-1234-123456789012"
	expectedResourceGroup := "rg-test"
	expectedClusterName := "aks-test"

	if subscriptionID != expectedSubscriptionID {
		t.Errorf("expected subscription ID %s, got %s", expectedSubscriptionID, subscriptionID)
	}
	if resourceGroup != expectedResourceGroup {
		t.Errorf("expected resource group %s, got %s", expectedResourceGroup, resourceGroup)
	}
	if clusterName != expectedClusterName {
		t.Errorf("expected cluster name %s, got %s", expectedClusterName, clusterName)
	}
}

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		name        string
		timeRange   string
		expected    time.Duration
		shouldError bool
	}{
		{"1 hour", "1h", time.Hour, false},
		{"24 hours", "24h", 24 * time.Hour, false},
		{"1 day", "1d", 24 * time.Hour, false},
		{"7 days", "7d", 7 * 24 * time.Hour, false},
		{"1 week", "1w", 7 * 24 * time.Hour, false},
		{"30 days", "30d", 30 * 24 * time.Hour, false},
		{"1 month", "1m", 30 * 24 * time.Hour, false},
		{"invalid", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := parseTimeRange(tt.timeRange)
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.shouldError && duration != tt.expected {
				t.Errorf("expected duration %v, got %v", tt.expected, duration)
			}
		})
	}
}

func TestNewAppLensClient(t *testing.T) {
	// Test with empty subscription ID
	_, err := NewAppLensClient("", nil)
	if err == nil {
		t.Error("expected error for empty subscription ID")
	}

	// Test with valid subscription ID (note: credential can be nil for this test)
	client, err := NewAppLensClient("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if client == nil {
		t.Error("expected client to be created")
	}
}

// TestNewDetectorManager tests
func TestNewDetectorManager(t *testing.T) {
	// Test with empty subscription ID
	_, err := NewDetectorManager("", nil)
	if err == nil {
		t.Error("expected error for empty subscription ID")
	}

	// Test with valid subscription ID (note: credential can be nil for this test)
	manager, err := NewDetectorManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if manager == nil {
		t.Error("expected detector manager to be created")
	}
}

// TestListDetectors tests the detector listing functionality
// Note: This test will fail in CI/local without proper Azure credentials
// In a real implementation, you'd mock the HTTP client or use integration test environment
func TestListDetectors(t *testing.T) {
	// Skip this test in CI environment since it requires real Azure credentials
	t.Skip("Skipping integration test - requires Azure credentials")

	// Create a detector manager
	manager, err := NewDetectorManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Fatalf("failed to create detector manager: %v", err)
	}

	clusterResourceID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test"

	// Test listing all detectors
	result, err := manager.ListDetectors(context.Background(), clusterResourceID, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// TestInvokeDetector tests the detector invocation functionality
// Note: This test will fail in CI/local without proper Azure credentials
// In a real implementation, you'd mock the HTTP client or use integration test environment
func TestInvokeDetector(t *testing.T) {
	// Skip this test in CI environment since it requires real Azure credentials
	t.Skip("Skipping integration test - requires Azure credentials")

	// Create a detector manager
	manager, err := NewDetectorManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Fatalf("failed to create detector manager: %v", err)
	}

	clusterResourceID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test"

	// Test invoking a specific detector
	result, err := manager.InvokeDetector(context.Background(), clusterResourceID, "cluster-health", "24h")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}
