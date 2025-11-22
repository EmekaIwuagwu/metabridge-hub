package batching

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/rs/zerolog"
)

// Aggregator collects messages and creates batches
type Aggregator struct {
	config         *BatchConfig
	db             *database.DB
	logger         zerolog.Logger
	mu             sync.RWMutex
	pendingBatches map[string]*Batch // key: "sourceChain-destChain"
	optimizer      *Optimizer
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewAggregator creates a new batch aggregator
func NewAggregator(
	config *BatchConfig,
	db *database.DB,
	logger zerolog.Logger,
) *Aggregator {
	if config == nil {
		config = DefaultBatchConfig()
	}

	return &Aggregator{
		config:         config,
		db:             db,
		logger:         logger.With().Str("component", "batch-aggregator").Logger(),
		pendingBatches: make(map[string]*Batch),
		optimizer:      NewOptimizer(config, logger),
		stopChan:       make(chan struct{}),
	}
}

// Start begins the aggregator service
func (a *Aggregator) Start(ctx context.Context) error {
	a.logger.Info().Msg("Starting batch aggregator")

	// Start background ticker for batch processing
	a.wg.Add(1)
	go a.processingLoop(ctx)

	return nil
}

// Stop gracefully stops the aggregator
func (a *Aggregator) Stop(ctx context.Context) error {
	a.logger.Info().Msg("Stopping batch aggregator")

	close(a.stopChan)
	a.wg.Wait()

	// Submit any remaining batches
	a.submitAllReadyBatches(ctx)

	return nil
}

// AddMessage adds a message to the aggregator
func (a *Aggregator) AddMessage(ctx context.Context, msg *types.CrossChainMessage) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if batching is enabled for this chain pair
	chainPair := getChainPairKey(msg.SourceChain.Name, msg.DestinationChain.Name)
	if enabled, exists := a.config.EnabledChainPairs[chainPair]; exists && !enabled {
		return fmt.Errorf("batching not enabled for chain pair: %s", chainPair)
	}

	// Get or create batch for this chain pair
	batch, exists := a.pendingBatches[chainPair]
	if !exists {
		batch = NewBatch([]*types.CrossChainMessage{}, msg.SourceChain.Name, msg.DestinationChain.Name)
		a.pendingBatches[chainPair] = batch
		a.logger.Debug().
			Str("chain_pair", chainPair).
			Str("batch_id", batch.ID).
			Msg("Created new batch")
	}

	// Add message to batch
	if err := batch.AddMessage(msg); err != nil {
		return fmt.Errorf("failed to add message to batch: %w", err)
	}

	a.logger.Debug().
		Str("batch_id", batch.ID).
		Str("message_id", msg.ID).
		Int("batch_size", len(batch.Messages)).
		Msg("Message added to batch")

	// Check if batch is ready for submission
	if batch.IsFull(a.config.MaxBatchSize) {
		a.logger.Info().
			Str("batch_id", batch.ID).
			Int("size", len(batch.Messages)).
			Msg("Batch is full, scheduling for submission")

		if err := a.submitBatch(ctx, batch); err != nil {
			a.logger.Error().Err(err).Str("batch_id", batch.ID).Msg("Failed to submit full batch")
			return err
		}

		// Remove from pending
		delete(a.pendingBatches, chainPair)
	}

	return nil
}

// processingLoop periodically checks for ready batches
func (a *Aggregator) processingLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.checkAndSubmitReadyBatches(ctx)
		}
	}
}

// checkAndSubmitReadyBatches checks all pending batches and submits ready ones
func (a *Aggregator) checkAndSubmitReadyBatches(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for chainPair, batch := range a.pendingBatches {
		if batch.IsReady(a.config) {
			a.logger.Info().
				Str("batch_id", batch.ID).
				Str("chain_pair", chainPair).
				Int("size", len(batch.Messages)).
				Dur("age", time.Since(batch.CreatedAt)).
				Msg("Batch is ready for submission")

			if err := a.submitBatch(ctx, batch); err != nil {
				a.logger.Error().
					Err(err).
					Str("batch_id", batch.ID).
					Msg("Failed to submit ready batch")
				continue
			}

			// Remove from pending
			delete(a.pendingBatches, chainPair)
		}
	}
}

// submitBatch submits a batch for processing
func (a *Aggregator) submitBatch(ctx context.Context, batch *Batch) error {
	// Calculate gas savings before submission
	savings, err := a.optimizer.CalculateGasSavings(batch)
	if err != nil {
		a.logger.Warn().Err(err).Msg("Failed to calculate gas savings")
	} else {
		batch.GasCostSaved = savings
	}

	// Generate Merkle tree and proofs
	merkleData, err := GenerateBatchMerkleData(batch)
	if err != nil {
		return fmt.Errorf("failed to generate merkle data: %w", err)
	}

	a.logger.Info().
		Str("batch_id", batch.ID).
		Str("merkle_root", merkleData.Root).
		Int("message_count", len(batch.Messages)).
		Str("total_value", batch.TotalValue.String()).
		Str("gas_saved", batch.GasCostSaved.String()).
		Msg("Batch prepared for submission")

	// Mark as ready
	batch.Status = BatchStatusReady

	// Store batch in database
	if err := a.storeBatch(ctx, batch, merkleData); err != nil {
		return fmt.Errorf("failed to store batch: %w", err)
	}

	// TODO: Send batch to relayer for on-chain submission
	// This will be handled by the batcher service

	return nil
}

// submitAllReadyBatches submits all batches that are ready
func (a *Aggregator) submitAllReadyBatches(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for chainPair, batch := range a.pendingBatches {
		if len(batch.Messages) >= a.config.MinBatchSize {
			a.logger.Info().
				Str("batch_id", batch.ID).
				Int("size", len(batch.Messages)).
				Msg("Submitting pending batch on shutdown")

			if err := a.submitBatch(ctx, batch); err != nil {
				a.logger.Error().Err(err).Str("batch_id", batch.ID).Msg("Failed to submit batch")
				continue
			}

			delete(a.pendingBatches, chainPair)
		}
	}
}

// storeBatch persists batch data to database
func (a *Aggregator) storeBatch(ctx context.Context, batch *Batch, merkleData *BatchMerkleData) error {
	// Save batch record
	dbBatch := &database.Batch{
		ID:               batch.ID,
		Status:           string(batch.Status),
		SourceChain:      batch.SourceChain,
		DestinationChain: batch.DestChain,
		MessageCount:     len(batch.Messages),
		TotalGasSaved:    batch.GasCostSaved.String(),
		CreatedAt:        batch.CreatedAt,
	}

	if err := a.db.SaveBatch(ctx, dbBatch); err != nil {
		return fmt.Errorf("failed to save batch: %w", err)
	}

	// Save batch messages
	for _, msg := range batch.Messages {
		if err := a.db.AddMessageToBatch(ctx, batch.ID, msg.ID); err != nil {
			a.logger.Warn().
				Err(err).
				Str("batch_id", batch.ID).
				Str("message_id", msg.ID).
				Msg("Failed to add message to batch")
		}
	}

	a.logger.Info().
		Str("batch_id", batch.ID).
		Str("status", string(batch.Status)).
		Int("message_count", len(batch.Messages)).
		Msg("Batch stored in database")

	return nil
}

// GetPendingBatches returns all pending batches
func (a *Aggregator) GetPendingBatches() []*Batch {
	a.mu.RLock()
	defer a.mu.RUnlock()

	batches := make([]*Batch, 0, len(a.pendingBatches))
	for _, batch := range a.pendingBatches {
		batches = append(batches, batch)
	}

	return batches
}

// GetBatchStats returns statistics about batching
func (a *Aggregator) GetBatchStats() *BatchStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := &BatchStats{
		PendingBatchCount:   len(a.pendingBatches),
		PendingMessageCount: 0,
		TotalValueLocked:    big.NewInt(0),
	}

	for _, batch := range a.pendingBatches {
		stats.PendingMessageCount += len(batch.Messages)
		stats.TotalValueLocked.Add(stats.TotalValueLocked, batch.TotalValue)
	}

	return stats
}

// BatchStats holds batching statistics
type BatchStats struct {
	PendingBatchCount   int
	PendingMessageCount int
	TotalValueLocked    *big.Int
	SubmittedToday      int
	TotalGasSaved       *big.Int
}

// Helper functions

func getChainPairKey(source, dest string) string {
	return fmt.Sprintf("%s-%s", source, dest)
}
