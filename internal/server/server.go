package server

import (
	"fmt"
	"log"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/components/advisor"
	"github.com/Azure/aks-mcp/internal/components/azaks"
	"github.com/Azure/aks-mcp/internal/components/compute"
	"github.com/Azure/aks-mcp/internal/components/detectors"
	"github.com/Azure/aks-mcp/internal/components/fleet"
	"github.com/Azure/aks-mcp/internal/components/monitor"
	"github.com/Azure/aks-mcp/internal/components/monitor/diagnostics"
	"github.com/Azure/aks-mcp/internal/components/network"
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

	// Register Azure resource tools (VNet, NSG, etc.)
	s.registerAzureResourceTools()

	// Register Azure Advisor tools
	s.registerAdvisorTools()

	// Register AKS Control Plane tools
	s.registerControlPlaneTools()

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
	for _, cmd := range azaks.GetReadOnlyAzCommands() {
		log.Println("Registering az command:", cmd.Name)
		azTool := azaks.RegisterAzCommand(cmd)
		commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}

	// Register read-only az monitor commands (available at all access levels)
	for _, cmd := range monitor.GetReadOnlyMonitorCommands() {
		log.Println("Registering az monitor command:", cmd.Name)
		azTool := monitor.RegisterMonitorCommand(cmd)
		commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}

	// Register generic az fleet tool with structured parameters (available at all access levels)
	log.Println("Registering az fleet tool: az_fleet")
	fleetTool := fleet.RegisterFleet()
	s.mcpServer.AddTool(fleetTool, tools.CreateToolHandler(azcli.NewFleetExecutor(), s.cfg))

	// Register Azure Resource Health monitoring tool (available at all access levels)
	log.Println("Registering monitor tool: az_monitor_activity_log_resource_health")
	resourceHealthTool := monitor.RegisterResourceHealthTool()
	s.mcpServer.AddTool(resourceHealthTool, tools.CreateResourceHandler(monitor.GetResourceHealthHandler(s.cfg), s.cfg))

	// Register Azure Application Insights monitoring tool (available at all access levels)
	log.Println("Registering monitor tool: az_monitor_app_insights_query")
	appInsightsTool := monitor.RegisterAppInsightsQueryTool()
	s.mcpServer.AddTool(appInsightsTool, tools.CreateResourceHandler(monitor.GetAppInsightsHandler(s.cfg), s.cfg))

	// Register account management commands (available at all access levels)
	for _, cmd := range azaks.GetAccountAzCommands() {
		log.Println("Registering az command:", cmd.Name)
		azTool := azaks.RegisterAzCommand(cmd)
		commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}

	// Register read-write commands if access level is readwrite or admin
	if s.cfg.AccessLevel == "readwrite" || s.cfg.AccessLevel == "admin" {
		// Register read-write az commands
		for _, cmd := range azaks.GetReadWriteAzCommands() {
			log.Println("Registering az command:", cmd.Name)
			azTool := azaks.RegisterAzCommand(cmd)
			commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}

		// Register read-write az monitor commands
		for _, cmd := range monitor.GetReadWriteMonitorCommands() {
			log.Println("Registering az monitor command:", cmd.Name)
			azTool := monitor.RegisterMonitorCommand(cmd)
			commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}

		// Fleet commands are handled by the generic az fleet tool registered above
		// No additional registration needed for read-write access
	}

	// Register admin commands only if access level is admin
	if s.cfg.AccessLevel == "admin" {
		// Register admin az commands
		for _, cmd := range azaks.GetAdminAzCommands() {
			log.Println("Registering az command:", cmd.Name)
			azTool := azaks.RegisterAzCommand(cmd)
			commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}

		// Register admin az monitor commands
		for _, cmd := range monitor.GetAdminMonitorCommands() {
			log.Println("Registering az monitor command:", cmd.Name)
			azTool := monitor.RegisterMonitorCommand(cmd)
			commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}

		// Fleet commands are handled by the generic az fleet tool registered above
		// No additional registration needed for admin access
	}
}

// registerControlPlaneTools registers all AKS control plane log-related tools
func (s *Service) registerControlPlaneTools() {
	log.Println("Registering AKS Control Plane tools...")

	// Register diagnostic settings tool
	log.Println("Registering control plane tool: aks_control_plane_diagnostic_settings")
	diagnosticTool := monitor.RegisterControlPlaneDiagnosticSettingsTool()
	s.mcpServer.AddTool(diagnosticTool, tools.CreateResourceHandler(diagnostics.GetControlPlaneDiagnosticSettingsHandler(s.cfg), s.cfg))

	// Register logs querying tool
	log.Println("Registering control plane tool: aks_control_plane_logs")
	logsTool := monitor.RegisterControlPlaneLogsTool()
	s.mcpServer.AddTool(logsTool, tools.CreateResourceHandler(diagnostics.GetControlPlaneLogsHandler(s.cfg), s.cfg))
}

func (s *Service) registerAzureResourceTools() {
	// Create Azure client for the resource tools (cache is internal to the client)
	azClient, err := azureclient.NewAzureClient(s.cfg)
	if err != nil {
		log.Printf("Warning: Failed to create Azure client: %v", err)
		return
	}

	// Register Network-related tools
	s.registerNetworkTools(azClient)

	// Register Detector tools
	s.registerDetectorTools(azClient)

	// Register Compute-related tools
	s.registerComputeTools(azClient)

	// TODO: Add other resource categories in the future:
}

// registerNetworkTools registers all network-related Azure resource tools
func (s *Service) registerNetworkTools(azClient *azureclient.AzureClient) {
	log.Println("Registering Network tools...")

	// Register VNet info tool
	log.Println("Registering network tool: get_vnet_info")
	vnetTool := network.RegisterVNetInfoTool()
	s.mcpServer.AddTool(vnetTool, tools.CreateResourceHandler(network.GetVNetInfoHandler(azClient, s.cfg), s.cfg))

	// Register NSG info tool
	log.Println("Registering network tool: get_nsg_info")
	nsgTool := network.RegisterNSGInfoTool()
	s.mcpServer.AddTool(nsgTool, tools.CreateResourceHandler(network.GetNSGInfoHandler(azClient, s.cfg), s.cfg))

	// Register RouteTable info tool
	log.Println("Registering network tool: get_route_table_info")
	routeTableTool := network.RegisterRouteTableInfoTool()
	s.mcpServer.AddTool(routeTableTool, tools.CreateResourceHandler(network.GetRouteTableInfoHandler(azClient, s.cfg), s.cfg))

	// Register Subnet info tool
	log.Println("Registering network tool: get_subnet_info")
	subnetTool := network.RegisterSubnetInfoTool()
	s.mcpServer.AddTool(subnetTool, tools.CreateResourceHandler(network.GetSubnetInfoHandler(azClient, s.cfg), s.cfg))

	// Register Load Balancers info tool
	log.Println("Registering network tool: get_load_balancers_info")
	lbTool := network.RegisterLoadBalancersInfoTool()
	s.mcpServer.AddTool(lbTool, tools.CreateResourceHandler(network.GetLoadBalancersInfoHandler(azClient, s.cfg), s.cfg))

	// Register Private Endpoint info tool
	log.Println("Registering network tool: get_private_endpoint_info")
	privateEndpointTool := network.RegisterPrivateEndpointInfoTool()
	s.mcpServer.AddTool(privateEndpointTool, tools.CreateResourceHandler(network.GetPrivateEndpointInfoHandler(azClient, s.cfg), s.cfg))
}

// registerDetectorTools registers all detector-related Azure resource tools
func (s *Service) registerDetectorTools(azClient *azureclient.AzureClient) {
	log.Println("Registering Detector tools...")

	// Register list detectors tool
	log.Println("Registering detector tool: list_detectors")
	listTool := detectors.RegisterListDetectorsTool()
	s.mcpServer.AddTool(listTool, tools.CreateResourceHandler(detectors.GetListDetectorsHandler(azClient, s.cfg), s.cfg))

	// Register run detector tool
	log.Println("Registering detector tool: run_detector")
	runTool := detectors.RegisterRunDetectorTool()
	s.mcpServer.AddTool(runTool, tools.CreateResourceHandler(detectors.GetRunDetectorHandler(azClient, s.cfg), s.cfg))

	// Register run detectors by category tool
	log.Println("Registering detector tool: run_detectors_by_category")
	categoryTool := detectors.RegisterRunDetectorsByCategoryTool()
	s.mcpServer.AddTool(categoryTool, tools.CreateResourceHandler(detectors.GetRunDetectorsByCategoryHandler(azClient, s.cfg), s.cfg))
}

// registerComputeTools registers all compute-related Azure resource tools (VMSS/VM)
func (s *Service) registerComputeTools(azClient *azureclient.AzureClient) {
	log.Println("Registering Compute tools...")

	// Register AKS VMSS info tool (supports both single node pool and all node pools)
	log.Println("Registering compute tool: get_aks_vmss_info")
	vmssInfoTool := compute.RegisterAKSVMSSInfoTool()
	s.mcpServer.AddTool(vmssInfoTool, tools.CreateResourceHandler(compute.GetAKSVMSSInfoHandler(azClient, s.cfg), s.cfg))

	// Register read-only az vmss commands (available at all access levels)
	for _, cmd := range compute.GetReadOnlyVmssCommands() {
		log.Println("Registering az vmss command:", cmd.Name)
		azTool := compute.RegisterAzComputeCommand(cmd)
		commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}

	// Register read-write commands if access level is readwrite or admin
	if s.cfg.AccessLevel == "readwrite" || s.cfg.AccessLevel == "admin" {
		// Register read-write az vmss commands
		for _, cmd := range compute.GetReadWriteVmssCommands() {
			log.Println("Registering az vmss command:", cmd.Name)
			azTool := compute.RegisterAzComputeCommand(cmd)
			commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}
	}

	// Register admin commands only if access level is admin
	if s.cfg.AccessLevel == "admin" {
		// Register admin az vmss commands
		for _, cmd := range compute.GetAdminVmssCommands() {
			log.Println("Registering az vmss command:", cmd.Name)
			azTool := compute.RegisterAzComputeCommand(cmd)
			commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}
	}
}

// registerAdvisorTools registers all Azure Advisor-related tools
func (s *Service) registerAdvisorTools() {
	log.Println("Registering Advisor tools...")

	// Register Azure Advisor recommendation tool (available at all access levels)
	log.Println("Registering advisor tool: az_advisor_recommendation")
	advisorTool := advisor.RegisterAdvisorRecommendationTool()
	s.mcpServer.AddTool(advisorTool, tools.CreateResourceHandler(advisor.GetAdvisorRecommendationHandler(s.cfg), s.cfg))
}
