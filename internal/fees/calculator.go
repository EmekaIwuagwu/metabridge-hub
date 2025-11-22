package fees

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/rs/zerolog"
)

// Calculator handles fee calculations for cross-chain transfers
type Calculator struct {
	config  *config.Config
	clients map[string]types.UniversalClient
	logger  zerolog.Logger
}

// FeeBreakdown represents the detailed breakdown of bridge fees
type FeeBreakdown struct {
	// Base protocol fee (in USD)
	ProtocolFee *big.Int `json:"protocol_fee"`

	// Source chain gas cost (in source chain native token)
	SourceGasFee *big.Int `json:"source_gas_fee"`

	// Destination chain gas cost (in dest chain native token, converted to USD)
	DestGasFee *big.Int `json:"dest_gas_fee"`

	// Relayer fee for processing (in USD)
	RelayerFee *big.Int `json:"relayer_fee"`

	// Validator signature fees (in USD)
	ValidatorFee *big.Int `json:"validator_fee"`

	// Liquidity provider fee (if using liquidity pools)
	LiquidityFee *big.Int `json:"liquidity_fee"`

	// Total fee in USD
	TotalFeeUSD *big.Int `json:"total_fee_usd"`

	// Total fee in source token
	TotalFeeSourceToken *big.Int `json:"total_fee_source_token"`

	// Exchange rates used
	SourceTokenUSDRate float64 `json:"source_token_usd_rate"`
	DestTokenUSDRate   float64 `json:"dest_token_usd_rate"`
}

// FeeEstimateRequest contains parameters for fee estimation
type FeeEstimateRequest struct {
	SourceChain  string
	DestChain    string
	TokenAddress string
	Amount       *big.Int
	MessageType  types.MessageType
	UseBatching  bool
	Priority     string // "low", "normal", "high"
	IsMultiHop   bool
	HopCount     int
}

// NewCalculator creates a new fee calculator
func NewCalculator(
	cfg *config.Config,
	clients map[string]types.UniversalClient,
	logger zerolog.Logger,
) *Calculator {
	return &Calculator{
		config:  cfg,
		clients: clients,
		logger:  logger.With().Str("component", "fee-calculator").Logger(),
	}
}

// CalculateFees computes the total fees for a bridge operation
func (c *Calculator) CalculateFees(ctx context.Context, req *FeeEstimateRequest) (*FeeBreakdown, error) {
	c.logger.Debug().
		Str("source", req.SourceChain).
		Str("dest", req.DestChain).
		Str("amount", req.Amount.String()).
		Msg("Calculating bridge fees")

	breakdown := &FeeBreakdown{
		ProtocolFee:  big.NewInt(0),
		SourceGasFee: big.NewInt(0),
		DestGasFee:   big.NewInt(0),
		RelayerFee:   big.NewInt(0),
		ValidatorFee: big.NewInt(0),
		LiquidityFee: big.NewInt(0),
	}

	// 1. Calculate base protocol fee (0.1% of transaction value)
	protocolFee := c.calculateProtocolFee(req.Amount)
	breakdown.ProtocolFee = protocolFee

	// 2. Estimate gas costs on source chain
	sourceGas, err := c.estimateSourceChainGas(ctx, req)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Failed to estimate source gas, using default")
		sourceGas = c.getDefaultGasCost(req.SourceChain)
	}
	breakdown.SourceGasFee = sourceGas

	// 3. Estimate gas costs on destination chain
	destGas, err := c.estimateDestChainGas(ctx, req)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Failed to estimate dest gas, using default")
		destGas = c.getDefaultGasCost(req.DestChain)
	}
	breakdown.DestGasFee = destGas

	// 4. Calculate relayer fee (covers infrastructure costs + margin)
	relayerFee := c.calculateRelayerFee(req, destGas)
	breakdown.RelayerFee = relayerFee

	// 5. Calculate validator fee (signature verification costs)
	validatorFee := c.calculateValidatorFee(req)
	breakdown.ValidatorFee = validatorFee

	// 6. Calculate liquidity fee (if applicable)
	if c.requiresLiquidity(req) {
		liquidityFee := c.calculateLiquidityFee(req.Amount)
		breakdown.LiquidityFee = liquidityFee
	}

	// 7. Apply batching discount if enabled
	if req.UseBatching {
		c.applyBatchingDiscount(breakdown)
	}

	// 8. Apply priority multiplier
	c.applyPriorityMultiplier(breakdown, req.Priority)

	// 9. Apply multi-hop multiplier
	if req.IsMultiHop {
		c.applyMultiHopMultiplier(breakdown, req.HopCount)
	}

	// 10. Get exchange rates and calculate totals
	sourceRate, destRate := c.getExchangeRates(req.SourceChain, req.DestChain)
	breakdown.SourceTokenUSDRate = sourceRate
	breakdown.DestTokenUSDRate = destRate

	// Sum all USD fees
	totalUSD := big.NewInt(0)
	totalUSD.Add(totalUSD, breakdown.ProtocolFee)
	totalUSD.Add(totalUSD, breakdown.RelayerFee)
	totalUSD.Add(totalUSD, breakdown.ValidatorFee)
	totalUSD.Add(totalUSD, breakdown.LiquidityFee)

	// Convert gas fees to USD
	sourceGasUSD := c.convertToUSD(breakdown.SourceGasFee, sourceRate)
	destGasUSD := c.convertToUSD(breakdown.DestGasFee, destRate)
	totalUSD.Add(totalUSD, sourceGasUSD)
	totalUSD.Add(totalUSD, destGasUSD)

	breakdown.TotalFeeUSD = totalUSD

	// Convert total USD fee to source token
	breakdown.TotalFeeSourceToken = c.convertFromUSD(totalUSD, sourceRate)

	c.logger.Info().
		Str("total_usd", totalUSD.String()).
		Str("total_source_token", breakdown.TotalFeeSourceToken.String()).
		Msg("Fee calculation complete")

	return breakdown, nil
}

// calculateProtocolFee computes the base protocol fee (0.1% of value)
func (c *Calculator) calculateProtocolFee(amount *big.Int) *big.Int {
	// 0.1% = 1/1000
	fee := new(big.Int).Div(amount, big.NewInt(1000))

	// Minimum fee: $0.10 (in wei: 100000000000000000 wei = 0.1 USD)
	minFee := big.NewInt(100000000000000000)
	if fee.Cmp(minFee) < 0 {
		fee = minFee
	}

	return fee
}

// estimateSourceChainGas estimates gas cost on source chain
func (c *Calculator) estimateSourceChainGas(ctx context.Context, req *FeeEstimateRequest) (*big.Int, error) {
	client, ok := c.clients[req.SourceChain]
	if !ok {
		return nil, fmt.Errorf("no client for chain %s", req.SourceChain)
	}

	chainCfg, err := c.config.GetChainConfig(req.SourceChain)
	if err != nil {
		return nil, err
	}

	var gasLimit uint64
	switch req.MessageType {
	case types.MessageTypeTokenTransfer:
		gasLimit = 150000 // Lock/burn token
	case types.MessageTypeNFTTransfer:
		gasLimit = 200000 // Lock/burn NFT
	default:
		gasLimit = 100000
	}

	// Get current gas price
	gasPrice, err := client.EstimateGas(ctx)
	if err != nil {
		// Use config max gas price as fallback
		maxGasPrice, _ := new(big.Int).SetString(chainCfg.MaxGasPrice, 10)
		gasPrice = maxGasPrice
	}

	// Total gas cost = gasLimit * gasPrice
	gasCost := new(big.Int).Mul(big.NewInt(int64(gasLimit)), gasPrice)

	return gasCost, nil
}

// estimateDestChainGas estimates gas cost on destination chain
func (c *Calculator) estimateDestChainGas(ctx context.Context, req *FeeEstimateRequest) (*big.Int, error) {
	client, ok := c.clients[req.DestChain]
	if !ok {
		return nil, fmt.Errorf("no client for chain %s", req.DestChain)
	}

	chainCfg, err := c.config.GetChainConfig(req.DestChain)
	if err != nil {
		return nil, err
	}

	var gasLimit uint64
	switch req.MessageType {
	case types.MessageTypeTokenTransfer:
		gasLimit = 180000 // Release/mint token + signatures verification
	case types.MessageTypeNFTTransfer:
		gasLimit = 250000 // Release/mint NFT + signatures
	default:
		gasLimit = 150000
	}

	// Get current gas price
	gasPrice, err := client.EstimateGas(ctx)
	if err != nil {
		maxGasPrice, _ := new(big.Int).SetString(chainCfg.MaxGasPrice, 10)
		gasPrice = maxGasPrice
	}

	// Total gas cost
	gasCost := new(big.Int).Mul(big.NewInt(int64(gasLimit)), gasPrice)

	return gasCost, nil
}

// calculateRelayerFee calculates the relayer service fee
func (c *Calculator) calculateRelayerFee(req *FeeEstimateRequest, destGas *big.Int) *big.Int {
	// Base relayer fee: $0.50 in wei
	baseFee := big.NewInt(500000000000000000) // 0.5 USD

	// Add 20% margin on destination gas to cover relayer operational costs
	gasMargin := new(big.Int).Div(destGas, big.NewInt(5)) // 20%

	totalFee := new(big.Int).Add(baseFee, gasMargin)

	return totalFee
}

// calculateValidatorFee calculates the fee paid to validators for signatures
func (c *Calculator) calculateValidatorFee(req *FeeEstimateRequest) *big.Int {
	// Validator fee: $0.05 per signature
	requiredSigs := c.config.Security.RequiredSignatures
	feePerSig := big.NewInt(50000000000000000) // 0.05 USD in wei

	totalFee := new(big.Int).Mul(feePerSig, big.NewInt(int64(requiredSigs)))

	return totalFee
}

// calculateLiquidityFee calculates the liquidity provider fee
func (c *Calculator) calculateLiquidityFee(amount *big.Int) *big.Int {
	// Liquidity fee: 0.05% of transaction amount
	fee := new(big.Int).Div(amount, big.NewInt(2000))

	return fee
}

// requiresLiquidity checks if the route requires liquidity pools
func (c *Calculator) requiresLiquidity(req *FeeEstimateRequest) bool {
	// Currently not using liquidity pools, but this would check
	// if the route goes through AMM pools for faster settlement
	return false
}

// applyBatchingDiscount applies a discount for batched transactions
func (c *Calculator) applyBatchingDiscount(breakdown *FeeBreakdown) {
	// 30% discount on gas fees when batching
	discount := big.NewInt(30)
	hundred := big.NewInt(100)

	// Reduce destination gas fee
	reduction := new(big.Int).Mul(breakdown.DestGasFee, discount)
	reduction.Div(reduction, hundred)
	breakdown.DestGasFee.Sub(breakdown.DestGasFee, reduction)

	// Reduce relayer fee
	reduction = new(big.Int).Mul(breakdown.RelayerFee, discount)
	reduction.Div(reduction, hundred)
	breakdown.RelayerFee.Sub(breakdown.RelayerFee, reduction)
}

// applyPriorityMultiplier applies fee multiplier based on priority
func (c *Calculator) applyPriorityMultiplier(breakdown *FeeBreakdown, priority string) {
	var multiplier int64

	switch priority {
	case "high":
		multiplier = 150 // 1.5x fees for fast processing
	case "low":
		multiplier = 80 // 0.8x fees for slower processing
	default: // "normal"
		return // 1.0x (no change)
	}

	hundred := big.NewInt(100)

	// Apply to all fee components
	breakdown.RelayerFee = new(big.Int).Mul(breakdown.RelayerFee, big.NewInt(multiplier))
	breakdown.RelayerFee.Div(breakdown.RelayerFee, hundred)

	breakdown.ValidatorFee = new(big.Int).Mul(breakdown.ValidatorFee, big.NewInt(multiplier))
	breakdown.ValidatorFee.Div(breakdown.ValidatorFee, hundred)
}

// applyMultiHopMultiplier applies fee multiplier for multi-hop routes
func (c *Calculator) applyMultiHopMultiplier(breakdown *FeeBreakdown, hopCount int) {
	// Each additional hop adds fees
	// First hop is already accounted for, so multiply by hopCount
	multiplier := big.NewInt(int64(hopCount))

	breakdown.RelayerFee = new(big.Int).Mul(breakdown.RelayerFee, multiplier)
	breakdown.ValidatorFee = new(big.Int).Mul(breakdown.ValidatorFee, multiplier)
	breakdown.DestGasFee = new(big.Int).Mul(breakdown.DestGasFee, multiplier)
}

// getDefaultGasCost returns default gas cost when estimation fails
func (c *Calculator) getDefaultGasCost(chainName string) *big.Int {
	defaults := map[string]int64{
		"polygon-amoy":     5000000000000000,       // 0.005 POL
		"bnb-testnet":      10000000000000000,      // 0.01 BNB
		"avalanche-fuji":   20000000000000000,      // 0.02 AVAX
		"ethereum-sepolia": 50000000000000000,      // 0.05 ETH
		"solana-devnet":    5000000,                // 0.000005 SOL
		"near-testnet":     1000000000000000000000, // 0.001 NEAR
	}

	if cost, ok := defaults[chainName]; ok {
		return big.NewInt(cost)
	}

	return big.NewInt(10000000000000000) // Default: 0.01 token
}

// getExchangeRates returns USD exchange rates for tokens
func (c *Calculator) getExchangeRates(sourceChain, destChain string) (float64, float64) {
	// In production, fetch from oracle like Chainlink or API
	// For now, use approximate rates
	rates := map[string]float64{
		"polygon-amoy":     0.50,   // POL ~ $0.50
		"bnb-testnet":      300.0,  // BNB ~ $300
		"avalanche-fuji":   20.0,   // AVAX ~ $20
		"ethereum-sepolia": 2000.0, // ETH ~ $2000
		"solana-devnet":    100.0,  // SOL ~ $100
		"near-testnet":     3.0,    // NEAR ~ $3
	}

	sourceRate := rates[sourceChain]
	destRate := rates[destChain]

	if sourceRate == 0 {
		sourceRate = 1.0
	}
	if destRate == 0 {
		destRate = 1.0
	}

	return sourceRate, destRate
}

// convertToUSD converts token amount to USD
func (c *Calculator) convertToUSD(amount *big.Int, rate float64) *big.Int {
	// Convert to float, multiply by rate, convert back
	amountFloat := new(big.Float).SetInt(amount)
	rateFloat := big.NewFloat(rate)

	result := new(big.Float).Mul(amountFloat, rateFloat)

	// Convert back to int
	usdAmount, _ := result.Int(nil)

	return usdAmount
}

// convertFromUSD converts USD amount to token amount
func (c *Calculator) convertFromUSD(usdAmount *big.Int, rate float64) *big.Int {
	// Divide USD by rate to get token amount
	usdFloat := new(big.Float).SetInt(usdAmount)
	rateFloat := big.NewFloat(rate)

	result := new(big.Float).Quo(usdFloat, rateFloat)

	// Convert back to int
	tokenAmount, _ := result.Int(nil)

	return tokenAmount
}

// GetFeeHistory returns historical fee data
func (c *Calculator) GetFeeHistory(ctx context.Context, chainName string, duration time.Duration) ([]FeeDataPoint, error) {
	// TODO: Implement fee history tracking
	return nil, fmt.Errorf("fee history not yet implemented")
}

// FeeDataPoint represents a historical fee data point
type FeeDataPoint struct {
	Timestamp time.Time
	AvgFee    *big.Int
	MinFee    *big.Int
	MaxFee    *big.Int
}
