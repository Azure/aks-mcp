package inspektorgadget

import (
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
	"runtime/debug"
	"strings"
	"time"
)

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

	return &GadgetInstance{
		ID:           instance.Id,
		GadgetName:   gadgetName,
		GadgetImage:  instance.GadgetConfig.ImageName,
		GadgetParams: instance.GadgetConfig.ParamValues,
		CreatedBy:    createdBy,
		StartedAt:    time.Unix(instance.TimeCreated, 0).Format(time.RFC3339),
	}
}

// isValidLifecycleAction checks if the provided action is a valid lifecycle action for Inspektor Gadget.
func isValidLifecycleAction(action string) bool {
	return action == deployAction || action == undeployAction
}

// isValidGadgetAction checks if the provided action is a valid gadget action for Inspektor Gadget.
func isValidGadgetAction(action string) bool {
	return action == runAction || action == startAction || action == stopAction ||
		action == getResultsAction || action == listGadgetsAction
}

// getValidLifecycleActions returns all valid lifecycle actions for Inspektor Gadget.
func getValidLifecycleActions() []string {
	return []string{deployAction, undeployAction}
}

// getValidGadgetActions returns all valid gadget actions for Inspektor Gadget.
func getValidGadgetActions() []string {
	return []string{runAction, startAction, stopAction, getResultsAction, listGadgetsAction}
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
