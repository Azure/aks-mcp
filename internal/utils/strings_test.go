package utils

import "testing"

func TestReplaceSpacesWithUnderscores(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"az monitor metrics list", "az_monitor_metrics_list"},
		{"az monitor metrics list-definitions", "az_monitor_metrics_list-definitions"},
		{"az monitor metrics list-namespaces", "az_monitor_metrics_list-namespaces"},
		{"simple command", "simple_command"},
		{"command with multiple spaces", "command_with_multiple_spaces"},
		{"no-spaces-here", "no-spaces-here"},
		{"", ""},
		{"single", "single"},
		{"  leading and trailing  ", "__leading_and_trailing__"},
	}

	for _, test := range tests {
		result := ReplaceSpacesWithUnderscores(test.input)
		if result != test.expected {
			t.Errorf("ReplaceSpacesWithUnderscores(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
