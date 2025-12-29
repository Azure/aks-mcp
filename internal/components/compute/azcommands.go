package compute

import (
	"fmt"
	"slices"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// ComputeOperationType defines the type of compute operation
type ComputeOperationType string

// ResourceType defines the compute resource type
type ResourceType string

const (
	// VM operations - safe operations only
	OpVMShow            ComputeOperationType = "show"
	OpVMList            ComputeOperationType = "list"
	OpVMStart           ComputeOperationType = "start"
	OpVMStop            ComputeOperationType = "stop"
	OpVMRestart         ComputeOperationType = "restart"
	OpVMGetInstanceView ComputeOperationType = "get-instance-view"

	// VMSS operations - only safe operations for AKS-managed VMSS
	OpVMSSShow            ComputeOperationType = "show"
	OpVMSSList            ComputeOperationType = "list"
	OpVMSSRestart         ComputeOperationType = "restart"
	OpVMSSReimage         ComputeOperationType = "reimage"
	OpVMSSGetInstanceView ComputeOperationType = "get-instance-view"

	// Resource types
	ResourceTypeVM   ResourceType = "vm"
	ResourceTypeVMSS ResourceType = "vmss"
)

// generateToolDescription creates a tool description based on access level
func generateToolDescription(accessLevel string) string {
	baseDesc := `Unified tool for managing Azure Virtual Machines (VMs) and Virtual Machine Scale Sets (VMSS) using Azure CLI.

IMPORTANT: VM/VMSS resources are managed by AKS. Write operations should be used carefully and only for debugging purposes.

Use resource_type="vm" for single virtual machines or resource_type="vmss" for virtual machine scale sets.

Available operation values:`

	// Add operations by access level
	desc := baseDesc + "\n"

	// Basic operations for all access levels
	desc += "- show: Get details of a VM/VMSS\n"
	desc += "- list: List VMs/VMSS in subscription or resource group\n"
	desc += "- get-instance-view: Get runtime status\n"

	// Management operations for readwrite/admin
	if accessLevel == "readwrite" || accessLevel == "admin" {
		desc += "- start: Start VM\n"
		desc += "- stop: Stop VM\n"
		desc += "- restart: Restart VM/VMSS instances\n"
		desc += "- reimage: Reimage VMSS instances (VM not supported for reimage)\n"
	}

	// Note: All destructive operations (create, delete, deallocate, update, resize, scale)
	// have been removed for AKS environment safety

	// Examples
	desc += "\nEXAMPLES:\n"
	desc += `List VMSS: operation="list", resource_type="vmss", args="--resource-group myRG"` + "\n"
	desc += `Show VMSS: operation="show", resource_type="vmss", args="--name myVMSS --resource-group myRG"` + "\n"
	desc += `List VMs: operation="list", resource_type="vm", args="--resource-group myRG"` + "\n"

	if accessLevel == "readwrite" || accessLevel == "admin" {
		desc += `Restart VMSS: operation="restart", resource_type="vmss", args="--name myVMSS --resource-group myRG"` + "\n"
		desc += `Reimage VMSS: operation="reimage", resource_type="vmss", args="--name myVMSS --resource-group myRG"` + "\n"
	}

	return desc
}

// RegisterAzComputeOperations registers the unified compute operations tool
func RegisterAzComputeOperations(cfg *config.ConfigData) mcp.Tool {
	description := generateToolDescription(cfg.AccessLevel)

	return mcp.NewTool("az_compute_operations",
		mcp.WithDescription(description),
		mcp.WithTitleAnnotation("Azure Compute Operations"),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("Operation to perform. Common operations: list, show, start, stop, restart, deallocate, scale, etc."),
		),
		mcp.WithString("resource_type",
			mcp.Required(),
			mcp.Description("Resource type: 'vm' (single virtual machine) or 'vmss' (virtual machine scale set)"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Azure CLI arguments: '--resource-group myRG' (required for most operations), '--name myVM' (for specific resources), '--new-capacity 3' (for scaling)"),
		),
	)
}

// GetOperationAccessLevel returns the required access level for an operation
func GetOperationAccessLevel(operation string) string {
	readOnlyOps := []string{
		string(OpVMShow), string(OpVMList), string(OpVMGetInstanceView),
		string(OpVMSSShow), string(OpVMSSList), string(OpVMSSGetInstanceView),
	}

	readWriteOps := []string{
		// VM operations - safe operations only
		string(OpVMStart), string(OpVMStop), string(OpVMRestart),
		// VMSS operations - only safe operations for AKS-managed VMSS
		string(OpVMSSRestart), string(OpVMSSReimage),
	}

	// No admin operations - all unsafe operations removed
	adminOps := []string{}

	if slices.Contains(readOnlyOps, operation) {
		return "readonly"
	}

	if slices.Contains(readWriteOps, operation) {
		return "readwrite"
	}

	if slices.Contains(adminOps, operation) {
		return "admin"
	}

	return "unknown"
}

// ValidateOperationAccess checks if the operation is allowed for the given access level
func ValidateOperationAccess(operation string, cfg *config.ConfigData) error {
	requiredLevel := GetOperationAccessLevel(operation)

	switch requiredLevel {
	case "admin":
		if cfg.AccessLevel != "admin" {
			return fmt.Errorf("operation '%s' requires admin access level", operation)
		}
	case "readwrite":
		if cfg.AccessLevel != "readwrite" && cfg.AccessLevel != "admin" {
			return fmt.Errorf("operation '%s' requires readwrite or admin access level", operation)
		}
	case "readonly":
		// All access levels can perform readonly operations
	case "unknown":
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return nil
}

// MapOperationToCommand maps an operation and resource type to its corresponding az command
func MapOperationToCommand(operation string, resourceType string) (string, error) {
	// Validate resource type
	if resourceType != string(ResourceTypeVM) && resourceType != string(ResourceTypeVMSS) {
		return "", fmt.Errorf("invalid resource type: %s (must be 'vm' or 'vmss')", resourceType)
	}

	commandMap := map[string]map[string]string{
		string(ResourceTypeVM): {
			// Safe VM operations only
			string(OpVMShow):            "az vm show",
			string(OpVMList):            "az vm list",
			string(OpVMStart):           "az vm start",
			string(OpVMStop):            "az vm stop",
			string(OpVMRestart):         "az vm restart",
			string(OpVMGetInstanceView): "az vm get-instance-view",
		},
		string(ResourceTypeVMSS): {
			// Read-only operations
			string(OpVMSSShow):            "az vmss show",
			string(OpVMSSList):            "az vmss list",
			string(OpVMSSGetInstanceView): "az vmss get-instance-view",
			// Safe operations for AKS-managed VMSS
			string(OpVMSSRestart): "az vmss restart",
			string(OpVMSSReimage): "az vmss reimage",
			// Removed unsafe operations: create, delete, start, stop, deallocate, scale, update
		},
	}

	resourceCommands, exists := commandMap[resourceType]
	if !exists {
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	cmd, exists := resourceCommands[operation]
	if !exists {
		return "", fmt.Errorf("unsupported operation '%s' for resource type '%s'", operation, resourceType)
	}

	return cmd, nil
}
