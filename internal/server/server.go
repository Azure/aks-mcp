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
	"github.com/Azure/aks-mcp/internal/components/inspektorgadget"
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

// Service represents the AKS MCP service
type Service struct {
	cfg       *config.ConfigData
	mcpServer *server.MCPServer
	azClient  *azureclient.AzureClient
}

// NewService creates a new AKS MCP service
func NewService(cfg *config.ConfigData) *Service {
	return &Service{
		cfg: cfg,
	}
}

// Initialize initializes the service
func (s *Service) Initialize() error {
	log.Println("Initializing AKS MCP service...")

	// Phase 1: Initialize core infrastructure
	if err := s.initializeInfrastructure(); err != nil {
		return err
	}

	// Phase 2: Register all component tools
	s.registerAllComponents()

	log.Println("AKS MCP service initialization completed successfully")
	return nil
}

// initializeInfrastructure sets up the Azure client and MCP server
func (s *Service) initializeInfrastructure() error {
	// Create shared Azure client
	azClient, err := azureclient.NewAzureClient(s.cfg)
	if err != nil {
		return fmt.Errorf("failed to create Azure client: %v", err)
	}
	s.azClient = azClient
	log.Println("Azure client initialized successfully")

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"AKS MCP",
		version.GetVersion(),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)
	log.Println("MCP server initialized successfully")

	return nil
}

// registerAllComponents registers all component tools organized by category
func (s *Service) registerAllComponents() {
	// Azure Components
	s.registerAzureComponents()

	// Kubernetes Components
	s.registerKubernetesComponents()
}

// Run starts the service with the specified transport
func (s *Service) Run() error {
	log.Println("AKS MCP version:", version.GetVersion())

	// Start the server
	switch s.cfg.Transport {
	case "stdio":
		log.Println("AKS MCP version:", version.GetVersion())
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

// registerAzureComponents registers all Azure tools (AKS operations, monitoring, fleet, network, compute, detectors, advisor)
func (s *Service) registerAzureComponents() {
	log.Println("Registering Azure Components...")

	// AKS Operations Component
	s.registerAksOpsComponent()

	// Monitoring Component
	s.registerMonitoringComponent()

	// Fleet Management Component
	s.registerFleetComponent()

	// Network Resources Component
	s.registerNetworkComponent()

	// Compute Resources Component
	s.registerComputeComponent()

	// Detector Resources Component
	s.registerDetectorComponent()

	// Azure Advisor Component
	s.registerAdvisorComponent()

	// Register Inspektor Gadget tools for observability
	s.registerInspektorGadgetComponent()

	log.Println("Azure Components registered successfully")
}

// registerKubernetesComponents registers Kubernetes-related tools (kubectl, helm, cilium, observability)
func (s *Service) registerKubernetesComponents() {
	log.Println("Registering Kubernetes Components...")

	// Core Kubernetes Component (kubectl)
	s.registerKubectlComponent()

	// Optional Kubernetes Components (based on configuration)
	s.registerOptionalKubernetesComponents()

	log.Println("Kubernetes Components registered successfully")
}

// registerKubectlComponent registers core kubectl commands based on access level
func (s *Service) registerKubectlComponent() {
	log.Println("Registering Core Kubernetes Component (kubectl)")

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

// registerOptionalKubernetesComponents registers optional Kubernetes tools based on configuration
func (s *Service) registerOptionalKubernetesComponents() {
	log.Println("Registering Optional Kubernetes Components")

	// Register helm if enabled
	s.registerHelmComponent()

	// Register cilium if enabled
	s.registerCiliumComponent()

	// Log if no optional components are enabled
	if !s.cfg.AdditionalTools["helm"] && !s.cfg.AdditionalTools["cilium"] {
		log.Println("No optional Kubernetes components enabled")
	}
}

// registerInspektorGadgetComponent registers Inspektor Gadget tools for observability
func (s *Service) registerInspektorGadgetComponent() {
	gadgetMgr := inspektorgadget.NewGadgetManager()

	// Register Inspektor Gadget tool
	log.Println("Registering Inspektor Gadget Observability tool: inspektor_gadget_observability")
	inspektorGadget := inspektorgadget.RegisterInspektorGadgetTool()
	s.mcpServer.AddTool(inspektorGadget, tools.CreateResourceHandler(inspektorgadget.InspektorGadgetHandler(gadgetMgr, s.cfg), s.cfg))
}

// registerAksOpsComponent registers AKS operations tools
func (s *Service) registerAksOpsComponent() {
	log.Println("Registering AKS operations tool: az_aks_operations")
	aksOperationsTool := azaks.RegisterAzAksOperations(s.cfg)
	s.mcpServer.AddTool(aksOperationsTool, tools.CreateToolHandler(azaks.NewAksOperationsExecutor(), s.cfg))
}

// registerMonitoringComponent registers Azure monitoring tools
func (s *Service) registerMonitoringComponent() {
	log.Println("Registering monitoring tool: az_monitoring")
	monitoringTool := monitor.RegisterAzMonitoring()
	s.mcpServer.AddTool(monitoringTool, tools.CreateResourceHandler(monitor.GetAzMonitoringHandler(s.azClient, s.cfg), s.cfg))
}

// registerFleetComponent registers Azure fleet management tools
func (s *Service) registerFleetComponent() {
	log.Println("Registering fleet tool: az_fleet")
	fleetTool := fleet.RegisterFleet()
	s.mcpServer.AddTool(fleetTool, tools.CreateToolHandler(azcli.NewFleetExecutor(), s.cfg))
}

// registerAdvisorComponent registers Azure advisor tools
func (s *Service) registerAdvisorComponent() {
	log.Println("Registering advisor tool: az_advisor_recommendation")
	advisorTool := advisor.RegisterAdvisorRecommendationTool()
	s.mcpServer.AddTool(advisorTool, tools.CreateResourceHandler(advisor.GetAdvisorRecommendationHandler(s.cfg), s.cfg))
}

// registerNetworkComponent registers network-related Azure resource tools
func (s *Service) registerNetworkComponent() {
	log.Println("Registering Network Resources Component")

	// Register network resources tool
	log.Println("Registering network tool: az_network_resources")
	networkTool := network.RegisterAzNetworkResources()
	s.mcpServer.AddTool(networkTool, tools.CreateResourceHandler(network.GetAzNetworkResourcesHandler(s.azClient, s.cfg), s.cfg))
}

// registerComputeComponent registers compute-related Azure resource tools (VMSS/VM)
func (s *Service) registerComputeComponent() {
	log.Println("Registering Compute Resources Component")

	// Register AKS VMSS info tool (supports both single node pool and all node pools)
	log.Println("Registering compute tool: get_aks_vmss_info")
	vmssInfoTool := compute.RegisterAKSVMSSInfoTool()
	s.mcpServer.AddTool(vmssInfoTool, tools.CreateResourceHandler(compute.GetAKSVMSSInfoHandler(s.azClient, s.cfg), s.cfg))

	// Register read-only az vmss commands (available at all access levels)
	for _, cmd := range compute.GetReadOnlyVmssCommands() {
		log.Printf("Registering az vmss command: %s (readonly)", cmd.Name)
		azTool := compute.RegisterAzComputeCommand(cmd)
		commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
		s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
	}

	// Register read-write commands if access level is readwrite or admin
	if s.cfg.AccessLevel == "readwrite" || s.cfg.AccessLevel == "admin" {
		for _, cmd := range compute.GetReadWriteVmssCommands() {
			log.Printf("Registering az vmss command: %s (readwrite)", cmd.Name)
			azTool := compute.RegisterAzComputeCommand(cmd)
			commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}
	}

	// Register admin commands only if access level is admin
	if s.cfg.AccessLevel == "admin" {
		for _, cmd := range compute.GetAdminVmssCommands() {
			log.Printf("Registering az vmss command: %s (admin)", cmd.Name)
			azTool := compute.RegisterAzComputeCommand(cmd)
			commandExecutor := azcli.CreateCommandExecutorFunc(cmd.Name)
			s.mcpServer.AddTool(azTool, tools.CreateToolHandler(commandExecutor, s.cfg))
		}
	}
}

// registerDetectorComponent registers detector-related Azure resource tools
func (s *Service) registerDetectorComponent() {
	log.Println("Registering Detector Resources Component")

	// Register list detectors tool
	log.Println("Registering detector tool: list_detectors")
	listTool := detectors.RegisterListDetectorsTool()
	s.mcpServer.AddTool(listTool, tools.CreateResourceHandler(detectors.GetListDetectorsHandler(s.azClient, s.cfg), s.cfg))

	// Register run detector tool
	log.Println("Registering detector tool: run_detector")
	runTool := detectors.RegisterRunDetectorTool()
	s.mcpServer.AddTool(runTool, tools.CreateResourceHandler(detectors.GetRunDetectorHandler(s.azClient, s.cfg), s.cfg))

	// Register run detectors by category tool
	log.Println("Registering detector tool: run_detectors_by_category")
	categoryTool := detectors.RegisterRunDetectorsByCategoryTool()
	s.mcpServer.AddTool(categoryTool, tools.CreateResourceHandler(detectors.GetRunDetectorsByCategoryHandler(s.azClient, s.cfg), s.cfg))
}

// registerHelmComponent registers helm tools if enabled
func (s *Service) registerHelmComponent() {
	if s.cfg.AdditionalTools["helm"] {
		log.Println("Registering Kubernetes tool: helm")
		helmTool := helm.RegisterHelm()
		helmExecutor := k8s.WrapK8sExecutor(helm.NewExecutor())
		s.mcpServer.AddTool(helmTool, tools.CreateToolHandler(helmExecutor, s.cfg))
	}
}

// registerCiliumComponent registers cilium tools if enabled
func (s *Service) registerCiliumComponent() {
	if s.cfg.AdditionalTools["cilium"] {
		log.Println("Registering Kubernetes tool: cilium")
		ciliumTool := cilium.RegisterCilium()
		ciliumExecutor := k8s.WrapK8sExecutor(cilium.NewExecutor())
		s.mcpServer.AddTool(ciliumTool, tools.CreateToolHandler(ciliumExecutor, s.cfg))
	}
}
