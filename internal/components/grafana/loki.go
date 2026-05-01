package grafana

import (
	"context"
	"fmt"
	"net/url"
)

func handleLokiQuery(ctx context.Context, params map[string]interface{}, c *client) (string, error) {
	if c.lokiURL == "" {
		return "", fmt.Errorf("LOKI_URL is not configured")
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing required 'query' parameter (LogQL expression)")
	}

	start, _ := params["start_time"].(string)
	end, _ := params["end_time"].(string)
	limit, _ := params["limit"].(string)

	if start == "" {
		start = "1h"
	}
	if limit == "" {
		limit = "100"
	}

	u, err := url.Parse(c.lokiURL + "/loki/api/v1/query_range")
	if err != nil {
		return "", fmt.Errorf("parsing Loki URL: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", start)
	if end != "" {
		q.Set("end", end)
	}
	q.Set("limit", limit)
	u.RawQuery = q.Encode()

	return c.get(ctx, u.String(), "basic", c.lokiUsername, c.lokiPassword)
}
