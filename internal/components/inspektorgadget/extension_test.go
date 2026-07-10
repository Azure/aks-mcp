package inspektorgadget

import (
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

func TestResolveClusterRef(t *testing.T) {
	const resourceID = "/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/cluster-test"

	tests := []struct {
		name    string
		params  map[string]interface{}
		cfg     *config.ConfigData
		want    ClusterRef
		wantErr bool
	}{
		{
			name: "explicit params take precedence",
			params: map[string]interface{}{
				"subscription_id": "sub-a",
				"resource_group":  "rg-a",
				"cluster_name":    "cluster-a",
				"aks_resource_id": resourceID,
			},
			cfg:  &config.ConfigData{DefaultAKSResourceID: resourceID},
			want: ClusterRef{SubscriptionID: "sub-a", ResourceGroup: "rg-a", ClusterName: "cluster-a"},
		},
		{
			name:   "aks_resource_id param",
			params: map[string]interface{}{"aks_resource_id": resourceID},
			cfg:    &config.ConfigData{},
			want:   ClusterRef{SubscriptionID: "sub-123", ResourceGroup: "rg-test", ClusterName: "cluster-test"},
		},
		{
			name:   "config default",
			params: map[string]interface{}{},
			cfg:    &config.ConfigData{DefaultAKSResourceID: resourceID},
			want:   ClusterRef{SubscriptionID: "sub-123", ResourceGroup: "rg-test", ClusterName: "cluster-test"},
		},
		{
			name:    "partial explicit params error",
			params:  map[string]interface{}{"subscription_id": "sub-a", "resource_group": "rg-a"},
			cfg:     &config.ConfigData{},
			wantErr: true,
		},
		{
			name:    "no identity available",
			params:  map[string]interface{}{},
			cfg:     &config.ConfigData{},
			wantErr: true,
		},
		{
			name:    "invalid resource id",
			params:  map[string]interface{}{"aks_resource_id": "not-a-resource-id"},
			cfg:     &config.ConfigData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveClusterRef(tt.params, tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestClusterArgs(t *testing.T) {
	c := ClusterRef{SubscriptionID: "sub-123", ResourceGroup: "rg-test", ClusterName: "cluster-test"}
	args := strings.Join(c.clusterArgs(), " ")

	for _, want := range []string{
		"--cluster-type managedClusters",
		"--cluster-name cluster-test",
		"--resource-group rg-test",
		"--subscription sub-123",
		"--name inspektor-gadget",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("cluster args %q missing %q", args, want)
		}
	}
}
