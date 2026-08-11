# AKS-MCP Prompts Folder

This folder includes all the prompts for the AKS-MCP server. These prompt files are designed for generating the functionality of the AKS-MCP (Azure Kubernetes Service - Model Context Protocol) server with AI assistants.


## Existing Files

### Current Prompt Files

- **`README.md`** - This file, describing the prompts folder and its contents
- **`monitoringservice.md`** - Feature requirements and implementation details for MCP monitoring service integration
- **`azure-diagnostics.md`** - Implementation specifications for Azure diagnostic and advisory tools (AppLens detectors, Resource Health, Azure Advisor)
- **`azure-advisor.md`** - Simplified implementation specifications for Azure Advisor recommendations integration
- **`azure-cli-tools.md`** - Azure CLI tools integration and command execution specifications
- **`azure-network-tools.md`** - Azure network resource tools and information retrieval specifications
- **`azure-resource-caching.md`** - Azure resource caching system implementation specifications
- **`azure-monitor-tools.md`** - Azure Monitor metrics and monitoring capabilities implementation specifications

### File Structure

```
prompts/
├── README.md                    # This documentation file
├── monitoringservice.md        # Monitoring service integration requirements
├── azure-diagnostics.md        # Azure diagnostics and advisory tools specifications
├── azure-advisor.md            # Azure Advisor recommendations integration
├── azure-cli-tools.md          # Azure CLI tools integration specifications
├── azure-network-tools.md      # Azure network resource tools specifications
├── azure-resource-caching.md   # Azure resource caching system specifications
└── azure-monitor-tools.md      # Azure Monitor metrics and monitoring specifications
```

## AKS-MCP Server Capabilities

The prompts in this folder are designed to test and validate the following AKS-MCP server capabilities:

### Core Features (Currently Implemented)

#### Azure CLI Tools Integration
- Azure CLI (`az aks`) command execution through MCP tools
- Support for read-only, read-write, and admin access levels
- Individual command registration with security validation
- Account management commands (login, account set, list subscriptions)

#### Azure Resource Information Tools
- Virtual Network (VNet) information and configuration retrieval
- Network Security Group (NSG) rules and policies access
- Route table information and network routing details
- Subnet details and IP address management
- Load balancer configuration access (external and internal)

#### Security and Access Control System
- Three-tier access control (readonly, readwrite, admin)
- Command injection protection and security validation
- Access level enforcement at server and tool level
- Operation categorization and permission management

#### MCP Server Framework
- Model Context Protocol server implementation
- Stdio transport support
- Dynamic tool registration based on access level
- AI assistant integration (VS Code Copilot, Claude, etc.)

#### Azure Resource Caching System
- In-memory caching for Azure resources and API responses
- Configurable cache timeouts and automatic expiration
- Thread-safe cache operations with performance optimization
- Multi-subscription cache management

### Future Planned Features

#### Monitoring and Observability (In Development)
- Azure Monitor integration and dashboard access
- Log Analytics workspace queries and log retrieval
- Application Insights performance monitoring and tracing
- Alert management and notification systems
- Performance metrics collection and analysis
- Real-time monitoring and data visualization

### Diagnostics and Advisory
- AppLens detector integration and execution
- Resource Health event monitoring and access
- Azure Advisor recommendations retrieval (simplified implementation in azure-advisor.md)
- Automated diagnostic workflows and reporting
- Proactive issue detection and remediation guidance

### Security and Access Control
- Access level validation (readonly, readwrite, admin)
- Security policy enforcement and validation
- Authentication and authorization testing
- Role-based access control (RBAC) verification

## Adding New Prompt Files

When adding new prompt files to this folder:

1. **Follow naming conventions**: Use descriptive, lowercase names with hyphens (e.g., `cluster-operations.md`)
2. **Include clear objectives**: Each file should specify what functionality it tests
3. **Provide expected behaviors**: Document what responses are expected for each prompt
4. **Add prerequisites**: List any required setup, permissions, or resources
5. **Update this README**: Add the new file to the "Existing Files" section above
