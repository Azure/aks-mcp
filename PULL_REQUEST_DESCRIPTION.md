# Pull Request: Azure Diagnostics and Resource Health Features

## Overview
This pull request enhances the AKS MCP (Model Context Protocol) server with comprehensive Azure diagnostics and resource health monitoring capabilities. It includes advisor recommendation refactoring and adds support for Azure Resource Health monitoring.

## Features Added

### 1. Azure Advisor Recommendations Refactoring ✅
- **Moved advisor package** from `internal/azure/advisor` to `internal/components/advisor`
- **Updated import paths** to align with new component-based architecture
- **Registered advisor tools at root level** in MCP server initialization
- **Enhanced recommendation filtering** using resource ID instead of impacted value
- **Improved accuracy** of AKS cluster name and resource group extraction

### 2. Azure Resource Health Monitoring 🆕
- **Added comprehensive prompt file** for implementing Resource Health feature
- **Defined new tool**: `az_monitor_activity_log_resource_health`
- **Support for Azure CLI command**: 
  ```bash
  az monitor activity-log list \
    --resource-id /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{cluster} \
    --start-time {start} \
    --query "[?category.value=='ResourceHealth']"
  ```
- **Focused on raw command results** - returns Azure CLI JSON output directly without summary/aggregation
- **Comprehensive implementation guidance** including validation and integration patterns

## Technical Changes

### Package Structure Updates
```
OLD: internal/azure/advisor/          NEW: internal/components/advisor/
OLD: internal/az/                     NEW: internal/azcli/ + internal/components/azaks/
OLD: internal/azure/resourcehandlers  NEW: internal/components/network/
```

### Files Modified/Added
- ✅ **Moved**: `internal/advisor/` → `internal/components/advisor/`
- ✅ **Updated**: Import paths in resource handlers
- ✅ **Updated**: MCP server registration logic
- ✅ **Updated**: Documentation to reflect new structure
- 🆕 **Added**: `prompts/azure-resource-health.md`

### Key Improvements
1. **Better Recommendation Filtering**: Now uses `rec.ID` instead of `rec.ImpactedValue` for more accurate cluster identification
2. **Component-based Architecture**: Aligned with latest project structure for better organization
3. **Root-level Tool Registration**: Advisor tools properly registered at MCP service root level
4. **Comprehensive Documentation**: Updated all documentation to reflect new paths and structure
5. **Simplified Resource Health**: Updated to return raw Azure CLI JSON output without summary/aggregation processing

## Testing Results ✅

### Build Verification
- ✅ Project builds successfully
- ✅ All advisor tests pass (10/10)
- ✅ All resource handler tests pass
- ✅ No compilation errors after refactoring

### Functional Testing
- ✅ Retrieved 13 Azure Advisor recommendations for "thomas" resource group
- ✅ Proper filtering for AKS clusters: `thomastest39`, `thomastest40`, `thomastest41`
- ✅ Correct categorization: HighAvailability, Cost, OperationalExcellence
- ✅ MCP server functions correctly with new component structure

## Implementation Guidelines

### Azure Resource Health Feature
The new prompt file provides comprehensive guidance for implementing:

**Parameters**:
- `subscription_id` (required): Azure subscription ID
- `resource_group` (required): Resource group name
- `cluster_name` (required): AKS cluster name  
- `start_time` (required): Start date (ISO 8601 format)
- `end_time` (optional): End date for filtering
- `status` (optional): Health status filter

**Output**:
- Returns raw Azure CLI JSON output directly
- No summary or aggregation processing
- Standard Azure Monitor activity log format

**Integration**:
- Extends existing `internal/components/monitor/` package
- Uses `internal/azcli/executor.go` for Azure CLI commands
- Follows established error handling and validation patterns

## User Experience

### Advisor Recommendations
```bash
# Get recommendations for resource group
az_advisor_recommendation --operation list --subscription-id {sub} --resource-group thomas
```

### Resource Health (Future Implementation)
```bash
# Get raw health events for AKS cluster (returns Azure CLI JSON output)
az_monitor_activity_log_resource_health \
  --subscription-id {sub} \
  --resource-group {rg} \
  --cluster-name {cluster} \
  --start-time 2025-07-01T00:00:00Z
```

## Validation Steps

1. **Code Quality**: All imports updated, no dead code
2. **Functionality**: Advisor recommendations working correctly
3. **Architecture**: Aligned with component-based structure
4. **Documentation**: All paths and examples updated
5. **Testing**: Comprehensive test coverage maintained

## Breaking Changes
None. All changes are backward compatible and maintain existing functionality while improving organization and adding new capabilities.

## Future Work
- Implement the Resource Health monitoring feature based on the provided prompt (returns raw Azure CLI JSON)
- Add more Azure Monitor integrations
- Enhance raw data retrieval capabilities

---

**Ready for Review** ✅
This PR successfully refactors the advisor functionality and provides comprehensive guidance for implementing Azure Resource Health monitoring in the AKS MCP server.
