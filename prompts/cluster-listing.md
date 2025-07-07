# AKS Cluster Listing Test Prompts

This file contains test prompts for validating the cluster listing functionality of AKS-MCP.

## Test Objectives
- Validate `list_aks_clusters` tool functionality
- Test subscription-level cluster discovery
- Test resource group filtering
- Verify output format and completeness

## Prerequisites
- Azure CLI authenticated with appropriate subscription access
- AKS-MCP server running with read access or higher
- At least one AKS cluster in the subscription for positive testing

---

## Test Cases

### 1. Basic Cluster Listing

**Test Objective**: List all AKS clusters in the current subscription

**Prompt**:
```
List all my AKS clusters in my subscription.
```

**Expected Behavior**:
- Returns list of all AKS clusters in the subscription
- Includes cluster names, resource groups, locations
- Shows cluster status and basic configuration details

---

### 2. Subscription-Specific Listing

**Test Objective**: List AKS clusters in a specific subscription

**Prompt**:
```
Show me all AKS clusters in subscription "OSTC Shanghai Dev".
```

**Expected Behavior**:
- Returns clusters only from the specified subscription
- Handles subscription name or ID format
- Shows appropriate error if subscription not accessible

---

### 3. Resource Group Filtering

**Test Objective**: List AKS clusters within a specific resource group

**Prompt**:
```
List all AKS clusters in resource group "rg-aks-prod".
```

**Expected Behavior**:
- Returns clusters only from the specified resource group
- Shows empty result if no clusters in the resource group
- Handles resource group name case sensitivity

---

### 4. Multiple Subscription Query

**Test Objective**: Attempt to list clusters across multiple subscriptions

**Prompt**:
```
List all AKS clusters across all my subscriptions.
```

**Expected Behavior**:
- Returns clusters from all accessible subscriptions
- Groups results by subscription
- Handles authentication for multiple subscriptions

---

### 5. Cluster Count and Summary

**Test Objective**: Get summary information about AKS clusters

**Prompt**:
```
How many AKS clusters do I have and what are their names?
```

**Expected Behavior**:
- Returns total count of clusters
- Lists cluster names clearly
- Provides brief summary information

---

### 6. Cluster Status Overview

**Test Objective**: Get status information for all clusters

**Prompt**:
```
Show me the status of all my AKS clusters.
```

**Expected Behavior**:
- Returns cluster names with their current status
- Shows provisioning state, power state
- Highlights any clusters with issues

---

### 7. Location-Based Filtering

**Test Objective**: List clusters in specific Azure regions

**Prompt**:
```
Show me all AKS clusters in East US region.
```

**Expected Behavior**:
- Returns clusters only from the specified region
- Handles region name variations (East US, eastus, etc.)
- Shows empty result if no clusters in the region

---

### 8. Detailed Cluster Information

**Test Objective**: Get comprehensive information about all clusters

**Prompt**:
```
Give me detailed information about all my AKS clusters including their network configuration.
```

**Expected Behavior**:
- Returns detailed cluster information
- May combine list_aks_clusters with get_cluster_info
- Shows network details, node pools, etc.

---

## Edge Cases and Error Handling

### 9. No Clusters Available

**Test Objective**: Handle empty cluster list gracefully

**Prompt**:
```
List all AKS clusters in resource group "empty-rg".
```

**Expected Behavior**:
- Returns appropriate "no clusters found" message
- Doesn't throw errors for empty results
- Provides helpful guidance

---

### 10. Invalid Subscription

**Test Objective**: Handle invalid subscription names/IDs

**Prompt**:
```
List AKS clusters in subscription "invalid-subscription-name".
```

**Expected Behavior**:
- Returns appropriate error message
- Doesn't expose sensitive information
- Suggests valid subscription names if possible

---

### 11. Permission Denied

**Test Objective**: Handle insufficient permissions gracefully

**Prompt**:
```
List all AKS clusters in subscription I don't have access to.
```

**Expected Behavior**:
- Returns appropriate permission error
- Doesn't expose unauthorized information
- Provides guidance on required permissions

---

## Security Validation

### 12. Subscription Boundary Testing

**Test Objective**: Ensure proper subscription isolation

**Prompt**:
```
List clusters from subscription "other-tenant-subscription".
```

**Expected Behavior**:
- Blocks access to unauthorized subscriptions
- Returns appropriate authorization error
- Maintains security boundaries

---

## Performance Testing

### 13. Large Subscription Handling

**Test Objective**: Handle subscriptions with many clusters

**Prompt**:
```
List all AKS clusters in my subscription with many clusters.
```

**Expected Behavior**:
- Returns results in reasonable time
- Handles pagination if necessary
- Doesn't timeout or crash

---

## Success Criteria

- ✅ All prompts return appropriate responses
- ✅ Error messages are helpful and secure
- ✅ No sensitive information is exposed
- ✅ Performance is acceptable
- ✅ Security boundaries are maintained
- ✅ Output format is consistent and readable
