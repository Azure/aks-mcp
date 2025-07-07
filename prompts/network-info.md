# Network Information Tests

## Test Objective
Validate network-related queries and information retrieval for AKS clusters.

## Prerequisites
- AKS clusters with various network configurations
- Clusters with VNets, subnets, NSGs, and route tables
- Appropriate permissions to read network resources

## Test Prompts

### Virtual Network Tests

#### Test 1: Get VNet Information
**Prompt:**
```
Show me the virtual network configuration for the AKS cluster "my-cluster" in resource group "my-rg".
```

**Expected Outcome:**
- VNet name and resource ID
- Address space information
- Associated subnets
- Location and resource group

#### Test 2: Multiple Clusters VNet Info
**Prompt:**
```
Compare the virtual network configurations of all AKS clusters in resource group "test-rg".
```

**Expected Outcome:**
- VNet information for each cluster
- Clear comparison or listing format
- Identification of shared vs dedicated VNets

### Subnet Configuration Tests

#### Test 3: Subnet Details
**Prompt:**
```
What subnet is being used by the node pools in cluster "production-cluster"?
```

**Expected Outcome:**
- Subnet name and address range
- Available IP addresses
- Associated route table and NSG
- Service endpoints if configured

#### Test 4: Subnet IP Availability
**Prompt:**
```
Check the IP address availability in the subnet used by cluster "test-cluster".
```

**Expected Outcome:**
- Total IP addresses in subnet
- Used vs available IPs
- Potential for cluster scaling

### Network Security Group Tests

#### Test 5: NSG Rules Analysis
**Prompt:**
```
Show me the network security group rules for cluster "secure-cluster" in resource group "security-rg".
```

**Expected Outcome:**
- NSG name and associated rules
- Inbound and outbound security rules
- Priority and action for each rule
- Source and destination information

#### Test 6: Security Recommendations
**Prompt:**
```
Analyze the network security configuration of cluster "web-cluster" and identify any potential security concerns.
```

**Expected Outcome:**
- Current security configuration summary
- Identification of overly permissive rules
- Recommendations for improvement (if applicable)

### Route Table Tests

#### Test 7: Route Table Configuration
**Prompt:**
```
What route table is associated with the AKS cluster "backend-cluster" and what routes are defined?
```

**Expected Outcome:**
- Route table name and resource ID
- All defined routes with destinations
- Next hop information
- Route priorities

#### Test 8: Custom Routes Impact
**Prompt:**
```
Explain how the custom routes in cluster "custom-net-cluster" might affect traffic flow.
```

**Expected Outcome:**
- Analysis of custom route effects
- Traffic flow implications
- Potential connectivity impacts

### Network Plugin Tests

#### Test 9: CNI Configuration
**Prompt:**
```
What network plugin is being used by cluster "cni-test-cluster" and what are its configuration details?
```

**Expected Outcome:**
- Network plugin type (Azure CNI, kubenet, etc.)
- Pod CIDR configuration
- Service CIDR information
- DNS configuration

#### Test 10: Network Policy
**Prompt:**
```
Check if network policies are enabled for cluster "policy-cluster" and show the configuration.
```

**Expected Outcome:**
- Network policy engine (if enabled)
- Configuration details
- Policy enforcement status

### Load Balancer Tests

#### Test 11: Load Balancer Configuration
**Prompt:**
```
Show me the load balancer configuration for cluster "lb-cluster" including any public IPs.
```

**Expected Outcome:**
- Load balancer type and SKU
- Public IP addresses
- Backend pool configuration
- Health probe settings

#### Test 12: Ingress Configuration
**Prompt:**
```
What ingress controller is configured for cluster "ingress-cluster" and what are its network settings?
```

**Expected Outcome:**
- Ingress controller type
- Associated load balancer or application gateway
- Public endpoints and SSL configuration

### Connectivity Tests

#### Test 13: Outbound Connectivity Check
**Prompt:**
```
Perform an outbound network connectivity check for cluster "connectivity-test" to verify internet access.
```

**Expected Outcome:**
- Connectivity test results
- Accessible external endpoints
- Any blocked or failed connections
- Network latency information

#### Test 14: Cross-Region Connectivity
**Prompt:**
```
Check if cluster "east-cluster" can communicate with cluster "west-cluster" across regions.
```

**Expected Outcome:**
- Cross-region connectivity status
- Network path analysis
- Potential connectivity issues
- Latency measurements

### DNS and Service Discovery

#### Test 15: DNS Configuration
**Prompt:**
```
Show the DNS configuration for cluster "dns-cluster" including any custom DNS settings.
```

**Expected Outcome:**
- DNS service IP and configuration
- Custom DNS servers (if configured)
- DNS resolution testing results
- CoreDNS configuration details

## Validation Checklist
- [ ] VNet information is accurately retrieved
- [ ] Subnet details include all relevant data
- [ ] NSG rules are properly displayed
- [ ] Route table information is complete
- [ ] Network plugin details are correct
- [ ] Load balancer configuration is shown
- [ ] Connectivity tests provide useful results
- [ ] DNS configuration is properly reported
- [ ] Error handling for network resource access
- [ ] Performance is acceptable for network queries
