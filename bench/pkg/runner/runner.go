package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/bench/pkg/agent"
	"github.com/Azure/aks-mcp/bench/pkg/loader"
	"github.com/Azure/aks-mcp/bench/pkg/mcp"
)

type Runner struct {
	mcpBinary string
	mcpArgs   []string
}

type RunnerConfig struct {
	MCPBinary   string
	MCPArgs     []string
}

func NewRunner(config RunnerConfig) *Runner {
	if config.MCPBinary == "" {
		config.MCPBinary = "../aks-mcp"
	}
	
	return &Runner{
		mcpBinary: config.MCPBinary,
		mcpArgs:   config.MCPArgs,
	}
}

func (r *Runner) RunTest(ctx context.Context, testCase *loader.TestCase, agentConfig agent.AgentConfig) (*loader.TestResult, error) {
	result := &loader.TestResult{
		TestCase:  testCase,
		StartTime: time.Now(),
		Status:    loader.TestStatusError,
	}

	if err := r.runBeforeTest(ctx, testCase); err != nil {
		result.ErrorMessage = fmt.Sprintf("before_test failed: %v", err)
		result.EndTime = time.Now()
		result.ExecutionTime = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	defer func() {
		if err := r.runAfterTest(ctx, testCase); err != nil {
			fmt.Printf("Warning: after_test failed: %v\n", err)
		}
	}()

	mcpClient := mcp.NewClient()
	defer mcpClient.Close()

	if err := mcpClient.Start(ctx, r.mcpBinary, r.mcpArgs); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to start MCP server: %v", err)
		result.EndTime = time.Now()
		result.ExecutionTime = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	if _, err := mcpClient.Initialize(ctx); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to initialize MCP: %v", err)
		result.EndTime = time.Now()
		result.ExecutionTime = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	agentConfig.MCPClient = mcpClient
	ag := agent.NewAgent(agentConfig)

	testCtx, cancel := context.WithTimeout(ctx, testCase.Timeout)
	defer cancel()

	agentResult, err := ag.Run(testCtx, testCase.UserPrompt)
	
	result.EndTime = time.Now()
	result.ExecutionTime = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.ErrorMessage = fmt.Sprintf("agent failed: %v", err)
		if agentResult != nil {
			result.ToolCalls = agentResult.ToolCalls
		}
		return result, err
	}

	result.FinalAnswer = agentResult.Answer
	result.ToolCalls = agentResult.ToolCalls
	result.Status = loader.TestStatusPass

	return result, nil
}

func (r *Runner) RunTests(ctx context.Context, testCases []*loader.TestCase, agentConfig agent.AgentConfig) (*loader.BenchmarkSummary, error) {
	summary := &loader.BenchmarkSummary{
		Timestamp:   time.Now(),
		TotalTests:  len(testCases),
		TestResults: make([]*loader.TestResult, 0, len(testCases)),
		MCPBinary:   r.mcpBinary,
	}

	for _, tc := range testCases {
		fmt.Printf("Running test: %s - %s\n", tc.ID, tc.Title)

		result, err := r.RunTest(ctx, tc, agentConfig)
		summary.TestResults = append(summary.TestResults, result)

		switch result.Status {
		case loader.TestStatusPass:
			summary.PassedTests++
		case loader.TestStatusFail:
			summary.FailedTests++
		case loader.TestStatusError:
			summary.ErrorTests++
		case loader.TestStatusSkipped:
			summary.SkippedTests++
		}

		summary.TotalExecutionTime += result.ExecutionTime

		if err != nil {
			fmt.Printf("  ❌ FAILED: %s\n", err)
		} else {
			fmt.Printf("  ✅ PASSED\n")
		}
	}

	if summary.TotalTests > 0 {
		summary.SuccessRate = float64(summary.PassedTests) / float64(summary.TotalTests)
	}

	return summary, nil
}

func (r *Runner) runBeforeTest(ctx context.Context, tc *loader.TestCase) error {
	if tc.BeforeTest == "" {
		return nil
	}

	fmt.Printf("Running before_test for %s...\n", tc.ID)
	
	cmd := exec.CommandContext(ctx, "bash", "-c", tc.BeforeTest)
	cmd.Dir = tc.FilePath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("before_test failed: %w\nOutput: %s", err, string(output))
	}

	if len(output) > 0 {
		fmt.Printf("before_test output:\n%s\n", string(output))
	}

	return nil
}

func (r *Runner) runAfterTest(ctx context.Context, tc *loader.TestCase) error {
	if tc.AfterTest == "" {
		return nil
	}

	fmt.Printf("Running after_test for %s...\n", tc.ID)

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", tc.AfterTest)
	cmd.Dir = tc.FilePath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "NotFound") || strings.Contains(outputStr, "not found") {
			return nil
		}
		return fmt.Errorf("after_test failed: %w\nOutput: %s", err, outputStr)
	}

	if len(output) > 0 {
		fmt.Printf("after_test output:\n%s\n", string(output))
	}

	return nil
}
