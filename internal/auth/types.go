package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeJWT    AuthType = "JWT"
	AuthTypeAPIKey AuthType = "API_KEY"
	AuthTypeNone   AuthType = "NONE"
)

// User represents an authenticated user
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Key         string     `json:"key"`
	KeyHash     string     `json:"-"` // Never expose hash
	Permissions []string   `json:"permissions"`
	Active      bool       `json:"active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	IssuedAt    int64    `json:"iat"`
	ExpiresAt   int64    `json:"exp"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTSecret          string
	JWTExpirationHours int
	APIKeyEnabled      bool
	RequireAuth        bool
	PublicEndpoints    []string
	RateLimitPerMinute int
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		JWTSecret:          generateSecret(32),
		JWTExpirationHours: 24,
		APIKeyEnabled:      true,
		RequireAuth:        true,
		PublicEndpoints: []string{
			"/health",
			"/ready",
			"/v1/chains",
			"/v1/routes/estimate",
		},
		RateLimitPerMinute: 100,
	}
}

// Permission represents an API permission
type Permission string

const (
	PermissionReadMessages  Permission = "messages:read"
	PermissionWriteMessages Permission = "messages:write"
	PermissionReadBatches   Permission = "batches:read"
	PermissionWriteBatches  Permission = "batches:write"
	PermissionReadWebhooks  Permission = "webhooks:read"
	PermissionWriteWebhooks Permission = "webhooks:write"
	PermissionReadRoutes    Permission = "routes:read"
	PermissionWriteRoutes   Permission = "routes:write"
	PermissionReadStats     Permission = "stats:read"
	PermissionAdmin         Permission = "admin"
)

// Role represents a user role
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleUser      Role = "user"
	RoleReadOnly  Role = "readonly"
)

// RolePermissions maps roles to their default permissions
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermissionReadMessages,
		PermissionWriteMessages,
		PermissionReadBatches,
		PermissionWriteBatches,
		PermissionReadWebhooks,
		PermissionWriteWebhooks,
		PermissionReadRoutes,
		PermissionWriteRoutes,
		PermissionReadStats,
		PermissionAdmin,
	},
	RoleDeveloper: {
		PermissionReadMessages,
		PermissionWriteMessages,
		PermissionReadBatches,
		PermissionReadWebhooks,
		PermissionWriteWebhooks,
		PermissionReadRoutes,
		PermissionWriteRoutes,
		PermissionReadStats,
	},
	RoleUser: {
		PermissionReadMessages,
		PermissionWriteMessages,
		PermissionReadBatches,
		PermissionReadRoutes,
		PermissionReadStats,
	},
	RoleReadOnly: {
		PermissionReadMessages,
		PermissionReadBatches,
		PermissionReadRoutes,
		PermissionReadStats,
	},
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user"`
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions,omitempty"`
	ExpiresIn   int      `json:"expires_in_days,omitempty"` // 0 = no expiration
}

// CreateAPIKeyResponse represents a response with a new API key
type CreateAPIKeyResponse struct {
	APIKey *APIKey `json:"api_key"`
	Key    string  `json:"key"` // Only returned on creation
}

// AuthContext represents authentication context in requests
type AuthContext struct {
	UserID      string
	Email       string
	Role        string
	Permissions []Permission
	AuthType    AuthType
	APIKeyID    string
}

// HasPermission checks if context has a specific permission
func (ac *AuthContext) HasPermission(perm Permission) bool {
	// Admin has all permissions
	if ac.Role == string(RoleAdmin) {
		return true
	}

	for _, p := range ac.Permissions {
		if p == perm || p == PermissionAdmin {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if context has any of the specified permissions
func (ac *AuthContext) HasAnyPermission(perms ...Permission) bool {
	for _, perm := range perms {
		if ac.HasPermission(perm) {
			return true
		}
	}
	return false
}

// RateLimitInfo represents rate limit information
type RateLimitInfo struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

// generateSecret generates a random secret
func generateSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "default-secret-change-me"
	}
	return hex.EncodeToString(bytes)
}

// GenerateAPIKey generates a new API key
func GenerateAPIKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return "mbh_" + hex.EncodeToString(bytes)
}
