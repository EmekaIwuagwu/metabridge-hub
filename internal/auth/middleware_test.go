package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mock database for testing
type mockDB struct{}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) interface{} {
	return nil
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	return nil, nil
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(5) // 5 requests per minute

	identifier := "test-user"

	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		if !limiter.Allow(identifier) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be blocked
	if limiter.Allow(identifier) {
		t.Error("6th request should be blocked")
	}

	// Check rate limit info
	info := limiter.GetInfo(identifier)
	if info.Remaining != 0 {
		t.Errorf("Expected 0 remaining tokens, got %d", info.Remaining)
	}
	if info.Limit != 5 {
		t.Errorf("Expected limit of 5, got %d", info.Limit)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter := NewRateLimiter(2) // 2 requests per minute

	identifier := "test-user"

	// Use up the limit
	limiter.Allow(identifier)
	limiter.Allow(identifier)

	// Should be blocked
	if limiter.Allow(identifier) {
		t.Error("Should be blocked after using limit")
	}

	// Manually reset the bucket to simulate time passing
	limiter.buckets[identifier] = &bucket{
		tokens:    2,
		lastReset: time.Now(),
	}

	// Should work again after reset
	if !limiter.Allow(identifier) {
		t.Error("Should be allowed after reset")
	}
}

func TestRateLimiter_MultipleIdentifiers(t *testing.T) {
	limiter := NewRateLimiter(3)

	// Each identifier should have independent limits
	if !limiter.Allow("user1") {
		t.Error("user1 first request should be allowed")
	}
	if !limiter.Allow("user2") {
		t.Error("user2 first request should be allowed")
	}

	// Use up user1's limit
	limiter.Allow("user1")
	limiter.Allow("user1")
	if limiter.Allow("user1") {
		t.Error("user1 should be blocked after 3 requests")
	}

	// user2 should still have tokens
	if !limiter.Allow("user2") {
		t.Error("user2 should still be allowed")
	}
}

func TestAuthContext_HasPermission(t *testing.T) {
	testCases := []struct {
		name       string
		ctx        *AuthContext
		permission Permission
		expected   bool
	}{
		{
			name: "admin has all permissions",
			ctx: &AuthContext{
				Role:        string(RoleAdmin),
				Permissions: []Permission{PermissionAdmin},
			},
			permission: PermissionWriteMessages,
			expected:   true,
		},
		{
			name: "user has specific permission",
			ctx: &AuthContext{
				Role:        string(RoleUser),
				Permissions: []Permission{PermissionReadMessages, PermissionWriteMessages},
			},
			permission: PermissionWriteMessages,
			expected:   true,
		},
		{
			name: "user lacks permission",
			ctx: &AuthContext{
				Role:        string(RoleReadOnly),
				Permissions: []Permission{PermissionReadMessages},
			},
			permission: PermissionWriteMessages,
			expected:   false,
		},
		{
			name: "empty permissions",
			ctx: &AuthContext{
				Role:        string(RoleUser),
				Permissions: []Permission{},
			},
			permission: PermissionReadMessages,
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.ctx.HasPermission(tc.permission)
			if result != tc.expected {
				t.Errorf("HasPermission(%s) = %v, want %v",
					tc.permission, result, tc.expected)
			}
		})
	}
}

func TestAuthContext_IsAdmin(t *testing.T) {
	testCases := []struct {
		name     string
		ctx      *AuthContext
		expected bool
	}{
		{
			name: "admin role",
			ctx: &AuthContext{
				Role: string(RoleAdmin),
			},
			expected: true,
		},
		{
			name: "developer role",
			ctx: &AuthContext{
				Role: string(RoleDeveloper),
			},
			expected: false,
		},
		{
			name: "user role",
			ctx: &AuthContext{
				Role: string(RoleUser),
			},
			expected: false,
		},
		{
			name: "readonly role",
			ctx: &AuthContext{
				Role: string(RoleReadOnly),
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.ctx.IsAdmin()
			if result != tc.expected {
				t.Errorf("IsAdmin() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestMiddleware_IsPublicEndpoint(t *testing.T) {
	config := &AuthConfig{
		PublicEndpoints: []string{"/health", "/ready", "/auth/login"},
	}

	middleware := &Middleware{
		config: config,
	}

	testCases := []struct {
		path     string
		expected bool
	}{
		{"/health", true},
		{"/ready", true},
		{"/auth/login", true},
		{"/auth/logout", false},
		{"/v1/messages", false},
		{"/health/deep", true},      // prefix match
		{"/auth/login/extra", true}, // prefix match
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := middleware.isPublicEndpoint(tc.path)
			if result != tc.expected {
				t.Errorf("isPublicEndpoint(%s) = %v, want %v",
					tc.path, result, tc.expected)
			}
		})
	}
}

func TestMiddleware_GetIdentifier(t *testing.T) {
	middleware := &Middleware{}

	testCases := []struct {
		name     string
		setupReq func(*http.Request)
		expected string
	}{
		{
			name: "with API key context",
			setupReq: func(r *http.Request) {
				ctx := &AuthContext{
					APIKeyID: "key-123",
				}
				*r = *r.WithContext(context.WithValue(r.Context(), AuthContextKey, ctx))
			},
			expected: "apikey:key-123",
		},
		{
			name: "with user context",
			setupReq: func(r *http.Request) {
				ctx := &AuthContext{
					UserID: "user-456",
				}
				*r = *r.WithContext(context.WithValue(r.Context(), AuthContextKey, ctx))
			},
			expected: "user:user-456",
		},
		{
			name: "with no auth context (fallback to IP)",
			setupReq: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:12345"
			},
			expected: "ip:192.168.1.1:12345",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tc.setupReq(req)

			result := middleware.getIdentifier(req)
			if result != tc.expected {
				t.Errorf("getIdentifier() = %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestGetAuthContext(t *testing.T) {
	testCases := []struct {
		name     string
		setupReq func(*http.Request)
		expected *AuthContext
	}{
		{
			name: "valid auth context",
			setupReq: func(r *http.Request) {
				ctx := &AuthContext{
					UserID: "user-123",
					Email:  "test@example.com",
					Role:   string(RoleDeveloper),
				}
				*r = *r.WithContext(context.WithValue(r.Context(), AuthContextKey, ctx))
			},
			expected: &AuthContext{
				UserID: "user-123",
				Email:  "test@example.com",
				Role:   string(RoleDeveloper),
			},
		},
		{
			name: "no auth context",
			setupReq: func(r *http.Request) {
				// Don't add auth context
			},
			expected: nil,
		},
		{
			name: "wrong type in context",
			setupReq: func(r *http.Request) {
				*r = *r.WithContext(context.WithValue(r.Context(), AuthContextKey, "wrong-type"))
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tc.setupReq(req)

			result := GetAuthContext(req)

			if tc.expected == nil {
				if result != nil {
					t.Errorf("Expected nil context, got %+v", result)
				}
			} else {
				if result == nil {
					t.Error("Expected non-nil context, got nil")
				} else {
					if result.UserID != tc.expected.UserID {
						t.Errorf("UserID = %s, want %s", result.UserID, tc.expected.UserID)
					}
					if result.Email != tc.expected.Email {
						t.Errorf("Email = %s, want %s", result.Email, tc.expected.Email)
					}
					if result.Role != tc.expected.Role {
						t.Errorf("Role = %s, want %s", result.Role, tc.expected.Role)
					}
				}
			}
		})
	}
}

func TestRolePermissions(t *testing.T) {
	// Test that all roles have defined permissions
	for role, perms := range RolePermissions {
		if len(perms) == 0 {
			t.Errorf("Role %s has no permissions defined", role)
		}
	}

	// Admin should have admin permission
	adminPerms := RolePermissions[RoleAdmin]
	hasAdmin := false
	for _, perm := range adminPerms {
		if perm == PermissionAdmin {
			hasAdmin = true
			break
		}
	}
	if !hasAdmin {
		t.Error("Admin role should have admin permission")
	}

	// Developer should have write permissions
	devPerms := RolePermissions[RoleDeveloper]
	hasWrite := false
	for _, perm := range devPerms {
		if perm == PermissionWriteMessages || perm == PermissionWriteBatches {
			hasWrite = true
			break
		}
	}
	if !hasWrite {
		t.Error("Developer role should have write permissions")
	}

	// Readonly should only have read permissions
	readonlyPerms := RolePermissions[RoleReadOnly]
	hasWrite = false
	for _, perm := range readonlyPerms {
		if perm == PermissionWriteMessages ||
			perm == PermissionWriteBatches ||
			perm == PermissionWriteWebhooks ||
			perm == PermissionWriteRoutes {
			hasWrite = true
			break
		}
	}
	if hasWrite {
		t.Error("Readonly role should not have write permissions")
	}
}
