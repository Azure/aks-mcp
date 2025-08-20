package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/security"
	"github.com/Azure/aks-mcp/internal/telemetry"
	"github.com/Azure/aks-mcp/internal/version"
	flag "github.com/spf13/pflag"
)

// ConfigData holds the global configuration
type ConfigData struct {
	// Command execution timeout in seconds
	Timeout int
	// Cache timeout for Azure resources
	CacheTimeout time.Duration
	// Security configuration
	SecurityConfig *security.SecurityConfig

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

	// Authentication configuration
	Auth *AuthConfig
}

// NewConfig creates and returns a new configuration instance
func NewConfig() *ConfigData {
	return &ConfigData{
		Timeout:         60,
		CacheTimeout:    1 * time.Minute,
		SecurityConfig:  security.NewSecurityConfig(),
		Transport:       "stdio",
		Port:            8000,
		AccessLevel:     "readonly",
		AdditionalTools: make(map[string]bool),
		AllowNamespaces: "",
		Auth:            NewAuthConfig(),
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

	// Kubernetes-specific settings
	additionalTools := flag.String("additional-tools", "",
		"Comma-separated list of additional Kubernetes tools to support (kubectl is always enabled). Available: helm,cilium")
	flag.StringVar(&cfg.AllowNamespaces, "allow-namespaces", "",
		"Comma-separated list of allowed Kubernetes namespaces (empty means all namespaces)")

	// Authentication settings
	flag.BoolVar(&cfg.Auth.Enabled, "auth-enabled", false, "Enable authentication")
	flag.StringVar(&cfg.Auth.EntraClientID, "auth-client-id", "", "Entra ID client ID")
	flag.StringVar(&cfg.Auth.EntraTenantID, "auth-tenant-id", "", "Entra ID tenant ID")
	flag.StringVar(&cfg.Auth.EntraAuthority, "auth-authority", "https://login.microsoftonline.com", "Entra ID authority URL for different Azure clouds (Public: login.microsoftonline.com, China: login.chinacloudapi.cn, Government: login.microsoftonline.us)")
	flag.IntVar(&cfg.Auth.JWKSCacheTimeout, "auth-jwks-cache-timeout", 3600, "JWKS cache timeout in seconds")
	flag.BoolVar(&cfg.Auth.RequireAuthForHTTP, "auth-require-for-http", true, "Require authentication for HTTP transports")

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

	// Parse additional tools
	if *additionalTools != "" {
		tools := strings.Split(*additionalTools, ",")
		for _, tool := range tools {
			cfg.AdditionalTools[strings.TrimSpace(tool)] = true
		}
	}

	// Load authentication configuration from environment variables if not set via flags
	cfg.loadAuthFromEnv()
}

// loadAuthFromEnv loads authentication configuration from environment variables
func (cfg *ConfigData) loadAuthFromEnv() {
	// Only load from env if values weren't set via flags
	if cfg.Auth.EntraClientID == "" {
		if clientID := os.Getenv("AKS_MCP_AUTH_ENTRA_CLIENT_ID"); clientID != "" {
			cfg.Auth.EntraClientID = clientID
		}
	}

	if cfg.Auth.EntraTenantID == "" {
		if tenantID := os.Getenv("AKS_MCP_AUTH_ENTRA_TENANT_ID"); tenantID != "" {
			cfg.Auth.EntraTenantID = tenantID
		}
	}

	if cfg.Auth.EntraAuthority == "https://login.microsoftonline.com" {
		if authority := os.Getenv("AKS_MCP_AUTH_ENTRA_AUTHORITY"); authority != "" {
			cfg.Auth.EntraAuthority = authority
		}
	}

	// Load enabled flag from environment if not set
	if !cfg.Auth.Enabled {
		if enabled := os.Getenv("AKS_MCP_AUTH_ENABLED"); enabled == "true" {
			cfg.Auth.Enabled = true
		}
	}
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
