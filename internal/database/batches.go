package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Batch represents a batch of messages
type Batch struct {
	ID               string    `json:"id"`
	Status           string    `json:"status"` // pending, confirmed, failed
	SourceChain      string    `json:"source_chain"`
	DestinationChain string    `json:"dest_chain"`
	MessageCount     int       `json:"message_count"`
	TotalGasSaved    string    `json:"total_gas_saved"`
	TxHash           string    `json:"tx_hash,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	ConfirmedAt      *time.Time `json:"confirmed_at,omitempty"`
}

// BatchMessage represents a message in a batch
type BatchMessage struct {
	BatchID   string    `json:"batch_id"`
	MessageID string    `json:"message_id"`
	AddedAt   time.Time `json:"added_at"`
}

// BatchStats represents daily batch statistics
type BatchStats struct {
	Date             time.Time `json:"date"`
	BatchesConfirmed int       `json:"batches_confirmed"`
	MessagesBatched  int       `json:"messages_batched"`
	TotalGasSaved    string    `json:"total_gas_saved"`
}

// SaveBatch saves a batch to the database
func (db *DB) SaveBatch(ctx context.Context, batch *Batch) error {
	query := `
		INSERT INTO batches (
			id, status, source_chain, destination_chain, message_count,
			total_gas_saved, tx_hash, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			message_count = EXCLUDED.message_count,
			total_gas_saved = EXCLUDED.total_gas_saved,
			tx_hash = EXCLUDED.tx_hash,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.ExecContext(ctx, query,
		batch.ID,
		batch.Status,
		batch.SourceChain,
		batch.DestinationChain,
		batch.MessageCount,
		batch.TotalGasSaved,
		batch.TxHash,
		batch.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save batch: %w", err)
	}

	db.logger.Debug().
		Str("batch_id", batch.ID).
		Str("status", batch.Status).
		Int("message_count", batch.MessageCount).
		Msg("Batch saved to database")

	return nil
}

// GetBatch retrieves a batch by ID
func (db *DB) GetBatch(ctx context.Context, batchID string) (*Batch, error) {
	query := `
		SELECT
			id, status, source_chain, destination_chain, message_count,
			total_gas_saved, tx_hash, created_at, confirmed_at
		FROM batches
		WHERE id = $1
	`

	var batch Batch
	var confirmedAt sql.NullTime

	err := db.QueryRowContext(ctx, query, batchID).Scan(
		&batch.ID,
		&batch.Status,
		&batch.SourceChain,
		&batch.DestinationChain,
		&batch.MessageCount,
		&batch.TotalGasSaved,
		&batch.TxHash,
		&batch.CreatedAt,
		&confirmedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("batch not found: %s", batchID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	if confirmedAt.Valid {
		batch.ConfirmedAt = &confirmedAt.Time
	}

	return &batch, nil
}

// GetBatchesByStatus retrieves batches by status
func (db *DB) GetBatchesByStatus(ctx context.Context, status string, limit, offset int) ([]Batch, error) {
	query := `
		SELECT
			id, status, source_chain, destination_chain, message_count,
			total_gas_saved, tx_hash, created_at, confirmed_at
		FROM batches
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query batches: %w", err)
	}
	defer rows.Close()

	var batches []Batch

	for rows.Next() {
		var batch Batch
		var confirmedAt sql.NullTime

		err := rows.Scan(
			&batch.ID,
			&batch.Status,
			&batch.SourceChain,
			&batch.DestinationChain,
			&batch.MessageCount,
			&batch.TotalGasSaved,
			&batch.TxHash,
			&batch.CreatedAt,
			&confirmedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan batch: %w", err)
		}

		if confirmedAt.Valid {
			batch.ConfirmedAt = &confirmedAt.Time
		}

		batches = append(batches, batch)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batches: %w", err)
	}

	return batches, nil
}

// GetAllBatches retrieves all batches
func (db *DB) GetAllBatches(ctx context.Context, limit, offset int) ([]Batch, error) {
	query := `
		SELECT
			id, status, source_chain, destination_chain, message_count,
			total_gas_saved, tx_hash, created_at, confirmed_at
		FROM batches
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query batches: %w", err)
	}
	defer rows.Close()

	var batches []Batch

	for rows.Next() {
		var batch Batch
		var confirmedAt sql.NullTime

		err := rows.Scan(
			&batch.ID,
			&batch.Status,
			&batch.SourceChain,
			&batch.DestinationChain,
			&batch.MessageCount,
			&batch.TotalGasSaved,
			&batch.TxHash,
			&batch.CreatedAt,
			&confirmedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan batch: %w", err)
		}

		if confirmedAt.Valid {
			batch.ConfirmedAt = &confirmedAt.Time
		}

		batches = append(batches, batch)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batches: %w", err)
	}

	return batches, nil
}

// GetBatchesCount returns the total count of batches
func (db *DB) GetBatchesCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM batches`

	var count int64
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get batches count: %w", err)
	}

	return count, nil
}

// GetBatchesToday returns the count of batches created today
func (db *DB) GetBatchesToday(ctx context.Context) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM batches
		WHERE DATE(created_at) = CURRENT_DATE
	`

	var count int64
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get batches today count: %w", err)
	}

	return count, nil
}

// GetTotalMessagesBatched returns the total count of messages in all batches
func (db *DB) GetTotalMessagesBatched(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(SUM(message_count), 0) FROM batches`

	var count int64
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total messages batched: %w", err)
	}

	return count, nil
}

// GetAverageBatchSize returns the average number of messages per batch
func (db *DB) GetAverageBatchSize(ctx context.Context) (float64, error) {
	query := `SELECT COALESCE(AVG(message_count), 0) FROM batches WHERE status = 'confirmed'`

	var avg float64
	err := db.QueryRowContext(ctx, query).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("failed to get average batch size: %w", err)
	}

	return avg, nil
}

// UpdateBatchStatus updates the status of a batch
func (db *DB) UpdateBatchStatus(ctx context.Context, batchID, status, txHash string) error {
	query := `
		UPDATE batches
		SET status = $1, tx_hash = $2,
			confirmed_at = CASE WHEN $1 = 'confirmed' THEN CURRENT_TIMESTAMP ELSE confirmed_at END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	result, err := db.ExecContext(ctx, query, status, txHash, batchID)
	if err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("batch not found: %s", batchID)
	}

	db.logger.Debug().
		Str("batch_id", batchID).
		Str("status", status).
		Str("tx_hash", txHash).
		Msg("Batch status updated")

	return nil
}

// AddMessageToBatch adds a message to a batch
func (db *DB) AddMessageToBatch(ctx context.Context, batchID, messageID string) error {
	query := `
		INSERT INTO batch_messages (batch_id, message_id, added_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (batch_id, message_id) DO NOTHING
	`

	_, err := db.ExecContext(ctx, query, batchID, messageID)
	if err != nil {
		return fmt.Errorf("failed to add message to batch: %w", err)
	}

	db.logger.Debug().
		Str("batch_id", batchID).
		Str("message_id", messageID).
		Msg("Message added to batch")

	return nil
}

// GetBatchMessages retrieves all messages in a batch
func (db *DB) GetBatchMessages(ctx context.Context, batchID string) ([]BatchMessage, error) {
	query := `
		SELECT batch_id, message_id, added_at
		FROM batch_messages
		WHERE batch_id = $1
		ORDER BY added_at ASC
	`

	rows, err := db.QueryContext(ctx, query, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch messages: %w", err)
	}
	defer rows.Close()

	var messages []BatchMessage

	for rows.Next() {
		var msg BatchMessage
		err := rows.Scan(&msg.BatchID, &msg.MessageID, &msg.AddedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan batch message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch messages: %w", err)
	}

	return messages, nil
}
