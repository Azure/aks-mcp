package az

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// AksCommand defines a specific az aks command to be registered as a tool
type AksCommand struct {
	Name        string
	Description string
	ArgsExample string // Example of command arguments
}

// replaceSpacesWithUnderscores converts spaces to underscores
// to create a valid tool name that follows the [a-z0-9_-] pattern
func replaceSpacesWithUnderscores(s string) string {
	return strings.ReplaceAll(s, " ", "_")
}

// // RegisterAz registers the generic az tool
// func RegisterAz() mcp.Tool {
// 	return mcp.NewTool("Run-az-command",
// 		mcp.WithDescription("Run az command and get result"),
// 		mcp.WithString("command",
// 			mcp.Required(),
// 			mcp.Description("The az command to execute"),
// 		),
// 	)
// }

// RegisterAzCommand registers a specific az command as an MCP tool
func RegisterAzCommand(cmd AksCommand) mcp.Tool {
	// Convert spaces to underscores for valid tool name
	commandName := cmd.Name
	validToolName := replaceSpacesWithUnderscores(commandName)

	description := "Run " + cmd.Name + " command: " + cmd.Description + "."

	// Add example if available, with proper punctuation
	if cmd.ArgsExample != "" {
		description += "\nExample: `" + cmd.ArgsExample + "`"
	}

	return mcp.NewTool(validToolName,
		mcp.WithDescription(description),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Arguments for the `"+cmd.Name+"` command"),
		),
	)
}

// Agents have limit on the number of tools they can register
// so we need to be selective about which commands we register.
// We comment out the commands that are not yet agreed upon,
// once we have a final list, we can uncomment them

// GetReadOnlyAzCommands returns all read-only az commands
func GetReadOnlyAzCommands() []AksCommand {
	return []AksCommand{
		// Cluster information commands
		{Name: "az aks show", Description: "Show the details of a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		{Name: "az aks list", Description: "List managed Kubernetes clusters", ArgsExample: "--resource-group myResourceGroup"},
		{Name: "az aks get-versions", Description: "Get the versions available for creating a managed Kubernetes cluster", ArgsExample: "--location eastus"},
		// {Name: "az aks get-upgrades", Description: "Get the upgrade versions available for a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks check-acr", Description: "Validate an ACR is accessible from an AKS cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --acr myAcrName"},
		{Name: "az aks check-network outbound", Description: "Perform outbound network connectivity check for a node in a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},

		// Addon information commands
		// {Name: "az aks addon list", Description: "List addons and their conditions in a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks addon show", Description: "Show details of an addon in a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --addon monitoring"},

		// Nodepool information commands
		{Name: "az aks nodepool list", Description: "List node pools in a managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup"},
		{Name: "az aks nodepool show", Description: "Show the details for a node pool in the managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1"},
		// {Name: "az aks nodepool get-upgrades", Description: "Get the available upgrade versions for an agent pool of the managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1"},

		// Operations and snapshot commands
		// {Name: "az aks operation", Description: "Show operation details on a managed Kubernetes cluster. Use 'show' with --operation-id for a specific operation, or 'show-latest' for the most recent operation", ArgsExample: "show --name myAKSCluster --resource-group myResourceGroup --operation-id 00000000-0000-0000-0000-000000000000 or show-latest --name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks snapshot list", Description: "List cluster snapshots", ArgsExample: "--resource-group myResourceGroup"},
		// {Name: "az aks snapshot show", Description: "Show the details of a cluster snapshot", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},

		// Trusted access read-only commands
		// {Name: "az aks trustedaccess rolebinding list", Description: "List all the trusted access role bindings", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks trustedaccess rolebinding show", Description: "Get the specific trusted access role binding according to binding name", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name myRoleBinding"},

		// Other read-only commands
		// {Name: "az aks install-cli", Description: "Download and install kubectl, the Kubernetes command-line tool", ArgsExample: ""},
	}
}

// GetReadWriteAzCommands returns all read-write az commands
func GetReadWriteAzCommands() []AksCommand {
	return []AksCommand{
		// Cluster management commands
		{Name: "az aks create", Description: "Create a new managed Kubernetes cluster, use --help if you are not clear about the arguments.", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --node-count 1 --enable-addons monitoring --generate-ssh-keys"},
		{Name: "az aks delete", Description: "Delete a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --yes"},
		{Name: "az aks scale", Description: "Scale the node pool in a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --node-count 3"},
		{Name: "az aks update", Description: "Update a managed Kubernetes cluster, use --help if you are not clear about the arguments.", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --enable-cluster-autoscaler --min-count 1 --max-count 3"},
		{Name: "az aks upgrade", Description: "Upgrade a managed Kubernetes cluster to a newer version", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --kubernetes-version 1.28.0"},
		// {Name: "az aks start", Description: "Starts a previously stopped Managed Cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks stop", Description: "Stop a managed cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks operation-abort", Description: "Abort last running operation on managed cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks rotate-certs", Description: "Rotate certificates and keys on a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},

		// Nodepool management commands
		{Name: "az aks nodepool add", Description: "Add a node pool to the managed Kubernetes cluster, use --help if you are not clear about the arguments.", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool2 --node-count 3"},
		{Name: "az aks nodepool delete", Description: "Delete a node pool from the managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool2"},
		{Name: "az aks nodepool scale", Description: "Scale a node pool in a managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1 --node-count 3"},
		{Name: "az aks nodepool upgrade", Description: "Upgrade a node pool to a newer version", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1 --kubernetes-version 1.28.0"},
		// {Name: "az aks nodepool update", Description: "Update a node pool properties", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1 --enable-cluster-autoscaler"},
		// {Name: "az aks nodepool start", Description: "Start stopped agent pool in the managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1"},
		// {Name: "az aks nodepool stop", Description: "Stop running agent pool in the managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1"},
		// {Name: "az aks nodepool operation-abort", Description: "Abort last running operation on nodepool", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1"},
		// {Name: "az aks nodepool delete-machines", Description: "Delete specific machines in an agentpool for a managed cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1 --machine-names machine1"},

		// Addon management
		// {Name: "az aks enable-addons", Description: "Enable Kubernetes addons", ArgsExample: "--addons monitoring --name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks disable-addons", Description: "Disable Kubernetes addons", ArgsExample: "--addons monitoring --name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks approuting enable", Description: "Enable App Routing addon for a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks approuting disable", Description: "Disable App Routing addon for a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},

		// Snapshot commands
		// {Name: "az aks snapshot create", Description: "Create a snapshot of a cluster", ArgsExample: "-g MyResourceGroup -n snapshot1 --cluster-id \"/subscriptions/00000/resourceGroups/AnotherResourceGroup/providers/Microsoft.ContainerService/managedClusters/akscluster1\""},
		// {Name: "az aks snapshot delete", Description: "Delete a cluster snapshot", ArgsExample: "--name myAKSSnapshot --resource-group myResourceGroup"},
		// {Name: "az aks nodepool snapshot create", Description: "Create a nodepool snapshot", ArgsExample: "-g MyResourceGroup -n snapshot1 --nodepool-id \"/subscriptions/00000/resourceGroups/AnotherResourceGroup/providers/Microsoft.ContainerService/managedClusters/akscluster1/agentPools/nodepool1\""},
		// {Name: "az aks nodepool snapshot delete", Description: "Delete a nodepool snapshot", ArgsExample: "--name myNodepoolSnapshot --resource-group myResourceGroup"},

		// Maintenance commands
		// {Name: "az aks maintenanceconfiguration add", Description: "Add a maintenance configuration in managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup -n default --weekday Monday --start-hour 1"},
		// {Name: "az aks maintenanceconfiguration delete", Description: "Delete a maintenance configuration in managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup -n default"},
		// {Name: "az aks maintenanceconfiguration update", Description: "Update a maintenance configuration of a managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup -n default --weekday Monday --start-hour 1"},

		// Command execution
		// {Name: "az aks command invoke", Description: "Invoke a command on a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --command \"kubectl get pods -n kube-system\""},
		// {Name: "az aks command result", Description: "Get the result of a previously invoked command", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --command-id 00000000-0000-0000-0000-000000000000"},

		// Security and advanced features
		// {Name: "az aks pod-identity add", Description: "Add a pod identity to a managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --namespace my-namespace --name my-identity --identity-resource-id /subscriptions/SUB_ID/resourcegroups/RG/providers/Microsoft.ManagedIdentity/userAssignedIdentities/ID"},
		// {Name: "az aks pod-identity delete", Description: "Remove a pod identity from a managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --namespace my-namespace --name my-identity"},
		// {Name: "az aks trustedaccess rolebinding create", Description: "Create a new trusted access role binding", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name myRoleBinding --source-resource-id /subscriptions/0000/resourceGroups/myResourceGroup/providers/Microsoft.Demo/samples --roles Microsoft.Demo/samples/reader,Microsoft.Demo/samples/writer"},
		// {Name: "az aks trustedaccess rolebinding delete", Description: "Delete a trusted access role binding according to name", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name myRoleBinding"},
		// {Name: "az aks trustedaccess rolebinding update", Description: "Update a trusted access role binding", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name myRoleBinding --roles Microsoft.Demo/samples/reader,Microsoft.Demo/samples/writer"},
		// {Name: "az aks oidc-issuer rotate-signing-keys", Description: "Rotate oidc issuer service account signing keys", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},

		// Service mesh
		// {Name: "az aks mesh enable", Description: "Enable Azure Service Mesh in a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks mesh disable", Description: "Disable Azure Service Mesh in a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
	}
}

// GetAccountAzCommands returns all Azure account management commands
func GetAccountAzCommands() []AksCommand {
	return []AksCommand{
		{Name: "az account list", Description: "List all subscriptions for the authenticated account", ArgsExample: "--output table"},
		{Name: "az login", Description: "Log in to Azure using service principal credentials", ArgsExample: "--service-principal --username APP_ID --password PASSWORD --tenant TENANT_ID"},
		{Name: "az account set", Description: "Set a subscription as the current active subscription", ArgsExample: "--subscription mySubscriptionNameOrId"},
	}
}

// GetAdminAzCommands returns all admin az commands
func GetAdminAzCommands() []AksCommand {
	return []AksCommand{
		// Credential management (admin only)
		{Name: "az aks get-credentials", Description: "Get access credentials for a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		// {Name: "az aks update-credentials", Description: "Update credentials for a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --reset-service-principal --service-principal CLIENT_ID --client-secret CLIENT_SECRET"},
	}
}

// GetKubectlCommands returns kubectl commands for browsing Kubernetes resources
func GetKubectlCommands() []AksCommand {
	return []AksCommand{
		// Resource listing and getting
		{Name: "kubectl get", Description: "List or get Kubernetes resources (pods, services, deployments, nodes, namespaces, etc.)", ArgsExample: "pods -n kube-system -o wide"},
		{Name: "kubectl describe", Description: "Show detailed information about specific Kubernetes resources", ArgsExample: "pod my-pod-name -n default"},
		
		// Logs and debugging
		{Name: "kubectl logs", Description: "Print container logs from pods", ArgsExample: "my-pod-name -n default --tail=100 --follow=false"},
		{Name: "kubectl top", Description: "Display resource usage (CPU/Memory) for nodes or pods", ArgsExample: "nodes"},
		
		// Cluster information
		{Name: "kubectl cluster-info", Description: "Display cluster information and component statuses", ArgsExample: ""},
		{Name: "kubectl api-resources", Description: "List all available API resources in the cluster", ArgsExample: "--namespaced=true"},
		{Name: "kubectl version", Description: "Display client and server Kubernetes version", ArgsExample: "--short"},
	}
}
