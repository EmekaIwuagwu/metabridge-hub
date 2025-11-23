package blockchain

import (
	"context"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/evm"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// EVMClientAdapter adapts an EVM client to the UniversalClient interface
type EVMClientAdapter struct {
	client *evm.Client
	config *types.ChainConfig
	logger zerolog.Logger
}

// GetChainType returns the chain type
func (a *EVMClientAdapter) GetChainType() types.ChainType {
	return types.ChainTypeEVM
}

// GetChainID returns the chain ID
func (a *EVMClientAdapter) GetChainID() string {
	return a.config.ChainID
}

// GetChainInfo returns chain information
func (a *EVMClientAdapter) GetChainInfo() types.ChainInfo {
	return types.ChainInfo{
		Name:        a.config.Name,
		Type:        types.ChainTypeEVM,
		ChainID:     a.config.ChainID,
		Environment: a.config.Environment,
	}
}

// IsHealthy checks if the client is healthy
func (a *EVMClientAdapter) IsHealthy(ctx context.Context) bool {
	return a.client.IsHealthy(ctx)
}

// Close closes the client
func (a *EVMClientAdapter) Close() error {
	return a.client.Close()
}

// GetLatestBlockNumber returns the latest block number
func (a *EVMClientAdapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	return a.client.GetLatestBlockNumber(ctx)
}

// GetBlockByNumber returns block information
func (a *EVMClientAdapter) GetBlockByNumber(ctx context.Context, number uint64) (*types.BlockInfo, error) {
	return a.client.GetBlockByNumber(ctx, number)
}

// GetBlockTime returns the block time
func (a *EVMClientAdapter) GetBlockTime() time.Duration {
	return a.client.GetBlockTime()
}

// GetConfirmationBlocks returns confirmation blocks
func (a *EVMClientAdapter) GetConfirmationBlocks() uint64 {
	return a.client.GetConfirmationBlocks()
}

// SendTransaction sends a transaction
func (a *EVMClientAdapter) SendTransaction(ctx context.Context, tx interface{}) (string, error) {
	return a.client.SendTransaction(ctx, tx)
}

// GetTransactionStatus returns transaction status
func (a *EVMClientAdapter) GetTransactionStatus(ctx context.Context, txHash string) (*types.TransactionStatus, error) {
	return a.client.GetTransactionStatus(ctx, txHash)
}

// WaitForConfirmation waits for transaction confirmation
func (a *EVMClientAdapter) WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) error {
	return a.client.WaitForConfirmation(ctx, txHash, timeout)
}

// GetNativeBalance returns native balance
func (a *EVMClientAdapter) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	return a.client.GetNativeBalance(ctx, address)
}

// GetTokenBalance returns token balance
func (a *EVMClientAdapter) GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error) {
	return a.client.GetTokenBalance(ctx, address, tokenAddress)
}

// SubscribeToEvents subscribes to contract events
func (a *EVMClientAdapter) SubscribeToEvents(ctx context.Context, contractAddress string, eventSignature string) (chan interface{}, error) {
	return a.client.SubscribeToEvents(ctx, contractAddress, eventSignature)
}

// GetUnderlyingClient returns the underlying EVM client for EVM-specific operations
func (a *EVMClientAdapter) GetUnderlyingClient() *evm.Client {
	return a.client
}
