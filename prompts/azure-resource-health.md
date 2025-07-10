# Azure Resource Health Tool for AKS-MCP

Implement Azure Resource Health monitoring capabilities for AKS clusters to track health events and service disruptions.

## Tool: `az_monitor_activity_log_resource_health`

**Purpose**: Retrieve resource health events for AKS clusters to monitor service availability and health status

**Parameters**:
- `subscription_id` (required): Azure subscription ID
- `resource_group` (required): Resource group name containing the AKS cluster
- `cluster_name` (required): AKS cluster name
- `start_time` (required): Start date for health event query (ISO 8601 format, e.g., "2025-01-01T00:00:00Z")
- `end_time` (optional): End date for health event query (defaults to current time)
- `status` (optional): Filter by health status (`Available`, `Unavailable`, `Degraded`, `Unknown`)

**Operations**:
- **list**: Return resource health events for the specified AKS cluster within the time range
- **summary**: Generate a summary of health events grouped by status and timeframe

## Implementation Steps

1. **Use existing executor** from `internal/azcli/executor.go` for Azure CLI commands
2. **Build resource ID** from subscription, resource group, and cluster name
3. **Parse JSON output** from Azure CLI responses
4. **Filter for ResourceHealth category** events
5. **Handle time range validation** and format conversion
6. **Return structured JSON** with health event details

## Key Azure CLI Command

```bash
# Get resource health events for AKS cluster
az monitor activity-log list \
  --resource-id /subscriptions/{{ SUBSCRIPTION_ID }}/resourceGroups/{{ RESOURCE_GROUP_NAME }}/providers/Microsoft.ContainerService/managedClusters/{{ CLUSTER_NAME }} \
  --start-time {{ START_DATE }} \
  --query "[?category.value=='ResourceHealth']" \
  --output json

# Example with specific parameters
az monitor activity-log list \
  --resource-id /subscriptions/82d6efa7-b1b6-4aa0-ab12-d10788552670/resourceGroups/thomas/providers/Microsoft.ContainerService/managedClusters/thomastest39 \
  --start-time 2025-01-01T00:00:00Z \
  --query "[?category.value=='ResourceHealth']" \
  --output json
```

## Resource Health Event Types
Monitor for these key health events:
- **Service Health**: Platform service issues affecting AKS
- **Resource Health**: Cluster-specific health status changes
- **Planned Maintenance**: Scheduled maintenance events
- **Service Issues**: Unplanned service disruptions

## Code Structure Requirements

### File Organization
```
internal/components/monitor/
├── resource_health.go     # Resource health event processing
├── handlers.go           # MCP tool handlers (extend existing)
├── registry.go           # Tool registration (extend existing)
└── types.go              # Data types for health events
```

### Tool Registration
```go
func RegisterResourceHealthCommands() []MonitorCommand {
    return []MonitorCommand{
        {
            Name:        "az monitor activity-log resource-health",
            Description: "Retrieves resource health events for an AKS cluster since a specific date",
            ArgsExample: "--resource-id /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{cluster} --start-time 2025-01-01T00:00:00Z --query \"[?category.value=='ResourceHealth']\"",
            Category:    "resource-health",
        },
    }
}
```

### Use Existing Executor
```go
import "github.com/Azure/aks-mcp/internal/azcli"

func HandleResourceHealthQuery(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
    // Extract and validate parameters
    subscriptionID, _ := params["subscription_id"].(string)
    resourceGroup, _ := params["resource_group"].(string)
    clusterName, _ := params["cluster_name"].(string)
    startTime, _ := params["start_time"].(string)
    
    // Build resource ID
    resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
        subscriptionID, resourceGroup, clusterName)
    
    // Build Azure CLI command
    executor := azcli.NewExecutor()
    args := []string{
        "monitor", "activity-log", "list",
        "--resource-id", resourceID,
        "--start-time", startTime,
        "--query", "[?category.value=='ResourceHealth']",
        "--output", "json",
    }
    
    // Add end time if provided
    if endTime, ok := params["end_time"].(string); ok && endTime != "" {
        args = append(args, "--end-time", endTime)
    }
    
    // Execute command
    cmdParams := map[string]interface{}{
        "command": "az " + strings.Join(args, " "),
    }
    
    result, err := executor.Execute(cmdParams, cfg)
    if err != nil {
        return "", fmt.Errorf("failed to execute resource health query: %w", err)
    }
    
    // Parse and process results
    return processResourceHealthEvents(result, params)
}
```

## Data Types

```go
// ResourceHealthEvent represents a resource health event from Azure Monitor
type ResourceHealthEvent struct {
    ID               string    `json:"id"`
    EventTimestamp   time.Time `json:"event_timestamp"`
    SubmissionTimestamp time.Time `json:"submission_timestamp"`
    Level            string    `json:"level"`
    Status           string    `json:"status"`
    SubStatus        string    `json:"sub_status"`
    ResourceID       string    `json:"resource_id"`
    ResourceGroupName string   `json:"resource_group_name"`
    ClusterName      string    `json:"cluster_name"`
    Category         string    `json:"category"`
    Description      string    `json:"description"`
    Properties       map[string]interface{} `json:"properties,omitempty"`
}

// ResourceHealthSummary provides aggregated health information
type ResourceHealthSummary struct {
    ClusterName      string                    `json:"cluster_name"`
    ResourceGroup    string                    `json:"resource_group"`
    TimeRange        TimeRange                 `json:"time_range"`
    TotalEvents      int                       `json:"total_events"`
    ByStatus         map[string]int            `json:"by_status"`
    ByLevel          map[string]int            `json:"by_level"`
    RecentEvents     []ResourceHealthEvent     `json:"recent_events"`
    HealthTrend      string                    `json:"health_trend"` // "improving", "stable", "degrading"
}

// TimeRange represents a time period for queries
type TimeRange struct {
    StartTime time.Time `json:"start_time"`
    EndTime   time.Time `json:"end_time"`
}
```

## Access Level Requirements
- **Readonly**: All operations (list, summary)
- **Readwrite**: Same as readonly (monitoring is read-only)
- **Admin**: Same as readonly (monitoring is read-only)

## Validation Requirements

### Parameter Validation
```go
func validateResourceHealthParams(params map[string]interface{}) error {
    // Validate required parameters
    required := []string{"subscription_id", "resource_group", "cluster_name", "start_time"}
    for _, param := range required {
        if value, ok := params[param].(string); !ok || value == "" {
            return fmt.Errorf("missing or invalid %s parameter", param)
        }
    }
    
    // Validate time format
    startTime := params["start_time"].(string)
    if _, err := time.Parse(time.RFC3339, startTime); err != nil {
        return fmt.Errorf("invalid start_time format, expected RFC3339 (ISO 8601): %w", err)
    }
    
    // Validate end_time if provided
    if endTime, ok := params["end_time"].(string); ok && endTime != "" {
        if _, err := time.Parse(time.RFC3339, endTime); err != nil {
            return fmt.Errorf("invalid end_time format, expected RFC3339 (ISO 8601): %w", err)
        }
    }
    
    return nil
}
```

## Expected Integration

- Extend existing `internal/components/monitor/registry.go` with resource health commands
- Add handler functions to `internal/components/monitor/handlers.go`
- Follow existing error handling patterns from advisor and network components
- Use standard JSON output format
- Integrate with existing security validation

## User Experience

**User Description Template**: 
"Get resource health events for AKS cluster {{ CLUSTER_NAME }} under resource group {{ RESOURCE_GROUP_NAME }} in subscription {{ SUBSCRIPTION_ID }} since {{ START_DATE }}"

**Example Usage**:
```bash
# Get health events for the last 7 days
az_monitor_activity_log_resource_health \
  --subscription-id 82d6efa7-b1b6-4aa0-ab12-d10788552670 \
  --resource-group thomas \
  --cluster-name thomastest39 \
  --start-time 2025-07-03T00:00:00Z

# Get health events for a specific time range
az_monitor_activity_log_resource_health \
  --subscription-id 82d6efa7-b1b6-4aa0-ab12-d10788552670 \
  --resource-group thomas \
  --cluster-name thomastest39 \
  --start-time 2025-07-01T00:00:00Z \
  --end-time 2025-07-10T00:00:00Z \
  --status Available
```

## Success Criteria
- ✅ Retrieve resource health events for specific AKS clusters
- ✅ Filter by time range and health status
- ✅ Parse and structure Azure Monitor activity log data
- ✅ Handle time zone and date format conversion
- ✅ Provide meaningful error messages for invalid parameters
- ✅ Generate summary reports of health trends
- ✅ Integrate with existing MCP tool framework

## Implementation Priority
1. Basic resource health event retrieval with time filtering
2. Health status filtering and categorization
3. Summary and trend analysis features
4. Integration with existing monitoring tools
5. Performance optimization for large time ranges

## Error Handling
- Validate Azure resource ID format
- Handle Azure CLI authentication errors
- Validate time range parameters (start before end, not future dates)
- Handle empty result sets gracefully
- Provide clear error messages for malformed queries

Generate the implementation following these high-level specifications and integrate with the existing `internal/components/monitor/` package structure.
