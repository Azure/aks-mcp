package inspektorgadget

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
)

var (
	gadgetVersionRegex  = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	githubVersionOnce   sync.Once
	cachedGithubVersion string
	githubVersionError  error
)

// gadgetVersionFor maps the Inspektor Gadget version to a corresponding gadget version.
// Returns the stable version if installed; otherwise, returns "latest".
func gadgetVersionFor(igVersion string) string {
	if gadgetVersionRegex.MatchString(igVersion) {
		return "v" + igVersion
	}
	return "latest"
}

// getChartVersion retrieves the version of the Inspektor Gadget Helm chart.
// It first attempts to get the version from GitHub releases, and if that fails,
// it falls back to the version from the build information.
func getChartVersion() string {
	if version, err := getLatestVersionFromGitHub(); err == nil {
		return version
	}
	return getChartVersionFromBuild()
}

// getChartVersionFromBuild retrieves the version of the Inspektor Gadget Helm chart from the build information.
func getChartVersionFromBuild() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/inspektor-gadget/inspektor-gadget" {
				if dep.Version != "" {
					return strings.TrimPrefix(dep.Version, "v")
				}
			}
		}
	}
	return "1.0.0-dev"
}

// getLatestVersionFromGitHub retrieves the version of the latest Inspektor Gadget release from GitHub.
func getLatestVersionFromGitHub() (string, error) {
	githubVersionOnce.Do(func() {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(inspektorGadgetReleaseURL)
		if err != nil {
			githubVersionError = fmt.Errorf("failed to get latest release: %w", err)
			return
		}
		defer func() {
			if err = resp.Body.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "closing response body: %v\n", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			githubVersionError = fmt.Errorf("failed to get latest release, status code: %d", resp.StatusCode)
			return
		}

		var release struct {
			TagName string `json:"tag_name"`
		}
		if err = json.NewDecoder(resp.Body).Decode(&release); err != nil {
			githubVersionError = fmt.Errorf("decoding latest release response: %w", err)
			return
		}
		cachedGithubVersion = strings.TrimPrefix(release.TagName, "v")
	})
	return cachedGithubVersion, githubVersionError
}

// getPodNamespace returns the namespace in which the current pod is running.
func getPodNamespace() string {
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(ns))
}

// getChartNamespace returns the namespace in which the Inspektor Gadget Helm chart should be deployed
func getChartNamespace() string {
	if ns := getPodNamespace(); ns != "" {
		return ns
	}
	return "gadget"
}

// gadgetInstanceFromAPI converts an API GadgetInstance to a GadgetInstance struct.
func gadgetInstanceFromAPI(instance *api.GadgetInstance) *GadgetInstance {
	if instance == nil {
		return nil
	}

	var createdBy string
	for _, tag := range instance.Tags {
		if strings.HasPrefix(tag, "createdBy=") {
			createdBy = strings.TrimPrefix(tag, "createdBy=")
			break
		}
	}
	var gadgetName string
	for _, tag := range instance.Tags {
		if strings.HasPrefix(tag, "gadgetName=") {
			gadgetName = strings.TrimPrefix(tag, "gadgetName=")
			break
		}
	}
	var filterParams map[string]string
	for _, tag := range instance.Tags {
		if strings.HasPrefix(tag, "filterParams=") {
			filterParamsStr := strings.TrimPrefix(tag, "filterParams=")
			filterParams = make(map[string]string)
			for _, param := range strings.Split(filterParamsStr, ",") {
				kv := strings.SplitN(param, "=", 2)
				if len(kv) == 2 {
					filterParams[kv[0]] = kv[1]
				}
			}
			break
		}
	}

	var namespaces []string
	for _, tag := range instance.Tags {
		if strings.HasPrefix(tag, "namespaces=") {
			namespacesStr := strings.TrimPrefix(tag, "namespaces=")
			if namespacesStr != "" {
				namespaces = strings.Split(namespacesStr, ",")
				break
			}
		}
	}

	return &GadgetInstance{
		ID:           instance.Id,
		GadgetName:   gadgetName,
		GadgetImage:  instance.GadgetConfig.ImageName,
		FilterParams: filterParams,
		Namespaces:   namespaces,
		CreatedBy:    createdBy,
		StartedAt:    time.Unix(instance.TimeCreated, 0).Format(time.RFC3339),
	}
}

// isValidLifecycleAction checks if the provided action is a valid lifecycle action for Inspektor Gadget.
func isValidLifecycleAction(action string) bool {
	return action == deployAction || action == undeployAction || action == isDeployedAction || action == upgradeAction
}

// getLifecycleActions returns all valid lifecycle actions for Inspektor Gadget.
func getLifecycleActions() []string {
	return []string{deployAction, undeployAction, upgradeAction, isDeployedAction}
}

// isValidAction checks if the provided action is a valid action for Inspektor Gadget.
func isValidAction(action string) bool {
	return action == runAction || action == startAction || action == stopAction ||
		action == getResultsAction || action == listGadgetsAction || isValidLifecycleAction(action)
}

// getActions returns all valid actions for Inspektor Gadget.
func getActions() []string {
	return append(getLifecycleActions(), []string{
		runAction,
		startAction,
		stopAction,
		getResultsAction,
		listGadgetsAction,
	}...)
}

func isValidFilterParamKey(key string) bool {
	validKeys := getFilterParamKeys()
	return slices.Contains(validKeys, key)
}

func getFilterParamKeys() []string {
	return append(getGadgetParamsKeys(), []string{
		"namespace",
		"pod",
		"container",
		"selector",
		"node",
	}...)
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range a {
		merged[k] = v
	}
	for k, v := range b {
		merged[k] = v
	}
	return merged
}
