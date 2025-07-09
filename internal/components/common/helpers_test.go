package common

import (
	"testing"
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
