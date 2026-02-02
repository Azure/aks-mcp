# AKS MCP Transport Lab Guide

This directory contains tools for lab exercises with the AKS-MCP server using different transport protocols and verbose logging capabilities.

## Overview

The AKS-MCP server supports three transport protocols:
- **STDIO**: Direct process communication (default)
- **SSE**: Server-Sent Events over HTTP
- **Streamable HTTP**: Standard HTTP with streaming support

## Files

- `test_aks_mcp.py` - Multi-transport lab client with comprehensive scenarios
- `aks-mcp-demo.ipynb` - Jupyter notebook demonstration
- `TRANSPORT_TESTING.md` - Detailed transport lab documentation

## Prerequisites

- Python 3.11+
- Go 1.21+ (for building the server)
- Azure CLI (configured with appropriate permissions)
- Azure OpenAI or OpenAI API access

## Quick Start

### 1. Build AKS-MCP Server
```bash
cd /Users/hieunguyennhu/github/aks-mcp
make build
```

### 2. Setup Environment
```bash
# Optional: Create virtual environment (recommended)
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate

# Install Python dependencies
pip install semantic-kernel[mcp] python-dotenv
# Copy and configure environment variables
cp .env.example .env
# Edit .env with your Azure OpenAI settings:
# AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
# AZURE_OPENAI_API_KEY=your-api-key
# AZURE_OPENAI_DEPLOYMENT_NAME=gpt-4o
```

### 3. Lab Exercise: Each Transport

#### STDIO Transport (Default)
```bash
# Lab exercise directly - no server needed
python test_aks_mcp.py
```

#### SSE Transport
```bash
# Terminal 1: Start SSE server with verbose logging
./aks-mcp --transport sse --port 8000 --access-level admin --verbose

# Terminal 2: Run lab client
python test_aks_mcp.py --transport sse --host localhost --port 8000
```

#### Streamable HTTP Transport
```bash
# Terminal 1: Start HTTP server with verbose logging  
./aks-mcp --transport streamable-http --port 8000 --access-level admin --verbose

# Terminal 2: Run lab client
python test_aks_mcp.py --transport streamable-http --host localhost --port 8000
```

## Verbose Logging

Add `--verbose` or `-v` to any server command to see detailed tool call information:

```bash
./aks-mcp --transport sse --port 8000 --access-level admin --verbose
```

**Example verbose output:**
```
>>> [az_aks_operations] {"args":"","operation":"list"}
    Result: 20291 bytes (truncated): [{"aadProfile":{"enableAzureRbac":true...

>>> [aks_monitoring] {"cluster_name":"hub","operation":"resource_health","parameters":{"start_time":"2025-01-01T00:00:00Z"}}
    ERROR: missing or invalid start_time parameter
```

## Lab Scenarios

The lab client runs 5 comprehensive scenarios:

1. **Cluster Discovery** - Lists all AKS clusters with health status
2. **Diagnostic Detectors** - Discovers available diagnostic tools
3. **Kubernetes Workloads** - Analyzes pods, services, deployments
4. **Fleet Management** - Checks Azure Kubernetes Fleet configuration
5. **Advisory Recommendations** - Retrieves Azure Advisor suggestions

## Command Reference

### Server Commands
```bash
# STDIO (default)
./aks-mcp --access-level admin --verbose

# SSE on custom port
./aks-mcp --transport sse --port 9000 --access-level admin --verbose

# HTTP on custom host/port
./aks-mcp --transport streamable-http --host 0.0.0.0 --port 8080 --access-level admin --verbose
```

### Client Commands
```bash
# Lab exercise with specific transport
python test_aks_mcp.py --transport <stdio|sse|streamable-http>

# Custom host/port for HTTP transports
python test_aks_mcp.py --transport sse --host localhost --port 9000

# Help
python test_aks_mcp.py --help
```

## Troubleshooting

### Connection Issues
1. **Check server is running**: Ensure the server started successfully
2. **Verify port availability**: `lsof -i :8000`
3. **Test connectivity**: `curl http://localhost:8000/mcp` (for HTTP transports)
4. **Check firewall**: Local firewall may block connections

### Transport-Specific Issues

**STDIO**:
- Ensure `aks-mcp` binary exists and is executable
- Check Azure credentials are configured

**SSE**:
- Server endpoint: `http://localhost:8000/sse`
- Some proxies may interfere with SSE connections

**Streamable HTTP**:
- Server endpoint: `http://localhost:8000/mcp` 
- May show harmless cleanup warnings on client exit

### Known Issues
- Streamable HTTP transport shows async cleanup warnings - these are harmless
- All transports require valid Azure credentials for AKS operations

## Jupyter Notebook

Install Jupyter and required dependencies:

```bash
# Install Jupyter and notebook dependencies
pip install jupyter semantic-kernel[mcp] python-dotenv pandas

# Start Jupyter
jupyter notebook aks-mcp-demo.ipynb
```

The notebook demonstrates the same functionality with step-by-step execution and detailed output.

## Performance Tips

- Use `--verbose` only during lab exercises - it generates significant log output
- STDIO transport has the lowest overhead
- HTTP transports allow multiple concurrent clients
- Set appropriate `--timeout` values for long-running operations

## Security Notes

- `--access-level admin` provides full cluster access
- Use `--access-level readonly` for safer lab exercises
- Never expose HTTP/SSE servers publicly without proper authentication
- All transports require valid Azure RBAC permissions

## Getting Help

```bash
# Server help
./aks-mcp --help

# Client help  
python test_aks_mcp.py --help

# Check available AKS functions
./aks-mcp --transport stdio --access-level admin | grep "Registering"
```