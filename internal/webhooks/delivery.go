package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DeliveryService handles webhook delivery with retry logic
type DeliveryService struct {
	config    *WebhookDeliveryConfig
	registry  *Registry
	db        *database.DB
	logger    zerolog.Logger
	client    *http.Client
	eventChan chan *WebhookEvent
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewDeliveryService creates a new webhook delivery service
func NewDeliveryService(
	config *WebhookDeliveryConfig,
	registry *Registry,
	db *database.DB,
	logger zerolog.Logger,
) *DeliveryService {
	if config == nil {
		config = DefaultDeliveryConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DeliveryService{
		config:   config,
		registry: registry,
		db:       db,
		logger:   logger.With().Str("component", "webhook-delivery").Logger(),
		client: &http.Client{
			Timeout: config.TimeoutDuration,
		},
		eventChan: make(chan *WebhookEvent, 1000),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the webhook delivery service
func (s *DeliveryService) Start(ctx context.Context) error {
	s.logger.Info().
		Int("max_concurrent", s.config.MaxConcurrent).
		Int("max_retries", s.config.MaxRetries).
		Msg("Starting webhook delivery service")

	// Start worker goroutines
	for i := 0; i < s.config.MaxConcurrent; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start retry processor
	s.wg.Add(1)
	go s.retryProcessor()

	return nil
}

// Stop stops the webhook delivery service
func (s *DeliveryService) Stop() error {
	s.logger.Info().Msg("Stopping webhook delivery service")
	s.cancel()
	close(s.eventChan)
	s.wg.Wait()
	return nil
}

// Dispatch queues a webhook event for delivery
func (s *DeliveryService) Dispatch(event *WebhookEvent) error {
	select {
	case s.eventChan <- event:
		RecordWebhookDispatched()
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout queuing webhook event")
	}
}

// DispatchToWebhooks dispatches an event to all registered webhooks
func (s *DeliveryService) DispatchToWebhooks(ctx context.Context, eventType EventType, payload map[string]interface{}) error {
	// Get all active webhooks for this event type
	webhooks, err := s.registry.GetActiveWebhooksForEvent(ctx, eventType)
	if err != nil {
		return fmt.Errorf("failed to get webhooks: %w", err)
	}

	if len(webhooks) == 0 {
		s.logger.Debug().
			Str("event_type", string(eventType)).
			Msg("No active webhooks for event type")
		return nil
	}

	// Filter webhooks based on payload criteria
	filteredWebhooks := s.filterWebhooks(webhooks, payload)

	// Create and dispatch events
	for _, webhook := range filteredWebhooks {
		event := &WebhookEvent{
			ID:          uuid.New().String(),
			WebhookID:   webhook.ID,
			EventType:   eventType,
			Payload:     payload,
			Timestamp:   time.Now().UTC(),
			DeliveryURL: webhook.URL,
		}

		// Generate HMAC signature
		event.Signature = s.generateSignature(webhook.Secret, payload)

		// Save event to database
		if err := s.saveEvent(ctx, event); err != nil {
			s.logger.Error().
				Err(err).
				Str("event_id", event.ID).
				Msg("Failed to save webhook event")
			continue
		}

		// Dispatch for delivery
		if err := s.Dispatch(event); err != nil {
			s.logger.Error().
				Err(err).
				Str("event_id", event.ID).
				Msg("Failed to dispatch webhook event")
		}
	}

	s.logger.Info().
		Str("event_type", string(eventType)).
		Int("webhooks", len(filteredWebhooks)).
		Msg("Dispatched webhook event to webhooks")

	return nil
}

// worker processes webhook events
func (s *DeliveryService) worker(id int) {
	defer s.wg.Done()

	s.logger.Debug().Int("worker_id", id).Msg("Webhook worker started")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug().Int("worker_id", id).Msg("Webhook worker stopped")
			return
		case event, ok := <-s.eventChan:
			if !ok {
				return
			}
			s.deliverEvent(event, 1)
		}
	}
}

// deliverEvent attempts to deliver a webhook event
func (s *DeliveryService) deliverEvent(event *WebhookEvent, attemptNumber int) {
	start := time.Now()

	s.logger.Debug().
		Str("event_id", event.ID).
		Str("webhook_id", event.WebhookID).
		Str("url", event.DeliveryURL).
		Int("attempt", attemptNumber).
		Msg("Delivering webhook")

	// Prepare request
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("event_id", event.ID).
			Msg("Failed to marshal webhook payload")
		s.recordAttempt(event, attemptNumber, 0, "", err.Error(), false, time.Since(start), nil)
		return
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", event.DeliveryURL, bytes.NewReader(payload))
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("event_id", event.ID).
			Msg("Failed to create webhook request")
		s.recordAttempt(event, attemptNumber, 0, "", err.Error(), false, time.Since(start), nil)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Metabridge-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", event.WebhookID)
	req.Header.Set("X-Event-ID", event.ID)
	req.Header.Set("X-Event-Type", string(event.EventType))
	req.Header.Set("X-Webhook-Signature", event.Signature)
	req.Header.Set("X-Webhook-Timestamp", event.Timestamp.Format(time.RFC3339))

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Str("event_id", event.ID).
			Int("attempt", attemptNumber).
			Msg("Webhook delivery failed")

		s.recordAttempt(event, attemptNumber, 0, "", err.Error(), false, time.Since(start), s.calculateNextRetry(attemptNumber))
		s.registry.IncrementFailCount(s.ctx, event.WebhookID)
		RecordWebhookFailed()
		return
	}
	defer resp.Body.Close()

	// Read response
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024)) // Limit to 10KB

	// Check status code
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	s.recordAttempt(event, attemptNumber, resp.StatusCode, string(responseBody), "", success, time.Since(start), s.calculateNextRetry(attemptNumber))

	if success {
		s.logger.Info().
			Str("event_id", event.ID).
			Int("status_code", resp.StatusCode).
			Int("attempt", attemptNumber).
			Msg("Webhook delivered successfully")

		s.registry.IncrementSuccessCount(s.ctx, event.WebhookID)
		RecordWebhookDelivered()
		RecordWebhookLatency(time.Since(start).Seconds())
	} else {
		s.logger.Warn().
			Str("event_id", event.ID).
			Int("status_code", resp.StatusCode).
			Int("attempt", attemptNumber).
			Msg("Webhook delivery failed with non-2xx status")

		s.registry.IncrementFailCount(s.ctx, event.WebhookID)
		RecordWebhookFailed()
	}
}

// retryProcessor periodically checks for failed deliveries and retries them
func (s *DeliveryService) retryProcessor() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processRetries()
		}
	}
}

// processRetries finds and retries failed webhook deliveries
func (s *DeliveryService) processRetries() {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	query := `
		SELECT DISTINCT ON (event_id)
			wa.event_id,
			we.webhook_id,
			we.event_type,
			we.payload,
			we.timestamp,
			we.signature,
			we.delivery_url,
			wa.attempt_number
		FROM webhook_attempts wa
		JOIN webhook_events we ON wa.event_id = we.id
		WHERE wa.success = false
		AND wa.next_retry_at IS NOT NULL
		AND wa.next_retry_at <= NOW()
		AND wa.attempt_number < $1
		ORDER BY event_id, wa.attempted_at DESC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, query, s.config.MaxRetries)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to query webhook retries")
		return
	}
	defer rows.Close()

	retryCount := 0
	for rows.Next() {
		var event WebhookEvent
		var attemptNumber int
		var payloadJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.WebhookID,
			&event.EventType,
			&payloadJSON,
			&event.Timestamp,
			&event.Signature,
			&event.DeliveryURL,
			&attemptNumber,
		)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to scan webhook retry")
			continue
		}

		// Unmarshal payload
		if err := json.Unmarshal(payloadJSON, &event.Payload); err != nil {
			s.logger.Error().Err(err).Msg("Failed to unmarshal webhook payload")
			continue
		}

		// Retry delivery
		go s.deliverEvent(&event, attemptNumber+1)
		retryCount++
	}

	if retryCount > 0 {
		s.logger.Info().
			Int("count", retryCount).
			Msg("Queued webhook retries")
	}
}

// saveEvent saves a webhook event to the database
func (s *DeliveryService) saveEvent(ctx context.Context, event *WebhookEvent) error {
	query := `
		INSERT INTO webhook_events (
			id, webhook_id, event_type, payload,
			timestamp, signature, delivery_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		event.ID,
		event.WebhookID,
		event.EventType,
		payloadJSON,
		event.Timestamp,
		event.Signature,
		event.DeliveryURL,
	)

	return err
}

// recordAttempt records a delivery attempt
func (s *DeliveryService) recordAttempt(
	event *WebhookEvent,
	attemptNumber int,
	statusCode int,
	responseBody string,
	errorMessage string,
	success bool,
	duration time.Duration,
	nextRetry *time.Time,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO webhook_attempts (
			id, event_id, webhook_id, attempt_number,
			status_code, response_body, error_message,
			success, duration_ms, attempted_at, next_retry_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := s.db.ExecContext(ctx, query,
		uuid.New().String(),
		event.ID,
		event.WebhookID,
		attemptNumber,
		statusCode,
		truncateString(responseBody, 1000),
		errorMessage,
		success,
		duration.Milliseconds(),
		time.Now().UTC(),
		nextRetry,
	)

	if err != nil {
		s.logger.Error().
			Err(err).
			Str("event_id", event.ID).
			Msg("Failed to record webhook attempt")
	}
}

// calculateNextRetry calculates the next retry time
func (s *DeliveryService) calculateNextRetry(attemptNumber int) *time.Time {
	if attemptNumber >= s.config.MaxRetries {
		return nil
	}

	var delay time.Duration
	if attemptNumber-1 < len(s.config.RetryDelays) {
		delay = s.config.RetryDelays[attemptNumber-1]
	} else {
		// Use last configured delay for remaining attempts
		delay = s.config.RetryDelays[len(s.config.RetryDelays)-1]
	}

	nextRetry := time.Now().UTC().Add(delay)
	return &nextRetry
}

// generateSignature generates an HMAC signature for the payload
func (s *DeliveryService) generateSignature(secret string, payload map[string]interface{}) string {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payloadJSON)
	return hex.EncodeToString(h.Sum(nil))
}

// filterWebhooks filters webhooks based on payload criteria
func (s *DeliveryService) filterWebhooks(webhooks []*Webhook, payload map[string]interface{}) []*Webhook {
	filtered := make([]*Webhook, 0, len(webhooks))

	for _, webhook := range webhooks {
		if s.matchesFilters(webhook, payload) {
			filtered = append(filtered, webhook)
		}
	}

	return filtered
}

// matchesFilters checks if a webhook's filters match the payload
func (s *DeliveryService) matchesFilters(webhook *Webhook, payload map[string]interface{}) bool {
	// Check source chain filter
	if len(webhook.SourceChains) > 0 {
		sourceChain, _ := payload["source_chain"].(string)
		if !contains(webhook.SourceChains, sourceChain) {
			return false
		}
	}

	// Check dest chain filter
	if len(webhook.DestChains) > 0 {
		destChain, _ := payload["dest_chain"].(string)
		if !contains(webhook.DestChains, destChain) {
			return false
		}
	}

	// Add more filter logic as needed (min_amount, max_amount, etc.)

	return true
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
