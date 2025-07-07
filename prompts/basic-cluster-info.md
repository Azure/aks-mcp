# Basic AKS Cluster Information Tests

## Test Objective
Validate basic cluster information retrieval functionality of the AKS-MCP server.

## Prerequisites
- AKS-MCP server running with at least `readonly` access level
- Valid Azure subscription with AKS clusters
- Proper Azure authentication configured

## Test Prompts

### Test 1: List All AKS Clusters
**Prompt:**
```
List all AKS clusters in my Azure subscription.
```

**Expected Outcome:**
- Should return a table/list of AKS clusters
- Include cluster names, locations, resource groups, and status
- Should work with readonly access level

### Test 2: Show Specific Cluster Details
**Prompt:**
```
Show me detailed information about the AKS cluster named "my-test-cluster" in resource group "my-rg".
```

**Expected Outcome:**
- Detailed cluster information including:
  - Kubernetes version
  - Node pools
  - Networking configuration
  - Add-ons enabled
  - Resource allocation

### Test 3: Get Cluster Version Information
**Prompt:**
```
What Kubernetes versions are available for AKS clusters in the East US region?
```

**Expected Outcome:**
- List of available Kubernetes versions
- Should include current and supported versions
- May include preview versions if available

### Test 4: Check Network Connectivity
**Prompt:**
```
Check the outbound network connectivity for the cluster "test-cluster" in resource group "test-rg".
```

**Expected Outcome:**
- Network connectivity check results
- Should identify any connectivity issues
- List accessible endpoints

### Test 5: Invalid Cluster Name
**Prompt:**
```
Show me information about the cluster "non-existent-cluster" in resource group "test-rg".
```

**Expected Outcome:**
- Graceful error handling
- Clear error message indicating cluster not found
- No security information leakage

## Validation Checklist
- [ ] All commands execute without errors
- [ ] Responses contain expected information
- [ ] Error handling works appropriately
- [ ] No sensitive information exposed in errors
- [ ] Response times are reasonable (< 30 seconds)
- [ ] Commands work with readonly access level
