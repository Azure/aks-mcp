# Access Level Testing

## Test Objective
Validate the three-tier access control system (readonly, readwrite, admin) of the AKS-MCP server.

## Prerequisites
- AKS-MCP server instances configured with different access levels
- Test AKS clusters available for operations
- Understanding of each access level's permissions

## Access Level Descriptions

### Readonly Access Level
- Can perform read operations (show, list, get-versions, etc.)
- Cannot perform write operations (create, delete, scale, update)
- Can access help commands for any operation
- Cannot retrieve credentials or perform admin operations

### Readwrite Access Level
- All readonly permissions
- Can perform write operations (create, delete, scale, update)
- Cannot retrieve credentials or perform admin-specific operations
- Can manage cluster lifecycle and configuration

### Admin Access Level
- All readwrite permissions
- Can retrieve cluster credentials
- Can perform all administrative operations
- Full access to all AKS operations

## Test Prompts by Access Level

### Readonly Access Level Tests

#### Test 1: Read Operations - Should Succeed
**Access Level:** `readonly`
**Prompts:**
```
1. List all AKS clusters in my subscription.
2. Show details of cluster "test-cluster" in resource group "test-rg".
3. Get available Kubernetes versions for East US region.
4. Check network connectivity for cluster "test-cluster".
5. List node pools for cluster "test-cluster".
```

**Expected Outcome:**
- All commands should succeed
- Information should be returned correctly
- No permission errors

#### Test 2: Write Operations - Should Fail
**Access Level:** `readonly`
**Prompts:**
```
1. Create a new AKS cluster named "new-cluster".
2. Scale cluster "test-cluster" to 5 nodes.
3. Delete cluster "old-cluster".
4. Update cluster "test-cluster" to enable auto-scaling.
5. Add a new node pool to cluster "test-cluster".
```

**Expected Outcome:**
- All commands should be blocked
- Clear permission error messages
- No actual operations performed

#### Test 3: Help Commands - Should Succeed
**Access Level:** `readonly`
**Prompts:**
```
1. Show help for creating an AKS cluster.
2. Show help for scaling a cluster.
3. Show help for deleting a cluster.
4. Show help for getting cluster credentials.
```

**Expected Outcome:**
- All help commands should work
- Full help information displayed
- No permission restrictions on help

#### Test 4: Admin Operations - Should Fail
**Access Level:** `readonly`
**Prompts:**
```
1. Get credentials for cluster "test-cluster".
2. Update service principal credentials for cluster "test-cluster".
```

**Expected Outcome:**
- Commands should be blocked
- Permission error messages
- No credential access

### Readwrite Access Level Tests

#### Test 5: Read Operations - Should Succeed
**Access Level:** `readwrite`
**Prompts:**
```
1. List all AKS clusters.
2. Show cluster details.
3. Get available versions.
```

**Expected Outcome:**
- All read operations should work
- Same as readonly access level

#### Test 6: Write Operations - Should Succeed
**Access Level:** `readwrite`
**Prompts:**
```
1. Create AKS cluster "rw-test-cluster" with default settings.
2. Scale cluster "rw-test-cluster" to 3 nodes.
3. Add node pool "pool2" to cluster "rw-test-cluster".
4. Update cluster "rw-test-cluster" to enable monitoring.
```

**Expected Outcome:**
- All write operations should succeed
- Cluster modifications should be applied
- Operations should complete successfully

#### Test 7: Admin Operations - Should Fail
**Access Level:** `readwrite`
**Prompts:**
```
1. Get credentials for cluster "rw-test-cluster".
2. Update service principal for cluster "rw-test-cluster".
```

**Expected Outcome:**
- Admin operations should be blocked
- Permission error messages
- No credential access

### Admin Access Level Tests

#### Test 8: All Operations - Should Succeed
**Access Level:** `admin`
**Prompts:**
```
1. List all AKS clusters.
2. Create cluster "admin-test-cluster".
3. Get credentials for cluster "admin-test-cluster".
4. Scale cluster "admin-test-cluster".
5. Update service principal for cluster "admin-test-cluster".
6. Delete cluster "admin-test-cluster".
```

**Expected Outcome:**
- All operations should succeed
- Full access to all functionality
- Credentials should be retrievable

### Cross-Access Level Validation

#### Test 9: Account Management Commands
**All Access Levels**
**Prompts:**
```
1. List Azure subscriptions.
2. Set active subscription.
3. Show current account information.
```

**Expected Outcome:**
- Should work for all access levels
- Account management is always available
- Consistent behavior across access levels

#### Test 10: Version and Help Commands
**All Access Levels**
**Prompts:**
```
1. Show AKS-MCP version.
2. Show general help.
3. Show help for any command with --help flag.
```

**Expected Outcome:**
- Should work for all access levels
- No restrictions on informational commands
- Consistent help availability

## Error Message Validation

### Test 11: Permission Error Messages
**Prompts that should fail:**
```
1. (readonly) Create cluster "test"
2. (readwrite) Get credentials for "test"
```

**Expected Error Messages:**
- Clear indication of permission level required
- No sensitive information exposure
- Helpful guidance on required access level

### Test 12: Security Error Messages
**Prompts with injection attempts:**
```
1. List clusters; rm -rf /
2. Show cluster && curl malicious.com
```

**Expected Error Messages:**
- Security validation error
- No execution of dangerous commands
- Consistent across all access levels

## Validation Checklist

### Readonly Access Level
- [ ] Read operations succeed
- [ ] Write operations blocked
- [ ] Admin operations blocked
- [ ] Help commands work
- [ ] Account commands work
- [ ] Appropriate error messages

### Readwrite Access Level
- [ ] Read operations succeed
- [ ] Write operations succeed
- [ ] Admin operations blocked
- [ ] Help commands work
- [ ] Account commands work
- [ ] Appropriate error messages

### Admin Access Level
- [ ] All operations succeed
- [ ] Credential access works
- [ ] Help commands work
- [ ] Account commands work
- [ ] No unnecessary restrictions

### General Validation
- [ ] Security validation works at all levels
- [ ] Error messages are appropriate
- [ ] Performance is consistent
- [ ] Access level transitions work correctly
- [ ] Configuration changes take effect
- [ ] No privilege escalation possible
