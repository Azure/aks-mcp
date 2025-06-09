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

		// Build the path to the template file and prevent path traversal
		templatePath := filepath.Join(examplesDir, templateName)

		// Ensure the path is still inside the examples directory to prevent path traversal
		resolvedPath, err := filepath.EvalSymlinks(templatePath)
		if err != nil {
			return nil, fmt.Errorf("invalid template path: %v", err)
		}

		// Check that the resolved path is still within the examples directory
		if !strings.HasPrefix(resolvedPath, examplesDir) {
			return nil, fmt.Errorf("invalid template path: path traversal detected")
		}

		// Verify the file exists
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("template not found: %s", templateName)
		}

		// Read the template content
		// #nosec G304 -- This is safe as we check that resolvedPath is within the allowed directory
		content, err := os.ReadFile(resolvedPath)
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
	// Extract relevant information
	templateName := filepath.Base(templatePath)

	// TODO: Try to find meaningful description
	description := templateName

	return description
}
