package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTService handles JWT token operations
type JWTService struct {
	secret []byte
	expiry time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secret string, expiryHours int) *JWTService {
	return &JWTService{
		secret: []byte(secret),
		expiry: time.Duration(expiryHours) * time.Hour,
	}
}

// GenerateToken generates a JWT token for a user
func (j *JWTService) GenerateToken(user *User) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(j.expiry)

	// Get permissions for role
	permissions := RolePermissions[Role(user.Role)]
	permStrings := make([]string, len(permissions))
	for i, p := range permissions {
		permStrings[i] = string(p)
	}

	claims := JWTClaims{
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role,
		Permissions: permStrings,
		IssuedAt:    now.Unix(),
		ExpiresAt:   expiresAt.Unix(),
	}

	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Base64 encode
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signature
	message := headerEncoded + "." + claimsEncoded
	signature := j.sign(message)
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)

	// Combine
	token := message + "." + signatureEncoded

	return token, expiresAt, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTService) ValidateToken(token string) (*JWTClaims, error) {
	// Split token
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerEncoded := parts[0]
	claimsEncoded := parts[1]
	signatureEncoded := parts[2]

	// Verify signature
	message := headerEncoded + "." + claimsEncoded
	expectedSignature := j.sign(message)
	expectedSignatureEncoded := base64.RawURLEncoding.EncodeToString(expectedSignature)

	if signatureEncoded != expectedSignatureEncoded {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsEncoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// Check expiration
	if time.Now().UTC().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// sign creates an HMAC signature
func (j *JWTService) sign(message string) []byte {
	h := hmac.New(sha256.New, j.secret)
	h.Write([]byte(message))
	return h.Sum(nil)
}

// RefreshToken generates a new token with extended expiry
func (j *JWTService) RefreshToken(token string) (string, time.Time, error) {
	// Validate existing token
	claims, err := j.ValidateToken(token)
	if err != nil {
		// Allow refresh even if expired (within grace period)
		if !strings.Contains(err.Error(), "token expired") {
			return "", time.Time{}, err
		}
	}

	// Create new token with same claims but new expiry
	now := time.Now().UTC()
	expiresAt := now.Add(j.expiry)

	claims.IssuedAt = now.Unix()
	claims.ExpiresAt = expiresAt.Unix()

	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Base64 encode
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signature
	message := headerEncoded + "." + claimsEncoded
	signature := j.sign(message)
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)

	// Combine
	newToken := message + "." + signatureEncoded

	return newToken, expiresAt, nil
}
