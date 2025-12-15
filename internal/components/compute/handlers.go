// Package compute provides handler functions for Azure compute resource tools.
package compute

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/components/common"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
)

// =============================================================================
// Compute-related Handlers
// =============================================================================

// GetAKSVMSSInfoHandler returns a handler for the get_aks_vmss_info command
func GetAKSVMSSInfoHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := common.ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}

		// Get the cluster details
		cluster, err := common.GetClusterDetails(ctx, client, subID, rg, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get cluster details: %v", err)
		}

		// Check if node_pool_name is provided
		nodePoolName, hasNodePool := params["node_pool_name"].(string)

		// If no node pool name or empty string, return all node pools
		if !hasNodePool || nodePoolName == "" {
			return getAllVMSSByCluster(ctx, client, cluster, subID, rg, clusterName)
		}

		// Handle single node pool case
		return getSingleVMSSByNodePool(ctx, client, cluster, nodePoolName, rg, clusterName)
	})
}

// getSingleVMSSByNodePool handles getting VMSS info for a single node pool
func getSingleVMSSByNodePool(ctx context.Context, client *azureclient.AzureClient, cluster interface{}, nodePoolName, rg, clusterName string) (string, error) {
	// Type assert cluster to the correct type
	aksCluster, ok := cluster.(*armcontainerservice.ManagedCluster)
	if !ok {
		return "", fmt.Errorf("invalid cluster type")
	}

	// Get the VMSS ID from the cluster
	vmssID, err := GetVMSSIDFromNodePool(ctx, aksCluster, nodePoolName, client)
	if err != nil {
		return "", fmt.Errorf("failed to get VMSS ID: %v", err)
	}

	if vmssID == "" {
		// Return a message indicating no VMSS found for this node pool
		response := map[string]interface{}{
			"message":        fmt.Sprintf("No VMSS found for node pool '%s'", nodePoolName),
			"node_pool":      nodePoolName,
			"cluster_name":   clusterName,
			"resource_group": rg,
		}
		resultJSON, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal response to JSON: %v", err)
		}
		return string(resultJSON), nil
	}

	// Get the VMSS details using the resource ID
	vmssInterface, err := client.GetResourceByID(ctx, vmssID)
	if err != nil {
		return "", fmt.Errorf("failed to get VMSS details: %v", err)
	}

	vmss, ok := vmssInterface.(*armcompute.VirtualMachineScaleSet)
	if !ok {
		return "", fmt.Errorf("unexpected resource type returned for VMSS")
	}

	// Return the VMSS details directly as JSON
	resultJSON, err := json.MarshalIndent(vmss, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal VMSS info to JSON: %v", err)
	}

	return string(resultJSON), nil
}

// getAllVMSSByCluster handles getting VMSS info for all node pools in the cluster
func getAllVMSSByCluster(ctx context.Context, client *azureclient.AzureClient, cluster interface{}, subID, rg, clusterName string) (string, error) {
	// Type assert cluster to the correct type
	aksCluster, ok := cluster.(*armcontainerservice.ManagedCluster)
	if !ok {
		return "", fmt.Errorf("invalid cluster type")
	}

	// Get all node pools from the cluster
	nodePools, err := GetNodePoolsFromAKS(ctx, aksCluster, client)
	if err != nil {
		return "", fmt.Errorf("failed to get node pools: %v", err)
	}

	// Get VMSS information for each node pool
	var vmssInfo []map[string]interface{}

	for _, nodePool := range nodePools {
		if nodePool.Name == nil {
			continue
		}

		nodePoolName := *nodePool.Name

		// Get the VMSS ID for this node pool
		vmssID, err := GetVMSSIDFromNodePool(ctx, aksCluster, nodePoolName, client)
		if err != nil {
			// Log the error but continue with other node pools
			vmssInfo = append(vmssInfo, map[string]interface{}{
				"node_pool": nodePoolName,
				"error":     fmt.Sprintf("Failed to get VMSS ID: %v", err),
			})
			continue
		}

		if vmssID == "" {
			vmssInfo = append(vmssInfo, map[string]interface{}{
				"node_pool": nodePoolName,
				"message":   "No VMSS found for this node pool",
			})
			continue
		}

		// Get the VMSS details
		vmssInterface, err := client.GetResourceByID(ctx, vmssID)
		if err != nil {
			vmssInfo = append(vmssInfo, map[string]interface{}{
				"node_pool": nodePoolName,
				"vmss_id":   vmssID,
				"error":     fmt.Sprintf("Failed to get VMSS details: %v", err),
			})
			continue
		}

		vmss, ok := vmssInterface.(*armcompute.VirtualMachineScaleSet)
		if !ok {
			vmssInfo = append(vmssInfo, map[string]interface{}{
				"node_pool": nodePoolName,
				"vmss_id":   vmssID,
				"error":     "Unexpected resource type returned for VMSS",
			})
			continue
		}

		vmssInfo = append(vmssInfo, map[string]interface{}{
			"node_pool": nodePoolName,
			"vmss_id":   vmssID,
			"vmss":      vmss,
		})
	}

	// Return the results
	result := map[string]interface{}{
		"cluster_name":     clusterName,
		"resource_group":   rg,
		"node_pools_count": len(nodePools),
		"vmss_info":        vmssInfo,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal VMSS info to JSON: %v", err)
	}

	return string(resultJSON), nil
}
