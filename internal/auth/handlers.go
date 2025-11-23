package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/database"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// Handler provides authentication HTTP handlers
type Handler struct {
	db         *database.DB
	jwtService *JWTService
	config     *AuthConfig
	logger     zerolog.Logger
}

// NewHandler creates a new auth handler
func NewHandler(db *database.DB, config *AuthConfig, logger zerolog.Logger) *Handler {
	if config == nil {
		config = DefaultAuthConfig()
	}

	return &Handler{
		db:         db,
		jwtService: NewJWTService(config.JWTSecret, config.JWTExpirationHours),
		config:     config,
		logger:     logger.With().Str("component", "auth-handler").Logger(),
	}
}

// HandleLogin handles user login
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Get user from database
	user, passwordHash, err := h.getUserByEmail(r.Context(), req.Email)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "invalid credentials", nil)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		h.respondError(w, http.StatusUnauthorized, "invalid credentials", nil)
		return
	}

	// Check if user is active
	if !user.Active {
		h.respondError(w, http.StatusUnauthorized, "user account is disabled", nil)
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.jwtService.GenerateToken(user)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to generate token", err)
		return
	}

	response := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HandleRefreshToken refreshes a JWT token
func (h *Handler) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.respondError(w, http.StatusUnauthorized, "missing authorization header", nil)
		return
	}

	token := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	} else {
		h.respondError(w, http.StatusUnauthorized, "invalid authorization format", nil)
		return
	}

	newToken, expiresAt, err := h.jwtService.RefreshToken(token)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "invalid or expired token", err)
		return
	}

	response := map[string]interface{}{
		"token":      newToken,
		"expires_at": expiresAt,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HandleCreateAPIKey creates a new API key
func (h *Handler) HandleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Generate API key
	apiKey := GenerateAPIKey()
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		expires := time.Now().UTC().Add(time.Duration(req.ExpiresIn) * 24 * time.Hour)
		expiresAt = &expires
	}

	// Default permissions from user role if not specified
	permissions := req.Permissions
	if len(permissions) == 0 {
		rolePerms := RolePermissions[Role(authCtx.Role)]
		permissions = make([]string, len(rolePerms))
		for i, p := range rolePerms {
			permissions[i] = string(p)
		}
	}

	// Insert into database
	apiKeyID := uuid.New().String()
	query := `
		INSERT INTO api_keys (
			id, user_id, name, key_hash, permissions,
			active, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now().UTC()
	_, err := h.db.ExecContext(r.Context(), query,
		apiKeyID,
		authCtx.UserID,
		req.Name,
		keyHash,
		fmt.Sprintf("{%s}", joinStrings(permissions, ",")),
		true,
		expiresAt,
		now,
		now,
	)

	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create API key", err)
		return
	}

	response := CreateAPIKeyResponse{
		APIKey: &APIKey{
			ID:          apiKeyID,
			UserID:      authCtx.UserID,
			Name:        req.Name,
			Permissions: permissions,
			Active:      true,
			ExpiresAt:   expiresAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Key: apiKey, // Only returned on creation
	}

	h.logger.Info().
		Str("api_key_id", apiKeyID).
		Str("user_id", authCtx.UserID).
		Msg("API key created")

	h.respondJSON(w, http.StatusCreated, response)
}

// HandleListAPIKeys lists user's API keys
func (h *Handler) HandleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	query := `
		SELECT
			id, user_id, name, permissions, active,
			expires_at, last_used_at, created_at, updated_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.QueryContext(r.Context(), query, authCtx.UserID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list API keys", err)
		return
	}
	defer rows.Close()

	apiKeys := []*APIKey{}
	for rows.Next() {
		var apiKey APIKey
		var permissionsStr string

		err := rows.Scan(
			&apiKey.ID,
			&apiKey.UserID,
			&apiKey.Name,
			&permissionsStr,
			&apiKey.Active,
			&apiKey.ExpiresAt,
			&apiKey.LastUsedAt,
			&apiKey.CreatedAt,
			&apiKey.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Parse permissions (simple split for now)
		if permissionsStr != "" {
			apiKey.Permissions = []string{permissionsStr}
		}

		apiKeys = append(apiKeys, &apiKey)
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_keys": apiKeys,
		"count":    len(apiKeys),
	})
}

// HandleRevokeAPIKey revokes an API key
func (h *Handler) HandleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	vars := mux.Vars(r)
	apiKeyID := vars["id"]

	// Deactivate API key
	query := `
		UPDATE api_keys
		SET active = false, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`

	result, err := h.db.ExecContext(r.Context(), query, apiKeyID, authCtx.UserID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to revoke API key", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		h.respondError(w, http.StatusNotFound, "API key not found", nil)
		return
	}

	h.logger.Info().
		Str("api_key_id", apiKeyID).
		Str("user_id", authCtx.UserID).
		Msg("API key revoked")

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "API key revoked successfully",
	})
}

// HandleGetMe returns current user info
func (h *Handler) HandleGetMe(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	user, _, err := h.getUserByID(r.Context(), authCtx.UserID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "user not found", err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// Private methods

func (h *Handler) getUserByEmail(ctx context.Context, email string) (*User, string, error) {
	query := `
		SELECT id, email, name, role, password_hash, active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	var passwordHash string

	err := h.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Role,
		&passwordHash,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, "", err
	}

	return &user, passwordHash, nil
}

func (h *Handler) getUserByID(ctx context.Context, userID string) (*User, string, error) {
	query := `
		SELECT id, email, name, role, password_hash, active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	var passwordHash string

	err := h.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Role,
		&passwordHash,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, "", err
	}

	return &user, passwordHash, nil
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		h.logger.Error().Err(err).Msg(message)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
