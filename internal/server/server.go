package server

import (
	"fmt"
	"log"

	"github.com/Azure/aks-mcp/internal/az"
	"github.com/Azure/aks-mcp/internal/azure"
	"github.com/Azure/aks-mcp/internal/azure/resourcehandlers"
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
	// Initialize configuration

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

	// Register kubectl commands for Kubernetes resource browsing
	s.registerKubectlCommands()

	// Register Azure resource tools (VNet, NSG, etc.)
	s.registerAzureResourceTools()

	// Register Azure Advisor tools
	s.registerAdvisorTools()

	return nil
}

// Run starts the service with the specified transport
func (s *Service) Run() error {
	log.Println("MCP Kubernetes version:", version.GetVersion())

	// Start the server
	switch s.cfg.Transport {
	case "stdio":
		log.Println("MCP Kubernetes version:", version.GetVersion())
		log.Println("Listening for requests on STDIO...")
		return server.ServeStdio(s.mcpServer)
	case "sse":
		sse := server.NewSSEServer(s.mcpServer)
		addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
		log.Printf("SSE server listening on %s", addr)
		return sse.Start(addr)
	case "streamable-http":
		streamableServer := server.NewStreamableHTTPServer(s.mcpServer)
		addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
		log.Printf("Streamable HTTP server listening on %s", addr)
		return streamableServer.Start(addr)
	default:
		return fmt.Errorf("invalid transport type: %s (must be 'stdio', 'sse' or 'streamable-http')", s.cfg.Transport)
	}
}

// registerAzCommands registers individual az commands as separate tools
func (s *Service) registerAzCommands() {
	// Register read-only az commands (available at all access levels)
	for _, cmd := range az.GetReadOnlyAzCommands() {
		log.Println("Registering az command:", cmd.Name)
		azTool := az.RegisterAzCommand(cmd)
		commandExecutor := az.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}

	// Register account management commands (available at all access levels)
	for _, cmd := range az.GetAccountAzCommands() {
		log.Println("Registering az command:", cmd.Name)
		azTool := az.RegisterAzCommand(cmd)
		commandExecutor := az.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}

	// Register read-write commands if access level is readwrite or admin
	if s.cfg.AccessLevel == "readwrite" || s.cfg.AccessLevel == "admin" {
		// Register read-write az commands
		for _, cmd := range az.GetReadWriteAzCommands() {
			log.Println("Registering az command:", cmd.Name)
			azTool := az.RegisterAzCommand(cmd)
			commandExecutor := az.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}
	}

	// Register admin commands only if access level is admin
	if s.cfg.AccessLevel == "admin" {
		// Register admin az commands
		for _, cmd := range az.GetAdminAzCommands() {
			log.Println("Registering az command:", cmd.Name)
			azTool := az.RegisterAzCommand(cmd)
			commandExecutor := az.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}
	}
}

// registerKubectlCommands registers kubectl commands for browsing Kubernetes resources
func (s *Service) registerKubectlCommands() {
	// Check if kubectl commands should be available based on access level
	// kubectl commands are read-only by default, so available at all access levels
	for _, cmd := range az.GetKubectlCommands() {
		log.Println("Registering kubectl command:", cmd.Name)
		kubectlTool := az.RegisterAzCommand(cmd)
		commandExecutor := az.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(kubectlTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}
}

func (s *Service) registerAzureResourceTools() {
	// Create Azure client for the resource tools (cache is internal to the client)
	azClient, err := azure.NewAzureClient(s.cfg)
	if err != nil {
		log.Printf("Warning: Failed to create Azure client: %v", err)
		return
	}

	// Register Network-related tools
	s.registerNetworkTools(azClient)

	// TODO: Add other resource categories in the future:
}

// registerNetworkTools registers all network-related Azure resource tools
func (s *Service) registerNetworkTools(azClient *azure.AzureClient) {
	log.Println("Registering Network tools...")

	// Register VNet info tool
	log.Println("Registering network tool: get_vnet_info")
	vnetTool := resourcehandlers.RegisterVNetInfoTool()
	s.mcpServer.AddTool(vnetTool, tools.CreateResourceHandler(resourcehandlers.GetVNetInfoHandler(azClient, s.cfg), s.cfg))

	// Register NSG info tool
	log.Println("Registering network tool: get_nsg_info")
	nsgTool := resourcehandlers.RegisterNSGInfoTool()
	s.mcpServer.AddTool(nsgTool, tools.CreateResourceHandler(resourcehandlers.GetNSGInfoHandler(azClient, s.cfg), s.cfg))

	// Register RouteTable info tool
	log.Println("Registering network tool: get_route_table_info")
	routeTableTool := resourcehandlers.RegisterRouteTableInfoTool()
	s.mcpServer.AddTool(routeTableTool, tools.CreateResourceHandler(resourcehandlers.GetRouteTableInfoHandler(azClient, s.cfg), s.cfg))

	// Register Subnet info tool
	log.Println("Registering network tool: get_subnet_info")
	subnetTool := resourcehandlers.RegisterSubnetInfoTool()
	s.mcpServer.AddTool(subnetTool, tools.CreateResourceHandler(resourcehandlers.GetSubnetInfoHandler(azClient, s.cfg), s.cfg))

	// Register Load Balancers info tool
	log.Println("Registering network tool: get_load_balancers_info")
	lbTool := resourcehandlers.RegisterLoadBalancersInfoTool()
	s.mcpServer.AddTool(lbTool, tools.CreateResourceHandler(resourcehandlers.GetLoadBalancersInfoHandler(azClient, s.cfg), s.cfg))
}

// registerAdvisorTools registers all Azure Advisor-related tools
func (s *Service) registerAdvisorTools() {
	log.Println("Registering Advisor tools...")

	// Register Azure Advisor recommendation tool (available at all access levels)
	log.Println("Registering advisor tool: az_advisor_recommendation")
	advisorTool := resourcehandlers.RegisterAdvisorRecommendationTool()
	s.mcpServer.AddTool(advisorTool, tools.CreateResourceHandler(resourcehandlers.GetAdvisorRecommendationHandler(s.cfg), s.cfg))
}
