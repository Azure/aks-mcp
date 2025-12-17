package azcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/components/fleet/kubernetes"
	"github.com/Azure/aks-mcp/internal/config"
)

// FleetExecutor handles structured fleet command execution
type FleetExecutor struct {
	*AzExecutor
	k8sClient            *kubernetes.Client
	placementOps         *kubernetes.PlacementOperations
	k8sClientInitialized bool
}

// NewFleetExecutor creates a new fleet command executor
func NewFleetExecutor() *FleetExecutor {
	return &FleetExecutor{
		AzExecutor:           NewExecutor(),
		k8sClientInitialized: false,
	}
}

// Execute processes structured fleet commands
func (e *FleetExecutor) Execute(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Extract structured parameters
	operation, ok := params["operation"].(string)
	if !ok {
		return "", fmt.Errorf("operation parameter is required and must be a string")
	}

	resource, ok := params["resource"].(string)
	if !ok {
		return "", fmt.Errorf("resource parameter is required and must be a string")
	}

	args, ok := params["args"].(string)
	if !ok {
		return "", fmt.Errorf("args parameter is required and must be a string")
	}

	// Route clusterresourceplacement operations to Kubernetes
	if resource == "clusterresourceplacement" {
		// Validate clusterresourceplacement operations separately
		if err := e.validateClusterResourcePlacementCombination(operation); err != nil {
			return "", err
		}
		return e.executeKubernetesClusterResourcePlacement(ctx, operation, args, cfg)
	}

	// Validate operation/resource combination for non-placement resources
	if err := e.validateCombination(operation, resource); err != nil {
		return "", err
	}

	// Construct the full command
	var command string
	if operation == "list" && resource == "fleet" {
		// Special case: "az fleet list" without resource in between
		command = "az fleet list"
	} else if operation == "get-credentials" && resource == "fleet" {
		// Special case: "az fleet get-credentials"
		command = "az fleet get-credentials"
	} else {
		command = fmt.Sprintf("az fleet %s %s", resource, operation)
	}

	// Check access level
	if err := e.checkAccessLevel(operation, resource, cfg.AccessLevel); err != nil {
		return "", err
	}

	// Build full command with args
	fullCommand := command
	if args != "" {
		fullCommand = fmt.Sprintf("%s %s", command, args)
	}

	// Create params for the base executor
	execParams := map[string]interface{}{
		"command": fullCommand,
	}

	// Execute using the base executor
	return e.AzExecutor.Execute(ctx, execParams, cfg)
}

// validateCombination validates if the operation/resource combination is valid
func (e *FleetExecutor) validateCombination(operation, resource string) error {
	validCombinations := map[string][]string{
		"fleet":          {"list", "show", "create", "update", "delete", "get-credentials"},
		"member":         {"list", "show", "create", "update", "delete"},
		"updaterun":      {"list", "show", "create", "start", "stop", "delete"},
		"updatestrategy": {"list", "show", "create", "delete"},
	}

	validOps, exists := validCombinations[resource]
	if !exists {
		return fmt.Errorf("invalid resource type: %s", resource)
	}

	for _, validOp := range validOps {
		if operation == validOp {
			return nil
		}
	}

	return fmt.Errorf("invalid operation '%s' for resource '%s'. Valid operations: %s",
		operation, resource, strings.Join(validOps, ", "))
}

// checkAccessLevel ensures the operation is allowed for the current access level
func (e *FleetExecutor) checkAccessLevel(operation, resource string, accessLevel string) error {
	// Read-only operations are allowed for all access levels
	readOnlyOps := []string{"list", "show", "get", "get-credentials"}
	for _, op := range readOnlyOps {
		if operation == op {
			return nil
		}
	}

	// Write operations require readwrite or admin access
	if accessLevel == "readonly" {
		return fmt.Errorf("operation '%s' requires readwrite or admin access level, current level is readonly", operation)
	}

	// All operations are allowed for readwrite and admin
	return nil
}

// GetCommandForValidation returns the constructed command for security validation
func (e *FleetExecutor) GetCommandForValidation(operation, resource, args string) string {
	var command string
	if operation == "list" && resource == "fleet" {
		command = "az fleet list"
	} else if operation == "get-credentials" && resource == "fleet" {
		command = "az fleet get-credentials"
	} else {
		command = fmt.Sprintf("az fleet %s %s", resource, operation)
	}

	if args != "" {
		command = fmt.Sprintf("%s %s", command, args)
	}

	return command
}

// executeKubernetesClusterResourcePlacement handles clusterresourceplacement operations via Kubernetes API
func (e *FleetExecutor) executeKubernetesClusterResourcePlacement(ctx context.Context, operation, args string, cfg *config.ConfigData) (string, error) {
	// Check access level for clusterresourceplacement operations
	if err := e.checkAccessLevel(operation, "clusterresourceplacement", cfg.AccessLevel); err != nil {
		return "", err
	}

	// Initialize Kubernetes client if needed
	if !e.k8sClientInitialized {
		if err := e.initializeKubernetesClient(); err != nil {
			return "", err
		}
	}

	// Check if placement operations are initialized
	if e.placementOps == nil {
		return "", fmt.Errorf("clusterresourceplacement operations not initialized")
	}

	// Parse arguments
	parsedArgs, parseErr := kubernetes.ParsePlacementArgs(args)
	if parseErr != nil {
		return "", fmt.Errorf("failed to parse clusterresourceplacement arguments: %w", parseErr)
	}

	// Execute the clusterresourceplacement operation with error recovery
	var result string
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				// Provide a helpful error message without stdout pollution
				err = fmt.Errorf("kubectl operation failed. Please ensure kubectl is installed, properly configured, and the cluster is accessible. Error: %v", r)
			}
		}()

		switch operation {
		case "create":
			result, err = e.createClusterResourcePlacement(ctx, parsedArgs, cfg)
		case "get", "show":
			result, err = e.getClusterResourcePlacement(ctx, parsedArgs, cfg)
		case "list":
			result, err = e.placementOps.ListPlacements(ctx, cfg)
		case "delete":
			result, err = e.deleteClusterResourcePlacement(ctx, parsedArgs, cfg)
		default:
			err = fmt.Errorf("unsupported clusterresourceplacement operation: %s", operation)
		}

	}()

	// Clean and validate the result for MCP compatibility
	if err == nil {
		result = cleanResult(result)
	}

	return result, err
}

// cleanResult sanitizes the result for MCP compatibility
func cleanResult(result string) string {
	// Just clean problematic characters and return as-is
	cleaned := result
	cleaned = strings.ReplaceAll(cleaned, "\x00", "") // Null bytes
	cleaned = strings.ReplaceAll(cleaned, "\r", "")   // Carriage returns
	cleaned = strings.ReplaceAll(cleaned, "\x1b", "") // Escape sequences
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// initializeKubernetesClient initializes the Kubernetes client
func (e *FleetExecutor) initializeKubernetesClient() error {
	defer func() {
		// Recover from any panics during client initialization
		if r := recover(); r != nil {
			e.k8sClientInitialized = false
		}
	}()

	client, err := kubernetes.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Kubernetes client. Please ensure kubectl is installed and kubeconfig is properly configured: %w", err)
	}

	// Test if the client is actually usable
	if client == nil {
		return fmt.Errorf("kubernetes client is nil after initialization")
	}

	e.k8sClient = client
	e.placementOps = kubernetes.NewPlacementOperations(client)
	e.k8sClientInitialized = true

	return nil
}

// validateClusterResourcePlacementCombination validates clusterresourceplacement operations
func (e *FleetExecutor) validateClusterResourcePlacementCombination(operation string) error {
	validOps := []string{"list", "show", "get", "create", "delete"}

	for _, validOp := range validOps {
		if operation == validOp {
			return nil
		}
	}

	return fmt.Errorf("invalid operation '%s' for resource 'clusterresourceplacement'. Valid operations: %s",
		operation, strings.Join(validOps, ", "))
}

// createClusterResourcePlacement creates a clusterresourceplacement using placement operations
func (e *FleetExecutor) createClusterResourcePlacement(ctx context.Context, args map[string]string, cfg *config.ConfigData) (string, error) {
	name, ok := args["name"]
	if !ok || name == "" {
		return "", fmt.Errorf("--name is required for create operation")
	}

	selector := args["selector"]
	policy := args["policy"]

	// Default policy if not specified
	if policy == "" {
		policy = "PickAll"
	}

	// Validate policy
	validPolicies := []string{"PickAll", "PickFixed", "PickN"}
	isValidPolicy := false
	for _, validPolicy := range validPolicies {
		if strings.EqualFold(policy, validPolicy) {
			policy = validPolicy
			isValidPolicy = true
			break
		}
	}
	if !isValidPolicy {
		return "", fmt.Errorf("invalid policy '%s'. Valid policies: %s", policy, strings.Join(validPolicies, ", "))
	}

	if e.placementOps == nil {
		return "", fmt.Errorf("clusterresourceplacement operations not initialized")
	}

	return e.placementOps.CreatePlacement(ctx, name, selector, policy, cfg)
}

// getClusterResourcePlacement retrieves a clusterresourceplacement using placement operations
func (e *FleetExecutor) getClusterResourcePlacement(ctx context.Context, args map[string]string, cfg *config.ConfigData) (string, error) {
	name, ok := args["name"]
	if !ok || name == "" {
		return "", fmt.Errorf("--name is required for get/show operation")
	}

	if e.placementOps == nil {
		return "", fmt.Errorf("clusterresourceplacement operations not initialized")
	}

	return e.placementOps.GetPlacement(ctx, name, cfg)
}

// deleteClusterResourcePlacement deletes a clusterresourceplacement using placement operations
func (e *FleetExecutor) deleteClusterResourcePlacement(ctx context.Context, args map[string]string, cfg *config.ConfigData) (string, error) {
	name, ok := args["name"]
	if !ok || name == "" {
		return "", fmt.Errorf("--name is required for delete operation")
	}

	if e.placementOps == nil {
		return "", fmt.Errorf("clusterresourceplacement operations not initialized")
	}

	return e.placementOps.DeletePlacement(ctx, name, cfg)
}
