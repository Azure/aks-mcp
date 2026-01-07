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
	AzureToken    string `json:"azure_token"`
	AKSResourceID string `json:"aks_resource_id"`
}

type ParsedResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
}

func parseAKSResourceID(resourceID string) (*ParsedResourceID, error) {
	parts := strings.Split(strings.TrimPrefix(resourceID, "/"), "/")
	if len(parts) < 8 {
		return nil, fmt.Errorf("invalid AKS resource ID format: %s", resourceID)
	}

	var subscriptionID, resourceGroup, clusterName string
	for i := 0; i < len(parts)-1; i++ {
		switch parts[i] {
		case "subscriptions":
			subscriptionID = parts[i+1]
		case "resourceGroups":
			resourceGroup = parts[i+1]
		case "managedClusters":
			clusterName = parts[i+1]
		}
	}

	if subscriptionID == "" || resourceGroup == "" || clusterName == "" {
		return nil, fmt.Errorf("failed to parse AKS resource ID: missing required fields in %s", resourceID)
	}

	return &ParsedResourceID{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ClusterName:    clusterName,
	}, nil
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
	if reqCtx.AKSResourceID == "" {
		return nil, fmt.Errorf("aks_resource_id is required in request_context")
	}

	return &reqCtx, nil
}

func (e *RunCommandExecutor) Execute(ctx context.Context, params map[string]interface{}, cfg *k8sconfig.ConfigData) (string, error) {
	logger.Debugf("RunCommandExecutor: Executing kubectl command via AKS RunCommand API")

	reqCtx, err := extractRequestContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to extract request context: %w", err)
	}

	parsedID, err := parseAKSResourceID(reqCtx.AKSResourceID)
	if err != nil {
		return "", fmt.Errorf("failed to parse AKS resource ID: %w", err)
	}

	logger.Debugf("RunCommandExecutor: Using cluster %s in resource group %s", parsedID.ClusterName, parsedID.ResourceGroup)

	command, err := e.buildCommand(params)
	if err != nil {
		return "", fmt.Errorf("failed to build command: %w", err)
	}

	logger.Debugf("RunCommandExecutor: Command to execute: %s", command)

	cred := azureclient.NewStaticTokenCredential(reqCtx.AzureToken)

	clientFactory, err := armcontainerservice.NewClientFactory(parsedID.SubscriptionID, cred, &arm.ClientOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create Azure client for cluster %s/%s, command '%s': %w", parsedID.ResourceGroup, parsedID.ClusterName, command, err)
	}

	managedClustersClient := clientFactory.NewManagedClustersClient()

	runCommandRequest := armcontainerservice.RunCommandRequest{
		Command: &command,
	}

	poller, err := managedClustersClient.BeginRunCommand(ctx, parsedID.ResourceGroup, parsedID.ClusterName, runCommandRequest, nil)
	if err != nil {
		return "", fmt.Errorf("failed to start run command '%s' on cluster %s/%s: %w", command, parsedID.ResourceGroup, parsedID.ClusterName, err)
	}

	logger.Debugf("RunCommandExecutor: Waiting for command to complete...")
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to execute command '%s' on cluster %s/%s: %w", command, parsedID.ResourceGroup, parsedID.ClusterName, err)
	}

	if resp.Properties == nil {
		return "", fmt.Errorf("run command '%s' on cluster %s/%s returned nil properties", command, parsedID.ResourceGroup, parsedID.ClusterName)
	}

	var output strings.Builder
	if resp.Properties.Logs != nil {
		output.WriteString(*resp.Properties.Logs)
	}

	// Check if command execution failed
	if resp.Properties.ExitCode != nil && *resp.Properties.ExitCode != 0 {
		return output.String(), fmt.Errorf("command '%s' on cluster %s/%s failed with exit code %d", command, parsedID.ResourceGroup, parsedID.ClusterName, *resp.Properties.ExitCode)
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
