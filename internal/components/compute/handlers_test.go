package compute

import (
	"context"
	"testing"

	"github.com/Azure/aks-mcp/internal/components/common"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subID, rg, clusterName, err := common.ExtractAKSParameters(tt.params)
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

// TestGetAKSVMSSInfoHandler tests the AKS VMSS info handler basic functionality
func TestGetAKSVMSSInfoHandler(t *testing.T) {
	cfg := &config.ConfigData{}

	// Test that the handler function is created successfully
	handler := GetAKSVMSSInfoHandler(nil, cfg)
	if handler == nil {
		t.Error("GetAKSVMSSInfoHandler() returned nil")
		return
	}

	// Test with completely invalid parameters (should fail parameter extraction)
	invalidParams := map[string]interface{}{
		"invalid": "params",
	}

	_, err := handler.Handle(context.Background(), invalidParams, cfg)
	if err == nil {
		t.Error("Expected error with invalid parameters, got nil")
		return
	}

	// Verify the error is about missing required parameters, not about node_pool_name specifically
	if err.Error() == "missing or invalid node_pool_name parameter" {
		t.Error("Handler should not require node_pool_name parameter - it should be optional")
	} else {
		t.Logf("Expected error for missing parameters: %v", err.Error())
	}

	// Note: We don't test with valid parameters because that would require a real Azure client
	// and would make actual Azure API calls. The handler creation test above is sufficient
	// to verify the basic functionality works.
}
