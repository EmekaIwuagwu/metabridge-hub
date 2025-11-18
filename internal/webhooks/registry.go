package webhooks

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

// Registry manages webhook registrations
type Registry struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRegistry creates a new webhook registry
func NewRegistry(db *database.DB, logger zerolog.Logger) *Registry {
	return &Registry{
		db:     db,
		logger: logger.With().Str("component", "webhook-registry").Logger(),
	}
}

// Register creates a new webhook registration
func (r *Registry) Register(ctx context.Context, webhook *Webhook) error {
	// Generate ID and secret
	webhook.ID = uuid.New().String()
	webhook.Secret = generateSecret()
	webhook.Status = WebhookStatusActive
	webhook.CreatedAt = time.Now().UTC()
	webhook.UpdatedAt = webhook.CreatedAt
	webhook.FailCount = 0
	webhook.SuccessCount = 0

	// Validate webhook
	if err := r.validateWebhook(webhook); err != nil {
		return fmt.Errorf("invalid webhook: %w", err)
	}

	// Insert into database
	query := `
		INSERT INTO webhooks (
			id, url, secret, events, status, description,
			created_by, created_at, updated_at, fail_count, success_count,
			source_chains, dest_chains, min_amount, max_amount
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	events := make([]string, len(webhook.Events))
	for i, e := range webhook.Events {
		events[i] = string(e)
	}

	_, err := r.db.ExecContext(ctx, query,
		webhook.ID,
		webhook.URL,
		webhook.Secret,
		pq.Array(events),
		webhook.Status,
		webhook.Description,
		webhook.CreatedBy,
		webhook.CreatedAt,
		webhook.UpdatedAt,
		webhook.FailCount,
		webhook.SuccessCount,
		pq.Array(webhook.SourceChains),
		pq.Array(webhook.DestChains),
		webhook.MinAmount,
		webhook.MaxAmount,
	)

	if err != nil {
		return fmt.Errorf("failed to register webhook: %w", err)
	}

	r.logger.Info().
		Str("webhook_id", webhook.ID).
		Str("url", webhook.URL).
		Int("events", len(webhook.Events)).
		Msg("Webhook registered successfully")

	return nil
}

// Get retrieves a webhook by ID
func (r *Registry) Get(ctx context.Context, webhookID string) (*Webhook, error) {
	query := `
		SELECT
			id, url, secret, events, status, description,
			created_by, created_at, updated_at, last_used_at,
			fail_count, success_count,
			source_chains, dest_chains, min_amount, max_amount
		FROM webhooks
		WHERE id = $1
	`

	webhook := &Webhook{}
	var events pq.StringArray
	var sourceChains, destChains pq.StringArray

	err := r.db.QueryRowContext(ctx, query, webhookID).Scan(
		&webhook.ID,
		&webhook.URL,
		&webhook.Secret,
		&events,
		&webhook.Status,
		&webhook.Description,
		&webhook.CreatedBy,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
		&webhook.LastUsedAt,
		&webhook.FailCount,
		&webhook.SuccessCount,
		&sourceChains,
		&destChains,
		&webhook.MinAmount,
		&webhook.MaxAmount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("webhook not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	// Convert string arrays to typed arrays
	webhook.Events = make([]EventType, len(events))
	for i, e := range events {
		webhook.Events[i] = EventType(e)
	}
	webhook.SourceChains = sourceChains
	webhook.DestChains = destChains

	return webhook, nil
}

// List retrieves all webhooks for a user
func (r *Registry) List(ctx context.Context, createdBy string) ([]*Webhook, error) {
	query := `
		SELECT
			id, url, secret, events, status, description,
			created_by, created_at, updated_at, last_used_at,
			fail_count, success_count,
			source_chains, dest_chains, min_amount, max_amount
		FROM webhooks
		WHERE created_by = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer rows.Close()

	webhooks := []*Webhook{}
	for rows.Next() {
		webhook := &Webhook{}
		var events pq.StringArray
		var sourceChains, destChains pq.StringArray

		err := rows.Scan(
			&webhook.ID,
			&webhook.URL,
			&webhook.Secret,
			&events,
			&webhook.Status,
			&webhook.Description,
			&webhook.CreatedBy,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
			&webhook.LastUsedAt,
			&webhook.FailCount,
			&webhook.SuccessCount,
			&sourceChains,
			&destChains,
			&webhook.MinAmount,
			&webhook.MaxAmount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}

		webhook.Events = make([]EventType, len(events))
		for i, e := range events {
			webhook.Events[i] = EventType(e)
		}
		webhook.SourceChains = sourceChains
		webhook.DestChains = destChains

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// Update updates an existing webhook
func (r *Registry) Update(ctx context.Context, webhook *Webhook) error {
	webhook.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE webhooks
		SET url = $2, events = $3, status = $4, description = $5,
		    updated_at = $6, source_chains = $7, dest_chains = $8,
		    min_amount = $9, max_amount = $10
		WHERE id = $1
	`

	events := make([]string, len(webhook.Events))
	for i, e := range webhook.Events {
		events[i] = string(e)
	}

	result, err := r.db.ExecContext(ctx, query,
		webhook.ID,
		webhook.URL,
		pq.Array(events),
		webhook.Status,
		webhook.Description,
		webhook.UpdatedAt,
		pq.Array(webhook.SourceChains),
		pq.Array(webhook.DestChains),
		webhook.MinAmount,
		webhook.MaxAmount,
	)

	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("webhook not found")
	}

	r.logger.Info().
		Str("webhook_id", webhook.ID).
		Msg("Webhook updated successfully")

	return nil
}

// Delete removes a webhook
func (r *Registry) Delete(ctx context.Context, webhookID string) error {
	query := `DELETE FROM webhooks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, webhookID)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("webhook not found")
	}

	r.logger.Info().
		Str("webhook_id", webhookID).
		Msg("Webhook deleted successfully")

	return nil
}

// UpdateStatus updates the status of a webhook
func (r *Registry) UpdateStatus(ctx context.Context, webhookID string, status WebhookStatus) error {
	query := `
		UPDATE webhooks
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, webhookID, status, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update webhook status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("webhook not found")
	}

	return nil
}

// IncrementSuccessCount increments the success count and updates last used time
func (r *Registry) IncrementSuccessCount(ctx context.Context, webhookID string) error {
	query := `
		UPDATE webhooks
		SET success_count = success_count + 1,
		    last_used_at = $2,
		    fail_count = 0,
		    status = CASE WHEN status = 'FAILED' THEN 'ACTIVE' ELSE status END
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, webhookID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to increment success count: %w", err)
	}

	return nil
}

// IncrementFailCount increments the fail count
func (r *Registry) IncrementFailCount(ctx context.Context, webhookID string) error {
	query := `
		UPDATE webhooks
		SET fail_count = fail_count + 1,
		    status = CASE
		        WHEN fail_count >= 10 THEN 'FAILED'
		        ELSE status
		    END
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, webhookID)
	if err != nil {
		return fmt.Errorf("failed to increment fail count: %w", err)
	}

	return nil
}

// GetActiveWebhooksForEvent retrieves all active webhooks subscribed to an event
func (r *Registry) GetActiveWebhooksForEvent(ctx context.Context, eventType EventType) ([]*Webhook, error) {
	query := `
		SELECT
			id, url, secret, events, status, description,
			created_by, created_at, updated_at, last_used_at,
			fail_count, success_count,
			source_chains, dest_chains, min_amount, max_amount
		FROM webhooks
		WHERE status = 'ACTIVE'
		AND $1 = ANY(events)
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, string(eventType))
	if err != nil {
		return nil, fmt.Errorf("failed to get active webhooks: %w", err)
	}
	defer rows.Close()

	webhooks := []*Webhook{}
	for rows.Next() {
		webhook := &Webhook{}
		var events pq.StringArray
		var sourceChains, destChains pq.StringArray

		err := rows.Scan(
			&webhook.ID,
			&webhook.URL,
			&webhook.Secret,
			&events,
			&webhook.Status,
			&webhook.Description,
			&webhook.CreatedBy,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
			&webhook.LastUsedAt,
			&webhook.FailCount,
			&webhook.SuccessCount,
			&sourceChains,
			&destChains,
			&webhook.MinAmount,
			&webhook.MaxAmount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}

		webhook.Events = make([]EventType, len(events))
		for i, e := range events {
			webhook.Events[i] = EventType(e)
		}
		webhook.SourceChains = sourceChains
		webhook.DestChains = destChains

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// validateWebhook validates a webhook configuration
func (r *Registry) validateWebhook(webhook *Webhook) error {
	if webhook.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if len(webhook.Events) == 0 {
		return fmt.Errorf("at least one event type is required")
	}

	if webhook.CreatedBy == "" {
		return fmt.Errorf("created_by is required")
	}

	// Validate event types
	validEvents := map[EventType]bool{
		EventMessageCreated:   true,
		EventMessagePending:   true,
		EventMessageSubmitted: true,
		EventMessageConfirmed: true,
		EventMessageFinalized: true,
		EventMessageFailed:    true,
		EventBatchCreated:     true,
		EventBatchSubmitted:   true,
		EventBatchConfirmed:   true,
		EventBatchFailed:      true,
	}

	for _, event := range webhook.Events {
		if !validEvents[event] {
			return fmt.Errorf("invalid event type: %s", event)
		}
	}

	return nil
}

// generateSecret generates a random secret for webhook signing
func generateSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID if random fails
		return uuid.New().String()
	}
	return hex.EncodeToString(bytes)
}
