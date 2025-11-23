package batching

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
)

// Batch represents a collection of transactions to be processed together
type Batch struct {
	ID            string
	Messages      []*types.CrossChainMessage
	MerkleRoot    string
	TotalValue    *big.Int
	Status        BatchStatus
	CreatedAt     time.Time
	SubmittedAt   *time.Time
	ConfirmedAt   *time.Time
	SourceChain   string
	DestChain     string
	GasCostSaved  *big.Int
	SubmitterAddr string
	TxHash        string
}

// BatchStatus represents the current state of a batch
type BatchStatus string

const (
	BatchStatusPending   BatchStatus = "PENDING"
	BatchStatusReady     BatchStatus = "READY"
	BatchStatusSubmitted BatchStatus = "SUBMITTED"
	BatchStatusConfirmed BatchStatus = "CONFIRMED"
	BatchStatusFailed    BatchStatus = "FAILED"
)

// BatchConfig holds configuration for batching behavior
type BatchConfig struct {
	// Maximum number of messages in a batch
	MaxBatchSize int

	// Minimum number of messages before submitting
	MinBatchSize int

	// Maximum time to wait before submitting batch (even if not full)
	MaxWaitTime time.Duration

	// Minimum time between batch submissions
	MinSubmissionInterval time.Duration

	// Cost threshold - submit batch if savings exceed this amount
	CostSavingsThreshold *big.Int

	// Enable/disable batching per chain pair
	EnabledChainPairs map[string]bool
}

// DefaultBatchConfig returns sensible default configuration
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		MaxBatchSize:          100,
		MinBatchSize:          5,
		MaxWaitTime:           30 * time.Second,
		MinSubmissionInterval: 10 * time.Second,
		CostSavingsThreshold:  big.NewInt(1000000000000000), // 0.001 ETH
		EnabledChainPairs:     make(map[string]bool),
	}
}

// NewBatch creates a new batch from messages
func NewBatch(messages []*types.CrossChainMessage, sourceChain, destChain string) *Batch {
	batch := &Batch{
		ID:          generateBatchID(messages),
		Messages:    messages,
		TotalValue:  calculateTotalValue(messages),
		Status:      BatchStatusPending,
		CreatedAt:   time.Now(),
		SourceChain: sourceChain,
		DestChain:   destChain,
	}

	return batch
}

// AddMessage adds a message to the batch
func (b *Batch) AddMessage(msg *types.CrossChainMessage) error {
	// Validate message belongs to same chain pair
	if msg.SourceChain.Name != b.SourceChain || msg.DestinationChain.Name != b.DestChain {
		return ErrChainMismatch
	}

	b.Messages = append(b.Messages, msg)
	b.TotalValue.Add(b.TotalValue, extractMessageValue(msg))

	return nil
}

// IsFull checks if batch has reached maximum size
func (b *Batch) IsFull(maxSize int) bool {
	return len(b.Messages) >= maxSize
}

// IsReady checks if batch is ready for submission
func (b *Batch) IsReady(config *BatchConfig) bool {
	// Check minimum size
	if len(b.Messages) < config.MinBatchSize {
		return false
	}

	// Check if full
	if b.IsFull(config.MaxBatchSize) {
		return true
	}

	// Check timeout
	if time.Since(b.CreatedAt) >= config.MaxWaitTime {
		return true
	}

	// Check cost savings threshold
	if config.CostSavingsThreshold != nil && b.GasCostSaved != nil {
		if b.GasCostSaved.Cmp(config.CostSavingsThreshold) >= 0 {
			return true
		}
	}

	return false
}

// MarkSubmitted marks the batch as submitted
func (b *Batch) MarkSubmitted(txHash string) {
	now := time.Now()
	b.Status = BatchStatusSubmitted
	b.SubmittedAt = &now
	b.TxHash = txHash
}

// MarkConfirmed marks the batch as confirmed
func (b *Batch) MarkConfirmed() {
	now := time.Now()
	b.Status = BatchStatusConfirmed
	b.ConfirmedAt = &now
}

// MarkFailed marks the batch as failed
func (b *Batch) MarkFailed() {
	b.Status = BatchStatusFailed
}

// GetMessageIDs returns all message IDs in the batch
func (b *Batch) GetMessageIDs() []string {
	ids := make([]string, len(b.Messages))
	for i, msg := range b.Messages {
		ids[i] = msg.ID
	}
	return ids
}

// Helper functions

func generateBatchID(messages []*types.CrossChainMessage) string {
	hasher := sha256.New()
	for _, msg := range messages {
		hasher.Write([]byte(msg.ID))
	}
	hasher.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

func calculateTotalValue(messages []*types.CrossChainMessage) *big.Int {
	total := big.NewInt(0)
	for _, msg := range messages {
		total.Add(total, extractMessageValue(msg))
	}
	return total
}

func extractMessageValue(msg *types.CrossChainMessage) *big.Int {
	// Decode payload to get amount
	if err := msg.DecodePayload(); err != nil {
		return big.NewInt(0)
	}

	switch msg.Type {
	case types.MessageTypeTokenTransfer:
		if payload, ok := msg.DecodedPayload.(types.TokenTransferPayload); ok {
			amount, _ := new(big.Int).SetString(payload.Amount, 10)
			return amount
		}
	}

	return big.NewInt(0)
}

// Errors
var (
	ErrChainMismatch = &BatchError{Message: "message chain does not match batch"}
	ErrBatchFull     = &BatchError{Message: "batch is full"}
	ErrBatchEmpty    = &BatchError{Message: "batch is empty"}
)

// BatchError represents a batching error
type BatchError struct {
	Message string
}

func (e *BatchError) Error() string {
	return e.Message
}
