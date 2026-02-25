package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	mcpclient "github.com/Azure/aks-mcp/test/e2e/pkg/client"
	"github.com/Azure/aks-mcp/test/e2e/pkg/runner"
	"github.com/Azure/aks-mcp/test/e2e/pkg/tests"
)

const (
	// Default to localhost for local development/testing
	// For in-cluster testing: http://aks-mcp.default.svc.cluster.local:8000
	defaultServerURL = "http://localhost:8000"
	defaultTimeout   = 5 * time.Minute
)

func main() {
	// Parse command-line flags
	serverURL := flag.String("server-url", getEnv("MCP_SERVER_URL", defaultServerURL), "MCP server URL")
	subscriptionID := flag.String("subscription-id", os.Getenv("AZURE_SUBSCRIPTION_ID"), "Azure subscription ID")
	resourceGroup := flag.String("resource-group", os.Getenv("RESOURCE_GROUP"), "Resource group name")
	clusterName := flag.String("cluster-name", os.Getenv("CLUSTER_NAME"), "AKS cluster name")
	nodePoolName := flag.String("node-pool-name", os.Getenv("NODE_POOL_NAME"), "Optional: specific node pool to test")
	verbose := flag.Bool("verbose", false, "Enable verbose output (show tool parameters and results)")
	verboseShort := flag.Bool("v", false, "Enable verbose output (short form)")
	timeout := flag.Duration("timeout", defaultTimeout, "Test timeout duration")

	flag.Parse()

	// Check for -v short form
	if *verboseShort {
		*verbose = true
	}

	// Validate required parameters
	if *subscriptionID == "" {
		fmt.Fprintf(os.Stderr, "Error: AZURE_SUBSCRIPTION_ID environment variable or --subscription-id flag is required\n")
		os.Exit(1)
	}
	if *resourceGroup == "" {
		fmt.Fprintf(os.Stderr, "Error: RESOURCE_GROUP environment variable or --resource-group flag is required\n")
		os.Exit(1)
	}
	if *clusterName == "" {
		fmt.Fprintf(os.Stderr, "Error: CLUSTER_NAME environment variable or --cluster-name flag is required\n")
		os.Exit(1)
	}

	// Print configuration
	fmt.Println("AKS-MCP E2E Test Runner")
	fmt.Println("=======================")
	fmt.Printf("MCP Server URL: %s\n", *serverURL)
	fmt.Printf("Subscription ID: %s\n", *subscriptionID)
	fmt.Printf("Resource Group: %s\n", *resourceGroup)
	fmt.Printf("Cluster Name: %s\n", *clusterName)
	if *nodePoolName != "" {
		fmt.Printf("Node Pool Name: %s\n", *nodePoolName)
	}
	fmt.Printf("Verbose Mode: %v\n", *verbose)
	fmt.Printf("Timeout: %s\n", *timeout)
	fmt.Println()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Create MCP client
	fmt.Printf("Connecting to MCP server at %s...\n", *serverURL)
	client, err := mcpclient.NewMCPClient(*serverURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create MCP client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Initialize MCP session
	fmt.Println("Initializing MCP session...")
	initResult, err := client.Initialize(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize MCP session: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Connected to MCP server: %s v%s\n", initResult.ServerInfo.Name, initResult.ServerInfo.Version)

	// List available tools
	fmt.Println("Listing available tools...")
	toolsResult, err := client.ListTools(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list tools: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d tools\n", len(toolsResult.Tools))
	if *verbose {
		fmt.Println("Available tools:")
		for _, tool := range toolsResult.Tools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
		fmt.Println()
	}

	// Create test runner
	testRunner := runner.NewTestRunner(client, *verbose)

	// Register tests
	fmt.Println("Registering tests...")

	// Test 1: Get VMSS info for all node pools
	testRunner.AddTest(&tests.GetVMSSInfoTest{
		SubscriptionID: *subscriptionID,
		ResourceGroup:  *resourceGroup,
		ClusterName:    *clusterName,
		NodePoolName:   "", // Empty means all node pools
	})

	// Test 2: Get VMSS info for specific node pool (if provided)
	if *nodePoolName != "" {
		testRunner.AddTest(&tests.GetVMSSInfoTest{
			SubscriptionID: *subscriptionID,
			ResourceGroup:  *resourceGroup,
			ClusterName:    *clusterName,
			NodePoolName:   *nodePoolName,
		})
	}

	// Run all tests
	fmt.Println("Starting test execution...")
	results, err := testRunner.RunAll(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Test execution failed: %v\n", err)
		os.Exit(1)
	}

	// Print results
	runner.PrintResults(results, *verbose)

	// Exit with appropriate code
	for _, result := range results {
		if !result.Passed {
			os.Exit(1)
		}
	}

	fmt.Println("\n✅ All tests passed!")
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
