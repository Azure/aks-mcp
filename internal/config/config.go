// Package config provides configuration management for AKS MCP server.
package config

import (
	"log"

	flag "github.com/spf13/pflag"
)

// Config holds the configuration for the AKS MCP server.
type Config struct {
	AKSResourceID string
	Transport     string
	Address       string
}

// NewConfig creates a new configuration with default values.
func NewConfig() *Config {
	return &Config{
		Transport: "stdio",
		Address:   "localhost:8080",
	}
}

// ParseFlags parses command-line flags and returns a Config.
func ParseFlags() *Config {
	config := NewConfig()

	flag.StringVarP(&config.Transport, "transport", "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&config.AKSResourceID, "aks-resource-id", "", "AKS Resource ID (required)")
	flag.StringVar(&config.Address, "address", "localhost:8080", "Address to listen on when using SSE transport")
	flag.Parse()

	// Validate required arguments
	if config.AKSResourceID == "" {
		log.Fatal("--aks-resource-id is required")
	}

	return config
}
