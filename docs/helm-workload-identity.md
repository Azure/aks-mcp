# AKS-MCP Helm Chart - Azure Workload Identity Guide

A comprehensive guide for deploying AKS-MCP on Kubernetes with Azure Workload Identity support for passwordless authentication.

## Overview

This guide walks through deploying the AKS-MCP Helm chart on Azure Kubernetes Service (AKS) using Azure Workload Identity for secure, passwordless authentication. Workload Identity eliminates the need to store credentials in Kubernetes Secrets by leveraging federated identity credentials.

**Default Configuration:**
- **Transport**: `streamable-http` (HTTP-based MCP protocol)
- **Access Level**: `readonly` (read-only operations)
- **Authentication**: Service Principal via Secret (default), or Workload Identity (optional, requires setup)
- **Port**: `8000`

## Prerequisites

- Azure Kubernetes Service (AKS) cluster
- Helm 3.x installed
- Azure CLI installed and authenticated (`az login`)
- AKS cluster with OIDC Issuer enabled (required for Workload Identity)
- Permissions to:
  - Create Azure Managed Identities
  - Assign Azure RBAC roles
  - Create federated identity credentials

## Deployment Guide

Follow these steps in order to deploy AKS-MCP with Workload Identity:

### Step 1: Enable OIDC on AKS Cluster

Azure Workload Identity requires the AKS cluster to have an OIDC issuer endpoint enabled.

```bash
export RESOURCE_GROUP="<your-resource-group>"
export AKS_CLUSTER_NAME="<your-aks-cluster>"

# Enable OIDC Issuer and Workload Identity on the cluster
az aks update \
  --resource-group $RESOURCE_GROUP \
  --name $AKS_CLUSTER_NAME \
  --enable-oidc-issuer \
  --enable-workload-identity

# Get the OIDC Issuer URL (needed for federated credential)
export AKS_OIDC_ISSUER=$(az aks show \
  --resource-group $RESOURCE_GROUP \
  --name $AKS_CLUSTER_NAME \
  --query "oidcIssuerProfile.issuerUrl" \
  --output tsv)

echo "OIDC Issuer: $AKS_OIDC_ISSUER"
```

**Note**: This operation may take a few minutes to complete.

### Step 2: Create Azure Managed Identity

Create a managed identity that will be used by the AKS-MCP pods to authenticate with Azure.

```bash
export IDENTITY_NAME="aks-mcp-identity"
export LOCATION="<your-location>"  # e.g., eastus, westeurope

# Create the Managed Identity
az identity create \
  --resource-group $RESOURCE_GROUP \
  --name $IDENTITY_NAME \
  --location $LOCATION

# Get the Client ID (needed for Helm installation)
export IDENTITY_CLIENT_ID=$(az identity show \
  --resource-group $RESOURCE_GROUP \
  --name $IDENTITY_NAME \
  --query "clientId" \
  --output tsv)

# Get the Principal ID (needed for RBAC assignments)
export IDENTITY_PRINCIPAL_ID=$(az identity show \
  --resource-group $RESOURCE_GROUP \
  --name $IDENTITY_NAME \
  --query "principalId" \
  --output tsv)

echo "Identity Client ID: $IDENTITY_CLIENT_ID"
echo "Identity Principal ID: $IDENTITY_PRINCIPAL_ID"
```

### Step 3: Configure Azure RBAC Permissions

Grant the managed identity appropriate permissions to access Azure resources. The permissions required depend on your use case.

#### For Read-Only Access (Default)

```bash
export SUBSCRIPTION_ID=$(az account show --query id --output tsv)

# Grant Reader role at subscription level
az role assignment create \
  --role "Reader" \
  --assignee-object-id $IDENTITY_PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope "/subscriptions/$SUBSCRIPTION_ID"
```

#### For Read-Write Access

If you need to perform write operations (e.g., scaling node pools, updating clusters), use `--access-level=readwrite` and assign appropriate roles:

```bash
# Grant Contributor role at subscription level
az role assignment create \
  --role "Contributor" \
  --assignee-object-id $IDENTITY_PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope "/subscriptions/$SUBSCRIPTION_ID"
```

#### For Resource Group-Scoped Access

For better security, scope permissions to specific resource groups:

```bash
export TARGET_RESOURCE_GROUP="<target-resource-group>"

az role assignment create \
  --role "Reader" \
  --assignee-object-id $IDENTITY_PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$TARGET_RESOURCE_GROUP"
```

#### AKS-Specific Roles

For AKS cluster management, you may need specific roles:

```bash
# Azure Kubernetes Service Cluster User Role
az role assignment create \
  --role "Azure Kubernetes Service Cluster User Role" \
  --assignee-object-id $IDENTITY_PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ContainerService/managedClusters/$AKS_CLUSTER_NAME"

# Azure Kubernetes Service Cluster Admin Role (for admin operations)
az role assignment create \
  --role "Azure Kubernetes Service Cluster Admin Role" \
  --assignee-object-id $IDENTITY_PRINCIPAL_ID \
  --assignee-principal-type ServicePrincipal \
  --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ContainerService/managedClusters/$AKS_CLUSTER_NAME"
```

### Step 4: Create Federated Identity Credential

Link the Kubernetes ServiceAccount to the Azure Managed Identity using a federated credential.

```bash
export SERVICE_ACCOUNT_NAMESPACE="default"
export SERVICE_ACCOUNT_NAME="aks-mcp"  # Default name format: <release-name>

# Create the federated credential
az identity federated-credential create \
  --name "aks-mcp-federated-credential" \
  --identity-name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --issuer $AKS_OIDC_ISSUER \
  --subject "system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}" \
  --audience api://AzureADTokenExchange
```

**Important Notes:**
- The ServiceAccount name follows Helm's naming convention: `<release-name>` (or custom name from `fullnameOverride`)
- If using a different release name or namespace, adjust accordingly
- You **must** create the federated credential **before** installing the Helm chart
- The credential creation is instantaneous but may take a few seconds to propagate

**Verify the credential:**

```bash
az identity federated-credential list \
  --identity-name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --output table
```

### Step 5: Install the Helm Chart

Now install the AKS-MCP Helm chart with Workload Identity enabled.

```bash
# Install with Workload Identity enabled (readonly mode)
helm install aks-mcp ./chart \
  --set workloadIdentity.enabled=true \
  --set azure.clientId=$IDENTITY_CLIENT_ID \
  --set azure.subscriptionId=$SUBSCRIPTION_ID
```

**Install to a specific namespace:**

```bash
helm install aks-mcp ./chart \
  --namespace aks-mcp \
  --create-namespace \
  --set workloadIdentity.enabled=true \
  --set azure.clientId=$IDENTITY_CLIENT_ID \
  --set azure.subscriptionId=$SUBSCRIPTION_ID
```

**Note**: If installing to a non-default namespace, update the federated credential subject in Step 4.

### Step 6: Verify Deployment

Check that the deployment is successful and authentication is working.

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=aks-mcp

# Check ServiceAccount has correct annotations
kubectl get serviceaccount aks-mcp -o yaml

# Check pod logs
kubectl logs -l app.kubernetes.io/name=aks-mcp --tail=50

# Verify Azure authentication (should show account details)
kubectl exec -it $(kubectl get pod -l app.kubernetes.io/name=aks-mcp -o jsonpath='{.items[0].metadata.name}') -- az account show

# Test AKS cluster listing
kubectl exec -it $(kubectl get pod -l app.kubernetes.io/name=aks-mcp -o jsonpath='{.items[0].metadata.name}') -- az aks list --output table
```

**Expected ServiceAccount annotations:**
```yaml
metadata:
  annotations:
    azure.workload.identity/client-id: "<your-client-id>"
  labels:
    azure.workload.identity/use: "true"
```

**Expected Pod labels:**
```yaml
metadata:
  labels:
    azure.workload.identity/use: "true"
```

## Configuration Examples

### Enable OAuth Authentication

```bash
helm upgrade aks-mcp ./chart \
  --reuse-values \
  --set oauth.enabled=true \
  --set oauth.tenantId=$TENANT_ID \
  --set oauth.clientId=$OAUTH_CLIENT_ID
```

### Enable Debug Logging

```bash
helm upgrade aks-mcp ./chart \
  --reuse-values \
  --set app.logLevel=debug
```

### Change Image Tag

```bash
helm upgrade aks-mcp ./chart \
  --reuse-values \
  --set image.tag=v1.2.3
```

### Scale Deployment (Not Recommended)

Note: AKS-MCP is designed to run as a single replica. Scaling is not recommended unless you have specific requirements.

```bash
# If you must scale (not recommended)
kubectl scale deployment aks-mcp --replicas=2
```

### Use Service Principal Instead (Default)

If you prefer to use Service Principal authentication (default behavior):

```bash
# Create a secret with Service Principal credentials
kubectl create secret generic aks-mcp-azure-credentials \
  --from-literal=tenant-id=$TENANT_ID \
  --from-literal=client-id=$SP_CLIENT_ID \
  --from-literal=client-secret=$SP_CLIENT_SECRET \
  --from-literal=subscription-id=$SUBSCRIPTION_ID

# Install with default settings (Workload Identity disabled)
helm install aks-mcp ./chart \
  --set azure.existingSecret=aks-mcp-azure-credentials
```

**Or provide credentials directly via values:**

```bash
helm install aks-mcp ./chart \
  --set azure.tenantId=$TENANT_ID \
  --set azure.clientId=$SP_CLIENT_ID \
  --set azure.clientSecret=$SP_CLIENT_SECRET \
  --set azure.subscriptionId=$SUBSCRIPTION_ID
```

## Configuration Reference

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `workloadIdentity.enabled` | Enable Azure Workload Identity | `false` |
| `azure.clientId` | Azure Client ID (required for Workload Identity) | `""` |
| `azure.subscriptionId` | Azure Subscription ID set the current default active subscription| `""` |
| `app.accessLevel` | Access level: `readonly`, `readwrite`, `admin` | `readonly` |
| `app.transport` | Transport: `stdio`, `sse`, `streamable-http` | `streamable-http` |
| `app.port` | Server port | `8000` |
| `app.logLevel` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `app.cache` | Enable resource caching | `true` |

### All Parameters

See `chart/values.yaml` for complete list of configurable parameters.

## Accessing the Service

### Port Forward (Development/Testing)

```bash
kubectl port-forward svc/aks-mcp 8000:8000

# Test health endpoint
curl http://localhost:8000/health

# Test MCP endpoint (requires MCP client)
# Example: use MCP Inspector
npx @modelcontextprotocol/inspector http://localhost:8000
```

### Using with MCP Clients

Configure your MCP client to connect to the service:

**For local development (with port-forward):**
```json
{
  "mcpServers": {
    "aks-mcp": {
      "url": "http://localhost:8000",
      "transport": "streamable-http"
    }
  }
}
```

**For in-cluster access:**
```
http://aks-mcp.default.svc.cluster.local:8000
```

### Expose via Ingress (Production)

```bash
helm upgrade aks-mcp ./chart \
  --reuse-values \
  --set ingress.enabled=true
```

See `docs/helm-chart.md` for detailed ingress configuration.

## Upgrading

```bash
# Upgrade to new version
helm upgrade aks-mcp ./chart \
  --reuse-values \
  --set image.tag=v1.3.0

# View release history
helm history aks-mcp

# Rollback if needed
helm rollback aks-mcp 1
```

## Uninstalling

### Remove Helm Release

```bash
helm uninstall aks-mcp
```

### Clean Up Azure Resources

```bash
# Delete federated credential
az identity federated-credential delete \
  --name "aks-mcp-federated-credential" \
  --identity-name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP

# Remove role assignments
az role assignment delete \
  --assignee $IDENTITY_PRINCIPAL_ID \
  --scope "/subscriptions/$SUBSCRIPTION_ID"

# Delete managed identity
az identity delete \
  --resource-group $RESOURCE_GROUP \
  --name $IDENTITY_NAME
```