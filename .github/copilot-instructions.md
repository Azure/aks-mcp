# Copilot Instructions

This file provides guidance to Github Copilot when working with code in this repository.

## Project Overview

AKS-MCP is a Model Context Protocol (MCP) server that enables AI assistants to interact with Azure Kubernetes Service (AKS) clusters. It serves as a bridge between AI tools and AKS, translating natural language requests into AKS operations.

## Architecture

### High-Level Structure
- **Entry Point**: `cmd/aks-mcp/main.go` - Main application entry point
- **Core Server**: `internal/server/server.go` - MCP server implementation with tool registration
- **Configuration**: `internal/config/` - Command line parsing and configuration management
- **Azure Integration**: `internal/azure/` - Azure SDK clients with caching layer
- **Resource Handlers**: `internal/azure/resourcehandlers/` - Handlers for Azure resource operations (VNet, NSG, Route Tables, Subnets)
- **Security**: `internal/security/` - Access level validation and security controls
- **Tools**: `internal/tools/` - MCP tool adapters and interfaces

### Key Components
- **Access Levels**: Three security levels (readonly, readwrite, admin) control available operations
- **Resource Caching**: Azure resources are cached internally for performance
- **Transport Support**: Supports stdio transport
- **Tool Registration**: Tools are dynamically registered based on access level

## Development Commands

### Build
```bash
go build -o aks-mcp ./cmd/aks-mcp
```

### Test
```bash
go test -v ./...                           # Run all tests
go test -race -coverprofile=coverage.txt -covermode=atomic ./...  # Run tests with coverage
go test -v ./internal/azure/...         # Run specific package tests
```

### Lint
```bash
golangci-lint run                       # Run linter
golangci-lint run --timeout=5m          # Run with extended timeout
```

## Configuration

### Command Line Arguments
- `--timeout`: Command execution timeout in seconds (default: 600)
- `--access-level`: Access level (readonly, readwrite, admin)

### Environment Variables
Standard Azure authentication variables are supported:
- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`
- `AZURE_SUBSCRIPTION_ID`

## Testing Strategy

The codebase includes comprehensive unit tests:
- **Azure Client Tests**: `internal/azure/client_test.go`
- **Resource Handler Tests**: `internal/azure/resourcehandlers/handlers_test.go`
- **Security Validation Tests**: `internal/security/validator_test.go`
- **Cache Tests**: `internal/azure/cache_test.go`
- **Helper Tests**: `internal/azure/resourcehelpers/helpers_test.go`

## MCP Tool Development

### Adding New Tools
1. Create handler function in appropriate `internal/azure/resourcehandlers/` file
2. Register tool in `internal/azure/resourcehandlers/registry.go`
3. Add tool registration in `internal/server/server.go`
4. Implement security validation based on access level

### Tool Handler Pattern
All tools follow the `ResourceHandler` interface:
```go
type ResourceHandler interface {
    Handle(params map[string]interface{}, cfg *config.ConfigData) (string, error)
}
```

## Security Considerations

- Access levels control available operations
- Input validation is performed through `internal/security/validator.go`
- Azure credentials are managed through Azure SDK default authentication
- All Azure operations respect the configured access level

## Dependencies

- **Azure SDK**: `github.com/Azure/azure-sdk-for-go/sdk`
- **MCP Go**: `github.com/mark3labs/mcp-go`
- **Command Line**: `github.com/spf13/pflag`
- **Shell Parsing**: `github.com/google/shlex`

## CI/CD

The project uses GitHub Actions for:
- **Linting**: golangci-lint across multiple OS platforms
- **Testing**: Unit tests with coverage reporting
- **Security**: Gosec security scanning
- **Building**: Go binary builds
- **Publishing**: SLSA3 compliant releases
