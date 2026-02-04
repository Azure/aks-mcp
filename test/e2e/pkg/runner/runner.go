package runner

import (
	"context"
	"fmt"
	"time"

	mcpclient "github.com/Azure/aks-mcp/test/e2e/pkg/client"
	"github.com/Azure/aks-mcp/test/e2e/pkg/tests"
)

// TestRunner executes E2E tests
type TestRunner struct {
	client *mcpclient.MCPClient
	tests  []tests.ToolTest
}

// TestResult holds the result of a single test
type TestResult struct {
	TestName  string
	Passed    bool
	Error     error
	Duration  time.Duration
	ToolError bool // true if error was from tool call, false if from validation
}

// NewTestRunner creates a new test runner
func NewTestRunner(client *mcpclient.MCPClient) *TestRunner {
	return &TestRunner{
		client: client,
		tests:  []tests.ToolTest{},
	}
}

// AddTest adds a test to the runner
func (r *TestRunner) AddTest(test tests.ToolTest) {
	r.tests = append(r.tests, test)
}

// RunAll executes all registered tests
func (r *TestRunner) RunAll(ctx context.Context) ([]TestResult, error) {
	results := make([]TestResult, 0, len(r.tests))

	for _, test := range r.tests {
		result := r.runSingleTest(ctx, test)
		results = append(results, result)
	}

	return results, nil
}

// runSingleTest executes a single test and returns the result
func (r *TestRunner) runSingleTest(ctx context.Context, test tests.ToolTest) TestResult {
	result := TestResult{
		TestName: test.Name(),
		Passed:   false,
	}

	start := time.Now()

	// Execute the tool
	toolResult, err := test.Run(ctx, r.client.GetInternalClient())
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err
		result.ToolError = true
		return result
	}

	// Validate the result
	if err := test.Validate(toolResult); err != nil {
		result.Error = fmt.Errorf("validation failed: %w", err)
		result.ToolError = false
		return result
	}

	result.Passed = true
	return result
}

// PrintResults prints test results in a human-readable format
func PrintResults(results []TestResult) {
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	fmt.Println("\n" + repeatString("=", 80))
	fmt.Println("Test Results")
	fmt.Println(repeatString("=", 80))

	for i, result := range results {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(results), result.TestName)
		fmt.Printf("    Duration: %s\n", result.Duration)

		if result.Passed {
			fmt.Printf("    Status: ✅ PASS\n")
			passed++
		} else {
			fmt.Printf("    Status: ❌ FAIL\n")
			if result.ToolError {
				fmt.Printf("    Error Type: Tool call error\n")
			} else {
				fmt.Printf("    Error Type: Validation error\n")
			}
			fmt.Printf("    Error: %v\n", result.Error)
			failed++
		}

		totalDuration += result.Duration
	}

	fmt.Println("\n" + repeatString("=", 80))
	fmt.Printf("Summary: %d total, %d passed, %d failed\n", len(results), passed, failed)
	fmt.Printf("Total Duration: %s\n", totalDuration)
	fmt.Println(repeatString("=", 80))
}

// repeatString repeats a string n times
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
