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
     - 平台：Web
     - URL：`http://localhost:8080/oauth/callback`

3. **记录重要信息**
   创建完成后，在 "概述" 页面记录：
   - **应用程序(客户端) ID** - 这是你的 `CLIENT_ID` 9e1516b4-1a60-4836-8049-571594cdd74d
   - **目录(租户) ID** - 这是你的 `TENANT_ID` 84f68ef7-1b9b-45ae-9817-8706a841c544

### 第二步：配置 API 权限

1. **添加 API 权限**
   ```
   左侧菜单：API 权限 → 添加权限
   ```

2. **选择 Azure Service Management**
   ```
   Microsoft API → Azure Service Management → 委托权限
   ```

3. **添加权限**
   - 勾选 `user_impersonation`
   - 点击 "添加权限"

4. **授予管理员同意**
   ```
   点击 "为 [组织] 授予管理员同意" 按钮
   ```

### 第三步：配置身份验证设置

1. **访问身份验证页面**
   ```
   左侧菜单：身份验证
   ```

2. **配置重定向 URI**
   确认重定向 URI 包含：
   ```
   http://localhost:8080/oauth/callback
   ```
   
   如果需要测试不同端口，可以添加多个：
   ```
   http://localhost:8080/oauth/callback
   http://localhost:3000/oauth/callback
   http://localhost:8000/oauth/callback
   ```

3. **配置高级设置**
   - **允许公共客户端流**：是
   - **Live SDK 支持**：否

### 第四步：配置令牌配置（可选）

1. **访问令牌配置页面**
   ```
   左侧菜单：令牌配置
   ```

2. **添加可选声明**
   - 点击 "添加可选声明"
   - 选择 "访问令牌"
   - 添加：`email`, `family_name`, `given_name`

## AKS-MCP 环境搭建

<!-- ### 第一步：准备配置文件

1. **创建 OAuth 配置文件**
   ```bash
   # 创建配置目录
   mkdir -p ~/aks-mcp-test
   cd ~/aks-mcp-test
   
   # 创建配置文件
   cat > oauth-config.yaml << 'EOF'
   oauth:
     enabled: true
     tenant_id: "84f68ef7-1b9b-45ae-9817-8706a841c544"
     client_id: "9e1516b4-1a60-4836-8049-571594cdd74d"
     required_scopes:
       - "https://management.azure.com/.default"
     allowed_redirects:
       - "http://localhost:8080/oauth/callback"
       - "http://localhost:3000/oauth/callback"
     token_validation:
       validate_jwt: true
       validate_audience: true
       expected_audience: "https://management.azure.com/"
       cache_ttl: "5m"
       clock_skew: "1m"
   
   # 其他 AKS-MCP 配置
   access_level: "readonly"
   timeout: "30s"
   EOF
   ```

2. **替换配置中的占位符**
   ```bash
   # 使用你从 Azure AD 获取的实际值替换
   sed -i 's/YOUR_TENANT_ID_HERE/你的租户ID/' oauth-config.yaml
   sed -i 's/YOUR_CLIENT_ID_HERE/你的客户端ID/' oauth-config.yaml
   ``` -->

3. **设置环境变量**
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

2. **启动服务器（HTTP 模式）**
   <!-- ```bash
   # 使用配置文件启动
   ./aks-mcp --config ~/aks-mcp-test/oauth-config.yaml --transport http --port 8080 -->
   
   # 或者使用命令行参数
   ./aks-mcp \
     --transport http \
     --port 8080 \
     --oauth-enabled \
     --oauth-tenant-id "$AZURE_TENANT_ID" \
     --oauth-client-id "$AZURE_CLIENT_ID" \
     --access-level readonly
   ```

3. **验证服务器启动**
   ```bash
   # 检查健康状态
   curl http://localhost:8080/health
   
   # 应该返回类似：
   # {"status":"healthy","oauth":{"enabled":true}}
   ```

## 使用浏览器测试 OAuth 流程

### 第一步：获取授权 URL

1. **构建授权 URL**
   ```bash
   # 设置参数
   TENANT_ID="你的租户ID"
   CLIENT_ID="你的客户端ID"
   REDIRECT_URI="http://localhost:8080/oauth/callback"
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
   - 浏览器会被重定向到 `http://localhost:8080/oauth/callback`
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
echo "https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/authorize?client_id=${CLIENT_ID}&response_type=code&redirect_uri=http://localhost:8080/oauth/callback&scope=${SCOPE}&state=test-$(date +%s)"
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
   curl -X GET http://localhost:8080/health
   ```

2. **测试受保护的端点（需要认证）**
   ```bash
   # 设置访问令牌
   export ACCESS_TOKEN="从浏览器获取的访问令牌"
   
   # 测试 MCP 端点
   curl -X POST http://localhost:8080/mcp \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 1,
       "method": "tools/list"
     }'
   ```

3. **测试 OAuth 元数据端点**
   ```bash
   # 获取受保护资源元数据
   curl -X GET http://localhost:8080/.well-known/oauth-protected-resource
   
   # 获取授权服务器元数据
   curl -X GET http://localhost:8080/.well-known/oauth-authorization-server
   ```

4. **测试令牌内省端点**
   ```bash
   curl -X POST http://localhost:8080/oauth/introspect \
     -H "Content-Type: application/x-www-form-urlencoded" \
     -d "token=$ACCESS_TOKEN"
   ```

### 第三步：测试无效令牌的情况

```bash
# 测试无效令牌
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer invalid-token" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}'

# 应该返回 401 Unauthorized
```

## 使用 MCP Inspector 测试

### 第一步：配置 MCP Inspector

1. **创建 Inspector 配置文件**
   ```bash
   mkdir -p ~/.config/mcp-inspector
   cat > ~/.config/mcp-inspector/aks-mcp-oauth.json << 'EOF'
   {
     "name": "AKS-MCP with OAuth",
     "type": "http",
     "config": {
       "url": "http://localhost:8080/mcp",
       "headers": {
         "Authorization": "Bearer YOUR_ACCESS_TOKEN_HERE"
       }
     }
   }
   EOF
   ```

2. **替换访问令牌**
   ```bash
   # 使用实际的访问令牌替换占位符
   sed -i 's/YOUR_ACCESS_TOKEN_HERE/你的实际访问令牌/' ~/.config/mcp-inspector/aks-mcp-oauth.json
   ```

### 第二步：启动 MCP Inspector

1. **编译和启动 Inspector**
   ```bash
   # 切换到 inspector 目录
   cd ./inspector
   
   # 编译（如果需要）
   go build -o mcp-inspector .
   
   # 启动 Inspector
   ./mcp-inspector
   ```

2. **在 Inspector 中测试**
   - 添加新的服务器连接
   - 使用配置文件中的设置
   - 测试连接和工具列表

### 第三步：验证 OAuth 功能

1. **测试有效令牌**
   - 使用正确的访问令牌
   - 验证能够成功连接和获取工具列表

2. **测试无效令牌**
   ```bash
   # 修改配置使用无效令牌
   sed -i 's/Bearer .*/Bearer invalid-token"/' ~/.config/mcp-inspector/aks-mcp-oauth.json
   ```
   - 重启 Inspector
   - 验证连接失败并显示认证错误

3. **测试令牌过期**
   - 等待令牌过期（通常 1 小时）
   - 验证需要重新获取令牌

## 故障排除

### 常见问题和解决方案

#### 1. 回调 URL 不匹配

**错误信息**：`AADSTS50011: The reply URL specified in the request does not match`

**解决方案**：
```bash
# 检查 Azure AD 应用注册中的重定向 URI
# 确保包含：http://localhost:8080/oauth/callback

# 或者修改服务器端口匹配 Azure AD 配置
./aks-mcp --port 3000  # 如果 Azure AD 配置的是 3000 端口
```

#### 2. 权限不足

**错误信息**：`AADSTS65001: The user or administrator has not consented`

**解决方案**：
1. 在 Azure Portal 中为应用授予管理员同意
2. 或者在授权 URL 中添加 `&prompt=consent` 参数

#### 3. 令牌验证失败

**错误信息**：`Token validation failed`

**解决方案**：
```bash
# 检查系统时间是否正确
date

# 检查租户 ID 是否正确
curl "https://login.microsoftonline.com/你的租户ID/v2.0/.well-known/openid_configuration"

# 检查客户端 ID 是否正确
```

#### 4. 端口被占用

**错误信息**：`bind: address already in use`

**解决方案**：
```bash
# 查找占用端口的进程
lsof -i :8080

# 使用不同端口
./aks-mcp --port 8081

# 相应更新 Azure AD 重定向 URI
```

#### 5. CORS 错误（浏览器中）

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
   telnet localhost 8080
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