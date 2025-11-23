package security

import (
	"context"
	"fmt"
	"math/big"

	"github.com/EmekaIwuagwu/articium-hub/internal/config"
	"github.com/EmekaIwuagwu/articium-hub/internal/monitoring"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// Validator validates cross-chain messages based on security rules
type Validator struct {
	config *config.SecurityConfig
	env    types.Environment
	logger zerolog.Logger

	// Rate limiting
	rateLimiter *RateLimiter

	// Fraud detection
	fraudDetector *FraudDetector

	// Emergency pause state
	isPaused bool
}

// NewValidator creates a new security validator
func NewValidator(
	securityConfig *config.SecurityConfig,
	env types.Environment,
	logger zerolog.Logger,
) *Validator {
	return &Validator{
		config:        securityConfig,
		env:           env,
		logger:        logger.With().Str("component", "security").Logger(),
		rateLimiter:   NewRateLimiter(securityConfig, logger),
		fraudDetector: NewFraudDetector(securityConfig, logger),
		isPaused:      false,
	}
}

// ValidateMessage performs comprehensive security validation on a message
func (v *Validator) ValidateMessage(ctx context.Context, msg *types.CrossChainMessage) error {
	// Check if bridge is paused
	if v.isPaused {
		return fmt.Errorf("bridge is currently paused")
	}

	// Parse amount from payload
	amount, err := v.extractAmount(msg)
	if err != nil {
		return fmt.Errorf("failed to extract amount: %w", err)
	}

	// Validate transaction amount limits
	if err := v.validateTransactionLimit(amount); err != nil {
		v.logger.Warn().
			Str("amount", amount.String()).
			Err(err).
			Msg("Transaction limit exceeded")
		return err
	}

	// Check daily volume limit
	if err := v.validateDailyVolumeLimit(ctx, amount); err != nil {
		v.logger.Warn().
			Str("amount", amount.String()).
			Err(err).
			Msg("Daily volume limit exceeded")
		return err
	}

	// Check rate limits
	if err := v.rateLimiter.CheckRateLimit(ctx, msg.Sender.Raw); err != nil {
		v.logger.Warn().
			Str("sender", msg.Sender.Raw).
			Err(err).
			Msg("Rate limit exceeded")
		monitoring.RecordRateLimitExceeded(msg.SourceChain.Name, msg.Sender.Raw)
		return err
	}

	// Fraud detection (if enabled)
	if v.config.EnableFraudDetection {
		if suspicious, reason := v.fraudDetector.IsSuspicious(ctx, msg); suspicious {
			v.logger.Warn().
				Str("message_id", msg.ID).
				Str("reason", reason).
				Msg("Suspicious transaction detected")
			monitoring.RecordSuspiciousTransaction(reason, msg.SourceChain.Name)
			return fmt.Errorf("transaction flagged as suspicious: %s", reason)
		}
	}

	// Validate signatures count
	if len(msg.ValidatorSignatures) < msg.RequiredSignatures {
		return fmt.Errorf("insufficient signatures: got %d, need %d",
			len(msg.ValidatorSignatures), msg.RequiredSignatures)
	}

	// Alert on large transactions
	if v.shouldAlertOnLargeTransaction(amount) {
		v.alertLargeTransaction(msg, amount)
	}

	v.logger.Debug().
		Str("message_id", msg.ID).
		Msg("Message passed security validation")

	return nil
}

// validateTransactionLimit validates transaction amount against max limit
func (v *Validator) validateTransactionLimit(amount *big.Int) error {
	maxAmount, ok := new(big.Int).SetString(v.config.MaxTransactionAmount, 10)
	if !ok {
		return fmt.Errorf("invalid max transaction amount configuration")
	}

	if amount.Cmp(maxAmount) > 0 {
		return fmt.Errorf("amount exceeds maximum: %s > %s", amount.String(), maxAmount.String())
	}

	return nil
}

// validateDailyVolumeLimit validates against daily volume limit
func (v *Validator) validateDailyVolumeLimit(ctx context.Context, amount *big.Int) error {
	// TODO: Implement daily volume tracking in database
	// For now, just check against the limit
	_, ok := new(big.Int).SetString(v.config.DailyVolumeLimit, 10)
	if !ok {
		return fmt.Errorf("invalid daily volume limit configuration")
	}

	// This would query the database for today's volume
	// todayVolume := queryTodayVolume(ctx)
	// if todayVolume + amount > dailyLimit { return error }

	return nil
}

// shouldAlertOnLargeTransaction checks if transaction is large enough to alert
func (v *Validator) shouldAlertOnLargeTransaction(amount *big.Int) bool {
	threshold, ok := new(big.Int).SetString(v.config.LargeTransactionThreshold, 10)
	if !ok {
		return false
	}

	return amount.Cmp(threshold) >= 0
}

// alertLargeTransaction sends alert for large transaction
func (v *Validator) alertLargeTransaction(msg *types.CrossChainMessage, amount *big.Int) {
	v.logger.Warn().
		Str("message_id", msg.ID).
		Str("amount", amount.String()).
		Str("source", msg.SourceChain.Name).
		Str("dest", msg.DestinationChain.Name).
		Msg("ALERT: Large transaction detected")

	// TODO: Send webhook notification if configured
	if v.config.AlertingWebhook != "" {
		// Send alert to webhook
	}
}

// extractAmount extracts the amount from message payload
func (v *Validator) extractAmount(msg *types.CrossChainMessage) (*big.Int, error) {
	if err := msg.DecodePayload(); err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	switch msg.Type {
	case types.MessageTypeTokenTransfer:
		payload, ok := msg.DecodedPayload.(types.TokenTransferPayload)
		if !ok {
			return nil, fmt.Errorf("invalid token transfer payload")
		}
		amount, ok := new(big.Int).SetString(payload.Amount, 10)
		if !ok {
			return nil, fmt.Errorf("invalid amount format: %s", payload.Amount)
		}
		return amount, nil

	case types.MessageTypeNFTTransfer:
		// NFTs have no amount, use 0
		return big.NewInt(0), nil

	default:
		return big.NewInt(0), nil
	}
}

// SetPaused sets the emergency pause state
func (v *Validator) SetPaused(paused bool) {
	v.isPaused = paused
	if paused {
		monitoring.EmergencyPauseActivations.Inc()
		v.logger.Warn().Msg("EMERGENCY PAUSE ACTIVATED")
	} else {
		v.logger.Info().Msg("Emergency pause deactivated")
	}
}

// IsPaused returns the current pause state
func (v *Validator) IsPaused() bool {
	return v.isPaused
}

// GetEnvironment returns the current environment
func (v *Validator) GetEnvironment() types.Environment {
	return v.env
}

// GetRequiredSignatures returns the required number of signatures
func (v *Validator) GetRequiredSignatures() int {
	return v.config.RequiredSignatures
}

// IsValidator checks if an address is a valid validator
func (v *Validator) IsValidator(address string) bool {
	for _, validatorAddr := range v.config.ValidatorAddresses {
		if validatorAddr == address {
			return true
		}
	}
	return false
}

// GetValidators returns all validator addresses
func (v *Validator) GetValidators() []string {
	return v.config.ValidatorAddresses
}
