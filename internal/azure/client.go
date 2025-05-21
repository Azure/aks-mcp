// Package azure provides Azure SDK integration for AKS MCP server.
package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
)

// AzureClient represents an Azure API client.
type AzureClient struct {
	containerServiceClient *armcontainerservice.ManagedClustersClient
	vnetClient             *armnetwork.VirtualNetworksClient
	routeTableClient       *armnetwork.RouteTablesClient
	nsgClient              *armnetwork.SecurityGroupsClient
}

// NewAzureClient creates a new Azure client using default credentials.
func NewAzureClient() (*AzureClient, error) {
	// Create a credential using DefaultAzureCredential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %v", err)
	}

	// Create the Azure clients
	containerServiceClient, err := armcontainerservice.NewManagedClustersClient("", cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create container service client: %v", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient("", cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual network client: %v", err)
	}

	routeTableClient, err := armnetwork.NewRouteTablesClient("", cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create route table client: %v", err)
	}

	nsgClient, err := armnetwork.NewSecurityGroupsClient("", cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create network security group client: %v", err)
	}

	return &AzureClient{
		containerServiceClient: containerServiceClient,
		vnetClient:             vnetClient,
		routeTableClient:       routeTableClient,
		nsgClient:              nsgClient,
	}, nil
}

// GetAKSCluster retrieves information about the specified AKS cluster.
func (c *AzureClient) GetAKSCluster(ctx context.Context, subscriptionID, resourceGroup, clusterName string) (*armcontainerservice.ManagedCluster, error) {
	resp, err := c.containerServiceClient.Get(ctx, resourceGroup, clusterName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get AKS cluster: %v", err)
	}
	return &resp.ManagedCluster, nil
}

// GetVirtualNetwork retrieves information about the specified virtual network.
func (c *AzureClient) GetVirtualNetwork(ctx context.Context, subscriptionID, resourceGroup, vnetName string) (*armnetwork.VirtualNetwork, error) {
	resp, err := c.vnetClient.Get(ctx, resourceGroup, vnetName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual network: %v", err)
	}
	return &resp.VirtualNetwork, nil
}

// GetRouteTable retrieves information about the specified route table.
func (c *AzureClient) GetRouteTable(ctx context.Context, subscriptionID, resourceGroup, routeTableName string) (*armnetwork.RouteTable, error) {
	resp, err := c.routeTableClient.Get(ctx, resourceGroup, routeTableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get route table: %v", err)
	}
	return &resp.RouteTable, nil
}

// GetNetworkSecurityGroup retrieves information about the specified network security group.
func (c *AzureClient) GetNetworkSecurityGroup(ctx context.Context, subscriptionID, resourceGroup, nsgName string) (*armnetwork.SecurityGroup, error) {
	resp, err := c.nsgClient.Get(ctx, resourceGroup, nsgName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get network security group: %v", err)
	}
	return &resp.SecurityGroup, nil
}
