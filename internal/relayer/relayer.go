package relayer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/config"
	"github.com/EmekaIwuagwu/articium-hub/internal/crypto"
	"github.com/EmekaIwuagwu/articium-hub/internal/database"
	"github.com/EmekaIwuagwu/articium-hub/internal/monitoring"
	"github.com/EmekaIwuagwu/articium-hub/internal/queue"
	"github.com/EmekaIwuagwu/articium-hub/internal/security"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// Relayer manages the message relay workers
type Relayer struct {
	config    *config.Config
	db        *database.DB
	queue     queue.Queue
	processor *Processor
	logger    zerolog.Logger
	workers   int
	wg        sync.WaitGroup
	stopChan  chan struct{}
	clients   map[string]types.UniversalClient
}

// NewRelayer creates a new relayer
func NewRelayer(
	cfg *config.Config,
	db *database.DB,
	q queue.Queue,
	clients map[string]types.UniversalClient,
	signers map[string]crypto.UniversalSigner,
	logger zerolog.Logger,
) (*Relayer, error) {
	// Create security validator
	validator := security.NewValidator(&cfg.Security, cfg.Environment, logger)

	// Create processor
	processor := NewProcessor(clients, signers, db, cfg, validator, logger)

	return &Relayer{
		config:    cfg,
		db:        db,
		queue:     q,
		processor: processor,
		logger:    logger.With().Str("component", "relayer").Logger(),
		workers:   cfg.Relayer.Workers,
		stopChan:  make(chan struct{}),
		clients:   clients,
	}, nil
}

// Start starts the relayer workers
func (r *Relayer) Start(ctx context.Context) error {
	r.logger.Info().
		Int("workers", r.workers).
		Msg("Starting relayer workers")

	// Start workers
	for i := 0; i < r.workers; i++ {
		r.wg.Add(1)
		go r.worker(ctx, i)
	}

	// Start health check goroutine
	r.wg.Add(1)
	go r.healthCheck(ctx)

	// Start metrics collection
	r.wg.Add(1)
	go r.collectMetrics(ctx)

	r.logger.Info().Msg("All relayer workers started")
	return nil
}

// Stop stops the relayer
func (r *Relayer) Stop() error {
	r.logger.Info().Msg("Stopping relayer")
	close(r.stopChan)
	r.wg.Wait()
	r.logger.Info().Msg("Relayer stopped")
	return nil
}

// worker is a worker goroutine that processes messages from the queue
func (r *Relayer) worker(ctx context.Context, id int) {
	defer r.wg.Done()

	logger := r.logger.With().Int("worker_id", id).Logger()
	logger.Info().Msg("Worker started")

	// Subscribe to queue
	err := r.queue.Subscribe(ctx, func(ctx context.Context, msg *types.CrossChainMessage) error {
		return r.handleMessage(ctx, msg, logger)
	})

	if err != nil && err != context.Canceled {
		logger.Error().Err(err).Msg("Queue subscription error")
	}

	logger.Info().Msg("Worker stopped")
}

// handleMessage handles a single message
func (r *Relayer) handleMessage(ctx context.Context, msg *types.CrossChainMessage, logger zerolog.Logger) error {
	logger.Info().
		Str("message_id", msg.ID).
		Str("source", msg.SourceChain.Name).
		Str("destination", msg.DestinationChain.Name).
		Msg("Handling message")

	// Update metrics
	monitoring.MessagesTotal.WithLabelValues(
		msg.SourceChain.Name,
		msg.DestinationChain.Name,
		string(msg.Type),
		"received",
	).Inc()

	// Process message
	startTime := time.Now()
	err := r.processor.ProcessMessage(ctx, msg)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Dur("duration", duration).
			Msg("Failed to process message")

		monitoring.MessagesTotal.WithLabelValues(
			msg.SourceChain.Name,
			msg.DestinationChain.Name,
			string(msg.Type),
			"failed",
		).Inc()

		// Update message status to failed
		if dbErr := r.db.UpdateMessageStatus(ctx, msg.ID, types.MessageStatusFailed, ""); dbErr != nil {
			logger.Error().
				Err(dbErr).
				Str("message_id", msg.ID).
				Msg("Failed to update message status")
		}

		return err
	}

	logger.Info().
		Str("message_id", msg.ID).
		Dur("duration", duration).
		Msg("Message processed successfully")

	monitoring.MessagesTotal.WithLabelValues(
		msg.SourceChain.Name,
		msg.DestinationChain.Name,
		string(msg.Type),
		"completed",
	).Inc()

	return nil
}

// healthCheck periodically checks the health of blockchain clients
func (r *Relayer) healthCheck(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck checks the health of all blockchain clients
func (r *Relayer) performHealthCheck(ctx context.Context) {
	for chainName, client := range r.clients {
		// Check client health
		healthy := true
		var blockNumber uint64

		// Try to get latest block number
		var err error
		blockNumber, err = client.GetLatestBlockNumber(ctx)
		if err != nil {
			r.logger.Warn().
				Err(err).
				Str("chain", chainName).
				Msg("Chain health check failed")
			healthy = false
		}

		// Update metrics
		chainType := string(client.GetChainType())
		monitoring.UpdateChainHealth(chainName, chainType, healthy)
		if healthy {
			monitoring.UpdateChainBlockNumber(chainName, blockNumber)
		}

		r.logger.Debug().
			Str("chain", chainName).
			Bool("healthy", healthy).
			Uint64("block_number", blockNumber).
			Msg("Chain health check")
	}
}

// collectMetrics periodically collects and updates metrics
func (r *Relayer) collectMetrics(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.updateMetrics(ctx)
		}
	}
}

// updateMetrics updates various metrics
func (r *Relayer) updateMetrics(ctx context.Context) {
	// Get queue stats
	if statsGetter, ok := r.queue.(*queue.NATSQueue); ok {
		stats, err := statsGetter.GetStats()
		if err != nil {
			r.logger.Warn().Err(err).Msg("Failed to get queue stats")
		} else {
			// Queue metrics (if available in monitoring package)
			r.logger.Debug().
				Uint64("messages", stats.Messages).
				Int("consumers", stats.Consumers).
				Msg("Queue stats")
		}
	}

	// Get pending messages count from database
	pendingCount, err := r.db.GetPendingMessagesCount(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to get pending messages count")
	} else {
		r.logger.Debug().Int64("pending_count", pendingCount).Msg("Pending messages")
	}

	// Get failed messages count
	failedCount, err := r.db.GetFailedMessagesCount(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to get failed messages count")
	} else {
		r.logger.Debug().Int64("failed_count", failedCount).Msg("Failed messages")
	}
}

// ProcessPendingMessages processes messages that are stuck in pending state
// This should be called periodically to handle any messages that failed to process
func (r *Relayer) ProcessPendingMessages(ctx context.Context) error {
	r.logger.Info().Msg("Processing pending messages")

	// Get pending messages from database
	messages, err := r.db.GetPendingMessages(ctx, 100) // Process up to 100 at a time
	if err != nil {
		return fmt.Errorf("failed to get pending messages: %w", err)
	}

	if len(messages) == 0 {
		return nil
	}

	r.logger.Info().
		Int("count", len(messages)).
		Msg("Found pending messages to process")

	// Process each message
	for _, msg := range messages {
		// Check if message is too old
		if time.Since(msg.CreatedAt) > 24*time.Hour {
			r.logger.Warn().
				Str("message_id", msg.ID).
				Msg("Message too old, marking as failed")

			if err := r.db.UpdateMessageStatus(ctx, msg.ID, types.MessageStatusFailed, "timeout"); err != nil {
				r.logger.Error().
					Err(err).
					Str("message_id", msg.ID).
					Msg("Failed to update message status")
			}
			continue
		}

		// Re-publish to queue
		if err := r.queue.Publish(ctx, &msg); err != nil {
			r.logger.Error().
				Err(err).
				Str("message_id", msg.ID).
				Msg("Failed to re-publish message")
			continue
		}

		r.logger.Info().
			Str("message_id", msg.ID).
			Msg("Re-published pending message")
	}

	return nil
}

// GetStats returns relayer statistics
func (r *Relayer) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		Workers: r.workers,
	}

	// Get message counts
	var err error
	stats.PendingMessages, err = r.db.GetPendingMessagesCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}

	stats.ProcessedMessages, err = r.db.GetProcessedMessagesCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed count: %w", err)
	}

	stats.FailedMessages, err = r.db.GetFailedMessagesCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed count: %w", err)
	}

	return stats, nil
}

// Stats represents relayer statistics
type Stats struct {
	Workers           int
	PendingMessages   int64
	ProcessedMessages int64
	FailedMessages    int64
}
