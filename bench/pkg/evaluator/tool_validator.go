package evaluator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/aks-mcp/bench/pkg/loader"
)

type ToolValidator struct{}

func NewToolValidator() *ToolValidator {
	return &ToolValidator{}
}

func (v *ToolValidator) ValidateToolUsage(
	expected []loader.ExpectedToolUsage,
	actual []loader.ToolCallRecord,
) []*loader.ToolValidationResult {
	results := make([]*loader.ToolValidationResult, 0, len(expected))

	for _, exp := range expected {
		result := v.validateSingleTool(exp, actual)
		results = append(results, result)
	}

	return results
}

func (v *ToolValidator) validateSingleTool(
	expected loader.ExpectedToolUsage,
	actual []loader.ToolCallRecord,
) *loader.ToolValidationResult {
	result := &loader.ToolValidationResult{
		ExpectedTool:    expected.ToolName,
		ToolCalled:      false,
		CallCount:       0,
		MinCalls:        expected.MinCalls,
		ArgsValid:       false,
		MatchedPatterns: make([]string, 0),
		MissingPatterns: make([]string, 0),
		Score:           0.0,
	}

	toolCalls := v.filterToolCalls(actual, expected.ToolName)
	result.CallCount = len(toolCalls)

	if result.CallCount == 0 {
		result.MissingPatterns = expected.RequiredArgsPatterns
		return result
	}

	result.ToolCalled = true

	if result.CallCount < expected.MinCalls {
		result.Score = 0.3
		result.MissingPatterns = expected.RequiredArgsPatterns
		return result
	}

	if len(expected.RequiredArgsPatterns) == 0 {
		result.ArgsValid = true
		result.Score = 1.0
		return result
	}

	matchedPatterns := make(map[string]bool)
	for _, call := range toolCalls {
		argsStr := v.serializeArguments(call.Arguments)

		for _, pattern := range expected.RequiredArgsPatterns {
			if matchedPatterns[pattern] {
				continue
			}

			matched, err := regexp.MatchString(pattern, argsStr)
			if err != nil {
				fmt.Printf("Warning: invalid pattern %s: %v\n", pattern, err)
				continue
			}

			if matched {
				matchedPatterns[pattern] = true
				result.MatchedPatterns = append(result.MatchedPatterns, pattern)
			}
		}
	}

	for _, pattern := range expected.RequiredArgsPatterns {
		if !matchedPatterns[pattern] {
			result.MissingPatterns = append(result.MissingPatterns, pattern)
		}
	}

	if len(result.MissingPatterns) == 0 {
		result.ArgsValid = true
		result.Score = 1.0
	} else {
		matchRatio := float64(len(result.MatchedPatterns)) / float64(len(expected.RequiredArgsPatterns))
		result.Score = 0.5 + (matchRatio * 0.5)
	}

	return result
}

func (v *ToolValidator) filterToolCalls(calls []loader.ToolCallRecord, toolName string) []loader.ToolCallRecord {
	filtered := make([]loader.ToolCallRecord, 0)
	for _, call := range calls {
		if call.ToolName == toolName {
			filtered = append(filtered, call)
		}
	}
	return filtered
}

func (v *ToolValidator) serializeArguments(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}

	parts := make([]string, 0, len(args))
	for key, value := range args {
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case []interface{}:
			jsonBytes, _ := json.Marshal(v)
			valueStr = string(jsonBytes)
		case map[string]interface{}:
			jsonBytes, _ := json.Marshal(v)
			valueStr = string(jsonBytes)
		default:
			valueStr = fmt.Sprintf("%v", v)
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, valueStr))
	}

	return strings.Join(parts, " ")
}

func (v *ToolValidator) CalculateOverallScore(results []*loader.ToolValidationResult) (float64, float64) {
	if len(results) == 0 {
		return 0.0, 0.0
	}

	var toolSelectionScore float64
	var parameterScore float64

	for _, result := range results {
		if result.ToolCalled {
			toolSelectionScore += 1.0
		}

		parameterScore += result.Score
	}

	toolSelectionScore /= float64(len(results))
	parameterScore /= float64(len(results))

	return toolSelectionScore, parameterScore
}
