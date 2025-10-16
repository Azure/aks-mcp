# AKS-MCP Project Introduction

## What is AKS-MCP?

AKS-MCP (Azure Kubernetes Service - Model Context Protocol) is a specialized server that enables AI assistants to interact with Azure Kubernetes Service (AKS) clusters through natural language. It acts as a bridge between AI tools (like GitHub Copilot, Claude, and other MCP-compatible assistants) and Azure AKS resources, translating natural language requests into Azure operations and returning structured results.

Built using Go 1.24, AKS-MCP leverages the Model Context Protocol (MCP) to provide seamless communication between AI tools and Azure infrastructure.

## Projects
- [aks-mcp](https://github.com/Azure/aks-mcp): this project
- [mcp-kubernetes](https://github.com/Azure/mcp-kubernetes): subproject that to handle k8s related

## Key Features

### 🚀 Comprehensive AKS Management
- **Cluster Operations**: Create, delete, scale, start, stop, upgrade AKS clusters
- **Node Pool Management**: Add, delete, scale, and upgrade node pools
- **Kubernetes Integration**: Native kubectl, helm, and cilium tool support
- **Multi-cluster Support**: Azure Fleet management for complex scenarios

### 🌐 Network Resource Discovery
- Virtual Networks (VNet), Subnets, Network Security Groups (NSG)
- Route Tables, Load Balancers, Private Endpoints
- Network connectivity diagnostics and analysis

### 📊 Monitoring & Diagnostics
- Azure Monitor integration with KQL query support
- Application Insights telemetry analysis
- Built-in AKS diagnostic detectors across many categories
- Real-time observability with eBPF using Inspektor Gadget

### 🔧 Compute Resources
- Virtual Machine Scale Sets (VMSS) management
- VM operations and runtime status monitoring
- Integration with AKS node pool infrastructure

### 💡 Smart Recommendations
- Azure Advisor integration for cost, security, and performance recommendations
- Automated best practices analysis
- Resource optimization suggestions

### 🔐 Security & Access Control
- Three-tier access control: `readonly`, `readwrite`, `admin`
- Namespace isolation
- OAuth 2.0 authentication support for web-based scenarios

### 📈 Observability & Telemetry
- Built-in telemetry collection with OpenTelemetry integration
- Application Insights integration for usage analytics
- Optional OTLP endpoint support for custom telemetry backends
- Privacy-conscious design with opt-out capability (`AKS_MCP_COLLECT_TELEMETRY=false`)

## How It Works

### Architecture Overview

AKS-MCP is built with a modular, component-based architecture that separates concerns and enables scalable development:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   AI Assistant  │───▶│    AKS-MCP       │───▶│  Azure Cloud    │
│  (Copilot etc.) │    │    Server        │    │   Resources     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   Kubernetes     │
                       │   Clusters       │
                       └──────────────────┘
```

### Core Implementation

**1. Entry Point (`cmd/aks-mcp/main.go`)**
- Application initialization and configuration validation
- Signal handling for graceful shutdown
- Telemetry service initialization
- Error handling and logging setup

**2. Configuration Management (`internal/config/`)**
- Command-line argument parsing with `spf13/pflag`
- Environment variable handling for Azure authentication
- Access level validation and security controls
- Timeout and transport configuration

**3. Server Core (`internal/server/server.go`)**
- MCP server implementation using `mark3labs/mcp-go`
- Component registration and tool discovery
- Request routing and response handling
- Concurrent operation support

**4. Component System (`internal/components/`)**

Each Azure service area is implemented as a separate component:

- **AKS** (`azaks/`): Core cluster operations using Azure CLI
- **Network** (`network/`): VNet, Subnet, NSG operations via Azure SDK
- **Monitor** (`monitor/`): Metrics, logs, and diagnostics
- **Compute** (`compute/`): VMSS and VM management
- **Fleet** (`fleet/`): Multi-cluster Azure Fleet operations
- **Advisor** (`advisor/`): Recommendation engine integration
- **Detectors** (`detectors/`): AppLens diagnostic integration
- **Inspektor Gadget** (`inspektorgadget/`): eBPF observability tools

**5. Azure Integration**

- **SDK Client** (`internal/azureclient/`): Azure resource management with caching
- **CLI Integration** (`internal/azcli/`): Azure CLI command execution for AKS operations
- **Authentication**: Multi-method authentication chain with fallback support
- **OAuth Provider** (`internal/auth/oauth/`): Web-based OAuth 2.0 authentication for browser scenarios

**6. Telemetry & Observability**

- **Telemetry Service** (`internal/telemetry/`): OpenTelemetry-based usage tracking
- **Application Insights**: Microsoft telemetry backend integration
- **Custom OTLP**: Support for custom OpenTelemetry endpoints

**7. Kubernetes Integration**

- **K8s Client** (`internal/k8s/`): Direct Kubernetes API access
- **MCP-Kubernetes** (`github.com/Azure/mcp-kubernetes`): Unified kubectl tool interface
- **Tool Abstraction**: Consistent tool interface across different K8s operations


### Key Technologies Used

- **Go 1.24**: 
- **Azure SDK for Go**: Official Azure resource management
- **MCP Protocol**: `mark3labs/mcp-go` for AI assistant integration
- **Kubernetes Client**: `k8s.io/client-go` for direct cluster access
- **OpenTelemetry**: Distributed tracing and observability
- **Azure CLI**: Fallback for AKS-specific operations
- **eBPF**: Low-level system observability via Inspektor Gadget

### Security Implementation

- **Access Control**: Role-based tool availability (`readonly` → `readwrite` → `admin`)
- **Credential Validation**: Strict validation of authentication tokens and certificates
- **Command Sanitization**: Input validation and command injection prevention
- **Audit Logging**: Comprehensive logging of all operations and access attempts
- **Namespace Isolation**: Optional Kubernetes namespace restrictions

## How to Use AKS-MCP

### Installation Methods

**Development from Source:**
```bash
git clone https://github.com/Azure/aks-mcp.git
cd aks-mcp
make build
./aks-mcp --transport stdio
```

### Configuration Options

```bash
# Access levels
--access-level readonly|readwrite|admin

# Additional tools
--additional-tools helm,cilium,hubble

# Namespace restrictions
--allow-namespaces default,kube-system

# Timeout settings
--timeout 600

# Telemetry configuration
--otlp-endpoint localhost:4317

# Transport options
--transport stdio|sse|streamable-http
```

### Authentication

AKS-MCP supports multiple Azure authentication methods:
1. **Service Principal**: `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`
2. **Workload Identity**: Federated token authentication
3. **Managed Identity**: User-assigned or system-assigned
4. **Azure CLI**: Existing `az login` session
5. **OAuth 2.0**: Web-based authentication for browser scenarios

### Telemetry

Telemetry collection is enabled by default for usage analytics and improvement purposes
