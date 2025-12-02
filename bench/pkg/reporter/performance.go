package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/bench/pkg/loader"
)

type PerformanceReporter struct{}

func NewPerformanceReporter() *PerformanceReporter {
	return &PerformanceReporter{}
}

func (r *PerformanceReporter) Generate(summary *loader.BenchmarkSummary, outputPath string) error {
	content := r.buildReport(summary)

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write performance report: %w", err)
	}

	return nil
}

type toolCallMetric struct {
	testID        string
	toolName      string
	executionTime time.Duration
	arguments     string
}

func (r *PerformanceReporter) buildReport(summary *loader.BenchmarkSummary) string {
	var sb strings.Builder

	sb.WriteString("# AKS-MCP Performance Analysis Report\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", summary.Timestamp.Format(time.RFC3339)))

	sb.WriteString("## Overall Performance\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Tests | %d |\n", summary.TotalTests))
	sb.WriteString(fmt.Sprintf("| Total Execution Time | %s |\n", summary.TotalExecutionTime))
	sb.WriteString(fmt.Sprintf("| Avg Test Duration | %s |\n", summary.AvgTestDuration))
	sb.WriteString(fmt.Sprintf("| Avg Tool Call Duration | %s |\n", summary.AvgToolCallDuration))
	sb.WriteString(fmt.Sprintf("| Avg Tool Calls Per Test | %.1f |\n", summary.AvgToolCallsPerTest))
	sb.WriteString(fmt.Sprintf("| Avg LLM Iterations | %.1f |\n\n", summary.AvgLLMIterations))

	toolCalls := r.collectToolCalls(summary)
	
	sb.WriteString("## Slowest Tool Calls (Top 10)\n\n")
	sb.WriteString("| Test ID | Tool Name | Duration | Arguments |\n")
	sb.WriteString("|---------|-----------|----------|----------|\n")
	
	sort.Slice(toolCalls, func(i, j int) bool {
		return toolCalls[i].executionTime > toolCalls[j].executionTime
	})

	topN := 10
	if len(toolCalls) < topN {
		topN = len(toolCalls)
	}

	for i := 0; i < topN; i++ {
		tc := toolCalls[i]
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			tc.testID,
			tc.toolName,
			tc.executionTime,
			r.truncateString(tc.arguments, 50),
		))
	}

	toolStats := r.calculateToolStats(toolCalls)
	
	sb.WriteString("\n## Tool Performance Statistics\n\n")
	sb.WriteString("| Tool Name | Call Count | Avg Duration | Max Duration | Total Time |\n")
	sb.WriteString("|-----------|------------|--------------|--------------|------------|\n")

	var toolNames []string
	for name := range toolStats {
		toolNames = append(toolNames, name)
	}
	sort.Slice(toolNames, func(i, j int) bool {
		return toolStats[toolNames[i]].totalTime > toolStats[toolNames[j]].totalTime
	})

	for _, name := range toolNames {
		stats := toolStats[name]
		sb.WriteString(fmt.Sprintf("| %s | %d | %s | %s | %s |\n",
			name,
			stats.count,
			stats.avgDuration,
			stats.maxDuration,
			stats.totalTime,
		))
	}

	sb.WriteString("\n## Performance Recommendations\n\n")
	recommendations := r.generateRecommendations(summary, toolStats)
	for i, rec := range recommendations {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
	}

	return sb.String()
}

type toolStats struct {
	count       int
	totalTime   time.Duration
	avgDuration time.Duration
	maxDuration time.Duration
}

func (r *PerformanceReporter) collectToolCalls(summary *loader.BenchmarkSummary) []toolCallMetric {
	var metrics []toolCallMetric

	for _, result := range summary.TestResults {
		for _, tc := range result.ToolCalls {
			argsStr := r.formatArguments(tc.Arguments)
			metrics = append(metrics, toolCallMetric{
				testID:        result.TestCase.ID,
				toolName:      tc.ToolName,
				executionTime: tc.ExecutionTime,
				arguments:     argsStr,
			})
		}
	}

	return metrics
}

func (r *PerformanceReporter) calculateToolStats(toolCalls []toolCallMetric) map[string]*toolStats {
	stats := make(map[string]*toolStats)

	for _, tc := range toolCalls {
		if _, exists := stats[tc.toolName]; !exists {
			stats[tc.toolName] = &toolStats{}
		}

		s := stats[tc.toolName]
		s.count++
		s.totalTime += tc.executionTime
		if tc.executionTime > s.maxDuration {
			s.maxDuration = tc.executionTime
		}
	}

	for _, s := range stats {
		if s.count > 0 {
			s.avgDuration = s.totalTime / time.Duration(s.count)
		}
	}

	return stats
}

func (r *PerformanceReporter) generateRecommendations(summary *loader.BenchmarkSummary, toolStats map[string]*toolStats) []string {
	var recommendations []string

	if summary.AvgToolCallDuration > 2*time.Second {
		recommendations = append(recommendations,
			fmt.Sprintf("Average tool call duration (%.2fs) is high. Consider optimizing tool implementations or using caching.",
				summary.AvgToolCallDuration.Seconds()))
	}

	for toolName, stats := range toolStats {
		if stats.avgDuration > 3*time.Second {
			recommendations = append(recommendations,
				fmt.Sprintf("Tool '%s' has high average duration (%.2fs). Consider optimizing this tool or its queries.",
					toolName, stats.avgDuration.Seconds()))
		}
	}

	if summary.AvgLLMIterations > 5 {
		recommendations = append(recommendations,
			fmt.Sprintf("Average LLM iterations (%.1f) is high. Consider improving system prompt or tool descriptions to reduce back-and-forth.",
				summary.AvgLLMIterations))
	}

	if summary.AvgToolCallsPerTest < 1.5 {
		recommendations = append(recommendations,
			fmt.Sprintf("Average tool calls per test (%.1f) is low. Tests might not be exercising enough functionality.",
				summary.AvgToolCallsPerTest))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No significant performance issues detected. All metrics are within acceptable ranges.")
	}

	return recommendations
}

func (r *PerformanceReporter) formatArguments(args map[string]interface{}) string {
	if len(args) == 0 {
		return "(no args)"
	}

	var parts []string
	for k, v := range args {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ", ")
}

func (r *PerformanceReporter) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
