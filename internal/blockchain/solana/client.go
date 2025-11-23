package solana

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/rs/zerolog"
)

// Client represents a Solana blockchain client
type Client struct {
	config     *types.ChainConfig
	rpcClients []*rpc.Client
	wsClient   *rpc.Client
	logger     zerolog.Logger
	chainInfo  types.ChainInfo
}

// NewClient creates a new Solana client
func NewClient(
	config *types.ChainConfig,
	logger zerolog.Logger,
) (*Client, error) {
	if config.ChainType != types.ChainTypeSolana {
		return nil, fmt.Errorf("invalid chain type: expected SOLANA, got %s", config.ChainType)
	}

	client := &Client{
		config:     config,
		rpcClients: make([]*rpc.Client, 0),
		logger:     logger.With().Str("chain", config.Name).Logger(),
		chainInfo: types.ChainInfo{
			Name:        config.Name,
			Type:        types.ChainTypeSolana,
			NetworkID:   config.NetworkID,
			Environment: config.Environment,
		},
	}

	// Connect to RPC endpoints
	for _, endpoint := range config.RPCEndpoints {
		rpcClient := rpc.New(endpoint)
		client.rpcClients = append(client.rpcClients, rpcClient)
	}

	if len(client.rpcClients) == 0 {
		return nil, fmt.Errorf("no RPC clients initialized")
	}

	// Connect to WebSocket endpoint if available
	if config.WSEndpoint != "" {
		wsClient := rpc.New(config.WSEndpoint)
		client.wsClient = wsClient
	}

	client.logger.Info().
		Int("rpc_clients", len(client.rpcClients)).
		Msg("Solana client initialized")

	return client, nil
}

// GetChainType returns the chain type
func (c *Client) GetChainType() types.ChainType {
	return types.ChainTypeSolana
}

// GetChainID returns the network ID (cluster name)
func (c *Client) GetChainID() string {
	return c.config.NetworkID
}

// GetChainInfo returns chain information
func (c *Client) GetChainInfo() types.ChainInfo {
	return c.chainInfo
}

// IsHealthy checks if the client is healthy
func (c *Client) IsHealthy(ctx context.Context) bool {
	_, err := c.GetSlot(ctx)
	return err == nil
}

// Close closes all connections
func (c *Client) Close() error {
	// Solana client doesn't require explicit closing
	c.logger.Info().Msg("Solana client closed")
	return nil
}

// GetSlot returns the current slot
func (c *Client) GetSlot(ctx context.Context) (uint64, error) {
	commitment := c.getCommitment()

	for _, client := range c.rpcClients {
		slot, err := client.GetSlot(ctx, commitment)
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to get slot from endpoint")
			continue
		}
		return slot, nil
	}

	return 0, fmt.Errorf("failed to get slot from all endpoints")
}

// GetLatestBlockNumber returns the latest slot (Solana's equivalent of block number)
func (c *Client) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	return c.GetSlot(ctx)
}

// GetBlockByNumber returns block information by slot
func (c *Client) GetBlockByNumber(ctx context.Context, slot uint64) (*types.BlockInfo, error) {
	for _, client := range c.rpcClients {
		block, err := client.GetBlock(ctx, slot)
		if err != nil {
			c.logger.Warn().
				Err(err).
				Uint64("slot", slot).
				Msg("Failed to get block")
			continue
		}

		if block == nil {
			continue
		}

		return &types.BlockInfo{
			Number:    slot,
			Hash:      block.Blockhash.String(),
			Timestamp: time.Unix(int64(*block.BlockTime), 0),
			TxCount:   len(block.Transactions),
		}, nil
	}

	return nil, fmt.Errorf("failed to get block %d from all endpoints", slot)
}

// GetBlockTime returns the expected slot time
func (c *Client) GetBlockTime() time.Duration {
	return c.config.GetBlockTimeDuration()
}

// GetConfirmationBlocks returns the number of confirmation slots
func (c *Client) GetConfirmationBlocks() uint64 {
	if c.config.ConfirmationSlots > 0 {
		return c.config.ConfirmationSlots
	}
	return 32 // default
}

// SendTransaction sends a signed transaction
func (c *Client) SendTransaction(ctx context.Context, tx interface{}) (string, error) {
	solTx, ok := tx.(*solana.Transaction)
	if !ok {
		return "", fmt.Errorf("invalid transaction type: expected *solana.Transaction")
	}

	for _, client := range c.rpcClients {
		sig, err := client.SendTransactionWithOpts(
			ctx,
			solTx,
			rpc.TransactionOpts{
				SkipPreflight:       false,
				PreflightCommitment: c.getCommitment(),
			},
		)
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to send transaction")
			continue
		}

		c.logger.Info().
			Str("signature", sig.String()).
			Msg("Solana transaction sent")

		return sig.String(), nil
	}

	return "", fmt.Errorf("failed to send transaction to all endpoints")
}

// GetTransactionStatus returns transaction status
func (c *Client) GetTransactionStatus(ctx context.Context, signature string) (*types.TransactionStatus, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	for _, client := range c.rpcClients {
		result, err := client.GetSignatureStatuses(ctx, true, sig)
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to get signature status")
			continue
		}

		if result == nil || len(result.Value) == 0 || result.Value[0] == nil {
			return &types.TransactionStatus{
				Hash:      signature,
				Success:   false,
				Confirmed: false,
				Finalized: false,
			}, nil
		}

		status := result.Value[0]

		var success bool
		if status.Err != nil {
			success = false
		} else {
			success = true
		}

		confirmed := status.ConfirmationStatus == rpc.ConfirmationStatusConfirmed ||
			status.ConfirmationStatus == rpc.ConfirmationStatusFinalized

		finalized := status.ConfirmationStatus == rpc.ConfirmationStatusFinalized

		return &types.TransactionStatus{
			Hash:        signature,
			BlockNumber: status.Slot,
			Success:     success,
			Confirmed:   confirmed,
			Finalized:   finalized,
		}, nil
	}

	return nil, fmt.Errorf("failed to get transaction status from all endpoints")
}

// WaitForConfirmation waits for transaction confirmation
func (c *Client) WaitForConfirmation(ctx context.Context, signature string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.GetBlockTime())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for confirmation")
		case <-ticker.C:
			status, err := c.GetTransactionStatus(ctx, signature)
			if err != nil {
				c.logger.Warn().
					Err(err).
					Str("signature", signature).
					Msg("Error checking transaction status")
				continue
			}

			if !status.Success {
				return fmt.Errorf("transaction failed")
			}

			if status.Finalized {
				c.logger.Info().
					Str("signature", signature).
					Uint64("slot", status.BlockNumber).
					Msg("Transaction finalized")
				return nil
			}
		}
	}
}

// GetNativeBalance returns SOL balance
func (c *Client) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	pubkey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	commitment := c.getCommitment()

	for _, client := range c.rpcClients {
		balance, err := client.GetBalance(ctx, pubkey, commitment)
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to get balance")
			continue
		}

		return big.NewInt(int64(balance.Value)), nil
	}

	return nil, fmt.Errorf("failed to get balance from all endpoints")
}

// GetTokenBalance returns SPL token balance
func (c *Client) GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error) {
	// This would require SPL token account derivation and balance check
	// Placeholder implementation
	return nil, fmt.Errorf("GetTokenBalance for Solana not fully implemented")
}

// SubscribeToEvents subscribes to program events
func (c *Client) SubscribeToEvents(ctx context.Context, programAddress string, eventSignature string) (chan interface{}, error) {
	// This would use WebSocket subscriptions to monitor program logs
	// Placeholder implementation
	return nil, fmt.Errorf("SubscribeToEvents for Solana not fully implemented")
}

// getCommitment returns the configured commitment level
func (c *Client) getCommitment() rpc.CommitmentType {
	switch c.config.Commitment {
	case "processed":
		return rpc.CommitmentProcessed
	case "confirmed":
		return rpc.CommitmentConfirmed
	case "finalized":
		return rpc.CommitmentFinalized
	default:
		return rpc.CommitmentFinalized
	}
}

// GetProgramAccounts retrieves accounts owned by a program
func (c *Client) GetProgramAccounts(ctx context.Context, programID solana.PublicKey) (rpc.GetProgramAccountsResult, error) {
	for _, client := range c.rpcClients {
		accounts, err := client.GetProgramAccounts(ctx, programID)
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to get program accounts")
			continue
		}

		return accounts, nil
	}

	return nil, fmt.Errorf("failed to get program accounts from all endpoints")
}
