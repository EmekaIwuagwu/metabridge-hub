package blockchain

import (
	"context"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/near"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// NEARClientAdapter adapts a NEAR client to the UniversalClient interface
type NEARClientAdapter struct {
	client *near.Client
	config *types.ChainConfig
	logger zerolog.Logger
}

// GetChainType returns the chain type
func (a *NEARClientAdapter) GetChainType() types.ChainType {
	return types.ChainTypeNEAR
}

// GetChainID returns the network ID
func (a *NEARClientAdapter) GetChainID() string {
	return a.config.NetworkID
}

// GetChainInfo returns chain information
func (a *NEARClientAdapter) GetChainInfo() types.ChainInfo {
	return types.ChainInfo{
		Name:        a.config.Name,
		Type:        types.ChainTypeNEAR,
		NetworkID:   a.config.NetworkID,
		Environment: a.config.Environment,
	}
}

// IsHealthy checks if the client is healthy
func (a *NEARClientAdapter) IsHealthy(ctx context.Context) bool {
	return a.client.IsHealthy(ctx)
}

// Close closes the client
func (a *NEARClientAdapter) Close() error {
	return a.client.Close()
}

// GetLatestBlockNumber returns the latest block height
func (a *NEARClientAdapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	return a.client.GetLatestBlockNumber(ctx)
}

// GetBlockByNumber returns block information
func (a *NEARClientAdapter) GetBlockByNumber(ctx context.Context, height uint64) (*types.BlockInfo, error) {
	return a.client.GetBlockByNumber(ctx, height)
}

// GetBlockTime returns the block time
func (a *NEARClientAdapter) GetBlockTime() time.Duration {
	return a.client.GetBlockTime()
}

// GetConfirmationBlocks returns confirmation blocks
func (a *NEARClientAdapter) GetConfirmationBlocks() uint64 {
	return a.client.GetConfirmationBlocks()
}

// SendTransaction sends a transaction
func (a *NEARClientAdapter) SendTransaction(ctx context.Context, tx interface{}) (string, error) {
	return a.client.SendTransaction(ctx, tx)
}

// GetTransactionStatus returns transaction status
func (a *NEARClientAdapter) GetTransactionStatus(ctx context.Context, txHash string) (*types.TransactionStatus, error) {
	return a.client.GetTransactionStatus(ctx, txHash)
}

// WaitForConfirmation waits for transaction confirmation
func (a *NEARClientAdapter) WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) error {
	return a.client.WaitForConfirmation(ctx, txHash, timeout)
}

// GetNativeBalance returns NEAR balance
func (a *NEARClientAdapter) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	return a.client.GetNativeBalance(ctx, address)
}

// GetTokenBalance returns NEP-141 token balance
func (a *NEARClientAdapter) GetTokenBalance(ctx context.Context, address string, tokenContract string) (*big.Int, error) {
	return a.client.GetTokenBalance(ctx, address, tokenContract)
}

// SubscribeToEvents subscribes to contract events
func (a *NEARClientAdapter) SubscribeToEvents(ctx context.Context, contractAddress string, eventSignature string) (chan interface{}, error) {
	return a.client.SubscribeToEvents(ctx, contractAddress, eventSignature)
}

// GetUnderlyingClient returns the underlying NEAR client
func (a *NEARClientAdapter) GetUnderlyingClient() *near.Client {
	return a.client
}
