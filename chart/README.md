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

For detailed configuration parameters, see [helm-chart.md](./helm-chart.md).

## Deployment Examples

### Basic Readonly Deployment (Service Principal)
```bash
helm install my-aks-mcp . \
  --set app.accessLevel=readonly \
  --set azure.tenantId=your-tenant-id \
  --set azure.clientId=your-client-id \
  --set azure.clientSecret=your-client-secret
```

### Workload Identity Deployment
For passwordless authentication using Azure Workload Identity, see the detailed guide: [helm-workload-identity.md](../docs/helm-workload-identity.md)

```bash
helm install my-aks-mcp . \
  --set workloadIdentity.enabled=true \
  --set azure.tenantId=your-tenant-id \
  --set azure.clientId=your-managed-identity-client-id \
  --set azure.subscriptionId=your-subscription-id
```

**Note:** Workload Identity requires additional Azure setup (OIDC Issuer, Managed Identity, Federated Credential). See the full guide for step-by-step instructions.

### Readwrite Deployment with OAuth
```bash
helm install my-aks-mcp . \
  --set app.accessLevel=readwrite \
  --set oauth.enabled=true \
  --set oauth.tenantId=your-tenant-id \
  --set oauth.clientId=your-oauth-client-id \
  --set azure.subscriptionId=your-subscription-id
```

### OAuth with Custom Scope (Assignment Required)

To restrict access to only assigned users/identities using Azure AD's "Assignment Required" feature, use a custom App ID URI scope:

```bash
# First, configure your Azure AD app with an Application ID URI:
# Azure Portal → App registrations → [Your App] → Expose an API → Add Application ID URI
# Set to: api://<your-client-id>

# Then deploy with custom scope
helm install my-aks-mcp . \
  --set app.accessLevel=readonly \
  --set oauth.enabled=true \
  --set oauth.tenantId=your-tenant-id \
  --set oauth.clientId=your-oauth-client-id \
  --set oauth.scopes[0]="api://your-oauth-client-id/.default" \
  --set azure.subscriptionId=your-subscription-id
```

**Important:** When using custom scopes, clients must request tokens for your app specifically:
```bash
# User authentication
az account get-access-token --resource api://your-oauth-client-id --query accessToken -o tsv

# Managed Identity (from Azure VM/AKS)
curl -s 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=api://your-oauth-client-id' \
  -H 'Metadata: true' | jq -r '.access_token'
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

### Deployment with Ingress Support

#### Basic Ingress Deployment
```bash
helm install my-aks-mcp . \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=aks-mcp.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix \
  --set azure.existingSecret=azure-credentials
```

#### Azure App Routing with OAuth
```bash
# Enable Azure App Routing on your AKS cluster first
az aks approuting enable \
    --resource-group your-resource-group \
    --name your-cluster-name

# Deploy with Azure App Routing configuration
helm install my-aks-mcp . \
  -f values-mcp-inspector-azure-approuting.yaml \
  --namespace aks-mcp \
  --create-namespace
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
