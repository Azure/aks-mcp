package az

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// AksCommand defines a specific az aks command to be registered as a tool
type AksCommand struct {
	Name        string
	Description string
	ArgsExample string // Example of command arguments
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
	commandName := cmd.Name
	description := "Run " + cmd.Name + " command: " + cmd.Description + "."

	// Add example if available, with proper punctuation
	if cmd.ArgsExample != "" {
		description += "\nExample: `" + cmd.ArgsExample + "`"
	}

	return mcp.NewTool(commandName,
		mcp.WithDescription(description),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Arguments for the `"+cmd.Name+"` command"),
		),
	)
}

// GetReadOnlyAzCommands returns all read-only az commands
func GetReadOnlyAzCommands() []AksCommand {
	return []AksCommand{
		{Name: "az aks show", Description: "Show the details of a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		{Name: "az aks list", Description: "List managed Kubernetes clusters", ArgsExample: "--resource-group myResourceGroup"},
		{Name: "az aks get-versions", Description: "Get the versions available for creating a managed Kubernetes cluster", ArgsExample: "--location eastus"},
		{Name: "az aks browse", Description: "Show the dashboard for a Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		{Name: "az aks nodepool list", Description: "List node pools in a managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup"},
		{Name: "az aks nodepool show", Description: "Show the details for a node pool in the managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1"},
	}
}

// GetReadWriteAzCommands returns all read-write az commands
func GetReadWriteAzCommands() []AksCommand {
	return []AksCommand{
		{Name: "az aks create", Description: "Create a new managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --node-count 1 --enable-addons monitoring --generate-ssh-keys"},
		{Name: "az aks delete", Description: "Delete a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --yes"},
		{Name: "az aks scale", Description: "Scale the node pool in a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --node-count 3"},
		{Name: "az aks update", Description: "Update a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --enable-cluster-autoscaler --min-count 1 --max-count 3"},
		{Name: "az aks upgrade", Description: "Upgrade a managed Kubernetes cluster to a newer version", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --kubernetes-version 1.28.0"},
		{Name: "az aks nodepool add", Description: "Add a node pool to the managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool2 --node-count 3"},
		{Name: "az aks nodepool delete", Description: "Delete a node pool from the managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool2"},
		{Name: "az aks nodepool scale", Description: "Scale a node pool in a managed Kubernetes cluster", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1 --node-count 3"},
		{Name: "az aks nodepool upgrade", Description: "Upgrade a node pool to a newer version", ArgsExample: "--cluster-name myAKSCluster --resource-group myResourceGroup --name nodepool1 --kubernetes-version 1.28.0"},
	}
}

// GetAdminAzCommands returns all admin az commands
func GetAdminAzCommands() []AksCommand {
	return []AksCommand{
		{Name: "az aks rotate-certs", Description: "Rotate certificates and keys on a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
		{Name: "az aks enable-addons", Description: "Enable Kubernetes addons", ArgsExample: "--addons monitoring --name myAKSCluster --resource-group myResourceGroup"},
		{Name: "az aks disable-addons", Description: "Disable Kubernetes addons", ArgsExample: "--addons monitoring --name myAKSCluster --resource-group myResourceGroup"},
		{Name: "az aks update-credentials", Description: "Update credentials for a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup --reset-service-principal --service-principal CLIENT_ID --client-secret CLIENT_SECRET"},
		{Name: "az aks get-credentials", Description: "Get access credentials for a managed Kubernetes cluster", ArgsExample: "--name myAKSCluster --resource-group myResourceGroup"},
	}
}
