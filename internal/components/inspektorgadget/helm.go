package inspektorgadget

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// HelmClient defines the minimal interface used by the Inspektor Gadget handlers
type HelmClient interface {
	InstallChart(chartUrl, releaseName, namespace string) (string, error)
	UninstallChart(releaseName, namespace string) (string, error)
	CheckRelease(releaseName, namespace string) error
	UpgradeChart(chartUrl, releaseName, namespace string) (string, error)
}

type helmClient struct {
	registryClient *registry.Client
	verbose        bool
}

func newHelmClient(verbose bool) (*helmClient, error) {
	hc := http.Client{Timeout: 5 * time.Second}
	opts := []registry.ClientOption{
		registry.ClientOptHTTPClient(&hc),
	}
	rc, err := registry.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating registry client: %w", err)
	}

	return &helmClient{
		registryClient: rc,
		verbose:        verbose,
	}, nil
}

func (c *helmClient) InstallChart(chartUrl, releaseName, namespace string) (string, error) {
	actionCfg, err := c.getActionConfig(namespace, KubernetesFlags)
	if err != nil {
		return "", fmt.Errorf("getting action config: %w", err)
	}
	install := action.NewInstall(actionCfg)
	install.ReleaseName = releaseName
	install.Namespace = namespace
	// Don't create the namespace if it already exists.
	install.CreateNamespace = getPodNamespace() == ""
	install.Wait = true
	install.Timeout = 5 * time.Minute

	setting := cli.New()
	// Use /tmp for the cache to avoid permission issues when running in a pod.
	setting.RepositoryCache = "/tmp/helm/repository/.cache"
	chartPath, err := install.LocateChart(chartUrl, setting)
	if err != nil {
		return "", fmt.Errorf("locating chart: %w", err)
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("loading chart: %w", err)
	}

	release, err := install.RunWithContext(context.TODO(), chart, map[string]interface{}{})
	if err != nil {
		return "", fmt.Errorf("installing chart: %w", err)
	}

	return fmt.Sprintf("Inspektor Gadget (chartUrl: %s, release: %s) installed successfully in namespace %s", chartUrl, release.Name, namespace), nil
}

func (c *helmClient) UninstallChart(releaseName, namespace string) (string, error) {
	actionCfg, err := c.getActionConfig(namespace, KubernetesFlags)
	if err != nil {
		return "", fmt.Errorf("getting action config: %w", err)
	}
	uninstall := action.NewUninstall(actionCfg)
	uninstall.DisableHooks = true
	uninstall.Timeout = 5 * time.Minute

	_, err = uninstall.Run(releaseName)
	if err != nil {
		return "", fmt.Errorf("uninstalling chart: %w", err)
	}

	return fmt.Sprintf("Inspektor Gadget (release: %s) uninstalled successfully from namespace %s", releaseName, namespace), nil
}

func (c *helmClient) CheckRelease(releaseName, namespace string) error {
	actionCfg, err := c.getActionConfig(namespace, KubernetesFlags)
	if err != nil {
		return fmt.Errorf("getting action config: %w", err)
	}
	status := action.NewStatus(actionCfg)

	_, err = status.Run(releaseName)
	if err != nil {
		return fmt.Errorf("getting release status: %w", err)
	}

	return nil
}

func (c *helmClient) UpgradeChart(chartUrl, releaseName, namespace string) (string, error) {
	actionCfg, err := c.getActionConfig(namespace, KubernetesFlags)
	if err != nil {
		return "", fmt.Errorf("getting action config: %w", err)
	}
	upgrade := action.NewUpgrade(actionCfg)
	upgrade.Namespace = namespace
	upgrade.Wait = true
	upgrade.Timeout = 5 * time.Minute

	setting := cli.New()
	chartPath, err := upgrade.LocateChart(chartUrl, setting)
	if err != nil {
		return "", fmt.Errorf("locating chart: %w", err)
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("loading chart: %w", err)
	}

	release, err := upgrade.RunWithContext(context.TODO(), releaseName, chart, map[string]interface{}{})
	if err != nil {
		return "", fmt.Errorf("upgrading chart: %w", err)
	}

	return fmt.Sprintf("Inspektor Gadget (chartUrl: %s, release: %s) upgraded successfully in namespace %s", chartUrl, release.Name, namespace), nil
}

func (c *helmClient) getActionConfig(namespace string, k8sConfig *genericclioptions.ConfigFlags) (*action.Configuration, error) {
	actionConfig := action.Configuration{RegistryClient: c.registryClient}
	if err := actionConfig.Init(k8sConfig, namespace, os.Getenv("HELM_DRIVER"), c.debugLog); err != nil {
		return nil, fmt.Errorf("initializing action configuration: %w", err)
	}
	return &actionConfig, nil
}

func (c *helmClient) debugLog(format string, args ...any) {
	if c.verbose {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
