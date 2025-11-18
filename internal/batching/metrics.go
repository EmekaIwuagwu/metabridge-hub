package batching

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Batch counters
	BatchesCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_batches_created_total",
		Help: "Total number of batches created",
	})

	BatchesSubmitted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_batches_submitted_total",
		Help: "Total number of batches submitted to blockchain",
	})

	BatchesConfirmed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_batches_confirmed_total",
		Help: "Total number of batches confirmed on blockchain",
	})

	BatchesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_batches_failed_total",
		Help: "Total number of batches that failed",
	})

	// Message counters
	MessagesBatched = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_messages_batched_total",
		Help: "Total number of messages processed via batching",
	})

	// Batch sizes
	BatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_batch_size",
		Help:    "Number of messages per batch",
		Buckets: []float64{5, 10, 20, 30, 50, 75, 100},
	})

	BatchWaitTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_batch_wait_time_seconds",
		Help:    "Time messages wait before batch submission",
		Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
	})

	// Gas savings
	GasSavedWei = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_gas_saved_wei_total",
		Help: "Total gas saved through batching (in wei)",
	})

	SavingsPercentage = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_batch_savings_percentage",
		Help:    "Percentage of gas saved per batch",
		Buckets: []float64{50, 60, 70, 80, 85, 90, 95},
	})

	// Pending batches
	PendingBatches = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_pending_batches",
		Help: "Number of batches currently pending submission",
	})

	PendingMessagesInBatches = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_pending_messages_in_batches",
		Help: "Number of messages currently in pending batches",
	})

	// Batch value
	BatchTotalValue = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_batch_total_value_wei",
		Help:    "Total value of assets in batch (in wei)",
		Buckets: prometheus.ExponentialBuckets(1e15, 10, 10), // From 0.001 ETH to 10000 ETH
	})

	// Processing time
	BatchProcessingTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_batch_processing_time_seconds",
		Help:    "Time taken to process a batch from creation to confirmation",
		Buckets: []float64{60, 120, 300, 600, 1200, 1800, 3600},
	})
)

// RecordBatchCreated records metrics when a batch is created
func RecordBatchCreated(messageCount int) {
	BatchesCreated.Inc()
	BatchSize.Observe(float64(messageCount))
	PendingBatches.Inc()
	PendingMessagesInBatches.Add(float64(messageCount))
}

// RecordBatchSubmitted records metrics when a batch is submitted
func RecordBatchSubmitted(messageCount int, waitTimeSeconds float64) {
	BatchesSubmitted.Inc()
	BatchWaitTime.Observe(waitTimeSeconds)
	PendingBatches.Dec()
	PendingMessagesInBatches.Sub(float64(messageCount))
}

// RecordBatchConfirmed records metrics when a batch is confirmed
func RecordBatchConfirmed(messageCount int, gasSavedWei float64, savingsPercent float64, processingTimeSeconds float64) {
	BatchesConfirmed.Inc()
	MessagesBatched.Add(float64(messageCount))
	GasSavedWei.Add(gasSavedWei)
	SavingsPercentage.Observe(savingsPercent)
	BatchProcessingTime.Observe(processingTimeSeconds)
}

// RecordBatchFailed records metrics when a batch fails
func RecordBatchFailed() {
	BatchesFailed.Inc()
}
