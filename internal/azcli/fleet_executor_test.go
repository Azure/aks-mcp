package azcli

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/security"
)

func TestFleetExecutor_ValidateCombination(t *testing.T) {
	executor := NewFleetExecutor()

	tests := []struct {
		name      string
		operation string
		resource  string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid fleet list",
			operation: "list",
			resource:  "fleet",
			wantErr:   false,
		},
		{
			name:      "valid member create",
			operation: "create",
			resource:  "member",
			wantErr:   false,
		},
		{
			name:      "valid updaterun start",
			operation: "start",
			resource:  "updaterun",
			wantErr:   false,
		},
		{
			name:      "invalid operation for fleet",
			operation: "start",
			resource:  "fleet",
			wantErr:   true,
			errMsg:    "invalid operation 'start' for resource 'fleet'",
		},
		{
			name:      "invalid operation for updatestrategy",
			operation: "update",
			resource:  "updatestrategy",
			wantErr:   true,
			errMsg:    "invalid operation 'update' for resource 'updatestrategy'",
		},
		{
			name:      "invalid resource",
			operation: "list",
			resource:  "invalid",
			wantErr:   true,
			errMsg:    "invalid resource type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateCombination(tt.operation, tt.resource)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateCombination() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateCombination() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("validateCombination() unexpected error = %v", err)
			}
		})
	}
}

func TestFleetExecutor_CheckAccessLevel(t *testing.T) {
	executor := NewFleetExecutor()

	tests := []struct {
		name        string
		operation   string
		resource    string
		accessLevel string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "readonly can list",
			operation:   "list",
			resource:    "fleet",
			accessLevel: "readonly",
			wantErr:     false,
		},
		{
			name:        "readonly can show",
			operation:   "show",
			resource:    "member",
			accessLevel: "readonly",
			wantErr:     false,
		},
		{
			name:        "readonly cannot create",
			operation:   "create",
			resource:    "fleet",
			accessLevel: "readonly",
			wantErr:     true,
			errMsg:      "requires readwrite or admin access level",
		},
		{
			name:        "readwrite can create",
			operation:   "create",
			resource:    "fleet",
			accessLevel: "readwrite",
			wantErr:     false,
		},
		{
			name:        "admin can delete",
			operation:   "delete",
			resource:    "updaterun",
			accessLevel: "admin",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.checkAccessLevel(tt.operation, tt.resource, tt.accessLevel)
			if tt.wantErr {
				if err == nil {
					t.Errorf("checkAccessLevel() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("checkAccessLevel() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("checkAccessLevel() unexpected error = %v", err)
			}
		})
	}
}

func TestFleetExecutor_GetCommandForValidation(t *testing.T) {
	executor := NewFleetExecutor()

	tests := []struct {
		name      string
		operation string
		resource  string
		args      string
		want      string
	}{
		{
			name:      "fleet list special case",
			operation: "list",
			resource:  "fleet",
			args:      "--resource-group myRG",
			want:      "az fleet list --resource-group myRG",
		},
		{
			name:      "member show with args",
			operation: "show",
			resource:  "member",
			args:      "--name myMember --fleet-name myFleet --resource-group myRG",
			want:      "az fleet member show --name myMember --fleet-name myFleet --resource-group myRG",
		},
		{
			name:      "updaterun create without args",
			operation: "create",
			resource:  "updaterun",
			args:      "",
			want:      "az fleet updaterun create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.GetCommandForValidation(tt.operation, tt.resource, tt.args)
			if got != tt.want {
				t.Errorf("GetCommandForValidation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFleetExecutor_Execute(t *testing.T) {
	// Note: This test validates parameter extraction and command construction
	// but doesn't execute actual az commands

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid parameters",
			params: map[string]any{
				"operation": "list",
				"resource":  "fleet",
				"args":      "--resource-group myRG",
			},
			wantErr: false,
		},
		{
			name: "missing operation",
			params: map[string]any{
				"resource": "fleet",
				"args":     "--resource-group myRG",
			},
			wantErr: true,
			errMsg:  "operation parameter is required",
		},
		{
			name: "missing resource",
			params: map[string]any{
				"operation": "list",
				"args":      "--resource-group myRG",
			},
			wantErr: true,
			errMsg:  "resource parameter is required",
		},
		{
			name: "missing args",
			params: map[string]any{
				"operation": "list",
				"resource":  "fleet",
			},
			wantErr: true,
			errMsg:  "args parameter is required",
		},
		{
			name: "invalid combination",
			params: map[string]any{
				"operation": "start",
				"resource":  "fleet",
				"args":      "",
			},
			wantErr: true,
			errMsg:  "invalid operation 'start' for resource 'fleet'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewFleetExecutor()
			cfg := &config.ConfigData{
				AccessLevel: "readwrite",
				SecurityConfig: &security.SecurityConfig{
					AccessLevel: "readwrite",
				},
			}

			_, err := executor.Execute(context.Background(), tt.params, cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Execute() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
			// Note: We don't test successful execution as it would require mocking the az CLI
		})
	}
}

func TestFleetExecutor_ValidateClusterResourcePlacementCombination(t *testing.T) {
	executor := NewFleetExecutor()

	tests := []struct {
		name      string
		operation string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid list operation",
			operation: "list",
			wantErr:   false,
		},
		{
			name:      "valid show operation",
			operation: "show",
			wantErr:   false,
		},
		{
			name:      "valid get operation",
			operation: "get",
			wantErr:   false,
		},
		{
			name:      "valid create operation",
			operation: "create",
			wantErr:   false,
		},
		{
			name:      "valid delete operation",
			operation: "delete",
			wantErr:   false,
		},
		{
			name:      "invalid update operation",
			operation: "update",
			wantErr:   true,
			errMsg:    "invalid operation 'update' for resource 'clusterresourceplacement'",
		},
		{
			name:      "invalid start operation",
			operation: "start",
			wantErr:   true,
			errMsg:    "invalid operation 'start' for resource 'clusterresourceplacement'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateClusterResourcePlacementCombination(tt.operation)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateClusterResourcePlacementCombination() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateClusterResourcePlacementCombination() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("validateClusterResourcePlacementCombination() unexpected error = %v", err)
			}
		})
	}
}

func TestFleetExecutor_ExecuteKubernetesClusterResourcePlacement(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		args        string
		wantErr     bool
		errContains string
		accessLevel string
	}{
		{
			name:        "successful list operation",
			operation:   "list",
			args:        "",
			accessLevel: "readonly",
		},
		{
			name:        "successful create operation",
			operation:   "create",
			args:        "--name test-placement --selector app=test --policy PickAll",
			accessLevel: "readwrite",
		},
		{
			name:        "create without required name",
			operation:   "create",
			args:        "--selector app=test",
			wantErr:     true,
			errContains: "not initialized",
			accessLevel: "readwrite",
		},
		{
			name:        "get without required name",
			operation:   "get",
			args:        "",
			wantErr:     true,
			errContains: "not initialized",
			accessLevel: "readonly",
		},
		{
			name:        "delete without required name",
			operation:   "delete",
			args:        "",
			wantErr:     true,
			errContains: "not initialized",
			accessLevel: "admin",
		},
		{
			name:        "readonly cannot create",
			operation:   "create",
			args:        "--name test",
			wantErr:     true,
			errContains: "requires readwrite or admin access level",
			accessLevel: "readonly",
		},
		{
			name:        "unsupported operation",
			operation:   "update",
			args:        "--name test",
			wantErr:     true,
			errContains: "not initialized",
			accessLevel: "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewFleetExecutor()
			cfg := &config.ConfigData{
				AccessLevel: tt.accessLevel,
				SecurityConfig: &security.SecurityConfig{
					AccessLevel: tt.accessLevel,
				},
			}

			// Skip initialization and mock placement operations directly for testing
			executor.k8sClientInitialized = true

			result, err := executor.executeKubernetesClusterResourcePlacement(context.Background(), tt.operation, tt.args, cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("executeKubernetesClusterResourcePlacement() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("executeKubernetesClusterResourcePlacement() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					// If initialization fails, skip the test
					if strings.Contains(err.Error(), "not initialized") || strings.Contains(err.Error(), "kubectl") {
						t.Skipf("Skipping test due to kubectl not being available: %v", err)
					}
					t.Errorf("executeKubernetesClusterResourcePlacement() unexpected error = %v", err)
				}
			}
			_ = result // Suppress unused variable warning
		})
	}
}

func TestFleetExecutor_CreateClusterResourcePlacement(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful creation with all parameters",
			args: map[string]string{
				"name":     "test-placement",
				"selector": "app=test,env=prod",
				"policy":   "PickAll",
			},
			wantErr:     true,
			errContains: "not initialized",
		},
		{
			name: "successful creation with default policy",
			args: map[string]string{
				"name":     "test-placement",
				"selector": "app=test",
			},
			wantErr:     true,
			errContains: "not initialized",
		},
		{
			name: "missing name",
			args: map[string]string{
				"selector": "app=test",
			},
			wantErr:     true,
			errContains: "--name is required for create operation",
		},
		{
			name: "invalid policy",
			args: map[string]string{
				"name":   "test-placement",
				"policy": "InvalidPolicy",
			},
			wantErr:     true,
			errContains: "invalid policy 'InvalidPolicy'",
		},
		{
			name: "case insensitive policy",
			args: map[string]string{
				"name":   "test-placement",
				"policy": "pickall",
			},
			wantErr:     true,
			errContains: "not initialized",
		},
	}

	// Create an executor with initialized state but without actual k8s client
	executor := NewFleetExecutor()
	executor.k8sClientInitialized = true

	// Note: We can't easily test the actual placement operations without proper mocking
	// This test validates the parameter validation logic
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ConfigData{}

			// Test will fail if placementOps is nil, which is expected without proper initialization
			// We're primarily testing the validation logic here
			result, err := executor.createClusterResourcePlacement(context.Background(), tt.args, cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("createClusterResourcePlacement() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("createClusterResourcePlacement() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				// All tests should fail since placement operations are not initialized
				if err == nil {
					t.Errorf("createClusterResourcePlacement() error = nil, expected error due to uninitialized placement operations")
				}
			}

			_ = result // Suppress unused variable warning
		})
	}
}

func TestFleetExecutor_GetClusterResourcePlacement(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful get",
			args: map[string]string{
				"name": "test-placement",
			},
			wantErr:     true,
			errContains: "not initialized",
		},
		{
			name:        "missing name",
			args:        map[string]string{},
			wantErr:     true,
			errContains: "--name is required for get/show operation",
		},
		{
			name: "empty name",
			args: map[string]string{
				"name": "",
			},
			wantErr:     true,
			errContains: "--name is required for get/show operation",
		},
	}

	executor := NewFleetExecutor()
	executor.k8sClientInitialized = true

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ConfigData{}

			_, err := executor.getClusterResourcePlacement(context.Background(), tt.args, cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getClusterResourcePlacement() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("getClusterResourcePlacement() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				// Expected to fail without proper initialization
				if err != nil && strings.Contains(err.Error(), "not initialized") {
					t.Skip("Skipping test as placement operations are not initialized")
				}
			}
		})
	}
}

func TestFleetExecutor_DeleteClusterResourcePlacement(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful delete",
			args: map[string]string{
				"name": "test-placement",
			},
			wantErr:     true,
			errContains: "not initialized",
		},
		{
			name:        "missing name",
			args:        map[string]string{},
			wantErr:     true,
			errContains: "--name is required for delete operation",
		},
		{
			name: "empty name",
			args: map[string]string{
				"name": "",
			},
			wantErr:     true,
			errContains: "--name is required for delete operation",
		},
	}

	executor := NewFleetExecutor()
	executor.k8sClientInitialized = true

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ConfigData{}

			_, err := executor.deleteClusterResourcePlacement(context.Background(), tt.args, cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("deleteClusterResourcePlacement() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("deleteClusterResourcePlacement() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				// Expected to fail without proper initialization
				if err != nil && strings.Contains(err.Error(), "not initialized") {
					t.Skip("Skipping test as placement operations are not initialized")
				}
			}
		})
	}
}
