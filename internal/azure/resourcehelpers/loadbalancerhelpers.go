package resourcehelpers

import (
	"context"
	"fmt"

	"github.com/Azure/aks-mcp/internal/azure"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
)

// GetLoadBalancerIDsFromAKS extracts all load balancer IDs from an AKS cluster.
// It looks for load balancers in the node resource group that match the AKS cluster
// naming convention. AKS clusters can have multiple load balancers (e.g., kubernetes, kubernetes-internal).
func GetLoadBalancerIDsFromAKS(
	ctx context.Context,
	cluster *armcontainerservice.ManagedCluster,
	client *azure.AzureClient,
) ([]string, error) {
	// Ensure the cluster is valid
	if cluster == nil || cluster.Properties == nil {
		return nil, fmt.Errorf("invalid cluster or cluster properties")
	}

	// Get subscription ID and node resource group
	subscriptionID := getSubscriptionFromCluster(cluster)
	if cluster.Properties.NodeResourceGroup == nil {
		return nil, fmt.Errorf("node resource group not found for AKS cluster")
	}
	nodeResourceGroup := *cluster.Properties.NodeResourceGroup

	// Get cluster name for naming convention matching
	clusterName := ""
	if cluster.Name != nil {
		clusterName = *cluster.Name
	}

	// Look for load balancers in the node resource group
	return findLoadBalancersInNodeResourceGroup(ctx, client, subscriptionID, nodeResourceGroup, clusterName)
}

// findLoadBalancersInNodeResourceGroup looks for all load balancers in the node resource group
// that follow AKS naming conventions. AKS typically creates load balancers with names:
// - kubernetes (for the main external load balancer)
// - kubernetes-internal (for internal load balancer)
func findLoadBalancersInNodeResourceGroup(
	ctx context.Context,
	client *azure.AzureClient,
	subscriptionID string,
	nodeResourceGroup string,
	clusterName string,
) ([]string, error) {
	// Get clients for the subscription
	clients, err := client.GetOrCreateClientsForSubscription(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clients for subscription %s: %v", subscriptionID, err)
	}

	var loadBalancerIDs []string

	// List load balancers in the node resource group
	pager := clients.LoadBalancerClient.NewListPager(nodeResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list load balancers in resource group %s: %v", nodeResourceGroup, err)
		}

		for _, lb := range page.Value {
			if lb.Name != nil && lb.ID != nil {
				lbName := *lb.Name
				lbID := *lb.ID

				// Only check for standard AKS load balancer names
				if lbName == "kubernetes" || lbName == "kubernetes-internal" {
					loadBalancerIDs = append(loadBalancerIDs, lbID)
				}
			}
		}
	}

	// If no standard AKS load balancers found, return empty slice
	if len(loadBalancerIDs) == 0 {
		return []string{}, nil
	}

	return loadBalancerIDs, nil
}
