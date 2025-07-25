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
	"github.com/Azure/aks-mcp/internal/components/network"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/k8s"
	"github.com/Azure/aks-mcp/internal/tools"
	"github.com/Azure/aks-mcp/internal/version"
	"github.com/Azure/mcp-kubernetes/pkg/cilium"
	"github.com/Azure/mcp-kubernetes/pkg/helm"
	"github.com/Azure/mcp-kubernetes/pkg/kubectl"
	k8stools "github.com/Azure/mcp-kubernetes/pkg/tools"
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

	// Register individual az commands
	s.registerAzCommands()

	// Register Azure resource tools (VNet, NSG, etc.)
	s.registerAzureResourceTools()

	// Register Azure Advisor tools
	s.registerAdvisorTools()

	// Register Kubernetes tools
	s.registerKubernetesTools()

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

// registerAzCommands registers AKS operations tool
func (s *Service) registerAzCommands() {
	// Register AKS operations tool
	log.Println("Registering tool: az_aks_operations")
	aksOperationsTool := azaks.RegisterAzAksOperations(s.cfg)
	s.mcpServer.AddTool(aksOperationsTool, tools.CreateToolHandler(azaks.NewAksOperationsExecutor(), s.cfg))

	// Register monitoring tool
	log.Println("Registering tool: az_monitoring")
	monitoringTool := monitor.RegisterAzMonitoring()
	s.mcpServer.AddTool(monitoringTool, tools.CreateResourceHandler(monitor.GetAzMonitoringHandler(s.cfg), s.cfg))

	// Register generic az fleet tool with structured parameters (available at all access levels)
	log.Println("Registering az fleet tool: az_fleet")
	fleetTool := fleet.RegisterFleet()
	s.mcpServer.AddTool(fleetTool, tools.CreateToolHandler(azcli.NewFleetExecutor(), s.cfg))

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

// registerNetworkTools registers the network resources tool
func (s *Service) registerNetworkTools(azClient *azureclient.AzureClient) {
	log.Println("Registering Network tool...")

	// Register network resources tool
	log.Println("Registering network tool: az_network_resources")
	networkTool := network.RegisterAzNetworkResources()
	s.mcpServer.AddTool(networkTool, tools.CreateResourceHandler(network.GetAzNetworkResourcesHandler(azClient, s.cfg), s.cfg))

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

// registerKubernetesTools registers Kubernetes-related tools (kubectl, helm, cilium)
func (s *Service) registerKubernetesTools() {
	log.Println("Registering Kubernetes tools...")

	// Register kubectl commands based on access level
	s.registerKubectlCommands()

	// Register helm if enabled
	if s.cfg.AdditionalTools["helm"] {
		log.Println("Registering Kubernetes tool: helm")
		helmTool := helm.RegisterHelm()
		helmExecutor := k8s.WrapK8sExecutor(helm.NewExecutor())
		s.mcpServer.AddTool(helmTool, tools.CreateToolHandler(helmExecutor, s.cfg))
	}

	// Register cilium if enabled
	if s.cfg.AdditionalTools["cilium"] {
		log.Println("Registering Kubernetes tool: cilium")
		ciliumTool := cilium.RegisterCilium()
		ciliumExecutor := k8s.WrapK8sExecutor(cilium.NewExecutor())
		s.mcpServer.AddTool(ciliumTool, tools.CreateToolHandler(ciliumExecutor, s.cfg))
	}
}

// registerKubectlCommands registers kubectl commands based on access level
func (s *Service) registerKubectlCommands() {
	// Get kubectl tools filtered by access level
	kubectlTools := kubectl.RegisterKubectlTools(s.cfg.AccessLevel)

	// Create a kubectl executor
	kubectlExecutor := kubectl.NewKubectlToolExecutor()

	// Convert aks-mcp config to k8s config
	k8sCfg := k8s.ConvertConfig(s.cfg)

	// Register each kubectl tool
	for _, tool := range kubectlTools {
		log.Printf("Registering kubectl tool: %s", tool.Name)
		// Create a handler that injects the tool name into params
		handler := k8stools.CreateToolHandlerWithName(kubectlExecutor, k8sCfg, tool.Name)
		s.mcpServer.AddTool(tool, handler)
	}
}
