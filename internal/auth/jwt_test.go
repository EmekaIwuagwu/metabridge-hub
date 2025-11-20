package auth

import (
	"testing"
	"time"
)

func TestJWTService_GenerateToken(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	expiryHours := 24
	service := NewJWTService(secret, expiryHours)

	user := &User{
		ID:    "user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  string(RoleDeveloper),
	}

	token, expiresAt, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}

	if expiresAt.Before(time.Now()) {
		t.Error("Token expiration should be in the future")
	}

	// Token should expire approximately 24 hours from now
	expectedExpiry := time.Now().Add(24 * time.Hour)
	if expiresAt.Sub(expectedExpiry) > time.Minute {
		t.Errorf("Token expiry mismatch: got %v, expected around %v", expiresAt, expectedExpiry)
	}
}

func TestJWTService_ValidateToken(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	service := NewJWTService(secret, 24)

	user := &User{
		ID:    "user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  string(RoleDeveloper),
	}

	// Generate a token
	token, _, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate the token
	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Check claims
	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, user.ID)
	}

	if claims.Email != user.Email {
		t.Errorf("Email mismatch: got %s, want %s", claims.Email, user.Email)
	}

	if claims.Role != user.Role {
		t.Errorf("Role mismatch: got %s, want %s", claims.Role, user.Role)
	}

	// Check permissions were assigned
	if len(claims.Permissions) == 0 {
		t.Error("Permissions should not be empty for developer role")
	}
}

func TestJWTService_ValidateToken_InvalidSignature(t *testing.T) {
	service := NewJWTService("correct-secret", 24)
	wrongService := NewJWTService("wrong-secret", 24)

	user := &User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  string(RoleUser),
	}

	// Generate token with correct secret
	token, _, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with wrong secret
	_, err = wrongService.ValidateToken(token)
	if err == nil {
		t.Error("Expected validation to fail with wrong secret")
	}

	if err.Error() != "invalid signature" {
		t.Errorf("Expected 'invalid signature' error, got: %v", err)
	}
}

func TestJWTService_ValidateToken_ExpiredToken(t *testing.T) {
	// Create service with very short expiry
	service := NewJWTService("test-secret", 0) // 0 hours = immediate expiry

	user := &User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  string(RoleUser),
	}

	// Generate token (will be expired immediately)
	token, _, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait a moment to ensure token is expired
	time.Sleep(100 * time.Millisecond)

	// Try to validate expired token
	_, err = service.ValidateToken(token)
	if err == nil {
		t.Error("Expected validation to fail for expired token")
	}

	if err.Error() != "token expired" {
		t.Errorf("Expected 'token expired' error, got: %v", err)
	}
}

func TestJWTService_ValidateToken_MalformedToken(t *testing.T) {
	service := NewJWTService("test-secret", 24)

	testCases := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"single part", "invalid"},
		{"two parts", "invalid.token"},
		{"four parts", "too.many.parts.here"},
		{"invalid base64", "not-base64.not-base64.not-base64"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.ValidateToken(tc.token)
			if err == nil {
				t.Errorf("Expected validation to fail for %s", tc.name)
			}
		})
	}
}

func TestJWTService_RefreshToken(t *testing.T) {
	secret := "test-secret-key"
	service := NewJWTService(secret, 1) // 1 hour expiry

	user := &User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  string(RoleDeveloper),
	}

	// Generate initial token
	oldToken, oldExpiresAt, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait a moment
	time.Sleep(100 * time.Millisecond)

	// Refresh the token
	newToken, newExpiresAt, err := service.RefreshToken(oldToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	// Tokens should be different
	if oldToken == newToken {
		t.Error("Refreshed token should be different from old token")
	}

	// New expiry should be later than old expiry
	if !newExpiresAt.After(oldExpiresAt) {
		t.Error("Refreshed token expiry should be later than original")
	}

	// Validate new token
	claims, err := service.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("Failed to validate refreshed token: %v", err)
	}

	// Claims should match original user
	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch after refresh: got %s, want %s", claims.UserID, user.ID)
	}
}

func TestJWTService_RolePermissions(t *testing.T) {
	service := NewJWTService("test-secret", 24)

	testCases := []struct {
		role               string
		expectedMinPerms   int
		shouldHaveAdmin    bool
		shouldHaveWriteAll bool
	}{
		{string(RoleAdmin), 1, true, false},
		{string(RoleDeveloper), 8, false, true},
		{string(RoleUser), 6, false, true},
		{string(RoleReadOnly), 4, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.role, func(t *testing.T) {
			user := &User{
				ID:    "user-123",
				Email: "test@example.com",
				Role:  tc.role,
			}

			token, _, err := service.GenerateToken(user)
			if err != nil {
				t.Fatalf("Failed to generate token for %s: %v", tc.role, err)
			}

			claims, err := service.ValidateToken(token)
			if err != nil {
				t.Fatalf("Failed to validate token for %s: %v", tc.role, err)
			}

			if len(claims.Permissions) < tc.expectedMinPerms {
				t.Errorf("Expected at least %d permissions for %s, got %d",
					tc.expectedMinPerms, tc.role, len(claims.Permissions))
			}

			// Check for admin permission
			hasAdmin := false
			hasWriteBatches := false
			for _, perm := range claims.Permissions {
				if perm == string(PermissionAdmin) {
					hasAdmin = true
				}
				if perm == string(PermissionWriteBatches) {
					hasWriteBatches = true
				}
			}

			if hasAdmin != tc.shouldHaveAdmin {
				t.Errorf("Admin permission mismatch for %s: got %v, want %v",
					tc.role, hasAdmin, tc.shouldHaveAdmin)
			}

			if tc.shouldHaveWriteAll && !hasWriteBatches {
				t.Errorf("Expected %s to have write permissions", tc.role)
			}
		})
	}
}
