# OAuth Authentication for AKS-MCP

This document describes how to configure and use OAuth authentication with AKS-MCP.

## Overview

AKS-MCP now supports OAuth 2.1 authentication using Azure Active Directory as the authorization server. When enabled, OAuth authentication provides secure access control for MCP endpoints using Bearer tokens.

## Features

- **Azure AD Integration**: Uses Azure Active Directory as the OAuth authorization server
- **JWT Token Validation**: Validates JWT tokens with Azure AD signing keys
- **Multiple Authentication Methods**: Supports user tokens, Managed Identity (MI), and Service Principal (SPN) authentication
- **OAuth 2.0 Metadata Endpoints**: Provides standard OAuth metadata discovery endpoints
- **Dynamic Client Registration**: Supports RFC 7591 dynamic client registration
- **Token Introspection**: Implements RFC 7662 token introspection
- **Transport Support**: Works with both SSE and HTTP Streamable transports
- **Flexible Configuration**: Supports environment variables and command-line configuration

## Managed Identity and Service Principal Support

AKS-MCP supports authentication from Managed Identities and Service Principals, enabling automated access from CI/CD pipelines, Azure services, and machine-to-machine scenarios.

### Token Validation for MI/SPN

MI and SPN tokens differ from user tokens - they may have empty `scp` claims. AKS-MCP validates these tokens by:

1. Checking standard scopes (`scp` claim)
2. Checking application roles (`roles` claim: Reader, Contributor, Owner)
3. Falling back to audience validation for MI tokens with empty claims

### Example: GitHub Actions with Azure OIDC

```yaml
jobs:
  call-mcp:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: read
    steps:
      - uses: azure/login@v2
        with:
          client-id: ${{ secrets.AZURE_CLIENT_ID }}
          tenant-id: ${{ secrets.AZURE_TENANT_ID }}
          subscription-id: ${{ secrets.AZURE_SUBSCRIPTION_ID }}
      
      - name: Call MCP Server
        run: |
          TOKEN=$(az account get-access-token --resource https://management.azure.com/ --query accessToken -o tsv)
          curl -X POST https://your-mcp-server/mcp \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

### Example: Azure VM with Managed Identity

```bash
TOKEN=$(curl -s 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/' \
  -H 'Metadata: true' | jq -r '.access_token')

curl -X POST https://your-mcp-server/mcp \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

## Environment Setup and Azure AD Configuration

### Prerequisites

Before setting up OAuth authentication, ensure you have:

- Azure CLI installed and configured (`az login`)
- An Azure subscription with appropriate permissions to create applications
- Azure Active Directory tenant access

### Important: Environment Variables Shared Between OAuth and Azure CLI

AKS-MCP uses the same environment variables (`AZURE_TENANT_ID`, `AZURE_CLIENT_ID`) for both OAuth authentication and Azure CLI operations. This design provides configuration simplicity but requires careful permission setup:

**When `AZURE_CLIENT_ID` is set:**
- OAuth: Used for validating user tokens accessing the MCP server
- Azure CLI: Used for managed identity/workload identity authentication to access Azure resources

**Permission Requirements:**
- The Azure AD application must have both **OAuth permissions** (for user authentication) AND **Azure resource permissions** (for az CLI operations)
- Missing either set of permissions will cause authentication failures

### Step 1: Create Azure AD Application

#### Using Azure Portal (Recommended)

1. **Navigate to Azure Portal**
   - Go to https://portal.azure.com
   - Sign in with your Azure account

2. **Create App Registration**
   ```
   Navigation: Azure Active Directory → App registrations → New registration
   ```
   
   Configure the following:
   - **Name**: `AKS-MCP-OAuth` (or your preferred name)
   - **Supported account types**: "Accounts in this organizational directory only"
   - **Redirect URI Platform Options**:

#### Supported Platform Types

**✅ Mobile and desktop applications (Recommended)**
- **Platform**: "Mobile and desktop applications" 
- **Redirect URIs**:
  - `http://localhost:8000/oauth/callback`
- **Benefits**: 
  - Native support for PKCE (required by OAuth 2.1)
  - No client secret required (public client)
  - Better security for localhost redirects
- **Status**: ✅ **Confirmed working**

**❌ Single-page application (SPA) - Not Recommended**
- **Platform**: "Single-page application (SPA)"
- **Redirect URIs**: Same as above
- **Benefits**: 
  - Designed for PKCE flow
  - No client secret required
- **Critical Limitations**: 
  - **Token exchange restriction**: Azure AD error AADSTS9002327 - "Tokens issued for the 'Single-Page Application' client-type may only be redeemed via cross-origin requests"
  - **Architecture mismatch**: SPA platform expects frontend JavaScript to handle token exchange, but AKS-MCP performs backend token exchange
  - **CORS requirements**: Requires complex frontend-backend coordination for OAuth flow
- **Status**: ❌ **Not compatible with AKS-MCP's backend OAuth implementation**

**❌ Web application**
- **Platform**: "Web"
- **Why not supported**: 
  - Requires client secret (confidential client)
  - AKS-MCP implements public client flow without secrets
  - PKCE handling may differ

**Choose Platform Recommendation:**
1. **Primary**: Use "Mobile and desktop applications" (✅ confirmed working)
2. **Avoid**: "Single-page application" - incompatible with backend OAuth implementation (AADSTS9002327 error)
3. **Avoid**: "Web" platform due to client secret requirements

3. **Record Essential Information**
   From the "Overview" page, note:
   - **Application (client) ID** - This is your `CLIENT_ID`
   - **Directory (tenant) ID** - This is your `TENANT_ID`

#### Using Azure CLI (Alternative)

**For Mobile and desktop applications platform:**
```bash
# Create Azure AD application with public client platform
az ad app create --display-name "AKS-MCP-OAuth" \
  --public-client-redirect-uris "http://localhost:8000/oauth/callback"

# Get application details
az ad app list --display-name "AKS-MCP-OAuth" --query "[0].{appId:appId,objectId:objectId}"

# Get your tenant ID
az account show --query "tenantId" -o tsv
```

### Step 2: Configure API Permissions

**Critical: Both OAuth and Azure CLI require proper permissions**

1. **Add Required API Permissions**
   ```
   Navigation: Azure Active Directory → App registrations → [Your App] → API permissions
   ```

2. **Add Azure Service Management Permission (Required for OAuth)**
   - Click "Add a permission"
   - Select "Microsoft APIs" → "Azure Service Management"
   - Choose "Delegated permissions"
   - Select `user_impersonation`
   - Click "Add permissions"

3. **Add Azure Resource Management Permissions (Required for Azure CLI)**
   
   When `AZURE_CLIENT_ID` is set, Azure CLI will use this application for authentication. Add these permissions based on your AKS-MCP access level:
   
   **For readonly access:**
   - Microsoft Graph → Application permissions → `Directory.Read.All`
   - Azure Service Management → Delegated permissions → `user_impersonation`
   
   **For readwrite/admin access:**
   - Microsoft Graph → Application permissions → `Directory.Read.All`
   - Azure Service Management → Delegated permissions → `user_impersonation`
   - Consider adding specific Azure resource permissions based on your needs

4. **Grant Admin Consent (Required)**
   - Click "Grant admin consent for [Your Organization]"
   - Confirm the consent

**⚠️ Important Notes:**
- Without proper Azure CLI permissions, you'll see "Insufficient privileges" errors when AKS-MCP tries to access Azure resources
- The same application serves both OAuth authentication (user access to MCP) and Azure CLI authentication (MCP access to Azure)
- Test both OAuth flow AND Azure resource access after permission changes

### Step 3: Environment Configuration

Set the required environment variables:

```bash
# Replace with your actual values from Step 1
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_SUBSCRIPTION_ID="your-subscription-id"  # Optional, for AKS operations
```

**⚠️ Important: Dual Authentication Impact**

When you set `AZURE_CLIENT_ID`, it affects both OAuth and Azure CLI authentication:

1. **OAuth Authentication**: Validates user tokens for MCP server access
2. **Azure CLI Authentication**: AKS-MCP uses this client ID for managed identity authentication when accessing Azure resources

**Common Issues:**
- If you only configured OAuth permissions, Azure CLI operations will fail with "Insufficient privileges"
- If you only configured Azure resource permissions, OAuth token validation may fail
- Solution: Ensure your Azure AD application has BOTH sets of permissions (see Step 2)

**Testing Both Authentication Paths:**
```bash
# Test OAuth (should work after proper setup)
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8000/mcp

# Test Azure CLI access (should work after proper permissions)
# This happens automatically when AKS-MCP tries to access Azure resources
./aks-mcp --oauth-enabled --access-level=readonly
```

### Step 4: Start AKS-MCP with OAuth

```bash
# Using HTTP Streamable transport with OAuth (recommended)
./aks-mcp \
  --transport=streamable-http \
  --port=8000 \
  --oauth-enabled \
  --oauth-tenant-id="$AZURE_TENANT_ID" \
  --oauth-client-id="$AZURE_CLIENT_ID" \
  --oauth-redirects="http://localhost:8000/oauth/callback" \
  --access-level=readonly

# Using SSE transport with OAuth (alternative)
./aks-mcp \
  --transport=sse \
  --port=8000 \
  --oauth-enabled \
  --oauth-tenant-id="$AZURE_TENANT_ID" \
  --oauth-client-id="$AZURE_CLIENT_ID" \
  --oauth-redirects="http://localhost:8000/oauth/callback" \
  --access-level=readonly

# Environment variables are automatically used if set
# You can also just use:
./aks-mcp --transport=streamable-http --port=8000 --oauth-enabled --access-level=readonly
```

## Configuration Options

### Command Line Flags

- `--oauth-enabled`: Enable OAuth authentication (default: false)
- `--oauth-tenant-id`: Azure AD tenant ID (or use AZURE_TENANT_ID env var)
- `--oauth-client-id`: Azure AD client ID (or use AZURE_CLIENT_ID env var)
- `--oauth-redirects`: Comma-separated list of allowed redirect URIs (required when OAuth enabled)
- `--oauth-cors-origins`: Comma-separated list of allowed CORS origins for OAuth endpoints (e.g. http://localhost:6274 for MCP Inspector). If empty, no cross-origin requests are allowed for security
- `--oauth-scopes`: Comma-separated list of OAuth scopes to require (e.g., `api://your-app-id/.default`). If empty, defaults to `https://management.azure.com/.default`

## Restricted Scope Authentication (Assignment Required)

By default, AKS-MCP validates tokens issued for `https://management.azure.com/.default`. This means **any authenticated Azure AD user** can access your MCP server.

To restrict access to only users/identities explicitly assigned to your application, use the `--oauth-scopes` flag with a custom Application ID URI scope.

### Why Use Restricted Scopes?

| Scenario | Default Scope | Custom App ID URI Scope |
|----------|--------------|------------------------|
| Audience | `https://management.azure.com` | `api://your-app-id` |
| Who can access | Any Azure AD user | Only assigned users/groups/SPNs |
| Azure AD "Assignment Required" | Not enforced | Enforced |
| Token request | `az account get-access-token --resource https://management.azure.com/` | `az account get-access-token --resource api://your-app-id/` |

### Step 1: Configure Azure AD Application for Restricted Access

1. **Set Application ID URI**
   ```
   Azure Portal → App registrations → [Your App] → Expose an API → Set Application ID URI
   ```
   - Set to: `api://<your-client-id>` (e.g., `api://12345678-1234-1234-1234-123456789012`)

2. **Add a Scope** (required for user tokens)
   ```
   Azure Portal → App registrations → [Your App] → Expose an API → Add a scope
   ```
   - Scope name: `access_as_user`
   - Who can consent: Admins and users
   - Admin consent display name: "Access MCP Server"
   - Admin consent description: "Allows access to the MCP server"

3. **Enable Assignment Required**
   ```
   Azure Portal → Enterprise applications → [Your App] → Properties
   ```
   - Set **Assignment required?** to **Yes**

4. **Assign Users/Groups/Service Principals**
   ```
   Azure Portal → Enterprise applications → [Your App] → Users and groups → Add user/group
   ```
   - Add users, groups, or service principals that should have access

5. **Pre-authorize Azure CLI** (required for `az account get-access-token`)
   ```
   Azure Portal → App registrations → [Your App] → Expose an API → Add a client application
   ```
   - Client ID: `04b07795-8ddb-461a-bbee-02f9e1bf7b46` (Azure CLI)
   - Authorized scopes: Select your `access_as_user` scope

### Step 2: Start AKS-MCP with Custom Scope

```bash
# Using custom App ID URI scope for restricted access
./aks-mcp \
  --transport=streamable-http \
  --port=8000 \
  --oauth-enabled \
  --oauth-tenant-id="$AZURE_TENANT_ID" \
  --oauth-client-id="$AZURE_CLIENT_ID" \
  --oauth-scopes="api://$AZURE_CLIENT_ID/.default" \
  --oauth-redirects="http://localhost:8000/oauth/callback" \
  --access-level=readonly
```

### Step 3: Request Token with Custom Scope

**User authentication (Azure CLI):**
```bash
# Request token for your custom App ID URI
TOKEN=$(az account get-access-token --resource "api://<your-client-id>" --query accessToken -o tsv)

# Call MCP server
curl -X POST http://localhost:8000/mcp \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

**Managed Identity / Service Principal:**
```bash
# Using Azure CLI with MI
TOKEN=$(az account get-access-token --resource "api://<your-client-id>" --query accessToken -o tsv)

# Using IMDS (Azure VM)
TOKEN=$(curl -s 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=api://<your-client-id>' \
  -H 'Metadata: true' | jq -r '.access_token')
```

### Helm Chart Configuration

```yaml
# values.yaml
oauth:
  enabled: true
  tenantId: "your-tenant-id"
  clientId: "your-client-id"
  # Restrict to only assigned users/SPNs
  scopes:
    - "api://your-client-id/.default"
  redirectURIs:
    - "http://localhost:8000/oauth/callback"
```

### GitHub Actions with Restricted Scope

```yaml
jobs:
  call-mcp:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: read
    env:
      MCP_SERVER_URL: ${{ vars.MCP_SERVER_URL }}
      MCP_SCOPE: "api://${{ secrets.AZURE_CLIENT_ID }}/.default"
    steps:
      - uses: azure/login@v2
        with:
          client-id: ${{ secrets.AZURE_CLIENT_ID }}
          tenant-id: ${{ secrets.AZURE_TENANT_ID }}
          subscription-id: ${{ secrets.AZURE_SUBSCRIPTION_ID }}
      
      - name: Call MCP Server
        run: |
          # Request token for custom scope
          TOKEN=$(az account get-access-token --resource "${MCP_SCOPE%/.default}" --query accessToken -o tsv)
          curl -X POST "$MCP_SERVER_URL/mcp" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

### Token Validation Behavior

When `--oauth-scopes` is set to a custom App ID URI scope, AKS-MCP validates:

1. **Token audience** - Must match the resource (e.g., `api://your-app-id`)
2. **Scopes/Roles** - Checks for:
   - User scopes (`scp` claim): `access_as_user`, `user_impersonation`
   - App roles (`roles` claim): `Reader`, `Contributor`, `Owner`, `Admin`
3. **Fallback** - For MI tokens with empty claims, validates audience match

### Troubleshooting Restricted Scopes

| Error | Cause | Solution |
|-------|-------|----------|
| `AADSTS650057` | Application ID URI not configured | Set Application ID URI in App Registration |
| `AADSTS65001` | User not assigned to application | Add user in Enterprise Applications → Users and groups |
| `AADSTS700016` | Azure CLI not pre-authorized | Add Azure CLI (`04b07795-8ddb-461a-bbee-02f9e1bf7b46`) as authorized client |
| `insufficient scope` | Token missing required scopes | Ensure user has app role assigned, or scope is defined |

### Example with Command Line Flags

```bash
# Default scope (any Azure AD user)
./aks-mcp --transport=sse --oauth-enabled=true \
  --oauth-tenant-id="12345678-1234-1234-1234-123456789012" \
  --oauth-client-id="87654321-4321-4321-4321-210987654321"

# Restricted scope (only assigned users/SPNs)
./aks-mcp --transport=streamable-http --oauth-enabled=true \
  --oauth-tenant-id="12345678-1234-1234-1234-123456789012" \
  --oauth-client-id="87654321-4321-4321-4321-210987654321" \
  --oauth-scopes="api://87654321-4321-4321-4321-210987654321/.default"
```

## OAuth Endpoints

When OAuth is enabled, the following endpoints are available:

### Metadata Endpoints (Unauthenticated)

- `GET /.well-known/oauth-protected-resource` - OAuth 2.0 Protected Resource Metadata (RFC 9728)
- `GET /.well-known/oauth-authorization-server` - OAuth 2.0 Authorization Server Metadata (RFC 8414)
- `GET /.well-known/openid-configuration` - OpenID Connect Discovery (alias for authorization server metadata)
- `GET /health` - Health check endpoint

### OAuth Flow Endpoints (Unauthenticated)

- `GET /oauth2/v2.0/authorize` - Authorization endpoint proxy to Azure AD
- `POST /oauth2/v2.0/token` - Token exchange endpoint proxy to Azure AD
- `GET /oauth/callback` - Authorization Code flow callback handler
- `POST /oauth/register` - Dynamic Client Registration (RFC 7591)

### Token Management (Unauthenticated for simplicity)

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

#### Authentication and Token Issues
1. **Invalid Token**: Ensure the token is valid and not expired
2. **Wrong Audience**: Verify the token audience matches `https://management.azure.com`
3. **Missing Scopes**: Ensure the token includes `https://management.azure.com/.default` scope
4. **JWT Signature Validation Failed**: 
   - Check that Azure AD application platform is set correctly
   - Verify tenant ID matches the issuer in the token
   - Ensure token is using v2.0 format (from Azure Management API scope)

#### Azure AD Application Configuration Issues
5. **Client ID Not Found**: Verify the Application (client) ID is correct
6. **Redirect URI Mismatch**: Ensure redirect URIs match exactly in Azure AD app registration
7. **Wrong Platform Type**: Use "Mobile and desktop applications", NOT "Web" or "Single-page application"
8. **Insufficient Permissions**: Verify both OAuth and Azure resource permissions are configured
9. **SPA Platform Incompatibility (AADSTS9002327)**: 
   - Error: "Tokens issued for the 'Single-Page Application' client-type may only be redeemed via cross-origin requests"
   - Solution: Change Azure AD app platform to "Mobile and desktop applications"
   - Cause: SPA platform requires frontend token exchange, incompatible with AKS-MCP's backend implementation

#### Network and Endpoint Issues  
10. **CORS Errors**: Check that redirect URIs are properly configured for localhost
11. **Network Issues**: Check connectivity to Azure AD endpoints
11. **Port Conflicts**: Ensure the configured port (default 8000) is available

#### Scope and Permission Issues
12. **Scope Mixing Error**: 
    - Error: "scope can't be combined with resource-specific scopes"
    - Solution: Our implementation automatically handles this by using only Azure Management API scope
13. **Resource Parameter Issues**: 
    - Azure AD doesn't support RFC 8707 resource parameter
    - Our implementation works around this limitation automatically

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

Expected response with OAuth enabled:
```json
{
  "status": "healthy",
  "oauth": {
    "enabled": true
  }
}
```

### Testing OAuth Flow Step by Step

1. **Test Metadata Discovery**:
```bash
# Should return authorization server URLs
curl http://localhost:8000/.well-known/oauth-protected-resource

# Should return PKCE support and endpoints
curl http://localhost:8000/.well-known/oauth-authorization-server
```

2. **Test Client Registration**:
```bash
curl -X POST http://localhost:8000/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "redirect_uris": ["http://localhost:8000/oauth/callback"],
    "client_name": "Test Client"
  }'
```

3. **Test Authorization Flow**:
   - Open browser to: `http://localhost:8000/oauth2/v2.0/authorize?response_type=code&client_id=YOUR_CLIENT_ID&redirect_uri=http://localhost:8000/oauth/callback&scope=https://management.azure.com/.default&code_challenge=CHALLENGE&code_challenge_method=S256&state=STATE`

4. **Verify Token Validation**:
```bash
# Use a valid Azure AD token
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8000/mcp
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

### Important: Redirect URI Configuration for MCP Inspector

When using MCP Inspector with OAuth authentication, you need to add the Inspector's proxy redirect URI to your OAuth configuration:

```bash
# Add Inspector's redirect URI (typically http://localhost:6274/oauth/callback)
./aks-mcp \
  --transport=streamable-http \
  --port=8000 \
  --oauth-enabled \
  --oauth-redirects="http://localhost:8000/oauth/callback,http://localhost:6274/oauth/callback" \
  --access-level=readonly
```

**Key Points:**
- MCP Inspector typically runs on port 6274 by default
- The Inspector creates a proxy redirect URI at `/oauth/callback`
- You must include both your server's redirect URI AND the Inspector's redirect URI
- You must also configure CORS origins to allow the Inspector's web interface to make requests
- Comma-separate multiple redirect URIs in the `--oauth-redirects` parameter
- Comma-separate multiple CORS origins in the `--oauth-cors-origins` parameter
- Without the Inspector's redirect URI, OAuth authentication will fail with "redirect_uri not registered" error
- Without the Inspector's CORS origin, the web interface will be blocked by browser CORS policy

**Example with MCP Inspector configuration:**
```bash
./aks-mcp \
  --transport=streamable-http \
  --port=8000 \
  --oauth-enabled \
  --oauth-redirects="http://localhost:8000/oauth/callback,http://localhost:6274/oauth/callback" \
  --oauth-cors-origins="http://localhost:6274" \
  --access-level=readonly
```

For more information, see the MCP OAuth specification and Azure AD documentation.