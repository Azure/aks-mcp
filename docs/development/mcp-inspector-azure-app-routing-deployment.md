# Azure App Routing Deployment Guide

This guide explains how to deploy AKS-MCP using Azure App Routing with the `chart/values-mcp-inspector-azure-approuting.yaml` configuration file.

## Overview

Azure App Routing provides a managed nginx ingress controller that eliminates the need for manual ingress setup. This deployment configuration enables external access to AKS-MCP through a LoadBalancer IP with OAuth authentication.

## Prerequisites

### 1. AKS Cluster with Azure App Routing
Enable Azure App Routing on your AKS cluster:

```bash
az aks approuting enable \
    --resource-group <your-resource-group> \
    --name <your-cluster-name>
```

### 2. Azure AD App Registration
Create an Azure AD app registration for OAuth:

```bash
az ad app create \
    --display-name "AKS-MCP Azure App Routing" \
    --web-redirect-uris "http://localhost:6274/oauth/callback" \
    --web-redirect-uris "http://localhost:6274/oauth/callback/debug"
```

Note the `appId` and `tenant` from the output.

## Configuration

### 1. Get Azure App Routing IP
First, obtain the external IP of the Azure App Routing nginx service:

```bash
AZURE_APP_ROUTING_IP=$(kubectl get service nginx \
    -n app-routing-system \
    -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "Azure App Routing IP: $AZURE_APP_ROUTING_IP"
```

### 2. Update Values File
Edit `chart/values-mcp-inspector-azure-approuting.yaml` and replace the following:

- Replace `YOUR_TENANT_ID` with your actual tenant ID
- Replace `YOUR_CLIENT_ID` with your actual client ID  
- Replace `AZURE_APP_ROUTING_IP` placeholders with the actual IP from step 1

```yaml
oauth:
  tenantId: "YOUR_ACTUAL_TENANT_ID"
  clientId: "YOUR_ACTUAL_CLIENT_ID"
  redirectURIs:
    - "http://localhost:6274/oauth/callback"
    - "http://localhost:6274/oauth/callback/debug"
    - "http://YOUR_AZURE_APP_ROUTING_IP/oauth/callback"
  corsOrigins:
    - "http://localhost:6274"
    - "http://YOUR_AZURE_APP_ROUTING_IP"
```

### 3. Update Azure AD App Registration
Add the Azure App Routing redirect URI to your app registration:

```bash
az ad app update \
    --id <YOUR_APP_ID> \
    --add web.redirectUris "http://$AZURE_APP_ROUTING_IP/oauth/callback"
```

## Deployment

### 1. Deploy with Helm
```bash
helm install aks-mcp ./chart \
    -f chart/values-mcp-inspector-azure-approuting.yaml \
    --namespace aks-mcp \
    --create-namespace
```

### 2. Verify Deployment
Check pods status:
```bash
kubectl get pods -n aks-mcp -l app.kubernetes.io/name=aks-mcp
```

Check ingress:
```bash
kubectl get ingress -n aks-mcp
```

### 3. Test Access
Access AKS-MCP at:
```
http://<AZURE_APP_ROUTING_IP>/
```

## Key Configuration Features

### Ingress Setup
The values file configures Azure App Routing ingress:

```yaml
ingress:
  enabled: true
  ingressClassName: "webapprouting.kubernetes.azure.com"
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
    nginx.ingress.kubernetes.io/enable-cors: "true"
  hosts:
    - host: ""  # Empty to use IP directly
      paths:
        - path: /
          pathType: Prefix
```

### CORS Configuration
CORS is configured for both local and remote access:

```yaml
annotations:
  nginx.ingress.kubernetes.io/cors-allow-origin: "http://localhost:6274, http://AZURE_APP_ROUTING_IP"
  nginx.ingress.kubernetes.io/cors-allow-methods: "GET, POST, OPTIONS, PUT, DELETE"
  nginx.ingress.kubernetes.io/cors-allow-headers: "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,mcp-protocol-version,Content-Length,Accept,Accept-Encoding"
  nginx.ingress.kubernetes.io/cors-expose-headers: "mcp-protocol-version"
```

### OAuth Authentication
OAuth is configured for Azure AD authentication:

```yaml
oauth:
  enabled: true
  tenantId: "YOUR_TENANT_ID"
  clientId: "YOUR_CLIENT_ID"
  redirectURIs:
    - "http://localhost:6274/oauth/callback"
    - "http://localhost:6274/oauth/callback/debug"
    - "http://AZURE_APP_ROUTING_IP/oauth/callback"
```

## Using with MCP Inspector

### 1. Start MCP Inspector
```bash
npx @modelcontextprotocol/inspector
```

### 2. Connect to AKS-MCP
- Open browser to `http://localhost:6274`
- Enter server URL: `http://<AZURE_APP_ROUTING_IP>/mcp`
- Complete OAuth authentication flow

## Troubleshooting

### Common Issues

**CORS Errors**
- Verify Azure App Routing IP is correctly configured in values file
- Check CORS annotations in ingress
- Ensure redirect URIs match in Azure AD app registration

**OAuth Failures**
- Verify tenant ID and client ID are correct
- Check redirect URIs in Azure AD app registration
- Ensure OAuth endpoints are accessible

### Debug Commands

Check ingress details:
```bash
kubectl describe ingress -n aks-mcp
```

Check pod logs:
```bash
kubectl logs -n aks-mcp -l app.kubernetes.io/name=aks-mcp
```

Test health endpoint:
```bash
curl http://$AZURE_APP_ROUTING_IP/health
```
