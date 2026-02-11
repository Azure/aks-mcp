package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/ctx"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
	k8ssecurity "github.com/Azure/mcp-kubernetes/pkg/security"
)

type RunCommandExecutor struct{}

func NewRunCommandExecutor() *RunCommandExecutor {
	return &RunCommandExecutor{}
}

type RequestContext struct {
	AzureToken     string `json:"azure_token"`
	SubscriptionID string `json:"subscription_id"`
	ResourceGroup  string `json:"resource_group"`
	ClusterName    string `json:"cluster_name"`
}

func extractRequestContext(c context.Context, params map[string]interface{}) (*RequestContext, error) {
	tokenStr, ok := c.Value(ctx.AzureTokenKey).(string)
	if !ok || tokenStr == "" {
		return nil, fmt.Errorf("X-Azure-Token not found in context or empty")
	}

	var reqCtx RequestContext
	reqCtx.AzureToken = tokenStr

	// Extract aks_resource_id from params
	aksResourceID, ok := params["aks_resource_id"].(string)
	if !ok || aksResourceID == "" {
		return nil, fmt.Errorf("aks_resource_id not found in params or empty")
	}

	// Parse Azure Resource ID: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{cluster}
	resourceID, err := arm.ParseResourceID(aksResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse aks_resource_id: %w", err)
	}

	reqCtx.SubscriptionID = resourceID.SubscriptionID
	reqCtx.ResourceGroup = resourceID.ResourceGroupName
	reqCtx.ClusterName = resourceID.Name

	if reqCtx.AzureToken == "" {
		return nil, fmt.Errorf("azure_token is required in request_context")
	}
	if reqCtx.SubscriptionID == "" {
		return nil, fmt.Errorf("subscription_id is required in request_context")
	}
	if reqCtx.ResourceGroup == "" {
		return nil, fmt.Errorf("resource_group is required in request_context")
	}
	if reqCtx.ClusterName == "" {
		return nil, fmt.Errorf("cluster_name is required in request_context")
	}

	return &reqCtx, nil
}

func (e *RunCommandExecutor) Execute(ctx context.Context, params map[string]interface{}, cfg *k8sconfig.ConfigData) (string, error) {

	reqCtx, err := extractRequestContext(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to extract request context: %w", err)
	}

	command, err := e.buildCommand(params)
	if err != nil {
		return "", fmt.Errorf("failed to build command: %w", err)
	}

	// Validate command against security configuration
	if err := e.validateCommand(command, cfg); err != nil {
		return "", err
	}

	logger.Debugf("RunCommandExecutor: Command to execute: %s in cluster %s/%s", command, reqCtx.ResourceGroup, reqCtx.ClusterName)

	cred := azureclient.NewStaticTokenCredential(reqCtx.AzureToken)

	clientFactory, err := armcontainerservice.NewClientFactory(reqCtx.SubscriptionID, cred, &arm.ClientOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create Azure client for cluster %s/%s, command '%s': %w", reqCtx.ResourceGroup, reqCtx.ClusterName, command, err)
	}

	managedClustersClient := clientFactory.NewManagedClustersClient()

	runCommandRequest := armcontainerservice.RunCommandRequest{
		Command: &command,
	}

	poller, err := managedClustersClient.BeginRunCommand(ctx, reqCtx.ResourceGroup, reqCtx.ClusterName, runCommandRequest, nil)
	if err != nil {
		return "", fmt.Errorf("failed to start run command '%s' on cluster %s/%s: %w", command, reqCtx.ResourceGroup, reqCtx.ClusterName, err)
	}

	// Poll every 2 seconds instead of the default 30 seconds for faster response
	pollOptions := &runtime.PollUntilDoneOptions{
		Frequency: 2 * time.Second,
	}
	resp, err := poller.PollUntilDone(ctx, pollOptions)
	if err != nil {
		return "", fmt.Errorf("failed to execute command '%s' on cluster %s/%s: %w", command, reqCtx.ResourceGroup, reqCtx.ClusterName, err)
	}

	if resp.Properties == nil {
		return "", fmt.Errorf("run command '%s' on cluster %s/%s returned nil properties", command, reqCtx.ResourceGroup, reqCtx.ClusterName)
	}

	var output strings.Builder
	if resp.Properties.Logs != nil {
		output.WriteString(*resp.Properties.Logs)
	}

	// Check if command execution failed
	if resp.Properties.ExitCode != nil && *resp.Properties.ExitCode != 0 {
		return output.String(), fmt.Errorf("command '%s' on cluster %s/%s failed with exit code %d", command, reqCtx.ResourceGroup, reqCtx.ClusterName, *resp.Properties.ExitCode)
	}

	logger.Debugf("RunCommandExecutor: Command completed successfully")
	return output.String(), nil
}

func (e *RunCommandExecutor) buildCommand(params map[string]interface{}) (string, error) {
	cmd, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command parameter is required")
	}
	return cmd, nil
}

// validateCommand validates the kubectl command against security configuration
func (e *RunCommandExecutor) validateCommand(command string, cfg *k8sconfig.ConfigData) error {
	// Create validator with security config
	validator := k8ssecurity.NewValidator(cfg.SecurityConfig)

	// Remove "kubectl " prefix if present, as validator expects command without binary name
	fullCommand := strings.TrimPrefix(command, "kubectl ")

	// Validate command (includes both access level and namespace validation)
	if err := validator.ValidateCommand(fullCommand, k8ssecurity.CommandTypeKubectl); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}

	return nil
}
