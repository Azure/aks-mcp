// Package k8s provides adapters that let aks-mcp interoperate with the
// mcp-kubernetes libraries. It maps aks-mcp configuration and executors
// to the types expected by mcp-kubernetes without altering behavior.
package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
	k8ssecurity "github.com/Azure/mcp-kubernetes/pkg/security"
	k8stelemetry "github.com/Azure/mcp-kubernetes/pkg/telemetry"
	k8stools "github.com/Azure/mcp-kubernetes/pkg/tools"
	"github.com/google/shlex"
)

// ConvertConfig maps an aks-mcp ConfigData into the equivalent
// mcp-kubernetes ConfigData without mutating the input.
func ConvertConfig(cfg *config.ConfigData) *k8sconfig.ConfigData {
	k8sSecurityConfig := k8ssecurity.NewSecurityConfig()
	k8sSecurityConfig.SetAllowedNamespaces(cfg.AllowNamespaces)
	k8sSecurityConfig.AccessLevel = k8ssecurity.AccessLevel(cfg.AccessLevel)

	// Convert EnabledComponents []string to AdditionalTools map[string]bool
	// This is needed for compatibility with mcp-kubernetes which still uses the map format
	additionalTools := make(map[string]bool)

	// Only convert Kubernetes-related components (helm, cilium, hubble)
	// If EnabledComponents is empty, enable all additional tools
	if len(cfg.EnabledComponents) == 0 {
		// Empty list means all components enabled
		additionalTools["helm"] = true
		additionalTools["cilium"] = true
		additionalTools["hubble"] = true
	} else {
		// Check which Kubernetes components are enabled
		for _, component := range cfg.EnabledComponents {
			switch component {
			case "helm", "cilium", "hubble":
				additionalTools[component] = true
			}
		}
	}

	k8sCfg := &k8sconfig.ConfigData{
		AdditionalTools:  additionalTools,
		Timeout:          cfg.Timeout,
		SecurityConfig:   k8sSecurityConfig,
		Transport:        cfg.Transport,
		Host:             cfg.Host,
		Port:             cfg.Port,
		AccessLevel:      cfg.AccessLevel,
		AllowNamespaces:  cfg.AllowNamespaces,
		OTLPEndpoint:     cfg.OTLPEndpoint,
		TelemetryService: k8stelemetry.TelemetryInterface(cfg.TelemetryService),
	}

	return k8sCfg
}

// WrapK8sExecutor makes an mcp-kubernetes CommandExecutor
// compatible with the aks-mcp tools.CommandExecutor interface.
func WrapK8sExecutor(k8sExecutor k8stools.CommandExecutor, tokenAuthOnly bool) tools.CommandExecutor {
	return &executorAdapter{
		k8sExecutor:        k8sExecutor,
		runCommandExecutor: NewRunCommandExecutor(),
		tokenAuthOnly:      tokenAuthOnly,
	}
}

// executorAdapter bridges aks-mcp execution to mcp-kubernetes.
// Unexported; behavior is defined by the wrapped executor.
type executorAdapter struct {
	k8sExecutor        k8stools.CommandExecutor
	runCommandExecutor *RunCommandExecutor
	tokenAuthOnly      bool
}

// Execute adapts aks-mcp execution by converting its config
// and delegating to the wrapped mcp-kubernetes executor or RunCommand executor.
func (a *executorAdapter) Execute(ctx context.Context, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Defense-in-depth: reject "kubectl auth reconcile" in readonly mode
	// regardless of which downstream validator runs. The mcp-kubernetes
	// readonly classifier groups all "auth" subcommands as read-only, but
	// "auth reconcile" creates/updates RBAC objects and must not be reachable
	// from a readonly access level. This check fails closed even if the
	// dependency is downgraded or its classification regresses.
	if cfg.AccessLevel == "readonly" {
		if cmd, ok := params["command"].(string); ok {
			if isKubectlAuthReconcile(cmd) {
				return "", fmt.Errorf("security validation failed: kubectl auth reconcile is a write operation and cannot be executed in read-only mode")
			}
		}
	}

	if a.tokenAuthOnly {
		k8sCfg := ConvertConfig(cfg)
		return a.runCommandExecutor.Execute(ctx, params, k8sCfg)
	}
	k8sCfg := ConvertConfig(cfg)
	return a.k8sExecutor.Execute(ctx, params, k8sCfg)
}

// isKubectlAuthReconcile reports whether the kubectl command string invokes
// "auth reconcile". Tokenizes via shlex (matching the executor) so quoted or
// tab-separated forms cannot bypass a literal substring check, then walks the
// positional tokens skipping flags. Returns true only when the first
// positional after "auth" is "reconcile".
func isKubectlAuthReconcile(command string) bool {
	tokens, err := shlex.Split(command)
	if err != nil {
		tokens = strings.Fields(command)
	}
	// Drop everything after a free-standing "--" so subprocess args
	// (e.g. `kubectl exec ... -- auth reconcile`) are not misclassified.
	for i, t := range tokens {
		if t == "--" {
			tokens = tokens[:i]
			break
		}
	}
	seenAuth := false
	for _, t := range tokens {
		if strings.HasPrefix(t, "-") {
			continue
		}
		if t == "kubectl" {
			continue
		}
		if !seenAuth {
			if t == "auth" {
				seenAuth = true
				continue
			}
			// First positional is not "auth" — not an auth subcommand.
			return false
		}
		return t == "reconcile"
	}
	return false
}
