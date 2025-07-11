package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
)

// GetNodePoolsFromAKS extracts all node pools from an AKS cluster
func GetNodePoolsFromAKS(
	ctx context.Context,
	cluster *armcontainerservice.ManagedCluster,
	client *azureclient.AzureClient,
) ([]*armcontainerservice.ManagedClusterAgentPoolProfile, error) {
	// Ensure the cluster is valid
	if cluster == nil || cluster.Properties == nil {
		return nil, fmt.Errorf("invalid cluster or cluster properties")
	}

	var nodePools []*armcontainerservice.ManagedClusterAgentPoolProfile

	// Add the system node pool (default node pool)
	if cluster.Properties.AgentPoolProfiles != nil {
		nodePools = append(nodePools, cluster.Properties.AgentPoolProfiles...)
	}

	return nodePools, nil
}

// GetVMSSIDFromNodePool extracts the VMSS resource ID from a node pool
func GetVMSSIDFromNodePool(
	ctx context.Context,
	cluster *armcontainerservice.ManagedCluster,
	nodePoolName string,
	client *azureclient.AzureClient,
) (string, error) {
	// Ensure the cluster is valid
	if cluster == nil || cluster.Properties == nil {
		return "", fmt.Errorf("invalid cluster or cluster properties")
	}

	// Get subscription ID and node resource group
	subscriptionID := getSubscriptionFromCluster(cluster)
	if subscriptionID == "" {
		return "", fmt.Errorf("unable to extract subscription ID from cluster")
	}
	if cluster.Properties.NodeResourceGroup == nil {
		return "", fmt.Errorf("node resource group not found for AKS cluster")
	}
	nodeResourceGroup := *cluster.Properties.NodeResourceGroup

	// Find the VMSS for the specified node pool
	return findVMSSForNodePool(ctx, client, subscriptionID, nodeResourceGroup, nodePoolName)
}

// GetVMSSInstancesFromNodePool gets VM instances from a VMSS for a specific node pool
func GetVMSSInstancesFromNodePool(
	ctx context.Context,
	cluster *armcontainerservice.ManagedCluster,
	nodePoolName string,
	client *azureclient.AzureClient,
) ([]interface{}, error) {
	// Get the VMSS ID first
	vmssID, err := GetVMSSIDFromNodePool(ctx, cluster, nodePoolName, client)
	if err != nil {
		return nil, err
	}

	if vmssID == "" {
		return nil, fmt.Errorf("no VMSS found for node pool %s", nodePoolName)
	}

	// Extract resource group and VMSS name from the ID
	parts := strings.Split(vmssID, "/")
	if len(parts) < 9 {
		return nil, fmt.Errorf("invalid VMSS resource ID format: %s", vmssID)
	}

	subscriptionID := parts[2]
	resourceGroup := parts[4]
	vmssName := parts[8]

	// Get clients for the subscription
	clients, err := client.GetOrCreateClientsForSubscription(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clients for subscription %s: %v", subscriptionID, err)
	}

	// List VM instances in the VMSS
	var instances []interface{}
	pager := clients.VMSSVMsClient.NewListPager(resourceGroup, vmssName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list VMSS instances: %v", err)
		}

		for _, instance := range page.Value {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// findVMSSForNodePool looks for VMSS in the node resource group that matches the node pool
func findVMSSForNodePool(
	ctx context.Context,
	client *azureclient.AzureClient,
	subscriptionID string,
	nodeResourceGroup string,
	nodePoolName string,
) (string, error) {
	// Get clients for the subscription
	clients, err := client.GetOrCreateClientsForSubscription(subscriptionID)
	if err != nil {
		return "", fmt.Errorf("failed to get clients for subscription %s: %v", subscriptionID, err)
	}

	// List VMSS in the node resource group
	pager := clients.VMSSClient.NewListPager(nodeResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to list VMSS in resource group %s: %v", nodeResourceGroup, err)
		}

		for _, vmss := range page.Value {
			if vmss.Name != nil && vmss.ID != nil {
				vmssName := *vmss.Name
				vmssID := *vmss.ID

				// AKS VMSS naming convention: aks-{nodepool}-{random}-vmss
				// Match by node pool name in the VMSS name
				if strings.Contains(vmssName, fmt.Sprintf("aks-%s-", nodePoolName)) {
					return vmssID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no VMSS found for node pool %s", nodePoolName)
}

// getSubscriptionFromCluster extracts subscription ID from cluster resource ID
func getSubscriptionFromCluster(cluster *armcontainerservice.ManagedCluster) string {
	if cluster.ID == nil {
		return ""
	}

	// Parse the resource ID to extract subscription ID
	// Format: /subscriptions/{subscription-id}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{name}
	parts := strings.Split(*cluster.ID, "/")
	if len(parts) >= 3 && parts[1] == "subscriptions" {
		return parts[2]
	}

	return ""
}
