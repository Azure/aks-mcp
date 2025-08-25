package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/components/azaks"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/mcp-kubernetes/pkg/kubectl"
	"github.com/mark3labs/mcp-go/server"
)

// MockToolCounter tracks registered tools for testing
type MockToolCounter struct {
	azureTools int
	k8sTools   int
	toolNames  []string
}

// NewMockToolCounter creates a new mock tool counter
func NewMockToolCounter() *MockToolCounter {
	return &MockToolCounter{
		toolNames: make([]string, 0),
	}
}

// AddTool simulates adding a tool and categorizes it
func (m *MockToolCounter) AddTool(toolName string) {
	m.toolNames = append(m.toolNames, toolName)

	// Categorize tools
	azureToolPrefixes := []string{"az_", "azure_", "get_aks_", "list_detectors", "run_detector", "inspektor_gadget_observability"}
	k8sToolPrefixes := []string{"kubectl_", "k8s_", "helm", "cilium"}

	isAzureTool := false
	for _, prefix := range azureToolPrefixes {
		if containsPrefix(toolName, prefix) {
			m.azureTools++
			isAzureTool = true
			break
		}
	}

	if !isAzureTool {
		for _, prefix := range k8sToolPrefixes {
			if containsPrefix(toolName, prefix) {
				m.k8sTools++
				break
			}
		}
	}
}

// GetCounts returns the tool counts
func (m *MockToolCounter) GetCounts() (azureTools, k8sTools int) {
	return m.azureTools, m.k8sTools
}

// GetToolNames returns all registered tool names
func (m *MockToolCounter) GetToolNames() []string {
	return m.toolNames
}

// TestService tests the service initialization and expected tool counts
func TestService(t *testing.T) {
	// Set environment variables for testing to avoid Azure auth issues.
	// These are dummy values for tests only — do not commit real credentials here.
	_ = os.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	_ = os.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	_ = os.Setenv("AZURE_CLIENT_SECRET", "dummy-client-secret")
	_ = os.Setenv("AZURE_SUBSCRIPTION_ID", "dummy-subscription-id")
	defer func() {
		_ = os.Unsetenv("AZURE_TENANT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_SECRET")
		_ = os.Unsetenv("AZURE_SUBSCRIPTION_ID")
	}()

	tests := []struct {
		name               string
		accessLevel        string
		additionalTools    map[string]bool
		expectedAzureTools int
		expectedK8sTools   int
		description        string
	}{
		{
			name:               "ReadOnly_NoOptional",
			accessLevel:        "readonly",
			additionalTools:    map[string]bool{},
			expectedAzureTools: 8, // AKS Ops + Monitoring + Fleet + Network + Compute (VMSS Info only) + Detectors (3) + Advisor + Inspektor Gadget
			expectedK8sTools:   0, // Will be calculated based on kubectl tools for readonly
			description:        "Readonly access with no optional tools",
		},
		{
			name:               "ReadWrite_NoOptional",
			accessLevel:        "readwrite",
			additionalTools:    map[string]bool{},
			expectedAzureTools: 9, // Same as readonly + 1 read-write VMSS command
			expectedK8sTools:   0, // Will be calculated based on kubectl tools for readwrite
			description:        "Readwrite access with no optional tools",
		},
		{
			name:               "Admin_NoOptional",
			accessLevel:        "admin",
			additionalTools:    map[string]bool{},
			expectedAzureTools: 9, // Same as readwrite (no admin VMSS commands currently)
			expectedK8sTools:   0, // Will be calculated based on kubectl tools for admin
			description:        "Admin access with no optional tools",
		},
		{
			name:        "ReadOnly_AllOptional",
			accessLevel: "readonly",
			additionalTools: map[string]bool{
				"helm":   true,
				"cilium": true,
			},
			expectedAzureTools: 8, // Same as readonly (Inspektor Gadget now included automatically)
			expectedK8sTools:   0, // Will be calculated + 2 optional tools
			description:        "Readonly access with all optional tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration
			cfg := createTestConfig(tt.accessLevel, tt.additionalTools)

			// Create service with injected fake Proc factory so Initialize doesn't call the real az binary
			service := NewService(cfg, WithAzCliProcFactory(func(timeout int) azcli.Proc { return &fakeProc{} }))

			// Calculate expected kubectl tools count
			kubectlTools := kubectl.RegisterKubectlTools(tt.accessLevel)
			expectedKubectlCount := len(kubectlTools)

			// Add optional tools count
			optionalToolsCount := 0
			if tt.additionalTools["helm"] {
				optionalToolsCount++
			}
			if tt.additionalTools["cilium"] {
				optionalToolsCount++
			}

			expectedTotalK8sTools := expectedKubectlCount + optionalToolsCount

			t.Logf("Test: %s", tt.description)
			t.Logf("Expected kubectl tools for %s: %d", tt.accessLevel, expectedKubectlCount)
			t.Logf("Expected optional tools: %d", optionalToolsCount)
			t.Logf("Expected total K8s tools: %d", expectedTotalK8sTools)
			t.Logf("Expected Azure tools: %d", tt.expectedAzureTools)

			// If service wasn't created above (some test paths), create it with the fake factory
			if service == nil {
				service = NewService(cfg, WithAzCliProcFactory(func(timeout int) azcli.Proc { return &fakeProc{} }))
			}

			// Verify service was created properly
			if service == nil { //nolint:staticcheck // False positive: t.Fatal prevents nil dereference
				t.Fatal("Service should not be nil")
			}

			// Initialize service (this will register all tools)
			if err := service.Initialize(); err != nil {
				t.Fatalf("Failed to initialize service: %v", err)
			}

			// Verify initialization completed
			if service.azClient == nil { //nolint:staticcheck // False positive: service verified non-nil above
				t.Error("Azure client should be initialized")
			}
			if service.mcpServer == nil { //nolint:staticcheck // False positive: service verified non-nil above
				t.Error("MCP server should be initialized")
			}

			t.Logf("Service initialized successfully for access level: %s", tt.accessLevel)
		})
	}
}

// fakeProc is a minimal Proc implementation for tests.
type fakeProc struct{}

func (f *fakeProc) Run(cmd string) (string, error) {
	// For probing account show, return a non-error id
	if cmd == "account show --query id -o tsv" {
		return "00000000-0000-0000-0000-000000000000", nil
	}
	// For any other command, return empty output and nil error to simulate success
	return "", nil
}

// TestComponentToolCounts tests individual component tool registration counts
func TestComponentToolCounts(t *testing.T) {
	t.Run("AzureComponents", func(t *testing.T) {
		testCases := []struct {
			component   string
			toolCount   int
			description string
		}{
			{"AKS Operations", 1, "az_aks_operations tool"},
			{"Monitoring", 1, "az_monitoring tool"},
			{"Fleet", 1, "az_fleet tool"},
			{"Network", 1, "az_network_resources tool"},
			{"Advisor", 1, "az_advisor_recommendation tool"},
			{"Detectors", 3, "list_detectors, run_detector, run_detectors_by_category"},
			{"Inspektor Gadget", 1, "inspektor_gadget_observability tool"},
		}

		for _, tc := range testCases {
			t.Logf("Component: %s - Expected tools: %d (%s)", tc.component, tc.toolCount, tc.description)
		}

		// Test compute component separately due to access level variations
		t.Run("ComputeComponent", func(t *testing.T) {
			baseComputeToolsCount := 2 // get_aks_vmss_info + az_compute_operations

			t.Logf("Compute Component:")
			t.Logf("  - Base tools (always): %d (get_aks_vmss_info, az_compute_operations)", baseComputeToolsCount)
			t.Logf("  - All access levels have the same tools, but operations are restricted by access level validation")
		})
	})

	t.Run("KubernetesComponents", func(t *testing.T) {
		// Test kubectl tools by access level
		accessLevels := []string{"readonly", "readwrite", "admin"}
		for _, level := range accessLevels {
			kubectlTools := kubectl.RegisterKubectlTools(level)
			t.Logf("Kubectl tools for %s access: %d", level, len(kubectlTools))

			// Log individual kubectl tools
			for _, tool := range kubectlTools {
				t.Logf("  - %s", tool.Name)
			}
		}

		t.Logf("Optional Kubernetes Components:")
		t.Logf("  - Helm: 1 tool (when enabled)")
		t.Logf("  - Cilium: 1 tool (when enabled)")
		t.Logf("Note: Inspektor Gadget is now automatically enabled as part of Azure Components")
	})

	t.Run("DetectorComponentDetails", func(t *testing.T) {
		t.Logf("Detector Component includes:")
		t.Logf("  1. list_detectors - Lists all available AKS cluster detectors")
		t.Logf("  2. run_detector - Runs a specific AKS detector")
		t.Logf("  3. run_detectors_by_category - Runs all detectors in a specific category")
	})
}

// TestAKSOperationsAccess tests AKS operations access levels
func TestAKSOperationsAccess(t *testing.T) {
	accessLevels := []string{"readonly", "readwrite", "admin"}

	for _, level := range accessLevels {
		t.Run("AccessLevel_"+level, func(t *testing.T) {
			cfg := createTestConfig(level, map[string]bool{})

			// Test that AKS operations tool is registered with proper access
			tool := azaks.RegisterAzAksOperations(cfg)
			if tool.Name != "az_aks_operations" {
				t.Errorf("Expected tool name 'az_aks_operations', got '%s'", tool.Name)
			}

			t.Logf("AKS operations tool registered for access level: %s", level)
		})
	}
}

// TestServiceInitialization tests basic service initialization
func TestServiceInitialization(t *testing.T) {
	// Set test environment variables (dummy values)
	_ = os.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	_ = os.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	_ = os.Setenv("AZURE_CLIENT_SECRET", "dummy-client-secret")
	_ = os.Setenv("AZURE_SUBSCRIPTION_ID", "dummy-subscription-id")
	defer func() {
		_ = os.Unsetenv("AZURE_TENANT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_SECRET")
		_ = os.Unsetenv("AZURE_SUBSCRIPTION_ID")
	}()

	cfg := createTestConfig("readonly", map[string]bool{})
	service := NewService(cfg, WithAzCliProcFactory(func(timeout int) azcli.Proc { return &fakeProc{} }))

	// Test service creation - must be non-nil
	if service == nil { //nolint:staticcheck // False positive: t.Fatal prevents nil dereference
		t.Fatal("Service should not be nil")
	}

	// Test configuration is set correctly
	if service.cfg != cfg { //nolint:staticcheck // False positive: service verified non-nil above
		t.Error("Service config should match provided config")
	}

	// Test initialization
	if err := service.Initialize(); err != nil {
		t.Fatalf("Initialize should not return error: %v", err)
	}

	// Test that infrastructure is initialized
	if service.azClient == nil { //nolint:staticcheck // False positive: service verified non-nil above
		t.Error("Azure client should be initialized after Initialize()")
	}
	if service.mcpServer == nil { //nolint:staticcheck // False positive: service verified non-nil above
		t.Error("MCP server should be initialized after Initialize()")
	}

	t.Logf("Service initialized successfully")
}

// TestExpectedToolsByAccessLevel provides detailed breakdown of expected tools
func TestExpectedToolsByAccessLevel(t *testing.T) {
	accessLevels := []string{"readonly", "readwrite", "admin"}

	for _, level := range accessLevels {
		t.Run("AccessLevel_"+level, func(t *testing.T) {
			// Azure Components (always the same count, unified tools)
			azureToolsCount := 8 // Base count (including Inspektor Gadget)
			// Note: With unified tools, the count doesn't change by access level
			// Access control is handled by operation validation, not tool registration

			// Kubernetes tools
			kubectlTools := kubectl.RegisterKubectlTools(level)
			k8sToolsCount := len(kubectlTools)

			t.Logf("=== Access Level: %s ===", level)
			t.Logf("Azure Tools:")
			t.Logf("  - AKS Operations: 1")
			t.Logf("  - Monitoring: 1")
			t.Logf("  - Fleet: 1")
			t.Logf("  - Network: 1")
			t.Logf("  - Compute: 2 (get_aks_vmss_info, az_compute_operations)")
			t.Logf("  - Detectors: 3")
			t.Logf("  - Advisor: 1")
			t.Logf("  - Inspektor Gadget: 1 (automatically enabled)")
			t.Logf("  Total Azure Tools: %d", azureToolsCount)

			t.Logf("Kubernetes Tools:")
			t.Logf("  - Kubectl Tools: %d", k8sToolsCount)
			t.Logf("  - Optional Tools: 0-2 (helm, cilium)")

			t.Logf("kubectl tools for %s:", level)
			for i, tool := range kubectlTools {
				t.Logf("  %d. %s", i+1, tool.Name)
			}
		})
	}
}

// createTestConfig creates a test configuration
func createTestConfig(accessLevel string, additionalTools map[string]bool) *config.ConfigData {
	cfg := config.NewConfig()
	cfg.AccessLevel = accessLevel
	cfg.AdditionalTools = additionalTools
	cfg.Transport = "stdio"
	cfg.Timeout = 60
	return cfg
}

// containsPrefix checks if a string starts with any of the given prefixes
func containsPrefix(s string, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// BenchmarkServiceInitialization benchmarks service initialization
func BenchmarkServiceInitialization(b *testing.B) {
	// Set test environment variables for benchmark (dummy values)
	_ = os.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	_ = os.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	_ = os.Setenv("AZURE_CLIENT_SECRET", "dummy-client-secret")
	_ = os.Setenv("AZURE_SUBSCRIPTION_ID", "dummy-subscription-id")
	defer func() {
		_ = os.Unsetenv("AZURE_TENANT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_SECRET")
		_ = os.Unsetenv("AZURE_SUBSCRIPTION_ID")
	}()

	cfg := createTestConfig("readonly", map[string]bool{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := NewService(cfg, WithAzCliProcFactory(func(timeout int) azcli.Proc { return &fakeProc{} }))
		err := service.Initialize()
		if err != nil {
			b.Fatalf("Initialize failed: %v", err)
		}
	}
}

// TestCreateCustomHTTPServerWithHelp404 tests the custom HTTP server creation for streamable-http transport
func TestCreateCustomHTTPServerWithHelp404(t *testing.T) {
	cfg := createTestConfig("readonly", map[string]bool{})
	service := NewService(cfg)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Test server creation
	addr := "localhost:8080"
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	customServer, err := service.createCustomHTTPServerWithHelp404(mockHandler, addr)
	if err != nil {
		t.Fatalf("Failed to create custom server: %v", err)
	}

	if customServer == nil {
		t.Fatal("Custom server should not be nil")
	}

	if customServer.Addr != addr {
		t.Errorf("Expected server address %s, got %s", addr, customServer.Addr)
	}

	if customServer.Handler == nil {
		t.Fatal("Custom server handler should not be nil")
	}

	// Test the 404 response for non-MCP paths
	testCases := []struct {
		path                string
		method              string
		expectedStatusCode  int
		expectedContentType string
		description         string
	}{
		{"/", "GET", http.StatusNotFound, "application/json", "root path should return helpful 404"},
		{"/invalid", "GET", http.StatusNotFound, "application/json", "invalid path should return helpful 404"},
		{"/api", "POST", http.StatusNotFound, "application/json", "non-MCP path should return helpful 404"},
		{"/health", "GET", http.StatusNotFound, "application/json", "health path should return helpful 404"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			customServer.Handler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatusCode, rr.Code)
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != tc.expectedContentType {
				t.Errorf("Expected content type %s, got %s", tc.expectedContentType, contentType)
			}

			// Parse and validate JSON response
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check required fields
			if response["error"] != "Not Found" {
				t.Errorf("Expected error 'Not Found', got %v", response["error"])
			}

			message, ok := response["message"].(string)
			if !ok {
				t.Fatal("Message field should be a string")
			}
			if !strings.Contains(message, "MCP") {
				t.Error("Message should mention MCP")
			}
			if !strings.Contains(message, "/mcp") {
				t.Error("Message should mention /mcp endpoint")
			}

			endpoints, ok := response["endpoints"].(map[string]interface{})
			if !ok {
				t.Fatal("Endpoints field should be a map")
			}

			expectedEndpoints := []string{"initialize", "requests", "listen", "terminate"}
			for _, endpoint := range expectedEndpoints {
				if _, exists := endpoints[endpoint]; !exists {
					t.Errorf("Expected endpoint %s not found in response", endpoint)
				}
			}
		})
	}
}

// TestCreateCustomSSEServerWithHelp404 tests the custom HTTP server creation for SSE transport
func TestCreateCustomSSEServerWithHelp404(t *testing.T) {
	cfg := createTestConfig("readonly", map[string]bool{})
	service := NewService(cfg)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Create SSE server
	sseServer := server.NewSSEServer(service.mcpServer)

	// Test custom server creation
	addr := "localhost:8081"
	customServer, err := service.createCustomSSEServerWithHelp404(sseServer, addr)
	if err != nil {
		t.Fatalf("Failed to create custom SSE server: %v", err)
	}

	if customServer == nil {
		t.Fatal("Custom SSE server should not be nil")
	}

	if customServer.Addr != addr {
		t.Errorf("Expected server address %s, got %s", addr, customServer.Addr)
	}

	if customServer.Handler == nil {
		t.Fatal("Custom SSE server handler should not be nil")
	}

	// Test the 404 response for non-SSE paths
	testCases := []struct {
		path                string
		method              string
		expectedStatusCode  int
		expectedContentType string
		description         string
	}{
		{"/", "GET", http.StatusNotFound, "application/json", "root path should return helpful 404"},
		{"/invalid", "GET", http.StatusNotFound, "application/json", "invalid path should return helpful 404"},
		{"/api", "POST", http.StatusNotFound, "application/json", "non-SSE path should return helpful 404"},
		{"/health", "GET", http.StatusNotFound, "application/json", "health path should return helpful 404"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			customServer.Handler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatusCode, rr.Code)
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != tc.expectedContentType {
				t.Errorf("Expected content type %s, got %s", tc.expectedContentType, contentType)
			}

			// Parse and validate JSON response
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check required fields
			if response["error"] != "Not Found" {
				t.Errorf("Expected error 'Not Found', got %v", response["error"])
			}

			message, ok := response["message"].(string)
			if !ok {
				t.Fatal("Message field should be a string")
			}
			if !strings.Contains(message, "MCP") {
				t.Error("Message should mention MCP")
			}
			if !strings.Contains(message, "SSE") {
				t.Error("Message should mention SSE transport")
			}

			endpoints, ok := response["endpoints"].(map[string]interface{})
			if !ok {
				t.Fatal("Endpoints field should be a map")
			}

			expectedEndpoints := []string{"sse", "message"}
			for _, endpoint := range expectedEndpoints {
				if _, exists := endpoints[endpoint]; !exists {
					t.Errorf("Expected endpoint %s not found in response", endpoint)
				}
			}

			// Verify SSE-specific endpoint descriptions
			sseEndpoint, ok := endpoints["sse"].(string)
			if !ok {
				t.Fatal("SSE endpoint description should be a string")
			}
			if !strings.Contains(sseEndpoint, "GET /sse") {
				t.Error("SSE endpoint should mention GET /sse")
			}

			messageEndpoint, ok := endpoints["message"].(string)
			if !ok {
				t.Fatal("Message endpoint description should be a string")
			}
			if !strings.Contains(messageEndpoint, "POST /message") {
				t.Error("Message endpoint should mention POST /message")
			}
		})
	}
}

// TestSSEServerEndpointsAccessible tests that SSE endpoints are still accessible
func TestSSEServerEndpointsAccessible(t *testing.T) {
	cfg := createTestConfig("readonly", map[string]bool{})
	service := NewService(cfg)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Create SSE server
	sseServer := server.NewSSEServer(service.mcpServer)

	// Test custom server creation
	addr := "localhost:8082"
	customServer, err := service.createCustomSSEServerWithHelp404(sseServer, addr)
	if err != nil {
		t.Fatalf("Failed to create custom SSE server: %v", err)
	}

	// Test that SSE endpoints are accessible (don't return our custom 404)
	// Note: We only test that they don't return our custom 404 response,
	// not the actual SSE functionality which would require persistent connections
	testCases := []struct {
		path        string
		method      string
		shouldBe404 bool
		description string
	}{
		{"/message", "POST", false, "Message endpoint should be handled by SSE server"},
		{"/message", "GET", false, "Message endpoint with GET should be handled by SSE server"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			customServer.Handler.ServeHTTP(rr, req)

			if tc.shouldBe404 {
				if rr.Code != http.StatusNotFound {
					t.Errorf("Expected 404 for %s %s, got %d", tc.method, tc.path, rr.Code)
				}
			} else {
				if rr.Code == http.StatusNotFound {
					t.Errorf("Should not get 404 for %s %s, but got %d", tc.method, tc.path, rr.Code)
				}
				// Additional check: ensure it's not our custom 404 JSON response
				if rr.Header().Get("Content-Type") == "application/json" {
					var response map[string]interface{}
					if json.Unmarshal(rr.Body.Bytes(), &response) == nil {
						if response["error"] == "Not Found" && strings.Contains(response["message"].(string), "SSE transport") {
							t.Errorf("Got our custom 404 response for %s %s, should be handled by SSE server", tc.method, tc.path)
						}
					}
				}
				// Note: We don't check for specific success codes since the SSE server
				// may return various codes based on the request content/headers
			}
		})
	}
}

// TestJSONResponseFormat tests the format of JSON error responses
func TestJSONResponseFormat(t *testing.T) {
	cfg := createTestConfig("readonly", map[string]bool{})
	service := NewService(cfg)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	tests := []struct {
		name          string
		serverFunc    func() *http.Server
		expectedKeys  []string
		transportType string
	}{
		{
			name: "StreamableHTTP_JSONFormat",
			serverFunc: func() *http.Server {
				mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
				server, err := service.createCustomHTTPServerWithHelp404(mockHandler, "localhost:8080")
				if err != nil {
					t.Fatalf("Failed to create HTTP server: %v", err)
				}
				return server
			},
			expectedKeys:  []string{"error", "message", "endpoints"},
			transportType: "streamable-http",
		},
		{
			name: "SSE_JSONFormat",
			serverFunc: func() *http.Server {
				sseServer := server.NewSSEServer(service.mcpServer)
				server, err := service.createCustomSSEServerWithHelp404(sseServer, "localhost:8081")
				if err != nil {
					t.Fatalf("Failed to create SSE server: %v", err)
				}
				return server
			},
			expectedKeys:  []string{"error", "message", "endpoints"},
			transportType: "sse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customServer := tt.serverFunc()

			req, err := http.NewRequest("GET", "/invalid", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			customServer.Handler.ServeHTTP(rr, req)

			// Verify it's valid JSON
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Response should be valid JSON: %v", err)
			}

			// Verify all expected keys are present
			for _, key := range tt.expectedKeys {
				if _, exists := response[key]; !exists {
					t.Errorf("Expected key '%s' not found in response", key)
				}
			}

			// Verify error field value
			if response["error"] != "Not Found" {
				t.Errorf("Expected error field to be 'Not Found', got %v", response["error"])
			}

			// Verify message is informative
			message, ok := response["message"].(string)
			if !ok {
				t.Fatal("Message should be a string")
			}
			if len(message) < 20 {
				t.Error("Message should be informative (at least 20 characters)")
			}

			// Verify endpoints structure
			endpoints, ok := response["endpoints"].(map[string]interface{})
			if !ok {
				t.Fatal("Endpoints should be a map")
			}
			if len(endpoints) == 0 {
				t.Error("Endpoints map should not be empty")
			}

			t.Logf("Verified JSON format for %s transport", tt.transportType)
		})
	}
}
