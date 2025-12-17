package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

func TestClient_ExecuteKubectl(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		mockOutput  string
		mockError   error
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful command execution",
			command:    "get pods",
			mockOutput: "pod/nginx-1234 Running",
			mockError:  nil,
			wantErr:    false,
		},
		{
			name:        "command execution error",
			command:     "get invalidresource",
			mockOutput:  "",
			mockError:   fmt.Errorf("error: the server doesn't have a resource type \"invalidresource\""),
			wantErr:     true,
			errContains: "invalidresource",
		},
		{
			name:       "empty command",
			command:    "",
			mockOutput: "",
			mockError:  nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutor{
				ExecuteFunc: func(ctx context.Context, params map[string]any, cfg *config.ConfigData) (string, error) {
					// Verify the command parameter
					if cmd, ok := params["command"].(string); ok {
						if !strings.Contains(cmd, tt.command) {
							return "", fmt.Errorf("unexpected command: %s", cmd)
						}
					}
					return tt.mockOutput, tt.mockError
				},
			}

			client := &Client{
				executor: mockExecutor,
			}

			cfg := &config.ConfigData{}
			result, err := client.ExecuteKubectl(context.Background(), tt.command, cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExecuteKubectl() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ExecuteKubectl() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ExecuteKubectl() unexpected error = %v", err)
				}
				if result != tt.mockOutput {
					t.Errorf("ExecuteKubectl() = %v, want %v", result, tt.mockOutput)
				}
			}
		})
	}
}

func TestClient_ExecuteKubectlWithNilExecutor(t *testing.T) {
	client := &Client{
		executor: nil,
	}

	cfg := &config.ConfigData{}
	_, err := client.ExecuteKubectl(context.Background(), "get pods", cfg)

	if err == nil {
		t.Error("ExecuteKubectl() with nil executor should return error")
	}
	if !strings.Contains(err.Error(), "executor is nil") {
		t.Errorf("ExecuteKubectl() error = %v, want error containing 'executor is nil'", err)
	}
}

func TestClient_ExecuteKubectlWithNilClient(t *testing.T) {
	var client *Client = nil

	cfg := &config.ConfigData{}
	_, err := client.ExecuteKubectl(context.Background(), "get pods", cfg)

	if err == nil {
		t.Error("ExecuteKubectl() with nil client should return error")
	}
	if !strings.Contains(err.Error(), "Client is nil") {
		t.Errorf("ExecuteKubectl() error = %v, want error containing 'Client is nil'", err)
	}
}
