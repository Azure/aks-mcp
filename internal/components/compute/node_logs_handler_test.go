package compute

import (
	"testing"
)

// Test isValidLogType function
func TestIsValidLogType(t *testing.T) {
	tests := []struct {
		name     string
		logType  string
		expected bool
	}{
		{"valid kubelet", LogTypeKubelet, true},
		{"valid containerd", LogTypeContainerd, true},
		{"valid kernel", LogTypeKernel, true},
		{"valid syslog", LogTypeSyslog, true},
		{"invalid empty", "", false},
		{"invalid unknown", "unknown", false},
		{"invalid case", "KUBELET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidLogType(tt.logType)
			if result != tt.expected {
				t.Errorf("isValidLogType(%q) = %v, want %v", tt.logType, result, tt.expected)
			}
		})
	}
}

// Test isValidLogLevel function
func TestIsValidLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected bool
	}{
		{"valid ERROR", LogLevelError, true},
		{"valid WARN", LogLevelWarn, true},
		{"valid INFO", LogLevelInfo, true},
		{"invalid empty", "", false},
		{"invalid unknown", "DEBUG", false},
		{"invalid case", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidLogLevel(tt.level)
			if result != tt.expected {
				t.Errorf("isValidLogLevel(%q) = %v, want %v", tt.level, result, tt.expected)
			}
		})
	}
}

// Test buildJournalctlCommand function
func TestBuildJournalctlCommand(t *testing.T) {
	tests := []struct {
		name     string
		unit     string
		lines    int
		since    string
		level    string
		expected string
	}{
		{
			name:     "basic kubelet command",
			unit:     "kubelet",
			lines:    500,
			since:    "",
			level:    LogLevelInfo,
			expected: "journalctl -u kubelet --no-pager -n 500",
		},
		{
			name:     "kubelet with since 1h",
			unit:     "kubelet",
			lines:    500,
			since:    "1h",
			level:    LogLevelInfo,
			expected: "journalctl -u kubelet --no-pager --since '1 hour ago'",
		},
		{
			name:     "kubelet with since 30m",
			unit:     "kubelet",
			lines:    500,
			since:    "30m",
			level:    LogLevelInfo,
			expected: "journalctl -u kubelet --no-pager --since '30 minutes ago'",
		},
		{
			name:     "kubelet with ERROR level",
			unit:     "kubelet",
			lines:    500,
			since:    "",
			level:    LogLevelError,
			expected: "journalctl -u kubelet --no-pager -n 500 | grep -iE '^[A-Z][a-z]+ [0-9]+ [0-9:]+ .* E[0-9]+|error'",
		},
		{
			name:     "kubelet with WARN level",
			unit:     "kubelet",
			lines:    500,
			since:    "",
			level:    LogLevelWarn,
			expected: "journalctl -u kubelet --no-pager -n 500 | grep -iE '^[A-Z][a-z]+ [0-9]+ [0-9:]+ .* [EW][0-9]+|error|warn'",
		},
		{
			name:     "containerd with since and ERROR",
			unit:     "containerd",
			lines:    1000,
			since:    "2h",
			level:    LogLevelError,
			expected: "journalctl -u containerd --no-pager --since '2 hours ago' | grep -iE '^[A-Z][a-z]+ [0-9]+ [0-9:]+ .* E[0-9]+|error'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildJournalctlCommand(tt.unit, tt.lines, tt.since, tt.level)
			if result != tt.expected {
				t.Errorf("buildJournalctlCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test buildDmesgCommand function
func TestBuildDmesgCommand(t *testing.T) {
	tests := []struct {
		name     string
		lines    int
		level    string
		expected string
	}{
		{
			name:     "basic dmesg",
			lines:    500,
			level:    LogLevelInfo,
			expected: "dmesg -T | tail -n 500",
		},
		{
			name:     "dmesg with ERROR level",
			lines:    500,
			level:    LogLevelError,
			expected: "dmesg -T -l err,crit,alert,emerg | tail -n 500",
		},
		{
			name:     "dmesg with WARN level",
			lines:    1000,
			level:    LogLevelWarn,
			expected: "dmesg -T -l warn,err,crit,alert,emerg | tail -n 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDmesgCommand(tt.lines, tt.level)
			if result != tt.expected {
				t.Errorf("buildDmesgCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test buildSyslogCommand function
func TestBuildSyslogCommand(t *testing.T) {
	tests := []struct {
		name     string
		lines    int
		since    string
		level    string
		expected string
	}{
		{
			name:     "basic syslog",
			lines:    500,
			since:    "",
			level:    LogLevelInfo,
			expected: "journalctl --no-pager -n 500",
		},
		{
			name:     "syslog with since 1h",
			lines:    500,
			since:    "1h",
			level:    LogLevelInfo,
			expected: "journalctl --no-pager --since '1 hour ago'",
		},
		{
			name:     "syslog with ERROR level",
			lines:    500,
			since:    "",
			level:    LogLevelError,
			expected: "journalctl --no-pager -n 500 -p err",
		},
		{
			name:     "syslog with WARN level and since",
			lines:    1000,
			since:    "30m",
			level:    LogLevelWarn,
			expected: "journalctl --no-pager --since '30 minutes ago' -p warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSyslogCommand(tt.lines, tt.since, tt.level)
			if result != tt.expected {
				t.Errorf("buildSyslogCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test buildLogCommand function
func TestBuildLogCommand(t *testing.T) {
	tests := []struct {
		name        string
		logType     string
		lines       int
		since       string
		level       string
		expected    string
		expectError bool
	}{
		{
			name:        "kubelet log",
			logType:     LogTypeKubelet,
			lines:       500,
			since:       "",
			level:       LogLevelInfo,
			expected:    "journalctl -u kubelet --no-pager -n 500",
			expectError: false,
		},
		{
			name:        "containerd log with ERROR",
			logType:     LogTypeContainerd,
			lines:       500,
			since:       "",
			level:       LogLevelError,
			expected:    "journalctl -u containerd --no-pager -n 500 | grep -iE '^[A-Z][a-z]+ [0-9]+ [0-9:]+ .* E[0-9]+|error'",
			expectError: false,
		},
		{
			name:        "kernel log",
			logType:     LogTypeKernel,
			lines:       500,
			since:       "",
			level:       LogLevelInfo,
			expected:    "dmesg -T | tail -n 500",
			expectError: false,
		},
		{
			name:        "syslog with since 1h",
			logType:     LogTypeSyslog,
			lines:       500,
			since:       "1h",
			level:       LogLevelInfo,
			expected:    "journalctl --no-pager --since '1 hour ago'",
			expectError: false,
		},
		{
			name:        "invalid log type",
			logType:     "invalid",
			lines:       500,
			since:       "",
			level:       LogLevelInfo,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildLogCommand(tt.logType, tt.lines, tt.since, tt.level)
			if tt.expectError {
				if err == nil {
					t.Errorf("buildLogCommand() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("buildLogCommand() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("buildLogCommand() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

// Test formatSinceParameter function
func TestFormatSinceParameter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"1 hour", "1h", "1 hour ago"},
		{"2 hours", "2h", "2 hours ago"},
		{"1 minute", "1m", "1 minute ago"},
		{"30 minutes", "30m", "30 minutes ago"},
		{"1 second", "1s", "1 second ago"},
		{"45 seconds", "45s", "45 seconds ago"},
		{"1 day", "1d", "1 day ago"},
		{"7 days", "7d", "7 days ago"},
		{"1 week", "1w", "1 week ago"},
		{"2 weeks", "2w", "2 weeks ago"},
		{"already formatted", "1 hour ago", "1 hour ago"},
		{"full timestamp", "2024-01-01 10:00:00", "2024-01-01 10:00:00"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSinceParameter(tt.input)
			if result != tt.expected {
				t.Errorf("formatSinceParameter(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test formatLogOutput function
func TestFormatLogOutput(t *testing.T) {
	clusterName := "test-cluster"
	rg := "test-rg"
	vmssName := "test-vmss"
	instanceID := "0"
	logType := "kubelet"
	output := "test log output"

	result := formatLogOutput(clusterName, rg, vmssName, instanceID, logType, output)

	// Check if the result contains expected metadata
	expectedSubstrings := []string{
		"=== AKS Node Logs ===",
		"Cluster: test-cluster",
		"Resource Group: test-rg",
		"VMSS: test-vmss",
		"Instance ID: 0",
		"Log Type: kubelet",
		"test log output",
	}

	for _, substr := range expectedSubstrings {
		if !contains(result, substr) {
			t.Errorf("formatLogOutput() missing expected substring: %q", substr)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
