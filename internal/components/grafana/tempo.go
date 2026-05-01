package grafana

import (
	"context"
	"fmt"
	"net/url"
)

func handleTempoSearch(ctx context.Context, params map[string]interface{}, c *client) (string, error) {
	if c.tempoURL == "" {
		return "", fmt.Errorf("TEMPO_URL is not configured")
	}

	u, err := url.Parse(c.tempoURL + "/api/search")
	if err != nil {
		return "", fmt.Errorf("parsing Tempo URL: %w", err)
	}

	q := u.Query()
	if query, ok := params["query"].(string); ok && query != "" {
		q.Set("q", query)
	}
	if start, ok := params["start_time"].(string); ok && start != "" {
		q.Set("start", start)
	}
	if end, ok := params["end_time"].(string); ok && end != "" {
		q.Set("end", end)
	}
	if limit, ok := params["limit"].(string); ok && limit != "" {
		q.Set("limit", limit)
	}
	u.RawQuery = q.Encode()

	return c.get(ctx, u.String(), "", "", "")
}

func handleTempoGetTrace(ctx context.Context, params map[string]interface{}, c *client) (string, error) {
	if c.tempoURL == "" {
		return "", fmt.Errorf("TEMPO_URL is not configured")
	}

	traceID, ok := params["trace_id"].(string)
	if !ok || traceID == "" {
		return "", fmt.Errorf("missing required 'trace_id' parameter")
	}

	return c.get(ctx, c.tempoURL+"/api/traces/"+traceID, "", "", "")
}
