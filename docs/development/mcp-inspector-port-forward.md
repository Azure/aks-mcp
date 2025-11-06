# MCP Inspector Deployment Guide

This guide explains how to deploy AKS-MCP for use with MCP Inspector tool, enabling interactive testing and development of MCP connections.

## Overview

MCP Inspector is a debugging tool that allows you to interact with MCP servers through a web interface. This deployment configuration sets up AKS-MCP in a way that's optimized for Inspector connections.

## Prerequisites

- Kubernetes cluster with kubectl access
- Helm 3.8+ installed
- Docker (for building custom images, optional)
- Node.js and npm (for running MCP Inspector)

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   MCP Inspector │    │   Port Forward   │    │    AKS-MCP     │
│  localhost:6274 │◄──►│ localhost:8081   │◄──►│   Pod:8081     │
│                 │    │                  │    │  (Kubernetes)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

- **MCP Inspector**: Runs locally on port 6274
- **Port Forward**: kubectl port-forward maps local 8081 to pod 8081
- **OAuth Flow**: Inspector redirects to localhost:6274/oauth/callback

## Step-by-Step Deployment

### 1. Install MCP Inspector

```bash
# Install MCP Inspector globally
npm install -g @modelcontextprotocol/inspector

# Or run without installing
npx @modelcontextprotocol/inspector
```

### 2. Deploy AKS-MCP to Kubernetes

```bash
# Navigate to the chart directory
cd chart/

# Deploy using the inspector-optimized configuration
helm upgrade --install aks-mcp-inspector . \
  --namespace aks-mcp-inspector \
  --create-namespace \
  -f ./values-mcp-inspector.yaml
```

### 3. Verify Deployment

```bash
# Check pod status
kubectl get pods -n aks-mcp-inspector

# Check service
kubectl get svc -n aks-mcp-inspector

# View logs
kubectl logs -f deployment/aks-mcp-inspector -n aks-mcp-inspector
```

### 4. Set Up Port Forwarding

```bash
# Forward local port 8081 to the service
kubectl port-forward service/aks-mcp-inspector 8081:8081 -n aks-mcp-inspector
```

Keep this terminal open - the port-forward must remain active for Inspector to connect.

### 5. Start MCP Inspector

In a new terminal:

```bash
# Start MCP Inspector
npx @modelcontextprotocol/inspector
```

This will open a web browser at `http://localhost:6274`.

### 6. Connect Inspector to AKS-MCP

In the MCP Inspector web interface:

1. **Server URL**: `http://localhost:8081/mcp`
2. **Transport**: Select "Streamable HTTP" and Click "Connect"
3. **Authentication**: The OAuth flow will automatically redirect to localhost:6274

## Configuration Details

### OAuth Settings

The `values-mcp-inspector.yaml` configuration includes:

```yaml
oauth:
  enabled: false
  tenantId: "" ## YOUR TENEANT ID
  clientId: "" ## YOUR CLIENT ID
  redirectURIs:
    - "http://localhost:6274/oauth/callback"
  corsOrigins:
    - "http://localhost:6274"
```

## Alternative Deployment Methods

### Using Deployment Script

```bash
# From project root directory
./chart/scripts/deploy.sh mcp-inspector
```

Note: You'll need to create a corresponding `values-mcp-inspector.yaml` file.

### Direct kubectl with port-forward

```bash
# Alternative port-forward syntax
kubectl port-forward deployment/aks-mcp-inspector 8081:8081 -n aks-mcp-inspector

# Or port-forward to a specific pod
kubectl port-forward pod/<pod-name> 8081:8081 -n aks-mcp-inspector
```

## Development Workflow

### Typical Development Session

1. **Start Infrastructure**:
   ```bash
   # Deploy AKS-MCP
   helm upgrade --install aks-mcp-inspector . \
     --namespace aks-mcp-inspector \
     -f ./values-mcp-inspector.yaml
 
   # Start port-forward
   kubectl port-forward service/aks-mcp-inspector 8081:8081 -n aks-mcp-inspector &
   ```

2. **Start Inspector**:
   ```bash
   npx @modelcontextprotocol/inspector
   ```

3. **Develop and Test**:
   - Use Inspector web interface to test MCP calls
   - Monitor AKS-MCP logs: `kubectl logs -f deployment/aks-mcp-inspector -n aks-mcp-inspector`

4. **Update Configuration**:
   ```bash
   # Apply configuration changes
   helm upgrade aks-mcp-inspector . \
     --namespace aks-mcp-inspector \
     -f ./values-mcp-inspector.yaml
   ```

### Clean Up

```bash
# Remove the deployment
helm uninstall aks-mcp-inspector -n aks-mcp-inspector

# Remove the namespace
kubectl delete namespace aks-mcp-inspector

# Stop port-forward
pkill -f "kubectl port-forward"
```
