package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
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

func extractRequestContext(ctx context.Context) (*RequestContext, error) {
	requestContextStr, ok := ctx.Value("request_context").(string)
	if !ok || requestContextStr == "" {
		return nil, fmt.Errorf("request_context not found in context or empty")
	}

	var reqCtx RequestContext
	if err := json.Unmarshal([]byte(requestContextStr), &reqCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request_context: %w", err)
	}

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
	logger.Debugf("RunCommandExecutor: Executing kubectl command via AKS RunCommand API")

	reqCtx, err := extractRequestContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to extract request context: %w", err)
	}

	logger.Debugf("RunCommandExecutor: Using cluster %s in resource group %s", reqCtx.ClusterName, reqCtx.ResourceGroup)

	command, err := e.buildCommand(params)
	if err != nil {
		return "", fmt.Errorf("failed to build command: %w", err)
	}

	logger.Debugf("RunCommandExecutor: Command to execute: %s", command)

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

	logger.Debugf("RunCommandExecutor: Waiting for command to complete...")
	resp, err := poller.PollUntilDone(ctx, nil)
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
