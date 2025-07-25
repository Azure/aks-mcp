package inspektorgadget

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
)

// =============================================================================
// Inspektor Gadget Handler
// =============================================================================

var ErrNotDeployed = fmt.Errorf("inspektor gadget is not deployed, please deploy it first using 'inspektor_gadget_deploy_undeploy' tool")

// InspektorGadgetHandler returns a handler to manage gadgets
func InspektorGadgetHandler(mgr GadgetManager, cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(params map[string]interface{}, _ *config.ConfigData) (string, error) {
		ctx := context.Background()

		// Validate action parameter
		action, ok := params["action"].(string)
		if !ok || action == "" {
			return "", fmt.Errorf("missing 'action' parameter, must be a non-empty string")
		}
		if !isValidGadgetAction(action) {
			validActions := getValidGadgetActions()
			return "", fmt.Errorf("invalid action: %s, expected one of %v", action, validActions)
		}

		// Check if Inspektor Gadget is deployed
		deployed, _, err := mgr.IsDeployed(ctx)
		if err != nil {
			return "", fmt.Errorf("checking Inspektor Gadget deployment: %w", err)
		}
		if !deployed {
			return "", ErrNotDeployed
		}

		// Initialize action/filter parameters if not provided
		actionParams, ok := params["action_params"].(map[string]interface{})
		if !ok {
			actionParams = map[string]interface{}{}
		}
		filterParams, ok := params["filter_params"].(map[string]interface{})
		if !ok {
			filterParams = map[string]interface{}{}
		}

		switch action {
		case runAction:
			return handleRunAction(ctx, mgr, actionParams, filterParams, cfg)
		case startAction:
			return handleStartAction(ctx, mgr, actionParams, filterParams, cfg)
		case stopAction:
			return handleStopAction(ctx, mgr, actionParams)
		case getResultsAction:
			return handleGetResultsAction(ctx, mgr, actionParams)
		case listGadgetsAction:
			return handleListGadgetsAction(ctx, mgr)
		}

		return "", fmt.Errorf("unsupported action: %s", action)
	})
}

func handleRunAction(ctx context.Context, mgr GadgetManager, actionParams map[string]interface{}, filterParams map[string]interface{}, cfg *config.ConfigData) (string, error) {
	gadgetName, ok := actionParams["gadget_name"].(string)
	if !ok || gadgetName == "" {
		return "", fmt.Errorf("invalid or missing 'gadget_name' parameter in 'run' action, must be a non-empty string")
	}

	gadget, ok := getGadgetByName(gadgetName)
	if !ok {
		return "", fmt.Errorf("invalid or unsupported gadget name: %s", gadgetName)
	}

	duration, ok := actionParams["duration"].(float64)
	if !ok || duration <= 0 {
		duration = 10
	}

	gadgetParams, err := prepareCommonParams(filterParams, cfg)
	if err != nil {
		return "", fmt.Errorf("preparing common parameters: %w", err)
	}
	if gadget.ParamsFunc != nil {
		gadget.ParamsFunc(filterParams, gadgetParams)
	}
	// set map-fetch-interval to half of the timeout to limit the volume of data fetched
	dur := time.Duration(duration) * time.Second
	gadgetParams[paramFetchInterval] = (dur / 2).String()

	resp, err := mgr.RunGadget(ctx, gadget.Image, gadgetParams, dur)
	if err != nil {
		return "", fmt.Errorf("running gadget: %w", err)
	}
	return resp, nil
}

func handleStartAction(ctx context.Context, mgr GadgetManager, actionParams map[string]interface{}, filterParams map[string]interface{}, cfg *config.ConfigData) (string, error) {
	gadgetName, ok := actionParams["gadget_name"].(string)
	if !ok || gadgetName == "" {
		return "", fmt.Errorf("invalid or missing 'gadget_name' parameter in 'start' action, must be a non-empty string")
	}

	gadget, ok := getGadgetByName(gadgetName)
	if !ok {
		return "", fmt.Errorf("invalid or unsupported gadget name: %s", gadgetName)
	}

	gadgetParams, err := prepareCommonParams(filterParams, cfg)
	if err != nil {
		return "", fmt.Errorf("preparing common parameters: %w", err)
	}
	if gadget.ParamsFunc != nil {
		gadget.ParamsFunc(filterParams, gadgetParams)
	}

	id, err := mgr.StartGadget(ctx, gadgetName, gadget.Image, gadgetParams)
	if err != nil {
		return "", fmt.Errorf("starting gadget: %w", err)
	}
	return fmt.Sprintf("Gadget started with ID: %s", id), nil
}

func handleStopAction(ctx context.Context, mgr GadgetManager, actionParams map[string]interface{}) (string, error) {
	id, ok := actionParams["gadget_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("invalid or missing 'gadget_id' parameter in 'stop' action, must be a non-empty string")
	}
	err := mgr.StopGadget(ctx, id)
	if err != nil {
		return "", fmt.Errorf("stopping gadget: %w", err)
	}
	return fmt.Sprintf("Gadget with ID %s stopped successfully", id), nil
}

func handleGetResultsAction(ctx context.Context, mgr GadgetManager, actionParams map[string]interface{}) (string, error) {
	id, ok := actionParams["gadget_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("invalid or missing 'gadget_id' parameter in 'get_results' action, must be a non-empty string")
	}
	results, err := mgr.GetResults(ctx, id)
	if err != nil {
		return "", fmt.Errorf("getting gadget results: %w", err)
	}
	return results, nil
}

func handleListGadgetsAction(ctx context.Context, mgr GadgetManager) (string, error) {
	gs, err := mgr.ListGadgets(ctx)
	if err != nil {
		return "", fmt.Errorf("listing gadgets: %w", err)
	}
	if len(gs) == 0 {
		return "No gadgets are currently running", nil
	}
	JSONData, err := json.Marshal(gs)
	if err != nil {
		return "", fmt.Errorf("marshalling gadget list to JSON: %w", err)
	}
	return string(JSONData), nil
}

func prepareCommonParams(filterParams map[string]interface{}, cfg *config.ConfigData) (map[string]string, error) {
	// We need to ensure that the security policy allows the namespace
	ns, ok := filterParams["namespace"].(string)
	if ok && ns != "" && cfg.SecurityConfig != nil {
		if !cfg.SecurityConfig.IsNamespaceAllowed(ns) {
			return nil, fmt.Errorf("namespace %s is not allowed by security policy", ns)
		}
	}

	// If the namespace is provided, use it; otherwise, use the allowed namespaces or all namespaces
	// depending on the security policy
	params := make(map[string]string)
	if ns != "" {
		params[paramNamespace] = ns
	} else if cfg.SecurityConfig != nil && cfg.SecurityConfig.AllowedNamespaces != "" {
		params[paramNamespace] = cfg.SecurityConfig.AllowedNamespaces
	} else {
		params[paramAllNamespaces] = "true"
	}

	if pod, ok := filterParams["pod"].(string); ok && pod != "" {
		params[paramPod] = pod
	}

	if container, ok := filterParams["container"].(string); ok && container != "" {
		params[paramContainer] = container
	}

	if selector, ok := filterParams["selector"].(string); ok && selector != "" {
		params[paramSelector] = selector
	}

	return params, nil
}
