package security

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// AccessLevel controls the level of operations allowed (readonly, readwrite, admin)
	AccessLevel string
}

// NewSecurityConfig creates a new SecurityConfig instance
func NewSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		AccessLevel: "readwrite",
	}
}
