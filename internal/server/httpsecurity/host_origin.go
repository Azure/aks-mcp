// Package httpsecurity provides HTTP middleware that guards MCP-bearing
// endpoints against browser-origin cross-origin abuse, in particular DNS
// rebinding attacks.
//
// This middleware is NOT an authentication mechanism. It only enforces that
// the HTTP Host header matches an explicitly trusted name (or a loopback
// default) and that any non-empty Origin header matches an explicitly trusted
// origin. Authentication is the job of the OAuth middleware in
// internal/auth/oauth.
package httpsecurity

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/Azure/aks-mcp/internal/logger"
)

// Config holds the allowlists consulted by the middleware. A zero value falls
// back to the loopback-only Host policy and rejects every non-empty Origin.
type Config struct {
	// AllowedHosts is the set of Host header values (with or without port)
	// that the middleware will accept. When empty, the middleware defaults
	// to allowing only loopback hosts (localhost, 127.0.0.1, [::1]). The
	// literal "*" disables Host enforcement entirely; use it only when an
	// upstream reverse proxy validates Host on the operator's behalf.
	AllowedHosts []string

	// AllowedOrigins is the set of Origin header values (scheme://host[:port])
	// that the middleware will accept on cross-origin browser requests.
	// Empty Origin headers (non-browser clients) are always allowed. When
	// AllowedOrigins is empty every non-empty Origin is rejected. The
	// literal "*" disables Origin enforcement entirely.
	AllowedOrigins []string
}

// NewMiddleware returns an http middleware that enforces cfg's Host and Origin
// policy before delegating to next. Rejected requests receive a 403 response
// with a JSON error body.
func NewMiddleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isHostAllowed(r.Host, cfg.AllowedHosts) {
				writeForbidden(w, r, "host not allowed", r.Host, r.Header.Get("Origin"))
				return
			}
			if !isOriginAllowed(r.Header.Get("Origin"), cfg.AllowedOrigins) {
				writeForbidden(w, r, "origin not allowed", r.Host, r.Header.Get("Origin"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isHostAllowed reports whether the request's Host header satisfies the
// configured allowlist. With an empty allowlist, only loopback names match.
//
// Allowlist entries may be written with or without a port. Each request is
// compared against both forms of the entry: an entry like
// "aks-mcp.example.com" matches the request regardless of port, while
// "aks-mcp.example.com:8000" matches only that port.
func isHostAllowed(rawHost string, allowed []string) bool {
	rawHost = strings.TrimSpace(rawHost)
	if rawHost == "" {
		// A missing Host header is suspicious on its own — refuse it rather
		// than fall through to the default-allow loopback branch.
		return false
	}
	hostNoPort := strings.ToLower(stripPort(rawHost))
	hostFull := strings.ToLower(rawHost)
	if hostNoPort == "" {
		return false
	}

	if len(allowed) == 0 {
		return isLoopbackHost(hostNoPort)
	}

	for _, candidate := range allowed {
		c := strings.ToLower(strings.TrimSpace(candidate))
		if c == "" {
			continue
		}
		if c == "*" {
			return true
		}
		// Entry with explicit port: require exact match against full Host header.
		// Entry without port: match against the port-stripped host.
		if strings.Contains(c, ":") && !strings.HasPrefix(c, "[") {
			// IPv4 or hostname with port (e.g. "example.com:8000").
			if c == hostFull {
				return true
			}
		} else if strings.HasPrefix(c, "[") {
			// Bracketed IPv6, with or without port (e.g. "[::1]" or "[::1]:8000").
			if strings.Contains(c, "]:") {
				if c == hostFull {
					return true
				}
			} else if stripPort(c) == hostNoPort {
				return true
			}
		} else {
			// Plain hostname or IPv4 with no port — compare host-only form.
			if c == hostNoPort {
				return true
			}
		}
	}
	return false
}

// isOriginAllowed reports whether the request's Origin header satisfies the
// configured allowlist. An empty Origin is always allowed because non-browser
// clients do not send one.
func isOriginAllowed(rawOrigin string, allowed []string) bool {
	origin := strings.ToLower(strings.TrimSpace(rawOrigin))
	if origin == "" {
		return true
	}

	for _, candidate := range allowed {
		c := strings.ToLower(strings.TrimSpace(candidate))
		if c == "" {
			continue
		}
		if c == "*" {
			return true
		}
		if c == origin {
			return true
		}
	}
	return false
}

// isLoopbackHost reports whether host (already lowercased, port-stripped) is
// one of the loopback names that the default-empty allowlist permits.
func isLoopbackHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	// Any IPv4 address in 127.0.0.0/8 is loopback.
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// stripPort removes the optional ":port" suffix from a Host header value,
// handling bracketed IPv6 forms like "[::1]:8080" and "[::1]".
func stripPort(host string) string {
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "[") {
		// Bracketed IPv6 — find matching ']'.
		if end := strings.IndexByte(host, ']'); end != -1 {
			return host[1:end]
		}
		return host
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// writeForbidden returns a 403 JSON error describing the rejection and emits a
// structured warning so operators can detect probing or misconfiguration.
func writeForbidden(w http.ResponseWriter, r *http.Request, reason, host, origin string) {
	logger.Warnf(
		"httpsecurity: rejected request: reason=%q remote=%s method=%s path=%s host=%q origin=%q",
		reason, r.RemoteAddr, r.Method, r.URL.Path, host, origin,
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             "forbidden",
		"error_description": "request rejected by host/origin policy: " + reason,
	})
}
