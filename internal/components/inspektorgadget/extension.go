package inspektorgadget

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/command"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
)

// Inspektor Gadget AKS cluster extension constants.
// See https://learn.microsoft.com/en-us/azure/aks/inspektor-gadget-configure
const (
	// extensionName is the name given to the Inspektor Gadget cluster extension instance.
	extensionName = "inspektor-gadget"
	// extensionType is the Azure extension type for Inspektor Gadget.
	extensionType = "microsoft.inspektorgadget"
	// extensionClusterType is the cluster type for AKS managed clusters.
	extensionClusterType = "managedClusters"
	// defaultReleaseTrain is the default release train for the extension.
	// The extension is currently in preview.
	defaultReleaseTrain = "preview"
)

// ClusterRef identifies an AKS cluster in the Azure resource management plane.
type ClusterRef struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
}

// ExtensionClient defines the minimal interface used by the Inspektor Gadget handlers to
// manage the lifecycle of the Inspektor Gadget cluster extension via `az k8s-extension`.
type ExtensionClient interface {
	// Install creates the Inspektor Gadget cluster extension on the given cluster.
	Install(cluster ClusterRef, releaseTrain, version string, autoUpgradeMinor bool) (string, error)
	// Delete removes the Inspektor Gadget cluster extension from the given cluster.
	Delete(cluster ClusterRef) (string, error)
	// Show returns the current state of the Inspektor Gadget cluster extension.
	Show(cluster ClusterRef) (string, error)
}

// azExtensionClient is an ExtensionClient backed by the Azure CLI (`az k8s-extension`).
//
// Commands are executed directly via a shell process (not the security-validated az
// executor) so that lifecycle operations remain gated by the handler-level access-level
// and confirmation logic, mirroring the previous Helm-based behavior.
type azExtensionClient struct {
	timeout int
}

// newExtensionClient creates a new az-backed ExtensionClient.
func newExtensionClient(timeout int) *azExtensionClient {
	return &azExtensionClient{timeout: timeout}
}

// clusterArgs returns the common `az k8s-extension` arguments identifying the cluster and
// the Inspektor Gadget extension instance.
func (c ClusterRef) clusterArgs() []string {
	return []string{
		"--cluster-type", extensionClusterType,
		"--cluster-name", c.ClusterName,
		"--resource-group", c.ResourceGroup,
		"--subscription", c.SubscriptionID,
		"--name", extensionName,
	}
}

func (e *azExtensionClient) run(args []string) (string, error) {
	process := command.NewShellProcess("az", e.timeout)
	return process.Run(strings.Join(args, " "))
}

// azCommandString renders the equivalent `az` CLI command for a set of arguments so it can
// be surfaced to the user. This lets AKS customers see exactly how the Inspektor Gadget
// cluster extension is managed, and lets them run the command manually if a lifecycle
// operation fails or times out on the server side.
func azCommandString(args []string) string {
	return "az " + strings.Join(args, " ")
}

// deployCommandString renders the `az k8s-extension create` command used to deploy the
// Inspektor Gadget extension, for surfacing to the user (e.g. in the readonly-mode
// confirmation prompt).
func deployCommandString(cluster ClusterRef) string {
	args := []string{"k8s-extension", "create"}
	args = append(args, cluster.clusterArgs()...)
	args = append(args, "--extension-type", extensionType, "--release-train", defaultReleaseTrain)
	return azCommandString(args)
}

// undeployCommandString renders the `az k8s-extension delete` command used to undeploy the
// Inspektor Gadget extension, for surfacing to the user.
func undeployCommandString(cluster ClusterRef) string {
	args := []string{"k8s-extension", "delete"}
	args = append(args, cluster.clusterArgs()...)
	args = append(args, "--yes")
	return azCommandString(args)
}

func (e *azExtensionClient) Install(cluster ClusterRef, releaseTrain, version string, autoUpgradeMinor bool) (string, error) {
	if releaseTrain == "" {
		releaseTrain = defaultReleaseTrain
	}

	args := []string{"k8s-extension", "create"}
	args = append(args, cluster.clusterArgs()...)
	args = append(args,
		"--extension-type", extensionType,
		"--release-train", releaseTrain,
	)
	if version != "" {
		args = append(args, "--version", version)
	}
	if !autoUpgradeMinor {
		args = append(args, "--auto-upgrade-minor-version", "false")
	}

	out, err := e.run(args)
	if err != nil {
		return "", fmt.Errorf("creating Inspektor Gadget extension: %w: %s.\nYou can retry manually with: %s", err, out, azCommandString(args))
	}
	return fmt.Sprintf("Inspektor Gadget extension %q created successfully on cluster %q (resource group %q).\nEquivalent CLI: %s", extensionName, cluster.ClusterName, cluster.ResourceGroup, azCommandString(args)), nil
}

func (e *azExtensionClient) Delete(cluster ClusterRef) (string, error) {
	args := []string{"k8s-extension", "delete"}
	args = append(args, cluster.clusterArgs()...)
	args = append(args, "--yes")

	out, err := e.run(args)
	if err != nil {
		return "", fmt.Errorf("deleting Inspektor Gadget extension: %w: %s.\nYou can retry manually with: %s", err, out, azCommandString(args))
	}
	return fmt.Sprintf("Inspektor Gadget extension %q deleted successfully from cluster %q (resource group %q).\nEquivalent CLI: %s", extensionName, cluster.ClusterName, cluster.ResourceGroup, azCommandString(args)), nil
}

func (e *azExtensionClient) Show(cluster ClusterRef) (string, error) {
	args := []string{"k8s-extension", "show"}
	args = append(args, cluster.clusterArgs()...)

	out, err := e.run(args)
	if err != nil {
		return "", fmt.Errorf("showing Inspektor Gadget extension: %w: %s", err, out)
	}
	return out, nil
}

// resolveClusterRef determines the AKS cluster identity for lifecycle operations.
//
// Resolution priority:
//  1. explicit subscription_id + resource_group + cluster_name params
//  2. aks_resource_id param (full Azure resource ID)
//  3. cfg.DefaultAKSResourceID (from --default-aks-resource-id / AZURE_AKS_RESOURCE_ID)
func resolveClusterRef(params map[string]interface{}, cfg *config.ConfigData) (ClusterRef, error) {
	sub, _ := params["subscription_id"].(string)
	rg, _ := params["resource_group"].(string)
	name, _ := params["cluster_name"].(string)
	if sub != "" && rg != "" && name != "" {
		return ClusterRef{SubscriptionID: sub, ResourceGroup: rg, ClusterName: name}, nil
	}
	if sub != "" || rg != "" || name != "" {
		return ClusterRef{}, fmt.Errorf("subscription_id, resource_group and cluster_name must all be provided together")
	}

	if resourceID, ok := params["aks_resource_id"].(string); ok && resourceID != "" {
		return parseClusterRef(resourceID)
	}

	if cfg != nil && cfg.DefaultAKSResourceID != "" {
		return parseClusterRef(cfg.DefaultAKSResourceID)
	}

	return ClusterRef{}, fmt.Errorf("unable to determine AKS cluster identity: provide 'aks_resource_id' (or 'subscription_id', 'resource_group' and 'cluster_name'), or configure --default-aks-resource-id / AZURE_AKS_RESOURCE_ID")
}

// parseClusterRef parses an AKS cluster resource ID into a ClusterRef.
func parseClusterRef(resourceID string) (ClusterRef, error) {
	parsed, err := arm.ParseResourceID(resourceID)
	if err != nil {
		return ClusterRef{}, fmt.Errorf("failed to parse AKS resource ID %q: %w", resourceID, err)
	}
	if parsed.SubscriptionID == "" || parsed.ResourceGroupName == "" || parsed.Name == "" {
		return ClusterRef{}, fmt.Errorf("AKS resource ID %q is missing subscription, resource group, or cluster name", resourceID)
	}
	return ClusterRef{
		SubscriptionID: parsed.SubscriptionID,
		ResourceGroup:  parsed.ResourceGroupName,
		ClusterName:    parsed.Name,
	}, nil
}

// extensionInstalled reports whether the Inspektor Gadget cluster extension exists on the
// cluster, based on `az k8s-extension show`. A "not found" error is treated as not
// installed rather than a hard failure.
func extensionInstalled(client ExtensionClient, cluster ClusterRef) (bool, error) {
	_, err := client.Show(cluster)
	if err == nil {
		return true, nil
	}
	if isExtensionNotFound(err) {
		return false, nil
	}
	return false, err
}

// isExtensionNotFound reports whether an error from `az k8s-extension show` indicates that
// the Inspektor Gadget extension specifically does not exist on the cluster.
//
// It deliberately matches only the "extension instance not found" signal. Other
// not-found errors (for example a wrong resource group -> ResourceGroupNotFound, or a
// wrong cluster) must surface as hard errors rather than being misread as "not installed",
// which would otherwise produce a spurious OSS-conflict message.
func isExtensionNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Extension absent: "(ResourceNotFound) Extension instance with name '...' not found."
	if strings.Contains(msg, "extension instance with name") && strings.Contains(msg, "not found") {
		return true
	}
	// Older/alternate wording used by the extension backend.
	return strings.Contains(msg, "extensionnotfound")
}
