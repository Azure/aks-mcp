package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/bench/pkg/agent"
	"github.com/Azure/aks-mcp/bench/pkg/evaluator"
	"github.com/Azure/aks-mcp/bench/pkg/loader"
	"github.com/Azure/aks-mcp/bench/pkg/mcp"
)

type Runner struct {
	mcpBinary      string
	mcpArgs        []string
	toolValidator  *evaluator.ToolValidator
	llmJudge       *evaluator.LLMJudge
	parallel       int
}

type RunnerConfig struct {
	MCPBinary   string
	MCPArgs     []string
	LLMClient   agent.LLMClient
	Parallel    int
}

func NewRunner(config RunnerConfig) *Runner {
	if config.MCPBinary == "" {
		config.MCPBinary = "../aks-mcp"
	}
	
	var llmJudge *evaluator.LLMJudge
	if config.LLMClient != nil {
		llmJudge = evaluator.NewLLMJudge(config.LLMClient)
	}
	
	parallel := config.Parallel
	if parallel <= 0 {
		parallel = 1
	}
	
	return &Runner{
		mcpBinary:     config.MCPBinary,
		mcpArgs:       config.MCPArgs,
		toolValidator: evaluator.NewToolValidator(),
		llmJudge:      llmJudge,
		parallel:      parallel,
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
	result.LLMIterations = agentResult.LLMIterations
	
	// Calculate performance metrics
	for _, tc := range result.ToolCalls {
		result.TotalToolCallTime += tc.ExecutionTime
	}
	if len(result.ToolCalls) > 0 {
		result.AvgToolCallDuration = result.TotalToolCallTime / time.Duration(len(result.ToolCalls))
	}

	if len(testCase.ExpectedToolUsage) > 0 {
		result.ValidationResults = r.toolValidator.ValidateToolUsage(
			testCase.ExpectedToolUsage,
			result.ToolCalls,
		)

		toolSelectionScore, parameterAccuracy := r.toolValidator.CalculateOverallScore(result.ValidationResults)
		result.ToolSelectionScore = toolSelectionScore
		result.ParameterAccuracy = parameterAccuracy

		allValid := true
		for _, vr := range result.ValidationResults {
			if !vr.ToolCalled || !vr.ArgsValid {
				allValid = false
				break
			}
		}

		if !allValid {
			result.Status = loader.TestStatusFail
			return result, nil
		}
	}

	if len(testCase.ExpectedOutput) > 0 && r.llmJudge != nil {
		judgingResult, err := r.llmJudge.JudgeAnswer(ctx, testCase, result.FinalAnswer)
		if err != nil {
			fmt.Printf("Warning: LLM judge failed: %v\n", err)
		} else {
			result.JudgingResult = judgingResult
			result.OutputQualityScore = judgingResult.Score

			if judgingResult.Score < 0.7 {
				result.Status = loader.TestStatusFail
				return result, nil
			}
		}
	}

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

	if r.parallel == 1 {
		return r.runTestsSequential(ctx, testCases, agentConfig, summary)
	}
	return r.runTestsParallel(ctx, testCases, agentConfig, summary)
}

func (r *Runner) runTestsSequential(ctx context.Context, testCases []*loader.TestCase, agentConfig agent.AgentConfig, summary *loader.BenchmarkSummary) (*loader.BenchmarkSummary, error) {
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

		if result.ToolSelectionScore > 0 {
			summary.AvgToolSelectionScore += result.ToolSelectionScore
		}
		if result.ParameterAccuracy > 0 {
			summary.AvgParameterAccuracy += result.ParameterAccuracy
		}
		if result.OutputQualityScore > 0 {
			summary.AvgOutputQualityScore += result.OutputQualityScore
		}

		summary.TotalExecutionTime += result.ExecutionTime
		summary.AvgToolCallDuration += result.TotalToolCallTime
		summary.AvgToolCallsPerTest += float64(len(result.ToolCalls))
		summary.AvgLLMIterations += float64(result.LLMIterations)

		if err != nil {
			fmt.Printf("  ❌ FAILED: %s\n", err)
		} else if result.Status == loader.TestStatusFail {
			fmt.Printf("  ❌ FAILED: judging score %.2f < 0.7\n", result.OutputQualityScore)
		} else {
			fmt.Printf("  ✅ PASSED\n")
		}
	}

	if summary.TotalTests > 0 {
		summary.SuccessRate = float64(summary.PassedTests) / float64(summary.TotalTests)
		
		if summary.AvgToolSelectionScore > 0 {
			summary.AvgToolSelectionScore /= float64(summary.TotalTests)
		}
		if summary.AvgParameterAccuracy > 0 {
			summary.AvgParameterAccuracy /= float64(summary.TotalTests)
		}
		if summary.AvgOutputQualityScore > 0 {
			summary.AvgOutputQualityScore /= float64(summary.TotalTests)
		}
		
		summary.AvgTestDuration = summary.TotalExecutionTime / time.Duration(summary.TotalTests)
		
		totalToolCalls := int64(summary.AvgToolCallsPerTest)
		if totalToolCalls > 0 {
			summary.AvgToolCallDuration = summary.AvgToolCallDuration / time.Duration(totalToolCalls)
		}
		summary.AvgToolCallsPerTest /= float64(summary.TotalTests)
		summary.AvgLLMIterations /= float64(summary.TotalTests)
	}

	return summary, nil
}

func (r *Runner) runTestsParallel(ctx context.Context, testCases []*loader.TestCase, agentConfig agent.AgentConfig, summary *loader.BenchmarkSummary) (*loader.BenchmarkSummary, error) {
	type testJob struct {
		testCase *loader.TestCase
		index    int
	}

	type testResult struct {
		result *loader.TestResult
		index  int
		err    error
	}

	jobs := make(chan testJob, len(testCases))
	results := make(chan testResult, len(testCases))

	for i := 0; i < r.parallel; i++ {
		go func() {
			for job := range jobs {
				result, err := r.RunTest(ctx, job.testCase, agentConfig)
				results <- testResult{result: result, index: job.index, err: err}
			}
		}()
	}

	for i, tc := range testCases {
		jobs <- testJob{testCase: tc, index: i}
	}
	close(jobs)

	testResults := make([]*loader.TestResult, len(testCases))
	for i := 0; i < len(testCases); i++ {
		res := <-results
		testResults[res.index] = res.result
		
		status := "✅ PASSED"
		if res.err != nil {
			status = fmt.Sprintf("❌ FAILED: %s", res.err)
		} else if res.result.Status == loader.TestStatusFail {
			status = fmt.Sprintf("❌ FAILED: judging score %.2f < 0.7", res.result.OutputQualityScore)
		}
		fmt.Printf("Test %s - %s: %s\n", res.result.TestCase.ID, res.result.TestCase.Title, status)
	}
	close(results)

	summary.TestResults = testResults

	for _, result := range testResults {
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

		if result.ToolSelectionScore > 0 {
			summary.AvgToolSelectionScore += result.ToolSelectionScore
		}
		if result.ParameterAccuracy > 0 {
			summary.AvgParameterAccuracy += result.ParameterAccuracy
		}
		if result.OutputQualityScore > 0 {
			summary.AvgOutputQualityScore += result.OutputQualityScore
		}

		summary.TotalExecutionTime += result.ExecutionTime
		summary.AvgToolCallDuration += result.TotalToolCallTime
		summary.AvgToolCallsPerTest += float64(len(result.ToolCalls))
		summary.AvgLLMIterations += float64(result.LLMIterations)
	}

	if summary.TotalTests > 0 {
		summary.SuccessRate = float64(summary.PassedTests) / float64(summary.TotalTests)
		
		if summary.AvgToolSelectionScore > 0 {
			summary.AvgToolSelectionScore /= float64(summary.TotalTests)
		}
		if summary.AvgParameterAccuracy > 0 {
			summary.AvgParameterAccuracy /= float64(summary.TotalTests)
		}
		if summary.AvgOutputQualityScore > 0 {
			summary.AvgOutputQualityScore /= float64(summary.TotalTests)
		}
		
		summary.AvgTestDuration = summary.TotalExecutionTime / time.Duration(summary.TotalTests)
		
		totalToolCalls := int64(summary.AvgToolCallsPerTest)
		if totalToolCalls > 0 {
			summary.AvgToolCallDuration = summary.AvgToolCallDuration / time.Duration(totalToolCalls)
		}
		summary.AvgToolCallsPerTest /= float64(summary.TotalTests)
		summary.AvgLLMIterations /= float64(summary.TotalTests)
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
