package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/components/common"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/aks-mcp/internal/tools"
)

const (
	// Maximum number of log lines to prevent overwhelming output
	DefaultLogLines = 500
	MaxLogLines     = 2000

	// Log types
	LogTypeKubelet    = "kubelet"
	LogTypeContainerd = "containerd"
	LogTypeKernel     = "kernel"
	LogTypeSyslog     = "syslog"

	// Log levels
	LogLevelError = "ERROR"
	LogLevelWarn  = "WARN"
	LogLevelInfo  = "INFO"
)

// CollectAKSNodeLogsHandler returns a handler for the collect_aks_node_logs tool
func CollectAKSNodeLogsHandler(client *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Extract AKS resource parameters from aks_resource_id
		subID, rg, clusterName, err := common.ExtractAKSParametersFromResourceID(params)
		if err != nil {
			return "", err
		}

		// Get the cluster details to obtain node resource group
		cluster, err := common.GetClusterDetails(ctx, client, subID, rg, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get cluster details: %w", err)
		}

		if cluster.Properties == nil || cluster.Properties.NodeResourceGroup == nil {
			return "", fmt.Errorf("cluster node resource group not found")
		}

		nodeResourceGroup := *cluster.Properties.NodeResourceGroup

		// Extract VMSS parameters
		vmssName, ok := params["vmss_name"].(string)
		if !ok || vmssName == "" {
			return "", fmt.Errorf("vmss_name is required")
		}

		instanceID, ok := params["instance_id"].(string)
		if !ok || instanceID == "" {
			return "", fmt.Errorf("instance_id is required")
		}

		// Validate that the VMSS is Linux-based
		if err := validateVMSSSupport(ctx, client, subID, nodeResourceGroup, vmssName); err != nil {
			return "", err
		}

		// Extract log collection parameters
		logType, ok := params["log_type"].(string)
		if !ok || logType == "" {
			return "", fmt.Errorf("log_type is required")
		}

		// Validate log type
		if !isValidLogType(logType) {
			return "", fmt.Errorf("invalid log_type: %s (must be one of: kubelet, containerd, kernel, syslog)", logType)
		}

		// Extract lines parameter
		lines := DefaultLogLines
		if l, ok := params["lines"].(float64); ok {
			lines = int(l)
		}
		if lines > MaxLogLines {
			lines = MaxLogLines
		}
		if lines < 1 {
			lines = DefaultLogLines
		}

		// Extract since parameter (optional, takes precedence over lines)
		since := ""
		if s, ok := params["since"].(string); ok && s != "" {
			since = s
		}

		// Extract level parameter (optional)
		level := LogLevelInfo
		if l, ok := params["level"].(string); ok && l != "" {
			level = strings.ToUpper(l)
		}

		// Validate log level
		if !isValidLogLevel(level) {
			return "", fmt.Errorf("invalid level: %s (must be one of: ERROR, WARN, INFO)", level)
		}

		// Extract filter parameter (optional)
		filter := ""
		if f, ok := params["filter"].(string); ok && f != "" {
			filter = f
		}

		// Build the command
		command, err := buildLogCommand(logType, lines, since, level, filter)
		if err != nil {
			return "", fmt.Errorf("failed to build log command: %w", err)
		}

		logger.Debugf("CollectAKSNodeLogs: cluster=%s/%s, nodeRG=%s, vmss=%s, instance=%s, type=%s, lines=%d, since=%s, level=%s, filter=%s",
			rg, clusterName, nodeResourceGroup, vmssName, instanceID, logType, lines, since, level, filter)
		logger.Debugf("CollectAKSNodeLogs: command=%s", command)

		// Execute the command on VMSS instance (using node resource group)
		executor := NewVMRunCommandExecutor(client)
		output, err := executor.ExecuteOnVMSSInstance(ctx, subID, nodeResourceGroup, vmssName, instanceID, command)
		if err != nil {
			return "", fmt.Errorf("failed to collect %s logs from VMSS %s/%s instance %s: %w",
				logType, nodeResourceGroup, vmssName, instanceID, err)
		}

		// Format the output with metadata
		result := formatLogOutput(clusterName, nodeResourceGroup, vmssName, instanceID, logType, output)
		return result, nil
	})
}

// isValidLogType checks if the log type is valid
func isValidLogType(logType string) bool {
	switch logType {
	case LogTypeKubelet, LogTypeContainerd, LogTypeKernel, LogTypeSyslog:
		return true
	default:
		return false
	}
}

// isValidLogLevel checks if the log level is valid
func isValidLogLevel(level string) bool {
	switch level {
	case LogLevelError, LogLevelWarn, LogLevelInfo:
		return true
	default:
		return false
	}
}

// buildLogCommand builds the shell command to collect logs
func buildLogCommand(logType string, lines int, since string, level string, filter string) (string, error) {
	var cmd string

	switch logType {
	case LogTypeKubelet:
		cmd = buildJournalctlCommand("kubelet", lines, since, level, filter)
	case LogTypeContainerd:
		cmd = buildJournalctlCommand("containerd", lines, since, level, filter)
	case LogTypeKernel:
		cmd = buildDmesgCommand(lines, level, filter)
	case LogTypeSyslog:
		cmd = buildSyslogCommand(lines, since, level, filter)
	default:
		return "", fmt.Errorf("unsupported log type: %s", logType)
	}

	return cmd, nil
}

// formatSinceParameter converts shorthand time formats to journalctl-compatible format
// Examples: "1h" -> "1 hour ago", "30m" -> "30 minutes ago", "2d" -> "2 days ago"
func formatSinceParameter(since string) string {
	if since == "" {
		return since
	}

	// Check if it's already in a valid format (contains spaces or is a full timestamp)
	if strings.Contains(since, " ") || strings.Contains(since, "-") || strings.Contains(since, ":") {
		return since
	}

	// Convert shorthand format
	if len(since) < 2 {
		return since
	}

	unit := since[len(since)-1:]
	value := since[:len(since)-1]

	switch unit {
	case "h":
		if value == "1" {
			return "1 hour ago"
		}
		return value + " hours ago"
	case "m":
		if value == "1" {
			return "1 minute ago"
		}
		return value + " minutes ago"
	case "s":
		if value == "1" {
			return "1 second ago"
		}
		return value + " seconds ago"
	case "d":
		if value == "1" {
			return "1 day ago"
		}
		return value + " days ago"
	case "w":
		if value == "1" {
			return "1 week ago"
		}
		return value + " weeks ago"
	default:
		// If not a recognized shorthand, return as-is
		return since
	}
}

// buildJournalctlCommand builds journalctl command for systemd services
func buildJournalctlCommand(unit string, lines int, since string, level string, filter string) string {
	var parts []string
	parts = append(parts, "journalctl")
	parts = append(parts, fmt.Sprintf("-u %s", unit))
	parts = append(parts, "--no-pager")

	// Add time range filter if provided
	if since != "" {
		// Convert time format for journalctl
		sinceFormatted := formatSinceParameter(since)
		parts = append(parts, fmt.Sprintf("--since '%s'", sinceFormatted))
	}

	// For kubelet and containerd, use grep to filter by log level patterns
	// instead of syslog priority, as kubelet uses its own log format (E0202, W0202, I0202)
	cmd := strings.Join(parts, " ")

	switch level {
	case LogLevelError:
		// Filter for error patterns: lines starting with E (kubelet format) or containing "error"/"Error"/"ERROR"
		cmd += " | grep -iE '^[A-Z][a-z]+ [0-9]+ [0-9:]+ .* E[0-9]+|error'"
	case LogLevelWarn:
		// Filter for warning and error patterns
		cmd += " | grep -iE '^[A-Z][a-z]+ [0-9]+ [0-9:]+ .* [EW][0-9]+|error|warn'"
	}

	// Add text filter (case insensitive, fixed string match)
	if filter != "" {
		// Escape single quotes in the filter text
		escapedFilter := strings.ReplaceAll(filter, "'", "'\\''")
		cmd += fmt.Sprintf(" | grep -iF '%s'", escapedFilter)
	}

	// Add line limit at the end (after all filters)
	cmd += fmt.Sprintf(" | tail -n %d", lines)

	return cmd
}

// buildDmesgCommand builds dmesg command for kernel logs
func buildDmesgCommand(lines int, level string, filter string) string {
	var parts []string
	parts = append(parts, "dmesg -T")

	// Add level filter
	switch level {
	case LogLevelError:
		parts = append(parts, "-l err,crit,alert,emerg")
	case LogLevelWarn:
		parts = append(parts, "-l warn,err,crit,alert,emerg")
	}

	// Build command
	cmd := strings.Join(parts, " ")

	// Add text filter (case insensitive, fixed string match)
	if filter != "" {
		// Escape single quotes in the filter text
		escapedFilter := strings.ReplaceAll(filter, "'", "'\\''")
		cmd += fmt.Sprintf(" | grep -iF '%s'", escapedFilter)
	}

	// Add tail to limit output at the end (after all filters)
	cmd += fmt.Sprintf(" | tail -n %d", lines)

	return cmd
}

// buildSyslogCommand builds journalctl command for system logs
func buildSyslogCommand(lines int, since string, level string, filter string) string {
	var parts []string
	parts = append(parts, "journalctl")
	parts = append(parts, "--no-pager")

	// Add time range filter if provided
	if since != "" {
		// Convert time format for journalctl
		sinceFormatted := formatSinceParameter(since)
		parts = append(parts, fmt.Sprintf("--since '%s'", sinceFormatted))
	}

	// Add priority filter
	switch level {
	case LogLevelError:
		parts = append(parts, "-p err")
	case LogLevelWarn:
		parts = append(parts, "-p warning")
	}

	cmd := strings.Join(parts, " ")

	// Add text filter (case insensitive, fixed string match)
	if filter != "" {
		// Escape single quotes in the filter text
		escapedFilter := strings.ReplaceAll(filter, "'", "'\\''")
		cmd += fmt.Sprintf(" | grep -iF '%s'", escapedFilter)
	}

	// Add line limit at the end (after all filters)
	cmd += fmt.Sprintf(" | tail -n %d", lines)

	return cmd
}

// formatLogOutput formats the log output with metadata header
func formatLogOutput(clusterName, rg, vmssName, instanceID, logType, output string) string {
	var result strings.Builder

	// Add metadata header
	result.WriteString("=== AKS Node Logs ===\n")
	result.WriteString(fmt.Sprintf("Cluster: %s\n", clusterName))
	result.WriteString(fmt.Sprintf("Resource Group: %s\n", rg))
	result.WriteString(fmt.Sprintf("VMSS: %s\n", vmssName))
	result.WriteString(fmt.Sprintf("Instance ID: %s\n", instanceID))
	result.WriteString(fmt.Sprintf("Log Type: %s\n", logType))
	result.WriteString("=====================\n\n")

	// Add the actual log output
	result.WriteString(output)

	return result.String()
}

// validateVMSSSupport validates that the node is a Linux VMSS
// Currently only Linux VMSS are supported (not standalone VMs or Windows nodes)
func validateVMSSSupport(
	ctx context.Context,
	client *azureclient.AzureClient,
	subscriptionID string,
	nodeResourceGroup string,
	vmssName string,
) error {
	// Get VMSS client for the subscription
	clients, err := client.GetOrCreateClientsForSubscription(subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get clients for subscription: %w", err)
	}

	// Try to get VMSS to verify it exists (and is not a standalone VM)
	vmss, err := clients.VMSSClient.Get(ctx, nodeResourceGroup, vmssName, nil)
	if err != nil {
		// If VMSS not found, it's likely a standalone VM
		return fmt.Errorf("VMSS '%s' not found. This tool currently only supports VMSS-based nodes. Standalone VMs are not supported yet", vmssName)
	}

	// Check if it's a Windows VMSS
	if vmss.Properties != nil && vmss.Properties.VirtualMachineProfile != nil &&
		vmss.Properties.VirtualMachineProfile.OSProfile != nil &&
		vmss.Properties.VirtualMachineProfile.OSProfile.WindowsConfiguration != nil {
		return fmt.Errorf("windows nodes are not supported yet, this tool only supports Linux VMSS nodes")
	}

	return nil
}
