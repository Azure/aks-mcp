package config

import (
	"strings"
	"testing"
)

func TestParseFlagsAcceptsStdioTransportCompatibility(t *testing.T) {
	tests := [][]string{
		{"--transport", "stdio"},
		{"--transport=stdio"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cfg := NewConfig()
			if _, _, _, err := cfg.parseFlags(args); err != nil {
				t.Fatalf("parseFlags(%q) returned error: %v", args, err)
			}
		})
	}
}

func TestParseFlagsRejectsRemoteTransportAndListenerOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "http transport", args: []string{"--transport", "http"}, wantErr: `unsupported transport "http": only stdio is supported`},
		{name: "sse transport", args: []string{"--transport=sse"}, wantErr: `unsupported transport "sse": only stdio is supported`},
		{name: "streamable transport", args: []string{"--transport", "streamable-http"}, wantErr: `unsupported transport "streamable-http": only stdio is supported`},
		{name: "host", args: []string{"--host", "127.0.0.1"}, wantErr: "unknown flag: --host"},
		{name: "port", args: []string{"--port", "8080"}, wantErr: "unknown flag: --port"},
		{name: "oauth", args: []string{"--oauth-enabled"}, wantErr: "unknown flag: --oauth-enabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			_, _, _, err := cfg.parseFlags(tt.args)
			if err == nil {
				t.Fatalf("parseFlags(%q) succeeded, expected an error", tt.args)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("parseFlags(%q) error = %q, want it to contain %q", tt.args, err, tt.wantErr)
			}
		})
	}
}
