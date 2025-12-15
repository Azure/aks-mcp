package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/telemetry"
	"github.com/mark3labs/mcp-go/mcp"
)

// helper to extract first text content from result
func firstText(result *mcp.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			return tc.Text
		}
	}
	return ""
}

func TestCreateToolHandler_ErrorIncludesResultOutput(t *testing.T) {
	cfg := config.NewConfig()

	// Fake executor returns stderr-like output with an error
	exec := CommandExecutorFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return "ERROR: Azure CLI detailed message", errors.New("exit status 1")
	})

	handler := CreateToolHandler(exec, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dummy_tool",
			Arguments: map[string]any{"operation": "test"},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil {
		t.Fatalf("nil result returned")
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true on result")
	}
	msg := firstText(res)
	if !strings.Contains(msg, "exit status 1") || !strings.Contains(msg, "ERROR: Azure CLI detailed message") {
		t.Fatalf("expected combined error + output, got: %q", msg)
	}
}

func TestCreateToolHandler_ErrorWithoutOutput(t *testing.T) {
	cfg := config.NewConfig()

	exec := CommandExecutorFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return "", errors.New("exit status 1")
	})

	handler := CreateToolHandler(exec, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dummy_tool",
			Arguments: map[string]any{"operation": "test"},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected error result, got: %+v", res)
	}
	msg := firstText(res)
	if msg != "exit status 1" {
		t.Fatalf("expected only error text, got: %q", msg)
	}
}

func TestCreateResourceHandler_ErrorIncludesResultOutput(t *testing.T) {
	cfg := config.NewConfig()

	rh := ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return "API: detailed failure context", errors.New("bad request")
	})

	handler := CreateResourceHandler(rh, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dummy_resource",
			Arguments: map[string]any{"operation": "test"},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected error result, got: %+v", res)
	}
	msg := firstText(res)
	if !strings.Contains(msg, "bad request") || !strings.Contains(msg, "API: detailed failure context") {
		t.Fatalf("expected combined error + output, got: %q", msg)
	}
}

func TestCreateResourceHandler_ErrorWithoutOutput(t *testing.T) {
	cfg := config.NewConfig()

	rh := ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return "", errors.New("bad request")
	})

	handler := CreateResourceHandler(rh, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dummy_resource",
			Arguments: map[string]any{"operation": "test"},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected error result, got: %+v", res)
	}
	msg := firstText(res)
	if msg != "bad request" {
		t.Fatalf("expected only error text, got: %q", msg)
	}
}

func TestCreateToolHandler_Success_Verbose_Telemetry_LongResult(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LogLevel = "debug" // exercise logToolCall + logToolResult
	// Provide non-nil telemetry to exercise TrackToolInvocation path
	cfg.TelemetryService = telemetry.NewService(telemetry.NewConfig("svc", "1.0"))

	long := strings.Repeat("x", 600)
	exec := CommandExecutorFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return long, nil
	})

	handler := CreateToolHandler(exec, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "dummy_tool",
			Arguments: map[string]any{
				"operation": "op",
			},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil || res.IsError {
		t.Fatalf("expected success result, got: %+v", res)
	}
	if got := firstText(res); got != long {
		t.Fatalf("unexpected result text length=%d", len(got))
	}
}

func TestCreateToolHandler_InvalidArguments_Verbose_LogsFallback_TracksTelemetry(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LogLevel = "debug"
	cfg.TelemetryService = telemetry.NewService(telemetry.NewConfig("svc", "1.0"))

	exec := CommandExecutorFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return "should not run", nil
	})

	handler := CreateToolHandler(exec, cfg)

	// Use an argument type that fails json.Marshal to exercise logToolCall fallback branch
	ch := make(chan int)
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dummy_tool",
			Arguments: ch,
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected error result, got: %+v", res)
	}
	msg := firstText(res)
	if !strings.Contains(msg, "arguments must be a map[string]interface{}") {
		t.Fatalf("unexpected error message: %q", msg)
	}
}

func TestCreateResourceHandler_ShortSuccess_Verbose_Telemetry(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LogLevel = "debug"
	cfg.TelemetryService = telemetry.NewService(telemetry.NewConfig("svc", "1.0"))

	rh := ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return "ok", nil
	})

	handler := CreateResourceHandler(rh, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dummy_resource",
			Arguments: map[string]any{"operation": "x"},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil || res.IsError {
		t.Fatalf("expected success result, got: %+v", res)
	}
	if got := firstText(res); got != "ok" {
		t.Fatalf("unexpected text: %q", got)
	}
}

func TestCreateResourceHandler_InvalidArguments_Verbose_LogsFallback_TracksTelemetry(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LogLevel = "debug"
	cfg.TelemetryService = telemetry.NewService(telemetry.NewConfig("svc", "1.0"))

	rh := ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return "should not run", nil
	})

	handler := CreateResourceHandler(rh, cfg)

	// Unmarshalable type to drive logToolCall fallback branch
	ch := make(chan int)
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dummy_resource",
			Arguments: ch,
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected error result, got: %+v", res)
	}
	if msg := firstText(res); !strings.Contains(msg, "arguments must be a map[string]interface{}") {
		t.Fatalf("unexpected error message: %q", msg)
	}
}

func TestCreateToolHandler_Error_Verbose_LogErrorBranch(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LogLevel = "debug"

	exec := CommandExecutorFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		return "", errors.New("boom")
	})

	handler := CreateToolHandler(exec, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dummy_tool",
			Arguments: map[string]any{"operation": "op"},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected error result, got: %+v", res)
	}
}

func TestGetOperationValuePrefersOperation(t *testing.T) {
	args := map[string]any{
		"operation": "metrics",
		"action":    "run",
	}

	if got := getOperationValue(args); got != "metrics" {
		t.Fatalf("expected operation to win, got %q", got)
	}
}

func TestGetOperationValueFallsBackToAction(t *testing.T) {
	args := map[string]any{
		"action": "deploy",
	}

	if got := getOperationValue(args); got != "deploy" {
		t.Fatalf("expected action fallback, got %q", got)
	}
}

func TestGetOperationValueHandlesMissingKeys(t *testing.T) {
	if got := getOperationValue(map[string]any{}); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
