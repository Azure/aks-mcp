// Package resourcehandlers provides handler functions for Azure resource tools.
package resourcehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/azure"
	"github.com/Azure/aks-mcp/internal/azure/advisor"
	"github.com/Azure/aks-mcp/internal/azure/applens"
	"github.com/Azure/aks-mcp/internal/azure/resourcehealth"
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

// =============================================================================
// Resource Health Handlers
// =============================================================================

// GetResourceHealthStatusHandler returns a handler for the get_resource_health_status command
func GetResourceHealthStatusHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract resource IDs
		resourceIDsParam, ok := params["resource_ids"]
		if !ok {
			return "", fmt.Errorf("missing resource_ids parameter")
		}

		// Handle both single string and array of strings
		var resourceIDs []string
		switch v := resourceIDsParam.(type) {
		case string:
			resourceIDs = []string{v}
		case []interface{}:
			for _, id := range v {
				if strID, ok := id.(string); ok {
					resourceIDs = append(resourceIDs, strID)
				} else {
					return "", fmt.Errorf("invalid resource ID in array: must be string")
				}
			}
		case []string:
			resourceIDs = v
		default:
			return "", fmt.Errorf("invalid resource_ids parameter: must be string or array of strings")
		}

		// Extract optional include_history parameter
		includeHistory, _ := params["include_history"].(bool)

		// Validate resource IDs
		if err := resourcehealth.ValidateResourceIDs(resourceIDs); err != nil {
			return "", fmt.Errorf("invalid resource IDs: %v", err)
		}

		// Extract subscription ID from the first resource ID for client management
		parts := strings.Split(resourceIDs[0], "/")
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid resource ID format")
		}
		subscriptionID := parts[2]

		// Get clients for the subscription to ensure subscription is accessible
		_, err := client.GetOrCreateClientsForSubscription(subscriptionID)
		if err != nil {
			return "", fmt.Errorf("failed to get Azure clients: %v", err)
		}

		// Create event manager
		eventManager, err := resourcehealth.NewEventManager(subscriptionID, client.GetCredential())
		if err != nil {
			return "", fmt.Errorf("failed to create event manager: %v", err)
		}

		// Get resource health status
		ctx := context.Background()
		result, err := eventManager.GetResourceHealthStatus(ctx, resourceIDs, includeHistory)
		if err != nil {
			return "", fmt.Errorf("failed to get resource health status: %v", err)
		}

		return result, nil
	})
}

// GetResourceHealthEventsHandler returns a handler for the get_resource_health_events command
func GetResourceHealthEventsHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract resource ID
		resourceID, ok := params["resource_id"].(string)
		if !ok || resourceID == "" {
			return "", fmt.Errorf("missing or invalid resource_id parameter")
		}

		// Extract optional time range parameters
		startTimeStr, _ := params["start_time"].(string)
		endTimeStr, _ := params["end_time"].(string)

		// Extract optional health status filter
		var healthStatusFilter []string
		if filter, ok := params["health_status_filter"]; ok {
			switch v := filter.(type) {
			case string:
				healthStatusFilter = []string{v}
			case []interface{}:
				for _, status := range v {
					if strStatus, ok := status.(string); ok {
						healthStatusFilter = append(healthStatusFilter, strStatus)
					}
				}
			case []string:
				healthStatusFilter = v
			}
		}

		// Validate resource ID
		if err := resourcehealth.ValidateResourceIDs([]string{resourceID}); err != nil {
			return "", fmt.Errorf("invalid resource ID: %v", err)
		}

		// Parse time filters
		startTime, endTime, err := resourcehealth.ParseTimeFilter(startTimeStr, endTimeStr)
		if err != nil {
			return "", fmt.Errorf("invalid time filter: %v", err)
		}

		// Extract subscription ID for client management
		parts := strings.Split(resourceID, "/")
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid resource ID format")
		}
		subscriptionID := parts[2]

		// Get clients for the subscription to ensure subscription is accessible
		_, err = client.GetOrCreateClientsForSubscription(subscriptionID)
		if err != nil {
			return "", fmt.Errorf("failed to get Azure clients: %v", err)
		}

		// Create event manager
		eventManager, err := resourcehealth.NewEventManager(subscriptionID, client.GetCredential())
		if err != nil {
			return "", fmt.Errorf("failed to create event manager: %v", err)
		}

		// Get resource health events
		ctx := context.Background()
		result, err := eventManager.GetResourceHealthEvents(ctx, resourceID, startTime, endTime, healthStatusFilter)
		if err != nil {
			return "", fmt.Errorf("failed to get resource health events: %v", err)
		}

		return result, nil
	})
}

// =============================================================================
// Azure Advisor Handlers
// =============================================================================

// GetAzureAdvisorRecommendationsHandler returns a handler for the get_azure_advisor_recommendations command
func GetAzureAdvisorRecommendationsHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract subscription ID
		subscriptionID, ok := params["subscription_id"].(string)
		if !ok || subscriptionID == "" {
			return "", fmt.Errorf("missing or invalid subscription_id parameter")
		}

		// Extract optional resource group
		resourceGroup, _ := params["resource_group"].(string)

		// Extract optional category filter
		var categories []string
		if categoriesParam, ok := params["category"]; ok {
			switch v := categoriesParam.(type) {
			case string:
				categories = []string{v}
			case []interface{}:
				for _, cat := range v {
					if strCat, ok := cat.(string); ok {
						categories = append(categories, strCat)
					}
				}
			case []string:
				categories = v
			}
		}

		// Extract optional severity filter
		var severities []string
		if severitiesParam, ok := params["severity"]; ok {
			switch v := severitiesParam.(type) {
			case string:
				severities = []string{v}
			case []interface{}:
				for _, sev := range v {
					if strSev, ok := sev.(string); ok {
						severities = append(severities, strSev)
					}
				}
			case []string:
				severities = v
			}
		}

		// Validate filter parameters
		if err := advisor.ValidateFilterParameters(categories, severities); err != nil {
			return "", fmt.Errorf("invalid filter parameters: %v", err)
		}

		// Get clients for the subscription to ensure subscription is accessible
		_, err := client.GetOrCreateClientsForSubscription(subscriptionID)
		if err != nil {
			return "", fmt.Errorf("failed to get Azure clients: %v", err)
		}

		// Create recommendation manager
		recommendationManager, err := advisor.NewRecommendationManager(subscriptionID, client.GetCredential())
		if err != nil {
			return "", fmt.Errorf("failed to create recommendation manager: %v", err)
		}

		// Get Azure Advisor recommendations
		ctx := context.Background()
		result, err := recommendationManager.GetRecommendations(ctx, subscriptionID, resourceGroup, categories, severities)
		if err != nil {
			return "", fmt.Errorf("failed to get Azure Advisor recommendations: %v", err)
		}

		return result, nil
	})
}

// GetAdvisorRecommendationDetailsHandler returns a handler for the get_advisor_recommendation_details command
func GetAdvisorRecommendationDetailsHandler(client *azure.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract recommendation ID
		recommendationID, ok := params["recommendation_id"].(string)
		if !ok || recommendationID == "" {
			return "", fmt.Errorf("missing or invalid recommendation_id parameter")
		}

		// Extract optional include implementation status
		includeImplementationStatus, _ := params["include_implementation_status"].(bool)

		// Validate recommendation ID
		if err := advisor.ValidateRecommendationID(recommendationID); err != nil {
			return "", fmt.Errorf("invalid recommendation ID: %v", err)
		}

		// Extract subscription ID from recommendation ID for client management
		parts := strings.Split(recommendationID, "/")
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid recommendation ID format")
		}
		subscriptionID := parts[2]

		// Get clients for the subscription to ensure subscription is accessible
		_, err := client.GetOrCreateClientsForSubscription(subscriptionID)
		if err != nil {
			return "", fmt.Errorf("failed to get Azure clients: %v", err)
		}

		// Create recommendation manager
		recommendationManager, err := advisor.NewRecommendationManager(subscriptionID, client.GetCredential())
		if err != nil {
			return "", fmt.Errorf("failed to create recommendation manager: %v", err)
		}

		// Get recommendation details
		ctx := context.Background()
		result, err := recommendationManager.GetRecommendationDetails(ctx, recommendationID, includeImplementationStatus)
		if err != nil {
			return "", fmt.Errorf("failed to get recommendation details: %v", err)
		}

		return result, nil
	})
}
