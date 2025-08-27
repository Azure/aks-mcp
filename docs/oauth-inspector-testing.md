# 使用 MCP Inspector 测试 AKS-MCP OAuth 认证

本文档专门介绍如何使用 MCP Inspector 来测试 AKS-MCP 的 OAuth 认证功能。

## 目录

1. [MCP Inspector 介绍](#mcp-inspector-介绍)
2. [环境准备](#环境准备)
3. [配置 MCP Inspector](#配置-mcp-inspector)
4. [OAuth 认证流程](#oauth-认证流程)
5. [功能测试](#功能测试)
6. [故障排除](#故障排除)

## MCP Inspector 介绍

MCP Inspector 是一个用于测试和调试 Model Context Protocol (MCP) 服务器的工具。它可以：
- 连接到 MCP 服务器
- 列出可用的工具和资源
- 执行工具调用
- 查看服务器响应
- 支持多种传输方式（stdio、HTTP、SSE）

## 环境准备

### 第一步：启动 AKS-MCP 服务器

1. **确保 OAuth 配置正确**
   ```bash
   # 检查配置文件
   cat ~/aks-mcp-test/oauth-config.yaml
   
   # 确保包含正确的租户 ID 和客户端 ID
   grep -E "(tenant_id|client_id)" ~/aks-mcp-test/oauth-config.yaml
   ```

2. **启动 AKS-MCP 服务器**
   ```bash
   cd /path/to/aks-mcp
   
   # 启动服务器（HTTP 模式，端口 8080）
   ./aks-mcp \
     --config ~/aks-mcp-test/oauth-config.yaml \
     --transport http \
     --port 8080
   ```

3. **验证服务器运行**
   ```bash
   curl http://localhost:8080/health
   # 应该返回：{"status":"healthy","oauth":{"enabled":true}}
   ```

### 第二步：获取访问令牌

1. **使用浏览器获取令牌**
   ```bash
   # 生成授权 URL
   TENANT_ID="你的租户ID"
   CLIENT_ID="你的客户端ID"
   
   echo "请在浏览器中访问以下 URL："
   echo "https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/authorize?client_id=${CLIENT_ID}&response_type=code&redirect_uri=http://localhost:8080/oauth/callback&scope=https://management.azure.com/.default&state=inspector-test"
   ```

2. **完成授权并获取令牌**
   - 在浏览器中访问上述 URL
   - 登录 Azure 账户并同意权限
   - 从回调页面复制访问令牌

## 配置 MCP Inspector

### 第一步：编译 MCP Inspector

```bash
# 切换到 inspector 目录
cd ./inspector

# 查看 Inspector 的结构
ls -la

# 编译 Inspector（如果有 Go 代码）
go build -o mcp-inspector .

# 或者如果有 npm 项目
npm install
npm run build
```

### 第二步：创建配置文件

1. **创建 Inspector 配置目录**
   ```bash
   mkdir -p ~/.config/mcp-inspector
   ```

2. **创建 AKS-MCP 连接配置**
   ```bash
   cat > ~/.config/mcp-inspector/aks-mcp.json << 'EOF'
   {
     "name": "AKS-MCP OAuth Server",
     "type": "http",
     "url": "http://localhost:8080/mcp",
     "headers": {
       "Authorization": "Bearer YOUR_ACCESS_TOKEN_HERE",
       "Content-Type": "application/json"
     },
     "oauth": {
       "enabled": true,
       "auth_url": "https://login.microsoftonline.com/YOUR_TENANT_ID/oauth2/v2.0/authorize",
       "token_url": "https://login.microsoftonline.com/YOUR_TENANT_ID/oauth2/v2.0/token",
       "client_id": "YOUR_CLIENT_ID",
       "scope": "https://management.azure.com/.default",
       "redirect_uri": "http://localhost:8080/oauth/callback"
     }
   }
   EOF
   ```

3. **更新配置文件中的实际值**
   ```bash
   # 替换租户 ID
   sed -i 's/YOUR_TENANT_ID/你的实际租户ID/g' ~/.config/mcp-inspector/aks-mcp.json
   
   # 替换客户端 ID
   sed -i 's/YOUR_CLIENT_ID/你的实际客户端ID/' ~/.config/mcp-inspector/aks-mcp.json
   
   # 替换访问令牌
   sed -i 's/YOUR_ACCESS_TOKEN_HERE/你的实际访问令牌/' ~/.config/mcp-inspector/aks-mcp.json
   ```

### 第三步：验证配置

```bash
# 检查配置文件
cat ~/.config/mcp-inspector/aks-mcp.json | jq .

# 测试基本连接
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer $(jq -r '.headers.Authorization' ~/.config/mcp-inspector/aks-mcp.json | cut -d' ' -f2)" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}}}'
```

## OAuth 认证流程

### 方式一：手动令牌配置

1. **获取新的访问令牌**
   ```bash
   # 创建获取令牌的脚本
   cat > get_inspector_token.sh << 'EOF'
   #!/bin/bash
   
   TENANT_ID=$(jq -r '.oauth.client_id' ~/.config/mcp-inspector/aks-mcp.json)
   CLIENT_ID=$(jq -r '.oauth.client_id' ~/.config/mcp-inspector/aks-mcp.json)
   
   echo "=== MCP Inspector OAuth 令牌获取 ==="
   echo "1. 在浏览器中访问以下 URL："
   echo "https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/authorize?client_id=${CLIENT_ID}&response_type=code&redirect_uri=http://localhost:8080/oauth/callback&scope=https://management.azure.com/.default&state=inspector-$(date +%s)"
   echo ""
   echo "2. 完成授权后，从回调页面复制访问令牌"
   echo "3. 运行以下命令更新配置："
   echo "   update_inspector_token.sh YOUR_NEW_TOKEN"
   EOF
   
   chmod +x get_inspector_token.sh
   ```

2. **更新令牌脚本**
   ```bash
   cat > update_inspector_token.sh << 'EOF'
   #!/bin/bash
   
   if [ $# -eq 0 ]; then
       echo "用法: $0 <access_token>"
       exit 1
   fi
   
   NEW_TOKEN="$1"
   CONFIG_FILE="$HOME/.config/mcp-inspector/aks-mcp.json"
   
   # 备份原配置
   cp "$CONFIG_FILE" "$CONFIG_FILE.backup"
   
   # 更新访问令牌
   jq --arg token "Bearer $NEW_TOKEN" '.headers.Authorization = $token' "$CONFIG_FILE" > "$CONFIG_FILE.tmp" && mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
   
   echo "令牌已更新。测试连接..."
   curl -X POST http://localhost:8080/mcp \
     -H "Authorization: Bearer $NEW_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' | jq .
   EOF
   
   chmod +x update_inspector_token.sh
   ```

### 方式二：自动化 OAuth 流程（高级）

如果 MCP Inspector 支持自动 OAuth 流程：

```bash
# 创建自动化配置
cat > ~/.config/mcp-inspector/aks-mcp-auto.json << 'EOF'
{
  "name": "AKS-MCP Auto OAuth",
  "type": "http",
  "url": "http://localhost:8080/mcp",
  "oauth": {
    "flow": "authorization_code",
    "auth_url": "https://login.microsoftonline.com/YOUR_TENANT_ID/oauth2/v2.0/authorize",
    "token_url": "https://login.microsoftonline.com/YOUR_TENANT_ID/oauth2/v2.0/token",
    "client_id": "YOUR_CLIENT_ID",
    "scope": "https://management.azure.com/.default",
    "redirect_uri": "http://localhost:8080/oauth/callback",
    "response_type": "code",
    "grant_type": "authorization_code"
  }
}
EOF
```

## 功能测试

### 第一步：启动 MCP Inspector

```bash
# 启动 Inspector
cd ./inspector
./mcp-inspector

# 或者如果是 Node.js 项目
npm start
```

### 第二步：连接到 AKS-MCP 服务器

1. **添加服务器连接**
   - 在 Inspector 界面中点击 "Add Server"
   - 选择 "HTTP" 传输类型
   - 输入服务器 URL：`http://localhost:8080/mcp`
   - 添加认证头：`Authorization: Bearer YOUR_TOKEN`

2. **测试连接**
   - 点击 "Connect" 或 "Test Connection"
   - 验证连接状态显示为 "Connected"

### 第三步：测试 MCP 协议功能

1. **初始化协议**
   ```json
   {
     "jsonrpc": "2.0",
     "id": 1,
     "method": "initialize",
     "params": {
       "protocolVersion": "2024-11-05",
       "capabilities": {
         "tools": {}
       },
       "clientInfo": {
         "name": "mcp-inspector",
         "version": "1.0.0"
       }
     }
   }
   ```

2. **列出可用工具**
   ```json
   {
     "jsonrpc": "2.0",
     "id": 2,
     "method": "tools/list"
   }
   ```

3. **获取工具信息**
   ```json
   {
     "jsonrpc": "2.0",
     "id": 3,
     "method": "tools/call",
     "params": {
       "name": "aks-list-clusters",
       "arguments": {}
     }
   }
   ```

### 第四步：验证 OAuth 功能

1. **测试有效令牌**
   - 使用正确的访问令牌
   - 验证所有 MCP 调用都成功
   - 检查返回的数据是否正确

2. **测试无效令牌**
   ```bash
   # 临时更新为无效令牌
   ./update_inspector_token.sh "invalid-token"
   ```
   - 在 Inspector 中重新连接
   - 验证收到 401 Unauthorized 错误
   - 检查错误消息是否包含认证失败信息

3. **测试令牌过期**
   - 等待访问令牌过期（通常 1 小时）
   - 尝试执行 MCP 调用
   - 验证收到令牌过期错误
   - 重新获取令牌并测试

### 第五步：测试不同的 AKS 操作

1. **只读操作（readonly 访问级别）**
   ```json
   {
     "jsonrpc": "2.0",
     "id": 4,
     "method": "tools/call",
     "params": {
       "name": "aks-list-clusters",
       "arguments": {}
     }
   }
   ```

2. **网络信息查询**
   ```json
   {
     "jsonrpc": "2.0",
     "id": 5,
     "method": "tools/call",
     "params": {
       "name": "network-list-vnets",
       "arguments": {}
     }
   }
   ```

3. **监控和诊断**
   ```json
   {
     "jsonrpc": "2.0",
     "id": 6,
     "method": "tools/call",
     "params": {
       "name": "monitor-get-cluster-health",
       "arguments": {
         "cluster_name": "my-aks-cluster",
         "resource_group": "my-resource-group"
       }
     }
   }
   ```

## 故障排除

### 常见问题

#### 1. Inspector 无法连接到服务器

**检查步骤**：
```bash
# 1. 验证服务器运行
curl http://localhost:8080/health

# 2. 测试基本连接
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}'

# 3. 检查网络连接
telnet localhost 8080
```

#### 2. 认证失败

**检查步骤**：
```bash
# 1. 验证令牌格式
echo "你的访问令牌" | cut -d. -f2 | base64 -d | jq .

# 2. 测试令牌有效性
curl -X POST http://localhost:8080/oauth/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=你的访问令牌"

# 3. 检查配置文件
jq . ~/.config/mcp-inspector/aks-mcp.json
```

#### 3. MCP 协议错误

**常见错误和解决方案**：

- **`Method not found`**：检查方法名是否正确
- **`Invalid params`**：验证参数格式和必需字段
- **`Internal error`**：查看服务器日志了解详细错误

#### 4. Inspector 界面问题

```bash
# 重启 Inspector
pkill mcp-inspector
./mcp-inspector

# 清除配置缓存
rm -rf ~/.config/mcp-inspector/cache

# 检查日志
tail -f ~/.config/mcp-inspector/logs/inspector.log
```

### 调试技巧

1. **启用详细日志**
   ```bash
   # 启动 AKS-MCP 时启用调试
   ./aks-mcp --log-level debug --config oauth-config.yaml
   ```

2. **使用 curl 验证**
   ```bash
   # 创建测试脚本
   cat > test_mcp_auth.sh << 'EOF'
   #!/bin/bash
   TOKEN="$1"
   
   if [ -z "$TOKEN" ]; then
       echo "用法: $0 <access_token>"
       exit 1
   fi
   
   echo "=== 测试认证 ==="
   curl -X POST http://localhost:8080/mcp \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05"}}' | jq .
   
   echo -e "\n=== 测试工具列表 ==="
   curl -X POST http://localhost:8080/mcp \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}' | jq .
   EOF
   
   chmod +x test_mcp_auth.sh
   ./test_mcp_auth.sh "你的访问令牌"
   ```

3. **网络抓包分析**
   ```bash
   # 使用 tcpdump 监控网络流量
   sudo tcpdump -i lo -A -s 0 port 8080
   
   # 或者使用 wireshark
   wireshark -i lo -f "port 8080"
   ```

### 性能测试

1. **并发连接测试**
   ```bash
   # 创建并发测试脚本
   cat > concurrent_test.sh << 'EOF'
   #!/bin/bash
   TOKEN="$1"
   CONCURRENT=${2:-5}
   
   for i in $(seq 1 $CONCURRENT); do
       (
           echo "客户端 $i 开始测试..."
           curl -X POST http://localhost:8080/mcp \
             -H "Authorization: Bearer $TOKEN" \
             -H "Content-Type: application/json" \
             -d '{"jsonrpc": "2.0", "id": '$i', "method": "tools/list"}' \
             -w "客户端 $i: %{time_total}s\n" -o /dev/null -s
       ) &
   done
   wait
   EOF
   
   chmod +x concurrent_test.sh
   ./concurrent_test.sh "你的访问令牌" 10
   ```

2. **令牌刷新测试**
   ```bash
   # 监控令牌有效期
   while true; do
       echo "$(date): 测试令牌有效性..."
       curl -X POST http://localhost:8080/oauth/introspect \
         -H "Content-Type: application/x-www-form-urlencoded" \
         -d "token=你的访问令牌" | jq '.active'
       sleep 300  # 每5分钟检查一次
   done
   ```

## 最佳实践

1. **安全性**
   - 定期轮换访问令牌
   - 不要在配置文件中硬编码敏感信息
   - 使用环境变量存储令牌

2. **可维护性**
   - 使用脚本自动化常见任务
   - 保持配置文件的版本控制
   - 记录测试结果和问题

3. **监控**
   - 监控令牌过期时间
   - 跟踪 API 调用的成功率
   - 记录性能指标

通过以上步骤，你应该能够成功使用 MCP Inspector 测试 AKS-MCP 的 OAuth 认证功能。