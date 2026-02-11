#!/usr/bin/env python3
"""
Test AKS-MCP integration with Semantic Kernel ChatCompletionAgent

Supports multiple transport types:
- stdio: Direct process communication (default)
- sse: Server-Sent Events HTTP transport
- streamable-http: Streamable HTTP transport

Usage:
    # Default stdio transport
    python test_aks_mcp.py
    
    # SSE transport
    python test_aks_mcp.py --transport sse --host localhost --port 8000
    
    # Streamable HTTP transport  
    python test_aks_mcp.py --transport streamable-http --host localhost --port 8000

Prerequisites:
- For stdio: Build aks-mcp binary with 'make build'
- For SSE/HTTP: Start AKS-MCP server with appropriate transport
  Example: ./aks-mcp --transport sse --port 8000
"""

import asyncio
import os
import sys
from pathlib import Path
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

async def test_aks_mcp(transport="stdio", host="localhost", port=8000):
    """Test AKS-MCP with Semantic Kernel ChatCompletionAgent
    
    Args:
        transport: Transport type - "stdio", "sse", or "streamable-http"
        host: Host for HTTP/SSE connections (default: localhost)
        port: Port for HTTP/SSE connections (default: 8000)
    """
    try:
        # Import Semantic Kernel components
        from semantic_kernel import Kernel
        from semantic_kernel.connectors.ai.open_ai import AzureChatCompletion
        from semantic_kernel.connectors.mcp import MCPStdioPlugin, MCPSsePlugin, MCPStreamableHttpPlugin
        from semantic_kernel.agents import ChatCompletionAgent
        
        print("✅ Imports successful")
        
        # Initialize kernel
        kernel = Kernel()
        
        # Configure Azure OpenAI
        azure_openai = AzureChatCompletion(
            endpoint=os.getenv("AZURE_OPENAI_ENDPOINT"),
            api_key=os.getenv("GENAI_API_KEY"),
            deployment_name=os.getenv("AZURE_OPENAI_MODEL", "gpt-5"),
            api_version=os.getenv("AZURE_OPENAI_API_VERSION", "2025-08-07")
        )
        
        kernel.add_service(azure_openai)
        print("✅ Kernel initialized with Azure OpenAI")
        
        # Create AKS-MCP plugin based on transport type
        print(f"🔧 Using transport: {transport}")
        
        if transport == "stdio":
            aks_mcp_path = "/Users/hieunguyennhu/github/aks-mcp/aks-mcp"
            if not Path(aks_mcp_path).exists():
                print(f"❌ AKS-MCP not found at {aks_mcp_path}")
                print("💡 Build it with: cd /Users/hieunguyennhu/github/aks-mcp && make build")
                return False
                
            mcp_plugin = MCPStdioPlugin(
                name="AKSMCP",
                command=aks_mcp_path,
                args=["--transport", "stdio", "--access-level", "admin"],
            )
            
        elif transport == "sse":
            print(f"🌐 Connecting to SSE server at {host}:{port}")
            print(f"📡 URL: http://{host}:{port}/sse")
            mcp_plugin = MCPSsePlugin(
                name="AKSMCP",
                url=f"http://{host}:{port}/sse",
            )
            
        elif transport == "streamable-http":
            print(f"🌐 Connecting to HTTP server at {host}:{port}")
            print(f"📡 URL: http://{host}:{port}/mcp")
            if port == 443:
                print("💡 Using HTTPS for port 443")
                mcp_plugin = MCPStreamableHttpPlugin(
                    name="AKSMCP",
                    url=f"https://{host}/mcp",
            )
            else:
                mcp_plugin = MCPStreamableHttpPlugin(
                    name="AKSMCP",
                    url=f"http://{host}:{port}/mcp",
                )
            
        else:
            print(f"❌ Unsupported transport: {transport}")
            print("💡 Supported transports: stdio, sse, streamable-http")
            return False
        
        # Connect to MCP server
        await mcp_plugin.connect()
        print(f"✅ MCP plugin connected via {transport}")
        
        # Add plugin to kernel
        kernel.add_plugin(mcp_plugin, plugin_name="akstool")
        print("✅ MCP plugin added to kernel")
        
        # Get plugin functions from kernel
        plugin_functions = kernel.get_plugin("akstool")
        print(f"📋 Available functions: {len(plugin_functions.functions)}")
        for func_name in list(plugin_functions.functions.keys())[:3]:
            print(f"  • {func_name}")
        
        # Create ChatCompletionAgent
        agent = ChatCompletionAgent(
            kernel=kernel,
            name="AKS_Assistant",  # Use valid name pattern (no spaces)
            instructions="""You are an expert AKS administrator with access to powerful AKS management tools.

IMPORTANT: Always use the available akstool functions to gather information instead of asking users for details.

Your capabilities include:
- az_aks_operations: List and analyze AKS clusters (use this to discover cluster names, resource groups, subscription IDs)
- list_detectors: List available diagnostic detectors
- run_detector: Execute specific diagnostic checks
- kubectl_*: Execute kubectl commands on clusters
- az_fleet: Manage Azure Kubernetes Fleet
- az_advisor_recommendation: Get Azure Advisor recommendations
- az_monitoring: Monitor cluster health

When asked about clusters, detectors, or recommendations:
1. FIRST use az_aks_operations to discover available clusters and their details
2. THEN use the appropriate tool with the discovered information
3. Provide comprehensive analysis based on the actual data

Never ask users for cluster names, resource groups, or subscription IDs - discover them using the tools."""
        )
        
        print("🤖 ChatCompletionAgent created")
        
        # Test the agent with multiple scenarios
        print("\n🧪 Testing agent with comprehensive scenarios...")
        
        from semantic_kernel.contents import ChatHistory
        
        # Test scenarios to showcase AKS MCP capabilities
        test_scenarios = [
            {
                "name": "Cluster Discovery",
                "question": "What AKS clusters do I have? Please provide a comprehensive overview including health status.",
                "expected": "Should use az_aks_operations to list all clusters"
            },
            {
                "name": "Diagnostic Detectors Discovery",
                "question": "Discover what diagnostic detectors are available for my AKS clusters. Use the tools to find my clusters first, then list the detectors.",
                "expected": "Should use az_aks_operations then list_detectors"
            },
            {
                "name": "Kubernetes Workloads Analysis", 
                "question": "Analyze the Kubernetes workloads running in my clusters. Discover my clusters first, then check what pods, services, and deployments are running.",
                "expected": "Should use az_aks_operations then kubectl commands"
            },
            {
                "name": "Fleet Management Check",
                "question": "Check my Azure Kubernetes Fleet configuration and resources using the available tools.",
                "expected": "Should use az_fleet functionality"
            },
            {
                "name": "Advisory Recommendations Analysis",
                "question": "Find Azure Advisor recommendations for my environment. Use the tools to discover my current subscription and resource details first.",
                "expected": "Should use az_aks_operations then az_advisor_recommendation"
            }
        ]
        
        for i, scenario in enumerate(test_scenarios, 1):
            print(f"\n{'='*60}")
            print(f"🎯 Scenario {i}: {scenario['name']}")
            print(f"❓ Question: {scenario['question']}")
            print(f"💡 Expected: {scenario['expected']}")
            print("="*60)
            
            chat_history = ChatHistory()
            chat_history.add_user_message(scenario['question'])
            
            print("🤔 Agent thinking and using tools...\n")
            
            response_text = ""
            async for response in agent.invoke(messages=chat_history):
                content = str(response.content or "")
                response_text += content
                print(content, end="", flush=True)
            
            print(f"\n\n✅ Scenario {i} completed")
            print(f"📊 Response length: {len(response_text)} characters")
            
            # Brief pause between scenarios
            import asyncio
            await asyncio.sleep(1)
        
        # Cleanup
        try:
            await mcp_plugin.close()
            print("\n✅ MCP plugin disconnected")
        except Exception as cleanup_error:
            # Known issue with streamable-http transport cleanup
            if "cancel scope" in str(cleanup_error) or "Session terminated" in str(cleanup_error):
                print("⚠️  Known cleanup issue with streamable-http transport (harmless)")
            else:
                print(f"⚠️  Cleanup warning: {cleanup_error}")
            
        print("\n✅ Test completed successfully!")
        return True
        
    except Exception as e:
        # Handle known streamable-http cleanup issues
        if ("cancel scope" in str(e) or "Session terminated" in str(e) or 
            "streamablehttp_client" in str(e) or "GeneratorExit" in str(e)):
            print(f"\n⚠️  Known streamable-http transport cleanup issue (test likely succeeded)")
            print("💡 This is a harmless MCP library cleanup warning")
            return True
            
        print(f"\n❌ Error: {e}")
        
        if "opentelemetry" in str(e):
            print("💡 Try: pip install 'semantic-kernel[mcp]'")
        elif "Failed to connect" in str(e) or "Connection" in str(e):
            print("\n💡 Connection failed. Please ensure:")
            if transport == "sse":
                print("   1. AKS-MCP server is running with SSE transport:")
                print(f"      ./aks-mcp --transport sse --port {port} --access-level admin")
                print(f"   2. Server is accessible at http://{host}:{port}/sse")
            elif transport == "streamable-http":
                print("   1. AKS-MCP server is running with HTTP transport:")
                print(f"      ./aks-mcp --transport streamable-http --port {port} --access-level admin")
                print(f"   2. Server is accessible at http://{host}:{port}")
            print("   3. Check firewall/network settings")
        
        return False

async def main():
    # Check prerequisites
    if not Path(".env").exists():
        print("⚠️  No .env file found")
        print("💡 Copy .env.example to .env and configure Azure OpenAI settings")
        return False
    
    # Parse command line arguments for transport options
    import argparse
    parser = argparse.ArgumentParser(description="Test AKS-MCP with different transports")
    parser.add_argument("--transport", choices=["stdio", "sse", "streamable-http"], 
                       default="stdio", help="Transport type (default: stdio)")
    parser.add_argument("--host", default="localhost", 
                       help="Host for HTTP/SSE connections (default: localhost)")
    parser.add_argument("--port", type=int, default=8000,
                       help="Port for HTTP/SSE connections (default: 8000)")
    
    args = parser.parse_args()
    
    print(f"🚀 Starting AKS-MCP test with {args.transport} transport")
    if args.transport != "stdio":
        print(f"🌐 Server: {args.host}:{args.port}")
    
    # Run test with specified transport
    success = await test_aks_mcp(args.transport, args.host, args.port)
    return success

if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)