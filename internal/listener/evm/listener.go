package evm

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/blockchain/evm"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/monitoring"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog"
)

// Listener listens for events on an EVM blockchain
type Listener struct {
	client        *evm.Client
	config        *types.ChainConfig
	logger        zerolog.Logger
	eventChan     chan *types.CrossChainMessage
	stopChan      chan struct{}
	lastBlock     uint64
	bridgeAddress common.Address
}

// NewListener creates a new EVM event listener
func NewListener(
	client *evm.Client,
	config *types.ChainConfig,
	logger zerolog.Logger,
) (*Listener, error) {
	if config.BridgeContract == "" {
		return nil, fmt.Errorf("bridge contract address not configured")
	}

	bridgeAddress := common.HexToAddress(config.BridgeContract)

	return &Listener{
		client:        client,
		config:        config,
		logger:        logger.With().Str("chain", config.Name).Str("component", "listener").Logger(),
		eventChan:     make(chan *types.CrossChainMessage, 100),
		stopChan:      make(chan struct{}),
		lastBlock:     config.StartBlock,
		bridgeAddress: bridgeAddress,
	}, nil
}

// Start starts the listener
func (l *Listener) Start(ctx context.Context) error {
	l.logger.Info().
		Uint64("start_block", l.lastBlock).
		Str("bridge", l.bridgeAddress.Hex()).
		Msg("Starting EVM listener")

	// Start listening in a goroutine
	go l.listen(ctx)

	return nil
}

// Stop stops the listener
func (l *Listener) Stop() error {
	l.logger.Info().Msg("Stopping EVM listener")
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

	// Process in batches to avoid overwhelming
	batchSize := uint64(100)
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

	// Query logs
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{l.bridgeAddress},
	}

	logs, err := l.client.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to filter logs: %w", err)
	}

	l.logger.Debug().
		Int("logs", len(logs)).
		Msg("Logs retrieved")

	// Process each log
	for _, vLog := range logs {
		if err := l.processLog(ctx, vLog); err != nil {
			l.logger.Error().
				Err(err).
				Str("tx_hash", vLog.TxHash.Hex()).
				Msg("Error processing log")
			continue
		}
	}

	return nil
}

// processLog processes a single log entry
func (l *Listener) processLog(ctx context.Context, vLog ethtypes.Log) error {
	// Parse event based on topic
	if len(vLog.Topics) == 0 {
		return nil
	}

	topic := vLog.Topics[0].Hex()

	l.logger.Debug().
		Str("topic", topic).
		Str("tx_hash", vLog.TxHash.Hex()).
		Msg("Processing log")

	// Detect event type by topic hash
	// TokenLocked event: keccak256("TokenLocked(bytes32,address,address,uint256,string,string,uint256)")
	// NFTLocked event: keccak256("NFTLocked(bytes32,address,address,uint256,string,string,uint256)")

	// For now, we'll create a placeholder message
	// In production, you would decode the actual event data

	msg, err := l.createMessageFromLog(ctx, vLog)
	if err != nil {
		return fmt.Errorf("failed to create message from log: %w", err)
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

// createMessageFromLog creates a CrossChainMessage from a log entry
func (l *Listener) createMessageFromLog(ctx context.Context, vLog ethtypes.Log) (*types.CrossChainMessage, error) {
	// This is a simplified implementation
	// In production, you would decode the actual event data using the ABI

	// For now, return nil (no message created)
	// You would implement proper event decoding here
	return nil, nil
}
