package compute

import (
	"github.com/Azure/aks-mcp/internal/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

// ComputeCommand defines a specific az vmss command to be registered as a tool
type ComputeCommand struct {
	Name        string
	Description string
	ArgsExample string // Example of command arguments
}

// RegisterAzComputeCommand registers a specific az vmss command as an MCP tool
func RegisterAzComputeCommand(cmd ComputeCommand) mcp.Tool {
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

// GetReadOnlyVmssCommands returns all read-only az vmss commands
func GetReadOnlyVmssCommands() []ComputeCommand {
	return []ComputeCommand{
		// No read-only commands for now
	}
}

// GetReadWriteVmssCommands returns all read-write az vmss commands
func GetReadWriteVmssCommands() []ComputeCommand {
	return []ComputeCommand{
		// Run command execution
		{Name: "az vmss run-command invoke", Description: "Execute a command on instances of a Virtual Machine Scale Set", ArgsExample: "--name myVMSS --resource-group myResourceGroup --command-id RunShellScript --scripts 'echo Hello World' --instance-id 0"},
	}
}

// GetAdminVmssCommands returns all admin az vmss commands
func GetAdminVmssCommands() []ComputeCommand {
	return []ComputeCommand{
		// No admin commands for now
	}
}
