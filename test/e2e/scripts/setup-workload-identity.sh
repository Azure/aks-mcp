#!/bin/bash
set -e

# Configuration
RESOURCE_GROUP="${RESOURCE_GROUP:-aks-mcp-e2e-test-rg}"
CLUSTER_NAME="${CLUSTER_NAME:-aks-mcp-e2e-test}"
IDENTITY_NAME="${IDENTITY_NAME:-aks-mcp-e2e-identity}"
SERVICE_ACCOUNT_NAMESPACE="${SERVICE_ACCOUNT_NAMESPACE:-default}"
SERVICE_ACCOUNT_NAME="${SERVICE_ACCOUNT_NAME:-aks-mcp}"

echo "=================================================="
echo "Configuring Workload Identity for MCP Server"
echo "=================================================="
echo "Resource Group:       $RESOURCE_GROUP"
echo "Cluster Name:         $CLUSTER_NAME"
echo "Identity Name:        $IDENTITY_NAME"
echo "Service Account:      $SERVICE_ACCOUNT_NAMESPACE/$SERVICE_ACCOUNT_NAME"
echo "=================================================="
echo ""

# Check prerequisites
if ! command -v az &> /dev/null; then
    echo "❌ Error: Azure CLI is not installed"
    exit 1
fi

if ! az account show &> /dev/null; then
    echo "❌ Error: Not logged in to Azure CLI"
    exit 1
fi

# Get Azure subscription and tenant info
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
TENANT_ID=$(az account show --query tenantId -o tsv)
LOCATION=$(az group show --name "$RESOURCE_GROUP" --query location -o tsv)

echo "📋 Azure Information:"
echo "   Subscription ID: $SUBSCRIPTION_ID"
echo "   Tenant ID:       $TENANT_ID"
echo "   Location:        $LOCATION"
echo ""

# Get OIDC issuer URL
echo "🔍 Getting AKS OIDC issuer URL..."
OIDC_ISSUER=$(az aks show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CLUSTER_NAME" \
    --query "oidcIssuerProfile.issuerUrl" \
    -o tsv)

if [ -z "$OIDC_ISSUER" ]; then
    echo "❌ Error: Could not get OIDC issuer URL"
    echo "   Make sure the cluster was created with --enable-oidc-issuer"
    exit 1
fi

echo "   ✅ OIDC Issuer: $OIDC_ISSUER"
echo ""

# Create Azure Managed Identity
echo "🔐 Creating Azure Managed Identity..."
if az identity show --resource-group "$RESOURCE_GROUP" --name "$IDENTITY_NAME" &> /dev/null; then
    echo "   ⚠️  Managed Identity already exists, retrieving details..."
else
    az identity create \
        --resource-group "$RESOURCE_GROUP" \
        --name "$IDENTITY_NAME" \
        --location "$LOCATION" \
        --output none
    echo "   ✅ Managed Identity created"
fi

# Get the Client ID (needed for Helm installation)
CLIENT_ID=$(az identity show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$IDENTITY_NAME" \
    --query "clientId" \
    --output tsv)

# Get the Principal ID (needed for RBAC assignments)
PRINCIPAL_ID=$(az identity show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$IDENTITY_NAME" \
    --query "principalId" \
    --output tsv)

echo "   Client ID:    $CLIENT_ID"
echo "   Principal ID: $PRINCIPAL_ID"
echo ""

# Create federated credential
echo "🔗 Creating Federated Credential..."
CREDENTIAL_NAME="$CLUSTER_NAME-$SERVICE_ACCOUNT_NAMESPACE-$SERVICE_ACCOUNT_NAME"

if az identity federated-credential show \
    --identity-name "$IDENTITY_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CREDENTIAL_NAME" &> /dev/null; then
    echo "   ⚠️  Federated credential already exists, deleting and recreating..."
    az identity federated-credential delete \
        --identity-name "$IDENTITY_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --name "$CREDENTIAL_NAME" \
        --output none
fi

az identity federated-credential create \
    --name "$CREDENTIAL_NAME" \
    --identity-name "$IDENTITY_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --issuer "$OIDC_ISSUER" \
    --subject "system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}" \
    --audience api://AzureADTokenExchange \
    --output none

echo "   ✅ Federated credential created"
echo ""

# Assign Azure RBAC roles
echo "🔑 Assigning Azure RBAC roles..."

# Get AKS resource ID
AKS_RESOURCE_ID=$(az aks show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CLUSTER_NAME" \
    --query id \
    -o tsv)

# Get node resource group
NODE_RESOURCE_GROUP=$(az aks show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CLUSTER_NAME" \
    --query nodeResourceGroup \
    -o tsv)

# Assign Reader role on subscription (for listing resources)
echo "   - Assigning Reader role on subscription..."
az role assignment create \
    --role "Reader" \
    --assignee-object-id "$PRINCIPAL_ID" \
    --assignee-principal-type ServicePrincipal \
    --scope "/subscriptions/$SUBSCRIPTION_ID" \
    --output none 2>/dev/null || echo "     (Role assignment may already exist)"

# Assign Reader role on AKS cluster resource
echo "   - Assigning Reader role on AKS cluster..."
az role assignment create \
    --role "Reader" \
    --assignee-object-id "$PRINCIPAL_ID" \
    --assignee-principal-type ServicePrincipal \
    --scope "$AKS_RESOURCE_ID" \
    --output none 2>/dev/null || echo "     (Role assignment may already exist)"

# Assign Reader role on node resource group (for VMSS access)
echo "   - Assigning Reader role on node resource group..."
az role assignment create \
    --role "Reader" \
    --assignee-object-id "$PRINCIPAL_ID" \
    --assignee-principal-type ServicePrincipal \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$NODE_RESOURCE_GROUP" \
    --output none 2>/dev/null || echo "     (Role assignment may already exist)"

echo "   ✅ RBAC roles assigned"
echo ""

echo "=================================================="
echo "✅ Workload Identity Setup Complete!"
echo "=================================================="
echo ""
echo "Configuration Details:"
echo "  Identity Name:   $IDENTITY_NAME"
echo "  Client ID:       $CLIENT_ID"
echo "  Principal ID:    $PRINCIPAL_ID"
echo "  Subscription ID: $SUBSCRIPTION_ID"
echo "  OIDC Issuer:     $OIDC_ISSUER"
echo ""
echo "Export these for Helm deployment:"
echo "  export AZURE_CLIENT_ID=$CLIENT_ID"
echo "  export AZURE_TENANT_ID=$TENANT_ID"
echo "  export AZURE_SUBSCRIPTION_ID=$SUBSCRIPTION_ID"
echo ""
echo "Use these values in Helm deployment:"
echo "  helm install aks-mcp ../../../chart \\"
echo "    --set azure.tenantId=$TENANT_ID \\"
echo "    --set azure.clientId=$CLIENT_ID \\"
echo "    --set azure.subscriptionId=$SUBSCRIPTION_ID \\"
echo "    --set workloadIdentity.enabled=true \\"
echo "    --set app.transport=streamable-http \\"
echo "    --set app.accessLevel=readonly \\"
echo "    --set-json 'config.enabledComponents=[\"compute\",\"az_cli\",\"kubectl\"]'"
echo ""
echo "Verify federated credential:"
echo "  az identity federated-credential list \\"
echo "    --identity-name $IDENTITY_NAME \\"
echo "    --resource-group $RESOURCE_GROUP \\"
echo "    --output table"
echo ""
echo "=================================================="
