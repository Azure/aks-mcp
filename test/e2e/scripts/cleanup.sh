#!/bin/bash
set -e

# Configuration
RESOURCE_GROUP="${RESOURCE_GROUP:-aks-mcp-e2e-test-rg}"
CLUSTER_NAME="${CLUSTER_NAME:-aks-mcp-e2e-test}"
IDENTITY_NAME="${IDENTITY_NAME:-aks-mcp-e2e-identity}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"

echo "=================================================="
echo "Cleaning up E2E Test Resources"
echo "=================================================="
echo "Resource Group: $RESOURCE_GROUP"
echo "Cluster Name:   $CLUSTER_NAME"
echo "Identity Name:  $IDENTITY_NAME"
echo "=================================================="
echo ""

if [ "$SKIP_CLEANUP" = "true" ]; then
    echo "⏭️  SKIP_CLEANUP is set to true, skipping resource cleanup"
    echo "   Resources will remain for debugging"
    echo ""
    echo "To clean up later, run:"
    echo "  SKIP_CLEANUP=false ./cleanup.sh"
    exit 0
fi

# Confirm deletion
read -p "⚠️  This will delete the resource group and all resources. Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled"
    exit 0
fi

echo ""

# Delete AKS cluster
echo "🗑️  Deleting AKS cluster..."
if az aks show --resource-group "$RESOURCE_GROUP" --name "$CLUSTER_NAME" &> /dev/null; then
    az aks delete \
        --resource-group "$RESOURCE_GROUP" \
        --name "$CLUSTER_NAME" \
        --yes \
        --no-wait
    echo "   ✅ Cluster deletion initiated (running in background)"
else
    echo "   ⏭️  Cluster not found, skipping"
fi
echo ""

# Delete Azure Managed Identity
echo "🗑️  Deleting Azure Managed Identity..."
if az identity show --resource-group "$RESOURCE_GROUP" --name "$IDENTITY_NAME" &> /dev/null; then
    PRINCIPAL_ID=$(az identity show \
        --resource-group "$RESOURCE_GROUP" \
        --name "$IDENTITY_NAME" \
        --query "principalId" \
        --output tsv)

    SUBSCRIPTION_ID=$(az account show --query id -o tsv)

    # Delete role assignments first
    echo "   - Deleting role assignments..."
    az role assignment delete \
        --assignee "$PRINCIPAL_ID" \
        --scope "/subscriptions/$SUBSCRIPTION_ID" \
        --output none 2>/dev/null || true

    # Delete the managed identity
    az identity delete \
        --resource-group "$RESOURCE_GROUP" \
        --name "$IDENTITY_NAME" \
        --output none
    echo "   ✅ Managed Identity deleted"
else
    echo "   ⏭️  Managed Identity not found, skipping"
fi
echo ""

# Delete resource group
echo "🗑️  Deleting resource group (this may take a few minutes)..."
if az group show --name "$RESOURCE_GROUP" &> /dev/null; then
    az group delete \
        --name "$RESOURCE_GROUP" \
        --yes \
        --no-wait
    echo "   ✅ Resource group deletion initiated (running in background)"
else
    echo "   ⏭️  Resource group not found, skipping"
fi
echo ""

echo "=================================================="
echo "✅ Cleanup Complete!"
echo "=================================================="
echo ""
echo "Resources are being deleted in the background."
echo "This may take 5-10 minutes to complete."
echo ""
echo "To check deletion status:"
echo "  az group show --name $RESOURCE_GROUP"
echo "  (Should return 'ResourceGroupNotFound' when complete)"
echo ""
echo "To verify managed identity deletion:"
echo "  az identity show --resource-group $RESOURCE_GROUP --name $IDENTITY_NAME"
echo "  (Should return 'not found' when complete)"
echo "=================================================="
