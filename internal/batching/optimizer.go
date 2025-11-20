package batching

import (
	"fmt"
	"math/big"

	"github.com/rs/zerolog"
)

// Optimizer optimizes batch submission timing and calculates savings
type Optimizer struct {
	config *BatchConfig
	logger zerolog.Logger
}

// NewOptimizer creates a new batch optimizer
func NewOptimizer(config *BatchConfig, logger zerolog.Logger) *Optimizer {
	return &Optimizer{
		config: config,
		logger: logger.With().Str("component", "batch-optimizer").Logger(),
	}
}

// CalculateGasSavings calculates the gas cost savings from batching
func (o *Optimizer) CalculateGasSavings(batch *Batch) (*big.Int, error) {
	if len(batch.Messages) == 0 {
		return big.NewInt(0), nil
	}

	// Estimated gas costs (in wei)
	const (
		// Individual transaction costs
		baseGasCostPerTx    = 21000 // Base transaction cost
		callDataCostPerByte = 16    // Calldata cost
		storageSlotCost     = 20000 // SSTORE cost
		signatureVerifyCost = 3000  // ECRECOVER
		avgTxSize           = 500   // Average tx size in bytes

		// Batch transaction costs
		batchBaseCost       = 21000 // Base cost for batch tx
		batchCallDataPerMsg = 200   // Calldata per message in batch
		merkleVerifyCost    = 5000  // Merkle proof verification
	)

	messageCount := int64(len(batch.Messages))

	// Individual cost: Each message as separate transaction
	individualCostPerMsg := int64(
		baseGasCostPerTx +
			(callDataCostPerByte * avgTxSize) +
			storageSlotCost +
			signatureVerifyCost,
	)
	totalIndividualCost := big.NewInt(individualCostPerMsg * messageCount)

	// Batch cost: All messages in one transaction
	batchCostPerMsg := int64(
		batchCallDataPerMsg +
			merkleVerifyCost,
	)
	totalBatchCost := big.NewInt(batchBaseCost + (batchCostPerMsg * messageCount))

	// Calculate savings
	savings := new(big.Int).Sub(totalIndividualCost, totalBatchCost)

	// Apply gas price (assume 20 gwei for estimation)
	gasPrice := big.NewInt(20000000000) // 20 gwei
	savingsInWei := new(big.Int).Mul(savings, gasPrice)

	o.logger.Debug().
		Str("batch_id", batch.ID).
		Int("message_count", len(batch.Messages)).
		Str("individual_cost", totalIndividualCost.String()).
		Str("batch_cost", totalBatchCost.String()).
		Str("savings_gas", savings.String()).
		Str("savings_wei", savingsInWei.String()).
		Msg("Calculated gas savings")

	return savingsInWei, nil
}

// CalculateSavingsPercentage calculates the percentage of gas saved
func (o *Optimizer) CalculateSavingsPercentage(batch *Batch) (float64, error) {
	savings, err := o.CalculateGasSavings(batch)
	if err != nil {
		return 0, err
	}

	if len(batch.Messages) == 0 {
		return 0, nil
	}

	// Calculate individual cost
	const baseGasCostPerTx = 21000
	const callDataCostPerByte = 16
	const avgTxSize = 500
	const storageSlotCost = 20000
	const signatureVerifyCost = 3000

	individualCostPerMsg := int64(
		baseGasCostPerTx +
			(callDataCostPerByte * avgTxSize) +
			storageSlotCost +
			signatureVerifyCost,
	)
	totalIndividualCost := big.NewInt(individualCostPerMsg * int64(len(batch.Messages)))

	// Calculate percentage
	gasPrice := big.NewInt(20000000000)
	totalIndividualWei := new(big.Int).Mul(totalIndividualCost, gasPrice)

	savingsFloat := new(big.Float).SetInt(savings)
	totalFloat := new(big.Float).SetInt(totalIndividualWei)

	percentage, _ := new(big.Float).Quo(savingsFloat, totalFloat).Float64()
	percentage *= 100

	return percentage, nil
}

// ShouldSubmitNow determines if batch should be submitted now
func (o *Optimizer) ShouldSubmitNow(batch *Batch) (bool, string) {
	// Check if batch is full
	if batch.IsFull(o.config.MaxBatchSize) {
		return true, "batch is full"
	}

	// Check minimum size
	if len(batch.Messages) < o.config.MinBatchSize {
		return false, fmt.Sprintf("batch size %d below minimum %d", len(batch.Messages), o.config.MinBatchSize)
	}

	// Check timeout
	if batch.IsReady(o.config) {
		return true, "batch timeout reached"
	}

	// Check cost savings threshold
	if o.config.CostSavingsThreshold != nil && batch.GasCostSaved != nil {
		if batch.GasCostSaved.Cmp(o.config.CostSavingsThreshold) >= 0 {
			return true, "cost savings threshold met"
		}
	}

	return false, "waiting for more messages or timeout"
}

// EstimateOptimalBatchSize estimates the optimal batch size for maximum efficiency
func (o *Optimizer) EstimateOptimalBatchSize() int {
	// Based on gas costs, optimal batch size is typically 20-50 messages
	// This balances gas savings with transaction confirmation time
	return 30
}

// GetBatchEfficiency calculates efficiency metrics for a batch
func (o *Optimizer) GetBatchEfficiency(batch *Batch) (*BatchEfficiency, error) {
	savings, err := o.CalculateGasSavings(batch)
	if err != nil {
		return nil, err
	}

	percentage, err := o.CalculateSavingsPercentage(batch)
	if err != nil {
		return nil, err
	}

	efficiency := &BatchEfficiency{
		BatchID:           batch.ID,
		MessageCount:      len(batch.Messages),
		TotalGasSaved:     savings,
		SavingsPercentage: percentage,
		CostPerMessage:    o.calculateCostPerMessage(batch, savings),
	}

	return efficiency, nil
}

// calculateCostPerMessage calculates average cost per message in batch
func (o *Optimizer) calculateCostPerMessage(batch *Batch, totalSavings *big.Int) *big.Int {
	if len(batch.Messages) == 0 {
		return big.NewInt(0)
	}

	// Batch cost = individual cost - savings
	const baseGasCostPerTx = 21000
	const callDataCostPerByte = 16
	const avgTxSize = 500
	const storageSlotCost = 20000
	const signatureVerifyCost = 3000

	individualCostPerMsg := int64(
		baseGasCostPerTx +
			(callDataCostPerByte * avgTxSize) +
			storageSlotCost +
			signatureVerifyCost,
	)

	gasPrice := big.NewInt(20000000000)
	individualCostWei := new(big.Int).Mul(big.NewInt(individualCostPerMsg), gasPrice)

	savingsPerMsg := new(big.Int).Div(totalSavings, big.NewInt(int64(len(batch.Messages))))
	costPerMsg := new(big.Int).Sub(individualCostWei, savingsPerMsg)

	return costPerMsg
}

// BatchEfficiency holds efficiency metrics for a batch
type BatchEfficiency struct {
	BatchID           string
	MessageCount      int
	TotalGasSaved     *big.Int
	SavingsPercentage float64
	CostPerMessage    *big.Int
}

// FormatSavings returns a human-readable savings summary
func (e *BatchEfficiency) FormatSavings() string {
	// Convert to ETH
	wei := new(big.Float).SetInt(e.TotalGasSaved)
	eth := new(big.Float).Quo(wei, big.NewFloat(1e18))

	return fmt.Sprintf("%.4f ETH (%.1f%% savings, %d messages)",
		eth,
		e.SavingsPercentage,
		e.MessageCount,
	)
}
