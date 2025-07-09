# Logging in AKS-MCP Tools

This document explains how to implement and use logging in the AKS-MCP server tools.

## Overview

The AKS-MCP server uses Go's standard `log` package for logging. Logs are written to stderr by default, which allows the MCP client to see them during development and debugging.

## How to Add Logging

### 1. Import the log package

```go
import (
    "log"
    // ... other imports
)
```

### 2. Add logging statements

Use structured logging with prefixes to identify different components:

```go
// Info logging
log.Printf("[ADVISOR] Handling operation: %s", operation)
log.Printf("[ADVISOR] Found %d recommendations", len(recommendations))

// Error logging
log.Printf("[ADVISOR] Failed to execute command: %v", err)

// Debug logging
log.Printf("[ADVISOR] Command output length: %d characters", len(output))
```

### 3. Logging Best Practices

#### Use Component Prefixes
- `[ADVISOR]` - Azure Advisor recommendations
- `[AKS]` - AKS cluster operations
- `[NETWORK]` - Network-related operations
- `[SECURITY]` - Security validation
- `[CACHE]` - Cache operations

#### Log Levels
- **Info**: Normal operations, important state changes
- **Error**: Error conditions, failures
- **Debug**: Detailed information for troubleshooting

#### Examples

```go
// Good logging examples
log.Printf("[ADVISOR] Starting recommendation list for subscription: %s", subscriptionID)
log.Printf("[ADVISOR] Found %d total recommendations, %d AKS-related", total, aksCount)
log.Printf("[ADVISOR] Command execution failed: %v", err)

// Avoid logging sensitive information
log.Printf("[ADVISOR] Processing subscription: %s", subscriptionID) // OK
log.Printf("[ADVISOR] Using token: %s", token) // BAD - don't log secrets
```

## Viewing Logs

### In VS Code with MCP Extension
When you run MCP tools through VS Code, logs will appear in:
1. The VS Code Developer Console (Help > Toggle Developer Tools)
2. The MCP extension output panel

### In Terminal
If you run the MCP server directly:
```bash
./aks-mcp.exe --transport stdio 2>debug.log
```

### In Production
For production deployments, consider:
- Using structured logging (JSON format)
- Log rotation
- Centralized logging systems
- Different log levels for different environments

## Current Implementation

The Azure Advisor tool now includes comprehensive logging:

1. **Operation tracking**: Logs which operation is being performed
2. **Parameter validation**: Logs missing or invalid parameters
3. **Command execution**: Logs Azure CLI commands being executed
4. **Result processing**: Logs filtering and transformation steps
5. **Error handling**: Logs all error conditions with context

## Example Output

```
[ADVISOR] Handling operation: list
[ADVISOR] Listing recommendations for subscription: c4528d9e-c99a-48bb-b12d-fde2176a43b8, resource_group: thomas, category: , severity: 
[ADVISOR] Executing command: az advisor recommendation list --subscription c4528d9e-c99a-48bb-b12d-fde2176a43b8 --resource-group thomas --output json
[ADVISOR] Command output length: 2 characters
[ADVISOR] Successfully parsed 0 recommendations from CLI output
[ADVISOR] Found 0 total recommendations
[ADVISOR] Found 0 AKS-related recommendations
[ADVISOR] Returning 0 recommendation summaries
```

## Adding Logging to New Tools

When creating new MCP tools, follow this pattern:

```go
func HandleNewTool(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
    // Log the start of the operation
    log.Printf("[NEWTOOL] Starting operation with params: %v", params)
    
    // Validate parameters with logging
    requiredParam, ok := params["required_param"].(string)
    if !ok {
        log.Println("[NEWTOOL] Missing required_param parameter")
        return "", fmt.Errorf("required_param parameter is required")
    }
    
    // Log important steps
    log.Printf("[NEWTOOL] Processing with parameter: %s", requiredParam)
    
    // Execute operations with error logging
    result, err := someOperation(requiredParam)
    if err != nil {
        log.Printf("[NEWTOOL] Operation failed: %v", err)
        return "", fmt.Errorf("operation failed: %w", err)
    }
    
    // Log successful completion
    log.Printf("[NEWTOOL] Operation completed successfully")
    return result, nil
}
```

This logging approach helps with:
- Debugging issues during development
- Monitoring tool usage in production
- Understanding performance characteristics
- Troubleshooting user-reported problems
