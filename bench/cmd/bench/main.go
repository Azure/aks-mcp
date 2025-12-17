package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Azure/aks-mcp/bench/pkg/agent"
	"github.com/Azure/aks-mcp/bench/pkg/loader"
	"github.com/Azure/aks-mcp/bench/pkg/reporter"
	"github.com/Azure/aks-mcp/bench/pkg/runner"
)

const version = "0.1.0"

func main() {
	var (
		testPath           string
		mcpBinary          string
		accessLevel        string
		outputPath         string
		markdownPath       string
		perfPath           string
		tagsStr            string
		showVersion        bool
		parallel           int
		enableMultiCluster bool
	)

	flag.StringVar(&testPath, "test", "", "Path to test case file or directory (required)")
	flag.StringVar(&mcpBinary, "mcp-binary", "../aks-mcp", "Path to aks-mcp binary")
	flag.StringVar(&accessLevel, "access-level", "readonly", "Access level for MCP server (readonly/readwrite/admin)")
	flag.StringVar(&outputPath, "output", "results/latest.json", "Output path for results JSON")
	flag.StringVar(&markdownPath, "markdown", "results/latest.md", "Output path for markdown report")
	flag.StringVar(&perfPath, "perf", "results/performance.md", "Output path for performance analysis report")
	flag.StringVar(&tagsStr, "tags", "", "Comma-separated list of tags to filter tests")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.IntVar(&parallel, "parallel", 1, "Number of tests to run in parallel (default: 1 for sequential)")
	flag.BoolVar(&enableMultiCluster, "enable-multi-cluster", false, "Enable multi-cluster mode for kubectl (uses Azure AKS RunCommand API instead of local kubeconfig)")
	
	flag.Parse()

	if showVersion {
		fmt.Printf("aks-mcp-bench version %s\n", version)
		return
	}

	if testPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --test is required\n")
		flag.Usage()
		os.Exit(1)
	}

	azureConfig := agent.AzureConfig{
		Endpoint:   os.Getenv("AZURE_OPENAI_ENDPOINT"),
		APIKey:     os.Getenv("AZURE_OPENAI_API_KEY"),
		Deployment: os.Getenv("AZURE_OPENAI_DEPLOYMENT"),
	}

	if azureConfig.Endpoint == "" || azureConfig.APIKey == "" {
		fmt.Fprintf(os.Stderr, "Error: AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_API_KEY environment variables are required\n")
		os.Exit(1)
	}

	llmClient, err := agent.NewAzureOpenAIClient(azureConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create LLM client: %v\n", err)
		os.Exit(1)
	}

	testLoader := loader.NewLoader()
	testCases, err := testLoader.LoadTestCases(testPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load test cases: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d test case(s)\n", len(testCases))

	mcpArgs := []string{"--access-level", accessLevel}
	if enableMultiCluster {
		mcpArgs = append(mcpArgs, "--enable-multi-cluster")
	}
	testRunner := runner.NewRunner(runner.RunnerConfig{
		MCPBinary: mcpBinary,
		MCPArgs:   mcpArgs,
		LLMClient: llmClient,
		Parallel:  parallel,
	})

	agentConfig := agent.AgentConfig{
		LLMClient:     llmClient,
		MaxIterations: 10,
	}

	ctx := context.Background()

	fmt.Printf("\n=== Running Tests ===\n\n")
	summary, err := testRunner.RunTests(ctx, testCases, agentConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: test run failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("Total Tests:   %d\n", summary.TotalTests)
	fmt.Printf("Passed:        %d\n", summary.PassedTests)
	fmt.Printf("Failed:        %d\n", summary.FailedTests)
	fmt.Printf("Errors:        %d\n", summary.ErrorTests)
	fmt.Printf("Success Rate:  %.1f%%\n", summary.SuccessRate*100)
	if summary.AvgToolSelectionScore > 0 {
		fmt.Printf("Avg Tool Selection: %.1f%%\n", summary.AvgToolSelectionScore*100)
	}
	if summary.AvgParameterAccuracy > 0 {
		fmt.Printf("Avg Param Accuracy: %.1f%%\n", summary.AvgParameterAccuracy*100)
	}
	if summary.AvgOutputQualityScore > 0 {
		fmt.Printf("Avg Output Quality: %.2f\n", summary.AvgOutputQualityScore)
	}
	fmt.Printf("Total Time:    %s\n", summary.TotalExecutionTime)
	fmt.Printf("Avg Test Time: %s\n", summary.AvgTestDuration)
	fmt.Printf("Avg Tool Call Time: %s\n", summary.AvgToolCallDuration)
	fmt.Printf("Avg Tool Calls/Test: %.1f\n", summary.AvgToolCallsPerTest)
	fmt.Printf("Avg LLM Iterations: %.1f\n", summary.AvgLLMIterations)

	if outputPath != "" {
		jsonReporter := reporter.NewJSONReporter()
		if err := jsonReporter.Generate(summary, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save JSON results: %v\n", err)
		} else {
			fmt.Printf("\nJSON results saved to: %s\n", outputPath)
		}
	}

	if markdownPath != "" {
		markdownReporter := reporter.NewMarkdownReporter()
		if err := markdownReporter.Generate(summary, markdownPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save markdown report: %v\n", err)
		} else {
			fmt.Printf("Markdown report saved to: %s\n", markdownPath)
		}
	}

	if perfPath != "" {
		perfReporter := reporter.NewPerformanceReporter()
		if err := perfReporter.Generate(summary, perfPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save performance report: %v\n", err)
		} else {
			fmt.Printf("Performance report saved to: %s\n", perfPath)
		}
	}

	if summary.FailedTests > 0 || summary.ErrorTests > 0 {
		os.Exit(1)
	}
}

