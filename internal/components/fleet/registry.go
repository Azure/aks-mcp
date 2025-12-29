package fleet

import (
	"github.com/Azure/aks-mcp/internal/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

// FleetCommand defines a specific az fleet command to be registered as a tool
type FleetCommand struct {
	Name        string
	Description string
	ArgsExample string // Example of command arguments
}

// RegisterFleet registers the generic az fleet tool with structured parameters
func RegisterFleet() mcp.Tool {
	description := `Run Azure Kubernetes Service Fleet management commands.

Available operations and resources:
- fleet: list, show, create, update, delete, get-credentials
- member: list, show, create, update, delete  
- updaterun: list, show, create, start, stop, delete
- updatestrategy: list, show, create, delete
- clusterresourceplacement: list, show, get, create, delete (Kubernetes CRD operations)

Examples:
- List fleets: operation='list', resource='fleet', args='--resource-group myRG'
- Show fleet: operation='show', resource='fleet', args='--name myFleet --resource-group myRG'  
- Get fleet credentials: operation='get-credentials', resource='fleet', args='--name myFleet --resource-group myRG'
- Create member: operation='create', resource='member', args='--name myMember --fleet-name myFleet --resource-group myRG --member-cluster-id /subscriptions/.../myCluster'
- Create clusterresourceplacement: operation='create', resource='clusterresourceplacement', args='--name nginx --selector app=nginx --policy PickAll'
- List clusterresourceplacements: operation='list', resource='clusterresourceplacement', args=''`

	return mcp.NewTool("az_fleet",
		mcp.WithDescription(description),
		mcp.WithTitleAnnotation("Azure Fleet"),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform. Valid values: list, show, create, update, delete, start, stop, get-credentials"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type to operate on. Valid values: fleet, member, updaterun, updatestrategy, clusterresourceplacement"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Additional arguments for the command (e.g., '--name myFleet --resource-group myRG')"),
		),
	)
}

// RegisterFleetCommand registers a specific az fleet command as an MCP tool
func RegisterFleetCommand(cmd FleetCommand) mcp.Tool {
	// Convert spaces to underscores for valid tool name
	commandName := cmd.Name
	validToolName := utils.ReplaceSpacesWithUnderscores(commandName)

	description := "Run " + cmd.Name + " command: " + cmd.Description + "."

	// Add example if available
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

// GetReadOnlyFleetCommands returns all read-only fleet commands
func GetReadOnlyFleetCommands() []FleetCommand {
	return []FleetCommand{
		// Fleet commands
		{Name: "az fleet list", Description: "List all fleets", ArgsExample: "--resource-group myResourceGroup"},
		{Name: "az fleet show", Description: "Show details of a specific fleet", ArgsExample: "--name myFleet --resource-group myResourceGroup"},

		// Fleet member commands
		{Name: "az fleet member list", Description: "List all members of a fleet", ArgsExample: "--fleet-name myFleet --resource-group myResourceGroup"},
		{Name: "az fleet member show", Description: "Show details of a specific fleet member", ArgsExample: "--name myMember --fleet-name myFleet --resource-group myResourceGroup"},

		// Update run commands
		{Name: "az fleet updaterun list", Description: "List all update runs for a fleet", ArgsExample: "--fleet-name myFleet --resource-group myResourceGroup"},
		{Name: "az fleet updaterun show", Description: "Show details of a specific update run", ArgsExample: "--name myUpdateRun --fleet-name myFleet --resource-group myResourceGroup"},

		// Update strategy commands
		{Name: "az fleet updatestrategy list", Description: "List all update strategies for a fleet", ArgsExample: "--fleet-name myFleet --resource-group myResourceGroup"},
		{Name: "az fleet updatestrategy show", Description: "Show details of a specific update strategy", ArgsExample: "--name myStrategy --fleet-name myFleet --resource-group myResourceGroup"},
	}
}

// GetReadWriteFleetCommands returns all read-write fleet commands
func GetReadWriteFleetCommands() []FleetCommand {
	return []FleetCommand{
		// Fleet management
		{Name: "az fleet create", Description: "Create a new fleet", ArgsExample: "--name myFleet --resource-group myResourceGroup --location eastus"},
		{Name: "az fleet update", Description: "Update a fleet", ArgsExample: "--name myFleet --resource-group myResourceGroup --tags environment=production"},
		{Name: "az fleet delete", Description: "Delete a fleet", ArgsExample: "--name myFleet --resource-group myResourceGroup --yes"},

		// Fleet member management
		{Name: "az fleet member create", Description: "Add a member to a fleet", ArgsExample: "--name myMember --fleet-name myFleet --resource-group myResourceGroup --member-cluster-id /subscriptions/.../managedClusters/myCluster"},
		{Name: "az fleet member update", Description: "Update a fleet member", ArgsExample: "--name myMember --fleet-name myFleet --resource-group myResourceGroup --update-group staging"},
		{Name: "az fleet member delete", Description: "Remove a member from a fleet", ArgsExample: "--name myMember --fleet-name myFleet --resource-group myResourceGroup --yes"},

		// Update run management
		{Name: "az fleet updaterun create", Description: "Create a new update run", ArgsExample: "--name myUpdateRun --fleet-name myFleet --resource-group myResourceGroup --upgrade-type Full --kubernetes-version 1.28.0"},
		{Name: "az fleet updaterun start", Description: "Start an update run", ArgsExample: "--name myUpdateRun --fleet-name myFleet --resource-group myResourceGroup"},
		{Name: "az fleet updaterun stop", Description: "Stop an update run", ArgsExample: "--name myUpdateRun --fleet-name myFleet --resource-group myResourceGroup"},
		{Name: "az fleet updaterun delete", Description: "Delete an update run", ArgsExample: "--name myUpdateRun --fleet-name myFleet --resource-group myResourceGroup --yes"},

		// Update strategy management
		{Name: "az fleet updatestrategy create", Description: "Create a new update strategy", ArgsExample: "--name myStrategy --fleet-name myFleet --resource-group myResourceGroup --stages stage1"},
		{Name: "az fleet updatestrategy delete", Description: "Delete an update strategy", ArgsExample: "--name myStrategy --fleet-name myFleet --resource-group myResourceGroup --yes"},
	}
}

// GetAdminFleetCommands returns all admin fleet commands
func GetAdminFleetCommands() []FleetCommand {
	return []FleetCommand{
		// Currently no admin-only fleet commands defined
		// Admin users get all readwrite commands by default
	}
}
