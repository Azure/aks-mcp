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
	LLMIterations int
}

const systemPrompt = `You are an expert Kubernetes troubleshooting assistant.

Your task is to help users diagnose and solve Kubernetes problems by:
1. Analyzing the user's question carefully
2. Selecting appropriate tools to gather information
3. Calling tools with correct parameters
4. Analyzing tool results
5. Providing clear, accurate answers

Strategic Approach:
- Be aware that many Kubernetes resources are created by controllers with generated names
  * Deployments create pods with names like "deployment-name-xxxxx-xxxxx"
  * If querying by a simple name returns empty results, list resources first to discover actual names
  * Example: If "nginx pod" events return empty, try listing pods first to find "nginx-xxxxx-xxxxx"
  
- Use iterative discovery when needed:
  * If a query returns no results or empty output, investigate why
  * Consider listing resources to discover actual names
  * Use label selectors or get commands to find resources
  
- Gather comprehensive information:
  * Use multiple tool calls to build a complete picture
  * Describe resources to see their status, events, and configuration
  * Check logs when containers are failing
  * Look at events to understand state transitions

When using tools:
- Use call_kubectl to inspect Kubernetes resources
- Always specify the correct namespace when needed
- Use appropriate commands (get, describe, logs, events, etc.)
- Start with listing commands (get) before diving into details (describe)
- When results are empty or unexpected, iterate with different queries

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
				LLMIterations: i + 1,
			}, nil
		}

		// Ensure content is never empty for Azure OpenAI API compatibility
		assistantContent := response.Content
		if assistantContent == "" {
			assistantContent = " "
		}
		a.messages = append(a.messages, Message{
			Role:      "assistant",
			Content:   assistantContent,
			ToolCalls: response.ToolCalls,
		})

		type toolCallResult struct {
			toolCall   ToolCall
			result     *loader.ToolCallRecord
			err        error
		}

		results := make(chan toolCallResult, len(response.ToolCalls))

		for _, tc := range response.ToolCalls {
			go func(tc ToolCall) {
				toolResult, err := a.executeToolCall(ctx, tc)
				results <- toolCallResult{
					toolCall: tc,
					result:   toolResult,
					err:      err,
				}
			}(tc)
		}

		toolResultsMap := make(map[string]toolCallResult)
		for range response.ToolCalls {
			res := <-results
			toolResultsMap[res.toolCall.ID] = res
		}
		close(results)

		for _, tc := range response.ToolCalls {
			res := toolResultsMap[tc.ID]
			
			if res.err != nil {
				a.messages = append(a.messages, Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("Error: %s", res.err.Error()),
				})
				continue
			}

			toolContent := res.result.Result
			if toolContent == "" {
				toolContent = "(no output)"
			}
			a.messages = append(a.messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    toolContent,
			})

			a.toolCalls = append(a.toolCalls, *res.result)
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
