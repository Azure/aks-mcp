package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/bench/pkg/agent"
	"github.com/Azure/aks-mcp/bench/pkg/loader"
)

type LLMJudge struct {
	llmClient agent.LLMClient
}

func NewLLMJudge(llmClient agent.LLMClient) *LLMJudge {
	return &LLMJudge{
		llmClient: llmClient,
	}
}

const judgingPromptTemplate = `You are evaluating an AI assistant's answer to a Kubernetes troubleshooting question.

User Question: %s

Expected Answer Should Contain:
%s

Actual Answer:
%s

Rate the answer on a scale of 0-1:
- 1.0: Perfect answer, includes all expected points
- 0.7-0.9: Good answer, covers main points with minor gaps
- 0.4-0.6: Partial answer, missing important information
- 0.0-0.3: Poor answer, misses most key points or incorrect

Return ONLY a JSON object with this exact format:
{
  "score": <float between 0 and 1>,
  "reasoning": "<brief explanation>"
}`

func (j *LLMJudge) JudgeAnswer(
	ctx context.Context,
	testCase *loader.TestCase,
	actualAnswer string,
) (*loader.JudgingResult, error) {
	expectedPoints := strings.Join(testCase.ExpectedOutput, "\n- ")
	expectedPoints = "- " + expectedPoints

	prompt := fmt.Sprintf(
		judgingPromptTemplate,
		testCase.UserPrompt,
		expectedPoints,
		actualAnswer,
	)

	messages := []agent.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := j.llmClient.ChatCompletion(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM judge call failed: %w", err)
	}

	result, err := j.parseJudgingResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse judging response: %w", err)
	}

	return result, nil
}

func (j *LLMJudge) parseJudgingResponse(content string) (*loader.JudgingResult, error) {
	content = strings.TrimSpace(content)

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	jsonStr := content[start : end+1]

	var result loader.JudgingResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if result.Score < 0 || result.Score > 1 {
		return nil, fmt.Errorf("score out of range: %f", result.Score)
	}

	return &result, nil
}
