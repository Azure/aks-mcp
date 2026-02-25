package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	mcpclient "github.com/Azure/aks-mcp/test/e2e/pkg/client"
	"github.com/Azure/aks-mcp/test/e2e/pkg/tests"
)

// TestRunner executes E2E tests
type TestRunner struct {
	client  *mcpclient.MCPClient
	tests   []tests.ToolTest
	verbose bool
}

// TestResult holds the result of a single test
type TestResult struct {
	TestName   string
	Passed     bool
	Error      error
	Duration   time.Duration
	ToolError  bool // true if error was from tool call, false if from validation
	ToolParams map[string]interface{}
	ToolResult string
}

// NewTestRunner creates a new test runner
func NewTestRunner(client *mcpclient.MCPClient, verbose bool) *TestRunner {
	return &TestRunner{
		client:  client,
		tests:   []tests.ToolTest{},
		verbose: verbose,
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
		TestName:   test.Name(),
		Passed:     false,
		ToolParams: test.GetParams(),
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

	// Extract text content for verbose output
	if toolResult != nil && len(toolResult.Content) > 0 {
		for _, content := range toolResult.Content {
			if tc, ok := content.(mcp.TextContent); ok {
				result.ToolResult = tc.Text
				break
			}
		}
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
func PrintResults(results []TestResult, verbose bool) {
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	fmt.Println("\n" + repeatString("=", 80))
	fmt.Println("Test Results")
	fmt.Println(repeatString("=", 80))

	for i, result := range results {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(results), result.TestName)
		fmt.Printf("    Duration: %s\n", result.Duration)

		// Print parameters in verbose mode
		if verbose && result.ToolParams != nil {
			fmt.Println("    Parameters:")
			paramsJSON, _ := json.MarshalIndent(result.ToolParams, "      ", "  ")
			fmt.Printf("      %s\n", string(paramsJSON))
		}

		if result.Passed {
			fmt.Printf("    Status: ✅ PASS\n")
			passed++

			// Print result in verbose mode
			if verbose && result.ToolResult != "" {
				fmt.Println("    Result:")
				// Try to pretty-print JSON
				var jsonData interface{}
				if err := json.Unmarshal([]byte(result.ToolResult), &jsonData); err == nil {
					prettyJSON, _ := json.MarshalIndent(jsonData, "      ", "  ")
					fmt.Printf("      %s\n", string(prettyJSON))
				} else {
					// Not JSON, print as-is (show more for debugging, up to 2000 chars)
					if len(result.ToolResult) > 2000 {
						fmt.Printf("      %s...(truncated at 2000 chars)\n", result.ToolResult[:2000])
					} else {
						fmt.Printf("      %s\n", result.ToolResult)
					}
				}
			}
		} else {
			fmt.Printf("    Status: ❌ FAIL\n")
			if result.ToolError {
				fmt.Printf("    Error Type: Tool call error\n")
			} else {
				fmt.Printf("    Error Type: Validation error\n")
			}
			fmt.Printf("    Error: %v\n", result.Error)

			// Print result even on failure in verbose mode
			if verbose && result.ToolResult != "" {
				fmt.Println("    Full Error Response:")
				// Show more content for errors (up to 2000 chars)
				if len(result.ToolResult) > 2000 {
					fmt.Printf("      %s...(truncated at 2000 chars)\n", result.ToolResult[:2000])
				} else {
					fmt.Printf("      %s\n", result.ToolResult)
				}
			}
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
