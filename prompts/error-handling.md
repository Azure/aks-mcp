# Error Handling and Edge Cases Test Prompts

This file contains test prompts designed to validate error handling, edge cases, and resilience of the AKS-MCP server under various failure scenarios.

## Test Objectives
- Validate graceful error handling
- Test edge cases and boundary conditions
- Verify appropriate error messages and user guidance
- Test system resilience under adverse conditions

## Prerequisites
- AKS-MCP server running with various access levels
- Mix of accessible and inaccessible resources
- Understanding of expected error scenarios

---

## Authentication and Authorization Errors

### 1. Authentication Failure

**Test Objective**: Handle authentication failures gracefully

**Prompt**:
```
List all my AKS clusters
```

**Test Conditions**: 
- Azure CLI not authenticated
- Invalid credentials
- Expired authentication tokens

**Expected Behavior**:
- ✅ Clear error message about authentication failure
- ✅ Guidance on how to authenticate (az login)
- ✅ No sensitive information in error messages
- ✅ Consistent error format

---

### 2. Authorization Insufficient

**Test Objective**: Handle insufficient permissions

**Prompt**:
```
Get detailed information about cluster "production-cluster"
```

**Test Conditions**:
- User lacks read permissions on cluster
- Subscription access revoked
- Resource group access denied

**Expected Behavior**:
- ✅ Clear permission error message
- ✅ Indication of required permissions
- ✅ No exposure of unauthorized information
- ✅ Helpful guidance for permission requests

---

### 3. Subscription Access Denied

**Test Objective**: Handle subscription-level access issues

**Prompt**:
```
List clusters in subscription "restricted-subscription"
```

**Expected Behavior**:
- ✅ Appropriate subscription access error
- ✅ List of accessible subscriptions (if available)
- ✅ Clear indication of access restrictions
- ✅ No attempt to access restricted resources

---

## Resource Not Found Errors

### 4. Non-Existent Cluster

**Test Objective**: Handle requests for non-existent clusters

**Prompt**:
```
Get information about AKS cluster "non-existent-cluster"
```

**Expected Behavior**:
- ✅ Clear "cluster not found" error message
- ✅ Suggestions for valid cluster names
- ✅ No system errors or crashes
- ✅ Helpful troubleshooting information

---

### 5. Empty Subscription

**Test Objective**: Handle subscriptions with no AKS clusters

**Prompt**:
```
List all AKS clusters in my subscription
```

**Test Conditions**: Subscription has no AKS clusters

**Expected Behavior**:
- ✅ Clear message indicating no clusters found
- ✅ Suggestion to create clusters or check other subscriptions
- ✅ No errors for empty results
- ✅ Appropriate guidance for next steps

---

### 6. Deleted Resource References

**Test Objective**: Handle references to recently deleted resources

**Prompt**:
```
Get VNet information for cluster "recently-deleted-cluster"
```

**Expected Behavior**:
- ✅ Appropriate error for deleted/missing resources
- ✅ Clear distinction between "not found" and "access denied"
- ✅ No stale data or inconsistent states
- ✅ Graceful handling of resource dependencies

---

## Network and Connectivity Issues

### 7. Network Connectivity Failure

**Test Objective**: Handle network connectivity issues

**Prompt**:
```
List all my AKS clusters
```

**Test Conditions**: 
- Network connectivity issues
- Azure API endpoint unreachable
- DNS resolution failures

**Expected Behavior**:
- ✅ Clear network error messages
- ✅ Retry logic with appropriate backoff
- ✅ Timeout handling
- ✅ Guidance for network troubleshooting

---

### 8. API Rate Limiting

**Test Objective**: Handle Azure API rate limiting

**Prompt**:
```
Get detailed information for all 50 of my AKS clusters
```

**Test Conditions**: High API request volume triggering rate limits

**Expected Behavior**:
- ✅ Appropriate rate limit error handling
- ✅ Automatic retry with exponential backoff
- ✅ Progress indication for large operations
- ✅ Graceful degradation of service

---

### 9. Partial Network Configuration

**Test Objective**: Handle clusters with incomplete network setup

**Prompt**:
```
Get complete network configuration for cluster "partially-configured-cluster"
```

**Test Conditions**: Cluster missing some network resources

**Expected Behavior**:
- ✅ Reports available network information
- ✅ Clearly indicates missing components
- ✅ Suggests steps to complete configuration
- ✅ No errors for partial data

---

## Input Validation Errors

### 10. Invalid Resource Names

**Test Objective**: Handle malformed resource names

**Prompt**:
```
Get cluster information for ""
```

**Test Variations**:
- Empty string
- Special characters: `cluster-name!@#$%`
- Extremely long names
- Unicode characters

**Expected Behavior**:
- ✅ Clear input validation error messages
- ✅ Specification of valid name formats
- ✅ No system crashes or exceptions
- ✅ Consistent validation across all tools

---

### 11. Invalid Azure Resource IDs

**Test Objective**: Handle malformed Azure resource IDs

**Prompt**:
```
Get cluster info for resource ID "invalid-resource-id-format"
```

**Test Variations**:
- Incomplete resource IDs
- Wrong resource type in ID
- Invalid subscription GUIDs

**Expected Behavior**:
- ✅ Clear resource ID format error
- ✅ Example of correct resource ID format
- ✅ No parsing errors or crashes
- ✅ Validation before API calls

---

### 12. Null and Empty Parameters

**Test Objective**: Handle null, empty, or missing parameters

**Prompt**:
```
Get information about cluster in resource group
```

**Test Conditions**: Missing required parameters

**Expected Behavior**:
- ✅ Clear indication of missing parameters
- ✅ List of required parameters
- ✅ Example of correct usage
- ✅ No system errors for missing data

---

## Azure Service Issues

### 13. Azure Service Outage

**Test Objective**: Handle Azure service outages

**Prompt**:
```
List all my AKS clusters
```

**Test Conditions**: Azure AKS service experiencing outage

**Expected Behavior**:
- ✅ Clear error indicating service unavailability
- ✅ Suggestion to check Azure status page
- ✅ Retry recommendations
- ✅ No misleading error messages

---

### 14. API Version Compatibility

**Test Objective**: Handle API version mismatches

**Prompt**:
```
Get cluster information with latest features
```

**Test Conditions**: API version compatibility issues

**Expected Behavior**:
- ✅ Graceful handling of version mismatches
- ✅ Fallback to compatible API versions
- ✅ Clear indication of feature availability
- ✅ Appropriate error messages for unsupported features

---

### 15. Resource State Transitions

**Test Objective**: Handle resources in transitional states

**Prompt**:
```
Get information about cluster currently being created/deleted
```

**Test Conditions**: Cluster in provisioning or deleting state

**Expected Behavior**:
- ✅ Clear indication of resource state
- ✅ Appropriate handling of transitional states
- ✅ No errors for resources in flux
- ✅ Helpful status information

---

## Configuration and Access Level Issues

### 16. Insufficient Access Level

**Test Objective**: Handle operations beyond current access level

**Prompt**:
```
Modify cluster configuration
```

**Test Conditions**: Running with read-only access level

**Expected Behavior**:
- ✅ Clear access level restriction error
- ✅ Indication of required access level
- ✅ No attempt to perform unauthorized operations
- ✅ Guidance on changing access levels

---

### 17. Configuration Conflicts

**Test Objective**: Handle conflicting configuration settings

**Prompt**:
```
Get cluster information
```

**Test Conditions**: 
- Conflicting command line arguments
- Invalid configuration combinations

**Expected Behavior**:
- ✅ Clear configuration error messages
- ✅ Identification of conflicting settings
- ✅ Suggestions for resolution
- ✅ Validation of configuration on startup

---

## Performance and Resource Constraints

### 18. Large Dataset Handling

**Test Objective**: Handle operations with large result sets

**Prompt**:
```
Get detailed information for all resources in subscription with 1000+ clusters
```

**Expected Behavior**:
- ✅ Appropriate handling of large datasets
- ✅ Pagination or streaming for large results
- ✅ Memory usage optimization
- ✅ Progress indication for long operations

---

### 19. Timeout Scenarios

**Test Objective**: Handle operation timeouts

**Prompt**:
```
Get comprehensive analysis of complex cluster environment
```

**Test Conditions**: Operations exceeding timeout limits

**Expected Behavior**:
- ✅ Clear timeout error messages
- ✅ Partial results if available
- ✅ Suggestions for reducing scope
- ✅ Configurable timeout settings

---

### 20. Resource Exhaustion

**Test Objective**: Handle system resource constraints

**Prompt**:
```
Perform intensive operations on many clusters simultaneously
```

**Test Conditions**: System memory or CPU constraints

**Expected Behavior**:
- ✅ Graceful degradation under resource pressure
- ✅ Appropriate error messages for resource limits
- ✅ No system crashes or instability
- ✅ Resource usage optimization

---

## Data Consistency and Integrity

### 21. Stale Data Handling

**Test Objective**: Handle potentially stale cached data

**Prompt**:
```
Get real-time status of recently modified cluster
```

**Expected Behavior**:
- ✅ Appropriate cache invalidation
- ✅ Clear indication of data freshness
- ✅ Options for forcing fresh data retrieval
- ✅ Consistent data across multiple calls

---

### 22. Concurrent Modification

**Test Objective**: Handle resources modified during operation

**Prompt**:
```
Get detailed analysis of cluster being modified by another process
```

**Expected Behavior**:
- ✅ Consistent data snapshots
- ✅ Appropriate handling of concurrent changes
- ✅ Clear indication of data currency
- ✅ No data corruption or inconsistency

---

## Success Criteria

### Error Handling Requirements
- ✅ All errors produce clear, actionable messages
- ✅ No sensitive information exposed in error messages
- ✅ Consistent error format across all scenarios
- ✅ Appropriate HTTP status codes or error types
- ✅ No system crashes or unhandled exceptions

### User Experience Requirements
- ✅ Helpful guidance for resolving errors
- ✅ Clear indication of what went wrong
- ✅ Suggestions for alternative actions
- ✅ Professional, user-friendly error messages
- ✅ Consistent behavior across different error types

### System Resilience Requirements
- ✅ Graceful degradation under adverse conditions
- ✅ Appropriate retry logic with backoff
- ✅ Resource usage optimization
- ✅ No data corruption or loss
- ✅ Stable operation under various failure modes

### Security Requirements
- ✅ No information disclosure through error messages
- ✅ Proper access control enforcement
- ✅ Secure handling of authentication failures
- ✅ No privilege escalation through error conditions
- ✅ Audit trail for error conditions (if logging enabled)
