package server

import (
	"fmt"
	"log"

	"github.com/Azure/aks-mcp/internal/az"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
	"github.com/Azure/aks-mcp/internal/version"
	"github.com/mark3labs/mcp-go/server"
)

// Service represents the MCP Kubernetes service
type Service struct {
	cfg       *config.ConfigData
	mcpServer *server.MCPServer
}

// NewService creates a new MCP Kubernetes service
func NewService(cfg *config.ConfigData) *Service {
	return &Service{
		cfg: cfg,
	}
}

// Initialize initializes the service
func (s *Service) Initialize() error {
	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"AKS MCP",
		version.GetVersion(),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// // Register generic az tool
	// azTool := az.RegisterAz()
	// s.mcpServer.AddTool(azTool, tools.CreateToolHandler(az.NewExecutor(), s.cfg))

	// Register individual az commands
	s.registerAzCommands()

	return nil
}

// Run starts the service with the specified transport
func (s *Service) Run() error {
	// Start the server
	switch s.cfg.Transport {
	case "stdio":
		log.Println("MCP Kubernetes version:", version.GetVersion())
		log.Println("Listening for requests on STDIO...")
		return server.ServeStdio(s.mcpServer)
	case "sse":
		url := fmt.Sprintf("http://localhost:%d", s.cfg.Port)
		sse := server.NewSSEServer(s.mcpServer, server.WithBaseURL(url))

		log.Println("MCP Kubernetes version:", version.GetVersion())
		log.Printf("SSE server listening on %s", url)
		return sse.Start(fmt.Sprintf(":%d", s.cfg.Port))
	default:
		return fmt.Errorf("invalid transport type: %s (must be 'stdio' or 'sse')", s.cfg.Transport)
	}
}

// registerAzCommands registers individual az commands as separate tools
func (s *Service) registerAzCommands() {
	// Register read-only az commands (available at all access levels)
	for _, cmd := range az.GetReadOnlyAzCommands() {
		azTool := az.RegisterAzCommand(cmd)
		commandExecutor := az.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}

	// Register read-write commands if access level is readwrite or admin
	if s.cfg.AccessLevel == "readwrite" || s.cfg.AccessLevel == "admin" {
		// Register read-write az commands
		for _, cmd := range az.GetReadWriteAzCommands() {
			azTool := az.RegisterAzCommand(cmd)
			commandExecutor := az.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}
	}

	// Register admin commands only if access level is admin
	if s.cfg.AccessLevel == "admin" {
		// Register admin az commands
		for _, cmd := range az.GetAdminAzCommands() {
			azTool := az.RegisterAzCommand(cmd)
			commandExecutor := az.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}
	}
}
