package config

import (
	"github.com/Azure/aks-mcp/internal/security"
	flag "github.com/spf13/pflag"
)

// ConfigData holds the global configuration
type ConfigData struct {
	// Command execution timeout in seconds
	Timeout int
	// Security configuration
	SecurityConfig *security.SecurityConfig

	// Command-line specific options
	Transport   string
	Port        int
	AccessLevel string
}

// NewConfig creates and returns a new configuration instance
func NewConfig() *ConfigData {
	return &ConfigData{
		Timeout:        60,
		SecurityConfig: security.NewSecurityConfig(),
		Transport:      "stdio",
		Port:           8000,
		AccessLevel:    "readwrite",
	}
}

// ParseFlags parses command line arguments and updates the configuration
func (cfg *ConfigData) ParseFlags() {
	// Server configuration
	flag.StringVar(&cfg.Transport, "transport", "stdio", "Transport mechanism to use (stdio or sse)")
	flag.IntVar(&cfg.Port, "port", 8000, "Port to use for the server (only used with sse transport)")
	flag.IntVar(&cfg.Timeout, "timeout", 600, "Timeout for command execution in seconds, default is 600s")
	// Security settings
	flag.StringVar(&cfg.AccessLevel, "access-level", "readwrite", "Access level (readonly, readwrite, admin)")

	flag.Parse()

	// Update security config
	cfg.SecurityConfig.AccessLevel = cfg.AccessLevel
}
