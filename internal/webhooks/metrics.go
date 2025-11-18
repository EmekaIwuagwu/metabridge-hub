package webhooks

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Webhook registration metrics
	WebhooksRegistered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_webhooks_registered_total",
		Help: "Total number of registered webhooks",
	})

	WebhooksActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_webhooks_active",
		Help: "Number of active webhooks",
	})

	WebhooksPaused = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_webhooks_paused",
		Help: "Number of paused webhooks",
	})

	WebhooksFailed = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_webhooks_failed",
		Help: "Number of failed webhooks",
	})

	// Webhook delivery metrics
	WebhooksDispatched = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_webhooks_dispatched_total",
		Help: "Total number of webhooks dispatched",
	})

	WebhooksDelivered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_webhooks_delivered_total",
		Help: "Total number of successfully delivered webhooks",
	})

	WebhooksFailedDelivery = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_webhooks_failed_delivery_total",
		Help: "Total number of failed webhook deliveries",
	})

	WebhooksRetried = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_webhooks_retried_total",
		Help: "Total number of webhook retry attempts",
	})

	// Webhook latency
	WebhookDeliveryLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_webhook_delivery_latency_seconds",
		Help:    "Webhook delivery latency in seconds",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
	})

	// Webhook queue metrics
	WebhookQueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_webhook_queue_size",
		Help: "Current size of webhook delivery queue",
	})

	WebhookQueueCapacity = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_webhook_queue_capacity",
		Help: "Maximum capacity of webhook delivery queue",
	})

	// Event type metrics
	WebhookEventsByType = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_webhook_events_by_type_total",
			Help: "Total number of webhook events by type",
		},
		[]string{"event_type"},
	)

	// Webhook response status codes
	WebhookResponseStatus = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_webhook_response_status_total",
			Help: "Total number of webhook responses by status code",
		},
		[]string{"status_code"},
	)

	// Tracking metrics
	MessagesTracked = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_messages_tracked_total",
		Help: "Total number of message tracking requests",
	})

	TrackingQueriesExecuted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_tracking_queries_total",
		Help: "Total number of tracking queries executed",
	})

	TrackingQueryLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_tracking_query_latency_seconds",
		Help:    "Tracking query latency in seconds",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
	})
)

// Record functions for easier metric recording

// RecordWebhookRegistered increments the registered webhooks gauge
func RecordWebhookRegistered() {
	WebhooksRegistered.Inc()
}

// RecordWebhookDeleted decrements the registered webhooks gauge
func RecordWebhookDeleted() {
	WebhooksRegistered.Dec()
}

// RecordWebhookStatusChange updates webhook status gauges
func RecordWebhookStatusChange(oldStatus, newStatus WebhookStatus) {
	// Decrement old status
	switch oldStatus {
	case WebhookStatusActive:
		WebhooksActive.Dec()
	case WebhookStatusPaused:
		WebhooksPaused.Dec()
	case WebhookStatusFailed:
		WebhooksFailed.Dec()
	}

	// Increment new status
	switch newStatus {
	case WebhookStatusActive:
		WebhooksActive.Inc()
	case WebhookStatusPaused:
		WebhooksPaused.Inc()
	case WebhookStatusFailed:
		WebhooksFailed.Inc()
	}
}

// RecordWebhookDispatched records a webhook dispatch
func RecordWebhookDispatched() {
	WebhooksDispatched.Inc()
}

// RecordWebhookDelivered records a successful webhook delivery
func RecordWebhookDelivered() {
	WebhooksDelivered.Inc()
}

// RecordWebhookFailed records a failed webhook delivery
func RecordWebhookFailed() {
	WebhooksFailedDelivery.Inc()
}

// RecordWebhookRetry records a webhook retry attempt
func RecordWebhookRetry() {
	WebhooksRetried.Inc()
}

// RecordWebhookLatency records webhook delivery latency
func RecordWebhookLatency(seconds float64) {
	WebhookDeliveryLatency.Observe(seconds)
}

// RecordWebhookEvent records a webhook event by type
func RecordWebhookEvent(eventType EventType) {
	WebhookEventsByType.WithLabelValues(string(eventType)).Inc()
}

// RecordWebhookResponse records a webhook response status
func RecordWebhookResponse(statusCode int) {
	WebhookResponseStatus.WithLabelValues(fmt.Sprintf("%d", statusCode)).Inc()
}

// RecordMessageTracked records a message tracking request
func RecordMessageTracked() {
	MessagesTracked.Inc()
}

// RecordTrackingQuery records a tracking query execution
func RecordTrackingQuery() {
	TrackingQueriesExecuted.Inc()
}

// RecordTrackingQueryLatency records tracking query latency
func RecordTrackingQueryLatency(seconds float64) {
	TrackingQueryLatency.Observe(seconds)
}

// SetWebhookQueueSize updates the webhook queue size gauge
func SetWebhookQueueSize(size int) {
	WebhookQueueSize.Set(float64(size))
}

// SetWebhookQueueCapacity updates the webhook queue capacity gauge
func SetWebhookQueueCapacity(capacity int) {
	WebhookQueueCapacity.Set(float64(capacity))
}

// Import fmt for string formatting
import "fmt"
