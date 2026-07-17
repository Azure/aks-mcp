package inspektorgadget

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
)

var (
	gadgetVersionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

// gadgetVersionFor maps the Inspektor Gadget version to a corresponding gadget version.
// Returns the stable version if installed; otherwise, returns "latest".
func gadgetVersionFor(igVersion string) string {
	if gadgetVersionRegex.MatchString(igVersion) {
		return "v" + igVersion
	}
	return "latest"
}

// getPodNamespace returns the namespace in which the current pod is running.
func getPodNamespace() string {
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(ns))
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
	return action == deployAction || action == undeployAction || action == isDeployedAction
}

// getLifecycleActions returns all valid lifecycle actions for Inspektor Gadget.
func getLifecycleActions() []string {
	return []string{deployAction, undeployAction, isDeployedAction}
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
