package grafana

import (
	"context"
	"fmt"
	"net/url"
)

func handleMimirQuery(ctx context.Context, params map[string]interface{}, c *client) (string, error) {
	if c.mimirURL == "" {
		return "", fmt.Errorf("MIMIR_URL is not configured")
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing required 'query' parameter (PromQL expression)")
	}

	start, _ := params["start_time"].(string)
	end, _ := params["end_time"].(string)
	step, _ := params["step"].(string)

	if start == "" {
		start = "now-1h"
	}
	if end == "" {
		end = "now"
	}
	if step == "" {
		step = "60s"
	}

	u, err := url.Parse(c.mimirURL + "/prometheus/api/v1/query_range")
	if err != nil {
		return "", fmt.Errorf("parsing Mimir URL: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", start)
	q.Set("end", end)
	q.Set("step", step)
	u.RawQuery = q.Encode()

	return c.get(ctx, u.String(), "basic", c.mimirUsername, c.mimirPassword)
}
