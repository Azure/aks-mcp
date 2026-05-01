package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure/aks-mcp/internal/config"
)

type client struct {
	http          *http.Client
	grafanaURL    string
	grafanaToken  string
	lokiURL       string
	lokiUsername  string
	lokiPassword  string
	mimirURL      string
	mimirUsername string
	mimirPassword string
	tempoURL      string
}

func newClient(cfg *config.ConfigData) *client {
	return &client{
		http:          &http.Client{Timeout: 30 * time.Second},
		grafanaURL:    cfg.GrafanaURL,
		grafanaToken:  cfg.GrafanaToken,
		lokiURL:       cfg.LokiURL,
		lokiUsername:  cfg.LokiUsername,
		lokiPassword:  cfg.LokiPassword,
		mimirURL:      cfg.MimirURL,
		mimirUsername: cfg.MimirUsername,
		mimirPassword: cfg.MimirPassword,
		tempoURL:      cfg.TempoURL,
	}
}

func (c *client) get(ctx context.Context, rawURL, authType, authUser, authPass string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	switch authType {
	case "basic":
		req.SetBasicAuth(authUser, authPass)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+authUser)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body), nil
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(body), nil
	}
	return string(pretty), nil
}
