package prompts_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/components/detectors"
	"github.com/Azure/aks-mcp/internal/components/monitor"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/k8s"
	"github.com/Azure/aks-mcp/internal/prompts"
	azapimcp "github.com/Azure/azure-api-mcp/pkg/azcli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var promptToolCallPattern = regexp.MustCompile(`(?s)Invoke (?:the )?([a-z][a-z0-9_]+)(?: MCP)? tool(?: with inputs)?:\s*(\{.*?\n\s*\})`)

func TestShippedPromptsUseDefaultUnifiedToolSchemas(t *testing.T) {
	cfg := config.NewConfig()
	cfg.UseLegacyTools = false

	promptServer := server.NewMCPServer("prompt-test", "test")
	prompts.RegisterQueryAKSMetadataFromKubeconfigPrompt(promptServer, cfg)
	prompts.RegisterHealthPrompts(promptServer, cfg)

	defaultTools := make(map[string]mcp.Tool)
	for _, tool := range k8s.RegisterKubectlTools(cfg.AccessLevel, true) {
		defaultTools[tool.Name] = tool
	}
	for _, tool := range []mcp.Tool{
		azapimcp.RegisterCallAzTool(true, ""),
		monitor.RegisterAksMonitoring(),
		detectors.RegisterAksDetectorTool(),
	} {
		defaultTools[tool.Name] = tool
	}

	for promptName, registeredPrompt := range promptServer.ListPrompts() {
		t.Run(promptName, func(t *testing.T) {
			result, err := registeredPrompt.Handler(context.Background(), mcp.GetPromptRequest{})
			if err != nil {
				t.Fatalf("render prompt: %v", err)
			}

			var callCount int
			for _, message := range result.Messages {
				content, ok := message.Content.(mcp.TextContent)
				if !ok {
					continue
				}
				matches := promptToolCallPattern.FindAllStringSubmatch(content.Text, -1)
				callCount += len(matches)
				for _, match := range matches {
					validatePromptToolCall(t, defaultTools, match[1], match[2])
				}
			}
			if callCount == 0 {
				t.Fatal("prompt contains no structured tool calls")
			}
		})
	}
}

func validatePromptToolCall(t *testing.T, defaultTools map[string]mcp.Tool, toolName, rawArguments string) {
	t.Helper()

	tool, ok := defaultTools[toolName]
	if !ok {
		t.Fatalf("prompt references %q, which is not registered in the default unified configuration", toolName)
	}

	var arguments map[string]any
	if err := json.Unmarshal([]byte(rawArguments), &arguments); err != nil {
		t.Fatalf("%s arguments are not valid JSON: %v", toolName, err)
	}

	for argument := range arguments {
		if _, ok := tool.InputSchema.Properties[argument]; !ok {
			t.Errorf("%s prompt argument %q is not defined by the registered tool schema", toolName, argument)
		}
	}
	for _, required := range tool.InputSchema.Required {
		if _, ok := arguments[required]; !ok {
			t.Errorf("%s prompt call omits required argument %q", toolName, required)
		}
	}

	operation, ok := arguments["operation"].(string)
	if !ok {
		return
	}
	property, ok := tool.InputSchema.Properties["operation"].(map[string]any)
	if !ok {
		t.Fatalf("%s operation schema has unexpected type %T", toolName, tool.InputSchema.Properties["operation"])
	}
	description := fmt.Sprint(property["description"])
	if !strings.Contains(description, operation) {
		t.Errorf("%s prompt operation %q is not documented by the registered tool schema: %s", toolName, operation, description)
	}
}
