# OAuth Authentication for AKS-MCP

This document describes how to configure and use OAuth authentication with AKS-MCP.

## Overview

AKS-MCP now supports OAuth 2.1 authentication using Azure Active Directory as the authorization server. When enabled, OAuth authentication provides secure access control for MCP endpoints using Bearer tokens.

## Features

- **Azure AD Integration**: Uses Azure Active Directory as the OAuth authorization server
- **JWT Token Validation**: Validates JWT tokens with Azure AD signing keys
- **OAuth 2.0 Metadata Endpoints**: Provides standard OAuth metadata discovery endpoints
- **Dynamic Client Registration**: Supports RFC 7591 dynamic client registration
- **Token Introspection**: Implements RFC 7662 token introspection
- **Transport Support**: Works with both SSE and HTTP Streamable transports
- **Flexible Configuration**: Supports environment variables and command-line configuration

## Quick Start

### 1. Azure AD Application Setup

First, create an Azure AD application:

```bash
# Create Azure AD application
az ad app create --display-name "AKS-MCP-OAuth" \
  --web-redirect-uris "http://localhost:3000/oauth/callback"

# Get application details
az ad app list --display-name "AKS-MCP-OAuth" --query "[0].{appId:appId,objectId:objectId}"
```

### 2. Environment Configuration

Set the required environment variables:

```bash
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-app-id"
```

### 3. Start AKS-MCP with OAuth

```bash
# Using SSE transport with OAuth
./aks-mcp --transport=sse --oauth-enabled=true

# Using HTTP Streamable transport with OAuth
./aks-mcp --transport=streamable-http --oauth-enabled=true
```

## Configuration Options

### Command Line Flags

- `--oauth-enabled`: Enable OAuth authentication (default: false)
- `--oauth-tenant-id`: Azure AD tenant ID (or use AZURE_TENANT_ID env var)
- `--oauth-client-id`: Azure AD client ID (or use AZURE_CLIENT_ID env var)
- `--oauth-scopes`: Comma-separated list of required scopes
- `--oauth-redirects`: Comma-separated list of allowed redirect URIs

### Example with Command Line Flags

```bash
./aks-mcp --transport=sse --oauth-enabled=true \
  --oauth-tenant-id="12345678-1234-1234-1234-123456789012" \
  --oauth-client-id="87654321-4321-4321-4321-210987654321" \
  --oauth-scopes="https://management.azure.com/.default" \
  --oauth-redirects="http://localhost:3000/oauth/callback,https://myapp.com/callback"
```

## OAuth Endpoints

When OAuth is enabled, the following endpoints are available:

### Metadata Endpoints (Unauthenticated)

- `GET /.well-known/oauth-protected-resource` - OAuth 2.0 Protected Resource Metadata (RFC 9728)
- `GET /.well-known/oauth-authorization-server` - OAuth 2.0 Authorization Server Metadata (RFC 8414)
- `GET /health` - Health check endpoint

### Client Registration (Unauthenticated)

- `POST /oauth/register` - Dynamic Client Registration (RFC 7591)

### Token Introspection (Unauthenticated for simplicity)

- `POST /oauth/introspect` - Token Introspection (RFC 7662)

### Authenticated MCP Endpoints

When OAuth is enabled, these endpoints require Bearer token authentication:

- **SSE Transport**: `GET /sse`, `POST /message`  
- **HTTP Streamable Transport**: `POST /mcp`

## Client Integration

### Obtaining an Access Token

Use the Azure AD OAuth flow to obtain an access token:

```bash
# Example using Azure CLI (for testing)
az account get-access-token --resource https://management.azure.com/ --query accessToken -o tsv
```

### Making Authenticated Requests

Include the Bearer token in the Authorization header:

```bash
# Example authenticated request to SSE endpoint
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Accept: text/event-stream" \
  http://localhost:8000/sse

# Example authenticated request to HTTP Streamable endpoint  
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -X POST http://localhost:8000/mcp \
  -d '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}'
```

## Testing OAuth Integration

### 1. Test OAuth Metadata

```bash
# Get protected resource metadata
curl http://localhost:8000/.well-known/oauth-protected-resource

# Get authorization server metadata
curl http://localhost:8000/.well-known/oauth-authorization-server
```

### 2. Test Dynamic Client Registration

```bash
curl -X POST http://localhost:8000/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "redirect_uris": ["http://localhost:3000/oauth/callback"],
    "client_name": "Test MCP Client",
    "grant_types": ["authorization_code"]
  }'
```

### 3. Test Token Introspection

```bash
curl -X POST http://localhost:8000/oauth/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=YOUR_ACCESS_TOKEN"
```

## Security Considerations

1. **HTTPS in Production**: Always use HTTPS in production environments
2. **Token Validation**: JWT tokens are validated against Azure AD signing keys
3. **Scope Validation**: Tokens must include required scopes
4. **Audience Validation**: Tokens must have the correct audience claim
5. **Redirect URI Validation**: Only configured redirect URIs are allowed

## Troubleshooting

### Common Issues

1. **Invalid Token**: Ensure the token is valid and not expired
2. **Wrong Audience**: Verify the token audience matches configuration
3. **Missing Scopes**: Ensure the token includes required scopes
4. **Network Issues**: Check connectivity to Azure AD endpoints

### Debug Logging

Enable verbose logging for OAuth debugging:

```bash
./aks-mcp --oauth-enabled=true --verbose
```

### Health Check

Use the health endpoint to verify OAuth configuration:

```bash
curl http://localhost:8000/health
```

## Migration from Non-OAuth

To migrate from a non-OAuth AKS-MCP deployment:

1. Update clients to obtain and include Bearer tokens
2. Enable OAuth on the server with `--oauth-enabled=true`
3. Configure Azure AD application and credentials
4. Test with a subset of clients before full migration
5. Monitor logs for authentication errors

## Integration with MCP Inspector

The MCP Inspector tool can be used to test OAuth-enabled AKS-MCP servers. Configure the Inspector's OAuth settings to match your AKS-MCP OAuth configuration for testing.

For more information, see the MCP OAuth specification and Azure AD documentation.