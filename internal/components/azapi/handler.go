package azapi

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/logger"
	"github.com/Azure/azure-api-mcp/pkg/azcli"
	"github.com/google/shlex"
	"github.com/mark3labs/mcp-go/mcp"
)

// sanitizeCliCommand extracts only the "az <group> <subcommand>" prefix from a
// CLI command, stripping all flags and arguments. This prevents secrets and
// high-cardinality values from leaking into telemetry.
func sanitizeCliCommand(cmd string) string {
	tokens := strings.Fields(cmd)
	var kept []string
	for _, t := range tokens {
		if strings.HasPrefix(t, "-") {
			break
		}
		kept = append(kept, t)
	}
	return strings.Join(kept, " ")
}

// azureHostPattern matches known Azure/Microsoft hostnames.
var azureHostPattern = regexp.MustCompile(`(?i)(^|\.)(azure\.com|azure\.cn|azure\.us|azure\.de|microsoftonline\.com|microsoft\.com|windows\.net|azure-api\.net|azurecr\.io|azurewebsites\.net|azureedge\.net|msecnd\.net|msftauth\.net|msauth\.net|msftidentity\.com|visualstudio\.com|aka\.ms)$`)

// validateAzCommand provides defense-in-depth validation for az CLI commands.
// It blocks "az rest" commands where --url points to a non-Azure host,
// preventing token exfiltration attacks. It also blocks @file arguments across
// all command types to prevent arbitrary local file reads.
func validateAzCommand(cliCommand string) error {
	// Block az CLI @file expansion: any token starting with @ triggers file content
	// substitution in az CLI pre-processing, enabling arbitrary file reads.
	{
		tokens, err := shlex.Split(cliCommand)
		if err == nil {
			for _, token := range tokens {
				if strings.HasPrefix(token, "@") {
					return fmt.Errorf("command contains @file argument which would cause az CLI to read local files; this is blocked as a security measure")
				}
			}
		}
	}

	tokens := strings.Fields(cliCommand)

	// Only inspect "az rest" commands for URL validation
	if len(tokens) < 2 || tokens[0] != "az" || tokens[1] != "rest" {
		return nil
	}

	// Search for --url / -u flag value
	var urlVal string
	var hasURL bool
	for i := 2; i < len(tokens); i++ {
		t := tokens[i]
		if (t == "--url" || t == "--uri" || t == "-u") && i+1 < len(tokens) {
			urlVal = tokens[i+1]
			hasURL = true
			break
		}
		if strings.HasPrefix(t, "--url=") {
			urlVal = strings.TrimPrefix(t, "--url=")
			hasURL = true
			break
		}
		if strings.HasPrefix(t, "--uri=") {
			urlVal = strings.TrimPrefix(t, "--uri=")
			hasURL = true
			break
		}
	}

	if !hasURL {
		return nil
	}

	parsed, err := url.Parse(urlVal)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("az rest --url must point to a known Azure host; non-Azure URLs are blocked to prevent token exfiltration")
	}

	hostname := parsed.Hostname()
	if !azureHostPattern.MatchString(hostname) {
		return fmt.Errorf("az rest --url must point to a known Azure host; non-Azure URLs are blocked to prevent token exfiltration")
	}

	return nil
}

func AzApiHandler(azClient azcli.Client, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			errMsg := fmt.Sprintf("arguments must be a map[string]interface{}, got %T", req.Params.Arguments)
			logger.Errorf("AzApiHandler: %s", errMsg)
			if cfg.TelemetryService != nil {
				cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, "", false)
			}
			return mcp.NewToolResultError(errMsg), nil
		}

		cliCommand, ok := args["cli_command"].(string)
		if !ok {
			errMsg := "missing or invalid 'cli_command' parameter"
			logger.Errorf("AzApiHandler: %s", errMsg)
			if cfg.TelemetryService != nil {
				cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, "", false)
			}
			return mcp.NewToolResultError(errMsg), nil
		}

		timeout := time.Duration(cfg.Timeout) * time.Second
		if timeoutParam, ok := args["timeout"].(float64); ok && timeoutParam > 0 {
			timeout = time.Duration(timeoutParam) * time.Second
		}

		cmdCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		logger.Debugf("AzApiHandler: Executing Azure CLI command: %s", cliCommand)

		if err := validateAzCommand(cliCommand); err != nil {
			errMsg := fmt.Sprintf("command validation failed: %v", err)
			logger.Errorf("AzApiHandler: %s", errMsg)
			if cfg.TelemetryService != nil {
				cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, sanitizeCliCommand(cliCommand), false)
			}
			return mcp.NewToolResultError(errMsg), nil
		}

		result, err := azClient.ExecuteCommand(cmdCtx, cliCommand)
		if err != nil {
			errMsg := fmt.Sprintf("failed to execute Azure CLI command: %v", err)
			logger.Errorf("AzApiHandler: %s", errMsg)
			if cfg.TelemetryService != nil {
				cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, sanitizeCliCommand(cliCommand), false)
			}
			return mcp.NewToolResultError(errMsg), nil
		}

		if result.Error != "" {
			errMsg := fmt.Sprintf("Azure CLI command failed: %s", result.Error)
			logger.Errorf("AzApiHandler: %s", errMsg)
			if cfg.TelemetryService != nil {
				cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, sanitizeCliCommand(cliCommand), false)
			}
			return mcp.NewToolResultError(errMsg), nil
		}

		logger.Debugf("AzApiHandler: Command completed successfully")

		if cfg.TelemetryService != nil {
			cfg.TelemetryService.TrackToolInvocation(ctx, req.Params.Name, sanitizeCliCommand(cliCommand), true)
		}

		return mcp.NewToolResultText(string(result.Output)), nil
	}
}
