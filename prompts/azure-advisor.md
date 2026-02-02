# Azure Advisor Tool for AKS-MCP

Implement Azure Advisor recommendation capabilities for AKS clusters and related resources.

## Tool: `aks_advisor_recommendation`

**Purpose**: Retrieve and manage Azure Advisor recommendations for AKS clusters

**Parameters**:
- `operation` (required): `list`, `details`, or `report`
- `subscription_id` (required): Azure subscription ID
- `resource_group` (optional): Filter by resource group
- `cluster_names` (optional): Array of AKS cluster names to filter
- `category` (optional): `Cost`, `HighAvailability`, `Performance`, `Security`
- `severity` (optional): `High`, `Medium`, `Low`
- `recommendation_id` (optional): Required for `details` operation

**Operations**:
- **list**: Return AKS-related recommendations with basic details
- **details**: Get comprehensive information for a specific recommendation
- **report**: Generate summary report of AKS recommendations by category

## Implementation Steps

1. **Use existing executor** from `internal/az/executor.go` for Azure CLI commands
2. **Parse JSON output** from Azure CLI responses
3. **Filter for AKS resources** (managedClusters, agentPools, related networking)
4. **Handle errors** gracefully with meaningful messages
5. **Return structured JSON** matching expected output format

## Key Azure CLI Commands

```bash
# List recommendations
az advisor recommendation list --subscription {sub} --output json

# Get specific recommendation
az advisor recommendation show --recommendation-id {id} --output json

# Filter by category and resource group
az advisor recommendation list --category Cost --resource-group {rg} --output json
```

## AKS Resource Filtering
Filter recommendations for:
- `Microsoft.ContainerService/managedClusters`
- `Microsoft.ContainerService/managedClusters/agentPools`
- Kubernetes-related load balancers and public IPs

## Code Structure Requirements

### File Organization
```
internal/azure/advisor/
├── aks_recommendations.go # AKS-specific filtering and processing
├── reports.go            # Report generation
└── types.go              # Data types
```

### Tool Registration
```go
func (s *Server) registerAdvisorTools() {
    s.registerTool("aks_advisor_recommendation", s.handleAdvisorRecommendation)
}
```

### Use Existing Executor
```go
import "github.com/Azure/aks-mcp/internal/az"

func (s *Server) handleAdvisorRecommendation(params map[string]interface{}) (string, error) {
    // Use existing executor instead of creating new CLI client
    executor := az.NewExecutor()
    
    // Build Azure CLI command
    args := []string{"advisor", "recommendation", "list", "--output", "json"}
    
    // Execute command using existing executor
    result, err := executor.Execute("az", args, nil)
    if err != nil {
        return "", fmt.Errorf("failed to execute advisor command: %w", err)
    }
    
    // Parse and filter results for AKS resources
    // ...
}
```

## Access Level Requirements
- **Readonly**: All operations (list, details, report)
- **Readwrite**: Enhanced filtering and custom reports

## Access Levels

- **Readonly**: All operations supported
- **Readwrite**: Enhanced filtering options
- **Admin**: Same as readwrite (no admin-specific operations)

## Expected Integration

- Add tool registration in `internal/server/server.go`
- Create handlers in `internal/azure/advisor/` directory
- Follow existing error handling patterns
- Use standard JSON output format

## Success Criteria
- ✅ Retrieve Azure Advisor recommendations for AKS resources
- ✅ Filter by category, severity, and AKS clusters
- ✅ Provide actionable implementation guidance
- ✅ Generate comprehensive advisory reports
- ✅ Handle errors gracefully with proper authentication

## Implementation Priority
1. Basic recommendation listing with AKS filtering
2. Detailed recommendation information
3. Report generation and data aggregation
4. Performance optimization and caching

Generate the implementation following these high-level specifications.
