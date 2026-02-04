# AKS-MCP E2E Testing

End-to-end testing framework for AKS-MCP server running in Azure Kubernetes Service.

## Architecture

The E2E test framework consists of:

1. **AKS Cluster**: Test environment with Workload Identity enabled
2. **MCP Server**: Deployed via Helm with Workload Identity authentication
3. **Test Client**: Go application that connects to MCP server and validates tools
4. **Infrastructure Scripts**: Automated setup and teardown

### Key Design Decisions

- **No Token Pass-Through**: Test client does NOT pass Azure tokens to MCP server
- **Workload Identity**: MCP server authenticates to Azure using Workload Identity
- **Streamable HTTP**: Client connects to server via standard HTTP (no special headers)
- **Manual Triggering**: E2E tests are manually triggered (not automatic in CI)
- **Ephemeral Clusters**: Each test run creates a fresh AKS cluster and deletes it after

## Quick Start

### Prerequisites

- Azure CLI installed and configured (`az login`)
- kubectl installed
- Helm 3 installed
- Docker installed (for building test client image)
- Go 1.24+ (optional, for local development)

### Step 1: Create Test AKS Cluster

```bash
cd test/e2e/scripts
./setup-aks.sh
```

This will:
- Create a resource group (`aks-mcp-e2e-test-rg`)
- Create an AKS cluster with Workload Identity enabled
- Configure kubectl context
- Output the OIDC issuer URL

Export the environment variables from the script output:

```bash
export AZURE_SUBSCRIPTION_ID=<your-subscription-id>
export RESOURCE_GROUP=<resource-group-name>
export CLUSTER_NAME=<cluster-name>
export OIDC_ISSUER=<oidc-issuer-url>
```

### Step 2: Configure Workload Identity (for MCP Server)

```bash
./setup-workload-identity.sh
```

This will:
- Create an Azure Managed Identity
- Create a federated credential for the service account
- Assign Azure RBAC roles (Reader on subscription and node resource group)

Export the additional environment variables:

```bash
export AZURE_CLIENT_ID=<client-id>
export AZURE_TENANT_ID=<tenant-id>
```

### Step 3: Deploy MCP Server

```bash
cd ../../..  # Back to project root

helm install aks-mcp ./chart \
  --set azure.tenantId=$AZURE_TENANT_ID \
  --set azure.clientId=$AZURE_CLIENT_ID \
  --set azure.subscriptionId=$AZURE_SUBSCRIPTION_ID \
  --set workloadIdentity.enabled=true \
  --set app.transport=streamable-http \
  --set app.accessLevel=readonly \
  --set app.logLevel=debug \
  --set-json 'config.enabledComponents=["compute","az_cli","kubectl"]'
```

Wait for the pod to be ready:

```bash
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=aks-mcp --timeout=120s
```

Verify MCP server health:

```bash
kubectl port-forward svc/aks-mcp 8000:8000 &
curl http://localhost:8000/health
```

### Step 4: Run E2E Tests

#### Option A: Run Locally with Port-Forward

Build and run the test client on your local machine:

```bash
cd test/e2e

# Build the test client
go build -o e2e-test ./cmd/e2e-test

# Start port-forward
kubectl port-forward svc/aks-mcp 8000:8000 &
PF_PID=$!

# Wait for port-forward to be ready
sleep 3

# Set environment variables
export MCP_SERVER_URL=http://localhost:8000
export AZURE_SUBSCRIPTION_ID=<your-subscription-id>
export RESOURCE_GROUP=<your-resource-group>
export CLUSTER_NAME=<your-cluster-name>

# Run tests (with verbose output to see parameters and results)
./e2e-test --verbose

# Or run without verbose
./e2e-test

# Stop port-forward when done
kill $PF_PID
```

**Verbose Mode:**

Use `--verbose` or `-v` flag to see detailed tool call parameters and results:

```bash
./e2e-test --verbose
```

This will display:
- Tool call parameters (JSON formatted)
- Full tool response (pretty-printed JSON)
- Useful for debugging and understanding what the tools return

#### Option B: Run in Kubernetes (Advanced)

For testing the full containerized deployment:

```bash
cd test/e2e
docker build -t aks-mcp-e2e-test:local .

# If using Kind/Minikube, load the image
kind load docker-image aks-mcp-e2e-test:local
```

Deploy the test Job:

```bash
# Substitute environment variables in the manifest
envsubst < manifests/e2e-job.yaml | kubectl apply -f -
```

Wait for completion and check results:

```bash
kubectl wait --for=condition=complete job/aks-mcp-e2e-test --timeout=300s
kubectl logs job/aks-mcp-e2e-test
```

#### Option B: Run Locally (Quick Debug)

Set up port-forward and run locally:

```bash
kubectl port-forward svc/aks-mcp 8000:8000 &

export MCP_SERVER_URL=http://localhost:8000
export AZURE_SUBSCRIPTION_ID=<your-subscription-id>
export RESOURCE_GROUP=<your-resource-group>
export CLUSTER_NAME=<your-cluster-name>

go run ./cmd/e2e-test/main.go
```

### Step 5: Cleanup

Delete all test resources:

```bash
cd scripts
./cleanup.sh
```

To preserve resources for debugging:

```bash
SKIP_CLEANUP=true ./cleanup.sh
```

## Test Cases

### get_aks_vmss_info

Tests the `get_aks_vmss_info` tool which retrieves Virtual Machine Scale Set information for AKS node pools.

**Test Scenarios:**
1. Get VMSS info for all node pools (no `node_pool_name` specified)
2. Get VMSS info for a specific node pool (if `NODE_POOL_NAME` env var is set)

**Validation:**
- Response is valid JSON
- Contains required fields: `id`, `name`, `location`
- VMSS ID is non-empty
- For array responses, validates each VMSS object

## Project Structure

```
test/e2e/
├── cmd/
│   └── e2e-test/
│       └── main.go              # Test entry point
├── pkg/
│   ├── client/
│   │   └── mcp_client.go        # MCP client wrapper (no token logic)
│   ├── tests/
│   │   ├── interface.go         # Test interface
│   │   └── vmss_test.go         # VMSS info test implementation
│   └── runner/
│       └── runner.go            # Test execution engine
├── manifests/
│   └── e2e-job.yaml             # Kubernetes Job for test client
├── scripts/
│   ├── setup-aks.sh             # Create AKS cluster
│   ├── setup-workload-identity.sh  # Configure Workload Identity
│   └── cleanup.sh               # Delete all resources
├── Dockerfile                   # Test client container image
├── go.mod                       # Go module definition
└── README.md                    # This file
```

## Environment Variables

### Test Configuration

- `MCP_SERVER_URL`: URL of the MCP server (default: `http://aks-mcp.default.svc.cluster.local:8000`)
- `AZURE_SUBSCRIPTION_ID`: Azure subscription ID (required)
- `RESOURCE_GROUP`: Resource group containing the AKS cluster (required)
- `CLUSTER_NAME`: Name of the AKS cluster (required)
- `NODE_POOL_NAME`: Optional, specific node pool to test

### Infrastructure Scripts

- `RESOURCE_GROUP`: Resource group name (default: `aks-mcp-e2e-test-rg`)
- `CLUSTER_NAME`: AKS cluster name (default: `aks-mcp-e2e-test`)
- `LOCATION`: Azure region (default: `eastus`)
- `NODE_COUNT`: Number of nodes (default: `2`)
- `NODE_VM_SIZE`: VM size for nodes (default: `Standard_DS2_v2`)
- `IDENTITY_NAME`: Azure Managed Identity name (default: `aks-mcp-e2e-identity`)
- `SKIP_CLEANUP`: Set to `true` to preserve resources (default: `false`)

## Debugging

### View MCP Server Logs

```bash
kubectl logs -l app.kubernetes.io/name=aks-mcp -f
```

### View Test Client Logs

```bash
kubectl logs job/aks-mcp-e2e-test
```

### Interactive Debugging

Run a debug pod with the test client:

```bash
kubectl run -it --rm debug-client \
  --image=aks-mcp-e2e-test:local \
  --env="MCP_SERVER_URL=http://aks-mcp.default.svc.cluster.local:8000" \
  --env="AZURE_SUBSCRIPTION_ID=$AZURE_SUBSCRIPTION_ID" \
  --env="RESOURCE_GROUP=$RESOURCE_GROUP" \
  --env="CLUSTER_NAME=$CLUSTER_NAME" \
  -- /bin/sh

# Inside the pod:
/usr/local/bin/e2e-test
```

### Check MCP Server Health

```bash
kubectl port-forward svc/aks-mcp 8000:8000 &
curl http://localhost:8000/health

# Should return:
# {
#   "status": "healthy",
#   "version": "x.x.x",
#   "transport": "streamable-http"
# }
```

### Verify Workload Identity Configuration

Check service account annotations:

```bash
kubectl get sa aks-mcp -o yaml | grep azure
```

Check pod identity:

```bash
kubectl get pod -l app.kubernetes.io/name=aks-mcp -o yaml | grep -A 5 azure
```

Test Azure authentication from MCP server pod:

```bash
kubectl exec -it <mcp-pod-name> -- /bin/sh
# Inside pod:
az account show
```

## Adding New Tests

### 1. Create Test Implementation

Create a new file in `pkg/tests/`:

```go
package tests

import (
    "context"
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
)

type MyNewTest struct {
    // Test configuration fields
}

func (t *MyNewTest) Name() string {
    return "my_new_test"
}

func (t *MyNewTest) Run(ctx context.Context, mcpClient *client.StdioMCPClient) (*mcp.CallToolResult, error) {
    // Call the MCP tool
    result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
        Name:      "my_tool_name",
        Arguments: map[string]interface{}{
            // tool arguments
        },
    })
    return result, err
}

func (t *MyNewTest) Validate(result *mcp.CallToolResult) error {
    // Validate the result
    return nil
}
```

### 2. Register Test in Main

Edit `cmd/e2e-test/main.go`:

```go
testRunner.AddTest(&tests.MyNewTest{
    // Configuration
})
```

### 3. Run Tests

```bash
go run ./cmd/e2e-test/main.go
```
