package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalTransportContract(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "aks-mcp")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build command failed: %v\n%s", err, output)
	}

	for _, args := range [][]string{
		{"--transport", "stdio", "--version"},
		{"--transport=stdio", "--version"},
	} {
		command := exec.Command(binary, args...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Errorf("%v returned error: %v\n%s", args, err, output)
		}
	}

	rejected := []struct {
		flag       string
		wantOutput string
	}{
		{flag: "--transport=sse", wantOutput: `unsupported transport "sse": only stdio is supported`},
		{flag: "--host=127.0.0.1", wantOutput: "unknown flag"},
		{flag: "--port=8000", wantOutput: "unknown flag"},
		{flag: "--oauth-enabled", wantOutput: "unknown flag"},
	}
	for _, tt := range rejected {
		command := exec.Command(binary, tt.flag)
		output, err := command.CombinedOutput()
		if err == nil {
			t.Fatalf("%s unexpectedly succeeded", tt.flag)
		}
		if !strings.Contains(string(output), tt.wantOutput) {
			t.Errorf("%s returned unexpected output:\n%s", tt.flag, output)
		}
	}
}
