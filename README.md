# AKS-MCP

The AKS-MCP is a Model Context Protocol (MCP) server that enables AI assistants to interact with Azure Kubernetes Service (AKS) clusters. It serves as a bridge between AI tools (like GitHub Copilot, Claude, and other MCP-compatible AI assistants) and AKS, translating natural language requests into AKS operations and returning the results in a format the AI tools can understand.

It allows AI tools to:

- Operate (CRUD) AKS resources
- Retrieve details related to AKS clusters (VNets, Subnets, NSGs, Route Tables, etc.)

## How it works

AKS-MCP connects to Azure using the Azure SDK and provides a set of tools that AI assistants can use to interact with AKS resources. It leverages the Model Context Protocol (MCP) to facilitate this communication, enabling AI tools to make API calls to Azure and interpret the responses.

## How to install

### Local

<details>
<summary>Install prerequisites</summary>

1. Set up [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) and authenticate
```bash
az login
```
</details>

<br/>

Configure your MCP servers in supported AI clients like [GitHub Copilot](https://github.com/features/copilot), [Claude](https://claude.ai/), or other MCP-compatible clients:

```json
{
  "mcpServers": {
    "aks": {
      "command": "<path of binary aks-mcp>",
      "args": [
        "--transport", "stdio",
        "--enabled-components", "azaks,network,advisor"
      ]
    }
  }
}
```

### GitHub Copilot Configuration in VS Code

For GitHub Copilot in VS Code, configure the MCP server in your `.vscode/mcp.json` file:

```json
{
  "servers": {
    "aks-mcp-server": {
      "type": "stdio",
      "command": "<path of binary aks-mcp>",
      "args": [
        "--transport", "stdio",
        "--enabled-components", "azaks,network,compute,advisor"
      ]
    }
  }
}
```

**Component Selection Examples:**
- Minimal configuration (default): `--enabled-components azaks`
- All components: `--enabled-components azaks,network,compute,monitor,advisor,fleet,detectors,kubernetes`

### Options

Command line arguments:

```sh
Usage of ./aks-mcp:
      --access-level string         Access level (readonly, readwrite, admin) (default "readonly")
      --additional-tools string     Comma-separated list of additional Kubernetes tools to support (kubectl is always enabled). Available: helm,cilium
      --allow-namespaces string     Comma-separated list of allowed Kubernetes namespaces (empty means all namespaces)
      --enabled-components string   Comma-separated list of enabled components (azaks is always enabled). Available: azaks,network,compute,monitor,advisor,fleet,detectors,kubernetes (default "azaks")
      --host string                 Host to listen for the server (only used with transport sse or streamable-http) (default "127.0.0.1")
      --port int                    Port to listen for the server (only used with transport sse or streamable-http) (default 8000)
      --timeout int                 Timeout for command execution in seconds, default is 600s (default 600)
      --transport string            Transport mechanism to use (stdio, sse or streamable-http) (default "stdio")
```

**Component Configuration:**
- `azaks`: Core AKS commands and control plane tools (always enabled)
- `network`: VNet, NSG, Load Balancer, and other network-related tools
- `compute`: VMSS and VM-related tools for AKS node pools
- `monitor`: Azure Monitor and Application Insights tools
- `advisor`: Azure Advisor recommendations for AKS clusters
- `fleet`: Azure Fleet management tools
- `detectors`: AKS diagnostic detectors and troubleshooting tools
- `kubernetes`: kubectl, helm, and cilium tools for direct Kubernetes operations

**Environment variables:**
- Standard Azure authentication environment variables are supported (`AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_SUBSCRIPTION_ID`)

## Development

### Building from Source

This project includes a Makefile for convenient development, building, and testing. To see all available targets:

```bash
make help
```

#### Quick Start

```bash
# Build the binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format and lint code
make check

# Build for all platforms
make release
```

#### Common Development Tasks

```bash
# Install dependencies
make deps

# Build and run with --help
make run

# Clean build artifacts
make clean

# Install binary to GOBIN
make install

# Run security scan
make security
```

#### Docker

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

### Manual Build

If you prefer to build without the Makefile:

```bash
go build -o aks-mcp ./cmd/aks-mcp
```

## Usage

Ask any questions about your AKS clusters in your AI client, for example:

```
List all my AKS clusters in my subscription xxx.

What is the network configuration of my AKS cluster?

Show me the network security groups associated with my cluster.
```

## Available Tools

The AKS-MCP server provides the following tools for interacting with AKS clusters:

<details>
<summary>AKS Cluster Management Tools (Read-Only)</summary>

- `az_aks_show`: Show the details of a managed Kubernetes cluster
- `az_aks_list`: List managed Kubernetes clusters
- `az_aks_get-versions`: Get the versions available for creating a managed Kubernetes cluster
- `az_aks_check-network_outbound`: Perform outbound network connectivity check for a node
- `az_aks_nodepool_list`: List node pools in a managed Kubernetes cluster
- `az_aks_nodepool_show`: Show the details for a node pool in the managed Kubernetes cluster
</details>

<details>
<summary>AKS Cluster Management Tools (Read-Write)</summary>

*Available with `--access-level readwrite` or `admin`*

- `az_aks_create`: Create a new managed Kubernetes cluster
- `az_aks_delete`: Delete a managed Kubernetes cluster
- `az_aks_scale`: Scale the node pool in a managed Kubernetes cluster
- `az_aks_update`: Update a managed Kubernetes cluster
- `az_aks_upgrade`: Upgrade a managed Kubernetes cluster to a newer version
- `az_aks_nodepool_add`: Add a node pool to the managed Kubernetes cluster
- `az_aks_nodepool_delete`: Delete a node pool from the managed Kubernetes cluster
- `az_aks_nodepool_scale`: Scale a node pool in a managed Kubernetes cluster
- `az_aks_nodepool_upgrade`: Upgrade a node pool to a newer version
</details>

<details>
<summary>AKS Cluster Management Tools (Admin)</summary>

*Available with `--access-level admin` only*

- `az_aks_get-credentials`: Get access credentials for a managed Kubernetes cluster
</details>

<details>
<summary>Network Tools</summary>

- `get_vnet_info`: Get information about the VNet used by the AKS cluster
- `get_subnet_info`: Get information about the Subnet used by the AKS cluster
- `get_route_table_info`: Get information about the Route Table used by the AKS cluster
- `get_nsg_info`: Get information about the Network Security Group used by the AKS cluster
- `get_load_balancers_info`: Get information about all Load Balancers used by the AKS cluster
- `get_private_endpoint_info`: Get information about the private endpoint used by the AKS cluster
</details>

<details>
<summary>Compute Tools</summary>

- `get_aks_vmss_info`: Get detailed VMSS configuration for node pools in the AKS cluster
- `az_vmss_run-command_invoke`: Execute a command on instances of a Virtual Machine Scale Set (readwrite/admin)
</details>

<details>
<summary>Monitor Tools</summary>

- `az_monitor_metrics_list`: List the metric values for a resource
- `az_monitor_metrics_list-definitions`: List the metric definitions for a resource
- `az_monitor_metrics_list-namespaces`: List the metric namespaces for a resource
- `az_monitor_activity_log_resource_health`: Retrieve resource health events for AKS clusters
- `az_monitor_app_insights_query`: Execute KQL queries against Application Insights telemetry data
</details>

<details>
<summary>AKS Control Plane Tools</summary>

- `aks_control_plane_diagnostic_settings`: Check if AKS cluster has diagnostic settings configured
- `aks_control_plane_logs`: Query AKS control plane logs with safety constraints and time range validation
</details>

<details>
<summary>Fleet Tools</summary>

- `az_fleet`: Execute Azure Fleet commands with structured parameters for AKS Fleet management
  - Supports operations: list, show, create, update, delete, start, stop
  - Supports resources: fleet, member, updaterun, updatestrategy
  - Requires readwrite or admin access for write operations
</details>

<details>
<summary>Detector Tools</summary>

- `list_detectors`: List all available AKS cluster detectors
- `run_detector`: Run a specific AKS detector
- `run_detectors_by_category`: Run all detectors in a specific category
</details>

<details>
<summary>Azure Advisor Tools</summary>

- `az_advisor_recommendation`: Retrieve and manage Azure Advisor recommendations for AKS clusters
</details>

<details>
<summary>Kubernetes Tools</summary>

*Note: kubectl commands are available with all access levels. Additional tools (helm, cilium) require explicit enablement via `--additional-tools`*

**kubectl Commands (Read-Only):**
- `kubectl_get`: Display one or many resources
- `kubectl_describe`: Show details of a specific resource or group of resources
- `kubectl_explain`: Documentation of resources
- `kubectl_logs`: Print the logs for a container in a pod
- `kubectl_api-resources`: Print the supported API resources on the server
- `kubectl_api-versions`: Print the supported API versions on the server
- `kubectl_diff`: Diff live configuration against a would-be applied file
- `kubectl_cluster-info`: Display cluster info
- `kubectl_top`: Display resource usage
- `kubectl_events`: List events in the cluster
- `kubectl_auth`: Inspect authorization

**kubectl Commands (Read-Write/Admin):**
- `kubectl_create`: Create a resource from a file or from stdin
- `kubectl_delete`: Delete resources by file names, stdin, resources and names, or by resources and label selector
- `kubectl_apply`: Apply a configuration to a resource by file name or stdin
- `kubectl_expose`: Take a replication controller, service, deployment or pod and expose it as a new Kubernetes Service
- `kubectl_run`: Run a particular image on the cluster
- `kubectl_set`: Set specific features on objects
- `kubectl_rollout`: Manage the rollout of a resource
- `kubectl_scale`: Set a new size for a Deployment, ReplicaSet, Replication Controller, or StatefulSet
- `kubectl_autoscale`: Auto-scale a Deployment, ReplicaSet, or StatefulSet
- `kubectl_label`: Update the labels on a resource
- `kubectl_annotate`: Update the annotations on a resource
- `kubectl_patch`: Update field(s) of a resource
- `kubectl_replace`: Replace a resource by file name or stdin
- `kubectl_cp`: Copy files and directories to and from containers
- `kubectl_exec`: Execute a command in a container
- `kubectl_cordon`: Mark node as unschedulable
- `kubectl_uncordon`: Mark node as schedulable
- `kubectl_drain`: Drain node in preparation for maintenance
- `kubectl_taint`: Update the taints on one or more nodes
- `kubectl_certificate`: Modify certificate resources

**Additional Tools (Optional):**
- `helm`: Helm package manager for Kubernetes (requires `--additional-tools helm`)
- `cilium`: Cilium CLI for eBPF-based networking and security (requires `--additional-tools cilium`)
</details>

<details>
<summary>Account Management Tools</summary>

- `az_account_list`: List all subscriptions for the authenticated account
- `az_account_set`: Set a subscription as the current active subscription
- `az_login`: Log in to Azure using service principal credentials
</details>

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft
trademarks or logos is subject to and must follow
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
