package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// Queue represents a message queue interface
type Queue interface {
	// Publish publishes a message to the queue
	Publish(ctx context.Context, msg *types.CrossChainMessage) error

	// Subscribe subscribes to messages from the queue
	Subscribe(ctx context.Context, handler MessageHandler) error

	// Close closes the queue connection
	Close() error
}

// MessageHandler handles incoming messages
type MessageHandler func(ctx context.Context, msg *types.CrossChainMessage) error

// NATSQueue implements Queue using NATS JetStream
type NATSQueue struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	config     *config.QueueConfig
	logger     zerolog.Logger
	streamName string
	subject    string
}

// NewNATSQueue creates a new NATS queue
func NewNATSQueue(cfg *config.QueueConfig, logger zerolog.Logger) (*NATSQueue, error) {
	if cfg.Type != "nats" {
		return nil, fmt.Errorf("invalid queue type: %s, expected nats", cfg.Type)
	}

	// Connect to NATS
	opts := []nats.Option{
		nats.Name("metabridge"),
		nats.Timeout(10 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
	}

	// Use first URL for connection
	url := cfg.URLs[0]
	if len(cfg.URLs) > 1 {
		// If multiple URLs, use all for redundancy
		opts = append(opts, nats.DontRandomize())
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	queue := &NATSQueue{
		conn:       conn,
		js:         js,
		config:     cfg,
		logger:     logger.With().Str("component", "queue").Logger(),
		streamName: cfg.StreamName,
		subject:    cfg.Subject,
	}

	// Initialize stream
	if err := queue.initializeStream(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize stream: %w", err)
	}

	queue.logger.Info().
		Str("url", url).
		Str("stream", queue.streamName).
		Str("subject", queue.subject).
		Msg("NATS queue initialized")

	return queue, nil
}

// initializeStream creates or updates the JetStream stream
func (q *NATSQueue) initializeStream() error {
	// Check if stream exists
	stream, err := q.js.StreamInfo(q.streamName)
	if err == nil {
		q.logger.Info().
			Str("stream", q.streamName).
			Msg("Stream already exists")
		return nil
	}

	// Create stream
	streamConfig := &nats.StreamConfig{
		Name:      q.streamName,
		Subjects:  []string{q.subject},
		Storage:   nats.FileStorage,
		Retention: nats.WorkQueuePolicy,
		MaxAge:    7 * 24 * time.Hour, // Keep messages for 7 days
		MaxMsgs:   100000,             // Keep up to 100k messages
		Discard:   nats.DiscardOld,
	}

	stream, err = q.js.AddStream(streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	q.logger.Info().
		Str("stream", stream.Config.Name).
		Msg("Stream created successfully")

	return nil
}

// Publish publishes a message to the queue
func (q *NATSQueue) Publish(ctx context.Context, msg *types.CrossChainMessage) error {
	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to JetStream
	ack, err := q.js.Publish(q.subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	q.logger.Debug().
		Str("message_id", msg.ID).
		Uint64("stream_seq", ack.Sequence).
		Msg("Message published to queue")

	return nil
}

// Subscribe subscribes to messages from the queue
func (q *NATSQueue) Subscribe(ctx context.Context, handler MessageHandler) error {
	// Create durable consumer
	consumerName := "metabridge-relayer"

	// Subscribe to subject
	sub, err := q.js.QueueSubscribe(
		q.subject,
		consumerName,
		func(m *nats.Msg) {
			// Deserialize message
			var msg types.CrossChainMessage
			if err := json.Unmarshal(m.Data, &msg); err != nil {
				q.logger.Error().
					Err(err).
					Msg("Failed to unmarshal message")
				m.Nak() // Negative acknowledge
				return
			}

			q.logger.Debug().
				Str("message_id", msg.ID).
				Msg("Processing message from queue")

			// Handle message
			if err := handler(ctx, &msg); err != nil {
				q.logger.Error().
					Err(err).
					Str("message_id", msg.ID).
					Msg("Failed to handle message")

				// Check retry count
				metadata, _ := m.Metadata()
				if metadata != nil && metadata.NumDelivered >= uint64(q.config.MaxRetries) {
					q.logger.Warn().
						Str("message_id", msg.ID).
						Uint64("deliveries", metadata.NumDelivered).
						Msg("Max retries exceeded, message will be discarded")
					m.Term() // Terminate message
				} else {
					m.NakWithDelay(5 * time.Second) // Retry after delay
				}
				return
			}

			// Acknowledge successful processing
			m.Ack()

			q.logger.Info().
				Str("message_id", msg.ID).
				Msg("Message processed successfully")
		},
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(q.config.MaxRetries),
	)

	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	q.logger.Info().
		Str("subject", q.subject).
		Str("consumer", consumerName).
		Msg("Subscribed to queue")

	// Wait for context cancellation
	<-ctx.Done()

	// Unsubscribe
	if err := sub.Unsubscribe(); err != nil {
		q.logger.Error().Err(err).Msg("Error unsubscribing")
	}

	return nil
}

// Close closes the queue connection
func (q *NATSQueue) Close() error {
	q.logger.Info().Msg("Closing NATS queue connection")

	if q.conn != nil {
		q.conn.Close()
	}

	return nil
}

// GetStats returns queue statistics
func (q *NATSQueue) GetStats() (*QueueStats, error) {
	stream, err := q.js.StreamInfo(q.streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	return &QueueStats{
		Messages:  stream.State.Msgs,
		Bytes:     stream.State.Bytes,
		FirstSeq:  stream.State.FirstSeq,
		LastSeq:   stream.State.LastSeq,
		Consumers: stream.State.Consumers,
	}, nil
}

// QueueStats represents queue statistics
type QueueStats struct {
	Messages  uint64
	Bytes     uint64
	FirstSeq  uint64
	LastSeq   uint64
	Consumers int
}

// Drain drains pending messages (admin function)
func (q *NATSQueue) Drain(ctx context.Context) error {
	q.logger.Warn().Msg("Draining queue")

	if err := q.conn.Drain(); err != nil {
		return fmt.Errorf("failed to drain: %w", err)
	}

	return nil
}

// Purge purges all messages from the stream (admin function)
func (q *NATSQueue) Purge() error {
	q.logger.Warn().Msg("Purging queue")

	if err := q.js.PurgeStream(q.streamName); err != nil {
		return fmt.Errorf("failed to purge stream: %w", err)
	}

	return nil
}
