package resourcehandlers

import (
	"testing"
)

func TestExtractAKSParameters(t *testing.T) {
	tests := []struct {
		name            string
		params          map[string]interface{}
		expectedSubID   string
		expectedRG      string
		expectedCluster string
		expectError     bool
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"resource_group":  "rg-test",
				"cluster_name":    "cluster-test",
			},
			expectedSubID:   "sub-123",
			expectedRG:      "rg-test",
			expectedCluster: "cluster-test",
			expectError:     false,
		},
		{
			name: "missing subscription_id",
			params: map[string]interface{}{
				"resource_group": "rg-test",
				"cluster_name":   "cluster-test",
			},
			expectError: true,
		},
		{
			name: "empty subscription_id",
			params: map[string]interface{}{
				"subscription_id": "",
				"resource_group":  "rg-test",
				"cluster_name":    "cluster-test",
			},
			expectError: true,
		},
		{
			name: "missing resource_group",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"cluster_name":    "cluster-test",
			},
			expectError: true,
		},
		{
			name: "missing cluster_name",
			params: map[string]interface{}{
				"subscription_id": "sub-123",
				"resource_group":  "rg-test",
			},
			expectError: true,
		},
		{
			name: "invalid parameter types",
			params: map[string]interface{}{
				"subscription_id": 123,
				"resource_group":  "rg-test",
				"cluster_name":    "cluster-test",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subID, rg, clusterName, err := ExtractAKSParameters(tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if subID != tt.expectedSubID {
					t.Errorf("Expected subscription ID %s, got %s", tt.expectedSubID, subID)
				}
				if rg != tt.expectedRG {
					t.Errorf("Expected resource group %s, got %s", tt.expectedRG, rg)
				}
				if clusterName != tt.expectedCluster {
					t.Errorf("Expected cluster name %s, got %s", tt.expectedCluster, clusterName)
				}
			}
		})
	}
}

// TestGetLoadBalancersInfoHandler tests the load balancers info handler
func TestGetLoadBalancersInfoHandler(t *testing.T) {
	t.Run("missing subscription_id parameter", func(t *testing.T) {
		params := map[string]interface{}{
			"resource_group": "rg-test",
			"cluster_name":   "cluster-test",
		}

		handler := GetLoadBalancersInfoHandler(nil, nil)
		result, err := handler.Handle(params, nil)

		if err == nil {
			t.Error("Expected error for missing subscription_id")
		}
		if result != "" {
			t.Error("Expected empty result on error")
		}
		if err.Error() != "missing or invalid subscription_id parameter" {
			t.Errorf("Expected 'missing or invalid subscription_id parameter' error, got %v", err)
		}
	})

	t.Run("missing resource_group parameter", func(t *testing.T) {
		params := map[string]interface{}{
			"subscription_id": "sub-123",
			"cluster_name":    "cluster-test",
		}

		handler := GetLoadBalancersInfoHandler(nil, nil)
		result, err := handler.Handle(params, nil)

		if err == nil {
			t.Error("Expected error for missing resource_group")
		}
		if result != "" {
			t.Error("Expected empty result on error")
		}
		if err.Error() != "missing or invalid resource_group parameter" {
			t.Errorf("Expected 'missing or invalid resource_group parameter' error, got %v", err)
		}
	})

	t.Run("missing cluster_name parameter", func(t *testing.T) {
		params := map[string]interface{}{
			"subscription_id": "sub-123",
			"resource_group":  "rg-test",
		}

		handler := GetLoadBalancersInfoHandler(nil, nil)
		result, err := handler.Handle(params, nil)

		if err == nil {
			t.Error("Expected error for missing cluster_name")
		}
		if result != "" {
			t.Error("Expected empty result on error")
		}
		if err.Error() != "missing or invalid cluster_name parameter" {
			t.Errorf("Expected 'missing or invalid cluster_name parameter' error, got %v", err)
		}
	})

	t.Run("empty subscription_id parameter", func(t *testing.T) {
		params := map[string]interface{}{
			"subscription_id": "",
			"resource_group":  "rg-test",
			"cluster_name":    "cluster-test",
		}

		handler := GetLoadBalancersInfoHandler(nil, nil)
		result, err := handler.Handle(params, nil)

		if err == nil {
			t.Error("Expected error for empty subscription_id")
		}
		if result != "" {
			t.Error("Expected empty result on error")
		}
		if err.Error() != "missing or invalid subscription_id parameter" {
			t.Errorf("Expected 'missing or invalid subscription_id parameter' error, got %v", err)
		}
	})

	t.Run("invalid parameter types", func(t *testing.T) {
		params := map[string]interface{}{
			"subscription_id": 123, // Should be string
			"resource_group":  "rg-test",
			"cluster_name":    "cluster-test",
		}

		handler := GetLoadBalancersInfoHandler(nil, nil)
		result, err := handler.Handle(params, nil)

		if err == nil {
			t.Error("Expected error for invalid parameter type")
		}
		if result != "" {
			t.Error("Expected empty result on error")
		}
		if err.Error() != "missing or invalid subscription_id parameter" {
			t.Errorf("Expected 'missing or invalid subscription_id parameter' error, got %v", err)
		}
	})

	// Note: Testing with valid parameters and actual Azure client calls
	// would require integration tests with mocked Azure services
}
