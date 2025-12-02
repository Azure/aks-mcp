package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Azure/aks-mcp/bench/pkg/mcp"
)

type LLMClient interface {
	ChatCompletion(ctx context.Context, messages []Message, tools []mcp.Tool) (*Response, error)
}

type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function FunctionCall           `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Response struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type AzureOpenAIClient struct {
	endpoint   string
	apiKey     string
	deployment string
	apiVersion string
	httpClient *http.Client
}

type AzureConfig struct {
	Endpoint   string
	APIKey     string
	Deployment string
}

func NewAzureOpenAIClient(config AzureConfig) (*AzureOpenAIClient, error) {
	if config.Endpoint == "" {
		config.Endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	}
	if config.APIKey == "" {
		config.APIKey = os.Getenv("AZURE_OPENAI_API_KEY")
	}
	if config.Deployment == "" {
		config.Deployment = os.Getenv("AZURE_OPENAI_DEPLOYMENT")
		if config.Deployment == "" {
			config.Deployment = "gpt-4o"
		}
	}

	if config.Endpoint == "" || config.APIKey == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_API_KEY are required")
	}

	return &AzureOpenAIClient{
		endpoint:   config.Endpoint,
		apiKey:     config.APIKey,
		deployment: config.Deployment,
		apiVersion: "2024-08-01-preview",
		httpClient: &http.Client{},
	}, nil
}

func (c *AzureOpenAIClient) ChatCompletion(ctx context.Context, messages []Message, tools []mcp.Tool) (*Response, error) {
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		c.endpoint, c.deployment, c.apiVersion)

	reqBody := map[string]any{
		"messages": messages,
	}

	if len(tools) > 0 {
		azureTools := make([]map[string]any, len(tools))
		for i, tool := range tools {
			azureTools[i] = map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  convertSchemaToParameters(tool.InputSchema),
				},
			}
		}
		reqBody["tools"] = azureTools
		reqBody["tool_choice"] = "auto"
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Azure OpenAI API error [%d]: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	msg := apiResp.Choices[0].Message
	return &Response{
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
	}, nil
}

func convertSchemaToParameters(schema mcp.ToolInputSchema) map[string]any {
	params := map[string]any{
		"type":       schema.Type,
		"properties": make(map[string]any),
	}

	for name, prop := range schema.Properties {
		params["properties"].(map[string]any)[name] = map[string]any{
			"type":        prop.Type,
			"description": prop.Description,
		}
		if len(prop.Enum) > 0 {
			params["properties"].(map[string]any)[name].(map[string]any)["enum"] = prop.Enum
		}
	}

	if len(schema.Required) > 0 {
		params["required"] = schema.Required
	}

	return params
}
