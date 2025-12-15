package network

import (
	"context"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

// TestGetLoadBalancersInfoHandler tests the load balancers info handler
func TestGetLoadBalancersInfoHandler(t *testing.T) {
	cfg := &config.ConfigData{}

	t.Run("missing subscription_id parameter", func(t *testing.T) {
		params := map[string]interface{}{
			"resource_group": "rg-test",
			"cluster_name":   "cluster-test",
		}

		handler := GetLoadBalancersInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

		handler := GetLoadBalancersInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

		handler := GetLoadBalancersInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

		handler := GetLoadBalancersInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

		handler := GetLoadBalancersInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

// TestGetPrivateEndpointInfoHandlerValidation tests the private endpoint info handler parameter validation
func TestGetPrivateEndpointInfoHandlerValidation(t *testing.T) {
	cfg := &config.ConfigData{}

	t.Run("missing subscription_id parameter", func(t *testing.T) {
		params := map[string]interface{}{
			"resource_group": "rg-test",
			"cluster_name":   "cluster-test",
		}

		handler := GetPrivateEndpointInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

		handler := GetPrivateEndpointInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

		handler := GetPrivateEndpointInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

		handler := GetPrivateEndpointInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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

		handler := GetPrivateEndpointInfoHandler(nil, cfg)
		result, err := handler.Handle(context.Background(), params, cfg)

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
