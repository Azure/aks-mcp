package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestContainerBuild validates that the Docker image builds successfully
// This test requires Docker to be available and working
func TestContainerBuild(t *testing.T) {
	// Skip if running in CI or if Docker not available
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping Docker build test in CI environment")
	}

	// Check if docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping container build test")
	}

	// Build the Docker image from repository root
	cmd := exec.Command("docker", "build", "-t", "aks-mcp:test", "../..")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Docker image: %v\nOutput: %s", err, string(output))
	}
	t.Logf("Docker image built successfully")
}

// TestContainerAzureCLI validates Azure CLI is available inside the container
// This test assumes the Docker image 'aks-mcp:test' exists
func TestContainerAzureCLI(t *testing.T) {
	// Skip if running in CI environment unless image is pre-built
	if !isDockerImageAvailable("aks-mcp:test") {
		t.Skip("Docker image 'aks-mcp:test' not available, skipping Azure CLI test")
	}

	// Check Azure CLI is installed in container
	cmd := exec.Command("docker", "run", "--rm", "--entrypoint", "sh", "aks-mcp:test", "-c", "which az && az --version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Azure CLI not available in container: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "/usr/local/bin/az") && !strings.Contains(outputStr, "/usr/bin/az") {
		t.Fatalf("Azure CLI not found in expected location. Output: %s", outputStr)
	}

	if !strings.Contains(outputStr, "azure-cli") {
		t.Fatalf("Azure CLI version not displayed properly. Output: %s", outputStr)
	}

	t.Logf("Azure CLI successfully installed in container")
}

// TestContainerNetworkTransport validates the container starts with correct transport and network configuration
func TestContainerNetworkTransport(t *testing.T) {
	// Skip if Docker image not available
	if !isDockerImageAvailable("aks-mcp:test") {
		t.Skip("Docker image 'aks-mcp:test' not available, skipping network transport test")
	}

	// Start container with streamable-http transport. After the DNS-rebinding
	// fail-closed validator landed, the default CMD (--host 0.0.0.0 with no
	// auth and no --allowed-host) is intentionally rejected at startup, so
	// this test supplies --allowed-host=localhost to satisfy the validator
	// while still exercising the HTTP listener on localhost.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start container in background
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-p", "8000:8000",
		"aks-mcp:test",
		"--transport", "streamable-http",
		"--host", "0.0.0.0",
		"--allowed-host", "localhost",
	)

	// Start the container
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Ensure container is stopped when test completes
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// Give container time to start
	time.Sleep(5 * time.Second)

	// Try to connect to the streamable-http endpoint
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8000")
	if err != nil {
		t.Fatalf("Could not connect to container service: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	// Read a small amount of response to verify service is responding
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	t.Logf("Container service is accessible at localhost:8000, status: %d", resp.StatusCode)
	t.Logf("Response preview: %s", string(body)[:min(len(body), 100)])
}

// TestContainerConfiguration validates container environment and configuration
func TestContainerConfiguration(t *testing.T) {
	// Skip if Docker image not available
	if !isDockerImageAvailable("aks-mcp:test") {
		t.Skip("Docker image 'aks-mcp:test' not available, skipping configuration test")
	}

	// Test container has correct user and working directory
	cmd := exec.Command("docker", "run", "--rm", "--entrypoint", "sh", "aks-mcp:test", "-c", "whoami && pwd && echo $HOME")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to check container configuration: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")

	if len(lines) < 3 {
		t.Fatalf("Unexpected output format: %s", outputStr)
	}

	user := strings.TrimSpace(lines[0])
	workDir := strings.TrimSpace(lines[1])
	homeDir := strings.TrimSpace(lines[2])

	if user != "mcp" {
		t.Errorf("Expected user 'mcp', got '%s'", user)
	}

	if workDir != "/home/mcp" {
		t.Errorf("Expected working directory '/home/mcp', got '%s'", workDir)
	}

	if homeDir != "/home/mcp" {
		t.Errorf("Expected HOME directory '/home/mcp', got '%s'", homeDir)
	}

	t.Logf("Container configuration correct: user=%s, workdir=%s, home=%s", user, workDir, homeDir)
}

// TestContainerHelp validates the application responds to help command
func TestContainerHelp(t *testing.T) {
	// Skip if Docker image not available
	if !isDockerImageAvailable("aks-mcp:test") {
		t.Skip("Docker image 'aks-mcp:test' not available, skipping help test")
	}

	cmd := exec.Command("docker", "run", "--rm", "aks-mcp:test", "--help")
	output, err := cmd.CombinedOutput()

	// pflag returns exit code 2 for help requests, which is expected behavior
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 2 {
				t.Fatalf("Help command failed with unexpected exit code %d: %v\nOutput: %s", exitError.ExitCode(), err, string(output))
			}
		} else {
			t.Fatalf("Help command failed: %v\nOutput: %s", err, string(output))
		}
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "transport") || !strings.Contains(outputStr, "host") {
		t.Fatalf("Help output doesn't contain expected flags. Output: %s", outputStr)
	}

	t.Logf("Container help command works correctly")
}

// TestDockerfileConfiguration validates the Dockerfile configuration without building
func TestDockerfileConfiguration(t *testing.T) {
	// Read and validate Dockerfile content
	dockerfile, err := os.ReadFile("../../Dockerfile")
	if err != nil {
		t.Fatalf("Failed to read Dockerfile: %v", err)
	}

	content := string(dockerfile)

	// Validate Azure CLI installation
	if !strings.Contains(content, "pip3 install --break-system-packages --no-cache-dir azure-cli") {
		t.Error("Dockerfile missing Azure CLI installation")
	}

	// Validate required packages for Azure CLI
	if !strings.Contains(content, "gcc python3-dev musl-dev linux-headers") {
		t.Error("Dockerfile missing build dependencies for Azure CLI")
	}

	// Validate transport configuration
	if !strings.Contains(content, "streamable-http") {
		t.Error("Dockerfile not using streamable-http transport")
	}

	// Validate network binding
	if !strings.Contains(content, "0.0.0.0") {
		t.Error("Dockerfile not binding to all network interfaces")
	}

	// Validate port exposure
	if !strings.Contains(content, "EXPOSE 8000") {
		t.Error("Dockerfile not exposing port 8000")
	}

	// Validate user configuration
	if !strings.Contains(content, "USER mcp") {
		t.Error("Dockerfile not using non-root user")
	}

	t.Logf("Dockerfile configuration validated successfully")
}

// isDockerImageAvailable checks if a Docker image is available locally
func isDockerImageAvailable(imageName string) bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}

	cmd := exec.Command("docker", "image", "inspect", imageName)
	err := cmd.Run()
	return err == nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
