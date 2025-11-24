package config

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/auth"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/aks-mcp/internal/security"
	"github.com/Azure/aks-mcp/internal/telemetry"
	"github.com/Azure/aks-mcp/internal/version"
	flag "github.com/spf13/pflag"
)

// EnableCache controls whether caching is enabled globally
// Cache is enabled by default for production performance
// This affects both web cache headers and AzureOAuthProvider cache
// Can be disabled via DISABLE_CACHE environment variable
var EnableCache = os.Getenv("DISABLE_CACHE") != "true"

// validateGUID validates that a value is in valid GUID format
func validateGUID(value, name string) error {
	if value == "" {
		return nil // Empty values are allowed (will be handled by OAuth validation)
	}

	// GUID pattern: 8-4-4-4-12 hexadecimal digits with hyphens
	guidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !guidRegex.MatchString(value) {
		return fmt.Errorf("%s must be a valid GUID format (e.g., 12345678-1234-1234-1234-123456789abc), got: %s", name, value)
	}
	return nil
}

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

	// Log level (debug, info, warn, error)
	LogLevel string

	// OTLP endpoint for OpenTelemetry traces
	OTLPEndpoint string

	// Telemetry service
	TelemetryService *telemetry.Service

	// UseLegacyTools controls whether to use legacy tools or new unified tools
	// Azure tools: true = az_aks_operations/az_compute_operations, false = call_az
	// Kubectl tools: true = specialized tools (kubectl_resources, kubectl_workloads, etc.), false = call_kubectl
	// Default is false (use new unified tools)
	// This flag is provided for backward compatibility and may be removed in future versions
	UseLegacyTools bool
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
		LogLevel:        "info",
		UseLegacyTools:  os.Getenv("USE_LEGACY_TOOLS") == "true",
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

	// OAuth redirect URIs configuration
	additionalRedirectURIs := flag.String("oauth-redirects", "",
		"Comma-separated list of additional OAuth redirect URIs (e.g. http://localhost:8000/oauth/callback,http://localhost:6274/oauth/callback)")

	// OAuth CORS origins configuration
	allowedCORSOrigins := flag.String("oauth-cors-origins", "",
		"Comma-separated list of allowed CORS origins for OAuth endpoints (e.g. http://localhost:6274). If empty, no cross-origin requests are allowed for security")

	// Kubernetes-specific settings
	additionalTools := flag.String("additional-tools", "",
		"Comma-separated list of additional Kubernetes tools to support (kubectl is always enabled). Available: helm,cilium,hubble")
	flag.StringVar(&cfg.AllowNamespaces, "allow-namespaces", "",
		"Comma-separated list of allowed Kubernetes namespaces (empty means all namespaces)")

	// Logging settings
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")

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
	if err := cfg.parseOAuthConfig(*additionalRedirectURIs, *allowedCORSOrigins); err != nil {
		fmt.Printf("OAuth configuration error: %v\n", err)
		os.Exit(1)
	}

	// Parse additional tools
	if *additionalTools != "" {
		tools := strings.Split(*additionalTools, ",")
		for _, tool := range tools {
			cfg.AdditionalTools[strings.TrimSpace(tool)] = true
		}
	}
}

// parseOAuthConfig parses OAuth-related command line arguments
func (cfg *ConfigData) parseOAuthConfig(additionalRedirectURIs, allowedCORSOrigins string) error {
	// Note: OAuth scopes are automatically configured to use "https://management.azure.com/.default"
	// and are not configurable via command line per design

	// Track configuration sources for logging
	var tenantIDSource, clientIDSource string

	// Load OAuth configuration from environment variables if not set via CLI
	if cfg.OAuthConfig.TenantID == "" {
		if tenantID := os.Getenv("AZURE_TENANT_ID"); tenantID != "" {
			cfg.OAuthConfig.TenantID = tenantID
			tenantIDSource = "environment variable AZURE_TENANT_ID"
			logger.Debugf("OAuth Config: Using tenant ID from environment variable AZURE_TENANT_ID")
		}
	} else {
		tenantIDSource = "command line flag --oauth-tenant-id"
		logger.Debugf("OAuth Config: Using tenant ID from command line flag --oauth-tenant-id")
	}

	if cfg.OAuthConfig.ClientID == "" {
		if clientID := os.Getenv("AZURE_CLIENT_ID"); clientID != "" {
			cfg.OAuthConfig.ClientID = clientID
			clientIDSource = "environment variable AZURE_CLIENT_ID"
			logger.Debugf("OAuth Config: Using client ID from environment variable AZURE_CLIENT_ID")
		}
	} else {
		clientIDSource = "command line flag --oauth-client-id"
		logger.Debugf("OAuth Config: Using client ID from command line flag --oauth-client-id")
	}

	// Validate GUID formats for tenant ID and client ID
	if err := validateGUID(cfg.OAuthConfig.TenantID, "OAuth tenant ID"); err != nil {
		return fmt.Errorf("invalid OAuth tenant ID from %s: %w", tenantIDSource, err)
	}

	if err := validateGUID(cfg.OAuthConfig.ClientID, "OAuth client ID"); err != nil {
		return fmt.Errorf("invalid OAuth client ID from %s: %w", clientIDSource, err)
	}

	// Set redirect URIs based on configured host and port
	if cfg.OAuthConfig.Enabled {
		redirectURI := fmt.Sprintf("http://%s:%d/oauth/callback", cfg.Host, cfg.Port)
		cfg.OAuthConfig.RedirectURIs = []string{redirectURI}

		// Add localhost variant if using 127.0.0.1
		if cfg.Host == "127.0.0.1" {
			localhostURI := fmt.Sprintf("http://localhost:%d/oauth/callback", cfg.Port)
			cfg.OAuthConfig.RedirectURIs = append(cfg.OAuthConfig.RedirectURIs, localhostURI)
		}

		// Add additional redirect URIs from command line flag
		if additionalRedirectURIs != "" {
			additionalURIs := strings.Split(additionalRedirectURIs, ",")
			for _, uri := range additionalURIs {
				trimmedURI := strings.TrimSpace(uri)
				if trimmedURI != "" {
					cfg.OAuthConfig.RedirectURIs = append(cfg.OAuthConfig.RedirectURIs, trimmedURI)
				}
			}
		}
	}

	// Parse allowed CORS origins for OAuth endpoints
	if allowedCORSOrigins != "" {
		logger.Debugf("OAuth Config: Setting allowed CORS origins from command line flag --oauth-cors-origins")
		origins := strings.Split(allowedCORSOrigins, ",")
		for _, origin := range origins {
			trimmedOrigin := strings.TrimSpace(origin)
			if trimmedOrigin != "" {
				cfg.OAuthConfig.AllowedOrigins = append(cfg.OAuthConfig.AllowedOrigins, trimmedOrigin)
			}
		}
	} else {
		logger.Debugf("OAuth Config: No CORS origins configured - cross-origin requests will be blocked for security")
	}

	return nil
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
		logger.Errorf("Failed to initialize telemetry: %v", err)
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
