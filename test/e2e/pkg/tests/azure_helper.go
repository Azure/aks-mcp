package tests

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
)

// VMSSInstance contains VMSS instance information
type VMSSInstance struct {
	VMSSName          string
	InstanceID        string
	NodeResourceGroup string
}

// GetFirstVMSSInstance retrieves the first VMSS instance from an AKS cluster
func GetFirstVMSSInstance(ctx context.Context, subscriptionID, resourceGroup, clusterName string) (*VMSSInstance, error) {
	// 1. Create Azure credential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	// 2. Get AKS cluster information
	aksClient, err := armcontainerservice.NewManagedClustersClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create AKS client: %w", err)
	}

	cluster, err := aksClient.Get(ctx, resourceGroup, clusterName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	if cluster.Properties == nil || cluster.Properties.NodeResourceGroup == nil {
		return nil, fmt.Errorf("node resource group not found")
	}

	nodeResourceGroup := *cluster.Properties.NodeResourceGroup

	// 3. List VMSS in the node resource group
	vmssClient, err := armcompute.NewVirtualMachineScaleSetsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VMSS client: %w", err)
	}

	pager := vmssClient.NewListPager(nodeResourceGroup, nil)
	if !pager.More() {
		return nil, fmt.Errorf("no VMSS found in resource group %s", nodeResourceGroup)
	}

	page, err := pager.NextPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMSS: %w", err)
	}

	if len(page.Value) == 0 {
		return nil, fmt.Errorf("no VMSS found in resource group %s", nodeResourceGroup)
	}

	// 4. Get the first VMSS
	firstVMSS := page.Value[0]
	if firstVMSS.Name == nil {
		return nil, fmt.Errorf("VMSS name is nil")
	}

	return &VMSSInstance{
		VMSSName:          *firstVMSS.Name,
		InstanceID:        "0", // AKS VMSS instance IDs start from 0
		NodeResourceGroup: nodeResourceGroup,
	}, nil
}
