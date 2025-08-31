# AKS-MCP OAuth 认证手动测试指南

本文档提供完整的 OAuth 认证手动测试步骤，包括 Azure Entra ID 配置、环境搭建和测试验证。

## 目录

1. [Azure Entra ID 配置](#azure-entra-id-配置)
2. [AKS-MCP 环境搭建](#aks-mcp-环境搭建)
3. [使用浏览器测试 OAuth 流程](#使用浏览器测试-oauth-流程)
4. [使用 curl 测试 API](#使用-curl-测试-api)
5. [使用 MCP Inspector 测试](#使用-mcp-inspector-测试)
6. [故障排除](#故障排除)

## Azure Entra ID 配置

### 第一步：创建 Azure AD 应用注册

1. **登录 Azure Portal**
   - 打开 https://portal.azure.com
   - 使用你的 Azure 账户登录

2. **创建应用注册**
   ```
   导航路径：Azure Active Directory → 应用注册 → 新建注册
   ```
   
   填写以下信息：
   - **名称**：`AKS-MCP-OAuth-Test`
   - **受支持的账户类型**：选择 "仅此组织目录中的账户"
   - **重定向 URI**：
     - 平台：**单页应用程序 (SPA)**（重要：必须选择 SPA，不能选择 Web）
     - URL：添加以下两个回调地址：
       - `http://localhost:8000/oauth/callback`
       - `http://localhost:6274/oauth/callback/debug`（用于 MCP Inspector 测试）

3. **记录重要信息**
   创建完成后，在 "概述" 页面记录：
   - **应用程序(客户端) ID** - 这是你的 `CLIENT_ID`
   - **目录(租户) ID** - 这是你的 `TENANT_ID`

### 第二步：配置 API 权限（必须完成）

**重要提醒：OAuth 和 Azure CLI 都需要正确的权限配置**

由于 AKS-MCP 在设置了 `AZURE_CLIENT_ID` 环境变量时，会将同一个 Azure AD 应用用于两个不同的认证场景：
1. **OAuth 认证**：验证访问 MCP 服务器的用户令牌
2. **Azure CLI 认证**：AKS-MCP 服务器本身访问 Azure 资源的身份验证

因此需要配置两套权限：

1. **添加 OAuth 所需的 API 权限**
   ```
   左侧菜单：API 权限 → 添加权限
   ```

2. **选择 Azure Service Management（OAuth 必需）**
   ```
   Microsoft API → Azure Service Management → 委托权限
   ```

3. **添加 OAuth 权限**
   - 勾选 `user_impersonation`
   - 点击 "添加权限"

4. **添加 Azure 资源管理权限（Azure CLI 必需）**
   
   当设置了 `AZURE_CLIENT_ID` 时，Azure CLI 会使用这个应用进行身份验证。根据你的 AKS-MCP 访问级别添加以下权限：
   
   **对于 readonly 访问级别：**
   ```
   Microsoft Graph → 应用程序权限 → Directory.Read.All
   Azure Service Management → 委托权限 → user_impersonation
   ```
   
   **对于 readwrite/admin 访问级别：**
   ```
   Microsoft Graph → 应用程序权限 → Directory.Read.All
   Azure Service Management → 委托权限 → user_impersonation
   根据需要考虑添加特定的 Azure 资源权限
   ```

5. **授予管理员同意**（必须步骤）
   ```
   点击 "为 [组织] 授予管理员同意" 按钮
   ```
   
**⚠️ 重要说明**：
- 如果不授予管理员同意，OAuth 流程将在 scope 验证时失败
- 如果缺少 Azure CLI 权限，当 AKS-MCP 尝试访问 Azure 资源时会出现 "权限不足" 错误
- 同一个应用程序既处理 OAuth 认证（用户访问 MCP），也处理 Azure CLI 认证（MCP 访问 Azure）
- 权限更改后，需要测试 OAuth 流程和 Azure 资源访问两个方面

### 第三步：验证应用配置

1. **确认平台配置**
   ```
   左侧菜单：身份验证
   ```
   
   验证以下设置：
   - **平台配置**：应该显示 "单页应用程序"
   - **重定向 URI**：应该包含：
     ```
     http://localhost:8000/oauth/callback
     http://localhost:6274/oauth/callback/debug
     ```
   - **高级设置 → 允许公共客户端流**：应该设置为 "是"

2. **确认 API 权限**
   ```
   左侧菜单：API 权限
   ```
   
   应该显示：
   - **Azure Service Management** - `user_impersonation`（已授予管理员同意）
   - **Microsoft Graph** - `Directory.Read.All`（如果添加了 Azure CLI 权限）
   - 所有权限的状态应该是绿色的勾选标记

**重要说明 - MCP Inspector 回调地址**：
- MCP Inspector 使用特殊的调试回调地址 `http://localhost:6274/oauth/callback/debug`
- 这个地址必须在 Azure AD 应用的重定向 URI 中配置
- 6274 端口是 MCP Inspector 的默认调试端口

## AKS-MCP 环境搭建

### 第一步：设置环境变量

**⚠️ 重要：环境变量的双重影响**

使用你从 Azure AD 获取的实际值设置环境变量。需要注意的是，当你设置 `AZURE_CLIENT_ID` 时，它会影响两个认证流程：

1. **OAuth 认证**：用于验证访问 MCP 服务器的用户令牌
2. **Azure CLI 认证**：AKS-MCP 服务器使用这个客户端 ID 通过托管身份访问 Azure 资源

**常见问题**：
- 如果只配置了 OAuth 权限，Azure CLI 操作会失败并显示"权限不足"
- 如果只配置了 Azure 资源权限，OAuth 令牌验证可能失败
- **解决方案**：确保你的 Azure AD 应用具有两套权限（参见前面的第二步）

```bash
# 创建环境变量文件
cat > .env << 'EOF'
export AZURE_TENANT_ID="你的租户ID"
export AZURE_CLIENT_ID="你的客户端ID"
export AZURE_SUBSCRIPTION_ID="你的订阅ID"
EOF

# 加载环境变量
source .env
```

**示例（请替换为你的实际值）**：
```bash
export AZURE_TENANT_ID="84f68ef7-1b9b-45ae-9817-8706a841c544"
export AZURE_CLIENT_ID="9e1516b4-1a60-4836-8049-571594cdd74d"
export AZURE_SUBSCRIPTION_ID="your-subscription-id"
```

### 第二步：编译和启动 AKS-MCP 服务器

1. **编译项目**
   ```bash
   # 切换到项目目录
   cd /path/to/aks-mcp
   
   # 编译
   make build
   # 或者
   go build -o aks-mcp ./cmd/aks-mcp
   ```

2. **启动服务器（HTTP Streamable 模式 - 推荐）**
   ```bash
   ./aks-mcp \
     --transport streamable-http \
     --port 8000 \
     --oauth-enabled \
     --oauth-tenant-id "$AZURE_TENANT_ID" \
     --oauth-client-id "$AZURE_CLIENT_ID" \
     --oauth-redirects="http://localhost:8000/oauth/callback,http://localhost:6274/oauth/callback/debug" \
     --access-level readonly
   ```

   **重要配置说明**：
   - `--oauth-redirects` 必须包含两个地址：
     - `http://localhost:8000/oauth/callback` - AKS-MCP 主回调地址
     - `http://localhost:6274/oauth/callback/debug` - MCP Inspector 调试回调地址
   - 这两个地址必须与 Azure AD 应用中配置的重定向 URI 完全匹配

3. **验证服务器启动和双重认证**
   ```bash
   # 检查健康状态
   curl http://localhost:8000/health
   
   # 应该返回类似：
   # {"status":"healthy","oauth":{"enabled":true}}
   ```
   
   **同时测试两种认证路径**：
   ```bash
   # 1. 测试 OAuth 端点（应该正常工作）
   curl http://localhost:8000/.well-known/oauth-protected-resource
   
   # 2. 测试 Azure CLI 认证（检查服务器日志）
   # 当 AKS-MCP 尝试访问 Azure 资源时，会在后台使用 Azure CLI 认证
   # 如果权限配置正确，不会出现认证错误
   # 如果权限不足，会在日志中看到 "权限不足" 或 "认证失败" 错误
   ```

### 第三步：验证 OAuth 端点

```bash
# 测试受保护资源元数据
curl http://localhost:8000/.well-known/oauth-protected-resource

# 测试授权服务器元数据
curl http://localhost:8000/.well-known/oauth-authorization-server

# 应该都返回正常的 JSON 响应
```

## 使用浏览器测试 OAuth 流程

### 第一步：获取授权 URL

1. **构建授权 URL**
   ```bash
   # 设置参数
   TENANT_ID="你的租户ID"
   CLIENT_ID="你的客户端ID"
   REDIRECT_URI="http://localhost:8000/oauth/callback"
   SCOPE="https://management.azure.com/.default"
   STATE="test-state-$(date +%s)"
   
   # 构建完整的授权 URL
   AUTH_URL="https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/authorize?client_id=${CLIENT_ID}&response_type=code&redirect_uri=${REDIRECT_URI}&scope=${SCOPE}&state=${STATE}&response_mode=query"
   
   echo "授权 URL："
   echo "$AUTH_URL"
   ```

2. **在浏览器中打开授权 URL**
   - 复制上面生成的 URL
   - 在浏览器中打开
   - 使用 Azure 账户登录

### 第二步：完成授权流程

1. **登录并授权**
   - 输入你的 Azure 账户凭据
   - 如果提示权限同意，点击 "接受"

2. **查看回调结果**
   - 浏览器会被重定向到 `http://localhost:8000/oauth/callback`
   - 如果成功，你会看到一个绿色的成功页面
   - 页面会显示访问令牌和相关信息

3. **复制访问令牌**
   - 点击 "Copy Token" 按钮复制访问令牌
   - 或者手动复制显示的令牌字符串

## 使用 curl 测试 API

### 第一步：获取访问令牌

使用上面浏览器流程获取的访问令牌，或者使用以下脚本自动化获取：

```bash
# 创建获取令牌的脚本
cat > get_token.sh << 'EOF'
#!/bin/bash

# 配置参数
TENANT_ID="你的租户ID"
CLIENT_ID="你的客户端ID"
SCOPE="https://management.azure.com/.default"

echo "请访问以下 URL 完成授权："
echo "https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/authorize?client_id=${CLIENT_ID}&response_type=code&redirect_uri=http://localhost:8000/oauth/callback&scope=${SCOPE}&state=test-$(date +%s)"
echo ""
echo "授权完成后，从回调页面复制访问令牌，然后运行："
echo "export ACCESS_TOKEN=\"你的访问令牌\""
EOF

chmod +x get_token.sh
./get_token.sh
```

### 第二步：测试 API 端点

1. **测试健康检查（无需认证）**
   ```bash
   curl -X GET http://localhost:8000/health
   ```

2. **测试受保护的端点（需要认证）**
   ```bash
   # 设置访问令牌
   export ACCESS_TOKEN="从浏览器获取的访问令牌"

   # initialize, example:
   curl -s -i \
   -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: Bearer $ACCESS_TOKEN" \
   -d '{"jsonrpc": "2.0", "method": "initialize", "id": 1}' \
   "http://localhost:8000/mcp"
   HTTP/1.1 200 OK
   Content-Type: application/json
   Mcp-Session-Id: mcp-session-86d931cb-127e-4d8f-a409-ef37fbde528f
   Date: Wed, 27 Aug 2025 03:50:24 GMT
   Content-Length: 255

   SESSION_ID=mcp-session-86d931cb-127e-4d8f-a409-ef37fbde528f
   
   # 测试 MCP 端点
   curl -X POST http://localhost:8000/mcp \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "Content-Type: application/json" \
     -H "Mcp-Session-Id: $SESSION_ID" \
     -d '{
       "jsonrpc": "2.0",
       "id": 1,
       "method": "tools/list"
     }'
   ```

3. **测试 OAuth 元数据端点**
   ```bash
   # 获取受保护资源元数据
   curl -X GET http://localhost:8000/.well-known/oauth-protected-resource
   
   # 获取授权服务器元数据
   curl -X GET http://localhost:8000/.well-known/oauth-authorization-server
   ```

4. **测试令牌内省端点**
   ```bash
   curl -X POST http://localhost:8000/oauth/introspect \
     -H "Content-Type: application/x-www-form-urlencoded" \
     -d "token=$ACCESS_TOKEN"
   ```

### 第三步：测试无效令牌的情况

```bash
# 测试无效令牌
curl -X POST http://localhost:8000/mcp \
  -H "Authorization: Bearer invalid-token" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}'

# 应该返回 401 Unauthorized
```

## 使用 MCP Inspector 测试

### 第一步：启动 MCP Inspector

1. **安装和启动 MCP Inspector**
   ```bash
   # 安装 MCP Inspector（如果尚未安装）
   npm install -g @modelcontextprotocol/inspector
   
   # 启动 Inspector
   npx @modelcontextprotocol/inspector
   ```

   **重要说明**：
   - MCP Inspector 会在端口 6274 运行调试服务器
   - 它使用特殊的回调地址 `http://localhost:6274/oauth/callback/debug`
   - 确保此地址已在 Azure AD 应用的重定向 URI 中配置

2. **配置 MCP Inspector 连接**
   在 Inspector 界面中：
   - **Server URL**: `http://localhost:8000/mcp`
   - **Authentication**: 选择 "OAuth 2.0"
   - **Authorization Server**: `http://localhost:8000/.well-known/oauth-authorization-server`

### 第二步：完成 OAuth 流程

MCP Inspector 会自动执行以下步骤：

1. **Metadata Discovery** - 获取 OAuth 配置信息
2. **Client Registration** - 动态注册客户端
3. **Preparing Authorization** - 准备授权请求
4. **Request Authorization** - 获取授权码
5. **Token Request** - 交换访问令牌
6. **Authentication Complete** - 显示访问令牌
7. **MCP Connection** - 使用令牌连接 MCP 服务器

### 第三步：验证 OAuth 功能

1. **测试有效令牌**
   - 完成 OAuth 流程后，Inspector 应显示成功的连接状态
   - 可以看到 AKS-MCP 提供的工具列表

2. **测试会话管理**
   - 刷新页面后，Inspector 应能够重用缓存的令牌
   - 如果遇到缓存问题，清除浏览器数据：`Developer Tools > Application > Storage > Clear storage`

### 第四步：故障排除

**常见问题**：

1. **Client ID 显示为 '111'**
   - 清除浏览器的 sessionStorage 缓存
   - 开发者工具 > Application > Storage > Session Storage > 删除所有条目

2. **回调地址不匹配**
   - 确保 Azure AD 应用包含 `http://localhost:6274/oauth/callback/debug`
   - 检查端口 6274 是否被其他进程占用

3. **CORS 错误**
   - 确保 AKS-MCP 服务器正在运行
   - 检查防火墙设置

4. **令牌验证失败**
   - 确保 Azure AD 应用已授予管理员同意
   - 检查 API 权限配置是否正确

## 故障排除

### 常见问题和解决方案

#### 1. 回调 URL 不匹配

**错误信息**：`AADSTS50011: The reply URL specified in the request does not match`

**解决方案**：
```bash
# 检查 Azure AD 应用注册中的重定向 URI
# 确保包含以下两个地址：
# - http://localhost:8000/oauth/callback
# - http://localhost:6274/oauth/callback/debug

# 或者修改服务器端口匹配 Azure AD 配置
./aks-mcp --port 3000  # 如果 Azure AD 配置的是 3000 端口
```

#### 2. 权限不足

**错误信息**：`AADSTS65001: The user or administrator has not consented`

**解决方案**：
1. 在 Azure Portal 中为应用授予管理员同意
2. 或者在授权 URL 中添加 `&prompt=consent` 参数

#### 3. Scope 验证失败

**错误信息**：`insufficient_scope` 或 HTTP 403 错误

**根本原因**：Azure AD 应用缺少必要的 API 权限配置

**解决方案**：
1. **检查 Azure AD API 权限配置**：
   ```
   Azure Portal → Azure Active Directory → 应用注册 → [你的应用] → API 权限
   ```
   
   必须包含：
   - **Azure Service Management** - `user_impersonation`（委托权限）
   - 状态必须显示为"已为 [组织] 授予管理员同意"

2. **如果权限缺失，添加权限**：
   - 点击"添加权限"
   - 选择"Microsoft APIs" → "Azure Service Management"
   - 选择"委托权限" → `user_impersonation`
   - 点击"添加权限"
   - **重要**：点击"为 [组织] 授予管理员同意"

3. **验证权限生效**：
   ```bash
   # 重新获取访问令牌（权限更改后需要新令牌）
   # 重新执行 OAuth 流程
   ```

#### 4. 令牌被截断

**错误信息**：Token appears to be truncated (no dots found)

**解决方案**：
```bash
# 这是已知的 MCP Inspector 限制，目前有临时解决方案
# 检查系统时间是否正确
date

# 检查令牌内容（前50个字符）
# 完整的 JWT 应该有 3 部分，用 . 分隔
```

#### 5. Azure AD 应用类型配置错误

**错误信息**：`AADSTS9002326: Cross-origin token redemption is permitted only for the 'Single-Page Application' client-type`

**解决方案**：
1. 在 Azure Portal 中修改应用配置：
   ```
   Azure Active Directory → 应用注册 → [你的应用] → 身份验证
   ```
2. 确保平台配置显示为"单页应用程序"
3. 如果显示为"Web"，删除"Web"平台，添加"单页应用程序"平台

#### 6. 令牌验证失败 - issuer不匹配

**错误信息**：`Token validation failed: invalid issuer: expected https://login.microsoftonline.com/租户ID/v2.0, got https://sts.windows.net/租户ID/`

**解决方案**：
这是由于Azure AD v1.0和v2.0端点issuer格式不同造成的。AKS-MCP已支持两种格式，确保使用最新代码：
```bash
# 重新编译并启动
make build
./aks-mcp --oauth-enabled ...
```

#### 7. 令牌验证失败 - audience不匹配

**错误信息**：`Token validation failed: invalid audience: expected https://management.azure.com/ or 客户端ID, got https://management.azure.com`

**解决方案**：
这是由于audience字段的尾随斜杠不匹配造成的。AKS-MCP已修复audience验证逻辑，确保使用最新代码：
```bash
# 重新编译并启动
make build
./aks-mcp --oauth-enabled ...
```

#### 8. MCP Inspector 客户端 ID 缓存问题

**错误信息**：Client ID 显示为 '111' 而不是配置的值

**解决方案**：
```bash
# 清除浏览器缓存
# 开发者工具 > Application > Storage > Session Storage > 删除所有条目
# 或者完全清除浏览器数据
```

#### 9. 端口被占用

**错误信息**：`bind: address already in use`

**解决方案**：
```bash
# 查找占用端口的进程
lsof -i :8000
lsof -i :6274

# 使用不同端口
./aks-mcp --port 8081

# 相应更新 Azure AD 重定向 URI
```

#### 11. Azure CLI 权限不足错误

**错误信息**：
- "Insufficient privileges to complete the operation"
- "Access denied"
- "Authentication failed"
- Azure CLI 操作失败，但 OAuth 流程正常

**根本原因**：
当设置了 `AZURE_CLIENT_ID` 环境变量时，Azure CLI 会使用这个客户端 ID 进行托管身份认证来访问 Azure 资源。如果 Azure AD 应用缺少相应的 Azure 资源权限，就会出现此错误。

**解决方案**：

1. **检查当前权限配置**：
   ```bash
   # 在 Azure Portal 中检查应用权限
   # Azure Active Directory → 应用注册 → [你的应用] → API 权限
   ```

2. **添加必要的 Azure 资源权限**：
   
   **基本权限（最小要求）**：
   ```
   Microsoft Graph → 应用程序权限 → Directory.Read.All
   Azure Service Management → 委托权限 → user_impersonation
   ```
   
   **扩展权限（根据 AKS-MCP 功能需求）**：
   ```
   # 如果需要访问 Azure Resource Manager
   Azure Service Management → 应用程序权限 → user_impersonation
   
   # 如果需要访问 Key Vault
   Azure Key Vault → 委托权限 → user_impersonation
   
   # 如果需要访问 Storage Account
   Azure Storage → 委托权限 → user_impersonation
   ```

3. **授予管理员同意**：
   ```bash
   # 添加权限后，必须授予管理员同意
   # Azure Portal → API 权限 → "为 [组织] 授予管理员同意"
   ```

4. **验证权限生效**：
   ```bash
   # 重启 AKS-MCP 服务器
   ./aks-mcp --oauth-enabled --access-level=readonly
   
   # 检查服务器日志中是否还有权限错误
   # 尝试调用需要 Azure 资源访问的 MCP 工具
   ```

**调试技巧**：
```bash
# 查看详细的 Azure CLI 认证信息
export AZURE_CLI_ENABLE_DEBUG=true
./aks-mcp --oauth-enabled --verbose

# 手动测试 Azure CLI 认证
az login --identity --username $AZURE_CLIENT_ID
az account show
```

**⚠️ 重要说明**：
- 这个问题仅在设置了 `AZURE_CLIENT_ID` 环境变量时才会发生
- OAuth 用户认证和 Azure CLI 资源访问是两个独立的认证流程
- 同一个 Azure AD 应用需要支持两种用途，因此需要两套权限
- 如果只想测试 OAuth 功能，可以临时取消设置 `AZURE_CLIENT_ID`：
  ```bash
  unset AZURE_CLIENT_ID
  ./aks-mcp --oauth-enabled
  ```

#### 10. CORS 错误（浏览器中）

**解决方案**：
- 确保使用正确的重定向 URI
- 避免在浏览器中直接调用 API，使用 curl 或 Postman

### 调试技巧

1. **启用详细日志**
   ```bash
   ./aks-mcp --log-level debug
   ```

2. **检查令牌内容**
   ```bash
   # 解码 JWT 令牌（不验证签名）
   echo "你的JWT令牌" | cut -d. -f2 | base64 -d | jq .
   ```

3. **验证 Azure AD 配置**
   ```bash
   # 获取 OpenID 配置
   curl "https://login.microsoftonline.com/你的租户ID/v2.0/.well-known/openid_configuration" | jq .
   
   # 获取公钥
   curl "https://login.microsoftonline.com/你的租户ID/discovery/v2.0/keys" | jq .
   ```

4. **网络连接测试**
   ```bash
   # 测试到 Azure AD 的连接
   telnet login.microsoftonline.com 443
   
   # 测试本地服务器
   telnet localhost 8000
   ```

### 获取帮助

如果遇到其他问题：

1. **检查服务器日志**：查看 AKS-MCP 服务器的控制台输出
2. **查看 Azure AD 日志**：在 Azure Portal 的 "登录日志" 中查看认证详情
3. **使用 Azure CLI 调试**：
   ```bash
   az login
   az account show
   az ad app list --display-name "AKS-MCP-OAuth-Test"
   ```

## 安全注意事项

1. **访问令牌保护**：
   - 不要在日志中记录访问令牌
   - 不要在 URL 中传递访问令牌
   - 使用 HTTPS（生产环境）

2. **配置文件安全**：
   ```bash
   # 设置配置文件权限
   chmod 600 oauth-config.yaml
   chmod 600 .env
   ```

3. **令牌过期处理**：
   - 访问令牌通常在 1 小时后过期
   - 生产环境应实现自动令牌刷新

4. **网络安全**：
   - 生产环境使用 HTTPS
   - 配置适当的防火墙规则
   - 使用私有网络（如果可能）