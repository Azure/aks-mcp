package config

import (
	"os"
	"os/exec"
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

func TestParseFlagsObservableOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantExit   int
		wantOutput []string
	}{
		{
			name:       "help",
			args:       []string{"--help"},
			wantExit:   0,
			wantOutput: []string{"Usage of aks-mcp:", "--transport string", "stdio only"},
		},
		{
			name:       "version",
			args:       []string{"--version"},
			wantExit:   0,
			wantOutput: []string{"aks-mcp version", "Git commit:", "Platform:"},
		},
		{
			name:       "unsupported transport",
			args:       []string{"--transport=sse"},
			wantExit:   1,
			wantOutput: []string{`unsupported transport "sse": only stdio is supported`, "Usage of aks-mcp:", "--transport string"},
		},
		{
			name:       "removed listener option",
			args:       []string{"--host=127.0.0.1"},
			wantExit:   1,
			wantOutput: []string{"unknown flag: --host", "Usage of aks-mcp:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"-test.run=TestParseFlagsProcess", "--"}, tt.args...)
			command := exec.Command(os.Args[0], args...)
			command.Env = append(os.Environ(), "GO_WANT_PARSE_FLAGS_PROCESS=1")
			output, err := command.CombinedOutput()

			if tt.wantExit == 0 {
				if err != nil {
					t.Fatalf("ParseFlags(%q) returned error: %v\n%s", tt.args, err, output)
				}
			} else {
				exitErr, ok := err.(*exec.ExitError)
				if !ok || exitErr.ExitCode() != tt.wantExit {
					t.Fatalf("ParseFlags(%q) exit error = %v, want exit code %d\n%s", tt.args, err, tt.wantExit, output)
				}
			}
			for _, want := range tt.wantOutput {
				if !strings.Contains(string(output), want) {
					t.Errorf("ParseFlags(%q) output does not contain %q:\n%s", tt.args, want, output)
				}
			}
		})
	}
}

func TestParseFlagsProcess(t *testing.T) {
	if os.Getenv("GO_WANT_PARSE_FLAGS_PROCESS") != "1" {
		return
	}

	separator := 0
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	os.Args = append([]string{"aks-mcp"}, os.Args[separator+1:]...)
	NewConfig().ParseFlags()
}
