package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/rs/zerolog"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// AuthContextKey is the key for auth context in request context
	AuthContextKey contextKey = "auth_context"
)

// Middleware provides authentication middleware
type Middleware struct {
	config     *AuthConfig
	jwtService *JWTService
	db         *database.DB
	logger     zerolog.Logger
	rateLimiter *RateLimiter
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(config *AuthConfig, db *database.DB, logger zerolog.Logger) *Middleware {
	if config == nil {
		config = DefaultAuthConfig()
	}

	return &Middleware{
		config:      config,
		jwtService:  NewJWTService(config.JWTSecret, config.JWTExpirationHours),
		db:          db,
		logger:      logger.With().Str("component", "auth-middleware").Logger(),
		rateLimiter: NewRateLimiter(config.RateLimitPerMinute),
	}
}

// AuthRequired is middleware that requires authentication
func (m *Middleware) AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if endpoint is public
		if m.isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth if not required (development mode)
		if !m.config.RequireAuth {
			next.ServeHTTP(w, r)
			return
		}

		// Try JWT authentication first
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				authCtx, err := m.authenticateJWT(token)
				if err != nil {
					m.logger.Warn().Err(err).Msg("JWT authentication failed")
					m.respondUnauthorized(w, "Invalid or expired token")
					return
				}

				// Add auth context to request
				ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try API key authentication
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			authCtx, err := m.authenticateAPIKey(r.Context(), apiKey)
			if err != nil {
				m.logger.Warn().Err(err).Msg("API key authentication failed")
				m.respondUnauthorized(w, "Invalid API key")
				return
			}

			// Add auth context to request
			ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// No valid authentication provided
		m.respondUnauthorized(w, "Authentication required")
	})
}

// RateLimit is middleware that enforces rate limiting
func (m *Middleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get identifier (user ID, API key, or IP)
		identifier := m.getIdentifier(r)

		// Check rate limit
		if !m.rateLimiter.Allow(identifier) {
			info := m.rateLimiter.GetInfo(identifier)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", info.Reset.Format(time.RFC3339))

			m.respondRateLimited(w)
			return
		}

		// Add rate limit headers
		info := m.rateLimiter.GetInfo(identifier)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
		w.Header().Set("X-RateLimit-Reset", info.Reset.Format(time.RFC3339))

		next.ServeHTTP(w, r)
	})
}

// RequirePermission creates middleware that requires specific permissions
func (m *Middleware) RequirePermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				m.respondUnauthorized(w, "Authentication required")
				return
			}

			if !authCtx.HasPermission(perm) {
				m.respondForbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates middleware that requires specific role
func (m *Middleware) RequireRole(role Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				m.respondUnauthorized(w, "Authentication required")
				return
			}

			if authCtx.Role != string(role) && authCtx.Role != string(RoleAdmin) {
				m.respondForbidden(w, "Insufficient role")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Private methods

func (m *Middleware) authenticateJWT(token string) (*AuthContext, error) {
	claims, err := m.jwtService.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	permissions := make([]Permission, len(claims.Permissions))
	for i, p := range claims.Permissions {
		permissions[i] = Permission(p)
	}

	return &AuthContext{
		UserID:      claims.UserID,
		Email:       claims.Email,
		Role:        claims.Role,
		Permissions: permissions,
		AuthType:    AuthTypeJWT,
	}, nil
}

func (m *Middleware) authenticateAPIKey(ctx context.Context, apiKey string) (*AuthContext, error) {
	// Hash the API key
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// Query database for API key
	query := `
		SELECT
			ak.id, ak.user_id, ak.permissions, ak.active, ak.expires_at,
			u.email, u.role
		FROM api_keys ak
		JOIN users u ON ak.user_id = u.id
		WHERE ak.key_hash = $1
		AND ak.active = true
		AND u.active = true
	`

	var apiKeyID, userID, email, role string
	var active bool
	var expiresAt *time.Time
	var permissionsJSON []byte

	err := m.db.QueryRowContext(ctx, query, keyHash).Scan(
		&apiKeyID,
		&userID,
		&permissionsJSON,
		&active,
		&expiresAt,
		&email,
		&role,
	)

	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Check expiration
	if expiresAt != nil && time.Now().UTC().After(*expiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	// Parse permissions
	var permStrings []string
	if len(permissionsJSON) > 0 {
		// Assuming PostgreSQL array format
		// For simplicity, get permissions from role
		perms := RolePermissions[Role(role)]
		permStrings = make([]string, len(perms))
		for i, p := range perms {
			permStrings[i] = string(p)
		}
	}

	permissions := make([]Permission, len(permStrings))
	for i, p := range permStrings {
		permissions[i] = Permission(p)
	}

	// Update last used timestamp
	go m.updateAPIKeyLastUsed(apiKeyID)

	return &AuthContext{
		UserID:      userID,
		Email:       email,
		Role:        role,
		Permissions: permissions,
		AuthType:    AuthTypeAPIKey,
		APIKeyID:    apiKeyID,
	}, nil
}

func (m *Middleware) updateAPIKeyLastUsed(apiKeyID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`
	_, err := m.db.ExecContext(ctx, query, apiKeyID)
	if err != nil {
		m.logger.Warn().Err(err).Str("api_key_id", apiKeyID).Msg("Failed to update API key last used")
	}
}

func (m *Middleware) isPublicEndpoint(path string) bool {
	for _, endpoint := range m.config.PublicEndpoints {
		if strings.HasPrefix(path, endpoint) {
			return true
		}
	}
	return false
}

func (m *Middleware) getIdentifier(r *http.Request) string {
	authCtx := GetAuthContext(r)
	if authCtx != nil {
		if authCtx.APIKeyID != "" {
			return "apikey:" + authCtx.APIKeyID
		}
		if authCtx.UserID != "" {
			return "user:" + authCtx.UserID
		}
	}

	// Fallback to IP address
	return "ip:" + r.RemoteAddr
}

func (m *Middleware) respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `{"error": "%s"}`, message)
}

func (m *Middleware) respondForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `{"error": "%s"}`, message)
}

func (m *Middleware) respondRateLimited(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, `{"error": "Rate limit exceeded"}`)
}

// GetAuthContext retrieves auth context from request
func GetAuthContext(r *http.Request) *AuthContext {
	ctx := r.Context().Value(AuthContextKey)
	if ctx == nil {
		return nil
	}
	authCtx, ok := ctx.(*AuthContext)
	if !ok {
		return nil
	}
	return authCtx
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	limit   int
	buckets map[string]*bucket
	mu      sync.RWMutex
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		buckets: make(map[string]*bucket),
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[identifier]

	if !exists || now.Sub(b.lastReset) >= time.Minute {
		rl.buckets[identifier] = &bucket{
			tokens:    rl.limit - 1,
			lastReset: now,
		}
		return true
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// GetInfo returns rate limit information
func (rl *RateLimiter) GetInfo(identifier string) *RateLimitInfo {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	b, exists := rl.buckets[identifier]
	if !exists {
		return &RateLimitInfo{
			Limit:     rl.limit,
			Remaining: rl.limit,
			Reset:     time.Now().Add(time.Minute),
		}
	}

	return &RateLimitInfo{
		Limit:     rl.limit,
		Remaining: b.tokens,
		Reset:     b.lastReset.Add(time.Minute),
	}
}
