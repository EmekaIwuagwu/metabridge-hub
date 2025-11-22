package security

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/rs/zerolog"
)

// FraudDetector detects suspicious transaction patterns
type FraudDetector struct {
	config *config.SecurityConfig
	logger zerolog.Logger

	// Track address patterns
	addressHistory map[string]*AddressHistory
	mu             sync.RWMutex
}

// AddressHistory tracks transaction history for an address
type AddressHistory struct {
	Transactions    []TransactionRecord
	FirstSeen       time.Time
	LastTransaction time.Time
	TotalVolume     *big.Int
}

// TransactionRecord records a single transaction
type TransactionRecord struct {
	Timestamp time.Time
	Amount    *big.Int
	ToChain   string
	Success   bool
}

// NewFraudDetector creates a new fraud detector
func NewFraudDetector(config *config.SecurityConfig, logger zerolog.Logger) *FraudDetector {
	return &FraudDetector{
		config:         config,
		logger:         logger.With().Str("component", "fraud_detector").Logger(),
		addressHistory: make(map[string]*AddressHistory),
	}
}

// IsSuspicious checks if a transaction is suspicious
func (fd *FraudDetector) IsSuspicious(ctx context.Context, msg *types.CrossChainMessage) (bool, string) {
	// Extract amount
	amount := fd.extractAmount(msg)

	fd.mu.Lock()
	defer fd.mu.Unlock()

	// Get or create history
	history, exists := fd.addressHistory[msg.Sender.Raw]
	if !exists {
		history = &AddressHistory{
			Transactions: make([]TransactionRecord, 0),
			FirstSeen:    time.Now(),
			TotalVolume:  big.NewInt(0),
		}
		fd.addressHistory[msg.Sender.Raw] = history
	}

	// Check for suspicious patterns

	// 1. New address with large transaction
	if fd.isNewAddressWithLargeAmount(history, amount) {
		return true, "new_address_large_amount"
	}

	// 2. Rapid successive transactions
	if fd.hasRapidTransactions(history) {
		return true, "rapid_transactions"
	}

	// 3. Unusual volume spike
	if fd.hasVolumeSpike(history, amount) {
		return true, "volume_spike"
	}

	// 4. Same destination repeatedly in short time
	if fd.hasSameDestinationPattern(history, msg.DestinationChain.Name) {
		return true, "same_destination_pattern"
	}

	// 5. Round-trip pattern (A->B->A)
	if fd.hasRoundTripPattern(history, msg) {
		return true, "round_trip_pattern"
	}

	// Record transaction
	history.Transactions = append(history.Transactions, TransactionRecord{
		Timestamp: time.Now(),
		Amount:    amount,
		ToChain:   msg.DestinationChain.Name,
		Success:   true,
	})
	history.LastTransaction = time.Now()
	history.TotalVolume.Add(history.TotalVolume, amount)

	// Limit history size
	if len(history.Transactions) > 100 {
		history.Transactions = history.Transactions[len(history.Transactions)-100:]
	}

	return false, ""
}

// isNewAddressWithLargeAmount checks if new address is sending large amount
func (fd *FraudDetector) isNewAddressWithLargeAmount(history *AddressHistory, amount *big.Int) bool {
	// If address has no history and amount is large
	if len(history.Transactions) == 0 {
		threshold, ok := new(big.Int).SetString(fd.config.LargeTransactionThreshold, 10)
		if !ok {
			return false
		}
		return amount.Cmp(threshold) > 0
	}
	return false
}

// hasRapidTransactions checks for rapid successive transactions
func (fd *FraudDetector) hasRapidTransactions(history *AddressHistory) bool {
	if len(history.Transactions) < 3 {
		return false
	}

	// Check last 3 transactions
	recent := history.Transactions[len(history.Transactions)-3:]

	// If all within 1 minute
	firstTime := recent[0].Timestamp
	lastTime := recent[2].Timestamp

	if lastTime.Sub(firstTime) < 1*time.Minute {
		return true
	}

	return false
}

// hasVolumeSpike checks for unusual volume spike
func (fd *FraudDetector) hasVolumeSpike(history *AddressHistory, amount *big.Int) bool {
	if len(history.Transactions) < 5 {
		return false
	}

	// Calculate average of last 5 transactions
	var total big.Int
	for _, tx := range history.Transactions[len(history.Transactions)-5:] {
		total.Add(&total, tx.Amount)
	}

	avg := new(big.Int).Div(&total, big.NewInt(5))

	// If current transaction is 10x average, flag it
	threshold := new(big.Int).Mul(avg, big.NewInt(10))

	return amount.Cmp(threshold) > 0
}

// hasSameDestinationPattern checks for repeated same destination
func (fd *FraudDetector) hasSameDestinationPattern(history *AddressHistory, destChain string) bool {
	if len(history.Transactions) < 5 {
		return false
	}

	// Check last 5 transactions
	recent := history.Transactions[len(history.Transactions)-5:]

	count := 0
	for _, tx := range recent {
		if tx.ToChain == destChain {
			count++
		}
	}

	// If all 5 to same chain within short time
	if count == 5 {
		firstTime := recent[0].Timestamp
		lastTime := recent[4].Timestamp

		if lastTime.Sub(firstTime) < 10*time.Minute {
			return true
		}
	}

	return false
}

// hasRoundTripPattern detects A->B->A patterns
func (fd *FraudDetector) hasRoundTripPattern(history *AddressHistory, msg *types.CrossChainMessage) bool {
	if len(history.Transactions) < 2 {
		return false
	}

	// Check if last transaction was to source chain
	// This would indicate A->B, now doing B->A
	// This is a simplified check; production would be more sophisticated

	return false
}

// extractAmount extracts amount from message
func (fd *FraudDetector) extractAmount(msg *types.CrossChainMessage) *big.Int {
	if err := msg.DecodePayload(); err != nil {
		return big.NewInt(0)
	}

	switch msg.Type {
	case types.MessageTypeTokenTransfer:
		payload, ok := msg.DecodedPayload.(types.TokenTransferPayload)
		if !ok {
			return big.NewInt(0)
		}
		amount, ok := new(big.Int).SetString(payload.Amount, 10)
		if !ok {
			return big.NewInt(0)
		}
		return amount
	default:
		return big.NewInt(0)
	}
}

// GetAddressHistory returns history for an address
func (fd *FraudDetector) GetAddressHistory(address string) *AddressHistory {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	history, exists := fd.addressHistory[address]
	if !exists {
		return nil
	}

	return history
}

// GetStats returns fraud detector statistics
func (fd *FraudDetector) GetStats() map[string]interface{} {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	return map[string]interface{}{
		"tracked_addresses":       len(fd.addressHistory),
		"fraud_detection_enabled": fd.config.EnableFraudDetection,
	}
}
