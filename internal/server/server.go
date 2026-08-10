package server

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/components"
	"github.com/Azure/aks-mcp/internal/components/advisor"
	"github.com/Azure/aks-mcp/internal/components/azaks"
	"github.com/Azure/aks-mcp/internal/components/azapi"
	"github.com/Azure/aks-mcp/internal/components/compute"
	"github.com/Azure/aks-mcp/internal/components/detectors"
	"github.com/Azure/aks-mcp/internal/components/fleet"
	"github.com/Azure/aks-mcp/internal/components/inspektorgadget"
	"github.com/Azure/aks-mcp/internal/components/monitor"
	"github.com/Azure/aks-mcp/internal/components/network"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/k8s"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/aks-mcp/internal/prompts"
	"github.com/Azure/aks-mcp/internal/tools"
	"github.com/Azure/aks-mcp/internal/version"
	azapimcp "github.com/Azure/azure-api-mcp/pkg/azcli"
	"github.com/Azure/mcp-kubernetes/pkg/cilium"
	"github.com/Azure/mcp-kubernetes/pkg/helm"
	"github.com/Azure/mcp-kubernetes/pkg/hubble"
	"github.com/Azure/mcp-kubernetes/pkg/kubectl"
	"github.com/mark3labs/mcp-go/server"
)

// Service represents the AKS MCP service
type Service struct {
	cfg              *config.ConfigData
	mcpServer        *server.MCPServer
	azClient         *azureclient.AzureClient
	azcliProcFactory func(timeout int) azcli.Proc
}

// ServiceOption defines a function that configures the AKS MCP service
type ServiceOption func(*Service)

// WithAzCliProcFactory allows callers to inject a Proc factory for azcli execution.
// The factory returns an azcli.Proc which can be faked in tests.
func WithAzCliProcFactory(f func(timeout int) azcli.Proc) ServiceOption {
	return func(s *Service) { s.azcliProcFactory = f }
}

// NewService creates a new AKS MCP service with the provided configuration and options.
// Options can be used to inject dependencies like azcli execution factories.
func NewService(cfg *config.ConfigData, opts ...ServiceOption) *Service {
	s := &Service{cfg: cfg}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Initialize initializes the service
func (s *Service) Initialize() error {
	logger.Infof("Initializing AKS MCP service...")

	// Phase 1: Initialize core infrastructure
	if err := s.initializeInfrastructure(); err != nil {
		return err
	}

	// Phase 2: Register all component tools
	s.registerAllComponents()

	logger.Infof("AKS MCP service initialization completed successfully")
	return nil
}

// initializeInfrastructure sets up the Azure client and MCP server
func (s *Service) initializeInfrastructure() error {
	// Create shared Azure client
	azClient, err := azureclient.NewAzureClient(s.cfg)
	if err != nil {
		return fmt.Errorf("failed to create Azure client: %w", err)
	}
	s.azClient = azClient
	logger.Infof("Azure client initialized successfully")

	// Validation has already confirmed that the Azure CLI is installed when the
	// az_cli component is enabled. A failed login is non-fatal so components
	// that do not require Azure CLI credentials remain available.
	if s.azcliProcFactory != nil {
		// Use injected factory to create an azcli.Proc
		proc := s.azcliProcFactory(s.cfg.Timeout)
		if loginType, err := azcli.EnsureAzCliLoginWithProc(proc, s.cfg); err != nil {
			logger.Warnf("Azure CLI authentication failed: %v - Azure CLI tools will fail at runtime", err)
		} else {
			logger.Infof("Azure CLI initialized successfully (%s)", loginType)
		}
	} else {
		if loginType, err := azcli.EnsureAzCliLogin(s.cfg); err != nil {
			logger.Warnf("Azure CLI authentication failed: %v - Azure CLI tools will fail at runtime", err)
		} else {
			logger.Infof("Azure CLI initialized successfully (%s)", loginType)
		}
	}

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"AKS MCP",
		version.GetVersion(),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
	)
	logger.Infof("MCP server initialized successfully")

	return nil
}

// registerAllComponents registers all component tools organized by category
func (s *Service) registerAllComponents() {
	// Log enabled components (validation is done in config validator)
	if len(s.cfg.EnabledComponents) > 0 {
		logger.Infof("Enabled components: %s", strings.Join(s.cfg.EnabledComponents, ", "))
	} else {
		logger.Infof("All components enabled by default")
	}

	// Kubernetes Components
	s.registerKubernetesComponents()

	// Azure Components
	s.registerAzureComponents()

	// Prompts
	s.registerPrompts()
}

// registerPrompts registers all available prompts
func (s *Service) registerPrompts() {
	logger.Infof("Registering Prompts...")

	logger.Debugf("Registering config prompts (query_aks_cluster_metadata_from_kubeconfig)")
	prompts.RegisterQueryAKSMetadataFromKubeconfigPrompt(s.mcpServer, s.cfg)

	logger.Debugf("Registering health prompts (check_cluster_health)")
	prompts.RegisterHealthPrompts(s.mcpServer, s.cfg)
}

// Run starts the stdio service.
func (s *Service) Run() error {
	logger.Infof("AKS MCP version: %s", version.GetVersion())

	logger.Infof("Listening for requests on STDIO...")
	return server.ServeStdio(s.mcpServer)
}

// registerAzureComponents registers all Azure tools (AKS operations, monitoring, fleet, network, compute, detectors, advisor)
func (s *Service) registerAzureComponents() {
	logger.Infof("Registering Azure Components...")

	// Azure CLI Component (unified registration)
	if components.IsComponentEnabled("az_cli", s.cfg.EnabledComponents) {
		if s.cfg.UseLegacyTools {
			logger.Infof("Registering Azure CLI Component with legacy tools (az_aks_operations, az_compute_operations)")
			s.registerAksOpsComponent()
		} else {
			logger.Infof("Registering Azure CLI Component with unified tool (call_az)")
			s.registerAzureApiComponent()
		}
	}

	// Monitoring Component
	if components.IsComponentEnabled("monitor", s.cfg.EnabledComponents) {
		s.registerMonitoringComponent()
	}

	// Fleet Management Component
	if components.IsComponentEnabled("fleet", s.cfg.EnabledComponents) {
		s.registerFleetComponent()
	}

	// Network Resources Component
	if components.IsComponentEnabled("network", s.cfg.EnabledComponents) {
		s.registerNetworkComponent()
	}

	// Compute Resources Component
	if components.IsComponentEnabled("compute", s.cfg.EnabledComponents) {
		s.registerComputeComponent()
	}

	// Detector Resources Component
	if components.IsComponentEnabled("detectors", s.cfg.EnabledComponents) {
		s.registerDetectorComponent()
	}

	// Azure Advisor Component
	if components.IsComponentEnabled("advisor", s.cfg.EnabledComponents) {
		s.registerAdvisorComponent()
	}

	// Register Inspektor Gadget tools for observability
	if components.IsComponentEnabled("inspektorgadget", s.cfg.EnabledComponents) {
		s.registerInspektorGadgetComponent()
	}

	logger.Infof("Azure Components registered successfully")
}

// registerKubernetesComponents registers Kubernetes-related tools (kubectl, helm, cilium, observability)
func (s *Service) registerKubernetesComponents() {
	logger.Infof("Registering Kubernetes Components...")

	// Core Kubernetes Component (kubectl)
	s.registerKubectlComponent()

	// Optional Kubernetes Components (based on configuration)
	s.registerOptionalKubernetesComponents()

	logger.Infof("Kubernetes Components registered successfully")
}

// registerKubectlComponent registers core kubectl commands based on access level
func (s *Service) registerKubectlComponent() {
	logger.Debugf("Registering Core Kubernetes Component (kubectl)")

	// Use UseLegacyTools to control whether to use unified call_kubectl tool or specialized tools
	useUnifiedTool := !s.cfg.UseLegacyTools
	if s.cfg.UseLegacyTools {
		logger.Debugf("Using legacy kubectl specialized tools (kubectl_resources, kubectl_workloads, etc.)")
	} else {
		logger.Debugf("Using unified kubectl tool (call_kubectl)")
	}

	// Get kubectl tools filtered by access level and tool type
	kubectlTools := k8s.RegisterKubectlTools(s.cfg.AccessLevel, useUnifiedTool)

	// Create a kubectl executor
	kubectlExecutor := kubectl.NewKubectlToolExecutor()

	wrappedExecutor := k8s.WrapK8sExecutor(kubectlExecutor)

	// Register each kubectl tool
	for _, tool := range kubectlTools {
		logger.Debugf("Registering kubectl tool: %s", tool.Name)
		// Create a handler that uses our wrapped executor
		// Use CreateToolHandlerWithName to inject the tool name for KubectlToolExecutor
		handler := tools.CreateToolHandlerWithName(wrappedExecutor, s.cfg, tool.Name)
		s.mcpServer.AddTool(tool, handler)
	}
}

// registerOptionalKubernetesComponents registers optional Kubernetes tools based on configuration
func (s *Service) registerOptionalKubernetesComponents() {
	logger.Debugf("Registering Optional Kubernetes Components")

	registeredAny := false

	// Register helm if enabled
	if components.IsComponentEnabled("helm", s.cfg.EnabledComponents) {
		s.registerHelmComponent()
		registeredAny = true
	}

	// Register cilium if enabled
	if components.IsComponentEnabled("cilium", s.cfg.EnabledComponents) {
		s.registerCiliumComponent()
		registeredAny = true
	}

	// Register hubble if enabled
	if components.IsComponentEnabled("hubble", s.cfg.EnabledComponents) {
		s.registerHubbleComponent()
		registeredAny = true
	}

	// Log if no optional components are enabled
	if !registeredAny {
		logger.Infof("No optional Kubernetes components enabled")
	}
}

// registerInspektorGadgetComponent registers Inspektor Gadget tools for observability
func (s *Service) registerInspektorGadgetComponent() {
	gadgetMgr := inspektorgadget.NewGadgetManager()

	// Register Inspektor Gadget tool
	logger.Debugf("Registering Inspektor Gadget Observability tool: inspektor_gadget_observability")
	inspektorGadget := inspektorgadget.RegisterInspektorGadgetTool()
	s.mcpServer.AddTool(inspektorGadget, tools.CreateResourceHandler(inspektorgadget.InspektorGadgetHandler(gadgetMgr, s.cfg), s.cfg))
}

// registerAksOpsComponent registers AKS operations tools
func (s *Service) registerAksOpsComponent() {
	logger.Debugf("Registering AKS operations tool: az_aks_operations")
	aksOperationsTool := azaks.RegisterAzAksOperations(s.cfg)
	s.mcpServer.AddTool(aksOperationsTool, tools.CreateToolHandler(azaks.NewAksOperationsExecutor(), s.cfg))
}

// registerMonitoringComponent registers Azure monitoring tools
func (s *Service) registerMonitoringComponent() {
	logger.Debugf("Registering monitoring tool: aks_monitoring")
	monitoringTool := monitor.RegisterAksMonitoring()
	s.mcpServer.AddTool(monitoringTool, tools.CreateResourceHandler(monitor.GetAksMonitoringHandler(s.azClient, s.cfg), s.cfg))
}

// registerFleetComponent registers Azure fleet management tools
func (s *Service) registerFleetComponent() {
	logger.Debugf("Registering fleet tool: az_fleet")
	fleetTool := fleet.RegisterFleet()
	s.mcpServer.AddTool(fleetTool, tools.CreateToolHandler(azcli.NewFleetExecutor(), s.cfg))
}

// registerAdvisorComponent registers Azure advisor tools
func (s *Service) registerAdvisorComponent() {
	logger.Debugf("Registering advisor tool: aks_advisor_recommendation")
	advisorTool := advisor.RegisterAdvisorRecommendationTool()
	s.mcpServer.AddTool(advisorTool, tools.CreateResourceHandler(advisor.GetAdvisorRecommendationHandler(s.cfg), s.cfg))
}

// registerNetworkComponent registers network-related Azure resource tools
func (s *Service) registerNetworkComponent() {
	logger.Debugf("Registering Network Resources Component")

	// Register network resources tool
	logger.Debugf("Registering network tool: aks_network_resources")
	networkTool := network.RegisterAksNetworkResources()
	s.mcpServer.AddTool(networkTool, tools.CreateResourceHandler(network.GetAksNetworkResourcesHandler(s.azClient, s.cfg), s.cfg))
}

// registerComputeComponent registers compute-related Azure resource tools (VMSS/VM)
func (s *Service) registerComputeComponent() {
	logger.Debugf("Registering Compute Resources Component")

	// Register AKS VMSS info tool (supports both single node pool and all node pools)
	logger.Debugf("Registering compute tool: get_aks_vmss_info")
	vmssInfoTool := compute.RegisterAKSVMSSInfoTool()
	s.mcpServer.AddTool(vmssInfoTool, tools.CreateResourceHandler(compute.GetAKSVMSSInfoHandler(s.azClient, s.cfg), s.cfg))

	// Register AKS node logs collection tool
	logger.Debugf("Registering compute tool: collect_aks_node_logs")
	nodeLogsTool := compute.RegisterCollectAKSNodeLogsTool()
	s.mcpServer.AddTool(nodeLogsTool, tools.CreateResourceHandler(compute.CollectAKSNodeLogsHandler(s.azClient, s.cfg), s.cfg))

	// Register unified compute operations tool (only if using legacy tools)
	if s.cfg.UseLegacyTools {
		logger.Debugf("Registering compute tool: az_compute_operations")
		computeOperationsTool := compute.RegisterAzComputeOperations(s.cfg)
		s.mcpServer.AddTool(computeOperationsTool, tools.CreateToolHandler(compute.NewComputeOperationsExecutor(), s.cfg))
	}
}

// registerDetectorComponent registers detector-related Azure resource tools
func (s *Service) registerDetectorComponent() {
	logger.Debugf("Registering Detector Resources Component")

	// Register unified detector tool
	logger.Debugf("Registering detector tool: aks_detector")
	aksDetectorTool := detectors.RegisterAksDetectorTool()
	s.mcpServer.AddTool(aksDetectorTool, tools.CreateResourceHandler(detectors.GetAksDetectorHandler(s.azClient, s.cfg), s.cfg))
}

// registerHelmComponent registers helm tools if enabled
func (s *Service) registerHelmComponent() {
	logger.Debugf("Registering Kubernetes tool: helm")
	helmTool := helm.RegisterHelm()
	helmExecutor := k8s.WrapK8sExecutor(helm.NewExecutor())
	s.mcpServer.AddTool(helmTool, tools.CreateToolHandler(helmExecutor, s.cfg))
}

// registerCiliumComponent registers cilium tools if enabled
func (s *Service) registerCiliumComponent() {
	logger.Debugf("Registering Kubernetes tool: cilium")
	ciliumTool := cilium.RegisterCilium()
	ciliumExecutor := k8s.WrapK8sExecutor(cilium.NewExecutor())
	s.mcpServer.AddTool(ciliumTool, tools.CreateToolHandler(ciliumExecutor, s.cfg))
}

// registerHubbleComponent registers hubble tools if enabled
func (s *Service) registerHubbleComponent() {
	logger.Debugf("Registering Kubernetes tool: hubble")
	hubbleTool := hubble.RegisterHubble()
	hubbleExecutor := k8s.WrapK8sExecutor(hubble.NewExecutor())
	s.mcpServer.AddTool(hubbleTool, tools.CreateToolHandler(hubbleExecutor, s.cfg))
}

// registerAzureApiComponent registers the azure-api-mcp call_az tool
func (s *Service) registerAzureApiComponent() {
	logger.Debugf("Registering azure-api-mcp tool: call_az")

	// Determine read-only mode based on access level
	readOnlyMode := s.cfg.AccessLevel == "readonly"

	// Get default subscription from environment variable
	defaultSubscription := os.Getenv("AZURE_SUBSCRIPTION_ID")

	// Create AuthConfig for azure-api-mcp
	authConfig := azapimcp.AuthConfig{
		SkipSetup:           false,
		AuthMethod:          "",
		TenantID:            os.Getenv("AZURE_TENANT_ID"),
		ClientID:            os.Getenv("AZURE_CLIENT_ID"),
		FederatedTokenFile:  os.Getenv("AZURE_FEDERATED_TOKEN_FILE"),
		ClientSecret:        os.Getenv("AZURE_CLIENT_SECRET"),
		DefaultSubscription: defaultSubscription,
	}

	// Create AuthSetup for re-authentication on auth errors
	authSetup := azapimcp.NewDefaultAuthSetup(authConfig)

	// Create azure-api-mcp client
	clientConfig := azapimcp.ClientConfig{
		ReadOnlyMode:         readOnlyMode,
		EnableSecurityPolicy: false,
		Timeout:              time.Duration(s.cfg.Timeout) * time.Second,
		WorkingDir:           "",
		SecurityPolicyFile:   "",
		ReadOnlyPatternsFile: "",
		AuthSetup:            authSetup,
	}

	azClient, err := azapimcp.NewClient(clientConfig)
	if err != nil {
		logger.Errorf("Failed to create azure-api-mcp client: %v", err)
		os.Exit(1)
	}

	// Register the tool using azure-api-mcp's registry
	callAzTool := azapimcp.RegisterCallAzTool(readOnlyMode, defaultSubscription)

	// Create handler using our wrapper
	handler := azapi.AzApiHandler(azClient, s.cfg)

	// Register with MCP server
	s.mcpServer.AddTool(callAzTool, handler)
}
