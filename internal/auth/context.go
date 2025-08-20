package auth

// UserContext holds authenticated user information
type UserContext struct {
	UserID   string   `json:"user_id"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	TenantID string   `json:"tenant_id"`
	Scopes   []string `json:"scopes"`
	IsAdmin  bool     `json:"is_admin"`
}
