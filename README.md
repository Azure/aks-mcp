# AKS-MCP

The AKS-MCP is a Model Context Protocol (MCP) server that enables AI assistants
to interact with Azure Kubernetes Service (AKS) clusters. It serves as a bridge
between AI tools (like GitHub Copilot, Claude, and other MCP-compatible AI
assistants) and AKS, translating natural language requests into AKS operations
and returning the results in a format the AI tools can understand.

It allows AI tools to:

- Operate (CRUD) AKS resources
- Retrieve details related to AKS clusters (VNets, Subnets, NSGs, Route Tables, etc.)
- Manage Azure Fleet operations for multi-cluster scenarios

## How it works

AKS-MCP connects to Azure using the Azure SDK and provides a set of tools that
AI assistants can use to interact with AKS resources. It leverages the Model
Context Protocol (MCP) to facilitate this communication, enabling AI tools to
make API calls to Azure and interpret the responses.

## Azure CLI Authentication

AKS-MCP uses Azure CLI (az) for AKS operations. Azure CLI authentication is attempted in this order:

1. Service Principal (client secret): When `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID` environment variables are present, a service principal login is performed using the following command: `az login --service-principal -u CLIENT_ID -p CLIENT_SECRET --tenant TENANT_ID`

1. Workload Identity (federated token): When `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_FEDERATED_TOKEN_FILE` environment variables are present, a federated token login is performed using the following command: `az login --service-principal -u CLIENT_ID --tenant TENANT_ID --federated-token TOKEN`

1. User-assigned Managed Identity (managed identity client ID): When only `AZURE_CLIENT_ID` environment variable is present, a user-assigned managed identity login is performed using the following command: `az login --identity -u CLIENT_ID`

1. System-assigned Managed Identity: When `AZURE_MANAGED_IDENTITY` is set to `system`, a system-assigned managed identity login is performed using the following command: `az login --identity`

1. Existing Login: When none of the above environment variables are set, AKS-MCP assumes you have already authenticated (for example, via `az login`) and uses the existing session.

Optional subscription selection:

- If `AZURE_SUBSCRIPTION_ID` is set, AKS-MCP will run `az account set --subscription SUBSCRIPTION_ID` after login.

Notes and security:

- The federated token file must be exactly `/var/run/secrets/azure/tokens/azure-identity-token` and is strictly validated; other paths are rejected.
- After each login, AKS-MCP verifies authentication with `az account show --query id -o tsv`.
- Ensure the Azure CLI is installed and on PATH.

Environment variables used:

- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`
- `AZURE_FEDERATED_TOKEN_FILE`
- `AZURE_SUBSCRIPTION_ID`
- `AZURE_MANAGED_IDENTITY` (set to `system` to opt into system-assigned managed identity)

## Available Tools

The AKS-MCP server provides consolidated tools for interacting with AKS
clusters. By default, the server uses **unified tools** (`call_az` for Azure operations and `call_kubectl` for Kubernetes operations) which provide a more flexible interface. For backward compatibility, you can enable **legacy specialized tools** by setting the environment variable `USE_LEGACY_TOOLS=true`.

Some tools will require read-write or admin permissions to run debugging pods on your cluster. To enable read-write or admin permissions for the AKS-MCP server, add the **access level** parameter to your MCP configuration file:

1. Navigate to your **mcp.json** file, or go to MCP: List Servers -> AKS-MCP -> Show Configuration Details in the **Command Palette** (For VSCode; `Ctrl+Shift+P` on Windows/Linux or `Cmd+Shift+P` on macOS).
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
<summary>Azure CLI Operations (Unified Tool)</summary>

**Tool:** `call_az` *(default, available when `USE_LEGACY_TOOLS` is not set or set to `false`)*

Unified tool for executing Azure CLI commands directly. This tool provides a flexible interface to run any Azure CLI command.

**Parameters:**
- `cli_command`: The complete Azure CLI command to execute (e.g., `az aks list --resource-group myRG`, `az vm list --subscription <sub-id>`)
- `timeout`: Optional timeout in seconds (default: 120)

**Example Usage:**
```json
{
  "cli_command": "az aks list --resource-group myResourceGroup --output json"
}
```

**Access Control:**
- **readonly**: Only read operations are allowed
- **readwrite/admin**: Both read and write operations are allowed

**Important:** Commands must be simple Azure CLI invocations without shell features like pipes (|), redirects (>, <), command substitution, or semicolons (;).

</details>

<details>
<summary>AKS Cluster Management (Legacy Tool)</summary>

**Tool:** `az_aks_operations` *(available when `USE_LEGACY_TOOLS=true`)*

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

**Tool:** `aks_network_resources`

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

**Tool:** `aks_monitoring`

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

**Tool:** `az_compute_operations`

Unified tool for managing Azure Virtual Machines (VMs) and Virtual Machine Scale Sets (VMSS) used by AKS.

**Available Operations:**

- `show`: Get details of a VM/VMSS
- `list`: List VMs/VMSS in subscription or resource group
- `get-instance-view`: Get runtime status
- `start`: Start VM
- `stop`: Stop VM
- `restart`: Restart VM/VMSS instances
- `reimage`: Reimage VMSS instances (VM not supported for reimage)

**Resource Types:** `vm` (single virtual machines), `vmss` (virtual machine scale sets)

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
<summary>Kubernetes Operations</summary>

*Note: All Kubernetes tools (kubectl, helm, cilium, hubble) are enabled by default. Use `--enabled-components` to selectively enable specific components.*

### Unified kubectl Tool (Default)

**Tool:** `call_kubectl` *(default, available when `USE_LEGACY_TOOLS` is not set or set to `false`)*

Unified tool for executing kubectl commands directly. This tool provides a flexible interface to run any `kubectl` command with full argument support.

**Parameters:**
- `args`: The kubectl command arguments (e.g., `get pods`, `describe node mynode`, `apply -f deployment.yaml`)

**Example Usage:**
```json
{
  "args": "get pods -n kube-system -o wide"
}
```

**Access Control:** Operations are restricted based on the configured access level:
- **readonly**: Only read operations (get, describe, logs, etc.) are allowed
- **readwrite/admin**: All operations including mutating commands (create, delete, apply, etc.)

### Legacy kubectl Tools (Specialized)

**Available when `USE_LEGACY_TOOLS=true`:**

- **Read-Only** (all access levels):
  - `kubectl_resources`: View resources (get, describe) - filtered to read-only operations in readonly mode
  - `kubectl_diagnostics`: Debug and diagnose (logs, events, top, exec, cp)
  - `kubectl_cluster`: Cluster information (cluster-info, api-resources, api-versions, explain)
  - `kubectl_config`: Configuration management (diff, auth, config) - filtered to read-only operations in readonly mode

- **Read-Write/Admin** (`readwrite`/`admin` access levels):
  - `kubectl_resources`: Full resource management (get, describe, create, delete, apply, patch, replace, cordon, uncordon, drain, taint)
  - `kubectl_workloads`: Workload lifecycle (run, expose, scale, autoscale, rollout)
  - `kubectl_metadata`: Metadata management (label, annotate, set)
  - `kubectl_config`: Full configuration management (diff, auth, certificate, config)

### Helm

**Tool:** `call_helm`

Helm package manager for Kubernetes.

### Cilium

**Tool:** `call_cilium`

Cilium CLI for eBPF-based networking and security.

### Hubble

**Tool:** `call_hubble`

Hubble network observability for Cilium.

</details>

<details>
<summary>Real-time Observability</summary>

**Tool:** `inspektor_gadget_observability`

Real-time observability tool for Azure Kubernetes Service (AKS) clusters using
eBPF.

**Available Actions:**

- `deploy`: Deploy Inspektor Gadget to cluster
- `undeploy`: Remove Inspektor Gadget from cluster
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
- `tcpdump`: Capture network packets

</details>

## How to install

### Prerequisites

1. Set up [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) and authenticate:

   ```bash
   az login
   ```

### VS Code with GitHub Copilot (Recommended)

<details>
<summary> One-Click Installation with the AKS Extension </summary>

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

#### WSL Configuration

The MCP configuration differs depending on whether VS Code is running on Windows or inside WSL:

**🪟 Windows Host (VS Code on Windows)**: Use `"command": "wsl"` to invoke the WSL binary from Windows:

```json
{
  "servers": {
    "aks-mcp": {
      "type": "stdio",
      "command": "wsl",
      "args": [
        "--",
        "/home/you/.vs-kubernetes/tools/aks-mcp/aks-mcp",
        "--transport",
        "stdio"
      ]
    }
  }
}
```

**🐧 Remote-WSL (VS Code running inside WSL)**: Call the binary directly or use a shell wrapper:

```json
{
  "servers": {
    "aks-mcp": {
      "type": "stdio",
      "command": "bash",
      "args": [
        "-c",
        "/home/you/.vs-kubernetes/tools/aks-mcp/aks-mcp --transport stdio"
      ]
    }
  }
}
```


**🔧 Troubleshooting ENOENT Errors**

If you see "spawn ENOENT" errors, verify your VS Code environment:
- **Windows host**: Check if the WSL binary path is correct and accessible via `wsl -- ls /path/to/aks-mcp`
- **Remote-WSL**: Do NOT use `"command": "wsl"` - use direct paths or bash wrapper as shown above
</details>

> **💡 Benefits**: The AKS extension handles binary downloads, updates, and configuration automatically, ensuring you always have the latest version with optimal settings.


### Deploy the MCP server in-cluster (Remote MCP)
<details>
<summary> Remote MCP Installation </summary>
To enable the remote AKS MCP server in your AKS cluster, see the instructions below:

1. Helm chart installation with OAuth-based access: [Helm Chart](https://github.com/Azure/aks-mcp/tree/main/chart)

2. Helm chart installation with RBAC (Workload Identity): [Blog Post - Deploy AKS MCP server with Workload Identity](https://blog.aks.azure.com/2025/10/22/deploy-mcp-server-aks-workload-identity)
   
</details>


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

#### 🐳 Docker MCP Toolkit

You can enable the [AKS-MCP server directly from MCP Toolkit](https://hub.docker.com/mcp/server/aks/overview):

1. Open Docker Desktop
2. Click "MCP Toolkit" in the left sidebar
3. Search for "aks" in Catalog tab
4. Click on the AKS-MCP server card
5. Enable the server by clicking "+" in the top right corner
6. Configure the server using "Configuration" tab:
   - **azure_dir** `[REQUIRED]`: Path to your Azure credentials directory e.g `/home/user/.azure` (must be absolute – without `$HOME` or `~`)
   - **kubeconfig** `[REQUIRED]`: Path to your kubeconfig file e.g `/home/user/.kube/config` (must be absolute – without `$HOME` or `~`)
   - **access_level** `[REQUIRED]`: Set to `readonly`, `readwrite`, or `admin` as needed
   - **container_user** `[OPTIONAL]`: Username or UID to run the container as (default is `mcp`), e.g. use `1000` to match your host user ID (see note below). Only needed if you are using docker engine on Linux.
7. You are now ready to use the AKS-MCP server with your [preferred MCP client](https://hub.docker.com/mcp/server/aks/manual), see an example [here](https://docs.docker.com/ai/mcp-catalog-and-toolkit/toolkit/#install-an-mcp-client). (requires `>= v0.16.0` for MCP gateway)

> **Note**: When running the MCP gateway using Docker Engine, you have to set the `container_user` to match your host user ID (e.g using `id -u`) to ensure proper file permissions for accessing mounted volumes.
> On Docker Desktop, this is handled automatically if you use `desktop-*` contexts, confirmed by running `docker context ls`.

On **Windows**, the Azure credentials won't work by default, but you have two options:

1. **Long-lived servers**: Configure the [MCP gateway](https://docs.docker.com/ai/mcp-gateway/) to use long-lived servers using `--long-lived` flag and then authenticate with Azure CLI in the container, see option B in Containerized MCP configuration below on how to fetch credentials inside the container. 
2. **Custom Azure Directory**: Set up a custom Azure directory:
    ```powershell
    # Set custom Azure config directory
    $env:AZURE_CONFIG_DIR = "$env:USERPROFILE\.azure-for-docker"
    
    # Disable token cache encryption (to match behavior with Linux/macOS)
    $env:AZURE_CORE_ENCRYPT_TOKEN_CACHE = "false"
    
    # Login to Azure CLI
    az login
    ```

   This will store the credentials in `$env:USERPROFILE\.azure-for-docker` (e.g. `C:\Users\<username>\.azure-for-docker`),
   use this path in the AKS-MCP server configuration `azure_dir`.

You can also use the [MCP Gateway](https://docs.docker.com/ai/mcp-gateway/) to enable the AKS-MCP server directly using:

```bash
# Enable AKS-MCP server in Docker MCP Gateway
docker mcp server enable aks
```

Note: You still need to configure the server (e.g. using `docker mcp config`) with your Azure credentials, kubeconfig file, and access level.

#### 🐋 Containerized MCP configuration

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

- Host Azure CLI logins don’t automatically propagate into containers without mounting `~/.azure`.
- User ID should be set for option A, orelse the mcp user inside container won't be able to access the mounted files.

### 🤖 Custom MCP Client Installation

You can configure any MCP-compatible client to use the AKS-MCP server by running the binary directly:

```bash
# Run the server directly
./aks-mcp --transport stdio
```

### 🔧 Manual Binary Installation

For direct binary usage without package managers:

1. Download the latest release from the [releases page](https://github.com/Azure/aks-mcp/releases)
2. Extract the binary to your preferred location
3. Make it executable (on Unix systems):
   ```bash
   chmod +x aks-mcp
   ```
4. Configure your MCP client to use the binary path

</details>

### Options

Command line arguments:

```sh
Usage of ./aks-mcp:
      --access-level string       Access level (readonly, readwrite, admin) (default "readonly")
      --enabled-components string Comma-separated list of enabled components (empty means all components enabled). Available: az_cli,monitor,fleet,network,compute,detectors,advisor,inspektorgadget,kubectl,helm,cilium,hubble
      --allow-namespaces string   Comma-separated list of allowed Kubernetes namespaces (empty means all namespaces)
      --host string               Host to listen for the server (only used with transport sse or streamable-http) (default "127.0.0.1")
      --otlp-endpoint string      OTLP endpoint for OpenTelemetry traces (e.g. localhost:4317, default "")
      --port int                  Port to listen for the server (only used with transport sse or streamable-http) (default 8000)
      --timeout int               Timeout for command execution in seconds, default is 600s (default 600)
      --transport string          Transport mechanism to use (stdio, sse or streamable-http) (default "stdio")
      --log-level string          Log level (debug, info, warn, error) (default "info")
```

**Environment variables:**
- `USE_LEGACY_TOOLS`: Set to `true` to use legacy specialized tools instead of unified tools (default: `false`)
  - `false` (default): Uses `call_az` for Azure operations and `call_kubectl` for Kubernetes operations
  - `true`: Uses legacy tools like `az_aks_operations`, `az_compute_operations`, and specialized kubectl tools
- Standard Azure authentication environment variables are supported (`AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_SUBSCRIPTION_ID`)

## Development

### Prerequisites

- **Go** ≥ `1.24.x` installed on your local machine
- **Bash** available as `/usr/bin/env bash` (Makefile targets use multi-line recipes with fail-fast mode)
- **GNU Make** `4.x` or later
- **Docker** *(optional, for container builds and testing)*

> **Note:** If your login shell is different (e.g., `zsh` on **macOS**), you do **not** need to change it — the Makefile sets variables to run all recipes in `bash` for consistent behavior across platforms.

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

Create a new Azure Fleet named prod-fleet in eastus region.

List all members in my fleet.

Create a placement to deploy nginx workloads to clusters with app=frontend label.

Show me all ClusterResourcePlacements in my fleet.
```

## Telemetry

Telemetry collection is on by default.

To opt out, set the environment variable `AKS_MCP_COLLECT_TELEMETRY=false`.

## Contributing

We welcome contributions to AKS-MCP! Whether you're fixing bugs, adding features, or improving documentation, your help makes this project better.

**📖 [Read our detailed Contributing Guide](CONTRIBUTING.md)** for comprehensive information on:

- Setting up your development environment
- Running AKS-MCP locally and testing with AI agents
- Understanding the codebase architecture
- Adding new MCP tools and features
- Testing guidelines and best practices
- Submitting pull requests

### Quick Start for Contributors

1. **Prerequisites**: Go ≥ 1.24.x, Azure CLI, Git
2. **Setup**: Fork the repo, clone locally, run `make deps && make build`
3. **Test**: Run `make test` and `make check`
4. **Develop**: Follow the component-based architecture in [CONTRIBUTING.md](CONTRIBUTING.md)

### Contributor License Agreement

Most contributions require you to agree to a Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft
trademarks or logos is subject to and must follow
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
