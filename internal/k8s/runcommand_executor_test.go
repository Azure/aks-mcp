package k8s

import (
	"context"
	"testing"

	"github.com/Azure/aks-mcp/internal/ctx"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
)

func TestExtractRequestContext_Success(t *testing.T) {
	c := context.WithValue(context.Background(), ctx.AzureTokenKey, "test-token-123")

	params := map[string]interface{}{
		"aks_resource_id": "/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/cluster-test",
	}

	reqCtx, err := extractRequestContext(c, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if reqCtx.AzureToken != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", reqCtx.AzureToken)
	}

	if reqCtx.SubscriptionID != "sub-123" {
		t.Errorf("Expected subscription 'sub-123', got '%s'", reqCtx.SubscriptionID)
	}

	if reqCtx.ResourceGroup != "rg-test" {
		t.Errorf("Expected resource group 'rg-test', got '%s'", reqCtx.ResourceGroup)
	}

	if reqCtx.ClusterName != "cluster-test" {
		t.Errorf("Expected cluster name 'cluster-test', got '%s'", reqCtx.ClusterName)
	}
}

func TestExtractRequestContext_MissingToken(t *testing.T) {
	c := context.Background()

	params := map[string]interface{}{
		"aks_resource_id": "/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/cluster-test",
	}

	_, err := extractRequestContext(c, params)
	if err == nil {
		t.Fatal("Expected error for missing token, got nil")
	}

	expectedMsg := "X-Azure-Token not found in context or empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestExtractRequestContext_EmptyToken(t *testing.T) {
	c := context.WithValue(context.Background(), ctx.AzureTokenKey, "")

	params := map[string]interface{}{
		"aks_resource_id": "/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/cluster-test",
	}

	_, err := extractRequestContext(c, params)
	if err == nil {
		t.Fatal("Expected error for empty token, got nil")
	}

	expectedMsg := "X-Azure-Token not found in context or empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestExtractRequestContext_MissingResourceID(t *testing.T) {
	c := context.WithValue(context.Background(), ctx.AzureTokenKey, "test-token-123")

	params := map[string]interface{}{}

	_, err := extractRequestContext(c, params)
	if err == nil {
		t.Fatal("Expected error for missing resource ID, got nil")
	}

	expectedMsg := "aks_resource_id not found in params or empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestExtractRequestContext_EmptyResourceID(t *testing.T) {
	c := context.WithValue(context.Background(), ctx.AzureTokenKey, "test-token-123")

	params := map[string]interface{}{
		"aks_resource_id": "",
	}

	_, err := extractRequestContext(c, params)
	if err == nil {
		t.Fatal("Expected error for empty resource ID, got nil")
	}

	expectedMsg := "aks_resource_id not found in params or empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestExtractRequestContext_InvalidResourceID(t *testing.T) {
	c := context.WithValue(context.Background(), ctx.AzureTokenKey, "test-token-123")

	testCases := []struct {
		name        string
		resourceID  string
		expectError bool
	}{
		{
			name:        "Invalid format",
			resourceID:  "invalid-resource-id",
			expectError: true,
		},
		{
			name:        "Missing cluster name",
			resourceID:  "/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/",
			expectError: true,
		},
		{
			name:        "Wrong provider",
			resourceID:  "/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.Compute/virtualMachines/vm-test",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := map[string]interface{}{
				"aks_resource_id": tc.resourceID,
			}

			_, err := extractRequestContext(c, params)
			if tc.expectError && err == nil {
				t.Fatal("Expected error, got nil")
			}
			if tc.expectError && err != nil {
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}

func TestBuildCommand_ValidCommand(t *testing.T) {
	executor := &RunCommandExecutor{}

	params := map[string]interface{}{
		"command": "kubectl get pods -n default",
	}

	command, err := executor.buildCommand(params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if command != "kubectl get pods -n default" {
		t.Errorf("Expected command 'kubectl get pods -n default', got '%s'", command)
	}
}

func TestBuildCommand_MissingCommand(t *testing.T) {
	executor := &RunCommandExecutor{}

	params := map[string]interface{}{}

	_, err := executor.buildCommand(params)
	if err == nil {
		t.Fatal("Expected error for missing command, got nil")
	}
}

func TestBuildCommand_EmptyCommand(t *testing.T) {
	executor := &RunCommandExecutor{}

	params := map[string]interface{}{
		"command": "",
	}

	command, err := executor.buildCommand(params)
	if err != nil {
		t.Fatalf("Expected no error for empty string, got: %v", err)
	}

	if command != "" {
		t.Errorf("Expected empty command, got '%s'", command)
	}
}

func TestBuildCommand_InvalidCommandType(t *testing.T) {
	executor := &RunCommandExecutor{}

	params := map[string]interface{}{
		"command": 123,
	}

	_, err := executor.buildCommand(params)
	if err == nil {
		t.Fatal("Expected error for invalid command type, got nil")
	}
}

func TestExecute_MissingContext(t *testing.T) {
	executor := &RunCommandExecutor{}

	params := map[string]interface{}{
		"command":         "kubectl get pods",
		"aks_resource_id": "/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/cluster-test",
	}

	cfg := &k8sconfig.ConfigData{}
	c := context.Background()

	_, err := executor.Execute(c, params, cfg)
	if err == nil {
		t.Fatal("Expected error for missing context, got nil")
	}

	if err.Error() != "failed to extract request context: X-Azure-Token not found in context or empty" {
		t.Errorf("Expected context error, got: %v", err)
	}
}

func TestExecute_InvalidParams(t *testing.T) {
	executor := &RunCommandExecutor{}

	c := context.WithValue(context.Background(), ctx.AzureTokenKey, "test-token-123")

	testCases := []struct {
		name        string
		params      map[string]interface{}
		expectedErr string
	}{
		{
			name: "Missing aks_resource_id",
			params: map[string]interface{}{
				"command": "kubectl get pods",
			},
			expectedErr: "failed to extract request context: aks_resource_id not found in params or empty",
		},
		{
			name: "Invalid aks_resource_id",
			params: map[string]interface{}{
				"command":         "kubectl get pods",
				"aks_resource_id": "invalid",
			},
			expectedErr: "failed to extract request context: failed to parse aks_resource_id",
		},
		{
			name: "Missing command",
			params: map[string]interface{}{
				"aks_resource_id": "/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/cluster-test",
			},
			expectedErr: "failed to build command: command parameter is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &k8sconfig.ConfigData{}
			_, err := executor.Execute(c, tc.params, cfg)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !containsError(err.Error(), tc.expectedErr) {
				t.Errorf("Expected error containing '%s', got '%s'", tc.expectedErr, err.Error())
			}
		})
	}
}

func containsError(actual, expected string) bool {
	return len(actual) >= len(expected) && actual[:len(expected)] == expected
}
