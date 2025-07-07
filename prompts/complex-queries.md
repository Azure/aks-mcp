# Complex Multi-Step Operations Test Prompts

This file contains test prompts for validating complex, multi-step operations that combine multiple AKS-MCP tools and require sophisticated reasoning.

## Test Objectives
- Validate multi-tool orchestration capabilities
- Test complex workflow execution
- Verify data correlation across multiple API calls
- Test AI assistant's ability to chain operations logically

## Prerequisites
- AKS-MCP server running with appropriate access level
- Multiple AKS clusters available for testing
- AI assistant capable of multi-step reasoning
- Network resources (VNets, subnets, NSGs) associated with clusters

---

## Complex Query Test Cases

### 1. Comprehensive Cluster Analysis

**Test Objective**: Perform complete analysis of a cluster and its associated resources

**Prompt**:
```
Analyze my AKS cluster named "production-cluster" and give me a comprehensive report including:
- Cluster configuration and health status
- Network architecture (VNet, subnets, routing)
- Security configuration (NSGs, access controls)
- Resource utilization and capacity
- Any potential issues or recommendations
```

**Expected Behavior**:
- Uses `get_cluster_info` to get cluster details
- Uses `get_vnet_info` to analyze network configuration
- Uses `get_subnet_info` to examine subnet configuration
- Uses `get_nsg_info` to review security rules
- Uses `get_route_table_info` to understand routing
- Correlates data across all tools to provide insights
- Presents findings in organized, actionable format

---

### 2. Multi-Cluster Comparison

**Test Objective**: Compare configurations across multiple clusters

**Prompt**:
```
Compare the network configurations of all my AKS clusters and identify:
- Which clusters share the same VNet
- Differences in subnet configurations
- Variations in network security group rules
- Potential connectivity issues between clusters
- Recommendations for standardization
```

**Expected Behavior**:
- Uses `list_aks_clusters` to discover all clusters
- Uses `get_cluster_info` for each cluster
- Uses `get_vnet_info` and `get_subnet_info` for each cluster's network
- Uses `get_nsg_info` to compare security configurations
- Analyzes data to identify patterns and differences
- Provides comparative analysis and recommendations

---

### 3. Security Audit Across Environment

**Test Objective**: Perform comprehensive security audit

**Prompt**:
```
Perform a security audit of my AKS environment and report on:
- All network security groups and their rules
- Public vs private cluster configurations
- Network segmentation and isolation
- Potential security vulnerabilities
- Compliance with best practices
- Recommendations for improvement
```

**Expected Behavior**:
- Uses `list_aks_clusters` to identify all clusters
- Uses `get_cluster_info` to check cluster security settings
- Uses `get_nsg_info` for all associated NSGs
- Uses `get_vnet_info` and `get_subnet_info` for network analysis
- Correlates security settings across multiple resources
- Provides security assessment and recommendations

---

### 4. Network Troubleshooting Workflow

**Test Objective**: Diagnose network connectivity issues

**Prompt**:
```
I'm having connectivity issues with my AKS cluster "webapp-cluster". Help me troubleshoot by:
- Examining the cluster's network configuration
- Checking routing tables for proper routes
- Reviewing NSG rules for blocking traffic
- Identifying potential network conflicts
- Providing step-by-step troubleshooting recommendations
```

**Expected Behavior**:
- Uses `get_cluster_info` to understand cluster configuration
- Uses `get_vnet_info` to examine network setup
- Uses `get_subnet_info` to check subnet configuration
- Uses `get_route_table_info` to analyze routing
- Uses `get_nsg_info` to review security rules
- Correlates findings to identify potential issues
- Provides systematic troubleshooting steps

---

### 5. Capacity Planning Analysis

**Test Objective**: Analyze capacity and scaling requirements

**Prompt**:
```
Help me with capacity planning for my AKS clusters by analyzing:
- Current cluster sizes and node configurations
- Network subnet capacity and IP address usage
- Resource distribution across availability zones
- Potential bottlenecks or limitations
- Recommendations for scaling and optimization
```

**Expected Behavior**:
- Uses `list_aks_clusters` to get all clusters
- Uses `get_cluster_info` to examine node pools and scaling settings
- Uses `get_subnet_info` to check IP address space utilization
- Uses `get_vnet_info` to understand network capacity
- Analyzes data to identify capacity constraints
- Provides scaling recommendations

---

### 6. Compliance and Governance Check

**Test Objective**: Validate compliance with organizational policies

**Prompt**:
```
Audit my AKS environment for compliance with our organizational policies:
- All clusters must be in private mode
- Network security groups must block all inbound internet traffic
- Clusters must be deployed in approved regions only
- Subnets must follow naming conventions
- Generate a compliance report with findings and remediation steps
```

**Expected Behavior**:
- Uses `list_aks_clusters` to inventory all clusters
- Uses `get_cluster_info` to check cluster configurations
- Uses `get_nsg_info` to validate security rules
- Uses `get_subnet_info` and `get_vnet_info` for network validation
- Compares findings against stated policies
- Generates detailed compliance report

---

### 7. Cost Optimization Analysis

**Test Objective**: Identify cost optimization opportunities

**Prompt**:
```
Analyze my AKS environment to identify cost optimization opportunities:
- Oversized clusters or node pools
- Unused or underutilized network resources
- Redundant network security configurations
- Opportunities for resource consolidation
- Recommendations for cost reduction
```

**Expected Behavior**:
- Uses `list_aks_clusters` and `get_cluster_info` for cluster analysis
- Uses network tools to identify unused network resources
- Correlates resource usage patterns
- Identifies optimization opportunities
- Provides cost reduction recommendations

---

### 8. Disaster Recovery Assessment

**Test Objective**: Evaluate disaster recovery readiness

**Prompt**:
```
Assess the disaster recovery readiness of my AKS environment:
- Cluster distribution across regions and availability zones
- Network redundancy and failover capabilities
- Cross-region connectivity options
- Potential single points of failure
- Recommendations for improving resilience
```

**Expected Behavior**:
- Uses `list_aks_clusters` to map cluster locations
- Uses `get_cluster_info` to check zone distribution
- Uses network tools to analyze connectivity patterns
- Identifies resilience gaps
- Provides DR improvement recommendations

---

### 9. Migration Planning

**Test Objective**: Plan migration from one configuration to another

**Prompt**:
```
Help me plan migrating my AKS cluster "legacy-cluster" from its current network configuration to a new hub-spoke architecture. Analyze:
- Current network topology and dependencies
- Required changes for hub-spoke model
- Impact on existing connectivity
- Migration steps and considerations
- Risk assessment and mitigation strategies
```

**Expected Behavior**:
- Uses `get_cluster_info` to understand current cluster
- Uses all network tools to map current topology
- Analyzes dependencies and requirements
- Provides detailed migration plan
- Identifies risks and mitigation strategies

---

### 10. Performance Optimization

**Test Objective**: Optimize cluster and network performance

**Prompt**:
```
Optimize the performance of my AKS environment by analyzing:
- Network latency factors (subnet placement, routing efficiency)
- Cluster node distribution and locality
- Network security group rule efficiency
- Bandwidth and throughput considerations
- Recommendations for performance improvements
```

**Expected Behavior**:
- Uses multiple tools to gather comprehensive data
- Analyzes network topology for performance impact
- Identifies performance bottlenecks
- Provides optimization recommendations
- Considers trade-offs between security and performance

---

## Error Handling and Edge Cases

### 11. Partial Data Scenarios

**Test Objective**: Handle scenarios where some data is unavailable

**Prompt**:
```
Analyze my AKS cluster "test-cluster" comprehensively, even if some network resources are not accessible or don't exist.
```

**Expected Behavior**:
- Gracefully handles missing or inaccessible resources
- Provides analysis based on available data
- Clearly indicates what information is missing
- Suggests steps to obtain missing information

---

### 12. Large-Scale Environment

**Test Objective**: Handle environments with many resources

**Prompt**:
```
Provide a summary dashboard of my entire AKS environment with 20+ clusters, showing key metrics and status for each.
```

**Expected Behavior**:
- Efficiently processes large numbers of resources
- Presents data in digestible summary format
- Highlights important issues or outliers
- Maintains performance with large datasets

---

### 13. Conflicting Information

**Test Objective**: Handle scenarios with conflicting or inconsistent data

**Prompt**:
```
Audit my AKS environment and identify any configuration inconsistencies or conflicts between cluster settings and network configurations.
```

**Expected Behavior**:
- Identifies and reports inconsistencies
- Provides context for conflicts
- Suggests resolution approaches
- Maintains data integrity in reporting

---

## Success Criteria

### Functionality Requirements
- ✅ Successfully orchestrates multiple tools in logical sequence
- ✅ Correlates data across different API responses
- ✅ Provides comprehensive analysis and insights
- ✅ Handles complex multi-step reasoning
- ✅ Generates actionable recommendations

### Performance Requirements  
- ✅ Completes complex operations within reasonable time
- ✅ Handles large datasets efficiently
- ✅ Minimizes redundant API calls
- ✅ Provides progress indication for long operations

### Quality Requirements
- ✅ Accurate data correlation and analysis
- ✅ Clear, organized presentation of findings
- ✅ Actionable recommendations with context
- ✅ Proper error handling and graceful degradation
- ✅ Consistent results across multiple runs

### Usability Requirements
- ✅ Natural language interaction for complex requests
- ✅ Appropriate level of detail in responses
- ✅ Clear next steps and recommendations
- ✅ Professional report formatting when requested
