package inspektorgadget

import (
	"context"
	"fmt"

	k8scommand "github.com/Azure/mcp-kubernetes/pkg/command"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
)

// =============================================================================
// Inspektor Gadget deploy/undeploy Executor
// =============================================================================

// InspektorGadgetDeployUndeployExecutor is an executor for deploying and undeploying Inspektor Gadget
type InspektorGadgetDeployUndeployExecutor struct {
	mgr GadgetManager
}

// NewInspektorGadgetDeployUndeployExecutor creates a new executor for Inspektor Gadget deploy/undeploy operations
func NewInspektorGadgetDeployUndeployExecutor(mgr GadgetManager) *InspektorGadgetDeployUndeployExecutor {
	return &InspektorGadgetDeployUndeployExecutor{
		mgr: mgr,
	}
}

// Execute performs the deploy or undeploy action for Inspektor Gadget
func (e *InspektorGadgetDeployUndeployExecutor) Execute(params map[string]interface{}, cfg *k8sconfig.ConfigData) (string, error) {
	// TODO: use security.Validator once helm readwrite/admin operations are implemented
	if !cfg.SecurityConfig.IsNamespaceAllowed(inspektorGadgetChartNamespace) {
		return "", fmt.Errorf("namespace %s is not allowed by security policy", inspektorGadgetChartNamespace)
	}

	// validate params
	deployed, _, err := e.mgr.IsDeployed(context.Background())
	if err != nil {
		return "", fmt.Errorf("checking inspektor gadget deployment status: %w", err)
	}
	action, ok := params["action"].(string)
	if !ok || !isValidLifecycleAction(action) {
		validActions := getValidLifecycleActions()
		return "", fmt.Errorf("invalid action: %s, expected one of %v", action, validActions)
	}

	// handle undeploy action
	if action == undeployAction {
		if !deployed {
			return "inspektor gadget is not deployed, nothing to undeploy", nil
		}
		return inspektorGadgetUndeploy(cfg)
	}

	// handle deploy action
	if deployed {
		return "inspektor gadget is already deployed, nothing to do", nil
	}
	return inspektorGadgetDeploy(params, cfg)
}

func inspektorGadgetDeploy(params map[string]interface{}, cfg *k8sconfig.ConfigData) (string, error) {
	chartVersion, ok := params["chart_version"].(string)
	if !ok || chartVersion == "" {
		chartVersion = getChartVersionFromBuild()
	}
	chartUrl := fmt.Sprintf("%s:%s", inspektorGadgetChartURL, chartVersion)
	helmArgs := fmt.Sprintf("install %s -n %s --create-namespace %s", inspektorGadgetChartRelease, inspektorGadgetChartNamespace, chartUrl)
	process := k8scommand.NewShellProcess("helm", cfg.Timeout)
	return process.Run(helmArgs)
}

func inspektorGadgetUndeploy(cfg *k8sconfig.ConfigData) (string, error) {
	helmArgs := fmt.Sprintf("uninstall %s -n %s", inspektorGadgetChartRelease, inspektorGadgetChartNamespace)
	process := k8scommand.NewShellProcess("helm", cfg.Timeout)
	return process.Run(helmArgs)
}
