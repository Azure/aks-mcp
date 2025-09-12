#!/bin/bash
# Script to deploy AKS-MCP using Helm

set -e

ENVIRONMENT=${1:-dev}

# Special handling for mcp-inspector to use consistent naming
if [[ "$ENVIRONMENT" == "mcp-inspector" ]]; then
    NAMESPACE="aks-mcp-inspector"
    RELEASE_NAME="aks-mcp-inspector"
else
    NAMESPACE="aks-mcp-${ENVIRONMENT}"
    RELEASE_NAME="aks-mcp-${ENVIRONMENT}"
fi

CHART_PATH="./chart"

echo "Deploying AKS-MCP to environment: $ENVIRONMENT"
echo "Namespace: $NAMESPACE"
echo "Release: $RELEASE_NAME"

# Validate environment
if [[ ! -f "${CHART_PATH}/values-${ENVIRONMENT}.yaml" ]]; then
    echo "Error: Environment '$ENVIRONMENT' not found"
    echo "Available environments:"
    ls -1 ${CHART_PATH}/values-*.yaml | sed 's/.*values-\(.*\)\.yaml/\1/'
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    echo "Error: helm is not installed"
    echo "Please install helm: https://helm.sh/docs/intro/install/"
    exit 1
fi

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed"
    exit 1
fi

# Create namespace if it doesn't exist
echo "Creating namespace if it doesn't exist..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Lint the chart
echo "Linting Helm chart..."
helm lint "$CHART_PATH" -f "${CHART_PATH}/values-${ENVIRONMENT}.yaml"

# Run helm template to check for issues
echo "Validating Helm templates..."
helm template "$RELEASE_NAME" "$CHART_PATH" \
    --namespace "$NAMESPACE" \
    -f "${CHART_PATH}/values-${ENVIRONMENT}.yaml" \
    --dry-run > /dev/null

# Deploy with Helm
echo "Deploying with Helm..."
helm upgrade --install "$RELEASE_NAME" "$CHART_PATH" \
    --namespace "$NAMESPACE" \
    --create-namespace \
    -f "${CHART_PATH}/values-${ENVIRONMENT}.yaml" \
    --wait \
    --timeout=300s

# Show the status
echo "Deployment completed successfully!"
echo ""
echo "Release information:"
helm status "$RELEASE_NAME" -n "$NAMESPACE"
echo ""
echo "To check the status:"
echo "  kubectl get pods -n $NAMESPACE"
echo "  kubectl logs -f deployment/${RELEASE_NAME} -n $NAMESPACE"
echo ""

# Show service information
SERVICE_TYPE=$(kubectl get service "$RELEASE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.type}')
if [[ "$SERVICE_TYPE" == "NodePort" ]]; then
    NODE_PORT=$(kubectl get service "$RELEASE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.ports[0].nodePort}')
    SERVICE_PORT=$(kubectl get service "$RELEASE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.ports[0].port}')
    echo "Service is available via NodePort: $NODE_PORT"
    echo "To access locally: kubectl port-forward service/${RELEASE_NAME} ${SERVICE_PORT}:${SERVICE_PORT} -n $NAMESPACE"
elif [[ "$SERVICE_TYPE" == "LoadBalancer" ]]; then
    echo "Service is available via LoadBalancer"
    echo "External IP: $(kubectl get service "$RELEASE_NAME" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
else
    echo "Service is available via ClusterIP"
    echo "To access locally: kubectl port-forward service/${RELEASE_NAME} 8000:8000 -n $NAMESPACE"
fi