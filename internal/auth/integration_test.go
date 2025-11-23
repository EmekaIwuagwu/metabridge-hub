package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmekaIwuagwu/articium-hub/internal/auth"
)

// TestAuthenticationFlow demonstrates the complete authentication flow
func TestAuthenticationFlow(t *testing.T) {
	// Setup: Create a JWT service
	jwtService := auth.NewJWTService("test-secret-key", 24)

	// Step 1: Create a user
	user := &auth.User{
		ID:     "user-123",
		Email:  "developer@example.com",
		Name:   "Test Developer",
		Role:   string(auth.RoleDeveloper),
		Active: true,
	}

	// Step 2: Generate JWT token (simulating login)
	token, expiresAt, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	t.Logf("Generated token: %s", token[:50]+"...")
	t.Logf("Token expires at: %v", expiresAt)

	// Step 3: Validate the token
	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	t.Logf("Token validated for user: %s (%s)", claims.Email, claims.Role)

	// Step 4: Create auth context from claims
	permissions := make([]auth.Permission, len(claims.Permissions))
	for i, p := range claims.Permissions {
		permissions[i] = auth.Permission(p)
	}

	authContext := &auth.AuthContext{
		UserID:      claims.UserID,
		Email:       claims.Email,
		Role:        claims.Role,
		Permissions: permissions,
		AuthType:    auth.AuthTypeJWT,
	}

	// Step 5: Check permissions
	if !authContext.HasPermission(auth.PermissionReadMessages) {
		t.Error("Developer should have read messages permission")
	}

	if !authContext.HasPermission(auth.PermissionWriteMessages) {
		t.Error("Developer should have write messages permission")
	}

	if authContext.HasPermission(auth.PermissionAdmin) {
		t.Error("Developer should not have admin permission")
	}

	t.Log("✓ Authentication flow completed successfully")
}

// TestAPIRequestWithJWT demonstrates making an authenticated API request
func TestAPIRequestWithJWT(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key", 24)

	// Create a user and token
	user := &auth.User{
		ID:    "user-456",
		Email: "api-user@example.com",
		Role:  string(auth.RoleUser),
	}

	token, _, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create a mock HTTP handler that requires authentication
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := auth.GetAuthContext(r)
		if authCtx == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		response := map[string]interface{}{
			"message": "authenticated request successful",
			"user_id": authCtx.UserID,
			"email":   authCtx.Email,
			"role":    authCtx.Role,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Create authentication middleware
	config := auth.DefaultAuthConfig()
	config.RequireAuth = true
	config.JWTSecret = "test-secret-key"

	// Note: In a real test, we'd need a database connection
	// For this example, we're testing the JWT validation only

	// Create a request with JWT token
	req := httptest.NewRequest("GET", "/api/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Validate token and add to context manually (simulating middleware)
	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Token validation failed: %v", err)
	}

	permissions := make([]auth.Permission, len(claims.Permissions))
	for i, p := range claims.Permissions {
		permissions[i] = auth.Permission(p)
	}

	authCtx := &auth.AuthContext{
		UserID:      claims.UserID,
		Email:       claims.Email,
		Role:        claims.Role,
		Permissions: permissions,
		AuthType:    auth.AuthTypeJWT,
	}

	req = req.WithContext(auth.SetAuthContext(req.Context(), authCtx))

	// Execute request
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["user_id"] != user.ID {
		t.Errorf("Expected user_id %s, got %v", user.ID, response["user_id"])
	}

	t.Logf("✓ Authenticated API request successful: %+v", response)
}

// TestAPIRequestWithoutAuth demonstrates a request without authentication
func TestAPIRequestWithoutAuth(t *testing.T) {
	// Create a mock handler that requires auth
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := auth.GetAuthContext(r)
		if authCtx == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Create request without auth
	req := httptest.NewRequest("GET", "/api/v1/messages", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should be unauthorized
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	t.Log("✓ Unauthorized request correctly rejected")
}

// TestRateLimiting demonstrates rate limiting behavior
func TestRateLimiting(t *testing.T) {
	limiter := auth.NewRateLimiter(5) // 5 requests per minute

	identifier := "test-user-789"

	// Make requests up to the limit
	successCount := 0
	for i := 0; i < 10; i++ {
		if limiter.Allow(identifier) {
			successCount++
		}
	}

	if successCount != 5 {
		t.Errorf("Expected 5 successful requests, got %d", successCount)
	}

	// Check rate limit info
	info := limiter.GetInfo(identifier)
	t.Logf("Rate limit info: %d/%d requests remaining, resets at %v",
		info.Remaining, info.Limit, info.Reset)

	if info.Remaining != 0 {
		t.Errorf("Expected 0 remaining requests, got %d", info.Remaining)
	}

	t.Log("✓ Rate limiting working correctly")
}

// TestTokenRefresh demonstrates token refresh flow
func TestTokenRefresh(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key", 1) // 1 hour

	// Create initial token
	user := &auth.User{
		ID:    "user-999",
		Email: "refresh@example.com",
		Role:  string(auth.RoleDeveloper),
	}

	oldToken, oldExpiry, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	t.Logf("Original token expires at: %v", oldExpiry)

	// Refresh the token
	newToken, newExpiry, err := jwtService.RefreshToken(oldToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	t.Logf("New token expires at: %v", newExpiry)

	// Verify new token is valid
	claims, err := jwtService.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("Failed to validate refreshed token: %v", err)
	}

	// Claims should match original user
	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, user.ID)
	}

	if claims.Email != user.Email {
		t.Errorf("Email mismatch: got %s, want %s", claims.Email, user.Email)
	}

	// New expiry should be later
	if !newExpiry.After(oldExpiry) {
		t.Error("Refreshed token should have later expiry")
	}

	t.Log("✓ Token refresh completed successfully")
}

// TestRolePermissionMatrix demonstrates permission checking across different roles
func TestRolePermissionMatrix(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key", 24)

	testMatrix := []struct {
		role       auth.Role
		permission auth.Permission
		expected   bool
	}{
		// Admin should have all permissions
		{auth.RoleAdmin, auth.PermissionAdmin, true},
		{auth.RoleAdmin, auth.PermissionReadMessages, true},
		{auth.RoleAdmin, auth.PermissionWriteMessages, true},

		// Developer should have read and write, but not admin
		{auth.RoleDeveloper, auth.PermissionReadMessages, true},
		{auth.RoleDeveloper, auth.PermissionWriteMessages, true},
		{auth.RoleDeveloper, auth.PermissionAdmin, false},

		// User should have basic read/write
		{auth.RoleUser, auth.PermissionReadMessages, true},
		{auth.RoleUser, auth.PermissionWriteMessages, true},
		{auth.RoleUser, auth.PermissionAdmin, false},

		// Readonly should only have read permissions
		{auth.RoleReadOnly, auth.PermissionReadMessages, true},
		{auth.RoleReadOnly, auth.PermissionWriteMessages, false},
		{auth.RoleReadOnly, auth.PermissionAdmin, false},
	}

	for _, test := range testMatrix {
		user := &auth.User{
			ID:    "test-user",
			Email: "test@example.com",
			Role:  string(test.role),
		}

		token, _, err := jwtService.GenerateToken(user)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			t.Fatalf("Failed to validate token: %v", err)
		}

		permissions := make([]auth.Permission, len(claims.Permissions))
		for i, p := range claims.Permissions {
			permissions[i] = auth.Permission(p)
		}

		authCtx := &auth.AuthContext{
			Role:        claims.Role,
			Permissions: permissions,
		}

		hasPermission := authCtx.HasPermission(test.permission)
		if hasPermission != test.expected {
			t.Errorf("Role %s permission %s: expected %v, got %v",
				test.role, test.permission, test.expected, hasPermission)
		}
	}

	t.Log("✓ Role permission matrix validated")
}

// ExampleAuthenticationFlow demonstrates basic authentication usage
func ExampleAuthenticationFlow() {
	// Create JWT service
	jwtService := auth.NewJWTService("your-secret-key", 24)

	// Create a user
	user := &auth.User{
		ID:    "user-123",
		Email: "user@example.com",
		Role:  string(auth.RoleDeveloper),
	}

	// Generate token
	token, expiresAt, _ := jwtService.GenerateToken(user)
	println("Token:", token)
	println("Expires:", expiresAt.String())

	// Validate token
	claims, _ := jwtService.ValidateToken(token)
	println("User ID:", claims.UserID)
	println("Email:", claims.Email)
	println("Role:", claims.Role)
}

// BenchmarkJWTGeneration benchmarks token generation
func BenchmarkJWTGeneration(b *testing.B) {
	jwtService := auth.NewJWTService("test-secret-key", 24)
	user := &auth.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  string(auth.RoleDeveloper),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = jwtService.GenerateToken(user)
	}
}

// BenchmarkJWTValidation benchmarks token validation
func BenchmarkJWTValidation(b *testing.B) {
	jwtService := auth.NewJWTService("test-secret-key", 24)
	user := &auth.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  string(auth.RoleDeveloper),
	}

	token, _, _ := jwtService.GenerateToken(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = jwtService.ValidateToken(token)
	}
}

// BenchmarkRateLimiter benchmarks rate limiter performance
func BenchmarkRateLimiter(b *testing.B) {
	limiter := auth.NewRateLimiter(100)
	identifier := "test-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(identifier)
	}
}
