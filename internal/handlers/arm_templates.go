// Package handlers provides handler functions for AKS MCP tools.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/azure/aks-mcp/internal/azure"
	"github.com/azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetExampleARMTemplatesHandler returns a handler for the list_aks_example_arm_templates tool.
// It lists all available AKS example ARM templates.
func GetExampleARMTemplatesHandler(client *azure.AzureClient, cache *azure.AzureCache, cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get the examples directory from config
		examplesDir := filepath.Join(cfg.SpecDir, "examples")
		if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("examples directory not found: %s", examplesDir)
		}

		// List all example files in the directory
		files, err := os.ReadDir(examplesDir)
		if err != nil {
			return nil, fmt.Errorf("failed to list example templates: %v", err)
		}

		// Extract template names and descriptions
		templates := []map[string]string{}
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Only include JSON files
			if !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			templateName := file.Name()
			templatePath := filepath.Join(examplesDir, file.Name())

			//Get a short description from the file content
			description := getTemplateDescription(templatePath)

			templates = append(templates, map[string]string{
				"name":        templateName,
				"description": description,
			})
		}

		// Return the templates as JSON
		result := map[string]interface{}{
			"templates": templates,
		}

		jsonResult, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal templates: %v", err)
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	}
}

// GetExampleARMTemplateContentHandler returns a handler for the get_aks_example_arm_template tool.
// It retrieves the content of a specific AKS example ARM template.
func GetExampleARMTemplateContentHandler(client *azure.AzureClient, cache *azure.AzureCache, cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract template name from the request
		templateName, _ := request.GetArguments()["template_name"].(string)

		// Validate required parameters
		if templateName == "" {
			return nil, fmt.Errorf("missing required parameter: template_name")
		}

		// Ensure the template name ends with .json
		if !strings.HasSuffix(templateName, ".json") {
			templateName = templateName + ".json"
		}

		// Get the examples directory from config
		examplesDir := filepath.Join(cfg.SpecDir, "examples")
		if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("examples directory not found: %s", examplesDir)
		}

		// Build the path to the template file
		templatePath := filepath.Join(examplesDir, templateName)

		// Verify the file exists
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("template not found: %s", templateName)
		}

		// Read the template content
		content, err := os.ReadFile(templatePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read template: %v", err)
		}

		// Return the template content as JSON
		result := map[string]interface{}{
			"template_name": templateName,
			"content":       string(content),
		}

		jsonResult, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal template content: %v", err)
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	}
}

// getTemplateDescription extracts a short description from the template file.
func getTemplateDescription(templatePath string) string {
	// Read the template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "Unknown template"
	}

	// Parse the template content
	var template map[string]interface{}
	if err := json.Unmarshal(content, &template); err != nil {
		return filepath.Base(templatePath)
	}

	// Extract relevant information
	templateName := filepath.Base(templatePath)

	// Try to find meaningful description
	description := templateName

	// For example files that follow the standard pattern, derive a description
	if strings.HasPrefix(templateName, "ManagedClustersCreate") {
		description = "Create AKS cluster"

		// Extract specific feature from filename
		feature := strings.TrimPrefix(templateName, "ManagedClustersCreate_")
		feature = strings.TrimSuffix(feature, ".json")

		if feature != "Update" {
			description += " with " + strings.ReplaceAll(feature, "_", " ")
		}
	} else if strings.HasPrefix(templateName, "AgentPoolsCreate") {
		description = "Add node pool"

		// Extract specific feature from filename
		feature := strings.TrimPrefix(templateName, "AgentPoolsCreate_")
		feature = strings.TrimSuffix(feature, ".json")

		if feature != "Update" {
			description += " with " + strings.ReplaceAll(feature, "_", " ")
		}
	}

	return description
}
