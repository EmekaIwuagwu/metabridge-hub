package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
)

// SaveMessage saves a cross-chain message to the database
func (db *DB) SaveMessage(ctx context.Context, msg *types.CrossChainMessage) error {
	query := `
		INSERT INTO messages (
			id, type, source_chain_id, source_chain_name, destination_chain_id,
			destination_chain_name, sender, recipient, payload, status, nonce, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = CURRENT_TIMESTAMP
	`

	payloadJSON, err := json.Marshal(msg.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = db.ExecContext(ctx, query,
		msg.ID,
		msg.Type,
		msg.SourceChain.ChainID,
		msg.SourceChain.Name,
		msg.DestinationChain.ChainID,
		msg.DestinationChain.Name,
		msg.Sender.Raw,
		msg.Recipient.Raw,
		payloadJSON,
		msg.Status,
		msg.Nonce,
		msg.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	db.logger.Debug().
		Str("message_id", msg.ID).
		Str("status", string(msg.Status)).
		Msg("Message saved to database")

	return nil
}

// GetMessage retrieves a message by ID
func (db *DB) GetMessage(ctx context.Context, messageID string) (*types.CrossChainMessage, error) {
	query := `
		SELECT
			id, type, source_chain_id, source_chain_name, destination_chain_id,
			destination_chain_name, sender, recipient, payload, status, nonce,
			timestamp
		FROM messages
		WHERE id = $1
	`

	var msg types.CrossChainMessage
	var payloadJSON []byte

	err := db.QueryRowContext(ctx, query, messageID).Scan(
		&msg.ID,
		&msg.Type,
		&msg.SourceChain.ChainID,
		&msg.SourceChain.Name,
		&msg.DestinationChain.ChainID,
		&msg.DestinationChain.Name,
		&msg.Sender.Raw,
		&msg.Recipient.Raw,
		&payloadJSON,
		&msg.Status,
		&msg.Nonce,
		&msg.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found: %s", messageID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	msg.Payload = payloadJSON

	return &msg, nil
}

// GetMessageStatus retrieves the status of a message
func (db *DB) GetMessageStatus(ctx context.Context, messageID string) (types.MessageStatus, error) {
	query := `SELECT status FROM messages WHERE id = $1`

	var status types.MessageStatus
	err := db.QueryRowContext(ctx, query, messageID).Scan(&status)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("message not found: %s", messageID)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get message status: %w", err)
	}

	return status, nil
}

// UpdateMessageStatus updates the status of a message
func (db *DB) UpdateMessageStatus(ctx context.Context, messageID string, status types.MessageStatus, txHash string) error {
	query := `
		UPDATE messages
		SET status = $1, destination_tx_hash = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	result, err := db.ExecContext(ctx, query, status, txHash, messageID)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("message not found: %s", messageID)
	}

	db.logger.Debug().
		Str("message_id", messageID).
		Str("status", string(status)).
		Str("tx_hash", txHash).
		Msg("Message status updated")

	return nil
}

// GetPendingMessages retrieves pending messages
func (db *DB) GetPendingMessages(ctx context.Context, limit int) ([]types.CrossChainMessage, error) {
	query := `
		SELECT
			id, type, source_chain_id, source_chain_name, destination_chain_id,
			destination_chain_name, sender, recipient, payload, status, nonce,
			timestamp
		FROM messages
		WHERE status = $1
		ORDER BY timestamp ASC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, query, types.MessageStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending messages: %w", err)
	}
	defer rows.Close()

	var messages []types.CrossChainMessage

	for rows.Next() {
		var msg types.CrossChainMessage
		var payloadJSON []byte

		err := rows.Scan(
			&msg.ID,
			&msg.Type,
			&msg.SourceChain.ChainID,
			&msg.SourceChain.Name,
			&msg.DestinationChain.ChainID,
			&msg.DestinationChain.Name,
			&msg.Sender.Raw,
			&msg.Recipient.Raw,
			&payloadJSON,
			&msg.Status,
			&msg.Nonce,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.Payload = payloadJSON
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetPendingMessagesCount returns the count of pending messages
func (db *DB) GetPendingMessagesCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM messages WHERE status = $1`

	var count int64
	err := db.QueryRowContext(ctx, query, types.MessageStatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending messages count: %w", err)
	}

	return count, nil
}

// GetProcessedMessagesCount returns the count of processed messages
func (db *DB) GetProcessedMessagesCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM messages WHERE status = $1`

	var count int64
	err := db.QueryRowContext(ctx, query, types.MessageStatusCompleted).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get processed messages count: %w", err)
	}

	return count, nil
}

// GetFailedMessagesCount returns the count of failed messages
func (db *DB) GetFailedMessagesCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM messages WHERE status = $1`

	var count int64
	err := db.QueryRowContext(ctx, query, types.MessageStatusFailed).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get failed messages count: %w", err)
	}

	return count, nil
}

// GetMessagesByStatus retrieves messages by status
func (db *DB) GetMessagesByStatus(ctx context.Context, status types.MessageStatus, limit int, offset int) ([]types.CrossChainMessage, error) {
	query := `
		SELECT
			id, type, source_chain_id, source_chain_name, destination_chain_id,
			destination_chain_name, sender, recipient, payload, status, nonce,
			timestamp
		FROM messages
		WHERE status = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []types.CrossChainMessage

	for rows.Next() {
		var msg types.CrossChainMessage
		var payloadJSON []byte

		err := rows.Scan(
			&msg.ID,
			&msg.Type,
			&msg.SourceChain.ChainID,
			&msg.SourceChain.Name,
			&msg.DestinationChain.ChainID,
			&msg.DestinationChain.Name,
			&msg.Sender.Raw,
			&msg.Recipient.Raw,
			&payloadJSON,
			&msg.Status,
			&msg.Nonce,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.Payload = payloadJSON
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetMessagesByChains retrieves messages between specific chains
func (db *DB) GetMessagesByChains(ctx context.Context, sourceChain, destChain string, limit int) ([]types.CrossChainMessage, error) {
	query := `
		SELECT
			id, type, source_chain_id, source_chain_name, destination_chain_id,
			destination_chain_name, sender, recipient, payload, status, nonce,
			timestamp
		FROM messages
		WHERE source_chain_name = $1 AND destination_chain_name = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := db.QueryContext(ctx, query, sourceChain, destChain, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []types.CrossChainMessage

	for rows.Next() {
		var msg types.CrossChainMessage
		var payloadJSON []byte

		err := rows.Scan(
			&msg.ID,
			&msg.Type,
			&msg.SourceChain.ChainID,
			&msg.SourceChain.Name,
			&msg.DestinationChain.ChainID,
			&msg.DestinationChain.Name,
			&msg.Sender.Raw,
			&msg.Recipient.Raw,
			&payloadJSON,
			&msg.Status,
			&msg.Nonce,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.Payload = payloadJSON
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// SaveValidatorSignature saves a validator signature
func (db *DB) SaveValidatorSignature(ctx context.Context, messageID string, sig *types.ValidatorSignature) error {
	query := `
		INSERT INTO validator_signatures (
			message_id, validator_address, signature, signed_at
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, validator_address) DO NOTHING
	`

	_, err := db.ExecContext(ctx, query,
		messageID,
		sig.ValidatorAddress,
		sig.Signature,
		sig.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to save validator signature: %w", err)
	}

	return nil
}

// GetValidatorSignatures retrieves all validator signatures for a message
func (db *DB) GetValidatorSignatures(ctx context.Context, messageID string) ([]types.ValidatorSignature, error) {
	query := `
		SELECT validator_address, signature, signed_at
		FROM validator_signatures
		WHERE message_id = $1
		ORDER BY signed_at ASC
	`

	rows, err := db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query signatures: %w", err)
	}
	defer rows.Close()

	var signatures []types.ValidatorSignature

	for rows.Next() {
		var sig types.ValidatorSignature
		err := rows.Scan(&sig.ValidatorAddress, &sig.Signature, &sig.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signature: %w", err)
		}
		signatures = append(signatures, sig)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating signatures: %w", err)
	}

	return signatures, nil
}
