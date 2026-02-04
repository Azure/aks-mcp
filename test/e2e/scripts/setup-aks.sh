#!/bin/bash
set -e

# Configuration
RESOURCE_GROUP="${RESOURCE_GROUP:-aks-mcp-e2e-test-rg}"
CLUSTER_NAME="${CLUSTER_NAME:-aks-mcp-e2e-test}"
LOCATION="${LOCATION:-eastus}"
NODE_COUNT="${NODE_COUNT:-2}"
NODE_VM_SIZE="${NODE_VM_SIZE:-Standard_DS2_v2}"

echo "=================================================="
echo "Creating AKS Cluster for E2E Testing"
echo "=================================================="
echo "Resource Group: $RESOURCE_GROUP"
echo "Cluster Name:   $CLUSTER_NAME"
echo "Location:       $LOCATION"
echo "Node Count:     $NODE_COUNT"
echo "Node VM Size:   $NODE_VM_SIZE"
echo "=================================================="
echo ""

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    echo "❌ Error: Azure CLI is not installed"
    echo "   Please install it from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi

# Check if logged in
if ! az account show &> /dev/null; then
    echo "❌ Error: Not logged in to Azure CLI"
    echo "   Please run: az login"
    exit 1
fi

# Get current subscription
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
echo "📋 Using Azure subscription: $SUBSCRIPTION_ID"
echo ""

# Create resource group
echo "📦 Creating resource group..."
if az group show --name "$RESOURCE_GROUP" &> /dev/null; then
    echo "   ⚠️  Resource group already exists, skipping creation"
else
    az group create \
        --name "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --output none
    echo "   ✅ Resource group created"
fi
echo ""

# Create AKS cluster with Workload Identity
echo "🚀 Creating AKS cluster (this may take 10-15 minutes)..."
if az aks show --resource-group "$RESOURCE_GROUP" --name "$CLUSTER_NAME" &> /dev/null; then
    echo "   ⚠️  AKS cluster already exists, skipping creation"
else
    az aks create \
        --resource-group "$RESOURCE_GROUP" \
        --name "$CLUSTER_NAME" \
        --location "$LOCATION" \
        --node-count "$NODE_COUNT" \
        --node-vm-size "$NODE_VM_SIZE" \
        --enable-managed-identity \
        --enable-workload-identity \
        --enable-oidc-issuer \
        --network-plugin azure \
        --generate-ssh-keys \
        --output none

    echo "   ✅ AKS cluster created"
fi
echo ""

# Get cluster credentials
echo "🔑 Getting cluster credentials..."
az aks get-credentials \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CLUSTER_NAME" \
    --overwrite-existing \
    --output none
echo "   ✅ Credentials configured"
echo ""

# Get OIDC issuer URL
OIDC_ISSUER=$(az aks show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CLUSTER_NAME" \
    --query "oidcIssuerProfile.issuerUrl" \
    -o tsv)

echo "=================================================="
echo "✅ AKS Cluster Setup Complete!"
echo "=================================================="
echo ""
echo "Cluster Information:"
echo "  Resource Group:  $RESOURCE_GROUP"
echo "  Cluster Name:    $CLUSTER_NAME"
echo "  Subscription ID: $SUBSCRIPTION_ID"
echo "  OIDC Issuer:     $OIDC_ISSUER"
echo ""
echo "Export these for next steps:"
echo "  export AZURE_SUBSCRIPTION_ID=$SUBSCRIPTION_ID"
echo "  export RESOURCE_GROUP=$RESOURCE_GROUP"
echo "  export CLUSTER_NAME=$CLUSTER_NAME"
echo "  export OIDC_ISSUER=$OIDC_ISSUER"
echo ""
echo "Verify cluster:"
echo "  kubectl get nodes"
echo ""
echo "Next steps:"
echo "  1. Run ./setup-workload-identity.sh to configure Workload Identity for MCP server"
echo "  2. Deploy MCP server using Helm"
echo "  3. Run E2E tests"
echo "=================================================="
