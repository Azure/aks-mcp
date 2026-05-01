package grafana

import (
	"context"
	"fmt"
	"net/url"
)

func handleAlerts(ctx context.Context, params map[string]interface{}, c *client) (string, error) {
	if c.grafanaURL == "" {
		return "", fmt.Errorf("GRAFANA_URL is not configured")
	}
	if c.grafanaToken == "" {
		return "", fmt.Errorf("GRAFANA_TOKEN is not configured")
	}

	u, err := url.Parse(c.grafanaURL + "/api/alertmanager/grafana/api/v2/alerts")
	if err != nil {
		return "", fmt.Errorf("parsing Grafana URL: %w", err)
	}

	q := u.Query()
	// Optional label matchers, e.g. namespace="my-ns"
	if filter, ok := params["filter"].(string); ok && filter != "" {
		q.Add("filter", filter)
	}
	// Defaults to returning only active alerts; can be overridden
	if silenced, ok := params["silenced"].(string); ok {
		q.Set("silenced", silenced)
	} else {
		q.Set("silenced", "false")
	}
	if inhibited, ok := params["inhibited"].(string); ok {
		q.Set("inhibited", inhibited)
	} else {
		q.Set("inhibited", "false")
	}
	u.RawQuery = q.Encode()

	return c.get(ctx, u.String(), "bearer", c.grafanaToken, "")
}
