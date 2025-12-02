package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/bench/pkg/loader"
)

type MarkdownReporter struct{}

func NewMarkdownReporter() *MarkdownReporter {
	return &MarkdownReporter{}
}

func (r *MarkdownReporter) Generate(summary *loader.BenchmarkSummary, outputPath string) error {
	content := r.buildReport(summary)

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write markdown report: %w", err)
	}

	return nil
}

func (r *MarkdownReporter) buildReport(summary *loader.BenchmarkSummary) string {
	var sb strings.Builder

	sb.WriteString("# AKS-MCP Benchmark Results\n\n")

	sb.WriteString(fmt.Sprintf("**Date:** %s\n", summary.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Model:** %s\n", summary.Model))
	sb.WriteString(fmt.Sprintf("**MCP Binary:** %s\n", summary.MCPBinary))
	sb.WriteString(fmt.Sprintf("**Total Tests:** %d\n", summary.TotalTests))
	sb.WriteString(fmt.Sprintf("**Success Rate:** %.1f%% (%d/%d)\n\n",
		summary.SuccessRate*100, summary.PassedTests, summary.TotalTests))

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Tests | %d |\n", summary.TotalTests))
	sb.WriteString(fmt.Sprintf("| Passed | %d |\n", summary.PassedTests))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", summary.FailedTests))
	sb.WriteString(fmt.Sprintf("| Errors | %d |\n", summary.ErrorTests))
	sb.WriteString(fmt.Sprintf("| Skipped | %d |\n", summary.SkippedTests))
	sb.WriteString(fmt.Sprintf("| Success Rate | %.1f%% |\n", summary.SuccessRate*100))

	if summary.AvgToolSelectionScore > 0 {
		sb.WriteString(fmt.Sprintf("| Avg Tool Selection | %.1f%% |\n", summary.AvgToolSelectionScore*100))
	}
	if summary.AvgParameterAccuracy > 0 {
		sb.WriteString(fmt.Sprintf("| Avg Parameter Accuracy | %.1f%% |\n", summary.AvgParameterAccuracy*100))
	}
	if summary.AvgOutputQualityScore > 0 {
		sb.WriteString(fmt.Sprintf("| Avg Output Quality | %.2f/1.0 |\n", summary.AvgOutputQualityScore))
	}
	sb.WriteString(fmt.Sprintf("| Total Execution Time | %s |\n", summary.TotalExecutionTime))
	sb.WriteString(fmt.Sprintf("| Avg Test Duration | %s |\n", summary.AvgTestDuration))
	sb.WriteString(fmt.Sprintf("| Avg Tool Call Duration | %s |\n", summary.AvgToolCallDuration))
	sb.WriteString(fmt.Sprintf("| Avg Tool Calls Per Test | %.1f |\n", summary.AvgToolCallsPerTest))
	sb.WriteString(fmt.Sprintf("| Avg LLM Iterations | %.1f |\n\n", summary.AvgLLMIterations))

	sb.WriteString("## Test Results\n\n")
	sb.WriteString("| Test ID | Title | Status | Tool Calls | LLM Iters | Tool Time | Total Time |\n")
	sb.WriteString("|---------|-------|--------|------------|-----------|-----------|------------|\n")

	for _, result := range summary.TestResults {
		statusEmoji := r.getStatusEmoji(result.Status)
		toolCallsStr := fmt.Sprintf("%d", len(result.ToolCalls))
		llmItersStr := fmt.Sprintf("%d", result.LLMIterations)
		toolTimeStr := fmt.Sprintf("%.2fs", result.TotalToolCallTime.Seconds())
		totalTimeStr := fmt.Sprintf("%.1fs", result.ExecutionTime.Seconds())

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
			result.TestCase.ID,
			result.TestCase.Title,
			statusEmoji,
			toolCallsStr,
			llmItersStr,
			toolTimeStr,
			totalTimeStr,
		))
	}

	failedTests := r.getFailedTests(summary.TestResults)
	if len(failedTests) > 0 {
		sb.WriteString("\n## Failed Tests Details\n\n")
		for _, result := range failedTests {
			sb.WriteString(fmt.Sprintf("### %s: %s\n\n", result.TestCase.ID, result.TestCase.Title))

			if result.ErrorMessage != "" {
				sb.WriteString(fmt.Sprintf("**Error:** %s\n\n", result.ErrorMessage))
			}

			if len(result.ValidationResults) > 0 {
				sb.WriteString("**Tool Validation Issues:**\n\n")
				for _, vr := range result.ValidationResults {
					if !vr.ToolCalled {
						sb.WriteString(fmt.Sprintf("- ❌ Tool `%s` was not called\n", vr.ExpectedTool))
					} else if vr.CallCount < vr.MinCalls {
						sb.WriteString(fmt.Sprintf("- ⚠️  Tool `%s` called %d times (expected min %d)\n",
							vr.ExpectedTool, vr.CallCount, vr.MinCalls))
					} else if !vr.ArgsValid {
						sb.WriteString(fmt.Sprintf("- ⚠️  Tool `%s` called with incorrect arguments\n", vr.ExpectedTool))
						if len(vr.MissingPatterns) > 0 {
							sb.WriteString("  - Missing patterns:\n")
							for _, pattern := range vr.MissingPatterns {
								sb.WriteString(fmt.Sprintf("    - `%s`\n", pattern))
							}
						}
					}
				}
				sb.WriteString("\n")
			}

			if len(result.ToolCalls) > 0 {
				sb.WriteString("**Actual Tool Calls:**\n\n")
				for i, tc := range result.ToolCalls {
					sb.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, tc.ToolName))
					if len(tc.Arguments) > 0 {
						sb.WriteString("   - Arguments: ")
						argStrs := make([]string, 0, len(tc.Arguments))
						for k, v := range tc.Arguments {
							argStrs = append(argStrs, fmt.Sprintf("%s=%v", k, v))
						}
						sb.WriteString(strings.Join(argStrs, ", "))
						sb.WriteString("\n")
					}
				}
				sb.WriteString("\n")
			}

			if result.JudgingResult != nil {
				sb.WriteString(fmt.Sprintf("**Output Quality Score:** %.2f/1.0\n\n", result.JudgingResult.Score))
				sb.WriteString(fmt.Sprintf("**Reasoning:** %s\n\n", result.JudgingResult.Reasoning))
			}
		}
	}

	return sb.String()
}

func (r *MarkdownReporter) getStatusEmoji(status loader.TestStatus) string {
	switch status {
	case loader.TestStatusPass:
		return "✅ PASS"
	case loader.TestStatusFail:
		return "❌ FAIL"
	case loader.TestStatusError:
		return "💥 ERROR"
	case loader.TestStatusSkipped:
		return "⏭️  SKIP"
	default:
		return string(status)
	}
}

func (r *MarkdownReporter) formatScore(score float64) string {
	if score == 0 {
		return "-"
	}
	if score >= 0.9 {
		return fmt.Sprintf("✅ %.0f%%", score*100)
	} else if score >= 0.7 {
		return fmt.Sprintf("⚠️  %.0f%%", score*100)
	}
	return fmt.Sprintf("❌ %.0f%%", score*100)
}

func (r *MarkdownReporter) formatOutputScore(score float64) string {
	if score == 0 {
		return "-"
	}
	if score >= 0.9 {
		return fmt.Sprintf("✅ %.2f", score)
	} else if score >= 0.7 {
		return fmt.Sprintf("⚠️  %.2f", score)
	}
	return fmt.Sprintf("❌ %.2f", score)
}

func (r *MarkdownReporter) getFailedTests(results []*loader.TestResult) []*loader.TestResult {
	failed := make([]*loader.TestResult, 0)
	for _, result := range results {
		if result.Status == loader.TestStatusFail || result.Status == loader.TestStatusError {
			failed = append(failed, result)
		}
	}
	return failed
}
