package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
	"github.com/Azure/aks-mcp/internal/security"
	"github.com/Azure/aks-mcp/internal/telemetry"
	"github.com/Azure/aks-mcp/internal/version"
	flag "github.com/spf13/pflag"
)

// EnableCache controls whether caching is enabled globally
// Set to false during debugging to avoid cache-related issues
// This affects both web cache headers and AzureOAuthProvider cache
const EnableCache = false

// ConfigData holds the global configuration
type ConfigData struct {
	// Command execution timeout in seconds
	Timeout int
	// Cache timeout for Azure resources
	CacheTimeout time.Duration
	// Security configuration
	SecurityConfig *security.SecurityConfig
	// OAuth configuration
	OAuthConfig *auth.OAuthConfig

	// Command-line specific options
	Transport   string
	Host        string
	Port        int
	AccessLevel string

	// Kubernetes-specific options
	// Map of additional tools enabled (helm, cilium)
	AdditionalTools map[string]bool
	// Comma-separated list of allowed Kubernetes namespaces
	AllowNamespaces string

	// Verbose logging
	Verbose bool

	// OTLP endpoint for OpenTelemetry traces
	OTLPEndpoint string

	// Telemetry service
	TelemetryService *telemetry.Service
}

// NewConfig creates and returns a new configuration instance
func NewConfig() *ConfigData {
	return &ConfigData{
		Timeout:         60,
		CacheTimeout:    1 * time.Minute,
		SecurityConfig:  security.NewSecurityConfig(),
		OAuthConfig:     auth.NewDefaultOAuthConfig(),
		Transport:       "stdio",
		Port:            8000,
		AccessLevel:     "readonly",
		AdditionalTools: make(map[string]bool),
		AllowNamespaces: "",
	}
}

// ParseFlags parses command line arguments and updates the configuration
func (cfg *ConfigData) ParseFlags() {
	// Server configuration
	flag.StringVar(&cfg.Transport, "transport", "stdio", "Transport mechanism to use (stdio, sse or streamable-http)")
	flag.StringVar(&cfg.Host, "host", "127.0.0.1", "Host to listen for the server (only used with transport sse or streamable-http)")
	flag.IntVar(&cfg.Port, "port", 8000, "Port to listen for the server (only used with transport sse or streamable-http)")
	flag.IntVar(&cfg.Timeout, "timeout", 600, "Timeout for command execution in seconds, default is 600s")

	// Security settings
	flag.StringVar(&cfg.AccessLevel, "access-level", "readonly", "Access level (readonly, readwrite, admin)")

	// OAuth configuration
	flag.BoolVar(&cfg.OAuthConfig.Enabled, "oauth-enabled", false, "Enable OAuth authentication")
	flag.StringVar(&cfg.OAuthConfig.TenantID, "oauth-tenant-id", "", "Azure AD tenant ID for OAuth (fallback to AZURE_TENANT_ID env var)")
	flag.StringVar(&cfg.OAuthConfig.ClientID, "oauth-client-id", "", "Azure AD client ID for OAuth (fallback to AZURE_CLIENT_ID env var)")

	// Kubernetes-specific settings
	additionalTools := flag.String("additional-tools", "",
		"Comma-separated list of additional Kubernetes tools to support (kubectl is always enabled). Available: helm,cilium,hubble")
	flag.StringVar(&cfg.AllowNamespaces, "allow-namespaces", "",
		"Comma-separated list of allowed Kubernetes namespaces (empty means all namespaces)")

	// Logging settings
	flag.BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose logging")

	// OTLP settings
	flag.StringVar(&cfg.OTLPEndpoint, "otlp-endpoint", "", "OTLP endpoint for OpenTelemetry traces (e.g. localhost:4317)")

	// Custom help handling
	var showHelp bool
	flag.BoolVarP(&showHelp, "help", "h", false, "Show help message")

	// Version flag
	showVersion := flag.Bool("version", false, "Show version information and exit")

	// Parse flags and handle errors properly
	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("\nUsage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Handle help manually with proper exit code
	if showHelp {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Handle version flag
	if *showVersion {
		cfg.PrintVersion()
		os.Exit(0)
	}

	// Update security config
	cfg.SecurityConfig.AccessLevel = cfg.AccessLevel
	cfg.SecurityConfig.AllowedNamespaces = cfg.AllowNamespaces

	// Parse OAuth configuration
	cfg.parseOAuthConfig()

	// Parse additional tools
	if *additionalTools != "" {
		tools := strings.Split(*additionalTools, ",")
		for _, tool := range tools {
			cfg.AdditionalTools[strings.TrimSpace(tool)] = true
		}
	}
}

// parseOAuthConfig parses OAuth-related command line arguments
func (cfg *ConfigData) parseOAuthConfig() {
	// Note: OAuth scopes are automatically configured to use "https://management.azure.com/.default"
	// and are not configurable via command line per design

	// Load OAuth configuration from environment variables if not set via CLI
	if cfg.OAuthConfig.TenantID == "" {
		cfg.OAuthConfig.TenantID = os.Getenv("AZURE_TENANT_ID")
	}
	if cfg.OAuthConfig.ClientID == "" {
		cfg.OAuthConfig.ClientID = os.Getenv("AZURE_CLIENT_ID")
	}
}

// ValidateConfig validates the configuration for incompatible settings
func (cfg *ConfigData) ValidateConfig() error {
	// Validate OAuth + transport compatibility
	if cfg.OAuthConfig.Enabled && cfg.Transport == "stdio" {
		return fmt.Errorf("OAuth authentication is not supported with stdio transport per MCP specification")
	}

	return nil
}

// InitializeTelemetry initializes the telemetry service
func (cfg *ConfigData) InitializeTelemetry(ctx context.Context, serviceName, serviceVersion string) {
	// Create telemetry configuration
	telemetryConfig := telemetry.NewConfig(serviceName, serviceVersion)

	// Override OTLP endpoint from CLI if provided
	if cfg.OTLPEndpoint != "" {
		telemetryConfig.SetOTLPEndpoint(cfg.OTLPEndpoint)
	}

	// Initialize telemetry service
	cfg.TelemetryService = telemetry.NewService(telemetryConfig)
	if err := cfg.TelemetryService.Initialize(ctx); err != nil {
		log.Printf("Failed to initialize telemetry: %v", err)
		// Continue without telemetry - this is not a fatal error
	}

	// Track MCP server startup
	cfg.TelemetryService.TrackServiceStartup(ctx)
}

// PrintVersion prints version information
func (cfg *ConfigData) PrintVersion() {
	versionInfo := version.GetVersionInfo()
	fmt.Printf("aks-mcp version %s\n", versionInfo["version"])
	fmt.Printf("Git commit: %s\n", versionInfo["gitCommit"])
	fmt.Printf("Git tree state: %s\n", versionInfo["gitTreeState"])
	fmt.Printf("Go version: %s\n", versionInfo["goVersion"])
	fmt.Printf("Platform: %s\n", versionInfo["platform"])
}
