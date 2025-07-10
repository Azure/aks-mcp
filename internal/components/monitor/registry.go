package monitor

import (
	"github.com/Azure/aks-mcp/internal/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

// MonitorCommand defines a specific az monitor command to be registered as a tool
type MonitorCommand struct {
	Name        string
	Description string
	ArgsExample string // Example of command arguments
	Category    string // Category for the command (e.g., "metrics", "logs")
}

// RegisterMonitorCommand registers a specific az monitor command as an MCP tool
func RegisterMonitorCommand(cmd MonitorCommand) mcp.Tool {
	// Convert spaces to underscores for valid tool name
	commandName := cmd.Name
	validToolName := utils.ReplaceSpacesWithUnderscores(commandName)

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

// GetReadOnlyMonitorCommands returns all read-only az monitor commands
func GetReadOnlyMonitorCommands() []MonitorCommand {
	return []MonitorCommand{
		// Metrics commands
		{
			Name:        "az monitor metrics list",
			Description: "List the metric values for a resource",
			ArgsExample: "--resource /subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/virtualMachines/{vmName} --metric \"Percentage CPU\"",
			Category:    "metrics",
		},
		{
			Name:        "az monitor metrics list-definitions",
			Description: "List the metric definitions for a resource",
			ArgsExample: "--resource /subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{clusterName}",
			Category:    "metrics",
		},
		{
			Name:        "az monitor metrics list-namespaces",
			Description: "List the metric namespaces for a resource",
			ArgsExample: "--resource /subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{clusterName}",
			Category:    "metrics",
		},
	}
}

// GetReadWriteMonitorCommands returns all read-write az monitor commands
func GetReadWriteMonitorCommands() []MonitorCommand {
	return []MonitorCommand{}
}

// GetAdminMonitorCommands returns all admin az monitor commands
func GetAdminMonitorCommands() []MonitorCommand {
	return []MonitorCommand{}
}
