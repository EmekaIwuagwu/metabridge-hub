package blockchain

import (
	"context"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/solana"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// SolanaClientAdapter adapts a Solana client to the UniversalClient interface
type SolanaClientAdapter struct {
	client *solana.Client
	config *types.ChainConfig
	logger zerolog.Logger
}

// GetChainType returns the chain type
func (a *SolanaClientAdapter) GetChainType() types.ChainType {
	return types.ChainTypeSolana
}

// GetChainID returns the network ID
func (a *SolanaClientAdapter) GetChainID() string {
	return a.config.NetworkID
}

// GetChainInfo returns chain information
func (a *SolanaClientAdapter) GetChainInfo() types.ChainInfo {
	return types.ChainInfo{
		Name:        a.config.Name,
		Type:        types.ChainTypeSolana,
		NetworkID:   a.config.NetworkID,
		Environment: a.config.Environment,
	}
}

// IsHealthy checks if the client is healthy
func (a *SolanaClientAdapter) IsHealthy(ctx context.Context) bool {
	return a.client.IsHealthy(ctx)
}

// Close closes the client
func (a *SolanaClientAdapter) Close() error {
	return a.client.Close()
}

// GetLatestBlockNumber returns the latest slot number
func (a *SolanaClientAdapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	return a.client.GetLatestBlockNumber(ctx)
}

// GetBlockByNumber returns block information
func (a *SolanaClientAdapter) GetBlockByNumber(ctx context.Context, slot uint64) (*types.BlockInfo, error) {
	return a.client.GetBlockByNumber(ctx, slot)
}

// GetBlockTime returns the slot time
func (a *SolanaClientAdapter) GetBlockTime() time.Duration {
	return a.client.GetBlockTime()
}

// GetConfirmationBlocks returns confirmation slots
func (a *SolanaClientAdapter) GetConfirmationBlocks() uint64 {
	return a.client.GetConfirmationBlocks()
}

// SendTransaction sends a transaction
func (a *SolanaClientAdapter) SendTransaction(ctx context.Context, tx interface{}) (string, error) {
	return a.client.SendTransaction(ctx, tx)
}

// GetTransactionStatus returns transaction status
func (a *SolanaClientAdapter) GetTransactionStatus(ctx context.Context, signature string) (*types.TransactionStatus, error) {
	return a.client.GetTransactionStatus(ctx, signature)
}

// WaitForConfirmation waits for transaction confirmation
func (a *SolanaClientAdapter) WaitForConfirmation(ctx context.Context, signature string, timeout time.Duration) error {
	return a.client.WaitForConfirmation(ctx, signature, timeout)
}

// GetNativeBalance returns SOL balance
func (a *SolanaClientAdapter) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	return a.client.GetNativeBalance(ctx, address)
}

// GetTokenBalance returns SPL token balance
func (a *SolanaClientAdapter) GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error) {
	return a.client.GetTokenBalance(ctx, address, tokenAddress)
}

// SubscribeToEvents subscribes to program events
func (a *SolanaClientAdapter) SubscribeToEvents(ctx context.Context, programAddress string, eventSignature string) (chan interface{}, error) {
	return a.client.SubscribeToEvents(ctx, programAddress, eventSignature)
}

// GetUnderlyingClient returns the underlying Solana client
func (a *SolanaClientAdapter) GetUnderlyingClient() *solana.Client {
	return a.client
}
