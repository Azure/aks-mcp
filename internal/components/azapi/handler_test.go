package azapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/azure-api-mcp/pkg/azcli"
	"github.com/mark3labs/mcp-go/mcp"
)

type mockAzClient struct {
	executeFunc func(ctx context.Context, command string) (*azcli.Result, error)
}

func (m *mockAzClient) ExecuteCommand(ctx context.Context, command string) (*azcli.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, command)
	}
	return &azcli.Result{
		Output: json.RawMessage("mock output"),
		Error:  "",
	}, nil
}

func (m *mockAzClient) ValidateCommand(cmdStr string) error {
	return nil
}

func TestAzApiHandler_Success(t *testing.T) {
	mockClient := &mockAzClient{
		executeFunc: func(ctx context.Context, command string) (*azcli.Result, error) {
			if command != "az group list" {
				t.Errorf("expected command 'az group list', got '%s'", command)
			}
			return &azcli.Result{
				Output: json.RawMessage(`[{"name":"rg1","location":"eastus"}]`),
				Error:  "",
			}, nil
		},
	}

	cfg := &config.ConfigData{
		Timeout: 30,
	}

	handler := AzApiHandler(mockClient, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "call_az",
			Arguments: map[string]interface{}{
				"cli_command": "az group list",
			},
		},
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	expected := `[{"name":"rg1","location":"eastus"}]`
	if textContent.Text != expected {
		t.Errorf("expected output '%s', got '%s'", expected, textContent.Text)
	}
}

func TestAzApiHandler_InvalidArguments(t *testing.T) {
	mockClient := &mockAzClient{}
	cfg := &config.ConfigData{
		Timeout: 30,
	}

	handler := AzApiHandler(mockClient, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "call_az",
			Arguments: "invalid",
		},
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestAzApiHandler_MissingCliCommand(t *testing.T) {
	mockClient := &mockAzClient{}
	cfg := &config.ConfigData{
		Timeout: 30,
	}

	handler := AzApiHandler(mockClient, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "call_az",
			Arguments: map[string]interface{}{
				"other_param": "value",
			},
		},
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	if textContent.Text != "missing or invalid 'cli_command' parameter" {
		t.Errorf("unexpected error message: %s", textContent.Text)
	}
}

func TestAzApiHandler_ExecutionError(t *testing.T) {
	mockClient := &mockAzClient{
		executeFunc: func(ctx context.Context, command string) (*azcli.Result, error) {
			return nil, errors.New("execution failed")
		},
	}

	cfg := &config.ConfigData{
		Timeout: 30,
	}

	handler := AzApiHandler(mockClient, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "call_az",
			Arguments: map[string]interface{}{
				"cli_command": "az group list",
			},
		},
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestAzApiHandler_CommandError(t *testing.T) {
	mockClient := &mockAzClient{
		executeFunc: func(ctx context.Context, command string) (*azcli.Result, error) {
			return &azcli.Result{
				Output: json.RawMessage(""),
				Error:  "command error: resource not found",
			}, nil
		},
	}

	cfg := &config.ConfigData{
		Timeout: 30,
	}

	handler := AzApiHandler(mockClient, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "call_az",
			Arguments: map[string]interface{}{
				"cli_command": "az group show --name nonexistent",
			},
		},
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestAzApiHandler_CustomTimeout(t *testing.T) {
	executionStarted := make(chan bool, 1)
	mockClient := &mockAzClient{
		executeFunc: func(ctx context.Context, command string) (*azcli.Result, error) {
			executionStarted <- true
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Error("expected context to have deadline")
			} else {
				timeUntilDeadline := time.Until(deadline)
				if timeUntilDeadline > 61*time.Second || timeUntilDeadline < 59*time.Second {
					t.Errorf("expected timeout around 60s, got %v", timeUntilDeadline)
				}
			}
			return &azcli.Result{
				Output: json.RawMessage("success"),
				Error:  "",
			}, nil
		},
	}

	cfg := &config.ConfigData{
		Timeout: 30,
	}

	handler := AzApiHandler(mockClient, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "call_az",
			Arguments: map[string]interface{}{
				"cli_command": "az group list",
				"timeout":     float64(60),
			},
		},
	}

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case <-executionStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("execution did not start")
	}
}
