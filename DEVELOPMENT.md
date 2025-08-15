# Development Guide

This guide covers how to develop, build, test, and contribute to the AKS-MCP project.

## Prerequisites

- **Go** ≥ `1.24.x` installed on your local machine
- **Bash** available as `/usr/bin/env bash` (Makefile targets use multi-line recipes with fail-fast mode)
- **GNU Make** `4.x` or later
- **Docker** *(optional, for container builds and testing)*

> **Note:** If your login shell is different (e.g., `zsh` on **macOS**), you do **not** need to change it — the Makefile sets variables to run all recipes in `bash` for consistent behavior across platforms.

## Building from Source

This project includes a Makefile for convenient development, building, and testing. To see all available targets:

```bash
make help
```

### Quick Start

```bash
# Build the binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format and lint code
make check

# Build for all platforms
make release
```

### Common Development Tasks

```bash
# Install dependencies
make deps

# Build and run with --help
make run

# Clean build artifacts
make clean

# Install binary to GOBIN
make install
```

### Docker

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

## Manual Build

If you prefer to build without the Makefile:

```bash
go build -o aks-mcp ./cmd/aks-mcp
```

## Contributing

This project welcomes contributions and suggestions. Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
