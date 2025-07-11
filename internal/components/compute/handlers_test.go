package compute

import (
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

// TestGetVMSSInfoByNodePoolHandler tests the VMSS info by node pool handler with mock data
func TestGetVMSSInfoByNodePoolHandler(t *testing.T) {
	cfg := &config.ConfigData{}

	// Test with nil client
	handler := GetVMSSInfoByNodePoolHandler(nil, cfg)
	if handler == nil {
		t.Error("GetVMSSInfoByNodePoolHandler() returned nil")
	}

	// Test with missing node_pool_name parameter
	params := map[string]interface{}{
		"subscription_id": "test-sub",
		"resource_group":  "test-rg",
		"cluster_name":    "test-cluster",
		// missing node_pool_name
	}

	_, err := handler.Handle(params, cfg)
	if err == nil {
		t.Error("Expected error with missing node_pool_name parameter, got nil")
	}
}

// TestGetAllVMSSByClusterHandler tests the all VMSS by cluster handler with mock data
func TestGetAllVMSSByClusterHandler(t *testing.T) {
	cfg := &config.ConfigData{}

	// Test with nil client
	handler := GetAllVMSSByClusterHandler(nil, cfg)
	if handler == nil {
		t.Error("GetAllVMSSByClusterHandler() returned nil")
	}

	// Test with invalid parameters
	params := map[string]interface{}{
		"invalid": "params",
	}

	_, err := handler.Handle(params, cfg)
	if err == nil {
		t.Error("Expected error with invalid parameters, got nil")
	}
}
