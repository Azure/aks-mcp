// Package config provides configuration management for AKS MCP server.
package config

import (
	"github.com/azure/aks-mcp/internal/azure"
	flag "github.com/spf13/pflag"
)

// Config holds the configuration for the AKS MCP server.
type Config struct {
	AKSResourceID     string
	Transport         string
	Address           string
	SingleClusterMode bool
	ResourceID        *azure.AzureResourceID
}

// NewConfig creates a new configuration with default values.
func NewConfig() *Config {
	return &Config{
		Transport:         "stdio",
		Address:           "localhost:8080",
		SingleClusterMode: false,
		ResourceID:        nil,
	}
}

// ParseFlags parses command-line flags and returns a Config.
func ParseFlags() *Config {
	config := NewConfig()

	flag.StringVarP(&config.Transport, "transport", "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&config.AKSResourceID, "aks-resource-id", "", "AKS Resource ID (optional), set this when using single cluster mode")
	flag.StringVar(&config.Address, "address", "localhost:8080", "Address to listen on when using SSE transport")
	flag.Parse()

	// Set SingleClusterMode based on whether AKSResourceID is provided
	config.SingleClusterMode = config.AKSResourceID != ""

	// Parse resource ID if provided
	if config.AKSResourceID != "" {
		resourceID, err := azure.ParseAzureResourceID(config.AKSResourceID)
		if err != nil {
			// Log the error but continue - we'll let the main function handle this
			// to maintain consistent error handling
			return config
		}
		config.ResourceID = resourceID
	}

	return config
}
