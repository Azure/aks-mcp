// This includes automatic login detection and supports multiple authentication methods
// including service principals, managed identities, and federated tokens.
package azcli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Azure/aks-mcp/internal/command"
	"github.com/Azure/aks-mcp/internal/config"
)

// Login types
const (
	AuthTypeExisting                = "existing_login"
	AuthTypeServicePrincipal        = "service_principal"
	AuthTypeFederatedToken          = "federated_token"
	AuthTypeUserAssignedManagedID   = "user_assigned_managed_identity"
	AuthTypeSystemAssignedManagedID = "system_assigned_managed_identity"
)

// allowedFederatedTokenPaths is the allowlist of accepted federated token file paths.
// Two paths are supported:
//   - /var/run/secrets/azure/tokens/azure-identity-token  (open-source AKS webhook)
//   - /var/run/secrets/azure/wi/token/azure-identity-token (AKS managed webhook v1.6.0-alpha.1+)
var allowedFederatedTokenPaths = []string{ // #nosec G101 -- not credentials, these are fixed AKS token paths
	"/var/run/secrets/azure/tokens/azure-identity-token",
	"/var/run/secrets/azure/wi/token/azure-identity-token",
}

// validateFederatedTokenFile only allows paths from the fixed AKS identity token allowlist.
func validateFederatedTokenFile(filePath string) (string, error) {
	allowed := false
	for _, p := range allowedFederatedTokenPaths {
		if filePath == p {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("federated token file path must be one of the allowed AKS token paths")
	}
	fileInfo, err := os.Stat(filePath) // #nosec G304,G703 -- filePath is validated against a fixed allowlist above
	if err != nil {
		return "", fmt.Errorf("cannot stat federated token file %s: %w", filePath, err)
	}
	if !fileInfo.Mode().IsRegular() {
		return "", fmt.Errorf("federated token file is not a regular file: %s", filePath)
	}
	return filePath, nil
}

// Proc is a minimal interface used by this package so tests can inject a fake process.
type Proc interface {
	// Run executes the given command (arguments may be included in the string) and
	// returns the command output and an error. Implementations MUST return the
	// combined stdout+stderr output in the string return value. The error should
	// be non-nil when the underlying process fails to start or returns a
	// non-zero exit status. This contract is relied upon by callers that inspect
	// output text (for example searching for an "ERROR:" prefix) as well as
	// the returned error value.
	Run(cmd string) (string, error)
}

// EnsureAzCliLogin ensures az CLI is available and attempts to auto-login using environment variables
func EnsureAzCliLogin(cfg *config.ConfigData) (string, error) {
	if _, err := exec.LookPath("az"); err != nil {
		return "", fmt.Errorf("az cli is not installed or not in PATH: %w", err)
	}
	proc := NewShellProc(cfg.Timeout)
	return EnsureAzCliLoginWithProc(proc, cfg)
}

// NewShellProc is a package-level Proc factory used so tests can override process creation and avoid invoking the real `az` binary.
var NewShellProc = func(timeout int) Proc {
	return command.NewShellProcess("az", timeout)
}

// EnsureAzCliLoginWithProc is the testable implementation that uses an injected Proc.
func EnsureAzCliLoginWithProc(proc Proc, cfg *config.ConfigData) (string, error) {
	// Read environment variables to determine which auth methods to try
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	federatedTokenFile := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	azureManagedIdentityType := os.Getenv("AZURE_MANAGED_IDENTITY")

	// 1) Service Principal with secret
	if clientID != "" && clientSecret != "" && tenantID != "" {
		if err := runLoginCommand(proc, fmt.Sprintf("login --service-principal -u %s -p %s --tenant %s", clientID, clientSecret, tenantID), "service principal"); err != nil {
			return "", err
		}
		if err := setSubscription(proc, subscriptionID, "service principal"); err != nil {
			return "", err
		}
		if err := showAccount(proc, "service principal"); err != nil {
			return "", err
		}
		return AuthTypeServicePrincipal, nil
	}

	// 2) Workload Identity (federated token)
	if clientID != "" && tenantID != "" && federatedTokenFile != "" {
		// Validate the federated token file path for security and get canonical path
		validatedPath, err := validateFederatedTokenFile(federatedTokenFile)
		if err != nil {
			return "", fmt.Errorf("federated token file validation failed: %w", err)
		}

		// Open the only allowed federated token file (fixed path, safe)
		f, err := os.Open(validatedPath) // #nosec G304,G703 -- validated fixed path, not user-controlled
		if err != nil {
			return "", fmt.Errorf("failed to open federated token file %s: %w", validatedPath, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "error closing file: %v\n", err)
			}
		}()

		// Limit token size to 16KB which is far larger than typical JWT or k8s tokens
		// but protects against very large files.
		const maxTokenSize = 16 * 1024
		data, err := io.ReadAll(io.LimitReader(f, maxTokenSize))
		if err != nil {
			return "", fmt.Errorf("failed to read federated token file %s: %w", validatedPath, err)
		}
		federatedToken := strings.TrimSpace(string(data))
		if federatedToken == "" {
			return "", fmt.Errorf("federated token file %s is empty", validatedPath)
		}
		if err := runLoginCommand(proc, fmt.Sprintf("login --service-principal -u %s --tenant %s --federated-token %s", clientID, tenantID, federatedToken), "federated token"); err != nil {
			return "", err
		}
		if err := setSubscription(proc, subscriptionID, "federated token"); err != nil {
			return "", err
		}
		if err := showAccount(proc, "federated token"); err != nil {
			return "", err
		}
		return AuthTypeFederatedToken, nil
	}

	// 3) User-assigned Managed Identity (client ID provided)
	if clientID != "" {
		if err := runLoginCommand(proc, fmt.Sprintf("login --identity --client-id %s", clientID), "user-assigned managed identity"); err != nil {
			return "", err
		}
		if err := setSubscription(proc, subscriptionID, "user-assigned managed identity"); err != nil {
			return "", err
		}
		if err := showAccount(proc, "user-assigned managed identity"); err != nil {
			return "", err
		}
		return AuthTypeUserAssignedManagedID, nil
	}

	// 4) System-assigned Managed Identity
	if azureManagedIdentityType == "system" {
		if err := runLoginCommand(proc, "login --identity", "system-assigned managed identity"); err != nil {
			return "", err
		}
		if err := setSubscription(proc, subscriptionID, "system-assigned managed identity"); err != nil {
			return "", err
		}
		if err := showAccount(proc, "system-assigned managed identity"); err != nil {
			return "", err
		}
		return AuthTypeSystemAssignedManagedID, nil
	}

	// 5) Existing login (no credentials override provided)
	return AuthTypeExisting, nil
}

// Runs a command and returns a formatted error if the output indicates an ERROR or the command failed.
func runLoginCommand(proc Proc, cmd string, loginMethod string) error {
	// proc.Run is expected to return combined stdout+stderr. Some `az`
	// subcommands write human-readable error text to stderr (which will be
	// included in the returned string). We therefore inspect the returned
	// output for an "ERROR:" prefix as a quick check for failure messages,
	// and also respect the returned error (which should be non-nil for
	// non-zero exit codes).
	out, err := proc.Run(cmd)
	if strings.HasPrefix(strings.TrimSpace(out), "ERROR:") {
		return fmt.Errorf("%s login failed: %s", loginMethod, out)
	}
	if err != nil {
		return fmt.Errorf("%s login failed: %w; output: %s", loginMethod, err, out)
	}
	return nil
}

// Sets the subscription when provided and wraps errors with context.
func setSubscription(proc Proc, subscriptionID, loginMethod string) error {
	if subscriptionID == "" {
		return nil
	}
	out, err := proc.Run(fmt.Sprintf("account set --subscription %s", subscriptionID))
	if err != nil {
		return fmt.Errorf("%s set subscription failed: %w; output: %s", loginMethod, err, out)
	}
	return nil
}

// Checks that account info is returned after a login attempt.
func showAccount(proc Proc, loginMethod string) error {
	out, err := proc.Run("account show --query id -o tsv")
	if err != nil {
		return fmt.Errorf("%s account verification failed: %w; output: %s", loginMethod, err, out)
	}
	return nil
}
