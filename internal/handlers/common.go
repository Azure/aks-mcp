// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"encoding/json"
	"fmt"
)

// formatJSON is a utility function to format an object as a JSON string
func formatJSON(v interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %v", err)
	}
	return string(jsonBytes), nil
}
