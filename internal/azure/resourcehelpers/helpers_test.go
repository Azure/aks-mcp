package resourcehelpers

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
)

// TestGetVNetIDFromAKS tests the VNet ID extraction from an AKS cluster
func TestGetVNetIDFromAKS(t *testing.T) {
	ctx := context.Background()

	t.Run("valid cluster with VnetSubnetID", func(t *testing.T) {
		vnetSubnetID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/myVNet/subnets/mySubnet"

		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
					{
						VnetSubnetID: &vnetSubnetID,
					},
				},
			},
		}

		vnetID, err := GetVNetIDFromAKS(ctx, cluster, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expectedVNetID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/myVNet"
		if vnetID != expectedVNetID {
			t.Errorf("Expected VNet ID %s, got %s", expectedVNetID, vnetID)
		}
	})

	t.Run("cluster with nil properties", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: nil,
		}

		_, err := GetVNetIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil properties")
		}
	})

	t.Run("cluster with nil agent pool profiles", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: nil,
			},
		}

		_, err := GetVNetIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil agent pool profiles")
		}
	})

	t.Run("cluster with empty agent pool profiles", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{},
			},
		}

		_, err := GetVNetIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with empty agent pool profiles")
		}
	})

	t.Run("cluster with nil VnetSubnetID", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
					{
						VnetSubnetID: nil,
					},
				},
			},
		}

		_, err := GetVNetIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil VnetSubnetID")
		}
	})

	t.Run("cluster with invalid subnet ID format", func(t *testing.T) {
		invalidSubnetID := "invalid-subnet-id"

		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
					{
						VnetSubnetID: &invalidSubnetID,
					},
				},
			},
		}

		_, err := GetVNetIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for invalid subnet ID format")
		}
	})
}

// TestGetSubnetIDFromAKS tests the subnet ID extraction from an AKS cluster
func TestGetSubnetIDFromAKS(t *testing.T) {
	ctx := context.Background()

	t.Run("valid cluster with VnetSubnetID", func(t *testing.T) {
		vnetSubnetID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/myVNet/subnets/mySubnet"

		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
					{
						VnetSubnetID: &vnetSubnetID,
					},
				},
			},
		}

		subnetID, err := GetSubnetIDFromAKS(ctx, cluster, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if subnetID != vnetSubnetID {
			t.Errorf("Expected subnet ID %s, got %s", vnetSubnetID, subnetID)
		}
	})

	t.Run("cluster with nil properties", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: nil,
		}

		_, err := GetSubnetIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil properties")
		}
	})
}

// TestParseResourceID tests the resource ID parsing functionality
func TestParseResourceID(t *testing.T) {
	t.Run("valid VNet resource ID", func(t *testing.T) {
		resourceID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/myVNet"

		parsedID, err := arm.ParseResourceID(resourceID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if parsedID.SubscriptionID != "12345678-1234-1234-1234-123456789012" {
			t.Errorf("Expected subscription ID '12345678-1234-1234-1234-123456789012', got %s", parsedID.SubscriptionID)
		}

		if parsedID.ResourceGroupName != "myRG" {
			t.Errorf("Expected resource group 'myRG', got %s", parsedID.ResourceGroupName)
		}

		if parsedID.Name != "myVNet" {
			t.Errorf("Expected name 'myVNet', got %s", parsedID.Name)
		}

		if parsedID.ResourceType.String() != "Microsoft.Network/virtualNetworks" {
			t.Errorf("Expected resource type 'Microsoft.Network/virtualNetworks', got %s", parsedID.ResourceType.String())
		}
	})

	t.Run("valid subnet resource ID", func(t *testing.T) {
		resourceID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.Network/virtualNetworks/myVNet/subnets/mySubnet"

		parsedID, err := arm.ParseResourceID(resourceID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if parsedID.SubscriptionID != "12345678-1234-1234-1234-123456789012" {
			t.Errorf("Expected subscription ID '12345678-1234-1234-1234-123456789012', got %s", parsedID.SubscriptionID)
		}

		if parsedID.ResourceGroupName != "myRG" {
			t.Errorf("Expected resource group 'myRG', got %s", parsedID.ResourceGroupName)
		}

		if parsedID.Name != "mySubnet" {
			t.Errorf("Expected name 'mySubnet', got %s", parsedID.Name)
		}

		if parsedID.ResourceType.String() != "Microsoft.Network/virtualNetworks/subnets" {
			t.Errorf("Expected resource type 'Microsoft.Network/virtualNetworks/subnets', got %s", parsedID.ResourceType.String())
		}

		if parsedID.Parent == nil {
			t.Error("Expected parent to be set")
		} else {
			if parsedID.Parent.Name != "myVNet" {
				t.Errorf("Expected parent name 'myVNet', got %s", parsedID.Parent.Name)
			}
		}
	})

	t.Run("invalid resource ID", func(t *testing.T) {
		resourceID := "invalid-resource-id"

		_, err := arm.ParseResourceID(resourceID)
		if err == nil {
			t.Error("Expected error for invalid resource ID")
		}
	})

	t.Run("empty resource ID", func(t *testing.T) {
		resourceID := ""

		_, err := arm.ParseResourceID(resourceID)
		if err == nil {
			t.Error("Expected error for empty resource ID")
		}
	})
}

// TestGetRouteTableIDFromAKS tests the route table ID extraction from an AKS cluster
func TestGetRouteTableIDFromAKS(t *testing.T) {
	ctx := context.Background()

	t.Run("cluster with nil properties", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: nil,
		}

		_, err := GetRouteTableIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil properties")
		}
	})

	t.Run("cluster with nil agent pool profiles", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: nil,
			},
		}

		_, err := GetRouteTableIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil agent pool profiles")
		}
	})

	t.Run("cluster with empty agent pool profiles", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{},
			},
		}

		_, err := GetRouteTableIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with empty agent pool profiles")
		}
	})

	t.Run("cluster with nil VnetSubnetID", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
					{
						VnetSubnetID: nil,
					},
				},
			},
		}

		_, err := GetRouteTableIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil VnetSubnetID")
		}
	})

	t.Run("cluster with invalid subnet ID format", func(t *testing.T) {
		invalidSubnetID := "invalid-subnet-id"

		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
					{
						VnetSubnetID: &invalidSubnetID,
					},
				},
			},
		}

		_, err := GetRouteTableIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for invalid subnet ID format")
		}
	})
}

// TestGetNSGIDFromAKS tests the NSG ID extraction from an AKS cluster
func TestGetNSGIDFromAKS(t *testing.T) {
	ctx := context.Background()

	t.Run("cluster with nil properties", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: nil,
		}

		_, err := GetNSGIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil properties")
		}
	})

	t.Run("cluster with nil agent pool profiles", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: nil,
			},
		}

		_, err := GetNSGIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil agent pool profiles")
		}
	})

	t.Run("cluster with empty agent pool profiles", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{},
			},
		}

		_, err := GetNSGIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with empty agent pool profiles")
		}
	})

	t.Run("cluster with nil VnetSubnetID", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
					{
						VnetSubnetID: nil,
					},
				},
			},
		}

		_, err := GetNSGIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil VnetSubnetID")
		}
	})

	t.Run("cluster with invalid subnet ID format", func(t *testing.T) {
		invalidSubnetID := "invalid-subnet-id"

		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				AgentPoolProfiles: []*armcontainerservice.ManagedClusterAgentPoolProfile{
					{
						VnetSubnetID: &invalidSubnetID,
					},
				},
			},
		}

		_, err := GetNSGIDFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for invalid subnet ID format")
		}
	})
}

// TestGetLoadBalancerIDsFromAKS tests the load balancer IDs extraction from an AKS cluster
func TestGetLoadBalancerIDsFromAKS(t *testing.T) {
	ctx := context.Background()

	t.Run("cluster with nil properties", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: nil,
		}

		_, err := GetLoadBalancerIDsFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil properties")
		}
		if err.Error() != "invalid cluster or cluster properties" {
			t.Errorf("Expected 'invalid cluster or cluster properties' error, got %v", err)
		}
	})

	t.Run("cluster with nil node resource group", func(t *testing.T) {
		cluster := &armcontainerservice.ManagedCluster{
			Properties: &armcontainerservice.ManagedClusterProperties{
				NodeResourceGroup: nil,
			},
		}

		_, err := GetLoadBalancerIDsFromAKS(ctx, cluster, nil)
		if err == nil {
			t.Error("Expected error for cluster with nil node resource group")
		}
		if err.Error() != "node resource group not found for AKS cluster" {
			t.Errorf("Expected 'node resource group not found for AKS cluster' error, got %v", err)
		}
	})

	t.Run("nil cluster", func(t *testing.T) {
		_, err := GetLoadBalancerIDsFromAKS(ctx, nil, nil)
		if err == nil {
			t.Error("Expected error for nil cluster")
		}
		if err.Error() != "invalid cluster or cluster properties" {
			t.Errorf("Expected 'invalid cluster or cluster properties' error, got %v", err)
		}
	})
	t.Run("valid cluster with node resource group", func(t *testing.T) {
		// Since we can't easily mock the Azure client in this unit test,
		// and passing nil client causes a panic, we'll skip this test case
		// The actual Azure client interaction would be tested in integration tests
		t.Skip("Skipping test that requires Azure client - would be covered by integration tests")
	})
}
