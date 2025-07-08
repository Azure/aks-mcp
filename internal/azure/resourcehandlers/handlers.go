// Package resourcehandlers provides handler functions for Azure resource tools.
package resourcehandlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/aks-mcp/internal/azure"
	"github.com/Azure/aks-mcp/internal/azure/applens"
	"github.com/Azure/aks-mcp/internal/azure/resourcehelpers"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// =============================================================================
// Network-related Handlers
// =============================================================================

// GetVNetInfoHandler returns a handler for the get_vnet_info command
func GetVNetInfoHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}

		// Get the cluster details
		ctx := context.Background()
		cluster, err := GetClusterDetails(ctx, client, subID, rg, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get cluster details: %v", err)
		}

		// Get the VNet ID from the cluster
		vnetID, err := resourcehelpers.GetVNetIDFromAKS(ctx, cluster, client)
		if err != nil {
			return "", fmt.Errorf("failed to get VNet ID: %v", err)
		}

		// Get the VNet details using the resource ID
		vnetInterface, err := client.GetResourceByID(ctx, vnetID)
		if err != nil {
			return "", fmt.Errorf("failed to get VNet details: %v", err)
		}

		vnet, ok := vnetInterface.(*armnetwork.VirtualNetwork)
		if !ok {
			return "", fmt.Errorf("unexpected resource type returned for VNet")
		}

		// Return the VNet details directly as JSON
		resultJSON, err := json.MarshalIndent(vnet, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal VNet info to JSON: %v", err)
		}

		return string(resultJSON), nil
	})
}

// GetNSGInfoHandler returns a handler for the get_nsg_info command
func GetNSGInfoHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}

		// Get the cluster details
		ctx := context.Background()
		cluster, err := GetClusterDetails(ctx, client, subID, rg, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get cluster details: %v", err)
		}

		// Get the NSG ID from the cluster
		nsgID, err := resourcehelpers.GetNSGIDFromAKS(ctx, cluster, client)
		if err != nil {
			return "", fmt.Errorf("failed to get NSG ID: %v", err)
		}

		// Get the NSG details using the resource ID
		nsgInterface, err := client.GetResourceByID(ctx, nsgID)
		if err != nil {
			return "", fmt.Errorf("failed to get NSG details: %v", err)
		}

		nsg, ok := nsgInterface.(*armnetwork.SecurityGroup)
		if !ok {
			return "", fmt.Errorf("unexpected resource type returned for NSG")
		}

		// Return the NSG details directly as JSON
		resultJSON, err := json.MarshalIndent(nsg, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal NSG info to JSON: %v", err)
		}

		return string(resultJSON), nil
	})
}

// GetRouteTableInfoHandler returns a handler for the get_route_table_info command
func GetRouteTableInfoHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}

		// Get the cluster details
		ctx := context.Background()
		cluster, err := GetClusterDetails(ctx, client, subID, rg, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get cluster details: %v", err)
		}

		// Get the RouteTable ID from the cluster
		rtID, err := resourcehelpers.GetRouteTableIDFromAKS(ctx, cluster, client)
		if err != nil {
			return "", fmt.Errorf("failed to get RouteTable ID: %v", err)
		}

		// Check if no route table is attached (valid configuration state)
		if rtID == "" {
			// Return a message indicating no route table is attached
			response := map[string]interface{}{
				"message": "No route table attached to the AKS cluster subnet",
				"reason":  "This is normal for AKS clusters using Azure CNI with Overlay mode or clusters that rely on Azure's default routing",
			}
			resultJSON, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				return "", fmt.Errorf("failed to marshal response to JSON: %v", err)
			}
			return string(resultJSON), nil
		}

		// Get the RouteTable details using the resource ID
		rtInterface, err := client.GetResourceByID(ctx, rtID)
		if err != nil {
			return "", fmt.Errorf("failed to get RouteTable details: %v", err)
		}

		rt, ok := rtInterface.(*armnetwork.RouteTable)
		if !ok {
			return "", fmt.Errorf("unexpected resource type returned for RouteTable")
		}

		// Return the RouteTable details directly as JSON
		resultJSON, err := json.MarshalIndent(rt, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal RouteTable info to JSON: %v", err)
		}

		return string(resultJSON), nil
	})
}

// GetSubnetInfoHandler returns a handler for the get_subnet_info command
func GetSubnetInfoHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}

		// Get the cluster details
		ctx := context.Background()
		cluster, err := GetClusterDetails(ctx, client, subID, rg, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get cluster details: %v", err)
		}

		// Get the Subnet ID from the cluster
		subnetID, err := resourcehelpers.GetSubnetIDFromAKS(ctx, cluster, client)
		if err != nil {
			return "", fmt.Errorf("failed to get Subnet ID: %v", err)
		}

		// Get the Subnet details using the resource ID
		subnetInterface, err := client.GetResourceByID(ctx, subnetID)
		if err != nil {
			return "", fmt.Errorf("failed to get Subnet details: %v", err)
		}

		subnet, ok := subnetInterface.(*armnetwork.Subnet)
		if !ok {
			return "", fmt.Errorf("unexpected resource type returned for Subnet")
		}

		// Return the Subnet details directly as JSON
		resultJSON, err := json.MarshalIndent(subnet, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal Subnet info to JSON: %v", err)
		}

		return string(resultJSON), nil
	})
}

// GetLoadBalancersInfoHandler returns a handler for the get_load_balancers_info command
func GetLoadBalancersInfoHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}

		// Get the cluster details
		ctx := context.Background()
		cluster, err := GetClusterDetails(ctx, client, subID, rg, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get cluster details: %v", err)
		}

		// Get the Load Balancer IDs from the cluster
		lbIDs, err := resourcehelpers.GetLoadBalancerIDsFromAKS(ctx, cluster, client)
		if err != nil {
			return "", fmt.Errorf("failed to get Load Balancer IDs: %v", err)
		}

		// Check if no load balancers are found (valid configuration state)
		if len(lbIDs) == 0 {
			// Return a message indicating no standard AKS load balancers are found
			response := map[string]interface{}{
				"message": "No AKS load balancers (kubernetes/kubernetes-internal) found for this cluster",
				"reason":  "This cluster may not have standard AKS load balancers configured, or it may be using a different networking setup.",
			}
			resultJSON, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				return "", fmt.Errorf("failed to marshal response to JSON: %v", err)
			}
			return string(resultJSON), nil
		}

		// Get details for each load balancer
		var loadBalancers []interface{}
		for _, lbID := range lbIDs {
			lbInterface, err := client.GetResourceByID(ctx, lbID)
			if err != nil {
				return "", fmt.Errorf("failed to get Load Balancer details for %s: %v", lbID, err)
			}

			lb, ok := lbInterface.(*armnetwork.LoadBalancer)
			if !ok {
				return "", fmt.Errorf("unexpected resource type returned for Load Balancer %s", lbID)
			}

			loadBalancers = append(loadBalancers, lb)
		}

		// If only one load balancer, return it directly for backward compatibility
		if len(loadBalancers) == 1 {
			resultJSON, err := json.MarshalIndent(loadBalancers[0], "", "  ")
			if err != nil {
				return "", fmt.Errorf("failed to marshal Load Balancer info to JSON: %v", err)
			}
			return string(resultJSON), nil
		}

		// If multiple load balancers, return them as an array
		result := map[string]interface{}{
			"count":          len(loadBalancers),
			"load_balancers": loadBalancers,
		}

		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal Load Balancer info to JSON: %v", err)
		}

		return string(resultJSON), nil
	})
}

// =============================================================================
// Shared Helper Functions
// =============================================================================

// ExtractAKSParameters extracts and validates the common AKS parameters from the params map
func ExtractAKSParameters(params map[string]interface{}) (subscriptionID, resourceGroup, clusterName string, err error) {
	subID, ok := params["subscription_id"].(string)
	if !ok || subID == "" {
		return "", "", "", fmt.Errorf("missing or invalid subscription_id parameter")
	}

	rg, ok := params["resource_group"].(string)
	if !ok || rg == "" {
		return "", "", "", fmt.Errorf("missing or invalid resource_group parameter")
	}

	clusterNameParam, ok := params["cluster_name"].(string)
	if !ok || clusterNameParam == "" {
		return "", "", "", fmt.Errorf("missing or invalid cluster_name parameter")
	}

	return subID, rg, clusterNameParam, nil
}

// GetClusterDetails gets the details of an AKS cluster
func GetClusterDetails(ctx context.Context, client *azure.AzureClient, subscriptionID, resourceGroup, clusterName string) (*armcontainerservice.ManagedCluster, error) {
	// Get the cluster from Azure client (which now handles caching internally)
	return client.GetAKSCluster(ctx, subscriptionID, resourceGroup, clusterName)
}

// =============================================================================
// AppLens Diagnostic Handlers
// =============================================================================

// ListAppLensDetectorsHandler returns a handler for the list_applens_detectors command
func ListAppLensDetectorsHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract cluster resource ID
		clusterResourceID, ok := params["cluster_resource_id"].(string)
		if !ok || clusterResourceID == "" {
			return "", fmt.Errorf("missing or invalid cluster_resource_id parameter")
		}

		// Extract optional category filter
		category, _ := params["category"].(string)

		// Validate cluster resource ID format
		subscriptionID, _, _, err := applens.ExtractClusterInfo(clusterResourceID)
		if err != nil {
			return "", fmt.Errorf("invalid cluster resource ID: %v", err)
		}

		// Get clients for the subscription to ensure subscription is accessible
		_, err = client.GetOrCreateClientsForSubscription(subscriptionID)
		if err != nil {
			return "", fmt.Errorf("failed to get Azure clients: %v", err)
		}

		// Create detector manager
		detectorManager, err := applens.NewDetectorManager(subscriptionID, client.GetCredential())
		if err != nil {
			return "", fmt.Errorf("failed to create detector manager: %v", err)
		}

		// List detectors
		ctx := context.Background()
		result, err := detectorManager.ListDetectors(ctx, clusterResourceID, category)
		if err != nil {
			return "", fmt.Errorf("failed to list AppLens detectors: %v", err)
		}

		return result, nil
	})
}

// InvokeAppLensDetectorHandler returns a handler for the invoke_applens_detector command
func InvokeAppLensDetectorHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract cluster resource ID
		clusterResourceID, ok := params["cluster_resource_id"].(string)
		if !ok || clusterResourceID == "" {
			return "", fmt.Errorf("missing or invalid cluster_resource_id parameter")
		}

		// Extract detector name (optional - if not provided, list detectors)
		detectorName, _ := params["detector_name"].(string)
		
		// Extract optional time range
		timeRange, _ := params["time_range"].(string)

		// Validate cluster resource ID format
		subscriptionID, _, _, err := applens.ExtractClusterInfo(clusterResourceID)
		if err != nil {
			return "", fmt.Errorf("invalid cluster resource ID: %v", err)
		}

		// Get clients for the subscription to ensure subscription is accessible
		_, err = client.GetOrCreateClientsForSubscription(subscriptionID)
		if err != nil {
			return "", fmt.Errorf("failed to get Azure clients: %v", err)
		}

		// Create detector manager
		detectorManager, err := applens.NewDetectorManager(subscriptionID, client.GetCredential())
		if err != nil {
			return "", fmt.Errorf("failed to create detector manager: %v", err)
		}

		ctx := context.Background()

		// If no detector name provided, list available detectors
		if detectorName == "" {
			result, err := detectorManager.ListDetectors(ctx, clusterResourceID, "")
			if err != nil {
				return "", fmt.Errorf("failed to list AppLens detectors: %v", err)
			}
			return result, nil
		}

		// Invoke the specific detector
		result, err := detectorManager.InvokeDetector(ctx, clusterResourceID, detectorName, timeRange)
		if err != nil {
			return "", fmt.Errorf("failed to invoke AppLens detector: %v", err)
		}

		return result, nil
	})
}
