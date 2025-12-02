# AKS-MCP Benchmark

Benchmark framework for testing AKS-MCP tool descriptions and AI model tool-using capabilities.

## Overview

This benchmark is an AI agent that:
1. Connects to AKS-MCP server via MCP protocol
2. Receives user prompts (troubleshooting questions)
3. Uses AI models to select and call appropriate tools
4. Evaluates the correctness of tool selection, parameters, and final answers

## Quick Start

### Prerequisites

- Go 1.21+
- Kubernetes cluster (kind/minikube/AKS)
- Azure OpenAI access

### Setup

```bash
# Build AKS-MCP
cd /path/to/aks-mcp
make build

# Build benchmark
cd bench
go build -o bench-tool ./cmd/bench

# Set Azure OpenAI credentials
export AZURE_OPENAI_ENDPOINT=https://xxx.openai.azure.com/
export AZURE_OPENAI_API_KEY=your-api-key
export AZURE_OPENAI_DEPLOYMENT=gpt-4o
```

### Run Tests

```bash
# Run a single test
./bench-tool run --test fixtures/kubectl/easy/02_pod_oom

# Run all easy tests
./bench-tool run --test fixtures/kubectl/easy

# Run with custom MCP binary
./bench-tool run --test fixtures/kubectl/easy --mcp-binary ../aks-mcp
```

## Test Cases

Currently supports **33 kubectl-only test cases** migrated from holmesgpt:

- **15 Easy tests** - Highest priority regression tests
- **9 Medium tests** - More complex scenarios
- **4 Hard tests** - Most challenging cases
- **5 No-tag tests** - Tests without difficulty markers

See `fixtures/kubectl/` for all test cases.

## Architecture

See [design.md](./design.md) for detailed architecture and design decisions.

## Development

```bash
# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Build
go build -o bench-tool ./cmd/bench
```

## Reports

Test results are saved to:
- `results/latest.json` - Latest run results (JSON)
- `results/latest.md` - Latest run report (Markdown)
- `results/history/` - Historical results

## License

Same as aks-mcp project.
