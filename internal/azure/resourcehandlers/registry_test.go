package resourcehandlers

import (
	"testing"

	"github.com/Azure/aks-mcp/internal/azure"
	"github.com/Azure/aks-mcp/internal/config"
)

func TestRegisterVNetInfoTool(t *testing.T) {
	tool := RegisterVNetInfoTool()

	if tool.Name != "get_vnet_info" {
		t.Errorf("Expected tool name 'get_vnet_info', got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}
}

func TestRegisterNSGInfoTool(t *testing.T) {
	tool := RegisterNSGInfoTool()

	if tool.Name != "get_nsg_info" {
		t.Errorf("Expected tool name 'get_nsg_info', got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}
}

func TestRegisterRouteTableInfoTool(t *testing.T) {
	tool := RegisterRouteTableInfoTool()

	if tool.Name != "get_route_table_info" {
		t.Errorf("Expected tool name 'get_route_table_info', got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}
}

func TestRegisterSubnetInfoTool(t *testing.T) {
	tool := RegisterSubnetInfoTool()

	if tool.Name != "get_subnet_info" {
		t.Errorf("Expected tool name 'get_subnet_info', got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}
}

func TestGetVNetInfoHandler(t *testing.T) {
	mockClient := &azure.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetVNetInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetNSGInfoHandler(t *testing.T) {
	mockClient := &azure.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetNSGInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetRouteTableInfoHandler(t *testing.T) {
	mockClient := &azure.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetRouteTableInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetSubnetInfoHandler(t *testing.T) {
	mockClient := &azure.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetSubnetInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

// =============================================================================
// Monitoring-related tool registration tests
// =============================================================================

func TestRegisterLogAnalyticsQueryTool(t *testing.T) {
	tool := RegisterLogAnalyticsQueryTool()

	if tool.Name != "query_log_analytics" {
		t.Errorf("Expected tool name 'query_log_analytics', got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}

	// Verify required parameters are present in schema
	expectedRequiredParams := []string{"subscription_id", "resource_group", "cluster_name", "workspace_id", "kql_query"}
	for _, param := range expectedRequiredParams {
		found := false
		for _, required := range tool.InputSchema.Required {
			if required == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected required parameter '%s' not found in required list", param)
		}
		
		// Also verify the parameter exists in properties
		if _, exists := tool.InputSchema.Properties[param]; !exists {
			t.Errorf("Expected parameter '%s' not found in properties", param)
		}
	}
}

func TestRegisterPrometheusMetricsQueryTool(t *testing.T) {
	tool := RegisterPrometheusMetricsQueryTool()

	if tool.Name != "query_prometheus_metrics" {
		t.Errorf("Expected tool name 'query_prometheus_metrics', got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}

	// Verify required parameters are present in schema
	expectedRequiredParams := []string{"subscription_id", "resource_group", "cluster_name", "metric_names"}
	for _, param := range expectedRequiredParams {
		found := false
		for _, required := range tool.InputSchema.Required {
			if required == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected required parameter '%s' not found in required list", param)
		}
		
		// Also verify the parameter exists in properties
		if _, exists := tool.InputSchema.Properties[param]; !exists {
			t.Errorf("Expected parameter '%s' not found in properties", param)
		}
	}
}

func TestRegisterApplicationInsightsQueryTool(t *testing.T) {
	tool := RegisterApplicationInsightsQueryTool()

	if tool.Name != "query_application_insights" {
		t.Errorf("Expected tool name 'query_application_insights', got %s", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected tool description to be set")
	}

	// Verify required parameters are present in schema
	expectedRequiredParams := []string{"subscription_id", "resource_group", "cluster_name", "app_insights_name", "kql_query"}
	for _, param := range expectedRequiredParams {
		found := false
		for _, required := range tool.InputSchema.Required {
			if required == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected required parameter '%s' not found in required list", param)
		}
		
		// Also verify the parameter exists in properties
		if _, exists := tool.InputSchema.Properties[param]; !exists {
			t.Errorf("Expected parameter '%s' not found in properties", param)
		}
	}
}

func TestGetLogAnalyticsHandler(t *testing.T) {
	mockClient := &azure.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetLogAnalyticsHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetPrometheusMetricsHandler(t *testing.T) {
	mockClient := &azure.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetPrometheusMetricsHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetApplicationInsightsHandler(t *testing.T) {
	mockClient := &azure.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetApplicationInsightsHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}
