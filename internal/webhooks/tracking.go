package webhooks

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/rs/zerolog"
)

// TrackingService provides message tracking functionality
type TrackingService struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewTrackingService creates a new tracking service
func NewTrackingService(db *database.DB, logger zerolog.Logger) *TrackingService {
	return &TrackingService{
		db:     db,
		logger: logger.With().Str("component", "tracking").Logger(),
	}
}

// TrackMessage retrieves detailed tracking information for a message
func (t *TrackingService) TrackMessage(ctx context.Context, messageID string) (*MessageTimeline, error) {
	// Get message
	message, err := t.getMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Get timeline events
	events, err := t.getTimelineEvents(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Calculate estimated completion
	estimatedCompletion := t.estimateCompletion(message, events)

	timeline := &MessageTimeline{
		MessageID:           messageID,
		Events:              events,
		CurrentStatus:       message.Status,
		CreatedAt:           message.CreatedAt,
		UpdatedAt:           message.UpdatedAt,
		EstimatedCompletion: estimatedCompletion,
	}

	return timeline, nil
}

// QueryMessages searches for messages based on various criteria
func (t *TrackingService) QueryMessages(ctx context.Context, query *TrackingQuery) (*TrackingResult, error) {
	// Build SQL query dynamically
	sqlQuery, args := t.buildQuery(query)

	// Get total count
	countQuery := t.buildCountQuery(query)
	var totalCount int
	err := t.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	// Execute query
	rows, err := t.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	// Scan results
	messages := []*types.CrossChainMessage{}
	for rows.Next() {
		message, err := t.scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	result := &TrackingResult{
		Messages:   messages,
		TotalCount: totalCount,
		Limit:      query.Limit,
		Offset:     query.Offset,
		HasMore:    query.Offset+len(messages) < totalCount,
	}

	return result, nil
}

// GetMessageByTxHash retrieves a message by transaction hash
func (t *TrackingService) GetMessageByTxHash(ctx context.Context, txHash string) (*types.CrossChainMessage, error) {
	query := `
		SELECT
			id, source_chain, dest_chain, sender, recipient,
			token_address, amount, nonce, data, status,
			source_tx_hash, dest_tx_hash, validator_signatures,
			created_at, updated_at, submitted_at, confirmed_at
		FROM messages
		WHERE source_tx_hash = $1 OR dest_tx_hash = $1
		LIMIT 1
	`

	row := t.db.QueryRowContext(ctx, query, txHash)
	return t.scanMessage(row)
}

// RecordEvent records a timeline event for a message
func (t *TrackingService) RecordEvent(ctx context.Context, messageID string, event *TimelineEvent) error {
	query := `
		INSERT INTO message_timeline_events (
			id, message_id, event_type, timestamp,
			description, tx_hash, block_number, chain_id, metadata
		) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8)
	`

	metadataJSON := "{}"
	if len(event.Metadata) > 0 {
		// Convert metadata to JSON
		// For simplicity, we'll store as string
		metadataJSON = fmt.Sprintf("%v", event.Metadata)
	}

	_, err := t.db.ExecContext(ctx, query,
		messageID,
		event.EventType,
		event.Timestamp,
		event.Description,
		event.TxHash,
		event.BlockNumber,
		event.ChainID,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to record event: %w", err)
	}

	t.logger.Debug().
		Str("message_id", messageID).
		Str("event_type", event.EventType).
		Msg("Recorded timeline event")

	return nil
}

// GetRecentMessages retrieves recently created messages
func (t *TrackingService) GetRecentMessages(ctx context.Context, limit int) ([]*types.CrossChainMessage, error) {
	query := `
		SELECT
			id, source_chain, dest_chain, sender, recipient,
			token_address, amount, nonce, data, status,
			source_tx_hash, dest_tx_hash, validator_signatures,
			created_at, updated_at, submitted_at, confirmed_at
		FROM messages
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := t.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent messages: %w", err)
	}
	defer rows.Close()

	messages := []*types.CrossChainMessage{}
	for rows.Next() {
		message, err := t.scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// GetMessagesByStatus retrieves messages by status
func (t *TrackingService) GetMessagesByStatus(ctx context.Context, status string, limit int) ([]*types.CrossChainMessage, error) {
	query := `
		SELECT
			id, source_chain, dest_chain, sender, recipient,
			token_address, amount, nonce, data, status,
			source_tx_hash, dest_tx_hash, validator_signatures,
			created_at, updated_at, submitted_at, confirmed_at
		FROM messages
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := t.db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages by status: %w", err)
	}
	defer rows.Close()

	messages := []*types.CrossChainMessage{}
	for rows.Next() {
		message, err := t.scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// Helper functions

func (t *TrackingService) getMessage(ctx context.Context, messageID string) (*types.CrossChainMessage, error) {
	query := `
		SELECT
			id, source_chain, dest_chain, sender, recipient,
			token_address, amount, nonce, data, status,
			source_tx_hash, dest_tx_hash, validator_signatures,
			created_at, updated_at, submitted_at, confirmed_at
		FROM messages
		WHERE id = $1
	`

	row := t.db.QueryRowContext(ctx, query, messageID)
	return t.scanMessage(row)
}

func (t *TrackingService) getTimelineEvents(ctx context.Context, messageID string) ([]TimelineEvent, error) {
	query := `
		SELECT
			event_type, timestamp, description,
			tx_hash, block_number, chain_id, metadata
		FROM message_timeline_events
		WHERE message_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := t.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeline events: %w", err)
	}
	defer rows.Close()

	events := []TimelineEvent{}
	for rows.Next() {
		var event TimelineEvent
		var txHash, chainID sql.NullString
		var blockNumber sql.NullInt64
		var metadata string

		err := rows.Scan(
			&event.EventType,
			&event.Timestamp,
			&event.Description,
			&txHash,
			&blockNumber,
			&chainID,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeline event: %w", err)
		}

		if txHash.Valid {
			event.TxHash = txHash.String
		}
		if blockNumber.Valid {
			event.BlockNumber = uint64(blockNumber.Int64)
		}
		if chainID.Valid {
			event.ChainID = chainID.String
		}

		// Parse metadata if needed
		event.Metadata = make(map[string]interface{})

		events = append(events, event)
	}

	return events, nil
}

func (t *TrackingService) estimateCompletion(message *types.CrossChainMessage, events []TimelineEvent) *time.Time {
	// If already completed, return nil
	if message.Status == "CONFIRMED" || message.Status == "FINALIZED" {
		return nil
	}

	// If failed, return nil
	if message.Status == "FAILED" {
		return nil
	}

	// Estimate based on average processing time
	// For simplicity, estimate 5-10 minutes from creation for pending messages
	if message.Status == "PENDING" {
		estimated := message.CreatedAt.Add(10 * time.Minute)
		return &estimated
	}

	// For submitted messages, estimate 2-5 minutes
	if message.Status == "SUBMITTED" && message.SubmittedAt != nil {
		estimated := message.SubmittedAt.Add(5 * time.Minute)
		return &estimated
	}

	return nil
}

func (t *TrackingService) buildQuery(query *TrackingQuery) (string, []interface{}) {
	sql := `
		SELECT
			id, source_chain, dest_chain, sender, recipient,
			token_address, amount, nonce, data, status,
			source_tx_hash, dest_tx_hash, validator_signatures,
			created_at, updated_at, submitted_at, confirmed_at
		FROM messages
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if query.MessageID != "" {
		sql += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, query.MessageID)
		argIndex++
	}

	if query.TxHash != "" {
		sql += fmt.Sprintf(" AND (source_tx_hash = $%d OR dest_tx_hash = $%d)", argIndex, argIndex)
		args = append(args, query.TxHash)
		argIndex++
	}

	if query.Sender != "" {
		sql += fmt.Sprintf(" AND sender = $%d", argIndex)
		args = append(args, query.Sender)
		argIndex++
	}

	if query.Recipient != "" {
		sql += fmt.Sprintf(" AND recipient = $%d", argIndex)
		args = append(args, query.Recipient)
		argIndex++
	}

	if query.SourceChain != "" {
		sql += fmt.Sprintf(" AND source_chain = $%d", argIndex)
		args = append(args, query.SourceChain)
		argIndex++
	}

	if query.DestChain != "" {
		sql += fmt.Sprintf(" AND dest_chain = $%d", argIndex)
		args = append(args, query.DestChain)
		argIndex++
	}

	if query.Status != "" {
		sql += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, query.Status)
		argIndex++
	}

	if query.FromDate != nil {
		sql += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, query.FromDate)
		argIndex++
	}

	if query.ToDate != nil {
		sql += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, query.ToDate)
		argIndex++
	}

	sql += " ORDER BY created_at DESC"

	// Add limit and offset
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	sql += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, query.Offset)

	return sql, args
}

func (t *TrackingService) buildCountQuery(query *TrackingQuery) string {
	sql := "SELECT COUNT(*) FROM messages WHERE 1=1"

	if query.MessageID != "" {
		sql += " AND id = $1"
	}
	if query.TxHash != "" {
		sql += " AND (source_tx_hash = $1 OR dest_tx_hash = $1)"
	}
	if query.Sender != "" {
		sql += " AND sender = $1"
	}
	if query.Recipient != "" {
		sql += " AND recipient = $1"
	}
	if query.SourceChain != "" {
		sql += " AND source_chain = $1"
	}
	if query.DestChain != "" {
		sql += " AND dest_chain = $1"
	}
	if query.Status != "" {
		sql += " AND status = $1"
	}

	return sql
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (t *TrackingService) scanMessage(row scanner) (*types.CrossChainMessage, error) {
	message := &types.CrossChainMessage{}
	var submittedAt, confirmedAt sql.NullTime
	var sourceTxHash, destTxHash sql.NullString
	var validatorSignatures sql.NullString

	err := row.Scan(
		&message.ID,
		&message.SourceChain,
		&message.DestChain,
		&message.Sender,
		&message.Recipient,
		&message.TokenAddress,
		&message.Amount,
		&message.Nonce,
		&message.Data,
		&message.Status,
		&sourceTxHash,
		&destTxHash,
		&validatorSignatures,
		&message.CreatedAt,
		&message.UpdatedAt,
		&submittedAt,
		&confirmedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan message: %w", err)
	}

	if submittedAt.Valid {
		message.SubmittedAt = &submittedAt.Time
	}
	if confirmedAt.Valid {
		message.ConfirmedAt = &confirmedAt.Time
	}
	if sourceTxHash.Valid {
		message.SourceTxHash = sourceTxHash.String
	}
	if destTxHash.Valid {
		message.DestTxHash = destTxHash.String
	}

	return message, nil
}
