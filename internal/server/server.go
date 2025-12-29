package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/auth/oauth"
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
	oauthProvider    *oauth.AzureOAuthProvider
	authMiddleware   *oauth.AuthMiddleware
	endpointManager  *oauth.EndpointManager
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

	// Initialize OAuth components if enabled and transport is not stdio
	// OAuth is not supported with stdio transport per MCP specification
	if s.cfg.OAuthConfig.Enabled && s.cfg.Transport != "stdio" {
		if err := s.initializeOAuth(); err != nil {
			return fmt.Errorf("failed to initialize OAuth: %w", err)
		}
	}

	// Ensure Azure CLI exists and is logged in
	// Allow service to start even if az CLI is not available or authentication fails
	// Tools that require az will fail at runtime with appropriate error messages
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

// initializeOAuth initializes OAuth authentication components
func (s *Service) initializeOAuth() error {
	logger.Infof("Initializing OAuth authentication...")

	// Validate OAuth configuration
	if err := s.cfg.OAuthConfig.Validate(); err != nil {
		return fmt.Errorf("invalid OAuth configuration: %w", err)
	}

	// Create OAuth provider
	provider, err := oauth.NewAzureOAuthProvider(s.cfg.OAuthConfig)
	if err != nil {
		return fmt.Errorf("failed to create OAuth provider: %w", err)
	}
	s.oauthProvider = provider

	// Create server URL for OAuth metadata
	serverURL := fmt.Sprintf("http://%s:%d", s.cfg.Host, s.cfg.Port)

	// Create auth middleware
	s.authMiddleware = oauth.NewAuthMiddleware(provider, serverURL)

	// Create endpoint manager
	s.endpointManager = oauth.NewEndpointManager(provider, s.cfg)

	logger.Infof("OAuth authentication initialized with tenant: %s", s.cfg.OAuthConfig.TenantID)
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

	// Azure Components
	s.registerAzureComponents()

	// Kubernetes Components
	s.registerKubernetesComponents()

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

// healthHandler provides a simple health check endpoint
func (s *Service) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		response := map[string]interface{}{
			"status":    "healthy",
			"version":   version.GetVersion(),
			"transport": s.cfg.Transport,
			"oauth": map[string]interface{}{
				"enabled": s.cfg.OAuthConfig.Enabled,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// createCustomHTTPServerWithHelp404 creates a custom HTTP server that provides
// helpful 404 responses for the MCP server
func (s *Service) createCustomHTTPServerWithHelp404(addr string) *http.Server {
	mux := http.NewServeMux()

	// Register health check endpoint (always available)
	mux.HandleFunc("/health", s.healthHandler())

	// Register OAuth endpoints if OAuth is enabled
	if s.cfg.OAuthConfig.Enabled {
		if s.endpointManager == nil {
			logger.Errorf("OAuth is enabled but endpoint manager is not initialized - this indicates a bug in server initialization")
		}
		logger.Infof("Registering OAuth endpoints...")
		s.endpointManager.RegisterEndpoints(mux)
	}

	// Handle all other paths with a helpful 404 response
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" && r.URL.Path != "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)

			response := map[string]interface{}{
				"error":   "Not Found",
				"message": "This is an MCP (Model Context Protocol) server. Please send POST requests to /mcp to initialize a session and obtain an Mcp-Session-Id for subsequent requests.",
				"endpoints": map[string]string{
					"initialize": "POST /mcp - Initialize MCP session",
					"requests":   "POST /mcp - Send MCP requests (requires Mcp-Session-Id header)",
					"listen":     "GET /mcp - Listen for notifications (requires Mcp-Session-Id header)",
					"terminate":  "DELETE /mcp - Terminate session (requires Mcp-Session-Id header)",
					"health":     "GET /health - Health check",
				},
			}

			// Add OAuth endpoints to the response if enabled
			if s.cfg.OAuthConfig.Enabled {
				oauthEndpoints := map[string]string{
					"oauth-metadata":       "GET /.well-known/oauth-protected-resource - OAuth metadata",
					"auth-server-metadata": "GET /.well-known/oauth-authorization-server - Authorization server metadata",
					"client-registration":  "POST /oauth/register - Dynamic client registration",
					"token-introspection":  "POST /oauth/introspect - Token introspection",
				}
				for k, v := range oauthEndpoints {
					response["endpoints"].(map[string]string)[k] = v
				}
			}

			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
		}
	})

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// createCustomSSEServerWithHelp404 creates a custom HTTP server for SSE that provides
// helpful 404 responses for non-MCP endpoints
func (s *Service) createCustomSSEServerWithHelp404(sseServer *server.SSEServer, addr string) *http.Server {
	mux := http.NewServeMux()

	// Register health check endpoint (always available)
	mux.HandleFunc("/health", s.healthHandler())

	// Register OAuth endpoints if OAuth is enabled
	if s.cfg.OAuthConfig.Enabled {
		if s.endpointManager == nil {
			logger.Errorf("OAuth is enabled but endpoint manager is not initialized - this indicates a bug in server initialization")
		}
		logger.Infof("Registering OAuth endpoints for SSE server...")
		s.endpointManager.RegisterEndpoints(mux)
	}

	// Register SSE and Message handlers with authentication if enabled
	if s.cfg.OAuthConfig.Enabled {
		if s.authMiddleware == nil {
			logger.Errorf("OAuth is enabled but auth middleware is not initialized - this indicates a bug in server initialization")
		}
		// Apply authentication middleware to SSE and Message endpoints
		mux.Handle("/sse", s.authMiddleware.Middleware(sseServer.SSEHandler()))
		mux.Handle("/message", s.authMiddleware.Middleware(sseServer.MessageHandler()))
	} else {
		// Register without authentication
		mux.Handle("/sse", sseServer.SSEHandler())
		mux.Handle("/message", sseServer.MessageHandler())
	}

	// Handle all other paths with a helpful 404 response
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sse" && r.URL.Path != "/message" && r.URL.Path != "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)

			response := map[string]interface{}{
				"error":   "Not Found",
				"message": "This is an MCP (Model Context Protocol) server using SSE transport. Use the SSE endpoint to establish connections and the message endpoint to send requests.",
				"endpoints": map[string]string{
					"sse":     "GET /sse - Establish SSE connection for real-time notifications",
					"message": "POST /message - Send MCP JSON-RPC messages",
					"health":  "GET /health - Health check",
				},
			}

			// Add OAuth endpoints and authentication info if enabled
			if s.cfg.OAuthConfig.Enabled {
				response["authentication"] = map[string]interface{}{
					"required": true,
					"type":     "Bearer",
					"note":     "Include 'Authorization: Bearer <token>' header for authenticated endpoints",
				}

				oauthEndpoints := map[string]string{
					"oauth-metadata":       "GET /.well-known/oauth-protected-resource - OAuth metadata",
					"auth-server-metadata": "GET /.well-known/oauth-authorization-server - Authorization server metadata",
					"client-registration":  "POST /oauth/register - Dynamic client registration",
					"token-introspection":  "POST /oauth/introspect - Token introspection",
				}
				for k, v := range oauthEndpoints {
					response["endpoints"].(map[string]string)[k] = v
				}
			}

			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
		}
	})

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// Run starts the service with the specified transport
func (s *Service) Run() error {
	logger.Infof("AKS MCP version: %s", version.GetVersion())

	// Start the server
	switch s.cfg.Transport {
	case "stdio":
		logger.Infof("Listening for requests on STDIO...")
		return server.ServeStdio(s.mcpServer)
	case "sse":
		addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

		// Create SSE server
		sse := server.NewSSEServer(
			s.mcpServer,
		)

		// Create custom HTTP server with helpful 404 responses
		customServer := s.createCustomSSEServerWithHelp404(sse, addr)

		logger.Infof("SSE server listening on %s", addr)
		logger.Infof("SSE endpoint available at: http://%s/sse", addr)
		logger.Infof("Message endpoint available at: http://%s/message", addr)
		logger.Infof("Connect to /sse for real-time events, send JSON-RPC to /message")
		if s.cfg.OAuthConfig.Enabled {
			logger.Infof("OAuth authentication enabled - Bearer token required for SSE and Message endpoints")
			logger.Infof("OAuth metadata available at: http://%s/.well-known/oauth-protected-resource", addr)
		}

		return customServer.ListenAndServe()
	case "streamable-http":
		addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

		// Create a custom HTTP server with helpful 404 responses
		customServer := s.createCustomHTTPServerWithHelp404(addr)

		// Create the streamable HTTP server with the custom HTTP server
		streamableServer := server.NewStreamableHTTPServer(
			s.mcpServer,
			server.WithStreamableHTTPServer(customServer),
		)

		// Update the mux to use the actual streamable server as the MCP handler
		if mux, ok := customServer.Handler.(*http.ServeMux); ok {
			if s.cfg.OAuthConfig.Enabled {
				if s.authMiddleware == nil {
					logger.Errorf("OAuth is enabled but auth middleware is not initialized - this indicates a bug in server initialization")
				}
				// Apply authentication middleware to MCP endpoint
				mux.Handle("/mcp", s.authMiddleware.Middleware(streamableServer))
			} else {
				// Register without authentication
				mux.Handle("/mcp", streamableServer)
			}
		}

		logger.Infof("Streamable HTTP server listening on %s", addr)
		logger.Infof("MCP endpoint available at: http://%s/mcp", addr)
		logger.Infof("Send POST requests to /mcp to initialize session and obtain Mcp-Session-Id")
		if s.cfg.OAuthConfig.Enabled {
			logger.Infof("OAuth authentication enabled - Bearer token required for MCP endpoint")
			logger.Infof("OAuth metadata available at: http://%s/.well-known/oauth-protected-resource", addr)
		}

		return customServer.ListenAndServe()
	default:
		return fmt.Errorf("invalid transport type: %s (must be 'stdio', 'sse' or 'streamable-http')", s.cfg.Transport)
	}
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
	kubectlTools := kubectl.RegisterKubectlTools(s.cfg.AccessLevel, useUnifiedTool)

	// Create a kubectl executor
	kubectlExecutor := kubectl.NewKubectlToolExecutor()

	// Wrap the executor with multi-cluster support if enabled
	wrappedExecutor := k8s.WrapK8sExecutor(kubectlExecutor, s.cfg.EnableMultiCluster)
	if s.cfg.EnableMultiCluster {
		logger.Infof("Multi-cluster mode enabled: kubectl commands will use Azure AKS RunCommand API")
	}

	// Register each kubectl tool
	for _, tool := range kubectlTools {
		logger.Debugf("Registering kubectl tool: %s", tool.Name)
		// Create a handler that uses our wrapped executor
		handler := tools.CreateToolHandler(wrappedExecutor, s.cfg)
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
	logger.Debugf("Registering monitoring tool: az_monitoring")
	monitoringTool := monitor.RegisterAzMonitoring()
	s.mcpServer.AddTool(monitoringTool, tools.CreateResourceHandler(monitor.GetAzMonitoringHandler(s.azClient, s.cfg), s.cfg))
}

// registerFleetComponent registers Azure fleet management tools
func (s *Service) registerFleetComponent() {
	logger.Debugf("Registering fleet tool: az_fleet")
	fleetTool := fleet.RegisterFleet()
	s.mcpServer.AddTool(fleetTool, tools.CreateToolHandler(azcli.NewFleetExecutor(), s.cfg))
}

// registerAdvisorComponent registers Azure advisor tools
func (s *Service) registerAdvisorComponent() {
	logger.Debugf("Registering advisor tool: az_advisor_recommendation")
	advisorTool := advisor.RegisterAdvisorRecommendationTool()
	s.mcpServer.AddTool(advisorTool, tools.CreateResourceHandler(advisor.GetAdvisorRecommendationHandler(s.cfg), s.cfg))
}

// registerNetworkComponent registers network-related Azure resource tools
func (s *Service) registerNetworkComponent() {
	logger.Debugf("Registering Network Resources Component")

	// Register network resources tool
	logger.Debugf("Registering network tool: az_network_resources")
	networkTool := network.RegisterAzNetworkResources()
	s.mcpServer.AddTool(networkTool, tools.CreateResourceHandler(network.GetAzNetworkResourcesHandler(s.azClient, s.cfg), s.cfg))
}

// registerComputeComponent registers compute-related Azure resource tools (VMSS/VM)
func (s *Service) registerComputeComponent() {
	logger.Debugf("Registering Compute Resources Component")

	// Register AKS VMSS info tool (supports both single node pool and all node pools)
	logger.Debugf("Registering compute tool: get_aks_vmss_info")
	vmssInfoTool := compute.RegisterAKSVMSSInfoTool()
	s.mcpServer.AddTool(vmssInfoTool, tools.CreateResourceHandler(compute.GetAKSVMSSInfoHandler(s.azClient, s.cfg), s.cfg))

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

	// Register list detectors tool
	logger.Debugf("Registering detector tool: list_detectors")
	listTool := detectors.RegisterListDetectorsTool()
	s.mcpServer.AddTool(listTool, tools.CreateResourceHandler(detectors.GetListDetectorsHandler(s.azClient, s.cfg), s.cfg))

	// Register run detector tool
	logger.Debugf("Registering detector tool: run_detector")
	runTool := detectors.RegisterRunDetectorTool()
	s.mcpServer.AddTool(runTool, tools.CreateResourceHandler(detectors.GetRunDetectorHandler(s.azClient, s.cfg), s.cfg))

	// Register run detectors by category tool
	logger.Debugf("Registering detector tool: run_detectors_by_category")
	categoryTool := detectors.RegisterRunDetectorsByCategoryTool()
	s.mcpServer.AddTool(categoryTool, tools.CreateResourceHandler(detectors.GetRunDetectorsByCategoryHandler(s.azClient, s.cfg), s.cfg))
}

// registerHelmComponent registers helm tools if enabled
func (s *Service) registerHelmComponent() {
	logger.Debugf("Registering Kubernetes tool: helm")
	helmTool := helm.RegisterHelm()
	helmExecutor := k8s.WrapK8sExecutor(helm.NewExecutor(), s.cfg.EnableMultiCluster)
	s.mcpServer.AddTool(helmTool, tools.CreateToolHandler(helmExecutor, s.cfg))
}

// registerCiliumComponent registers cilium tools if enabled
func (s *Service) registerCiliumComponent() {
	logger.Debugf("Registering Kubernetes tool: cilium")
	ciliumTool := cilium.RegisterCilium()
	ciliumExecutor := k8s.WrapK8sExecutor(cilium.NewExecutor(), s.cfg.EnableMultiCluster)
	s.mcpServer.AddTool(ciliumTool, tools.CreateToolHandler(ciliumExecutor, s.cfg))
}

// registerHubbleComponent registers hubble tools if enabled
func (s *Service) registerHubbleComponent() {
	logger.Debugf("Registering Kubernetes tool: hubble")
	hubbleTool := hubble.RegisterHubble()
	hubbleExecutor := k8s.WrapK8sExecutor(hubble.NewExecutor(), s.cfg.EnableMultiCluster)
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
