// Package k8s provides adapters that let aks-mcp interoperate with the
// mcp-kubernetes libraries. It maps aks-mcp configuration and executors
// to the types expected by mcp-kubernetes without altering behavior.
package k8s

import (
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
	k8ssecurity "github.com/Azure/mcp-kubernetes/pkg/security"
	k8stelemetry "github.com/Azure/mcp-kubernetes/pkg/telemetry"
	k8stools "github.com/Azure/mcp-kubernetes/pkg/tools"
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
func WrapK8sExecutor(k8sExecutor k8stools.CommandExecutor) tools.CommandExecutor {
	return &executorAdapter{k8sExecutor: k8sExecutor}
}

// executorAdapter bridges aks-mcp execution to mcp-kubernetes.
// Unexported; behavior is defined by the wrapped executor.
type executorAdapter struct {
	k8sExecutor k8stools.CommandExecutor
}

// Execute adapts aks-mcp execution by converting its config
// and delegating to the wrapped mcp-kubernetes executor.
func (a *executorAdapter) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	k8sCfg := ConvertConfig(cfg)
	return a.k8sExecutor.Execute(params, k8sCfg)
}
