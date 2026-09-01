package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/aks-mcp/internal/security"
	"github.com/Azure/aks-mcp/internal/telemetry"
	"github.com/Azure/aks-mcp/internal/version"
	flag "github.com/spf13/pflag"
)

// EnableCache controls whether caching is enabled globally.
var EnableCache = os.Getenv("DISABLE_CACHE") != "true"

// ConfigData holds the global configuration.
type ConfigData struct {
	Timeout              int
	CacheTimeout         time.Duration
	SecurityConfig       *security.SecurityConfig
	AccessLevel          string
	EnabledComponents    []string
	AllowNamespaces      string
	LogLevel             string
	OTLPEndpoint         string
	TelemetryService     *telemetry.Service
	UseLegacyTools       bool
	DefaultAKSResourceID string
}

// NewConfig creates and returns a new configuration instance.
func NewConfig() *ConfigData {
	return &ConfigData{
		Timeout:           60,
		CacheTimeout:      time.Minute,
		SecurityConfig:    security.NewSecurityConfig(),
		AccessLevel:       "readonly",
		EnabledComponents: []string{},
		LogLevel:          "info",
		UseLegacyTools:    os.Getenv("USE_LEGACY_TOOLS") == "true",
	}
}

// ParseFlags parses command line arguments and updates the configuration.
func (cfg *ConfigData) ParseFlags() {
	flags, showHelp, showVersion, err := cfg.parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Printf("\nUsage of %s:\n", os.Args[0])
		flags.PrintDefaults()
		os.Exit(1)
	}
	if showHelp {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flags.PrintDefaults()
		os.Exit(0)
	}
	if showVersion {
		cfg.PrintVersion()
		os.Exit(0)
	}
}

func (cfg *ConfigData) parseFlags(args []string) (*flag.FlagSet, bool, bool, error) {
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.IntVar(&cfg.Timeout, "timeout", 600, "Timeout for command execution in seconds, default is 600s")
	flags.StringVar(&cfg.AccessLevel, "access-level", "readonly", "Access level (readonly, readwrite, admin)")
	enabledComponents := flags.String("enabled-components", "", "Comma-separated list of enabled components (empty means all components enabled). Available: az_cli,monitor,fleet,network,compute,detectors,advisor,inspektorgadget,kubectl,helm,cilium,hubble")
	flags.StringVar(&cfg.AllowNamespaces, "allow-namespaces", "", "Comma-separated list of allowed Kubernetes namespaces (empty means all namespaces)")
	flags.StringVar(&cfg.DefaultAKSResourceID, "default-aks-resource-id", "", "Default AKS cluster resource ID used when aks_resource_id is not supplied by the caller (e.g. /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{cluster}). Falls back to AZURE_AKS_RESOURCE_ID env var.")
	flags.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flags.StringVar(&cfg.OTLPEndpoint, "otlp-endpoint", "", "OTLP endpoint for OpenTelemetry traces (e.g. localhost:4317)")
	transport := flags.String("transport", "stdio", "Transport type (stdio only; retained for local client compatibility)")

	var showHelp bool
	flags.BoolVarP(&showHelp, "help", "h", false, "Show help message")
	showVersion := flags.Bool("version", false, "Show version information and exit")

	if err := flags.Parse(args); err != nil {
		return flags, false, false, err
	}
	if *transport != "stdio" {
		return flags, false, false, fmt.Errorf("unsupported transport %q: only stdio is supported", *transport)
	}

	cfg.SecurityConfig.AccessLevel = cfg.AccessLevel
	cfg.SecurityConfig.AllowedNamespaces = cfg.AllowNamespaces
	if cfg.DefaultAKSResourceID == "" {
		cfg.DefaultAKSResourceID = os.Getenv("AZURE_AKS_RESOURCE_ID")
	}
	if *enabledComponents != "" {
		for _, component := range strings.Split(*enabledComponents, ",") {
			if component = strings.TrimSpace(component); component != "" {
				cfg.EnabledComponents = append(cfg.EnabledComponents, component)
			}
		}
	}

	return flags, showHelp, *showVersion, nil
}

// InitializeTelemetry initializes the telemetry service.
func (cfg *ConfigData) InitializeTelemetry(ctx context.Context, serviceName, serviceVersion string) {
	telemetryConfig := telemetry.NewConfig(serviceName, serviceVersion)
	if cfg.OTLPEndpoint != "" {
		telemetryConfig.SetOTLPEndpoint(cfg.OTLPEndpoint)
	}
	cfg.TelemetryService = telemetry.NewService(telemetryConfig)
	if err := cfg.TelemetryService.Initialize(ctx); err != nil {
		logger.Errorf("Failed to initialize telemetry: %v", err)
	}
	cfg.TelemetryService.TrackServiceStartup(ctx)
}

// PrintVersion prints version information.
func (cfg *ConfigData) PrintVersion() {
	versionInfo := version.GetVersionInfo()
	fmt.Printf("aks-mcp version %s\n", versionInfo["version"])
	fmt.Printf("Git commit: %s\n", versionInfo["gitCommit"])
	fmt.Printf("Git tree state: %s\n", versionInfo["gitTreeState"])
	fmt.Printf("Go version: %s\n", versionInfo["goVersion"])
	fmt.Printf("Platform: %s\n", versionInfo["platform"])
}
