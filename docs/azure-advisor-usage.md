# Azure Advisor Tool for AKS-MCP - Usage Examples

This document provides examples of how to use the newly implemented Azure Advisor tool (`aks_advisor_recommendation`) in the AKS-MCP server.

## Tool Overview

The `aks_advisor_recommendation` tool provides Azure Advisor recommendations specifically for AKS clusters and related resources. It supports three main operations:

- **list**: Return AKS-related recommendations with basic details
- **details**: Get comprehensive information for a specific recommendation
- **report**: Generate summary report of AKS recommendations by category

## Prerequisites

1. Azure CLI installed and authenticated
2. Access to Azure subscription with AKS clusters
3. AKS-MCP server running with appropriate access level (readonly or higher)

## Usage Examples

### 1. List All AKS Recommendations

```json
{
  "operation": "list",
  "subscription_id": "12345678-1234-1234-1234-123456789012"
}
```

### 2. List Cost Recommendations for Specific Resource Group

```json
{
  "operation": "list",
  "subscription_id": "12345678-1234-1234-1234-123456789012",
  "resource_group": "my-aks-rg",
  "category": "Cost"
}
```

### 3. List High-Priority Recommendations for Specific Clusters

```json
{
  "operation": "list",
  "subscription_id": "12345678-1234-1234-1234-123456789012",
  "cluster_names": ["aks-prod-1", "aks-staging-2"],
  "severity": "High"
}
```

### 4. Get Detailed Information for a Specific Recommendation

```json
{
  "operation": "details",
  "subscription_id": "12345678-1234-1234-1234-123456789012",
  "recommendation_id": "/subscriptions/12345678-1234-1234-1234-123456789012/providers/Microsoft.Advisor/recommendations/abcd1234-5678-90ef-ghij-klmnopqrstuv"
}
```

### 5. Generate Summary Report

```json
{
  "operation": "report",
  "subscription_id": "12345678-1234-1234-1234-123456789012",
  "format": "summary"
}
```

### 6. Generate Detailed Report for Resource Group

```json
{
  "operation": "report",
  "subscription_id": "12345678-1234-1234-1234-123456789012",
  "resource_group": "production-aks",
  "format": "detailed"
}
```

### 7. Generate Actionable Report

```json
{
  "operation": "report",
  "subscription_id": "12345678-1234-1234-1234-123456789012",
  "format": "actionable"
}
```

## Expected Output Formats

### List Operation Output

```json
[
  {
    "id": "/subscriptions/.../recommendations/rec1",
    "category": "Cost",
    "impact": "High",
    "cluster_name": "aks-cluster-1",
    "resource_group": "my-rg",
    "impacted_resource": "/subscriptions/.../managedClusters/aks-cluster-1",
    "description": "Underutilized AKS cluster nodes Consider reducing node count or using smaller VM sizes",
    "severity": "High",
    "last_updated": "2024-01-15T10:30:00Z",
    "status": "Active",
    "aks_specific": {
      "configuration_area": "compute"
    }
  }
]
```

### Details Operation Output

```json
{
  "id": "/subscriptions/.../recommendations/rec1",
  "category": "Cost",
  "impact": "High",
  "cluster_name": "aks-cluster-1",
  "resource_group": "my-rg",
  "impacted_resource": "/subscriptions/.../managedClusters/aks-cluster-1",
  "description": "Detailed recommendation description with implementation guidance",
  "severity": "High",
  "potential_savings": {
    "currency": "USD",
    "annual_savings": 1200.00,
    "monthly_savings": 100.00
  },
  "last_updated": "2024-01-15T10:30:00Z",
  "status": "Active",
  "aks_specific": {
    "cluster_version": "1.28.5",
    "node_pool_names": ["nodepool1", "nodepool2"],
    "workload_type": "production",
    "configuration_area": "compute"
  }
}
```

### Report Operation Output

```json
{
  "subscription_id": "12345678-1234-1234-1234-123456789012",
  "generated_at": "2024-07-08T19:30:00Z",
  "summary": {
    "total_recommendations": 5,
    "by_category": {
      "Cost": 2,
      "Security": 2,
      "Performance": 1
    },
    "by_severity": {
      "High": 2,
      "Medium": 2,
      "Low": 1
    },
    "clusters_affected": 3
  },
  "recommendations": [...],
  "action_items": [
    {
      "priority": 1,
      "recommendation_id": "/subscriptions/.../recommendations/rec1",
      "cluster_name": "aks-cluster-1",
      "category": "Cost",
      "description": "High-priority cost optimization",
      "estimated_effort": "Medium",
      "potential_impact": "High"
    }
  ],
  "cluster_breakdown": [
    {
      "cluster_name": "aks-cluster-1",
      "resource_group": "my-rg",
      "recommendations": [...],
      "total_savings": {
        "currency": "USD",
        "annual_savings": 1200.00,
        "monthly_savings": 100.00
      }
    }
  ]
}
```

## Filtering Options

### By Category
- `Cost`: Cost optimization recommendations
- `HighAvailability`: High availability and reliability improvements
- `Performance`: Performance optimization suggestions
- `Security`: Security and compliance recommendations

### By Severity
- `High`: High-impact recommendations requiring immediate attention
- `Medium`: Medium-impact recommendations for planned implementation
- `Low`: Low-impact recommendations for future consideration

### By Resource Scope
- `subscription_id`: Required - Azure subscription to query
- `resource_group`: Optional - Filter to specific resource group
- `cluster_names`: Optional - Array of specific AKS cluster names

## Access Levels

- **Readonly**: All operations supported (list, details, report)
- **Readwrite**: Enhanced filtering and custom report options
- **Admin**: Same as readwrite (no admin-specific operations for advisor)

## Error Handling

The tool provides meaningful error messages for common scenarios:

- Missing required parameters
- Invalid operation types
- Azure CLI authentication issues
- Non-existent recommendation IDs
- Invalid subscription or resource group access

## Integration with Other Tools

The Azure Advisor tool can be used in conjunction with other AKS-MCP tools:

1. Use AKS cluster listing tools to identify cluster names
2. Use network tools to understand infrastructure context
3. Use monitoring tools to correlate recommendations with performance data

## Best Practices

1. **Regular Reviews**: Schedule periodic runs to check for new recommendations
2. **Priority Focus**: Address high-severity recommendations first
3. **Cost Monitoring**: Use cost category filtering to identify savings opportunities
4. **Security Posture**: Regularly review security recommendations
5. **Documentation**: Keep track of implemented recommendations and their outcomes

## Troubleshooting

### Common Issues

1. **"No recommendations found"**: 
   - Verify Azure CLI authentication
   - Check subscription access
   - Ensure AKS clusters exist in the subscription

2. **"Command execution failed"**:
   - Verify Azure CLI is installed and updated
   - Check network connectivity to Azure
   - Verify proper permissions

3. **"Invalid recommendation ID"**:
   - Use the exact ID from list operation
   - Ensure recommendation hasn't been dismissed or resolved

For additional support, check the Azure Advisor documentation and AKS-MCP server logs.
