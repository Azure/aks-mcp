# AKS-MCP Helm Chart

This Helm chart deploys AKS-MCP (Azure Kubernetes Service Model Context Protocol) server on Kubernetes clusters.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.8+

## Installation

### Install from Source

```bash
# Clone the repository
git clone https://github.com/Azure/aks-mcp.git
cd aks-mcp/chart

# Install the chart
helm install my-aks-mcp . --namespace aks-mcp --create-namespace
```

## Configuration

### Basic Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Container image repository | `ghcr.io/azure/aks-mcp` |
| `image.tag` | Container image tag | `""` (uses chart appVersion) |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |

### Application Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `app.transport` | Transport mechanism (stdio, sse, streamable-http) | `streamable-http` |
| `app.port` | Port to listen on | `8000` |
| `app.accessLevel` | Access level (readonly, readwrite, admin) | `readonly` |
| `app.timeout` | Command execution timeout in seconds | `600` |
| `app.verbose` | Enable verbose logging | `false` |
| `app.cache` | Enable cache for better performance | `true` |

### Azure Authentication

| Parameter | Description | Default |
|-----------|-------------|---------|
| `azure.existingSecret` | Use existing secret for Azure credentials | `""` |
| `azure.tenantId` | Azure tenant ID | `""` |
| `azure.clientId` | Azure client ID | `""` |
| `azure.clientSecret` | Azure client secret | `""` |
| `azure.subscriptionId` | Azure subscription ID | `""` |

### OAuth Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `oauth.enabled` | Enable OAuth authentication | `false` |
| `oauth.tenantId` | Azure AD tenant ID for OAuth | `""` |
| `oauth.clientId` | Azure AD client ID for OAuth | `""` |
| `oauth.redirectURIs` | Custom redirect URIs | `[]` |
| `oauth.corsOrigins` | Custom CORS origins | `[]` |

### Additional Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.additionalTools` | Additional Kubernetes tools (helm, cilium, hubble) | `[]` |
| `config.allowNamespaces` | Allowed Kubernetes namespaces (empty means all) | `[]` |
| `config.cacheTimeout` | Cache timeout | `"60s"` |
| `kubeconfig.enabled` | Enable kubeconfig secret mount | `false` |
| `kubeconfig.secretName` | Name of kubeconfig secret | `"kubeconfig"` |

### Security Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `podSecurityContext.runAsNonRoot` | Run as non-root user | `true` |
| `securityContext.readOnlyRootFilesystem` | Enable read-only root filesystem | `true` |

## Examples

### Basic Readonly Deployment
```bash
helm install my-aks-mcp . \
  --set app.accessLevel=readonly \
  --set azure.tenantId=your-tenant-id \
  --set azure.clientId=your-client-id \
  --set azure.clientSecret=your-client-secret
```

### Readwrite Deployment with OAuth
```bash
helm install my-aks-mcp . \
  --set app.accessLevel=readwrite \
  --set oauth.enabled=true \
  --set oauth.tenantId=your-tenant-id \
  --set oauth.clientId=your-oauth-client-id \
  --set azure.subscriptionId=your-subscription-id
```

### Using Existing Secret for Azure Credentials
```bash
# Create secret first
kubectl create secret generic azure-credentials \
  --from-literal=tenant-id=your-tenant-id \
  --from-literal=client-id=your-client-id \
  --from-literal=client-secret=your-client-secret \
  --from-literal=subscription-id=your-subscription-id

# Install with existing secret
helm install my-aks-mcp . \
  --set azure.existingSecret=azure-credentials
```

### Admin Deployment with Additional Tools
```bash
helm install my-aks-mcp . \
  --set app.accessLevel=admin \
  --set config.additionalTools="{helm,cilium,hubble}" \
  --set config.allowNamespaces="{kube-system,default}" \
  --set azure.existingSecret=azure-credentials
```

## Development and Testing

### MCP Inspector Deployment

For development and testing, you can deploy AKS-MCP for use with MCP Inspector:

```bash
# Deploy using the inspector-optimized configuration
helm upgrade --install aks-mcp-inspector . \
  --namespace aks-mcp-inspector \
  --create-namespace \
  -f ./values-mcp-inspector.yaml

# Set up port forwarding
kubectl port-forward service/aks-mcp-inspector 8081:8081 -n aks-mcp-inspector

# Start MCP Inspector (in another terminal)
npx @modelcontextprotocol/inspector
```

For detailed MCP Inspector setup instructions, see [MCP_INSPECTOR_DEPLOYMENT.md](./MCP_INSPECTOR_DEPLOYMENT.md).

## Monitoring and Observability

### Telemetry
Configure OpenTelemetry:
```bash
helm install my-aks-mcp . \
  --set telemetry.otlpEndpoint=http://jaeger:14268/api/traces
```

### Logging
Enable verbose logging for debugging:
```bash
helm upgrade my-aks-mcp . --set app.verbose=true
```

## Architecture Notes

- **Single Replica**: The deployment is hardcoded to 1 replica due to OAuth state management requirements
- **Security**: Read-only root filesystem with writable `/tmp` directory for temporary files
- **RBAC**: Dynamic permissions based on access level configuration
- **Configuration**: All application settings are passed via command-line arguments
