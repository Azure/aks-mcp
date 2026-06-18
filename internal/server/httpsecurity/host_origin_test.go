package httpsecurity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsHostAllowed(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		allowed []string
		want    bool
	}{
		// Default (empty allowlist) — loopback only.
		{"default allows localhost", "localhost", nil, true},
		{"default allows localhost with port", "localhost:8000", nil, true},
		{"default allows 127.0.0.1", "127.0.0.1", nil, true},
		{"default allows 127.0.0.1 with port", "127.0.0.1:8000", nil, true},
		{"default allows IPv6 loopback bracketed", "[::1]:8000", nil, true},
		{"default allows IPv6 loopback bare", "::1", nil, true},
		{"default allows 127.0.0.0/8", "127.5.5.5", nil, true},
		{"default rejects foreign host", "rebind.example.com", nil, false},
		{"default rejects 0.0.0.0", "0.0.0.0", nil, false},
		{"default rejects empty host", "", nil, false},

		// Explicit allowlist.
		{"explicit allows listed host", "aks-mcp.example.com", []string{"aks-mcp.example.com"}, true},
		{"explicit allows listed host case insensitive", "AKS-MCP.EXAMPLE.COM", []string{"aks-mcp.example.com"}, true},
		{"explicit allows listed host with port", "aks-mcp.example.com:8000", []string{"aks-mcp.example.com"}, true},
		{"explicit rejects unlisted loopback", "127.0.0.1", []string{"aks-mcp.example.com"}, false},
		{"explicit rejects unlisted host", "rebind.example.com", []string{"aks-mcp.example.com"}, false},
		{"explicit ignores empty entries", "aks-mcp.example.com", []string{"", "aks-mcp.example.com", ""}, true},

		// Wildcard escape valve.
		{"wildcard allows anything", "rebind.example.com", []string{"*"}, true},
		{"wildcard with other entries still allows anything", "rebind.example.com", []string{"aks-mcp.example.com", "*"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isHostAllowed(tc.host, tc.allowed); got != tc.want {
				t.Errorf("isHostAllowed(%q, %v) = %v, want %v", tc.host, tc.allowed, got, tc.want)
			}
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		allowed []string
		want    bool
	}{
		// Empty Origin always allowed.
		{"empty origin allowed (no allowlist)", "", nil, true},
		{"empty origin allowed (with allowlist)", "", []string{"http://localhost:8000"}, true},

		// Empty allowlist rejects any non-empty Origin.
		{"non-empty origin rejected with empty allowlist", "http://rebind.example.com:8806", nil, false},

		// Explicit allowlist.
		{"listed origin allowed", "http://localhost:8000", []string{"http://localhost:8000"}, true},
		{"listed origin case insensitive", "HTTP://LOCALHOST:8000", []string{"http://localhost:8000"}, true},
		{"unlisted origin rejected", "http://rebind.example.com", []string{"http://localhost:8000"}, false},

		// Wildcard escape valve.
		{"wildcard allows anything", "http://rebind.example.com", []string{"*"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isOriginAllowed(tc.origin, tc.allowed); got != tc.want {
				t.Errorf("isOriginAllowed(%q, %v) = %v, want %v", tc.origin, tc.allowed, got, tc.want)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	pass := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	tests := []struct {
		name           string
		cfg            Config
		requestHost    string // value to set on r.Host
		requestOrigin  string // value to set as Origin header (empty = unset)
		wantStatus     int
		wantBodyPrefix string // partial match on body
	}{
		{
			name:        "loopback default allows curl-style request",
			cfg:         Config{},
			requestHost: "127.0.0.1:8000",
			wantStatus:  http.StatusOK,
		},
		{
			name:          "loopback default rejects browser dns-rebind host",
			cfg:           Config{},
			requestHost:   "rebind.example.com:8000",
			requestOrigin: "http://rebind.example.com:8000",
			wantStatus:    http.StatusForbidden,
		},
		{
			name:          "loopback host accepted but foreign origin rejected when allowlist empty",
			cfg:           Config{},
			requestHost:   "127.0.0.1:8000",
			requestOrigin: "http://rebind.example.com:8000",
			wantStatus:    http.StatusForbidden,
		},
		{
			name:          "explicit allowlist accepts matching host and origin",
			cfg:           Config{AllowedHosts: []string{"aks-mcp.example.com"}, AllowedOrigins: []string{"https://chat.example.com"}},
			requestHost:   "aks-mcp.example.com",
			requestOrigin: "https://chat.example.com",
			wantStatus:    http.StatusOK,
		},
		{
			name:        "explicit allowlist rejects loopback when not listed",
			cfg:         Config{AllowedHosts: []string{"aks-mcp.example.com"}},
			requestHost: "127.0.0.1:8000",
			wantStatus:  http.StatusForbidden,
		},
		{
			name:          "wildcard origin allowlist passes any origin",
			cfg:           Config{AllowedOrigins: []string{"*"}},
			requestHost:   "127.0.0.1:8000",
			requestOrigin: "http://rebind.example.com",
			wantStatus:    http.StatusOK,
		},
		{
			name:        "wildcard host allowlist passes any host",
			cfg:         Config{AllowedHosts: []string{"*"}},
			requestHost: "rebind.example.com",
			wantStatus:  http.StatusOK,
		},
		{
			name:          "ipv6 loopback default allowed",
			cfg:           Config{},
			requestHost:   "[::1]:8000",
			requestOrigin: "",
			wantStatus:    http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw := NewMiddleware(tc.cfg)
			handler := mw(pass)

			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(""))
			req.Host = tc.requestHost
			if tc.requestOrigin != "" {
				req.Header.Set("Origin", tc.requestOrigin)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rr.Code, tc.wantStatus, rr.Body.String())
			}

			if tc.wantStatus == http.StatusForbidden {
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
				var body map[string]string
				if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
					t.Fatalf("body is not valid JSON: %v (body=%s)", err, rr.Body.String())
				}
				if body["error"] != "forbidden" {
					t.Errorf("body.error = %q, want forbidden", body["error"])
				}
				if !strings.Contains(body["error_description"], "host/origin policy") {
					t.Errorf("body.error_description = %q, want substring 'host/origin policy'", body["error_description"])
				}
			}
		})
	}
}
