package common

import (
	"os"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

// TestExtractAKSParameters tests the parameter extraction function
func TestExtractAKSParameters(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"subscription_id": "test-sub",
				"resource_group":  "test-rg",
				"cluster_name":    "test-cluster",
			},
			wantErr: false,
		},
		{
			name: "missing subscription_id",
			params: map[string]interface{}{
				"resource_group": "test-rg",
				"cluster_name":   "test-cluster",
			},
			wantErr: true,
		},
		{
			name: "missing resource_group",
			params: map[string]interface{}{
				"subscription_id": "test-sub",
				"cluster_name":    "test-cluster",
			},
			wantErr: true,
		},
		{
			name: "missing cluster_name",
			params: map[string]interface{}{
				"subscription_id": "test-sub",
				"resource_group":  "test-rg",
			},
			wantErr: true,
		},
		{
			name: "empty subscription_id",
			params: map[string]interface{}{
				"subscription_id": "",
				"resource_group":  "test-rg",
				"cluster_name":    "test-cluster",
			},
			wantErr: true,
		},
		{
			name: "empty resource_group",
			params: map[string]interface{}{
				"subscription_id": "test-sub",
				"resource_group":  "",
				"cluster_name":    "test-cluster",
			},
			wantErr: true,
		},
		{
			name: "empty cluster_name",
			params: map[string]interface{}{
				"subscription_id": "test-sub",
				"resource_group":  "test-rg",
				"cluster_name":    "",
			},
			wantErr: true,
		},
		{
			name: "invalid parameter types",
			params: map[string]interface{}{
				"subscription_id": 123,
				"resource_group":  "test-rg",
				"cluster_name":    "test-cluster",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subID, rg, clusterName, err := ExtractAKSParameters(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractAKSParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if subID == "" || rg == "" || clusterName == "" {
					t.Errorf("ExtractAKSParameters() returned empty values: subID=%s, rg=%s, clusterName=%s", subID, rg, clusterName)
				}
			}
		})
	}
}

func TestGetDefaultSubscriptionID_FromEnv(t *testing.T) {
	originalEnv := os.Getenv("AZURE_SUBSCRIPTION_ID")
	defer os.Setenv("AZURE_SUBSCRIPTION_ID", originalEnv)

	expectedSubID := "test-subscription-id-123"
	os.Setenv("AZURE_SUBSCRIPTION_ID", expectedSubID)

	cfg := &config.ConfigData{
		Timeout: 30000,
	}

	subID, err := GetDefaultSubscriptionID(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if subID != expectedSubID {
		t.Errorf("Expected subscription ID %s, got %s", expectedSubID, subID)
	}
}

func TestGetDefaultSubscriptionID_NoEnv(t *testing.T) {
	t.Skip("Skipping test that requires Azure CLI authentication")
}
