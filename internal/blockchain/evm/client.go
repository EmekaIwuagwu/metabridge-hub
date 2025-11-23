package evm

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
)

// Client represents an EVM blockchain client
type Client struct {
	config        *types.ChainConfig
	clients       []*ethclient.Client
	wsClient      *ethclient.Client
	currentIndex  int
	mu            sync.RWMutex
	logger        zerolog.Logger
	chainInfo     types.ChainInfo
	healthChecker *HealthChecker
}

// NewClient creates a new EVM client
func NewClient(
	config *types.ChainConfig,
	logger zerolog.Logger,
) (*Client, error) {
	if config.ChainType != types.ChainTypeEVM {
		return nil, fmt.Errorf("invalid chain type: expected EVM, got %s", config.ChainType)
	}

	client := &Client{
		config:       config,
		clients:      make([]*ethclient.Client, 0, len(config.RPCEndpoints)),
		logger:       logger.With().Str("chain", config.Name).Logger(),
		currentIndex: 0,
		chainInfo: types.ChainInfo{
			Name:        config.Name,
			Type:        types.ChainTypeEVM,
			ChainID:     config.ChainID,
			Environment: config.Environment,
		},
	}

	// Connect to all RPC endpoints
	for i, endpoint := range config.RPCEndpoints {
		rpcClient, err := ethclient.Dial(endpoint)
		if err != nil {
			client.logger.Warn().
				Err(err).
				Str("endpoint", endpoint).
				Int("index", i).
				Msg("Failed to connect to RPC endpoint")
			continue
		}
		client.clients = append(client.clients, rpcClient)
	}

	if len(client.clients) == 0 {
		return nil, fmt.Errorf("failed to connect to any RPC endpoint")
	}

	client.logger.Info().
		Int("connected_rpcs", len(client.clients)).
		Msg("EVM client initialized")

	// Connect to WebSocket endpoint if available
	if config.WSEndpoint != "" {
		wsClient, err := ethclient.Dial(config.WSEndpoint)
		if err != nil {
			client.logger.Warn().
				Err(err).
				Str("ws_endpoint", config.WSEndpoint).
				Msg("Failed to connect to WebSocket endpoint")
		} else {
			client.wsClient = wsClient
			client.logger.Info().Msg("WebSocket connection established")
		}
	}

	// Initialize health checker
	client.healthChecker = NewHealthChecker(client, logger)

	return client, nil
}

// GetChainType returns the chain type
func (c *Client) GetChainType() types.ChainType {
	return types.ChainTypeEVM
}

// GetChainID returns the chain ID
func (c *Client) GetChainID() string {
	return c.config.ChainID
}

// GetChainInfo returns chain information
func (c *Client) GetChainInfo() types.ChainInfo {
	return c.chainInfo
}

// IsHealthy checks if the client is healthy
func (c *Client) IsHealthy(ctx context.Context) bool {
	return c.healthChecker.IsHealthy(ctx)
}

// Close closes all client connections
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors []string

	for i, client := range c.clients {
		if client != nil {
			client.Close()
			c.logger.Debug().Int("index", i).Msg("Closed RPC client")
		}
	}

	if c.wsClient != nil {
		c.wsClient.Close()
		c.logger.Debug().Msg("Closed WebSocket client")
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing clients: %s", strings.Join(errors, "; "))
	}

	return nil
}

// getClient returns the current active client with failover
func (c *Client) getClient() *ethclient.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.clients) == 0 {
		return nil
	}

	return c.clients[c.currentIndex%len(c.clients)]
}

// rotateClient rotates to the next available client
func (c *Client) rotateClient() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.currentIndex = (c.currentIndex + 1) % len(c.clients)
	c.logger.Debug().
		Int("new_index", c.currentIndex).
		Msg("Rotated to next RPC client")
}

// executeWithFailover executes a function with automatic failover
func (c *Client) executeWithFailover(ctx context.Context, fn func(*ethclient.Client) error) error {
	maxRetries := len(c.clients)
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		client := c.getClient()
		if client == nil {
			return fmt.Errorf("no available clients")
		}

		err := fn(client)
		if err == nil {
			return nil
		}

		lastErr = err
		c.logger.Warn().
			Err(err).
			Int("attempt", i+1).
			Msg("RPC call failed, trying next endpoint")

		c.rotateClient()
	}

	return fmt.Errorf("all RPC endpoints failed: %w", lastErr)
}

// GetLatestBlockNumber returns the latest block number
func (c *Client) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	var blockNumber uint64

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		bn, err := client.BlockNumber(ctx)
		if err != nil {
			return err
		}
		blockNumber = bn
		return nil
	})

	return blockNumber, err
}

// GetBlockByNumber returns block information by number
func (c *Client) GetBlockByNumber(ctx context.Context, number uint64) (*types.BlockInfo, error) {
	var block *ethtypes.Header

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		b, err := client.HeaderByNumber(ctx, big.NewInt(int64(number)))
		if err != nil {
			return err
		}
		block = b
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &types.BlockInfo{
		Number:    block.Number.Uint64(),
		Hash:      block.Hash().Hex(),
		Timestamp: time.Unix(int64(block.Time), 0),
	}, nil
}

// GetBlockTime returns the expected block time
func (c *Client) GetBlockTime() time.Duration {
	return c.config.GetBlockTimeDuration()
}

// GetConfirmationBlocks returns the number of confirmation blocks required
func (c *Client) GetConfirmationBlocks() uint64 {
	return c.config.ConfirmationBlocks
}

// SendTransaction sends a signed transaction
func (c *Client) SendTransaction(ctx context.Context, tx interface{}) (string, error) {
	ethTx, ok := tx.(*ethtypes.Transaction)
	if !ok {
		return "", fmt.Errorf("invalid transaction type: expected *types.Transaction")
	}

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		return client.SendTransaction(ctx, ethTx)
	})

	if err != nil {
		return "", err
	}

	txHash := ethTx.Hash().Hex()
	c.logger.Info().
		Str("tx_hash", txHash).
		Msg("Transaction sent successfully")

	return txHash, nil
}

// GetTransactionStatus returns the status of a transaction
func (c *Client) GetTransactionStatus(ctx context.Context, txHash string) (*types.TransactionStatus, error) {
	hash := common.HexToHash(txHash)
	var receipt *ethtypes.Receipt

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		r, err := client.TransactionReceipt(ctx, hash)
		if err != nil {
			return err
		}
		receipt = r
		return nil
	})

	if err != nil {
		// Check if transaction is pending
		if strings.Contains(err.Error(), "not found") {
			return &types.TransactionStatus{
				Hash:      txHash,
				Success:   false,
				Confirmed: false,
				Finalized: false,
			}, nil
		}
		return nil, err
	}

	// Check if transaction is finalized (enough confirmations)
	latestBlock, err := c.GetLatestBlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	confirmations := latestBlock - receipt.BlockNumber.Uint64()
	confirmed := confirmations >= c.config.ConfirmationBlocks
	finalized := confirmations >= c.config.ConfirmationBlocks*2

	return &types.TransactionStatus{
		Hash:        txHash,
		BlockNumber: receipt.BlockNumber.Uint64(),
		Success:     receipt.Status == 1,
		Confirmed:   confirmed,
		Finalized:   finalized,
		GasUsed:     receipt.GasUsed,
	}, nil
}

// WaitForConfirmation waits for a transaction to be confirmed
func (c *Client) WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.GetBlockTime())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for confirmation: %w", ctx.Err())
		case <-ticker.C:
			status, err := c.GetTransactionStatus(ctx, txHash)
			if err != nil {
				c.logger.Warn().
					Err(err).
					Str("tx_hash", txHash).
					Msg("Error checking transaction status")
				continue
			}

			if !status.Success {
				return fmt.Errorf("transaction failed")
			}

			if status.Confirmed {
				c.logger.Info().
					Str("tx_hash", txHash).
					Uint64("block", status.BlockNumber).
					Msg("Transaction confirmed")
				return nil
			}
		}
	}
}

// GetNativeBalance returns the native token balance for an address
func (c *Client) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	addr := common.HexToAddress(address)
	var balance *big.Int

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		b, err := client.BalanceAt(ctx, addr, nil)
		if err != nil {
			return err
		}
		balance = b
		return nil
	})

	return balance, err
}

// GetTokenBalance returns the token balance for an address
func (c *Client) GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error) {
	// This would use ERC20 contract call
	// Implementation depends on having the ERC20 ABI
	// For now, return error indicating not implemented
	return nil, fmt.Errorf("GetTokenBalance not implemented - requires ERC20 contract integration")
}

// SubscribeToEvents subscribes to contract events
func (c *Client) SubscribeToEvents(ctx context.Context, contractAddress string, eventSignature string) (chan interface{}, error) {
	if c.wsClient == nil {
		return nil, fmt.Errorf("WebSocket client not available")
	}

	eventChan := make(chan interface{}, 100)

	// This would set up event subscription using FilterLogs
	// Implementation depends on specific event types
	// For now, return error indicating not fully implemented
	return eventChan, fmt.Errorf("SubscribeToEvents not fully implemented")
}

// GetTransactionByHash retrieves a transaction by hash
func (c *Client) GetTransactionByHash(ctx context.Context, txHash string) (*ethtypes.Transaction, bool, error) {
	hash := common.HexToHash(txHash)
	var tx *ethtypes.Transaction
	var isPending bool

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		t, pending, err := client.TransactionByHash(ctx, hash)
		if err != nil {
			return err
		}
		tx = t
		isPending = pending
		return nil
	})

	return tx, isPending, err
}

// EstimateGas estimates gas for a transaction
func (c *Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	var gasLimit uint64

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		gas, err := client.EstimateGas(ctx, msg)
		if err != nil {
			return err
		}
		gasLimit = gas
		return nil
	})

	if err != nil {
		return 0, err
	}

	// Apply gas limit multiplier
	multiplier := c.config.GasLimitMultiplier
	if multiplier == 0 {
		multiplier = 1.2 // default 20% buffer
	}

	adjustedGas := uint64(float64(gasLimit) * multiplier)

	return adjustedGas, nil
}

// SuggestGasPrice suggests a gas price
func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var gasPrice *big.Int

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		gp, err := client.SuggestGasPrice(ctx)
		if err != nil {
			return err
		}
		gasPrice = gp
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Check against max gas price if configured
	if c.config.MaxGasPrice != "" {
		maxGasPrice, ok := new(big.Int).SetString(c.config.MaxGasPrice, 10)
		if ok && gasPrice.Cmp(maxGasPrice) > 0 {
			c.logger.Warn().
				Str("suggested", gasPrice.String()).
				Str("max", maxGasPrice.String()).
				Msg("Gas price exceeds maximum, capping")
			gasPrice = maxGasPrice
		}
	}

	return gasPrice, nil
}

// ChainID returns the chain ID
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	var chainID *big.Int

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		cid, err := client.ChainID(ctx)
		if err != nil {
			return err
		}
		chainID = cid
		return nil
	})

	return chainID, err
}

// FilterLogs filters blockchain logs based on the provided query
func (c *Client) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]ethtypes.Log, error) {
	var logs []ethtypes.Log

	err := c.executeWithFailover(ctx, func(client *ethclient.Client) error {
		result, err := client.FilterLogs(ctx, query)
		if err != nil {
			return err
		}
		logs = result
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to filter logs: %w", err)
	}

	return logs, nil
}
