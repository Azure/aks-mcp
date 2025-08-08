package inspektorgadget

import "testing"

func TestGadgetVersionForIGVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"Valid version", "0.42.0", "v0.42.0"},
		{"Invalid version", "invalid", "latest"},
		{"Empty version", "", "latest"},
		{"Canonical version", "1.2.3+build", "latest"},
		{"Semver with pre-release", "1.2.3-alpha", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gadgetVersionFor(tt.version)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
