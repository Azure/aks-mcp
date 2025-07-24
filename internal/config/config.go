package config

import (
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/security"
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

	// Component-specific options
	// Map of enabled components (azaks always enabled, others optional)
	EnabledComponents map[string]bool
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
		EnabledComponents: map[string]bool{
			"azaks": true, // azaks is always enabled by default
		},
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

	// Component-specific settings
	enabledComponents := flag.String("enabled-components", "azaks",
		"Comma-separated list of enabled components (azaks is always enabled). Available: azaks,network,compute,monitor,advisor,fleet,detectors,kubernetes")

	flag.Parse()

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

	// Parse enabled components
	if *enabledComponents != "" {
		// Reset EnabledComponents but keep azaks always enabled
		cfg.EnabledComponents = map[string]bool{
			"azaks": true, // azaks is always enabled
		}

		components := strings.Split(*enabledComponents, ",")
		for _, component := range components {
			componentName := strings.TrimSpace(component)
			if componentName != "" {
				cfg.EnabledComponents[componentName] = true
			}
		}
	}
}
