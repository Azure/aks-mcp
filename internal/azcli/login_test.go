package azcli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

// Command to be run and expected output/error
type loginCommandResponses struct {
	cmd string
	out string
	err error
}

// Interface to mock azcli commands in tests
type loginCommands struct {
	idx  int
	resp []loginCommandResponses
}

// Simulate running a command and return the expected output/error
func (c *loginCommands) Run(cmd string) (string, error) {
	if c.idx >= len(c.resp) {
		return "", fmt.Errorf("no more responses, unexpected command: %s", cmd)
	}
	expected := c.resp[c.idx]
	c.idx++
	// match prefix so tests are less brittle
	if expected.cmd != "" && !strings.HasPrefix(cmd, expected.cmd) {
		return "", fmt.Errorf("expected cmd prefix %q but got %q", expected.cmd, cmd)
	}
	return expected.out, expected.err
}

func TestEnsureAzCliLogin_Existing(t *testing.T) {
	cfg := config.NewConfig()
	// ensure no env-based auth triggers; default to existing login
	t.Setenv("AZURE_CLIENT_ID", "")
	t.Setenv("AZURE_CLIENT_SECRET", "")
	t.Setenv("AZURE_TENANT_ID", "")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "")

	// proc should not be invoked
	p := &loginCommands{resp: []loginCommandResponses{}}

	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "existing_login" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_ServicePrincipal(t *testing.T) {
	cfg := config.NewConfig()
	// set envs for service principal (dummy values for testing only)
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_CLIENT_SECRET", "dummy-client-secret")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	// To exercise the login flow we must simulate probe failing (not logged in)
	p := &loginCommands{resp: []loginCommandResponses{
		// login command succeeds (matches dummy env values)
		{cmd: "login --service-principal -u dummy-client-id -p dummy-client-secret --tenant dummy-tenant-id", out: "", err: nil},
		// probe after login succeeds
		{cmd: "account show --query id -o tsv", out: "sub-id", err: nil},
	}}
	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "service_principal" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_ServicePrincipal_ErrorOutput(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_CLIENT_SECRET", "dummy-client-secret")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	// probe fails so login attempt is performed
	p := &loginCommands{resp: []loginCommandResponses{
		// login returns an ERROR: prefixed message on stderr (simulated via out)
		{cmd: "login --service-principal -u dummy-client-id -p dummy-client-secret --tenant dummy-tenant-id", out: "ERROR: something went wrong", err: nil},
	}}
	_, err := EnsureAzCliLoginWithProc(p, cfg)
	if err == nil || !strings.Contains(err.Error(), "service principal login failed: ERROR: something went wrong") {
		t.Fatalf("expected service principal error containing ERROR:, got %v", err)
	}
}

func TestEnsureAzCliLogin_ServicePrincipal_CommandError(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_CLIENT_SECRET", "dummy-client-secret")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	p := &loginCommands{resp: []loginCommandResponses{
		// login command fails to execute
		{cmd: "login --service-principal -u dummy-client-id -p dummy-client-secret --tenant dummy-tenant-id", out: "", err: errors.New("exec failed")},
	}}
	_, err := EnsureAzCliLoginWithProc(p, cfg)
	if err == nil || !strings.Contains(err.Error(), "service principal login failed") {
		t.Fatalf("expected wrapped service principal error, got %v", err)
	}
}

func TestEnsureAzCliLogin_NoAutoLogin(t *testing.T) {
	cfg := config.NewConfig()
	// ensure no AZURE_* env vars (explicitly clear for this test)
	t.Setenv("AZURE_CLIENT_ID", "")
	t.Setenv("AZURE_CLIENT_SECRET", "")
	t.Setenv("AZURE_TENANT_ID", "")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "")

	// should default to existing login without invoking proc
	p := &loginCommands{resp: []loginCommandResponses{}}
	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "existing_login" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_SubscriptionSetFailure(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_CLIENT_SECRET", "dummy-client-secret")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "dummy-subscription-id")

	p := &loginCommands{resp: []loginCommandResponses{
		{cmd: "login --service-principal -u dummy-client-id -p dummy-client-secret --tenant dummy-tenant-id", out: "", err: nil},
		// account set fails
		{cmd: "account set --subscription dummy-subscription-id", out: "", err: errors.New("set failed")},
	}}

	_, err := EnsureAzCliLoginWithProc(p, cfg)
	if err == nil || !strings.Contains(err.Error(), "service principal set subscription failed") {
		t.Fatalf("expected subscription set failure wrapped in service principal context, got %v", err)
	}
}

func TestEnsureAzCliLogin_ReprobeFailure(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_CLIENT_SECRET", "dummy-client-secret")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "dummy-subscription-id")

	p := &loginCommands{resp: []loginCommandResponses{
		{cmd: "login --service-principal -u dummy-client-id -p dummy-client-secret --tenant dummy-tenant-id", out: "", err: nil},
		{cmd: "account set --subscription dummy-subscription-id", out: "", err: nil},
		// re-probe fails
		{cmd: "account show --query id -o tsv", out: "", err: errors.New("probe failed")},
	}}

	_, err := EnsureAzCliLoginWithProc(p, cfg)
	if err == nil || !strings.Contains(err.Error(), "service principal account verification failed") {
		t.Fatalf("expected reprobe failure wrapped in service principal context, got %v", err)
	}
}

func TestEnsureAzCliLogin_Federated(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")

	// Only allow the fixed AKS federated token path
	const allowedTokenPath = "/var/run/secrets/azure/tokens/azure-identity-token"
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", allowedTokenPath)

	// If the file does not exist (not running in AKS), skip the test
	if _, err := os.Stat(allowedTokenPath); err != nil {
		t.Skipf("skipping: %s not present (only available in AKS)", allowedTokenPath)
	}

	p := &loginCommands{resp: []loginCommandResponses{}}

	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil && !strings.Contains(err.Error(), allowedTokenPath) {
		t.Fatalf("unexpected error: %v", err)
	}
	if err == nil && got == "federated_token" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_Federated_AKSManagedWebhookPath(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")

	// AKS managed webhook v1.6.0-alpha.1+ injects token at this path
	const aksWebhookTokenPath = "/var/run/secrets/azure/wi/token/azure-identity-token"
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", aksWebhookTokenPath)

	// If the file does not exist (not running in AKS with managed webhook), skip the test
	if _, err := os.Stat(aksWebhookTokenPath); err != nil {
		t.Skipf("skipping: %s not present (only available in AKS with managed webhook v1.6.0-alpha.1+)", aksWebhookTokenPath)
	}

	p := &loginCommands{resp: []loginCommandResponses{}}

	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil && !strings.Contains(err.Error(), aksWebhookTokenPath) {
		t.Fatalf("unexpected error: %v", err)
	}
	if err == nil && got == "federated_token" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_Federated_InvalidFile(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/tmp/non-existent-file")

	p := &loginCommands{resp: []loginCommandResponses{}}

	_, err := EnsureAzCliLoginWithProc(p, cfg)
	if err == nil || !strings.Contains(err.Error(), "federated token file validation failed") {
		t.Fatalf("expected federated token file validation error, got %v", err)
	}
}

func TestEnsureAzCliLogin_Federated_DirectoryTraversal(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "../../../etc/passwd")

	p := &loginCommands{resp: []loginCommandResponses{
		{cmd: "account show --query id -o tsv", out: "", err: errors.New("not logged in")},
	}}

	_, err := EnsureAzCliLoginWithProc(p, cfg)
	if err == nil || !strings.Contains(err.Error(), "federated token file validation failed") {
		t.Fatalf("expected federated token file validation error for directory traversal, got %v", err)
	}
}

func TestEnsureAzCliLogin_Federated_Success(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-client-id")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant-id")

	// Only allow the fixed AKS federated token path
	const allowedTokenPath = "/var/run/secrets/azure/tokens/azure-identity-token"
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", allowedTokenPath)

	// If the file does not exist (not running in AKS), skip the test
	tokenData := "dummy-federated-token"
	if _, err := os.Stat(allowedTokenPath); err != nil {
		t.Skipf("skipping: %s not present (only available in AKS)", allowedTokenPath)
	}

	// Optionally, try to write a dummy token if running in a testable AKS env (may require root)
	// _ = os.WriteFile(allowedTokenPath, []byte(tokenData), 0600)

	p := &loginCommands{resp: []loginCommandResponses{
		{cmd: "login --service-principal -u dummy-client-id --tenant dummy-tenant-id --federated-token " + tokenData, out: "", err: nil},
		{cmd: "account show --query id -o tsv", out: "sub-id", err: nil},
	}}

	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "federated_token" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_ManagedIdentity_UserAssigned(t *testing.T) {
	cfg := config.NewConfig()
	t.Setenv("AZURE_CLIENT_ID", "dummy-managed-identity-client-id")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "dummy-subscription-id")

	p := &loginCommands{resp: []loginCommandResponses{
		{cmd: "login --identity --client-id dummy-managed-identity-client-id", out: "", err: nil},
		{cmd: "account set --subscription dummy-subscription-id", out: "", err: nil},
		{cmd: "account show --query id -o tsv", out: "sub-id", err: nil},
	}}

	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "user_assigned_managed_identity" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_ManagedIdentity_SystemAssigned(t *testing.T) {
	cfg := config.NewConfig()
	// Trigger system-assigned MI via env
	t.Setenv("AZURE_CLIENT_ID", "")
	t.Setenv("AZURE_CLIENT_SECRET", "")
	t.Setenv("AZURE_TENANT_ID", "")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "")
	t.Setenv("AZURE_MANAGED_IDENTITY", "system")

	p := &loginCommands{resp: []loginCommandResponses{
		{cmd: "login --identity", out: "", err: nil},
		{cmd: "account show --query id -o tsv", out: "sub-id", err: nil},
	}}

	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "system_assigned_managed_identity" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_ManagedIdentity_SystemAssigned_Success(t *testing.T) {
	cfg := config.NewConfig()
	// System-assigned MI with subscription set via env flag
	t.Setenv("AZURE_CLIENT_ID", "")
	t.Setenv("AZURE_CLIENT_SECRET", "")
	t.Setenv("AZURE_TENANT_ID", "")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "dummy-subscription-id")
	t.Setenv("AZURE_MANAGED_IDENTITY", "system")

	p := &loginCommands{resp: []loginCommandResponses{
		{cmd: "login --identity", out: "", err: nil},
		{cmd: "account set --subscription dummy-subscription-id", out: "", err: nil},
		{cmd: "account show --query id -o tsv", out: "sub-id", err: nil},
	}}

	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "system_assigned_managed_identity" {
		t.Fatalf("unexpected result: %s", got)
	}
}

func TestEnsureAzCliLogin_LoginPromptInOutput(t *testing.T) {
	cfg := config.NewConfig()
	// ensure auto-login is attempted via user-assigned MI
	t.Setenv("AZURE_CLIENT_ID", "cid")

	// Simulate the case where Azure CLI returns "Please run 'az login'" message with an error
	// After the command.go fix, stderr content is returned WITH the error, not instead of it
	// This test ensures we don't incorrectly think there's a valid login when there isn't
	p := &loginCommands{resp: []loginCommandResponses{
		{cmd: "login --identity --client-id cid", out: "", err: nil},
		{cmd: "account show --query id -o tsv", out: "sub-id", err: nil},
	}}

	// should NOT return existing_login, should proceed with authentication
	got, err := EnsureAzCliLoginWithProc(p, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "user_assigned_managed_identity" {
		t.Fatalf("unexpected result: %s", got)
	}
}

// TestValidateFederatedTokenFile_Allowlist tests the validateFederatedTokenFile function
// directly to ensure the allowlist accepts both known AKS webhook paths and rejects unknown ones.
// These tests use temp files to simulate the token paths without needing a real AKS environment.
func TestValidateFederatedTokenFile_Allowlist(t *testing.T) {
	// Create a temp regular file to stand in for the token file.
	tmpFile, err := os.CreateTemp(t.TempDir(), "token")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	realTmpPath := tmpFile.Name()

	// Build a table of (path, wantErr) entries. For the two allowed paths we
	// point allowedFederatedTokenPaths at the temp file so os.Stat succeeds.
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "unknown path is rejected",
			path:    "/tmp/non-existent-file",
			wantErr: true,
		},
		{
			name:    "directory traversal is rejected",
			path:    "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "arbitrary absolute path is rejected",
			path:    "/etc/shadow",
			wantErr: true,
		},
		{
			name:    "empty path is rejected",
			path:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateFederatedTokenFile(tc.path)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for path %q but got none", tc.path)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for path %q: %v", tc.path, err)
			}
		})
	}

	// Test that each path in the allowlist is accepted when the file exists.
	// We temporarily override allowedFederatedTokenPaths with a path pointing
	// to the real temp file so os.Stat() succeeds.
	origPaths := allowedFederatedTokenPaths
	defer func() { allowedFederatedTokenPaths = origPaths }()

	for _, allowedPath := range origPaths {
		t.Run("allowlist path accepted: "+allowedPath, func(t *testing.T) {
			// Redirect the single entry under test to the temp file.
			allowedFederatedTokenPaths = []string{realTmpPath}
			got, err := validateFederatedTokenFile(realTmpPath)
			if err != nil {
				t.Errorf("expected allowed path %q to be accepted, got error: %v", allowedPath, err)
			}
			if got != realTmpPath {
				t.Errorf("expected returned path %q, got %q", realTmpPath, got)
			}
		})
	}
}
