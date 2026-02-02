# AKS-MCP Tool Consolidations

## Executive Summary

The current AKS-MCP server registers approximately 40+ individual tools, which can cause reliability issues with LLM AI systems that don't handle large numbers of tools well. This plan outlines a strategy to consolidate these tools into 6-8 meta-tools while maintaining all functionality, security checks, and access controls.

## Current State Analysis

### Tool Categories and Current Count

1. **Azure CLI Commands** (~25 tools)
   - AKS Commands: 14 tools (show, list, create, delete, scale, update, etc.)
   - Monitor Commands: 3 tools (metrics operations)
   - Account Commands: 3 tools (login, list, set)
   - Special Commands: 5 tools (fleet, resource health, app insights, etc.)

2. **Azure Resource Tools** (~10 tools)
   - Network Tools: 6 tools (VNet, NSG, RouteTable, Subnet, LoadBalancer, PrivateEndpoint)
   - Detector Tools: 3 tools (list, run, run by category)
   - Compute Tools: 1+ tools (VMSS operations)

3. **Control Plane Tools** (2 tools)
   - Diagnostic settings
   - Logs querying

4. **Kubernetes Tools** (3+ tools)
   - kubectl commands
   - helm (optional)
   - cilium (optional)
   - hubble (optional)

5. **Advisor Tools** (1 tool)
   - Advisor recommendations

### Current Registration Patterns

1. **Command-based tools**: Use generic "args" parameter with command executors
2. **Resource-based tools**: Use specific parameters (subscription_id, resource_group, etc.)
3. **Access control**: Enforced at registration time based on `cfg.AccessLevel`

## Proposed Consolidated Tool Structure

### 1. `az_aks_operations` - Unified AKS Management Tool

**Combines:**
- All AKS show/list/get operations
- All AKS create/update/delete operations
- All nodepool operations
- Account management commands

**Parameters:**
```json
{
  "operation": "show|list|create|delete|scale|update|upgrade|nodepool-*|account-*",
  "resource_type": "cluster|nodepool|account",
  "args": "additional arguments as needed"
}
```

**Benefits:**
- Reduces ~20 AKS-related tools to 1
- Maintains access control through operation validation
- Cleaner interface for users

### 2. `aks_network_resources` - Unified Network Resource Query Tool

**Combines:**
- VNet info
- NSG info
- RouteTable info
- Subnet info
- LoadBalancer info
- PrivateEndpoint info

**Parameters:**
```json
{
  "resource_type": "all|vnet|nsg|route_table|subnet|load_balancer|private_endpoint",
  "subscription_id": "subscription ID",
  "resource_group": "resource group name",
  "cluster_name": "AKS cluster name",
  "filters": "optional filters"
}
```

**Benefits:**
- Reduces 6 network tools to 1
- Can fetch all network resources or specific types
- More efficient for comprehensive network diagnostics

### 3. `aks_monitoring` - Unified Monitoring and Diagnostics Tool

**Combines:**
- Metrics operations
- Resource health monitoring
- App Insights queries
- Control plane diagnostics
- Control plane logs

**Parameters:**
```json
{
  "operation": "metrics|resource_health|app_insights|diagnostics|logs",
  "query_type": "specific to operation",
  "parameters": "operation-specific parameters"
}
```

**Benefits:**
- Reduces ~8 monitoring tools to 1
- Centralized monitoring interface
- Easier to add new monitoring capabilities

### 4. `aks_diagnostics` - Unified Diagnostics and Detectors Tool

**Combines:**
- List detectors
- Run specific detector
- Run detectors by category
- Advisor recommendations

**Parameters:**
```json
{
  "operation": "list_detectors|run_detector|run_category|advisor",
  "detector_name": "optional specific detector",
  "category": "optional category",
  "parameters": "operation-specific parameters"
}
```

**Benefits:**
- Reduces 4 diagnostic tools to 1
- Unified interface for all diagnostic operations
- Easier to discover available diagnostics

### 5. `az_compute_resources` - Unified Compute Resource Tool

**Combines:**
- VMSS info operations
- VMSS command operations
- Other compute-related operations

**Parameters:**
```json
{
  "operation": "info|run_command|scale|other",
  "resource_type": "vmss|other",
  "parameters": "operation-specific parameters"
}
```

**Benefits:**
- Consolidates compute operations
- Maintains access control for sensitive operations
- Extensible for future compute resources

### 6. `k8s_operations` - Unified Kubernetes Operations Tool

**Combines:**
- kubectl commands
- helm operations (if enabled)
- cilium operations (if enabled)
- hubble commands (if enabled)

**Parameters:**
```json
{
  "tool": "kubectl|helm|cilium|hubble",
  "command": "specific command",
  "args": "command arguments"
}
```

**Benefits:**
- Single entry point for all Kubernetes operations
- Dynamic tool availability based on configuration
- Consistent interface across K8s tools

## Implementation Strategy

### Phase 1: Infrastructure Preparation
1. Create common parameter parsing utilities
2. Develop unified error handling framework
3. Build access control validation layer
4. Create operation routing infrastructure

### Phase 2: Tool Implementation
1. Implement each consolidated tool with backward compatibility
2. Create comprehensive parameter validation
3. Ensure all security checks are preserved
4. Add detailed operation documentation

### Phase 3: Migration
1. Update server.go to use new consolidated tools
2. Maintain old tool registrations temporarily (with deprecation notices)
3. Update all tests to use new tool structure
4. Create migration guide for users

### Phase 4: Cleanup
1. Remove deprecated individual tools
2. Optimize performance for consolidated operations
3. Update all documentation
4. Release new version

## Security Considerations

1. **Access Control**: Each consolidated tool must validate access levels for specific operations
2. **Parameter Validation**: Strict validation to prevent injection attacks
3. **Audit Logging**: Maintain detailed logs of all operations
4. **Error Handling**: Never expose sensitive information in error messages

## Benefits of Consolidation

1. **Reduced Tool Count**: From ~40+ tools to 6-8 tools
2. **Better LLM Performance**: Fewer tools for AI to consider
3. **Improved Maintainability**: Less code duplication
4. **Enhanced User Experience**: More intuitive tool structure
5. **Easier Extension**: Clear patterns for adding new functionality

## Backward Compatibility

To ensure smooth transition:
1. Support legacy tool names with deprecation warnings
2. Automatic parameter translation where possible
3. Clear migration documentation
4. Phased rollout with opt-in period

## Success Metrics

1. Tool count reduced by 80%+
2. No loss of functionality
3. Improved response time from LLM
4. Reduced code complexity
5. Positive user feedback on new structure


## Conclusion

This consolidation plan will significantly improve the AKS-MCP server's compatibility with LLM AI systems while maintaining all existing functionality and security. The reduced tool count will lead to more reliable AI interactions and easier maintenance of the codebase.
