package near

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/blockchain/near"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/monitoring"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/rs/zerolog"
)

// Listener listens for events on NEAR blockchain
type Listener struct {
	client         *near.Client
	config         *types.ChainConfig
	logger         zerolog.Logger
	eventChan      chan *types.CrossChainMessage
	stopChan       chan struct{}
	lastBlock      uint64
	bridgeContract string
}

// NewListener creates a new NEAR event listener
func NewListener(
	client *near.Client,
	config *types.ChainConfig,
	logger zerolog.Logger,
) (*Listener, error) {
	if config.BridgeContract == "" {
		return nil, fmt.Errorf("bridge contract not configured")
	}

	return &Listener{
		client:         client,
		config:         config,
		logger:         logger.With().Str("chain", config.Name).Str("component", "listener").Logger(),
		eventChan:      make(chan *types.CrossChainMessage, 100),
		stopChan:       make(chan struct{}),
		lastBlock:      config.StartBlock,
		bridgeContract: config.BridgeContract,
	}, nil
}

// Start starts the listener
func (l *Listener) Start(ctx context.Context) error {
	l.logger.Info().
		Uint64("start_block", l.lastBlock).
		Str("contract", l.bridgeContract).
		Msg("Starting NEAR listener")

	// Start listening in a goroutine
	go l.listen(ctx)

	return nil
}

// Stop stops the listener
func (l *Listener) Stop() error {
	l.logger.Info().Msg("Stopping NEAR listener")
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
			if err := l.processBlocks(ctx); err != nil {
				l.logger.Error().Err(err).Msg("Error processing blocks")
			}
		}
	}
}

// processBlocks processes new blocks
func (l *Listener) processBlocks(ctx context.Context) error {
	// Get latest block
	latestBlock, err := l.client.GetLatestBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	// Update metrics
	monitoring.UpdateChainBlockNumber(l.config.Name, latestBlock)
	monitoring.ListenerLastBlockProcessed.WithLabelValues(l.config.Name).Set(float64(l.lastBlock))

	// Calculate safe block (with confirmations)
	safeBlock := latestBlock
	if latestBlock > l.config.ConfirmationBlocks {
		safeBlock = latestBlock - l.config.ConfirmationBlocks
	}

	// Process blocks from lastBlock to safeBlock
	if l.lastBlock > safeBlock {
		return nil // No new confirmed blocks
	}

	// Process in batches
	batchSize := uint64(20) // NEAR blocks can be large
	for fromBlock := l.lastBlock; fromBlock <= safeBlock; {
		toBlock := fromBlock + batchSize - 1
		if toBlock > safeBlock {
			toBlock = safeBlock
		}

		if err := l.processBlockRange(ctx, fromBlock, toBlock); err != nil {
			l.logger.Error().
				Err(err).
				Uint64("from", fromBlock).
				Uint64("to", toBlock).
				Msg("Error processing block range")
			return err
		}

		fromBlock = toBlock + 1
		l.lastBlock = toBlock + 1

		// Update metrics
		monitoring.ListenerBlocksProcessed.WithLabelValues(l.config.Name).Add(float64(toBlock - fromBlock + 1))
	}

	return nil
}

// processBlockRange processes a range of blocks
func (l *Listener) processBlockRange(ctx context.Context, fromBlock, toBlock uint64) error {
	l.logger.Debug().
		Uint64("from", fromBlock).
		Uint64("to", toBlock).
		Msg("Processing block range")

	// For each block in range
	for blockHeight := fromBlock; blockHeight <= toBlock; blockHeight++ {
		if err := l.processBlock(ctx, blockHeight); err != nil {
			l.logger.Error().
				Err(err).
				Uint64("block", blockHeight).
				Msg("Error processing block")
			continue
		}
	}

	return nil
}

// processBlock processes a single block
func (l *Listener) processBlock(ctx context.Context, blockHeight uint64) error {
	// Get block details
	block, err := l.client.GetBlockByNumber(ctx, blockHeight)
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	// Query for transaction receipts involving bridge contract
	// NEAR doesn't have event logs like EVM, so we need to query receipts
	// This is a simplified approach - in production you'd use indexer

	// Get account activity (simplified)
	// In production, use NEAR Indexer or Lake Framework
	events, err := l.queryContractEvents(ctx, blockHeight)
	if err != nil {
		l.logger.Warn().
			Err(err).
			Uint64("block", blockHeight).
			Msg("Failed to query contract events")
		return nil // Don't fail - continue to next block
	}

	// Process each event
	for _, event := range events {
		if err := l.processEvent(ctx, event); err != nil {
			l.logger.Error().
				Err(err).
				Msg("Error processing event")
			continue
		}
	}

	l.logger.Debug().
		Uint64("block", blockHeight).
		Str("hash", block.Hash).
		Int("events", len(events)).
		Msg("Block processed")

	return nil
}

// NEAREvent represents a NEAR contract event
type NEAREvent struct {
	Standard string                 `json:"standard"`
	Version  string                 `json:"version"`
	Event    string                 `json:"event"`
	Data     map[string]interface{} `json:"data"`
}

// queryContractEvents queries events from bridge contract
func (l *Listener) queryContractEvents(ctx context.Context, blockHeight uint64) ([]NEAREvent, error) {
	// In production, you would:
	// 1. Use NEAR Indexer for Explorer
	// 2. Use NEAR Lake Framework
	// 3. Query transaction outcomes for the bridge contract
	// 4. Parse execution outcomes for events

	// For this example, we'll demonstrate the event structure
	// Real implementation would use near.callRPC with proper queries

	// Placeholder - would query actual events from indexer
	events := []NEAREvent{}

	// Example of querying view function to get pending events
	// This assumes bridge contract stores events that can be queried
	result, err := l.client.ViewFunction(ctx, l.bridgeContract, "get_events", map[string]interface{}{
		"from_block": blockHeight,
		"to_block":   blockHeight,
	})

	if err != nil {
		return events, err
	}

	// Parse result
	if err := json.Unmarshal(result, &events); err != nil {
		return events, fmt.Errorf("failed to parse events: %w", err)
	}

	return events, nil
}

// processEvent processes a single contract event
func (l *Listener) processEvent(ctx context.Context, event NEAREvent) error {
	// Check if this is a bridge event
	if event.Standard != "bridge" {
		return nil
	}

	var msg *types.CrossChainMessage
	var err error

	switch event.Event {
	case "token_locked":
		msg, err = l.parseTokenLockedEvent(event.Data)
	case "nft_locked":
		msg, err = l.parseNFTLockedEvent(event.Data)
	default:
		return nil // Unknown event
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

// parseTokenLockedEvent parses a token locked event
func (l *Listener) parseTokenLockedEvent(data map[string]interface{}) (*types.CrossChainMessage, error) {
	// Extract fields from event data
	messageID, ok := data["message_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid message_id")
	}

	sender, ok := data["sender"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid sender")
	}

	recipient, ok := data["recipient"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid recipient")
	}

	token, ok := data["token"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid token")
	}

	// Amount might be string in NEAR
	var amount uint64
	switch v := data["amount"].(type) {
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %w", err)
		}
		amount = parsed
	case float64:
		amount = uint64(v)
	default:
		return nil, fmt.Errorf("invalid amount type")
	}

	destChain, ok := data["destination_chain"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid destination_chain")
	}

	// Build payload
	tokenAddr, err := types.NewAddress(token, types.ChainTypeNEAR)
	if err != nil {
		return nil, fmt.Errorf("invalid token address: %w", err)
	}

	payload := types.TokenTransferPayload{
		TokenAddress: tokenAddr,
		Amount:       fmt.Sprintf("%d", amount),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Build addresses
	senderAddr, err := types.NewAddress(sender, types.ChainTypeNEAR)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	recipientAddr, err := types.NewAddress(recipient, types.ChainTypeNEAR)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient address: %w", err)
	}

	// Build cross-chain message
	msg := &types.CrossChainMessage{
		ID:   messageID,
		Type: types.MessageTypeTokenTransfer,
		SourceChain: types.ChainInfo{
			Name:    l.config.Name,
			Type:    types.ChainTypeNEAR,
			ChainID: l.config.NetworkID,
		},
		DestinationChain: types.ChainInfo{
			Name: destChain,
		},
		Sender:    senderAddr,
		Recipient: recipientAddr,
		Payload:   payloadBytes,
		Nonce:     0, // Would extract from event
		Status:    types.MessageStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	l.logger.Info().
		Str("message_id", messageID).
		Str("sender", sender).
		Str("recipient", recipient).
		Str("token", token).
		Uint64("amount", amount).
		Str("dest_chain", destChain).
		Msg("Parsed token locked event")

	return msg, nil
}

// parseNFTLockedEvent parses an NFT locked event
func (l *Listener) parseNFTLockedEvent(data map[string]interface{}) (*types.CrossChainMessage, error) {
	// Extract fields from event data
	messageID, ok := data["message_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid message_id")
	}

	sender, ok := data["sender"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid sender")
	}

	recipient, ok := data["recipient"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid recipient")
	}

	nftContract, ok := data["nft_contract"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid nft_contract")
	}

	tokenID, ok := data["token_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid token_id")
	}

	destChain, ok := data["destination_chain"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid destination_chain")
	}

	// Build payload
	contractAddr, err := types.NewAddress(nftContract, types.ChainTypeNEAR)
	if err != nil {
		return nil, fmt.Errorf("invalid NFT contract address: %w", err)
	}

	payload := types.NFTTransferPayload{
		ContractAddress: contractAddr,
		TokenID:         tokenID,
		Standard:        "NEP171",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Build addresses
	senderAddr, err := types.NewAddress(sender, types.ChainTypeNEAR)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	recipientAddr, err := types.NewAddress(recipient, types.ChainTypeNEAR)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient address: %w", err)
	}

	// Build cross-chain message
	msg := &types.CrossChainMessage{
		ID:   messageID,
		Type: types.MessageTypeNFTTransfer,
		SourceChain: types.ChainInfo{
			Name:    l.config.Name,
			Type:    types.ChainTypeNEAR,
			ChainID: l.config.NetworkID,
		},
		DestinationChain: types.ChainInfo{
			Name: destChain,
		},
		Sender:    senderAddr,
		Recipient: recipientAddr,
		Payload:   payloadBytes,
		Nonce:     0, // Would extract from event
		Status:    types.MessageStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	l.logger.Info().
		Str("message_id", messageID).
		Str("sender", sender).
		Str("recipient", recipient).
		Str("nft_contract", nftContract).
		Str("token_id", tokenID).
		Str("dest_chain", destChain).
		Msg("Parsed NFT locked event")

	return msg, nil
}
