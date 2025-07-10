package security

import (
	"strings"
)

// Command type constants
const (
	CommandTypeAz = "az"
)

var (
	// AzReadOperations defines az operations that don't modify state
	AzReadOperations = []string{
		// Cluster information commands
		"az aks show",
		"az aks list",
		"az aks get-versions",
		"az aks get-upgrades",
		"az aks check-acr",
		"az aks check-network outbound",
		"az aks browse",

		// Addon commands
		"az aks addon list",
		"az aks addon show",

		// Nodepool commands
		"az aks nodepool list",
		"az aks nodepool show",
		"az aks nodepool get-upgrades",

		// Operation and snapshot commands
		"az aks operation",
		"az aks snapshot list",
		"az aks snapshot show",

		// Trusted access commands
		"az aks trustedaccess rolebinding list",
		"az aks trustedaccess rolebinding show",

		// Other read operations
		"az aks install-cli",
		// "az aks get-credentials", // Commented out as it may require special handling

		// Account management commands
		"az account list",
		"az account set",
		"az login",

		// Azure Advisor commands (read-only)
		"az advisor recommendation list",
		"az advisor recommendation show",

		// Azure Monitor metrics commands (read-only)
		"az monitor metrics list",
		"az monitor metrics list-definitions",
		"az monitor metrics list-namespaces",

		// Other general commands
		"az find",
		"az version",
		"az help",
		"az config",
		"az group list",
		"az group show",
		"az resource list",
		"az resource show",
	}
)

// Validator handles validation of commands against security configuration
type Validator struct {
	secConfig *SecurityConfig
}

// NewValidator creates a new Validator instance with the given security configuration
func NewValidator(secConfig *SecurityConfig) *Validator {
	return &Validator{
		secConfig: secConfig,
	}
}

// ValidationError represents a security validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// getReadOperationsList returns the appropriate list of read operations based on command type
func (v *Validator) getReadOperationsList(commandType string) []string {
	switch commandType {
	case CommandTypeAz:
		return AzReadOperations
	default:
		return []string{}
	}
}

// ValidateCommand validates a command against all security settings
// The command parameter should be the full command string (e.g., "az aks show --name myCluster")
// AzReadOperations should now contain full command prefixes with "az" included
func (v *Validator) ValidateCommand(command, commandType string) error {
	readOperations := v.getReadOperationsList(commandType)

	// Check for command injection attempts
	if err := v.validateCommandInjection(command); err != nil {
		return err
	}

	// Check access level restrictions
	if err := v.validateAccessLevel(command, readOperations); err != nil {
		return err
	}

	return nil
}

// validateCommandInjection checks for command injection patterns
func (v *Validator) validateCommandInjection(command string) error {
	// Check if this contains a here document operator
	containsHereDoc := strings.Contains(command, "<<")

	// Validate here document if present
	if containsHereDoc {
		if err := v.validateHereDocument(command); err != nil {
			return err
		}
	}

	// Define dangerous characters and patterns that could be used for command injection
	dangerousPatterns := []string{
		";",  // Command separator
		"|",  // Pipe
		"&",  // Background execution or AND operator
		"`",  // Command substitution (backticks)
		"&&", // AND operator
		"||", // OR operator
		">>", // Append redirection
		// Note: "<<" (here document) is allowed for legitimate use cases like providing JSON/YAML payloads
		">",  // Output redirection
		"$(", // Command substitution
		"${", // Variable substitution that could be misused
		// Note: "<" is handled separately below to allow "<<" but block single "<"
	}

	// Only block newlines and carriage returns if it's NOT a complete here document
	isCompleteHereDoc := containsHereDoc && v.isCompleteHereDocument(command)
	if !isCompleteHereDoc {
		dangerousPatterns = append(dangerousPatterns, "\n", "\r")
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return &ValidationError{Message: "Error: Command contains potentially dangerous characters or patterns"}
		}
	}

	// Special handling for input redirection - allow "<<" but block single "<"
	if strings.Contains(command, "<") {
		// If command contains "<", make sure all instances are part of "<<"
		// This prevents cases like "az aks show < malicious_file"
		for i := 0; i < len(command); i++ {
			if command[i] == '<' {
				// Check if this '<' is part of '<<'
				if i+1 >= len(command) || command[i+1] != '<' {
					// This is a standalone '<' which is dangerous
					return &ValidationError{Message: "Error: Command contains potentially dangerous characters or patterns"}
				}
				// Skip the next '<' since we've verified it's part of '<<'
				i++
			}
		}
	}

	return nil
}

// validateAccessLevel validates if a command is allowed based on the current access level
func (v *Validator) validateAccessLevel(command string, readOperations []string) error {
	// Check if this is a read operation
	isReadOperation := v.isReadOperation(command, readOperations)

	// Handle restrictions based on access level
	switch v.secConfig.AccessLevel {
	case "readonly":
		if !isReadOperation {
			return &ValidationError{Message: "Error: Cannot execute write operations in read-only mode"}
		}
	case "readwrite":
		// All read and write operations are allowed, but not admin operations
		// Admin operations are handled separately by not registering those commands
	case "admin":
		// All operations are allowed
	default:
		// Default to readwrite behavior for unknown access levels
		// This could alternatively return an error for invalid access levels
	}

	return nil
}

// isReadOperation checks if a command is a read operation
func (v *Validator) isReadOperation(command string, allowedOperations []string) bool {
	// Check if the command contains help flags - these are always read-only
	if strings.Contains(command, "--help") || strings.Contains(command, " -h ") || strings.HasSuffix(command, " -h") {
		return true
	}

	// Normalize command by removing any options/arguments
	// This extracts the base command like "az aks show" from "az aks show --name myCluster"
	cmdParts := strings.Fields(command)

	if len(cmdParts) == 0 || cmdParts[0] != CommandTypeAz {
		return false
	}

	// For az commands, we need to handle various command structures:
	// - "az version" (2 parts)
	// - "az aks show" (3 parts)
	// - "az aks check-network outbound" (4 parts)
	// - "az aks trustedaccess rolebinding list" (5 parts)
	// - "az aks nodepool get-upgrades" (4 parts)

	// We'll try to match the longest possible command first by checking against allowed operations
	for _, allowed := range allowedOperations {
		allowedParts := strings.Fields(allowed)

		// Skip if the allowed operation has more parts than our command
		if len(allowedParts) > len(cmdParts) {
			continue
		}

		// Check if the command starts with this allowed operation
		match := true
		for i, allowedPart := range allowedParts {
			if i >= len(cmdParts) || cmdParts[i] != allowedPart {
				match = false
				break
			}
		}

		if match {
			return true
		}
	}

	return false
}

// validateHereDocument validates the structure of here document commands
func (v *Validator) validateHereDocument(command string) error {
	// A complete here document should have:
	// 1. The << operator
	// 2. A delimiter after <<
	// 3. Either content with terminator or be a legitimate single-line command

	// Find all << occurrences
	hereDocIndex := strings.Index(command, "<<")
	if hereDocIndex == -1 {
		return nil // No here document
	}

	// Extract everything after <<
	afterHereDoc := command[hereDocIndex+2:]
	afterHereDoc = strings.TrimSpace(afterHereDoc)

	// If there's nothing after <<, it's malformed
	if afterHereDoc == "" {
		return &ValidationError{Message: "Error: Command contains potentially dangerous characters or patterns"}
	}

	// Split by whitespace to get the delimiter
	parts := strings.Fields(afterHereDoc)
	if len(parts) == 0 {
		return &ValidationError{Message: "Error: Command contains potentially dangerous characters or patterns"}
	}

	// Extract the part before << to check if it has sufficient arguments
	beforeHereDoc := command[:hereDocIndex]
	beforeHereDocParts := strings.Fields(beforeHereDoc)

	// If the command ends with just "< delimiter" and has minimal arguments
	// (like "az aks create << EOF"), consider it incomplete and dangerous
	// But if it has more arguments (like "az aks create --name test << EOF"), allow it
	if len(parts) == 1 && !strings.Contains(command, "\n") && !strings.Contains(command, "\r") {
		// Check if the command before << has sufficient arguments
		// Commands like "az aks create << EOF" (3 parts) should be blocked
		// Commands like "az aks create --name test << EOF" (5+ parts) should be allowed
		if len(beforeHereDocParts) <= 3 {
			return &ValidationError{Message: "Error: Command contains potentially dangerous characters or patterns"}
		}
	}

	return nil
}

// isCompleteHereDocument checks if a command contains a complete here document
func (v *Validator) isCompleteHereDocument(command string) bool {
	// A complete here document should have content and/or be multi-line
	if !strings.Contains(command, "<<") {
		return false
	}

	// If it contains newlines or carriage returns, it's likely a complete here document
	if strings.Contains(command, "\n") || strings.Contains(command, "\r") {
		return true
	}

	// For single-line here documents, we need to be more careful
	// Simple case: "az deployment create --template-body << EOF {content} EOF"
	hereDocIndex := strings.Index(command, "<<")
	afterHereDoc := command[hereDocIndex+2:]
	afterHereDoc = strings.TrimSpace(afterHereDoc)

	parts := strings.Fields(afterHereDoc)

	// If we have more than just the delimiter, it might be a complete single-line here doc
	if len(parts) > 1 {
		return true
	}

	return false
}

// isReadOperation determines if a command is a read-only operation
