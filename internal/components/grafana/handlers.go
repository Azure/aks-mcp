package grafana

import (
	"context"
	"fmt"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
)

// GetGrafanaHandler returns a ResourceHandler for the grafana_observability tool.
func GetGrafanaHandler(cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		operation, ok := params["operation"].(string)
		if !ok || operation == "" {
			return "", fmt.Errorf("missing or invalid 'operation' parameter")
		}

		c := newClient(cfg)

		switch operation {
		case "loki_query":
			return handleLokiQuery(ctx, params, c)
		case "mimir_query":
			return handleMimirQuery(ctx, params, c)
		case "tempo_search":
			return handleTempoSearch(ctx, params, c)
		case "tempo_get_trace":
			return handleTempoGetTrace(ctx, params, c)
		case "alerts":
			return handleAlerts(ctx, params, c)
		default:
			return "", fmt.Errorf("unsupported operation %q — valid operations: loki_query, mimir_query, tempo_search, tempo_get_trace, alerts", operation)
		}
	})
}
