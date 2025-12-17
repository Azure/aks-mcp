// Package resourcehandlers provides handler functions for Azure resource tools.
package network

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/components/common"
	"github.com/Azure/aks-mcp/internal/components/network/resourcehelpers"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// =============================================================================
// Network-related Handlers
// =============================================================================

// GetVNetInfoHandler returns a handler for the get_vnet_info command
func GetVNetInfoHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := common.ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}
		cluster, err := common.GetClusterDetails(ctx, client, subID, rg, clusterName)
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
func GetNSGInfoHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := common.ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}
		cluster, err := common.GetClusterDetails(ctx, client, subID, rg, clusterName)
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
func GetRouteTableInfoHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := common.ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}
		cluster, err := common.GetClusterDetails(ctx, client, subID, rg, clusterName)
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
func GetSubnetInfoHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := common.ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}
		cluster, err := common.GetClusterDetails(ctx, client, subID, rg, clusterName)
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
func GetLoadBalancersInfoHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters
		subID, rg, clusterName, err := common.ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}
		cluster, err := common.GetClusterDetails(ctx, client, subID, rg, clusterName)
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

// GetPrivateEndpointInfoHandler returns a handler for the get_private_endpoint_info command
func GetPrivateEndpointInfoHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract parameters using common helper
		subID, rg, clusterName, err := common.ExtractAKSParameters(params)
		if err != nil {
			return "", err
		}

		// Get the cluster details to verify it exists and get node resource group
		cluster, err := client.GetAKSCluster(ctx, subID, rg, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get AKS cluster: %v", err)
		}

		// Check if cluster is private and get private endpoint info
		privateEndpointID, err := resourcehelpers.GetPrivateEndpointIDFromAKS(ctx, cluster, client)
		if err != nil {
			return "", fmt.Errorf("failed to get private endpoint info: %v", err)
		}

		// If no private endpoint found, return appropriate message
		if privateEndpointID == "" {
			result := map[string]interface{}{
				"message":         "No private endpoint found. This AKS cluster is not configured as a private cluster.",
				"private_cluster": false,
			}
			jsonData, err := json.Marshal(result)
			if err != nil {
				return "", fmt.Errorf("failed to marshal response: %v", err)
			}
			return string(jsonData), nil
		}

		// Get the private endpoint details using the resource ID
		privateEndpoint, err := client.GetPrivateEndpointByID(ctx, privateEndpointID)
		if err != nil {
			return "", fmt.Errorf("failed to get private endpoint details: %v", err)
		}

		// Return the private endpoint details directly as JSON
		jsonData, err := json.Marshal(privateEndpoint)
		if err != nil {
			return "", fmt.Errorf("failed to marshal private endpoint details: %v", err)
		}

		return string(jsonData), nil
	})
}

// GetAzNetworkResourcesHandler returns a handler for the az_network_resources command
func GetAzNetworkResourcesHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		resourceType, subID, rg, clusterName, err := validateNetworkParams(params)
		if err != nil {
			return "", err
		}

		// Handle resource type
		return handleNetworkResourceType(ctx, client, resourceType, subID, rg, clusterName)
	})
}

// validateNetworkParams validates network resource parameters
func validateNetworkParams(params map[string]interface{}) (string, string, string, string, error) {
	// Extract resource_type parameter
	resourceType, ok := params["resource_type"].(string)
	if !ok {
		return "", "", "", "", fmt.Errorf("missing or invalid 'resource_type' parameter")
	}

	// Validate resource type
	if !ValidateNetworkResourceType(resourceType) {
		supportedTypes := GetSupportedNetworkResourceTypes()
		return "", "", "", "", fmt.Errorf("unsupported resource type: %s. Supported types: %v", resourceType, supportedTypes)
	}

	// Extract common AKS parameters
	subID, rg, clusterName, err := common.ExtractAKSParameters(params)
	if err != nil {
		return "", "", "", "", err
	}

	return resourceType, subID, rg, clusterName, nil
}

// handleNetworkResourceType routes to the appropriate resource handler based on type
func handleNetworkResourceType(ctx context.Context, client *azureclient.AzureClient, resourceType, subID, rg, clusterName string) (string, error) {
	switch resourceType {
	case string(ResourceTypeAll):
		return handleAllNetworkResources(ctx, client, subID, rg, clusterName)
	case string(ResourceTypeVNet):
		return handleVNetResource(ctx, client, subID, rg, clusterName)
	case string(ResourceTypeNSG):
		return handleNSGResource(ctx, client, subID, rg, clusterName)
	case string(ResourceTypeRouteTable):
		return handleRouteTableResource(ctx, client, subID, rg, clusterName)
	case string(ResourceTypeSubnet):
		return handleSubnetResource(ctx, client, subID, rg, clusterName)
	case string(ResourceTypeLoadBalancer):
		return handleLoadBalancerResource(ctx, client, subID, rg, clusterName)
	case string(ResourceTypePrivateEndpoint):
		return handlePrivateEndpointResource(ctx, client, subID, rg, clusterName)
	default:
		return "", fmt.Errorf("resource type '%s' not implemented", resourceType)
	}
}

// Helper functions for different resource types

func handleAllNetworkResources(ctx context.Context, client *azureclient.AzureClient, subID, rg, clusterName string) (string, error) {
	result := make(map[string]interface{})

	// Collect results and errors for each resource type
	resourceHandlers := map[string]func(context.Context, *azureclient.AzureClient, string, string, string) (string, error){
		"vnet":             handleVNetResource,
		"nsg":              handleNSGResource,
		"route_table":      handleRouteTableResource,
		"subnet":           handleSubnetResource,
		"load_balancer":    handleLoadBalancerResource,
		"private_endpoint": handlePrivateEndpointResource,
	}

	// Process each resource type and preserve error context
	for resourceType, handler := range resourceHandlers {
		resourceResult, err := handler(ctx, client, subID, rg, clusterName)
		if err != nil {
			// Preserve original error context and type for debugging
			result[resourceType+"_error"] = map[string]interface{}{
				"message": err.Error(),
				"type":    fmt.Sprintf("%T", err),
			}
		} else {
			result[resourceType] = json.RawMessage(resourceResult)
		}
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	return string(resultJSON), nil
}

func handleVNetResource(ctx context.Context, client *azureclient.AzureClient, subID, rg, clusterName string) (string, error) {
	// Use the existing VNet handler logic
	handler := GetVNetInfoHandler(client, nil)
	params := map[string]interface{}{
		"subscription_id": subID,
		"resource_group":  rg,
		"cluster_name":    clusterName,
	}
	return handler.Handle(ctx, params, nil)
}

func handleNSGResource(ctx context.Context, client *azureclient.AzureClient, subID, rg, clusterName string) (string, error) {
	// Use the existing NSG handler logic
	handler := GetNSGInfoHandler(client, nil)
	params := map[string]interface{}{
		"subscription_id": subID,
		"resource_group":  rg,
		"cluster_name":    clusterName,
	}
	return handler.Handle(ctx, params, nil)
}

func handleRouteTableResource(ctx context.Context, client *azureclient.AzureClient, subID, rg, clusterName string) (string, error) {
	// Use the existing Route Table handler logic
	handler := GetRouteTableInfoHandler(client, nil)
	params := map[string]interface{}{
		"subscription_id": subID,
		"resource_group":  rg,
		"cluster_name":    clusterName,
	}
	return handler.Handle(ctx, params, nil)
}

func handleSubnetResource(ctx context.Context, client *azureclient.AzureClient, subID, rg, clusterName string) (string, error) {
	// Use the existing Subnet handler logic
	handler := GetSubnetInfoHandler(client, nil)
	params := map[string]interface{}{
		"subscription_id": subID,
		"resource_group":  rg,
		"cluster_name":    clusterName,
	}
	return handler.Handle(ctx, params, nil)
}

func handleLoadBalancerResource(ctx context.Context, client *azureclient.AzureClient, subID, rg, clusterName string) (string, error) {
	// Use the existing Load Balancer handler logic
	handler := GetLoadBalancersInfoHandler(client, nil)
	params := map[string]interface{}{
		"subscription_id": subID,
		"resource_group":  rg,
		"cluster_name":    clusterName,
	}
	return handler.Handle(ctx, params, nil)
}

func handlePrivateEndpointResource(ctx context.Context, client *azureclient.AzureClient, subID, rg, clusterName string) (string, error) {
	// Use the existing Private Endpoint handler logic
	handler := GetPrivateEndpointInfoHandler(client, nil)
	params := map[string]interface{}{
		"subscription_id": subID,
		"resource_group":  rg,
		"cluster_name":    clusterName,
	}
	return handler.Handle(ctx, params, nil)
}
