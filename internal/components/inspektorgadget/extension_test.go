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

func TestAzCommandString(t *testing.T) {
	c := ClusterRef{SubscriptionID: "sub-123", ResourceGroup: "rg-test", ClusterName: "cluster-test"}
	args := append([]string{"k8s-extension", "delete"}, c.clusterArgs()...)
	args = append(args, "--yes")

	got := azCommandString(args)

	if !strings.HasPrefix(got, "az k8s-extension delete") {
		t.Errorf("expected command to start with 'az k8s-extension delete', got %q", got)
	}
	for _, want := range []string{
		"--cluster-type managedClusters",
		"--cluster-name cluster-test",
		"--resource-group rg-test",
		"--subscription sub-123",
		"--name inspektor-gadget",
		"--yes",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("command %q missing %q", got, want)
		}
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

type fakeExtensionClient struct {
	showOut string
	showErr error
}

func (f *fakeExtensionClient) Install(ClusterRef, string, string, bool) (string, error) {
	return "installed", nil
}
func (f *fakeExtensionClient) Delete(ClusterRef) (string, error) { return "deleted", nil }
func (f *fakeExtensionClient) Show(ClusterRef) (string, error)   { return f.showOut, f.showErr }

func TestExtensionInstalled(t *testing.T) {
	cluster := ClusterRef{SubscriptionID: "s", ResourceGroup: "rg", ClusterName: "c"}

	tests := []struct {
		name    string
		client  *fakeExtensionClient
		want    bool
		wantErr bool
	}{
		{
			name:   "installed",
			client: &fakeExtensionClient{showOut: `{"provisioningState":"Succeeded"}`},
			want:   true,
		},
		{
			name:   "not found",
			client: &fakeExtensionClient{showErr: errString("(ResourceNotFound) Extension instance with name 'inspektor-gadget' not found. Verify that the cluster-type is correct and the resource exists.")},
			want:   false,
		},
		{
			name:    "wrong resource group is a hard error",
			client:  &fakeExtensionClient{showErr: errString("(ResourceGroupNotFound) Resource group 'wrong-rg' could not be found.")},
			wantErr: true,
		},
		{
			name:    "transport error",
			client:  &fakeExtensionClient{showErr: errString("connection refused")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extensionInstalled(tt.client, cluster)
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
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }

type recordingExtensionClient struct {
	fakeExtensionClient
	installTrain   string
	installVersion string
	installAuto    bool
}

func (r *recordingExtensionClient) Install(_ ClusterRef, releaseTrain, version string, autoUpgradeMinor bool) (string, error) {
	r.installTrain = releaseTrain
	r.installVersion = version
	r.installAuto = autoUpgradeMinor
	return "installed", nil
}

func TestHandleDeployAction(t *testing.T) {
	cluster := ClusterRef{SubscriptionID: "s", ResourceGroup: "rg", ClusterName: "c"}

	t.Run("defaults", func(t *testing.T) {
		rec := &recordingExtensionClient{}
		if _, err := handleDeployAction(rec, cluster, map[string]interface{}{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.installTrain != "" || rec.installVersion != "" || !rec.installAuto {
			t.Fatalf("unexpected install args: train=%q version=%q auto=%v", rec.installTrain, rec.installVersion, rec.installAuto)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		rec := &recordingExtensionClient{}
		params := map[string]interface{}{
			"release_train":              "stable",
			"version":                    "1.2.3",
			"auto_upgrade_minor_version": false,
		}
		if _, err := handleDeployAction(rec, cluster, params); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.installTrain != "stable" || rec.installVersion != "1.2.3" || rec.installAuto {
			t.Fatalf("unexpected install args: train=%q version=%q auto=%v", rec.installTrain, rec.installVersion, rec.installAuto)
		}
	})
}
