package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

func TestNewPlacementOperations(t *testing.T) {
	t.Run("panics with nil client", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewPlacementOperations() did not panic with nil client")
			}
		}()
		_ = NewPlacementOperations(nil)
	})

	t.Run("creates successfully with valid client", func(t *testing.T) {
		mockClient := &Client{}
		ops := NewPlacementOperations(mockClient)
		if ops == nil {
			t.Error("NewPlacementOperations() returned nil")
			return
		}
		if ops.client != mockClient {
			t.Error("NewPlacementOperations() did not set client correctly")
		}
	})
}

func TestPlacementOperations_ListPlacements(t *testing.T) {
	tests := []struct {
		name        string
		mockOutput  string
		mockError   error
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful list",
			mockOutput: `{"items": [{"metadata": {"name": "test-placement"}}]}`,
			mockError:  nil,
			wantErr:    false,
		},
		{
			name:       "empty list",
			mockOutput: `{"items": []}`,
			mockError:  nil,
			wantErr:    false,
		},
		{
			name:        "kubectl error",
			mockOutput:  "",
			mockError:   fmt.Errorf("No resources found"),
			wantErr:     true,
			errContains: "No resources found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &Client{
				executor: &MockExecutor{
					ExecuteFunc: func(ctx context.Context, params map[string]any, cfg *config.ConfigData) (string, error) {
						if cmd, ok := params["command"].(string); ok && strings.Contains(cmd, "get clusterresourceplacement -o json") {
							return tt.mockOutput, tt.mockError
						}
						return "", fmt.Errorf("unexpected command: %v", params["command"])
					},
				},
			}
			ops := NewPlacementOperations(mockClient)

			cfg := &config.ConfigData{}
			result, err := ops.ListPlacements(context.Background(), cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListPlacements() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ListPlacements() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ListPlacements() unexpected error = %v", err)
				}
				if result != tt.mockOutput {
					t.Errorf("ListPlacements() = %v, want %v", result, tt.mockOutput)
				}
			}
		})
	}
}

func TestPlacementOperations_GetPlacement(t *testing.T) {
	placementName := "test-placement"
	mockOutput := `{"metadata": {"name": "test-placement"}}`

	mockClient := &Client{
		executor: &MockExecutor{
			ExecuteFunc: func(ctx context.Context, params map[string]any, cfg *config.ConfigData) (string, error) {
				expectedCmd := fmt.Sprintf("get clusterresourceplacement %s -o json", placementName)
				if cmd, ok := params["command"].(string); ok && strings.Contains(cmd, expectedCmd) {
					return mockOutput, nil
				}
				return "", fmt.Errorf("unexpected command: %v", params["command"])
			},
		},
	}
	ops := NewPlacementOperations(mockClient)

	cfg := &config.ConfigData{}
	result, err := ops.GetPlacement(context.Background(), placementName, cfg)

	if err != nil {
		t.Errorf("GetPlacement() unexpected error = %v", err)
	}
	if result != mockOutput {
		t.Errorf("GetPlacement() = %v, want %v", result, mockOutput)
	}
}

func TestPlacementOperations_CreatePlacement(t *testing.T) {
	placementName := "test-placement"
	selector := "app=nginx"
	policy := "PickAll"
	mockOutput := "clusterresourceplacement.placement.kubernetes-fleet.io/test-placement created"

	mockClient := &Client{
		executor: &MockExecutor{
			ExecuteFunc: func(ctx context.Context, params map[string]any, cfg *config.ConfigData) (string, error) {
				if cmd, ok := params["command"].(string); ok && strings.Contains(cmd, "apply -f") {
					return mockOutput, nil
				}
				return "", fmt.Errorf("unexpected command: %v", params["command"])
			},
		},
	}
	ops := NewPlacementOperations(mockClient)

	cfg := &config.ConfigData{}
	result, err := ops.CreatePlacement(context.Background(), placementName, selector, policy, cfg)

	if err != nil {
		t.Errorf("CreatePlacement() unexpected error = %v", err)
	}
	if result == "" {
		t.Error("CreatePlacement() returned empty result")
	}
}

func TestPlacementOperations_DeletePlacement(t *testing.T) {
	placementName := "test-placement"
	mockOutput := "clusterresourceplacement.placement.kubernetes-fleet.io \"test-placement\" deleted"

	mockClient := &Client{
		executor: &MockExecutor{
			ExecuteFunc: func(ctx context.Context, params map[string]any, cfg *config.ConfigData) (string, error) {
				expectedCmd := fmt.Sprintf("delete clusterresourceplacement %s", placementName)
				if cmd, ok := params["command"].(string); ok && strings.Contains(cmd, expectedCmd) {
					return mockOutput, nil
				}
				return "", fmt.Errorf("unexpected command: %v", params["command"])
			},
		},
	}
	ops := NewPlacementOperations(mockClient)

	cfg := &config.ConfigData{}
	result, err := ops.DeletePlacement(context.Background(), placementName, cfg)

	if err != nil {
		t.Errorf("DeletePlacement() unexpected error = %v", err)
	}
	if result == "" {
		t.Error("DeletePlacement() returned empty result")
	}
}

func TestParsePlacementArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		expected map[string]string
	}{
		{
			name: "parse multiple arguments",
			args: "--name test-placement --selector app=nginx,env=prod --policy PickAll",
			expected: map[string]string{
				"name":     "test-placement",
				"selector": "app=nginx,env=prod",
				"policy":   "PickAll",
			},
		},
		{
			name:     "parse empty arguments",
			args:     "",
			expected: map[string]string{},
		},
		{
			name: "parse single argument",
			args: "--name test-placement",
			expected: map[string]string{
				"name": "test-placement",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePlacementArgs(tt.args)
			if err != nil {
				t.Errorf("ParsePlacementArgs() unexpected error = %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ParsePlacementArgs() returned %d args, want %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("ParsePlacementArgs() missing key %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("ParsePlacementArgs() for key %s = %v, want %v", key, actualValue, expectedValue)
				}
			}
		})
	}
}
