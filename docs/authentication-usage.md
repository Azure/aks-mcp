# AKS MCP Authentication

This guide explains how to set up and use Microsoft Entra ID (Azure AD) authentication with the AKS MCP server.

## Overview

AKS MCP supports JWT token authentication using Microsoft Entra ID. This provides secure access control for all MCP operations including tool calls and resource access.

## Prerequisites

- Azure subscription with appropriate permissions
- Azure CLI installed and configured
- AKS MCP server built and ready to deploy

## Setup

### 1. Create Azure AD Application

```bash
# Login to Azure
az login

# Create application registration
APP_NAME="aks-mcp-app"
az ad app create \
  --display-name "$APP_NAME" \
  --sign-in-audience AzureADMyOrg

# Get application ID
APP_ID=$(az ad app list --display-name "$APP_NAME" --query "[0].appId" -o tsv)
echo "Application ID: $APP_ID"

# Create service principal
az ad sp create --id $APP_ID

# Get tenant ID
TENANT_ID=$(az account show --query tenantId -o tsv)
echo "Tenant ID: $TENANT_ID"
```

### 2. Configure Application (Choose One Option)

#### Option A: Minimal Setup (Recommended)
```bash
# No additional configuration needed
# Tokens will use Application ID as audience
echo "Using scope: $APP_ID/.default"
```

#### Option B: Custom API Setup
```bash
# Set application identifier URI
az ad app update --id $APP_ID --identifier-uris "api://$APP_ID"

# Add API scope
cat > api-scope.json << EOF
{
  "oauth2PermissionScopes": [
    {
      "id": "$(uuidgen)",
      "adminConsentDescription": "Access AKS MCP API",
      "adminConsentDisplayName": "AKS MCP API Access",
      "userConsentDescription": "Access AKS MCP API",
      "userConsentDisplayName": "AKS MCP API Access",
      "value": "api.access",
      "type": "User",
      "isEnabled": true
    }
  ]
}
EOF

az ad app update --id $APP_ID --set api=@api-scope.json
rm api-scope.json

echo "Using scope: api://$APP_ID/.default"
```

### 3. Create Client Credentials

```bash
# Create client secret for applications
CLIENT_SECRET=$(az ad app credential reset --id $APP_ID --query password -o tsv)
echo "Client Secret: $CLIENT_SECRET"

# Save these values securely
echo "Save these values:"
echo "TENANT_ID=$TENANT_ID"
echo "APP_ID=$APP_ID"
echo "CLIENT_SECRET=$CLIENT_SECRET"
```

## Server Configuration

### Start Server with Authentication

#### HTTP Transport (streamable-http)
```bash
# Start AKS MCP server with HTTP transport and authentication
./aks-mcp \
  --auth-enabled \
  --auth-tenant-id="$TENANT_ID" \
  --auth-client-id="$APP_ID" \
  --transport="streamable-http" \
  --host="0.0.0.0" \
  --port="8080"
```

#### SSE Transport (Server-Sent Events)
```bash
# Start AKS MCP server with SSE transport and authentication
./aks-mcp \
  --auth-enabled \
  --auth-tenant-id="$TENANT_ID" \
  --auth-client-id="$APP_ID" \
  --transport="sse" \
  --host="0.0.0.0" \
  --port="8080"
```

### Environment Variables (Alternative)

```bash
# Set environment variables
export AKS_MCP_AUTH_ENABLED=true
export AKS_MCP_AUTH_TENANT_ID="$TENANT_ID"
export AKS_MCP_AUTH_CLIENT_ID="$APP_ID"

# Start server with HTTP transport
./aks-mcp --transport="streamable-http" --host="0.0.0.0" --port="8080"

# Or start server with SSE transport
./aks-mcp --transport="sse" --host="0.0.0.0" --port="8080"
```

## Client Usage

### 1. Obtain Access Token

```bash
# Get access token using client credentials flow
TOKEN_RESPONSE=$(curl -s -X POST \
  "https://login.microsoftonline.com/$TENANT_ID/oauth2/v2.0/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=$APP_ID" \
  -d "client_secret=$CLIENT_SECRET" \
  -d "scope=$APP_ID/.default") # or "api://$APP_ID" if identifier URI is configured

# Extract access token
ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')
```

### 2. Transport-Specific Usage

The authentication flow differs between transport types. Both use the same JWT tokens but have different communication patterns.

## HTTP Transport (streamable-http)

### Initialize MCP Session

```bash
# Initialize session to get session ID
INIT_RESPONSE=$(curl -s -i \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"jsonrpc": "2.0", "method": "initialize", "id": 1}' \
  "http://localhost:8080/mcp")

# Extract session ID from headers
SESSION_ID=$(echo "$INIT_RESPONSE" | grep -i "mcp-session-id:" | cut -d: -f2 | tr -d ' \r\n')
```

### Call MCP Methods

#### List Available Tools
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{"jsonrpc": "2.0", "method": "tools/list", "id": 1}' \
  "http://localhost:8080/mcp"
```

#### Call a Tool
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "kubectl_cluster",
      "arguments": {
        "operation": "cluster-info",
        "resource": "",
        "args": ""
      }
    },
    "id": 2
  }' \
  "http://localhost:8080/mcp"
```

## SSE Transport (Server-Sent Events)

### Initialize SSE Connection

```bash
# Start SSE connection to get session ID
# ⚠️  IMPORTANT: Run this command in a DEDICATED TERMINAL/CONSOLE
# This command will block and show the session ID, then keep the connection open
curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
  "http://localhost:8080/sse"
```


**Example output:**
```
event: endpoint
data: /message?sessionId=c1f2da9b-8ab0-4686-bd57-5441facde36b
```

**Manual Step:** Copy the session ID from the output above. In this example, the SESSION_ID is:
```bash
SESSION_ID="c1f2da9b-8ab0-4686-bd57-5441facde36b"
```

**Set your session ID:**
```bash
# Replace with your actual session ID from the SSE output
export SESSION_ID="your-session-id-here"
```

**⚠️ CRITICAL: Keep the SSE connection running!**
- **DO NOT** close the terminal running the SSE connection
- The SSE connection must stay active to receive responses
- Responses to `/message` requests will appear in the SSE terminal
- Use a separate terminal for sending `/message` requests

### Send Messages via Message Endpoint

**Important:** All responses will appear in the SSE terminal (where you ran the `/sse` connection), not in the terminal where you send the `/message` requests.

#### Initialize Session
```bash
# Initialize MCP session via message endpoint with session ID
# Response will appear in the SSE terminal
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "method": "initialize",
    "id": 1,
    "params": {
      "protocolVersion": "2024-11-05",
      "clientInfo": {
        "name": "aks-mcp-client",
        "version": "1.0.0"
      }
    }
  }' \
  "http://localhost:8080/message?sessionId=$SESSION_ID"
```

#### List Available Tools
```bash
# Response will appear in the SSE terminal
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"jsonrpc": "2.0", "method": "tools/list", "id": 2}' \
  "http://localhost:8080/message?sessionId=$SESSION_ID"
```

#### Call a Tool
```bash
# Response will appear in the SSE terminal
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "kubectl_cluster",
      "arguments": {
        "operation": "cluster-info",
        "resource": "",
        "args": ""
      }
    },
    "id": 3
  }' \
  "http://localhost:8080/message?sessionId=$SESSION_ID"
```

#### Cleanup SSE Connection
```bash
# Stop the background SSE connection when done (if started)
if [ ! -z "$SSE_PID" ]; then
  kill $SSE_PID 2>/dev/null
  echo "SSE connection stopped"
fi
```

## Transport Comparison

| Feature | HTTP (streamable-http) | SSE (Server-Sent Events) |
|---------|----------------------|--------------------------|
| **Endpoints** | `/mcp` | `/sse` (receive) + `/message` (send) |
| **Communication** | Request-Response | Event-driven |
| **Session Management** | Session ID header | Query parameter sessionId |
| **Authentication** | Bearer token + Session ID | Bearer token per request |
| **Use Case** | Traditional API clients | Real-time applications |
| **Complexity** | Low | Medium |

## Authentication Flow Summary

1. **Token Acquisition**: Same OAuth2 client credentials flow for both transports
2. **Authorization**: Bearer token in `Authorization` header for all requests
3. **Session Management**: 
   - HTTP: Session ID header for subsequent requests
   - SSE: Session ID as query parameter, obtained from SSE endpoint
4. **Security**: JWT token validation with Microsoft Entra ID integration
