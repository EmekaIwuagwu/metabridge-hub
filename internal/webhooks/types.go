package webhooks

import (
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
)

// EventType represents different webhook event types
type EventType string

const (
	EventMessageCreated   EventType = "message.created"
	EventMessagePending   EventType = "message.pending"
	EventMessageSubmitted EventType = "message.submitted"
	EventMessageConfirmed EventType = "message.confirmed"
	EventMessageFinalized EventType = "message.finalized"
	EventMessageFailed    EventType = "message.failed"
	EventBatchCreated     EventType = "batch.created"
	EventBatchSubmitted   EventType = "batch.submitted"
	EventBatchConfirmed   EventType = "batch.confirmed"
	EventBatchFailed      EventType = "batch.failed"
)

// WebhookStatus represents the status of a webhook registration
type WebhookStatus string

const (
	WebhookStatusActive   WebhookStatus = "ACTIVE"
	WebhookStatusPaused   WebhookStatus = "PAUSED"
	WebhookStatusDisabled WebhookStatus = "DISABLED"
	WebhookStatusFailed   WebhookStatus = "FAILED"
)

// Webhook represents a webhook registration
type Webhook struct {
	ID          string        `json:"id"`
	URL         string        `json:"url"`
	Secret      string        `json:"secret,omitempty"`
	Events      []EventType   `json:"events"`
	Status      WebhookStatus `json:"status"`
	Description string        `json:"description,omitempty"`
	CreatedBy   string        `json:"created_by"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	LastUsedAt  *time.Time    `json:"last_used_at,omitempty"`
	FailCount   int           `json:"fail_count"`
	SuccessCount int          `json:"success_count"`

	// Filtering options
	SourceChains []string `json:"source_chains,omitempty"`
	DestChains   []string `json:"dest_chains,omitempty"`
	MinAmount    string   `json:"min_amount,omitempty"`
	MaxAmount    string   `json:"max_amount,omitempty"`
}

// WebhookEvent represents a webhook event to be delivered
type WebhookEvent struct {
	ID          string                 `json:"id"`
	WebhookID   string                 `json:"webhook_id"`
	EventType   EventType              `json:"event_type"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   time.Time              `json:"timestamp"`
	Signature   string                 `json:"signature,omitempty"`
	DeliveryURL string                 `json:"delivery_url"`
}

// WebhookDeliveryAttempt represents an attempt to deliver a webhook
type WebhookDeliveryAttempt struct {
	ID            string    `json:"id"`
	EventID       string    `json:"event_id"`
	WebhookID     string    `json:"webhook_id"`
	AttemptNumber int       `json:"attempt_number"`
	StatusCode    int       `json:"status_code"`
	ResponseBody  string    `json:"response_body,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	Duration      int64     `json:"duration_ms"`
	Success       bool      `json:"success"`
	AttemptedAt   time.Time `json:"attempted_at"`
	NextRetryAt   *time.Time `json:"next_retry_at,omitempty"`
}

// WebhookDeliveryConfig represents webhook delivery configuration
type WebhookDeliveryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	RetryDelays     []time.Duration `json:"retry_delays"`
	TimeoutDuration time.Duration `json:"timeout_duration"`
	MaxConcurrent   int           `json:"max_concurrent"`
}

// DefaultDeliveryConfig returns default webhook delivery configuration
func DefaultDeliveryConfig() *WebhookDeliveryConfig {
	return &WebhookDeliveryConfig{
		MaxRetries: 5,
		RetryDelays: []time.Duration{
			1 * time.Minute,
			5 * time.Minute,
			15 * time.Minute,
			1 * time.Hour,
			6 * time.Hour,
		},
		TimeoutDuration: 30 * time.Second,
		MaxConcurrent:   10,
	}
}

// MessageEvent represents a message-related webhook event payload
type MessageEvent struct {
	Message       *types.CrossChainMessage `json:"message"`
	Event         EventType                `json:"event"`
	Timestamp     time.Time                `json:"timestamp"`
	TxHash        string                   `json:"tx_hash,omitempty"`
	BlockNumber   uint64                   `json:"block_number,omitempty"`
	Confirmations uint64                   `json:"confirmations,omitempty"`
	ErrorMessage  string                   `json:"error_message,omitempty"`
}

// BatchEvent represents a batch-related webhook event payload
type BatchEvent struct {
	BatchID       string    `json:"batch_id"`
	MessageCount  int       `json:"message_count"`
	Event         EventType `json:"event"`
	Timestamp     time.Time `json:"timestamp"`
	TxHash        string    `json:"tx_hash,omitempty"`
	GasSavedWei   string    `json:"gas_saved_wei,omitempty"`
	SavingsPercent float64  `json:"savings_percent,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// TrackingQuery represents a query for tracking messages
type TrackingQuery struct {
	MessageID    string    `json:"message_id,omitempty"`
	TxHash       string    `json:"tx_hash,omitempty"`
	Sender       string    `json:"sender,omitempty"`
	Recipient    string    `json:"recipient,omitempty"`
	SourceChain  string    `json:"source_chain,omitempty"`
	DestChain    string    `json:"dest_chain,omitempty"`
	Status       string    `json:"status,omitempty"`
	FromDate     *time.Time `json:"from_date,omitempty"`
	ToDate       *time.Time `json:"to_date,omitempty"`
	Limit        int       `json:"limit,omitempty"`
	Offset       int       `json:"offset,omitempty"`
}

// TrackingResult represents the result of a tracking query
type TrackingResult struct {
	Messages     []*types.CrossChainMessage `json:"messages"`
	TotalCount   int                        `json:"total_count"`
	Limit        int                        `json:"limit"`
	Offset       int                        `json:"offset"`
	HasMore      bool                       `json:"has_more"`
}

// MessageTimeline represents the timeline of events for a message
type MessageTimeline struct {
	MessageID    string           `json:"message_id"`
	Events       []TimelineEvent  `json:"events"`
	CurrentStatus string          `json:"current_status"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
}

// TimelineEvent represents a single event in a message timeline
type TimelineEvent struct {
	EventType   string    `json:"event_type"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
	TxHash      string    `json:"tx_hash,omitempty"`
	BlockNumber uint64    `json:"block_number,omitempty"`
	ChainID     string    `json:"chain_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
