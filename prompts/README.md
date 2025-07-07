# AKS-MCP Prompt Test Files

This folder contains prompt files for testing the AKS-MCP server functionality with AI assistants. These prompts are designed to validate different features and security aspects of the MCP server.

## Test Categories

### Basic Operations
- `basic-cluster-info.md` - Test basic cluster information retrieval
- `cluster-listing.md` - Test AKS cluster listing functionality
- `network-info.md` - Test network-related queries

### Security Testing
- `security-validation.md` - Test security validation and access controls
- `command-injection.md` - Test command injection prevention
- `access-levels.md` - Test different access level restrictions

### Advanced Features
- `complex-queries.md` - Test complex multi-step operations
- `error-handling.md` - Test error handling and edge cases

## Usage

1. Start your AKS-MCP server with appropriate configuration
2. Configure your AI assistant to use the MCP server
3. Copy and paste prompts from these files into your AI assistant
4. Verify the expected responses and security behavior

## Contributing

When adding new prompt tests:
1. Include clear test objectives
2. Specify expected outcomes
3. Document any prerequisites (subscriptions, permissions, etc.)
4. Include both positive and negative test cases
