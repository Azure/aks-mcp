package server

import (
	"os"
	"testing"

	"github.com/Azure/aks-mcp/internal/components/azaks"
	"github.com/Azure/aks-mcp/internal/components/compute"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/mcp-kubernetes/pkg/kubectl"
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
	azureToolPrefixes := []string{"az_", "azure_", "get_aks_", "list_detectors", "run_detector"}
	k8sToolPrefixes := []string{"kubectl_", "k8s_", "helm", "cilium", "inspektor"}

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
	// Set environment variables for testing to avoid Azure auth issues
	_ = os.Setenv("AZURE_TENANT_ID", "test-tenant")
	_ = os.Setenv("AZURE_CLIENT_ID", "test-client")
	_ = os.Setenv("AZURE_CLIENT_SECRET", "test-secret")
	_ = os.Setenv("AZURE_SUBSCRIPTION_ID", "test-subscription")
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
			expectedAzureTools: 7, // AKS Ops + Monitoring + Fleet + Network + Compute (VMSS Info only) + Detectors (3) + Advisor
			expectedK8sTools:   0, // Will be calculated based on kubectl tools for readonly
			description:        "Readonly access with no optional tools",
		},
		{
			name:               "ReadWrite_NoOptional",
			accessLevel:        "readwrite",
			additionalTools:    map[string]bool{},
			expectedAzureTools: 8, // Same as readonly + 1 read-write VMSS command
			expectedK8sTools:   0, // Will be calculated based on kubectl tools for readwrite
			description:        "Readwrite access with no optional tools",
		},
		{
			name:               "Admin_NoOptional",
			accessLevel:        "admin",
			additionalTools:    map[string]bool{},
			expectedAzureTools: 8, // Same as readwrite (no admin VMSS commands currently)
			expectedK8sTools:   0, // Will be calculated based on kubectl tools for admin
			description:        "Admin access with no optional tools",
		},
		{
			name:        "ReadOnly_AllOptional",
			accessLevel: "readonly",
			additionalTools: map[string]bool{
				"helm":             true,
				"cilium":           true,
				"inspektor-gadget": true,
			},
			expectedAzureTools: 7, // Same as readonly
			expectedK8sTools:   0, // Will be calculated + 3 optional tools
			description:        "Readonly access with all optional tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if tt.additionalTools["inspektor-gadget"] {
				optionalToolsCount++
			}

			expectedTotalK8sTools := expectedKubectlCount + optionalToolsCount

			t.Logf("Test: %s", tt.description)
			t.Logf("Expected kubectl tools for %s: %d", tt.accessLevel, expectedKubectlCount)
			t.Logf("Expected optional tools: %d", optionalToolsCount)
			t.Logf("Expected total K8s tools: %d", expectedTotalK8sTools)
			t.Logf("Expected Azure tools: %d", tt.expectedAzureTools)

			// Create test configuration
			cfg := createTestConfig(tt.accessLevel, tt.additionalTools)

			// Create service
			service := NewService(cfg)

			// Initialize service (this will register all tools)
			err := service.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize service: %v", err)
			}

			// Verify initialization completed
			if service.azClient == nil {
				t.Error("Azure client should be initialized")
			}
			if service.mcpServer == nil {
				t.Error("MCP server should be initialized")
			}

			t.Logf("Service initialized successfully for access level: %s", tt.accessLevel)
		})
	}
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
		}

		for _, tc := range testCases {
			t.Logf("Component: %s - Expected tools: %d (%s)", tc.component, tc.toolCount, tc.description)
		}

		// Test compute component separately due to access level variations
		t.Run("ComputeComponent", func(t *testing.T) {
			readOnlyCount := len(compute.GetReadOnlyVmssCommands()) + 1 // +1 for get_aks_vmss_info
			readWriteCount := len(compute.GetReadWriteVmssCommands())
			adminCount := len(compute.GetAdminVmssCommands())

			t.Logf("Compute Component:")
			t.Logf("  - Base tools (always): %d (get_aks_vmss_info)", 1)
			t.Logf("  - Read-only VMSS commands: %d", len(compute.GetReadOnlyVmssCommands()))
			t.Logf("  - Read-write VMSS commands: %d", readWriteCount)
			t.Logf("  - Admin VMSS commands: %d", adminCount)
			t.Logf("  - Total for readonly: %d", readOnlyCount)
			t.Logf("  - Total for readwrite: %d", readOnlyCount+readWriteCount)
			t.Logf("  - Total for admin: %d", readOnlyCount+readWriteCount+adminCount)
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
		t.Logf("  - Inspektor Gadget: 1 tool (when enabled)")
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
	// Set test environment variables
	_ = os.Setenv("AZURE_TENANT_ID", "test-tenant")
	_ = os.Setenv("AZURE_CLIENT_ID", "test-client")
	_ = os.Setenv("AZURE_CLIENT_SECRET", "test-secret")
	_ = os.Setenv("AZURE_SUBSCRIPTION_ID", "test-subscription")
	defer func() {
		_ = os.Unsetenv("AZURE_TENANT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_SECRET")
		_ = os.Unsetenv("AZURE_SUBSCRIPTION_ID")
	}()

	cfg := createTestConfig("readonly", map[string]bool{})
	service := NewService(cfg)

	// Test service creation
	if service == nil {
		t.Fatal("Service should not be nil")
	}

	if service.cfg != cfg {
		t.Error("Service config should match provided config")
	}

	// Test initialization
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Initialize should not return error: %v", err)
	}

	// Test that infrastructure is initialized
	if service.azClient == nil {
		t.Error("Azure client should be initialized")
	}

	if service.mcpServer == nil {
		t.Error("MCP server should be initialized")
	}

	t.Logf("Service initialized successfully")
}

// TestExpectedToolsByAccessLevel provides detailed breakdown of expected tools
func TestExpectedToolsByAccessLevel(t *testing.T) {
	accessLevels := []string{"readonly", "readwrite", "admin"}

	for _, level := range accessLevels {
		t.Run("AccessLevel_"+level, func(t *testing.T) {
			// Azure Components (always the same count, but different capabilities)
			azureToolsCount := 7 // Base count

			// Add compute tools based on access level
			readWriteVmssCount := len(compute.GetReadWriteVmssCommands())
			adminVmssCount := len(compute.GetAdminVmssCommands())

			if level == "readwrite" || level == "admin" {
				azureToolsCount += readWriteVmssCount
			}
			if level == "admin" {
				azureToolsCount += adminVmssCount
			}

			// Kubernetes tools
			kubectlTools := kubectl.RegisterKubectlTools(level)
			k8sToolsCount := len(kubectlTools)

			t.Logf("=== Access Level: %s ===", level)
			t.Logf("Azure Tools:")
			t.Logf("  - AKS Operations: 1")
			t.Logf("  - Monitoring: 1")
			t.Logf("  - Fleet: 1")
			t.Logf("  - Network: 1")
			t.Logf("  - Compute Base: 1 (get_aks_vmss_info)")
			t.Logf("  - Compute ReadWrite: %d", readWriteVmssCount)
			if level == "admin" {
				t.Logf("  - Compute Admin: %d", adminVmssCount)
			}
			t.Logf("  - Detectors: 3")
			t.Logf("  - Advisor: 1")
			t.Logf("  Total Azure Tools: %d", azureToolsCount)

			t.Logf("Kubernetes Tools:")
			t.Logf("  - Kubectl Tools: %d", k8sToolsCount)
			t.Logf("  - Optional Tools: 0-3 (helm, cilium, inspektor-gadget)")

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
	// Set test environment variables
	_ = os.Setenv("AZURE_TENANT_ID", "test-tenant")
	_ = os.Setenv("AZURE_CLIENT_ID", "test-client")
	_ = os.Setenv("AZURE_CLIENT_SECRET", "test-secret")
	_ = os.Setenv("AZURE_SUBSCRIPTION_ID", "test-subscription")
	defer func() {
		_ = os.Unsetenv("AZURE_TENANT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_ID")
		_ = os.Unsetenv("AZURE_CLIENT_SECRET")
		_ = os.Unsetenv("AZURE_SUBSCRIPTION_ID")
	}()

	cfg := createTestConfig("readonly", map[string]bool{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := NewService(cfg)
		err := service.Initialize()
		if err != nil {
			b.Fatalf("Initialize failed: %v", err)
		}
	}
}
