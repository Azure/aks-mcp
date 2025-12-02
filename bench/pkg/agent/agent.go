package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/aks-mcp/bench/pkg/loader"
	"github.com/Azure/aks-mcp/bench/pkg/mcp"
)

type Agent struct {
	llmClient     LLMClient
	mcpClient     *mcp.Client
	maxIterations int
	messages      []Message
	toolCalls     []loader.ToolCallRecord
}

type AgentConfig struct {
	LLMClient     LLMClient
	MCPClient     *mcp.Client
	MaxIterations int
}

type AgentResult struct {
	Answer        string
	ToolCalls     []loader.ToolCallRecord
	TotalDuration time.Duration
}

const systemPrompt = `You are an expert Kubernetes troubleshooting assistant.

Your task is to help users diagnose and solve Kubernetes problems by:
1. Analyzing the user's question carefully
2. Selecting appropriate tools to gather information
3. Calling tools with correct parameters
4. Analyzing tool results
5. Providing clear, accurate answers

When using tools:
- Use call_kubectl to inspect Kubernetes resources
- Always specify the correct namespace when needed
- Use appropriate commands (get, describe, logs, etc.)
- Gather sufficient information before concluding

Provide concise, accurate answers that directly address the user's question.`

func NewAgent(config AgentConfig) *Agent {
	if config.MaxIterations == 0 {
		config.MaxIterations = 10
	}

	return &Agent{
		llmClient:     config.LLMClient,
		mcpClient:     config.MCPClient,
		maxIterations: config.MaxIterations,
		toolCalls:     make([]loader.ToolCallRecord, 0),
	}
}

func (a *Agent) Run(ctx context.Context, prompt string) (*AgentResult, error) {
	startTime := time.Now()

	tools, err := a.mcpClient.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	a.messages = []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	for i := 0; i < a.maxIterations; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		response, err := a.llmClient.ChatCompletion(ctx, a.messages, tools)
		if err != nil {
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		if len(response.ToolCalls) == 0 {
			return &AgentResult{
				Answer:        response.Content,
				ToolCalls:     a.toolCalls,
				TotalDuration: time.Since(startTime),
			}, nil
		}

		a.messages = append(a.messages, Message{
			Role:      "assistant",
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})

		for _, tc := range response.ToolCalls {
			toolResult, err := a.executeToolCall(ctx, tc)
			if err != nil {
				a.messages = append(a.messages, Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("Error: %s", err.Error()),
				})
				continue
			}

			a.messages = append(a.messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    toolResult.Result,
			})

			a.toolCalls = append(a.toolCalls, *toolResult)
		}
	}

	return nil, fmt.Errorf("max iterations (%d) reached without final answer", a.maxIterations)
}

func (a *Agent) executeToolCall(ctx context.Context, tc ToolCall) (*loader.ToolCallRecord, error) {
	startTime := time.Now()

	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	result, err := a.mcpClient.CallTool(ctx, tc.Function.Name, args)
	if err != nil {
		return &loader.ToolCallRecord{
			ToolName:      tc.Function.Name,
			Arguments:     args,
			Error:         err.Error(),
			Timestamp:     startTime,
			ExecutionTime: time.Since(startTime),
		}, err
	}

	resultText := ""
	if len(result.Content) > 0 {
		resultText = result.Content[0].Text
	}

	return &loader.ToolCallRecord{
		ToolName:      tc.Function.Name,
		Arguments:     args,
		Result:        resultText,
		Timestamp:     startTime,
		ExecutionTime: time.Since(startTime),
	}, nil
}

func (a *Agent) Reset() {
	a.messages = nil
	a.toolCalls = make([]loader.ToolCallRecord, 0)
}
