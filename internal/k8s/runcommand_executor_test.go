package k8s

import (
	"context"
	"testing"

	"github.com/Azure/aks-mcp/internal/ctx"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
	k8ssecurity "github.com/Azure/mcp-kubernetes/pkg/security"
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

// Test validateCommand with different access levels
func TestValidateCommand_AccessLevels(t *testing.T) {
	executor := &RunCommandExecutor{}

	testCases := []struct {
		name        string
		command     string
		accessLevel k8ssecurity.AccessLevel
		expectError bool
		errorMsg    string
	}{
		{
			name:        "readonly allows get command",
			command:     "kubectl get pods",
			accessLevel: k8ssecurity.AccessLevelReadOnly,
			expectError: false,
		},
		{
			name:        "readonly allows describe command",
			command:     "kubectl describe pod mypod",
			accessLevel: k8ssecurity.AccessLevelReadOnly,
			expectError: false,
		},
		{
			name:        "readonly rejects delete command",
			command:     "kubectl delete pod mypod",
			accessLevel: k8ssecurity.AccessLevelReadOnly,
			expectError: true,
			errorMsg:    "security validation failed",
		},
		{
			name:        "readonly rejects apply command",
			command:     "kubectl apply -f deployment.yaml",
			accessLevel: k8ssecurity.AccessLevelReadOnly,
			expectError: true,
			errorMsg:    "security validation failed",
		},
		{
			name:        "readwrite allows get command",
			command:     "kubectl get pods",
			accessLevel: k8ssecurity.AccessLevelReadWrite,
			expectError: false,
		},
		{
			name:        "readwrite allows delete command",
			command:     "kubectl delete pod mypod",
			accessLevel: k8ssecurity.AccessLevelReadWrite,
			expectError: false,
		},
		{
			name:        "readwrite allows apply command",
			command:     "kubectl apply -f deployment.yaml",
			accessLevel: k8ssecurity.AccessLevelReadWrite,
			expectError: false,
		},
		{
			name:        "readwrite rejects drain command",
			command:     "kubectl drain node1",
			accessLevel: k8ssecurity.AccessLevelReadWrite,
			expectError: true,
			errorMsg:    "security validation failed",
		},
		{
			name:        "readwrite rejects cordon command",
			command:     "kubectl cordon node1",
			accessLevel: k8ssecurity.AccessLevelReadWrite,
			expectError: true,
			errorMsg:    "security validation failed",
		},
		{
			name:        "admin allows get command",
			command:     "kubectl get pods",
			accessLevel: k8ssecurity.AccessLevelAdmin,
			expectError: false,
		},
		{
			name:        "admin allows delete command",
			command:     "kubectl delete pod mypod",
			accessLevel: k8ssecurity.AccessLevelAdmin,
			expectError: false,
		},
		{
			name:        "admin allows drain command",
			command:     "kubectl drain node1",
			accessLevel: k8ssecurity.AccessLevelAdmin,
			expectError: false,
		},
		{
			name:        "admin allows cordon command",
			command:     "kubectl cordon node1",
			accessLevel: k8ssecurity.AccessLevelAdmin,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			secConfig := k8ssecurity.NewSecurityConfig()
			secConfig.AccessLevel = tc.accessLevel
			cfg := &k8sconfig.ConfigData{
				SecurityConfig: secConfig,
			}

			err := executor.validateCommand(tc.command, cfg)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tc.errorMsg)
				} else if !containsError(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// Test validateCommand with namespace restrictions
func TestValidateCommand_NamespaceRestrictions(t *testing.T) {
	executor := &RunCommandExecutor{}

	testCases := []struct {
		name              string
		command           string
		allowedNamespaces string
		expectError       bool
		errorMsg          string
	}{
		{
			name:              "no namespace restriction allows any namespace",
			command:           "kubectl get pods -n kube-system",
			allowedNamespaces: "",
			expectError:       false,
		},
		{
			name:              "namespace restriction allows specified namespace",
			command:           "kubectl get pods -n default",
			allowedNamespaces: "default,kube-system",
			expectError:       false,
		},
		{
			name:              "namespace restriction rejects non-allowed namespace",
			command:           "kubectl get pods -n forbidden",
			allowedNamespaces: "default,kube-system",
			expectError:       true,
			errorMsg:          "security validation failed",
		},
		{
			name:              "namespace restriction allows all listed namespaces",
			command:           "kubectl get pods -n kube-system",
			allowedNamespaces: "default,kube-system",
			expectError:       false,
		},
		{
			name:              "namespace restriction with --namespace flag",
			command:           "kubectl get pods --namespace=production",
			allowedNamespaces: "default,production",
			expectError:       false,
		},
		{
			name:              "namespace restriction rejects --all-namespaces",
			command:           "kubectl get pods --all-namespaces",
			allowedNamespaces: "default",
			expectError:       true,
			errorMsg:          "security validation failed",
		},
		{
			name:              "namespace restriction rejects -A flag",
			command:           "kubectl get pods -A",
			allowedNamespaces: "default",
			expectError:       true,
			errorMsg:          "security validation failed",
		},
		{
			name:              "no restriction allows --all-namespaces",
			command:           "kubectl get pods --all-namespaces",
			allowedNamespaces: "",
			expectError:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			secConfig := k8ssecurity.NewSecurityConfig()
			secConfig.AccessLevel = k8ssecurity.AccessLevelReadOnly
			secConfig.SetAllowedNamespaces(tc.allowedNamespaces)
			cfg := &k8sconfig.ConfigData{
				SecurityConfig: secConfig,
			}

			err := executor.validateCommand(tc.command, cfg)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tc.errorMsg)
				} else if !containsError(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// Test validateCommand with combined access level and namespace restrictions
func TestValidateCommand_Combined(t *testing.T) {
	executor := &RunCommandExecutor{}

	testCases := []struct {
		name              string
		command           string
		accessLevel       k8ssecurity.AccessLevel
		allowedNamespaces string
		expectError       bool
		errorMsg          string
	}{
		{
			name:              "readonly with namespace restriction allows allowed read operation",
			command:           "kubectl get pods -n default",
			accessLevel:       k8ssecurity.AccessLevelReadOnly,
			allowedNamespaces: "default",
			expectError:       false,
		},
		{
			name:              "readonly with namespace restriction rejects write in allowed namespace",
			command:           "kubectl delete pod mypod -n default",
			accessLevel:       k8ssecurity.AccessLevelReadOnly,
			allowedNamespaces: "default",
			expectError:       true,
			errorMsg:          "security validation failed",
		},
		{
			name:              "readonly with namespace restriction rejects read in forbidden namespace",
			command:           "kubectl get pods -n kube-system",
			accessLevel:       k8ssecurity.AccessLevelReadOnly,
			allowedNamespaces: "default",
			expectError:       true,
			errorMsg:          "security validation failed",
		},
		{
			name:              "readwrite with namespace restriction allows write in allowed namespace",
			command:           "kubectl delete pod mypod -n production",
			accessLevel:       k8ssecurity.AccessLevelReadWrite,
			allowedNamespaces: "production,staging",
			expectError:       false,
		},
		{
			name:              "readwrite with namespace restriction rejects write in forbidden namespace",
			command:           "kubectl delete pod mypod -n forbidden",
			accessLevel:       k8ssecurity.AccessLevelReadWrite,
			allowedNamespaces: "production,staging",
			expectError:       true,
			errorMsg:          "security validation failed",
		},
		{
			name:              "admin with namespace restriction allows admin operation in allowed namespace",
			command:           "kubectl drain node1",
			accessLevel:       k8ssecurity.AccessLevelAdmin,
			allowedNamespaces: "default",
			expectError:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			secConfig := k8ssecurity.NewSecurityConfig()
			secConfig.AccessLevel = tc.accessLevel
			secConfig.SetAllowedNamespaces(tc.allowedNamespaces)
			cfg := &k8sconfig.ConfigData{
				SecurityConfig: secConfig,
			}

			err := executor.validateCommand(tc.command, cfg)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tc.errorMsg)
				} else if !containsError(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
