package grafana

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
)

// mockServer starts a test HTTP server that returns the given status and body.
// It also captures the last request for inspection.
func mockServer(t *testing.T, status int, body interface{}) (*httptest.Server, *http.Request) {
	t.Helper()
	var lastReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastReq = r
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, lastReq
}

func newTestConfig(srv *httptest.Server) *config.ConfigData {
	cfg := config.NewConfig()
	cfg.GrafanaURL = srv.URL
	cfg.GrafanaToken = "test-token"
	cfg.LokiURL = srv.URL
	cfg.LokiUsername = "test-user"
	cfg.LokiPassword = "test-pass"
	cfg.MimirURL = srv.URL
	cfg.MimirUsername = "test-user"
	cfg.MimirPassword = "test-pass"
	cfg.TempoURL = srv.URL
	return cfg
}

// --- dispatch ---

func TestGetGrafanaHandler_MissingOperation(t *testing.T) {
	cfg := config.NewConfig()
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{}, cfg)
	if err == nil || !strings.Contains(err.Error(), "operation") {
		t.Errorf("expected operation error, got: %v", err)
	}
}

func TestGetGrafanaHandler_UnknownOperation(t *testing.T) {
	cfg := config.NewConfig()
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{"operation": "unknown"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "unsupported operation") {
		t.Errorf("expected unsupported operation error, got: %v", err)
	}
}

// --- loki ---

func TestLokiQuery_MissingQuery(t *testing.T) {
	srv, _ := mockServer(t, 200, map[string]interface{}{"status": "success"})
	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{"operation": "loki_query"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "query") {
		t.Errorf("expected query error, got: %v", err)
	}
}

func TestLokiQuery_URLConstruction(t *testing.T) {
	var capturedPath, capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	}))
	t.Cleanup(srv.Close)

	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{
		"operation":  "loki_query",
		"query":      `{namespace="test"}`,
		"start_time": "1h",
		"limit":      "50",
	}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPath != "/loki/api/v1/query_range" {
		t.Errorf("unexpected path: %s", capturedPath)
	}
	if !strings.Contains(capturedQuery, "query=") {
		t.Errorf("query param missing from URL: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "limit=50") {
		t.Errorf("limit param missing from URL: %s", capturedQuery)
	}
}

func TestLokiQuery_MissingURL(t *testing.T) {
	cfg := config.NewConfig() // LokiURL intentionally empty
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{
		"operation": "loki_query",
		"query":     `{namespace="test"}`,
	}, cfg)
	if err == nil || !strings.Contains(err.Error(), "LOKI_URL") {
		t.Errorf("expected LOKI_URL error, got: %v", err)
	}
}

func TestLokiQuery_BasicAuthSent(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, _ = handler.Handle(context.Background(), map[string]interface{}{
		"operation": "loki_query",
		"query":     `{app="x"}`,
	}, cfg)
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("expected Basic auth header, got: %s", authHeader)
	}
}

// --- mimir ---

func TestMimirQuery_MissingQuery(t *testing.T) {
	srv, _ := mockServer(t, 200, nil)
	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{"operation": "mimir_query"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "query") {
		t.Errorf("expected query error, got: %v", err)
	}
}

func TestMimirQuery_URLConstruction(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	}))
	t.Cleanup(srv.Close)

	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{
		"operation": "mimir_query",
		"query":     `up`,
	}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPath != "/prometheus/api/v1/query_range" {
		t.Errorf("unexpected path: %s", capturedPath)
	}
}

func TestMimirQuery_MissingURL(t *testing.T) {
	cfg := config.NewConfig()
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{
		"operation": "mimir_query",
		"query":     "up",
	}, cfg)
	if err == nil || !strings.Contains(err.Error(), "MIMIR_URL") {
		t.Errorf("expected MIMIR_URL error, got: %v", err)
	}
}

// --- tempo ---

func TestTempoSearch_URLConstruction(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"traces":[]}`))
	}))
	t.Cleanup(srv.Close)

	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{
		"operation": "tempo_search",
		"query":     `{.service.name="my-svc"} && status = error`,
	}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPath != "/api/search" {
		t.Errorf("unexpected path: %s", capturedPath)
	}
}

func TestTempoSearch_NoAuthSent(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"traces":[]}`))
	}))
	t.Cleanup(srv.Close)

	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, _ = handler.Handle(context.Background(), map[string]interface{}{"operation": "tempo_search"}, cfg)
	if authHeader != "" {
		t.Errorf("expected no auth header for Tempo, got: %s", authHeader)
	}
}

func TestTempoGetTrace_MissingTraceID(t *testing.T) {
	srv, _ := mockServer(t, 200, nil)
	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{"operation": "tempo_get_trace"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "trace_id") {
		t.Errorf("expected trace_id error, got: %v", err)
	}
}

func TestTempoGetTrace_URLConstruction(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"traceID":"abc123","spans":[]}`))
	}))
	t.Cleanup(srv.Close)

	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{
		"operation": "tempo_get_trace",
		"trace_id":  "abc123",
	}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPath != "/api/traces/abc123" {
		t.Errorf("unexpected path: %s", capturedPath)
	}
}

func TestTempoQuery_MissingURL(t *testing.T) {
	cfg := config.NewConfig()
	handler := GetGrafanaHandler(cfg)
	for _, op := range []string{"tempo_search", "tempo_get_trace"} {
		params := map[string]interface{}{"operation": op, "trace_id": "abc"}
		_, err := handler.Handle(context.Background(), params, cfg)
		if err == nil || !strings.Contains(err.Error(), "TEMPO_URL") {
			t.Errorf("op %s: expected TEMPO_URL error, got: %v", op, err)
		}
	}
}

// --- alerts ---

func TestAlerts_MissingGrafanaURL(t *testing.T) {
	cfg := config.NewConfig()
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{"operation": "alerts"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "GRAFANA_URL") {
		t.Errorf("expected GRAFANA_URL error, got: %v", err)
	}
}

func TestAlerts_MissingToken(t *testing.T) {
	cfg := config.NewConfig()
	cfg.GrafanaURL = "http://example.com"
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{"operation": "alerts"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "GRAFANA_TOKEN") {
		t.Errorf("expected GRAFANA_TOKEN error, got: %v", err)
	}
}

func TestAlerts_BearerAuthSent(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, _ = handler.Handle(context.Background(), map[string]interface{}{"operation": "alerts"}, cfg)
	if authHeader != "Bearer test-token" {
		t.Errorf("expected Bearer auth, got: %s", authHeader)
	}
}

func TestAlerts_URLConstruction(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{
		"operation": "alerts",
		"filter":    `namespace="my-ns"`,
	}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPath != "/api/alertmanager/grafana/api/v2/alerts" {
		t.Errorf("unexpected path: %s", capturedPath)
	}
}

func TestAlerts_HTTPError(t *testing.T) {
	srv, _ := mockServer(t, 401, map[string]interface{}{"message": "Unauthorized"})
	cfg := newTestConfig(srv)
	handler := GetGrafanaHandler(cfg)
	_, err := handler.Handle(context.Background(), map[string]interface{}{"operation": "alerts"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("expected HTTP 401 error, got: %v", err)
	}
}
