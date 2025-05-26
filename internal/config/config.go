// Package config provides configuration management for AKS MCP server.
package config

import (
	"fmt"
	"os"

	"github.com/azure/aks-mcp/internal/azure"
	flag "github.com/spf13/pflag"
)

// Config holds the configuration for the AKS MCP server.
type Config struct {
	ResourceIDString  string // Raw resource ID string from command line
	Transport         string
	Address           string
	SingleClusterMode bool
	ParsedResourceID  *azure.AzureResourceID // Parsed version of the resource ID
	AccessLevel       string
}

// NewConfig creates a new configuration with default values.
func NewConfig() *Config {
	return &Config{
		Transport:         "stdio",
		Address:           "localhost:8080",
		SingleClusterMode: false,
		ParsedResourceID:  nil,
		AccessLevel:       "read",
	}
}

// ParseFlags parses command-line flags and returns a Config.
func ParseFlags() *Config {
	config := NewConfig()

	flag.StringVarP(&config.Transport, "transport", "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&config.ResourceIDString, "aks-resource-id", "", "AKS Resource ID (optional), set this when using single cluster mode")
	flag.StringVar(&config.Address, "address", "localhost:8080", "Address to listen on when using SSE transport")
	flag.StringVar(&config.AccessLevel, "access-level", "read", "Access level for tools (read, readwrite, admin)")
	flag.Parse()

	// Set SingleClusterMode based on whether ResourceIDString is provided
	config.SingleClusterMode = config.ResourceIDString != ""

	return config
}

// Validate checks the configuration values and returns an error if any are invalid.
func (c *Config) Validate() error {
	// Validate AccessLevel
	validAccessLevels := map[string]bool{
		"read":      true,
		"readwrite": true,
		"admin":     true,
	}

	if !validAccessLevels[c.AccessLevel] {
		return fmt.Errorf("invalid access level: %s, must be one of read, readwrite, admin", c.AccessLevel)
	}

	// Validate Transport
	validTransports := map[string]bool{
		"stdio": true,
		"sse":   true,
	}

	if !validTransports[c.Transport] {
		return fmt.Errorf("invalid transport: %s, must be either stdio or sse", c.Transport)
	}

	// Validate Address if using SSE transport
	if c.Transport == "sse" && c.Address == "" {
		return fmt.Errorf("address must be specified when using SSE transport")
	}

	// Parse and validate AKS resource ID if provided
	if c.ResourceIDString != "" {
		resourceID, err := azure.ParseAzureResourceID(c.ResourceIDString)
		if err != nil {
			return fmt.Errorf("invalid AKS resource ID: %v", err)
		}
		c.ParsedResourceID = resourceID
	}

	// Validate ResourceIDString if in single cluster mode
	if c.SingleClusterMode && c.ParsedResourceID == nil {
		return fmt.Errorf("invalid or missing AKS resource ID in single cluster mode")
	}

	return nil
}

// ValidateAndParseFlags parses command-line flags, validates the config, and returns it.
// If validation fails, it logs the error and exits.
func ValidateAndParseFlags() *Config {
	config := ParseFlags()

	if err := config.Validate(); err != nil {
		// This will be shown to the user
		fmt.Printf("Configuration error: %v\n", err)
		// Show usage information
		flag.Usage()
		// We use 2 as the exit code for configuration errors
		os.Exit(2)
	}

	return config
}

// GetResourceID is a helper method that returns the parsed resource ID.
// It ensures that the resource ID is parsed if it is provided but not yet parsed.
func (c *Config) GetResourceID() *azure.AzureResourceID {
	// If we have a resource ID string but not a parsed version, parse it
	if c.ParsedResourceID == nil && c.ResourceIDString != "" {
		c.ParsedResourceID, _ = azure.ParseAzureResourceID(c.ResourceIDString)
	}
	return c.ParsedResourceID
}
