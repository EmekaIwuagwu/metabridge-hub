package solana

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/blockchain/solana"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/monitoring"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	solanago "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/rs/zerolog"
)

// Listener listens for events on Solana blockchain
type Listener struct {
	client          *solana.Client
	config          *types.ChainConfig
	logger          zerolog.Logger
	eventChan       chan *types.CrossChainMessage
	stopChan        chan struct{}
	lastSlot        uint64
	bridgeProgramID solanago.PublicKey
}

// NewListener creates a new Solana event listener
func NewListener(
	client *solana.Client,
	config *types.ChainConfig,
	logger zerolog.Logger,
) (*Listener, error) {
	// Check BridgeProgram for Solana chains
	programID := config.BridgeProgram
	if programID == "" {
		// Fallback to BridgeContract for backward compatibility
		programID = config.BridgeContract
	}
	if programID == "" {
		return nil, fmt.Errorf("bridge program ID not configured")
	}

	bridgeProgramID, err := solanago.PublicKeyFromBase58(programID)
	if err != nil {
		return nil, fmt.Errorf("invalid bridge program ID: %w", err)
	}

	return &Listener{
		client:          client,
		config:          config,
		logger:          logger.With().Str("chain", config.Name).Str("component", "listener").Logger(),
		eventChan:       make(chan *types.CrossChainMessage, 100),
		stopChan:        make(chan struct{}),
		lastSlot:        config.StartBlock,
		bridgeProgramID: bridgeProgramID,
	}, nil
}

// Start starts the listener
func (l *Listener) Start(ctx context.Context) error {
	l.logger.Info().
		Uint64("start_slot", l.lastSlot).
		Str("program", l.bridgeProgramID.String()).
		Msg("Starting Solana listener")

	// Start listening in a goroutine
	go l.listen(ctx)

	return nil
}

// Stop stops the listener
func (l *Listener) Stop() error {
	l.logger.Info().Msg("Stopping Solana listener")
	close(l.stopChan)
	close(l.eventChan)
	return nil
}

// EventChan returns the channel for receiving events
func (l *Listener) EventChan() <-chan *types.CrossChainMessage {
	return l.eventChan
}

// listen is the main listening loop
func (l *Listener) listen(ctx context.Context) {
	ticker := time.NewTicker(l.config.GetPollIntervalDuration())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			l.logger.Info().Msg("Context cancelled, stopping listener")
			return
		case <-l.stopChan:
			l.logger.Info().Msg("Stop signal received")
			return
		case <-ticker.C:
			if err := l.processSlots(ctx); err != nil {
				l.logger.Error().Err(err).Msg("Error processing slots")
			}
		}
	}
}

// processSlots processes new slots
func (l *Listener) processSlots(ctx context.Context) error {
	// Get latest slot
	latestSlot, err := l.client.GetSlot(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest slot: %w", err)
	}

	// Update metrics
	monitoring.UpdateChainBlockNumber(l.config.Name, latestSlot)
	monitoring.ListenerLastBlockProcessed.WithLabelValues(l.config.Name).Set(float64(l.lastSlot))

	// Calculate safe slot (with confirmations)
	safeSlot := latestSlot
	if latestSlot > l.config.ConfirmationBlocks {
		safeSlot = latestSlot - l.config.ConfirmationBlocks
	}

	// Process slots from lastSlot to safeSlot
	if l.lastSlot > safeSlot {
		return nil // No new confirmed slots
	}

	// Process in batches
	batchSize := uint64(50) // Smaller batches for Solana
	for fromSlot := l.lastSlot; fromSlot <= safeSlot; {
		toSlot := fromSlot + batchSize - 1
		if toSlot > safeSlot {
			toSlot = safeSlot
		}

		if err := l.processSlotRange(ctx, fromSlot, toSlot); err != nil {
			l.logger.Error().
				Err(err).
				Uint64("from", fromSlot).
				Uint64("to", toSlot).
				Msg("Error processing slot range")
			return err
		}

		fromSlot = toSlot + 1
		l.lastSlot = toSlot + 1

		// Update metrics
		monitoring.ListenerBlocksProcessed.WithLabelValues(l.config.Name).Add(float64(toSlot - fromSlot + 1))
	}

	return nil
}

// processSlotRange processes a range of slots
func (l *Listener) processSlotRange(ctx context.Context, fromSlot, toSlot uint64) error {
	l.logger.Debug().
		Uint64("from", fromSlot).
		Uint64("to", toSlot).
		Msg("Processing slot range")

	// Get program accounts with filters
	accounts, err := l.client.GetProgramAccounts(ctx, l.bridgeProgramID)
	if err != nil {
		l.logger.Warn().Err(err).Msg("Failed to get program accounts")
		// Don't fail - continue to next batch
		return nil
	}

	l.logger.Debug().
		Int("accounts", len(accounts)).
		Msg("Program accounts retrieved")

	// Process each account for events
	for _, result := range accounts {
		if err := l.processAccount(ctx, result.Pubkey, result.Account); err != nil {
			l.logger.Error().
				Err(err).
				Str("account", result.Pubkey.String()).
				Msg("Error processing account")
			continue
		}
	}

	return nil
}

// processAccount processes a single program account
func (l *Listener) processAccount(ctx context.Context, pubkey solanago.PublicKey, account *rpc.Account) error {
	// Parse account data to detect lock events
	// Account data structure depends on your Solana program implementation

	// For this example, we'll assume the account data contains:
	// - Discriminator (8 bytes)
	// - Event type (1 byte)
	// - Message data

	if len(account.Data.GetBinary()) < 9 {
		return nil // Not enough data
	}

	data := account.Data.GetBinary()

	// Check discriminator (first 8 bytes)
	// This would match your Solana program's event discriminator
	discriminator := data[0:8]
	_ = discriminator // Would check if this is a lock event

	// Event type
	eventType := data[8]

	// Parse based on event type
	var msg *types.CrossChainMessage
	var err error

	switch eventType {
	case 0x01: // Token locked event
		msg, err = l.parseTokenLockedEvent(data[9:], pubkey)
	case 0x02: // NFT locked event
		msg, err = l.parseNFTLockedEvent(data[9:], pubkey)
	default:
		return nil // Unknown event type
	}

	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	if msg != nil {
		// Send message to channel
		select {
		case l.eventChan <- msg:
			l.logger.Info().
				Str("message_id", msg.ID).
				Str("type", string(msg.Type)).
				Msg("Message detected and queued")

			// Update metrics
			monitoring.ListenerEventsDetected.WithLabelValues(l.config.Name, string(msg.Type)).Inc()
		default:
			l.logger.Warn().Msg("Event channel full, message dropped")
		}
	}

	return nil
}

// parseTokenLockedEvent parses a token locked event from account data
func (l *Listener) parseTokenLockedEvent(data []byte, account solanago.PublicKey) (*types.CrossChainMessage, error) {
	// Expected data structure (simplified):
	// - message_id (32 bytes)
	// - sender (32 bytes)
	// - recipient (variable, prefixed with length)
	// - token_mint (32 bytes)
	// - amount (8 bytes)
	// - destination_chain (variable, prefixed with length)

	if len(data) < 104 { // Minimum size
		return nil, fmt.Errorf("insufficient data for token locked event")
	}

	offset := 0

	// Message ID (32 bytes)
	messageID := base64.StdEncoding.EncodeToString(data[offset : offset+32])
	offset += 32

	// Sender (32 bytes Solana public key)
	senderBytes := data[offset : offset+32]
	sender := solanago.PublicKeyFromBytes(senderBytes).String()
	offset += 32

	// Recipient (length-prefixed string)
	recipientLen := int(data[offset])
	offset++
	if offset+recipientLen > len(data) {
		return nil, fmt.Errorf("invalid recipient length")
	}
	recipient := string(data[offset : offset+recipientLen])
	offset += recipientLen

	// Token mint (32 bytes)
	tokenMintBytes := data[offset : offset+32]
	tokenMint := solanago.PublicKeyFromBytes(tokenMintBytes).String()
	offset += 32

	// Amount (8 bytes, little endian)
	amount := uint64(0)
	for i := 0; i < 8 && offset+i < len(data); i++ {
		amount |= uint64(data[offset+i]) << (i * 8)
	}
	offset += 8

	// Destination chain (length-prefixed string)
	destChainLen := int(data[offset])
	offset++
	if offset+destChainLen > len(data) {
		return nil, fmt.Errorf("invalid destination chain length")
	}
	destChain := string(data[offset : offset+destChainLen])

	// Build cross-chain message
	tokenAddr, err := types.NewAddress(tokenMint, types.ChainTypeSolana)
	if err != nil {
		return nil, fmt.Errorf("invalid token address: %w", err)
	}

	payload := types.TokenTransferPayload{
		TokenAddress:  tokenAddr,
		Amount:        fmt.Sprintf("%d", amount),
		TokenStandard: "SPL",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Build addresses
	senderAddr, err := types.NewAddress(sender, types.ChainTypeSolana)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	recipientAddr, err := types.NewAddress(recipient, types.ChainTypeSolana)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient address: %w", err)
	}

	msg := &types.CrossChainMessage{
		ID:   messageID,
		Type: types.MessageTypeTokenTransfer,
		SourceChain: types.ChainInfo{
			Name:    l.config.Name,
			Type:    types.ChainTypeSolana,
			ChainID: l.config.NetworkID,
		},
		DestinationChain: types.ChainInfo{
			Name: destChain,
		},
		Sender:    senderAddr,
		Recipient: recipientAddr,
		Payload:   payloadBytes,
		Nonce:     0, // Would extract from account
		Status:    types.MessageStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	l.logger.Info().
		Str("message_id", messageID).
		Str("sender", sender).
		Str("recipient", recipient).
		Str("token", tokenMint).
		Uint64("amount", amount).
		Str("dest_chain", destChain).
		Msg("Parsed token locked event")

	return msg, nil
}

// parseNFTLockedEvent parses an NFT locked event from account data
func (l *Listener) parseNFTLockedEvent(data []byte, account solanago.PublicKey) (*types.CrossChainMessage, error) {
	// Similar structure to token locked but for NFTs
	// Would parse Metaplex NFT metadata

	l.logger.Debug().
		Str("account", account.String()).
		Msg("Parsing NFT locked event")

	// Placeholder - full implementation would parse NFT-specific data
	return nil, fmt.Errorf("NFT locked event parsing not fully implemented")
}
