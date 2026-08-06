package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemovedHTTPFlagsAreRejected(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "aks-mcp")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build command failed: %v\n%s", err, output)
	}

	for _, removedFlag := range []string{"--transport=sse", "--host=127.0.0.1", "--port=8000", "--oauth-enabled"} {
		command := exec.Command(binary, removedFlag)
		output, err := command.CombinedOutput()
		if err == nil {
			t.Fatalf("%s unexpectedly succeeded", removedFlag)
		}
		if !strings.Contains(string(output), "unknown flag") {
			t.Errorf("%s returned unexpected output:\n%s", removedFlag, output)
		}
	}
}
