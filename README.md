# AKS-MCP

AKS-MCP is a Model Context Protocol (MCP) server that enables AI assistants to interact with Azure Kubernetes Service (AKS) clusters. It serves as a bridge between AI tools (like GitHub Copilot, Claude, and other MCP-compatible AI assistants) and AKS, translating natural language requests into AKS operations and returning the results in a format the AI tools can understand.

## Key Capabilities

- **AKS Cluster Management**: Create, manage, and monitor AKS clusters and node pools
- **Network Resources**: Retrieve details about VNets, Subnets, NSGs, Route Tables, and Load Balancers
- **Fleet Operations**: Manage Azure Fleet for multi-cluster scenarios
- **Real-time Observability**: Monitor clusters with eBPF-based tools like Inspektor Gadget
- **Kubernetes Operations**: Execute kubectl commands and manage Kubernetes resources
- **Diagnostics & Monitoring**: Access Azure Monitor, Application Insights, and diagnostic detectors
- **Azure Advisor**: Get recommendations for optimizing your AKS clusters

## How it works

AKS-MCP connects to Azure using the Azure SDK and provides a comprehensive set of tools that AI assistants can use to interact with AKS resources. It leverages the Model Context Protocol (MCP) to facilitate seamless communication, enabling AI tools to make API calls to Azure and interpret the responses naturally.

## Available Tools

The AKS-MCP server provides consolidated tools for interacting with AKS
clusters. Some tools will require read-write or admin permissions to run debugging pods on your cluster. To enable read-write or admin permissions for the AKS-MCP server, add the **access level** parameter to your MCP configuration file:

1. Navigate to your **mcp.json** file, or go to MCP: List Servers -> AKS-MCP -> Show Configuration Details in the **Command Palette** (`Ctrl+Shift+P` on Windows/Linux or `Cmd+Shift+P` on macOS).
2. In the "args" section of AKS-MCP, add the following parameters: "--access-level", "readwrite" / "admin"

For example:
```
"args": [
  "--transport",
  "stdio",
  "--access-level",
  "readwrite"
]
```

These tools have been designed to provide comprehensive functionality
through unified interfaces:

<details>
<summary>AKS Cluster Management</summary>

**Tool:** `az_aks_operations`

Unified tool for managing Azure Kubernetes Service (AKS) clusters and related operations.

**Available Operations:**

- **Read-Only** (all access levels):
  - `show`: Show cluster details
  - `list`: List clusters in subscription/resource group
  - `get-versions`: Get available Kubernetes versions
  - `check-network`: Perform outbound network connectivity check
  - `nodepool-list`: List node pools in cluster
  - `nodepool-show`: Show node pool details
  - `account-list`: List Azure subscriptions

- **Read-Write** (`readwrite`/`admin` access levels):
  - `create`: Create new cluster
  - `delete`: Delete cluster
  - `scale`: Scale cluster node count
  - `start`: Start a stopped cluster
  - `stop`: Stop a running cluster
  - `update`: Update cluster configuration
  - `upgrade`: Upgrade Kubernetes version
  - `nodepool-add`: Add node pool to cluster
  - `nodepool-delete`: Delete node pool
  - `nodepool-scale`: Scale node pool
  - `nodepool-upgrade`: Upgrade node pool
  - `account-set`: Set active subscription
  - `login`: Azure authentication

- **Admin-Only** (`admin` access level):
  - `get-credentials`: Get cluster credentials for kubectl access

</details>

<details>
<summary>Network Resource Management</summary>

**Tool:** `az_network_resources`

Unified tool for getting Azure network resource information used by AKS clusters.

**Available Resource Types:**

- `all`: Get information about all network resources
- `vnet`: Virtual Network information
- `subnet`: Subnet information
- `nsg`: Network Security Group information
- `route_table`: Route Table information
- `load_balancer`: Load Balancer information
- `private_endpoint`: Private endpoint information

</details>

<details>
<summary>Monitoring and Diagnostics</summary>

**Tool:** `az_monitoring`

Unified tool for Azure monitoring and diagnostics operations for AKS clusters.

**Available Operations:**

- `metrics`: List metric values for resources
- `resource_health`: Retrieve resource health events for AKS clusters
- `app_insights`: Execute KQL queries against Application Insights telemetry data
- `diagnostics`: Check if AKS cluster has diagnostic settings configured
- `control_plane_logs`: Query AKS control plane logs with safety constraints
  and time range validation

</details>

<details>
<summary>Compute Resources</summary>

**Tool:** `get_aks_vmss_info`

- Get detailed VMSS configuration for node pools in the AKS cluster

**Tool:** `az_vmss_run-command_invoke` *(readwrite/admin only)*

- Execute commands on Virtual Machine Scale Set instances

</details>

<details>
<summary>Fleet Management</summary>

**Tool:** `az_fleet`

Comprehensive Azure Fleet management for multi-cluster scenarios.

**Available Operations:**

- **Fleet Operations**: list, show, create, update, delete, get-credentials
- **Member Operations**: list, show, create, update, delete
- **Update Run Operations**: list, show, create, start, stop, delete
- **Update Strategy Operations**: list, show, create, delete
- **ClusterResourcePlacement Operations**: list, show, get, create, delete

Supports both Azure Fleet management and Kubernetes ClusterResourcePlacement
CRD operations.

</details>

<details>
<summary>Diagnostic Detectors</summary>

**Tool:** `list_detectors`

- List all available AKS cluster detectors

**Tool:** `run_detector`

- Run a specific AKS diagnostic detector

**Tool:** `run_detectors_by_category`

- Run all detectors in a specific category
- **Categories**: Best Practices, Cluster and Control Plane Availability and
  Performance, Connectivity Issues, Create/Upgrade/Delete and Scale,
  Deprecations, Identity and Security, Node Health, Storage

</details>

<details>
<summary>Azure Advisor</summary>

**Tool:** `az_advisor_recommendation`

Retrieve and manage Azure Advisor recommendations for AKS clusters.

**Available Operations:**

- `list`: List recommendations with filtering options
- `report`: Generate recommendation reports
- **Filter Options**: resource_group, cluster_names, category (Cost,
  HighAvailability, Performance, Security), severity (High, Medium, Low)

</details>

<details>
<summary>Kubernetes Tools</summary>

*Note: kubectl commands are available with all access levels. Additional tools
require explicit enablement via `--additional-tools`*

**kubectl Commands (Read-Only):**

- `kubectl_get`, `kubectl_describe`, `kubectl_explain`, `kubectl_logs`
- `kubectl_api-resources`, `kubectl_api-versions`, `kubectl_diff`
- `kubectl_cluster-info`, `kubectl_top`, `kubectl_events`, `kubectl_auth`

**kubectl Commands (Read-Write/Admin):**

- `kubectl_create`, `kubectl_delete`, `kubectl_apply`, `kubectl_expose`,
  `kubectl_run`
- `kubectl_set`, `kubectl_rollout`, `kubectl_scale`, `kubectl_autoscale`
- `kubectl_label`, `kubectl_annotate`, `kubectl_patch`, `kubectl_replace`
- `kubectl_cp`, `kubectl_exec`, `kubectl_cordon`, `kubectl_uncordon`
- `kubectl_drain`, `kubectl_taint`, `kubectl_certificate`

**Additional Tools (Optional):**

- `helm`: Helm package manager (requires `--additional-tools helm`)
- `cilium`: Cilium CLI for eBPF networking (requires `--additional-tools cilium`)

</details>

<details>
<summary>Real-time Observability</summary>

**Tool:** `inspektor_gadget_observability`

Real-time observability tool for Azure Kubernetes Service (AKS) clusters using
eBPF.

**Available Actions:**

- `deploy`: Deploy Inspektor Gadget to cluster (requires `readwrite`/`admin` access)
- `undeploy`: Remove Inspektor Gadget from cluster (requires `readwrite`/`admin` access)
- `is_deployed`: Check deployment status
- `run`: Run one-shot gadgets
- `start`: Start continuous gadgets
- `stop`: Stop running gadgets
- `get_results`: Retrieve gadget results
- `list_gadgets`: List available gadgets

**Available Gadgets:**

- `observe_dns`: Monitor DNS requests and responses
- `observe_tcp`: Monitor TCP connections
- `observe_file_open`: Monitor file system operations
- `observe_process_execution`: Monitor process execution
- `observe_signal`: Monitor signal delivery
- `observe_system_calls`: Monitor system calls
- `top_file`: Top files by I/O operations
- `top_tcp`: Top TCP connections by traffic

</details>

## How to install

### Prerequisites

1. Set up [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) and authenticate:

   ```bash
   az login
   ```

### VS Code with GitHub Copilot (Recommended)

#### 🚀 One-Click Installation with the AKS Extension

The easiest way to get started with AKS-MCP is through the **Azure Kubernetes Service Extension for VS Code**.

#### Step 1: Install the AKS Extension

1. Open VS Code and go to Extensions (`Ctrl+Shift+X` on Windows/Linux or `Cmd+Shift+X` on macOS).
1. Search for [Azure Kubernetes Service](https://marketplace.visualstudio.com/items?itemName=ms-kubernetes-tools.vscode-aks-tools).
1. Install the official Microsoft AKS extension.

#### Step 2: Launch the AKS-MCP Server

1. Open the **Command Palette** (`Ctrl+Shift+P` on Windows/Linux or `Cmd+Shift+P` on macOS).
2. Search and run: **AKS: Setup AKS MCP Server**.

Upon successful installation, the server will now be visible in **MCP: List Servers** (via Command Palette). From there, you can start the MCP server or view its status.

#### Step 3: Start Using AKS-MCP

Once started, the MCP server will appear in the **Copilot Chat: Configure Tools** dropdown under `MCP Server: AKS MCP`, ready to enhance contextual prompts based on your AKS environment. By default, all AKS-MCP server tools are enabled. You can review the list of available tools and disable any that are not required for your specific scenario.

Try a prompt like *"List all my AKS clusters"*, which will start using tools from the AKS-MCP server.

Note:
For VS Code that is runing inside of WSL environment, starting AKS MCP server could fail with error "Connection state: Error spawn /home/path/.vs-kubernetes/tools/aks-mcp/v0.0.3/aks-mcp ENOENT", it means VS code cannot find this file. The resolution is modifying the mcp.json file as following to make it running with wsl.

```json
"AKS MCP": {
	"command": "wsl",
	"args": [
		"<aks-mcp path>",
		"--transport",
		"stdio"
	]
}
```


> **💡 Benefits**: The AKS extension handles binary downloads, updates, and configuration automatically, ensuring you always have the latest version with optimal settings.

### Alternative Installation Methods

<details>
<summary>Manual Binary Installation</summary>

#### Step 1: Download the Binary

Choose your platform and download the latest AKS-MCP binary:

| Platform | Architecture | Download Link |
|----------|-------------|---------------|
| **Windows** | AMD64 | [📥 aks-mcp-windows-amd64.exe](https://github.com/Azure/aks-mcp/releases/latest/download/aks-mcp-windows-amd64.exe) |
| | ARM64 | [📥 aks-mcp-windows-arm64.exe](https://github.com/Azure/aks-mcp/releases/latest/download/aks-mcp-windows-arm64.exe) |
| **macOS** | Intel (AMD64) | [📥 aks-mcp-darwin-amd64](https://github.com/Azure/aks-mcp/releases/latest/download/aks-mcp-darwin-amd64) |
| | Apple Silicon (ARM64) | [📥 aks-mcp-darwin-arm64](https://github.com/Azure/aks-mcp/releases/latest/download/aks-mcp-darwin-arm64) |
| **Linux** | AMD64 | [📥 aks-mcp-linux-amd64](https://github.com/Azure/aks-mcp/releases/latest/download/aks-mcp-linux-amd64) |
| | ARM64 | [📥 aks-mcp-linux-arm64](https://github.com/Azure/aks-mcp/releases/latest/download/aks-mcp-linux-arm64) |

#### Step 2: Configure VS Code

After downloading, create a `.vscode/mcp.json` file in your workspace root with the path to your downloaded binary.

##### Option A: Automated Setup Script

For quick setup, you can use these one-liner scripts that download the binary
and create the configuration:

*Windows (PowerShell):*

```powershell
# Download binary and create VS Code configuration
mkdir -p .vscode ; Invoke-WebRequest -Uri "https://github.com/Azure/aks-mcp/releases/latest/download/aks-mcp-windows-amd64.exe" -OutFile "aks-mcp.exe" ; @{servers=@{"aks-mcp-server"=@{type="stdio";command="$PWD\aks-mcp.exe";args=@("--transport","stdio")}}} | ConvertTo-Json -Depth 3 | Out-File ".vscode/mcp.json" -Encoding UTF8
```

*macOS/Linux (Bash):*

```bash
# Download binary and create VS Code configuration
mkdir -p .vscode && curl -sL https://github.com/Azure/aks-mcp/releases/latest/download/aks-mcp-linux-amd64 -o aks-mcp && chmod +x aks-mcp && echo '{"servers":{"aks-mcp-server":{"type":"stdio","command":"'$PWD'/aks-mcp","args":["--transport","stdio"]}}}' > .vscode/mcp.json
```

##### Option B: Manual Configuration

> **✨ Simple Setup**: Download the binary for your platform, then use the manual configuration below to set up the MCP server in VS Code.

#### Manual VS Code Configuration

You can configure the AKS-MCP server in two ways:

**1. Workspace-specific configuration** (recommended for project-specific usage):

Create a `.vscode/mcp.json` file in your workspace with the path to your downloaded binary:

```json
{
  "servers": {
    "aks-mcp-server": {
      "type": "stdio",
      "command": "<enter the file path>",
      "args": [
        "--transport", "stdio"
      ]
    }
  }
}
```

**2. User-level configuration** (persistent across all workspaces):

For a persistent configuration that works across all your VS Code workspaces, add the MCP server to your VS Code user settings:

1. Open VS Code Settings (Ctrl+, or Cmd+,)
2. Search for "mcp" in the settings
3. Add the following to your User Settings JSON:

```json
{
  "github.copilot.chat.mcp.servers": {
    "aks-mcp-server": {
      "type": "stdio",
      "command": "<enter the file path>",
      "args": [
        "--transport", "stdio"
      ]
    }
  }
}
```

#### Step 3: Load the AKS-MCP server tools to Github Copilot

1. If running on an older version of VS Code: restart VS Code i.e. close and
   reopen VS Code to load the new MCP server configuration.
2. Open GitHub Copilot in VS Code and [switch to Agent mode](https://code.visualstudio.com/docs/copilot/chat/chat-agent-mode)
3. Click the **Tools** button or run /list in the Github Copilot window to see the list of available tools
4. You should see the AKS-MCP tools in the list
5. Try a prompt like: *"List all my AKS clusters in subscription xxx"*
6. The agent will automatically use AKS-MCP tools to complete your request

> **💡 Tip**: If you don't see the AKS-MCP tools after restarting, check the VS Code output panel for any MCP server connection errors and verify your binary path in `.vscode/mcp.json`.

**Note**: Ensure you have authenticated with Azure CLI (`az login`) for the server to access your Azure resources.

</details>

### Other MCP-Compatible Clients

<details>
<summary>Docker and Custom Client Installation</summary>

For other MCP-compatible AI clients like [Claude Desktop](https://claude.ai/), configure the server in your MCP configuration:

```json
{
  "mcpServers": {
    "aks": {
      "command": "<path of binary aks-mcp>",
      "args": [
        "--transport", "stdio"
      ]
    }
  }
}
```

#### 🐋 Docker MCP configuration

For containerized deployment, you can run AKS-MCP server using the official Docker image:

Option A: Mount credentials from host (recommended):

```json
{
  "mcpServers": {
    "aks": {
      "type": "stdio",
      "command": "docker",
      "args": [
          "run",
          "-i",
          "--rm",
          "--user",
          "<your-user-id (e.g. id -u)>",
          "-v",
          "~/.azure:/home/mcp/.azure",
          "-v",
          "~/.kube:/home/mcp/.kube",
          "ghcr.io/azure/aks-mcp:latest",
          "--transport",
          "stdio"
        ]
    }
  }
}
```

Option B: fetch the credentials inside the container:

```json
{
  "mcpServers": {
    "aks": {
      "type": "stdio",
      "command": "docker",
      "args": [
          "run",
          "-i",
          "--rm",
          "ghcr.io/azure/aks-mcp:latest",
          "--transport",
          "stdio"
        ]
    }
  }
}
```

Start the MCP server container first per above command, and then run the following commands to fetch the credentials:
- Login to Azure CLI: `docker exec -it <container-id> az login --use-device-code`
- Get kubeconfig: `docker exec -it <container-id> az aks get-credentials -g <resource-group> -n <cluster-name>`

Note that:

- Host Azure CLI logins don't automatically propagate into containers without mounting `~/.azure`.
- User ID should be set for option A, orelse the mcp user inside container won't be able to access the mounted files.

#### 🤖 Custom MCP Client Installation

You can configure any MCP-compatible client to use the AKS-MCP server by running the binary directly:

```bash
# Run the server directly
./aks-mcp --transport stdio
```

#### 🔧 Manual Binary Installation

For direct binary usage without package managers:

1. Download the latest release from the [releases page](https://github.com/Azure/aks-mcp/releases)
2. Extract the binary to your preferred location
3. Make it executable (on Unix systems):
   ```bash
   chmod +x aks-mcp
   ```
4. Configure your MCP client to use the binary path

</details>

## Configuration Options

### Command Line Arguments

```sh
Usage of ./aks-mcp:
      --access-level string       Access level (readonly, readwrite, admin) (default "readonly")
      --additional-tools string   Comma-separated list of additional Kubernetes tools to support (kubectl is always enabled). Available: helm,cilium
      --allow-namespaces string   Comma-separated list of allowed Kubernetes namespaces (empty means all namespaces)
      --host string               Host to listen for the server (only used with transport sse or streamable-http) (default "127.0.0.1")
      --otlp-endpoint string      OTLP endpoint for OpenTelemetry traces (e.g. localhost:4317, default "")
      --port int                  Port to listen for the server (only used with transport sse or streamable-http) (default 8000)
      --timeout int               Timeout for command execution in seconds, default is 600s (default 600)
      --transport string          Transport mechanism to use (stdio, sse or streamable-http) (default "stdio")
  -v, --verbose                   Enable verbose logging
```

### Environment Variables

Standard Azure authentication environment variables are supported:
- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID` 
- `AZURE_CLIENT_SECRET`
- `AZURE_SUBSCRIPTION_ID`

## Getting Started

Try asking your AI assistant questions like:

```
List all my AKS clusters in subscription xxx.

What is the network configuration of my AKS cluster?

Show me the network security groups associated with my cluster.

Create a new Azure Fleet named prod-fleet in eastus region.

List all members in my fleet.

Create a placement to deploy nginx workloads to clusters with app=frontend label.

Show me all ClusterResourcePlacements in my fleet.
```

## Development

For development, building from source, and advanced configuration options, see [DEVELOPMENT.md](DEVELOPMENT.md).

## Telemetry

Telemetry collection is on by default. To opt out, set the environment variable `AKS_MCP_COLLECT_TELEMETRY=false`.

## Contributing

This project welcomes contributions and suggestions. Most contributions require you to agree to a
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
