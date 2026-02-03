package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
)

// VMRunCommandExecutor executes commands on VMSS instances using Azure Run Command
type VMRunCommandExecutor struct {
	azClient *azureclient.AzureClient
}

// NewVMRunCommandExecutor creates a new VMRunCommandExecutor
func NewVMRunCommandExecutor(azClient *azureclient.AzureClient) *VMRunCommandExecutor {
	return &VMRunCommandExecutor{
		azClient: azClient,
	}
}

// ExecuteOnVMSSInstance executes a shell command on a specific VMSS instance
func (e *VMRunCommandExecutor) ExecuteOnVMSSInstance(
	ctx context.Context,
	subscriptionID, resourceGroup, vmssName, instanceID string,
	command string,
) (string, error) {
	logger.Debugf("VMRunCommandExecutor: Executing command on VMSS %s/%s instance %s in subscription %s",
		resourceGroup, vmssName, instanceID, subscriptionID)

	// Get or create clients for the subscription
	clients, err := e.azClient.GetOrCreateClientsForSubscription(subscriptionID)
	if err != nil {
		return "", fmt.Errorf("failed to get Azure clients for subscription %s: %w", subscriptionID, err)
	}

	if clients.VMSSVMsClient == nil {
		return "", fmt.Errorf("VMSS VMs client not available for subscription %s", subscriptionID)
	}

	// Build RunCommandInput
	runCommandInput := armcompute.RunCommandInput{
		CommandID: to.Ptr("RunShellScript"),
		Script:    []*string{to.Ptr(command)},
	}

	logger.Debugf("VMRunCommandExecutor: Starting run command on instance %s", instanceID)

	// Execute the command
	poller, err := clients.VMSSVMsClient.BeginRunCommand(
		ctx,
		resourceGroup,
		vmssName,
		instanceID,
		runCommandInput,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to start run command on VMSS %s/%s instance %s: %w",
			resourceGroup, vmssName, instanceID, err)
	}

	// Wait for completion
	logger.Debugf("VMRunCommandExecutor: Waiting for command to complete")
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to execute run command on VMSS %s/%s instance %s: %w",
			resourceGroup, vmssName, instanceID, err)
	}

	// Extract output from response
	output, err := e.extractOutput(resp.RunCommandResult)
	if err != nil {
		return "", fmt.Errorf("failed to extract output from run command on VMSS %s/%s instance %s: %w",
			resourceGroup, vmssName, instanceID, err)
	}

	logger.Debugf("VMRunCommandExecutor: Command completed successfully")
	return output, nil
}

// extractOutput extracts stdout and stderr from RunCommandResult
func (e *VMRunCommandExecutor) extractOutput(result armcompute.RunCommandResult) (string, error) {
	var output strings.Builder

	if len(result.Value) == 0 {
		return "", fmt.Errorf("run command returned no output")
	}

	// Iterate through output instances
	for _, instance := range result.Value {
		if instance == nil {
			continue
		}

		// Log the instance for debugging
		if instance.Code != nil {
			logger.Debugf("VMRunCommandExecutor: Output instance code: %s", *instance.Code)
		}
		if instance.Message != nil {
			logger.Debugf("VMRunCommandExecutor: Output instance message length: %d", len(*instance.Message))
		}

		// Extract message from any instance that has one
		// The Azure VM RunCommand API can return different code formats
		if instance.Message != nil && *instance.Message != "" {
			// Check if this is stderr
			if instance.Code != nil && strings.Contains(*instance.Code, "StdErr") {
				output.WriteString("STDERR:\n")
				output.WriteString(*instance.Message)
				output.WriteString("\n")
			} else {
				// Treat as stdout
				output.WriteString(*instance.Message)
			}
		}
	}

	outputStr := output.String()
	logger.Debugf("VMRunCommandExecutor: Total extracted output length: %d", len(outputStr))

	return outputStr, nil
}
