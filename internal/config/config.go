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
	// List of enabled components (empty means all components enabled)
	EnabledComponents []string
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

	// TokenAuthOnly enables token-only authentication mode for tools that support it
	// When enabled, supported tools (e.g., kubectl) are executed via Azure AKS RunCommand API using user-provided tokens
	// When disabled (default), tools are executed locally with default authentication (e.g., kubeconfig for Kubernetes tools)
	TokenAuthOnly bool

	// DefaultAKSResourceID is the default AKS cluster resource ID used when aks_resource_id is not provided by the caller.
	// Set via --default-aks-resource-id flag or AZURE_AKS_RESOURCE_ID environment variable.
	DefaultAKSResourceID string

	// AllowedHosts is the set of HTTP Host header values (with or without port)
	// that the streamable-http / sse transports will accept. Empty means
	// loopback-only (the safe default). The literal "*" disables Host
	// enforcement entirely — only use behind a reverse proxy that validates
	// Host on the operator's behalf.
	AllowedHosts []string

	// AllowedOrigins is the set of HTTP Origin header values that the
	// streamable-http / sse transports will accept on browser-style cross
	// origin requests. Empty Origin headers (non-browser callers) are always
	// allowed. Empty list rejects every non-empty Origin. The literal "*"
	// disables Origin enforcement entirely.
	AllowedOrigins []string
}

// NewConfig creates and returns a new configuration instance
func NewConfig() *ConfigData {
	return &ConfigData{
		Timeout:           60,
		CacheTimeout:      1 * time.Minute,
		SecurityConfig:    security.NewSecurityConfig(),
		OAuthConfig:       auth.NewDefaultOAuthConfig(),
		Transport:         "stdio",
		Port:              8000,
		AccessLevel:       "readonly",
		EnabledComponents: []string{},
		AllowNamespaces:   "",
		LogLevel:          "info",
		UseLegacyTools:    os.Getenv("USE_LEGACY_TOOLS") == "true",
		TokenAuthOnly:     false,
		AllowedHosts:      []string{},
		AllowedOrigins:    []string{},
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

	// OAuth external URL configuration
	flag.StringVar(&cfg.OAuthConfig.ExternalURL, "oauth-external-url", "",
		"External base URL of the server (e.g. https://aks-mcp.example.com). Required when behind a TLS-terminating reverse proxy to ensure OAuth metadata uses https:// URLs. Falls back to OAUTH_EXTERNAL_URL env var.")

	// OAuth scopes configuration
	oauthScopes := flag.String("oauth-scopes", "",
		"Comma-separated list of OAuth scopes to require (e.g. api://your-app-id/.default). If empty, defaults to https://management.azure.com/.default")

	// OBO configuration
	flag.BoolVar(&cfg.OAuthConfig.OBOEnabled, "oauth-obo-enabled", false,
		"Enable On-Behalf-Of token exchange: trades the user's MCP bearer token for an ARM token so tokenAuthOnly tools run as the calling user (requires AZURE_CLIENT_SECRET)")

	// Component configuration
	enabledComponents := flag.String("enabled-components", "",
		"Comma-separated list of enabled components (empty means all components enabled). Available: az_cli,monitor,fleet,network,compute,detectors,advisor,inspektorgadget,kubectl,helm,cilium,hubble")

	// HTTP transport security: DNS-rebinding / cross-origin protections
	// (apply to streamable-http and sse transports only).
	allowedHosts := flag.String("allowed-host", "",
		"Comma-separated list of HTTP Host header values to accept on /mcp, /sse and /message. Empty means loopback only (localhost, 127.0.0.1, [::1]). Use '*' as an escape valve when behind a trusted reverse proxy. Required for non-loopback bindings when OAuth is disabled.")
	trustedOrigins := flag.String("trusted-origin", "",
		"Comma-separated list of HTTP Origin header values to accept on browser-style cross-origin requests to /mcp, /sse and /message (e.g. https://chat.example.com). Empty Origin headers from non-browser clients are always allowed. Use '*' to disable Origin enforcement entirely.")

	// Kubernetes namespaces configuration
	flag.StringVar(&cfg.AllowNamespaces, "allow-namespaces", "",
		"Comma-separated list of allowed Kubernetes namespaces (empty means all namespaces)")

	// Token-only authentication configuration
	flag.BoolVar(&cfg.TokenAuthOnly, "token-auth-only", false,
		"Enable token-only authentication mode for supported tools (e.g., kubectl uses Azure AKS RunCommand API with user-provided tokens instead of local kubeconfig)")

	// Default AKS resource ID
	flag.StringVar(&cfg.DefaultAKSResourceID, "default-aks-resource-id", "",
		"Default AKS cluster resource ID used when aks_resource_id is not supplied by the caller (e.g. /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{cluster}). Falls back to AZURE_AKS_RESOURCE_ID env var.")

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
	if err := cfg.parseOAuthConfig(*additionalRedirectURIs, *allowedCORSOrigins, *oauthScopes); err != nil {
		fmt.Printf("OAuth configuration error: %v\n", err)
		os.Exit(1)
	}

	// Fall back to environment variable for default AKS resource ID
	if cfg.DefaultAKSResourceID == "" {
		cfg.DefaultAKSResourceID = os.Getenv("AZURE_AKS_RESOURCE_ID")
	}

	// Parse enabled components
	if *enabledComponents != "" {
		components := strings.Split(*enabledComponents, ",")
		for _, comp := range components {
			comp = strings.TrimSpace(comp)
			if comp != "" {
				cfg.EnabledComponents = append(cfg.EnabledComponents, comp)
			}
		}
	}

	// Parse HTTP transport allowlists.
	cfg.AllowedHosts = splitAndTrim(*allowedHosts)
	cfg.AllowedOrigins = splitAndTrim(*trustedOrigins)
}

// splitAndTrim splits raw on commas, trims whitespace, and drops empty entries.
// Returns nil for empty input so a zero allowlist remains a zero allowlist.
func splitAndTrim(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseOAuthConfig parses OAuth-related command line arguments
func (cfg *ConfigData) parseOAuthConfig(additionalRedirectURIs, allowedCORSOrigins, oauthScopes string) error {
	// Parse custom OAuth scopes if provided
	if oauthScopes != "" {
		scopes := strings.Split(oauthScopes, ",")
		cfg.OAuthConfig.RequiredScopes = []string{}
		for _, scope := range scopes {
			trimmedScope := strings.TrimSpace(scope)
			if trimmedScope != "" {
				cfg.OAuthConfig.RequiredScopes = append(cfg.OAuthConfig.RequiredScopes, trimmedScope)
			}
		}
		// Update expected audience to match the custom scope resource.
		// For api:// scopes the audience is the app URI (api://<app-id>),
		// regardless of the permission suffix (/.default, /access_as_user, etc.).
		if len(cfg.OAuthConfig.RequiredScopes) > 0 {
			firstScope := cfg.OAuthConfig.RequiredScopes[0]
			var audience string
			if strings.HasPrefix(firstScope, "api://") {
				// Strip permission suffix: "api://app-id/permission" → "api://app-id"
				withoutScheme := strings.TrimPrefix(firstScope, "api://")
				appID := strings.SplitN(withoutScheme, "/", 2)[0]
				audience = "api://" + appID
			} else {
				// For https:// scopes (e.g. https://management.azure.com/.default)
				audience = strings.TrimSuffix(firstScope, "/.default")
				audience = strings.TrimSuffix(audience, "/")
			}
			cfg.OAuthConfig.TokenValidation.ExpectedAudience = audience
			logger.Infof("OAuth Config: Using custom scopes %v with audience %s", cfg.OAuthConfig.RequiredScopes, audience)
		}
	}

	// Load external URL from environment variable if not set via CLI
	if cfg.OAuthConfig.ExternalURL == "" {
		if externalURL := os.Getenv("OAUTH_EXTERNAL_URL"); externalURL != "" {
			cfg.OAuthConfig.ExternalURL = externalURL
			logger.Debugf("OAuth Config: Using external URL from environment variable OAUTH_EXTERNAL_URL")
		}
	}

	// Load client secret for OBO flow from environment variable
	if cfg.OAuthConfig.ClientSecret == "" {
		if secret := os.Getenv("AZURE_CLIENT_SECRET"); secret != "" {
			cfg.OAuthConfig.ClientSecret = secret
			logger.Debugf("OAuth Config: Using client secret from environment variable AZURE_CLIENT_SECRET")
		}
	}

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

	// Validate token-only authentication + stdio transport compatibility
	if cfg.TokenAuthOnly && cfg.Transport == "stdio" {
		return fmt.Errorf("token-only authentication mode (--token-auth-only) is not supported with stdio transport, use sse or streamable-http instead")
	}

	// Validate token-only authentication + legacy tools compatibility
	if cfg.TokenAuthOnly && cfg.UseLegacyTools {
		return fmt.Errorf("token-only authentication mode (--token-auth-only) requires unified tools and is not compatible with legacy tools (USE_LEGACY_TOOLS=true)")
	}

	// Refuse to start a publicly reachable streamable-http / sse listener
	// with no authentication and no explicit trusted-host allowlist. This
	// is the DNS-rebinding posture: a browser-origin attacker can otherwise
	// reach /mcp through the victim's local network. The operator must pick
	// one of three safe combinations:
	//   (a) bind to loopback (default --host 127.0.0.1)
	//   (b) enable OAuth (--oauth-enabled)
	//   (c) declare an explicit trusted-host allowlist (--allowed-host)
	if isHTTPTransport(cfg.Transport) &&
		!isLoopbackBindHost(cfg.Host) &&
		!cfg.OAuthConfig.Enabled &&
		len(cfg.AllowedHosts) == 0 {
		return fmt.Errorf("transport %q bound to non-loopback host %q without OAuth and without --allowed-host: refusing to start to avoid DNS-rebinding exposure; choose one of: (a) bind --host 127.0.0.1, (b) --oauth-enabled, (c) --allowed-host=<your.hostname>",
			cfg.Transport, cfg.Host)
	}

	return nil
}

// isHTTPTransport reports whether transport opens an HTTP listener that
// browsers can target. stdio is excluded because it has no HTTP attack surface.
func isHTTPTransport(transport string) bool {
	switch transport {
	case "streamable-http", "sse":
		return true
	}
	return false
}

// isLoopbackBindHost reports whether host (as configured via --host) only
// binds to a loopback interface and therefore cannot be reached by a remote
// browser-origin attacker. Used for startup-time safety validation.
// An empty host is treated as loopback because callers that never invoke
// ParseFlags (notably unit tests) leave Host at its zero value, and the
// production default applied by ParseFlags is 127.0.0.1 anyway.
func isLoopbackBindHost(host string) bool {
	switch host {
	case "", "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
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
