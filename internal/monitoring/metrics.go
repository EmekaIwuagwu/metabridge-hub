package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Message metrics
	MessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_messages_total",
			Help: "Total number of bridge messages",
		},
		[]string{"source_chain", "dest_chain", "type", "status"},
	)

	MessagesProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bridge_message_processing_duration_seconds",
			Help:    "Time taken to process a bridge message",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"source_chain", "dest_chain"},
	)

	MessagesPending = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_messages_pending",
			Help: "Number of pending bridge messages",
		},
		[]string{"source_chain", "dest_chain"},
	)

	// Transaction metrics
	TransactionValueUSD = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bridge_transaction_value_usd",
			Help:    "Transaction value in USD",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"source_chain", "dest_chain", "token"},
	)

	TransactionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_transactions_total",
			Help: "Total number of blockchain transactions",
		},
		[]string{"chain", "type", "status"},
	)

	// Gas price metrics
	GasPrice = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_gas_price_gwei",
			Help: "Current gas price in Gwei",
		},
		[]string{"chain"},
	)

	GasUsed = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bridge_gas_used",
			Help:    "Gas used for transactions",
			Buckets: []float64{21000, 50000, 100000, 200000, 500000, 1000000},
		},
		[]string{"chain", "operation"},
	)

	// Chain health metrics
	ChainHealthy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_chain_healthy",
			Help: "Chain health status (1 = healthy, 0 = unhealthy)",
		},
		[]string{"chain", "chain_type"},
	)

	ChainBlockNumber = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_chain_block_number",
			Help: "Latest block number for each chain",
		},
		[]string{"chain"},
	)

	ChainBlockLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_chain_block_lag",
			Help: "Block lag between local and remote chain",
		},
		[]string{"chain"},
	)

	// Validator metrics
	ValidatorSignaturesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_validator_signatures_total",
			Help: "Total number of validator signatures",
		},
		[]string{"validator", "chain_type"},
	)

	ValidatorSignatureTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bridge_validator_signature_time_seconds",
			Help:    "Time taken to collect required signatures",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"message_type"},
	)

	// Security metrics
	SuspiciousTransactions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_suspicious_transactions_total",
			Help: "Number of transactions flagged as suspicious",
		},
		[]string{"reason", "source_chain"},
	)

	RateLimitExceeded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_rate_limit_exceeded_total",
			Help: "Number of times rate limit was exceeded",
		},
		[]string{"chain", "address"},
	)

	EmergencyPauseActivations = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "bridge_emergency_pause_activations_total",
			Help: "Number of times emergency pause was activated",
		},
	)

	// API metrics
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bridge_api_request_duration_seconds",
			Help:    "API request duration",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 5},
		},
		[]string{"method", "endpoint"},
	)

	// Database metrics
	DatabaseConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "bridge_database_connections_open",
			Help: "Number of open database connections",
		},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bridge_database_query_duration_seconds",
			Help:    "Database query duration",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 5},
		},
		[]string{"operation"},
	)

	// Queue metrics
	QueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_queue_depth",
			Help: "Number of messages in queue",
		},
		[]string{"queue"},
	)

	QueueProcessingRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_queue_processing_rate",
			Help: "Message processing rate (messages/second)",
		},
		[]string{"queue"},
	)

	// Relayer metrics
	RelayerWorkersActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "bridge_relayer_workers_active",
			Help: "Number of active relayer workers",
		},
	)

	RelayerRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_relayer_retries_total",
			Help: "Total number of message processing retries",
		},
		[]string{"reason"},
	)

	// Listener metrics
	ListenerEventsDetected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_listener_events_detected_total",
			Help: "Total number of events detected by listeners",
		},
		[]string{"chain", "event_type"},
	)

	ListenerBlocksProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_listener_blocks_processed_total",
			Help: "Total number of blocks processed by listeners",
		},
		[]string{"chain"},
	)

	ListenerLastBlockProcessed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_listener_last_block_processed",
			Help: "Last block number processed by listener",
		},
		[]string{"chain"},
	)
)

// RecordMessageProcessed records a processed message
func RecordMessageProcessed(sourceChain, destChain, msgType, status string, duration float64) {
	MessagesTotal.WithLabelValues(sourceChain, destChain, msgType, status).Inc()
	MessagesProcessingDuration.WithLabelValues(sourceChain, destChain).Observe(duration)
}

// RecordTransactionValue records a transaction value
func RecordTransactionValue(sourceChain, destChain, token string, valueUSD float64) {
	TransactionValueUSD.WithLabelValues(sourceChain, destChain, token).Observe(valueUSD)
}

// UpdateChainHealth updates chain health status
func UpdateChainHealth(chain, chainType string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	ChainHealthy.WithLabelValues(chain, chainType).Set(value)
}

// UpdateChainBlockNumber updates the latest block number for a chain
func UpdateChainBlockNumber(chain string, blockNumber uint64) {
	ChainBlockNumber.WithLabelValues(chain).Set(float64(blockNumber))
}

// RecordSuspiciousTransaction records a suspicious transaction
func RecordSuspiciousTransaction(reason, sourceChain string) {
	SuspiciousTransactions.WithLabelValues(reason, sourceChain).Inc()
}

// RecordRateLimitExceeded records a rate limit violation
func RecordRateLimitExceeded(chain, address string) {
	RateLimitExceeded.WithLabelValues(chain, address).Inc()
}

// RecordAPIRequest records an API request
func RecordAPIRequest(method, endpoint, status string, duration float64) {
	APIRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	APIRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}
