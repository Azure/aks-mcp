// Package azure provides Azure-specific functionality for the AKS MCP server.
package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SpecDownloader handles downloading Azure API specifications
type SpecDownloader struct {
	BaseURL     string
	SpecPath    string
	TargetDir   string
	ExamplesDir string
}

// NewSpecDownloader creates a new instance of SpecDownloader
func NewSpecDownloader(specURL string, specDir string) (*SpecDownloader, error) {
	// Parse the GitHub URL to get the raw content URL
	baseURL, specPath, err := parseGitHubURL(specURL)
	if err != nil {
		return nil, err
	}

	// Create the target directories
	targetDir := specDir
	examplesDir := filepath.Join(targetDir, "examples")

	// Create the directories if they don't exist with more restrictive permissions
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %v", err)
	}
	if err := os.MkdirAll(examplesDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create examples directory: %v", err)
	}

	return &SpecDownloader{
		BaseURL:     baseURL,
		SpecPath:    specPath,
		TargetDir:   targetDir,
		ExamplesDir: examplesDir,
	}, nil
}

// DownloadSpecsIfNeeded downloads the Azure API specs if they don't exist locally
func (d *SpecDownloader) DownloadSpecsIfNeeded() error {
	log.Printf("Checking if specs need to be downloaded to %s", d.TargetDir)

	// Check if the main spec file exists
	mainSpecFile := filepath.Join(d.TargetDir, "managedClusters.json")
	if _, err := os.Stat(mainSpecFile); os.IsNotExist(err) {
		// Download the specs since the main file doesn't exist
		if err := d.DownloadSpecs(); err != nil {
			return err
		}
	} else {
		log.Printf("Specs already exist in %s, skipping download", d.TargetDir)
	}

	return nil
}

// DownloadSpecs downloads the Azure API specs
func (d *SpecDownloader) DownloadSpecs() error {
	log.Printf("Downloading Azure API specs from %s/%s", d.BaseURL, d.SpecPath)

	// Download the example files
	examplesURL := fmt.Sprintf("%s/%s/examples", d.BaseURL, d.SpecPath)
	if err := d.downloadExamples(examplesURL); err != nil {
		return fmt.Errorf("failed to download example files: %v", err)
	}

	// Download the main spec file
	mainSpecURL := fmt.Sprintf("%s/%s/managedClusters.json", d.BaseURL, d.SpecPath)
	mainSpecFile := filepath.Join(d.TargetDir, "managedClusters.json")
	if err := downloadFile(mainSpecURL, mainSpecFile); err != nil {
		return fmt.Errorf("failed to download main spec file: %v", err)
	}
	log.Printf("Downloaded main spec file to %s", mainSpecFile)

	log.Printf("Successfully downloaded Azure API specs to %s", d.TargetDir)
	return nil
}

// downloadExamples downloads all example files from the specified URL
func (d *SpecDownloader) downloadExamples(examplesURL string) error {
	// Convert the raw content URL to GitHub API URL for listing files
	apiURL := convertToGitHubApiUrl(examplesURL)
	if apiURL == "" {
		return fmt.Errorf("failed to convert to GitHub API URL")
	}

	log.Printf("Listing example files from GitHub API: %s", apiURL)

	// Make a request to the GitHub API to list files
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Add a user agent to avoid GitHub API rate limiting issues
	req.Header.Set("User-Agent", "AKS-MCP-Agent")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Error closing response body: %v", cerr)
		}
	}()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the JSON response
	var files []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
	}

	if err := json.Unmarshal(body, &files); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	// Download each file
	log.Printf("Found %d files in examples directory", len(files))

	for _, file := range files {
		// Skip directories
		if file.Type != "file" {
			continue
		}

		// Skip files that aren't JSON
		if !strings.HasSuffix(file.Name, ".json") {
			continue
		}

		localPath := filepath.Join(d.ExamplesDir, file.Name)

		// Skip if the file already exists
		if _, err := os.Stat(localPath); err == nil {
			log.Printf("Example file %s already exists, skipping", file.Name)
			continue
		}

		log.Printf("Downloading example file: %s", file.Name)

		// Use the download_url from the API response
		if err := downloadFile(file.DownloadURL, localPath); err != nil {
			log.Printf("Warning: Failed to download example file %s: %v", file.Name, err)
			// Continue with other files even if one fails
			continue
		}
		log.Printf("Downloaded example file to %s", localPath)
	}

	return nil
}

// downloadFile downloads a file from the specified URL to the local path
func downloadFile(url, localPath string) error {
	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Error closing response body: %v", cerr)
		}
	}()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	// Note: We're trusting that the calling code is controlling the localPath input
	// since this is an internal utility function and the localPath is constructed within our code

	// Create the file
	// #nosec G304 -- This is safe as localPath is controlled by our code
	out, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			log.Printf("Error closing output file: %v", cerr)
		}
	}()

	// Write the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// convertToGitHubApiUrl converts a raw content URL to a GitHub API URL for listing repository contents
func convertToGitHubApiUrl(rawURL string) string {
	// Expected format: https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main/specification/...
	if !strings.Contains(rawURL, "raw.githubusercontent.com") {
		return ""
	}

	// Replace the domain
	apiURL := strings.Replace(rawURL, "raw.githubusercontent.com", "api.github.com/repos", 1)

	// Extract the organization, repo, and path
	parts := strings.SplitN(apiURL, "/repos/", 2)
	if len(parts) != 2 {
		return ""
	}

	pathParts := strings.SplitN(parts[1], "/", 3)
	if len(pathParts) < 3 {
		return ""
	}

	org := pathParts[0]
	repo := pathParts[1]
	branch := ""
	path := ""

	// The third part contains both the branch and the path
	remainingPath := pathParts[2]
	branchAndPath := strings.SplitN(remainingPath, "/", 2)
	if len(branchAndPath) == 2 {
		branch = branchAndPath[0]
		path = branchAndPath[1]
	}

	// Construct the API URL for listing directory contents
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		org, repo, path, branch)
}

// parseGitHubURL parses a GitHub URL and returns the raw content URL components
func parseGitHubURL(url string) (string, string, error) {
	// Handle GitHub URLs
	if strings.Contains(url, "github.com") {
		// Convert from web UI URL to raw content URL
		// Example:
		// From: https://github.com/Azure/azure-rest-api-specs/tree/main/specification/containerservice/resource-manager/Microsoft.ContainerService/aks/stable/2025-03-01
		// To: https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main/specification/containerservice/resource-manager/Microsoft.ContainerService/aks/stable/2025-03-01

		// Remove "tree/" to convert to raw URL path
		url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		url = strings.Replace(url, "/tree/", "/", 1)

		// Split into base URL and spec path
		parts := strings.SplitN(url, "raw.githubusercontent.com/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GitHub URL format")
		}

		// Further split the path
		pathParts := strings.SplitN(parts[1], "/", 3)
		if len(pathParts) != 3 {
			return "", "", fmt.Errorf("invalid GitHub path format")
		}

		baseURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s", pathParts[0], pathParts[1])
		specPath := pathParts[2]

		return baseURL, specPath, nil
	}

	return "", "", fmt.Errorf("unsupported URL format")
}
