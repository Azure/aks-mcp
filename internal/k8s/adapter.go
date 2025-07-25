package k8s

import (
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
	k8ssecurity "github.com/Azure/mcp-kubernetes/pkg/security"
	k8stools "github.com/Azure/mcp-kubernetes/pkg/tools"
)

// ConfigAdapter converts aks-mcp config to mcp-kubernetes config
func ConvertConfig(cfg *config.ConfigData) *k8sconfig.ConfigData {
	// Create K8s security config
	k8sSecurityConfig := k8ssecurity.NewSecurityConfig()

	// Map allowed namespaces and access level
	k8sSecurityConfig.SetAllowedNamespaces(cfg.AllowNamespaces)
	k8sSecurityConfig.AccessLevel = k8ssecurity.AccessLevel(cfg.AccessLevel)

	// Create K8s config
	k8sCfg := &k8sconfig.ConfigData{
		AdditionalTools: cfg.AdditionalTools,
		Timeout:         cfg.Timeout,
		SecurityConfig:  k8sSecurityConfig,
		Transport:       cfg.Transport,
		Host:            cfg.Host,
		Port:            cfg.Port,
		AccessLevel:     cfg.AccessLevel,
		AllowNamespaces: cfg.AllowNamespaces,
	}

	return k8sCfg
}

// WrapK8sExecutor wraps a mcp-kubernetes executor to work with aks-mcp config
func WrapK8sExecutor(k8sExecutor k8stools.CommandExecutor) tools.CommandExecutor {
	return &executorAdapter{k8sExecutor: k8sExecutor}
}

// executorAdapter adapts between aks-mcp and mcp-kubernetes configs
type executorAdapter struct {
	k8sExecutor k8stools.CommandExecutor
}

func (a *executorAdapter) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Convert aks-mcp config to k8s config
	k8sCfg := ConvertConfig(cfg)

	// Execute using the k8s executor
	return a.k8sExecutor.Execute(params, k8sCfg)
}
