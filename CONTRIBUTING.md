# Contributing to AKS-MCP

Thank you for your interest in contributing to AKS-MCP! This guide will help you get started with contributing to the project.

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Getting Started

### Prerequisites

Before contributing, ensure you have the following installed:

- **Go** ≥ `1.24.x` - [Download Go](https://golang.org/dl/)
- **Azure CLI** - [Install Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
- **Git** - [Install Git](https://git-scm.com/downloads)
- **GNU Make** `4.x` or later
- **Node.js** _(optional, for MCP inspector)_

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/aks-mcp.git
   cd aks-mcp
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/Azure/aks-mcp.git
   ```

## Development Environment Setup

### 1. Authenticate with Azure

```bash
# Login to Azure CLI
az login

# Set your preferred subscription (optional)
az account set --subscription "Your Subscription Name or ID"
```

### 2. Build the Project

```bash
# Build the binary
make build
```

### 3. Verify Installation

```bash
# Run the binary to see help
./aks-mcp --help

# Or using make
make run
```

## Running AKS-MCP Locally

### Basic Local Testing

1. **Start the MCP server locally:**

   ```bash
   # Run with default settings (readonly access)
   ./aks-mcp

   # Run with elevated permissions for testing
   ./aks-mcp --access-level readwrite

   # Run with admin permissions (full access)
   ./aks-mcp --access-level admin
   ```

2. **Use MCP Inspector for debugging:**
   ```bash
   npx @modelcontextprotocol/inspector ./aks-mcp --access-level=readwrite
   ```

## Testing with AI Agents

### GitHub Copilot in VS Code

1. **Create MCP configuration for your workspace:**

   ```bash
   mkdir -p .vscode
   ```

2. **Create `.vscode/mcp.json`:**

   ```json
   {
     "servers": {
       "aks-mcp-dev": {
         "type": "stdio",
         "command": "./aks-mcp",
         "args": [
	   "--access-level",
           "readwrite",
           "--verbose"
         ]
       }
     }
   }
   ```

3. **Restart VS Code and test:**
   - Open GitHub Copilot Chat
   - Switch to Agent mode
   - Verify the tools are loaded by clicking **Tools** button
   - Try: _"List all my AKS clusters"_

### Claude Desktop

1. **Create MCP configuration for Claude Desktop:**

   ```json
   {
     "mcpServers": {
       "aks-mcp-dev": {
         "command": "/absolute/path/to/your/aks-mcp/aks-mcp",
         "args": ["--access-level", "readwrite"]
       }
     }
   }
   ```

2. **Test with Claude:**
   - Start a conversation
   - Ask: _"What AKS clusters do I have?"_
   - Verify the MCP tools are working

## Making Changes

### Branching Strategy

1. **Create a feature branch:**

   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

2. **Keep your branch updated:**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

### Adding New Features

The project uses a component-based architecture where each Azure service has its own component. Here's how to add new MCP tools:

#### 1. **Choose the Right Component Type:**

- **Azure CLI-based tools** (like `az aks`, `az fleet`): Use `CommandExecutor` interface
- **Azure SDK-based tools** (like monitoring, network resources): Use `ResourceHandler` interface

#### 2. **For Azure CLI-based Tools:**

1. **Create executor in the appropriate component directory:**

   ```go
   // internal/components/yourcomponent/executor.go
   type YourExecutor struct{}

   func NewYourExecutor() *YourExecutor {
       return &YourExecutor{}
   }

   func (e *YourExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
       // Your CLI command logic here
       return result, nil
   }
   ```

2. **Create tool registration:**

   ```go
   // internal/components/yourcomponent/registry.go
   func RegisterYourTool(cfg *config.ConfigData) mcp.Tool {
       return mcp.NewTool(
           "your_tool_name",
           mcp.WithDescription("Description of your tool"),
           mcp.WithString("operation", mcp.Description("Operation to perform"), mcp.Required()),
           // Add other parameters as needed
       )
   }
   ```

3. **Register in server.go:**
   ```go
   // internal/server/server.go - Add to appropriate register function
   func (s *Service) registerYourComponent() {
       log.Println("Registering your tool: your_tool_name")
       yourTool := yourcomponent.RegisterYourTool(s.cfg)
       s.mcpServer.AddTool(yourTool, tools.CreateToolHandler(yourcomponent.NewYourExecutor(), s.cfg))
   }
   ```

#### 3. **For Azure SDK-based Tools:**

1. **Create handler:**

   ```go
   // internal/components/yourcomponent/handlers.go
   func GetYourHandler(azClient *azureclient.AzureClient, cfg *config.ConfigData) tools.ResourceHandler {
       return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
           // Your Azure SDK logic here
           return result, nil
       })
   }
   ```

2. **Create tool registration:** (same as above)

3. **Register in server.go:**
   ```go
   // internal/server/server.go - Add to appropriate register function
   func (s *Service) registerYourComponent() {
       log.Println("Registering your tool: your_tool_name")
       yourTool := yourcomponent.RegisterYourTool()
       s.mcpServer.AddTool(yourTool, tools.CreateResourceHandler(yourcomponent.GetYourHandler(s.azClient, s.cfg), s.cfg))
   }
   ```

#### 4. **Component Structure:**

Each component should follow this structure:

```
internal/components/yourcomponent/
├── registry.go        # Tool definitions and parameters
├── handlers.go        # Business logic handlers
├── executor.go        # CLI command executors (if needed)
├── types.go          # Data structures (if needed)
├── helpers.go        # Helper functions (if needed)
└── *_test.go         # Unit tests
```

#### 5. **Interface Definitions:**

```go
// For CLI-based tools
type CommandExecutor interface {
    Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error)
}

// For SDK-based tools
type ResourceHandler interface {
    Handle(params map[string]interface{}, cfg *config.ConfigData) (string, error)
}
```

#### 6. **Access Level Validation:**

Tools automatically respect access levels through the configuration. For custom validation:

```go
import "github.com/Azure/aks-mcp/internal/security"

// In your handler/executor
if !security.ValidateAccessLevel(cfg.AccessLevel, "readwrite") {
    return "", fmt.Errorf("this operation requires readwrite or admin access")
}
```

#### 7. **Register Your Component:**

Add your component registration to the appropriate function in `server.go`:

- `registerAzureComponents()` for Azure-specific tools
- `registerKubernetesComponents()` for Kubernetes-related tools

#### 8. **Testing Your Component:**

Create comprehensive tests:

```go
// internal/components/yourcomponent/handlers_test.go
func TestYourHandler(t *testing.T) {
    // Test your handler implementation
}

// internal/components/yourcomponent/registry_test.go
func TestRegisterYourTool(t *testing.T) {
    // Test tool registration
}
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Run tests with coverage
make test-coverage

# Run specific package tests
go test -v ./internal/server/...

# Run tests in verbose mode
make test-verbose
```

### Code Quality Checks

```bash
# Run all quality checks
make check

# Individual checks
make fmt      # Format code
make vet      # Run go vet
make lint     # Run golangci-lint

# Fix linting issues automatically
make lint-fix
```

## Submitting Changes

### Before Submitting

1. **Ensure code quality:**

   ```bash
   make check
   ```

2. **Update documentation if needed:**

   - Update README.md for user-facing changes
   - Add inline code comments for complex logic

3. **Test thoroughly:**
   ```bash
   make test-coverage
   ```

### Pull Request Process

1. **Commit your changes:**

   ```bash
   git add .
   git commit -m "feat: add new AKS monitoring tool"
   ```

2. **Push to your fork:**

   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create Pull Request:**
   - Use clear, descriptive titles
   - Include detailed description of changes
   - Reference related issues: "Fixes #123"
   - Ensure CI checks pass

### Commit Message Guidelines

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>[(optional scope)]: <description>

[optional body]

[optional footer(s)]
```

**Types:**

- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation updates
- `style`: Code style changes (no logic changes)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

- `feat(azure): add support for AKS diagnostic settings`
- `fix(security): validate access level for admin operations`
- `docs: update contribution guidelines`

## Release Process

### Version Management

Versions are managed using Git tags following [Semantic Versioning](https://semver.org/):

- `MAJOR.MINOR.PATCH` (e.g., `1.0.0`)
- Pre-release: `1.0.0-alpha.1`, `1.0.0-beta.1`, `1.0.0-rc.1`

### Creating Releases

1. **Prepare release branch:**

   ```bash
   git checkout -b release/v1.0.0
   ```

2. **Update version-related files:**

   - Ensure changelog is updated
   - Update version references in docs

3. **Test release build:**

   ```bash
   make release
   make checksums
   ```

4. **Create release PR and merge**

5. **Tag the release:**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push upstream v1.0.0
   ```

### Automated Release

The project uses GitHub Actions for automated releases:

- **SLSA3 compliant** release artifacts
- **Multi-platform** binaries (Linux, macOS, Windows)
- **Checksums** and signatures for verification

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` for code formatting
- Run `golangci-lint` for linting
- Write clear, self-documenting code
- Add comments for complex logic

### Error Handling

```go
// Good: Specific error messages
if err != nil {
    return fmt.Errorf("failed to list AKS clusters in resource group %s: %w", resourceGroup, err)
}

// Avoid: Generic error messages
if err != nil {
    return err
}
```

### Logging

```go
// Use structured logging
log.Info("Processing AKS operation",
    "operation", operation,
    "cluster", clusterName,
    "resourceGroup", resourceGroup)
```

### Testing Guidelines

- Write unit tests for all new functionality
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error conditions
- Aim for >80% code coverage

### Security Best Practices

- Validate all user inputs
- Respect access level permissions
- Never log sensitive information
- Use secure defaults
- Follow principle of least privilege

## Getting Help

### Documentation

- [README.md](README.md) - Project overview and installation
- [Documentation](docs/) - Detailed technical documentation

### Communication

- **GitHub Issues**: Report bugs or request features
- **GitHub Discussions**: Ask questions or discuss ideas
- **Security Issues**: Report [security advisory](https://github.com/Azure/aks-mcp/security/advisories)

## License

By contributing to AKS-MCP, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).

---

Thank you for contributing to AKS-MCP! Your contributions help make Azure Kubernetes Service more accessible through AI assistants.
