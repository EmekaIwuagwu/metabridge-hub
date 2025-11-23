package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/aptos"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// AptosClientAdapter adapts Aptos client to UniversalClient interface
type AptosClientAdapter struct {
	client *aptos.Client
	config *types.ChainConfig
	logger zerolog.Logger
}

func (a *AptosClientAdapter) GetChainType() types.ChainType {
	return types.ChainTypeAptos
}

func (a *AptosClientAdapter) GetChainID() string {
	return a.config.NetworkID
}

func (a *AptosClientAdapter) GetChainInfo() types.ChainInfo {
	return types.ChainInfo{
		Name:        a.config.Name,
		Type:        types.ChainTypeAptos,
		NetworkID:   a.config.NetworkID,
		Environment: a.config.Environment,
	}
}

func (a *AptosClientAdapter) IsHealthy(ctx context.Context) bool {
	return a.client.IsHealthy(ctx)
}

func (a *AptosClientAdapter) Close() error {
	return a.client.Close()
}

func (a *AptosClientAdapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	return a.client.GetLatestVersion(ctx)
}

func (a *AptosClientAdapter) GetBlockByNumber(ctx context.Context, number uint64) (*types.BlockInfo, error) {
	return a.client.GetBlockByVersion(ctx, number)
}

func (a *AptosClientAdapter) GetBlockTime() time.Duration {
	return a.config.GetBlockTimeDuration()
}

func (a *AptosClientAdapter) GetConfirmationBlocks() uint64 {
	return a.config.ConfirmationBlocks
}

func (a *AptosClientAdapter) SendTransaction(ctx context.Context, tx interface{}) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (a *AptosClientAdapter) GetTransactionStatus(ctx context.Context, txHash string) (*types.TransactionStatus, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *AptosClientAdapter) WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (a *AptosClientAdapter) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *AptosClientAdapter) GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *AptosClientAdapter) SubscribeToEvents(ctx context.Context, contractAddress string, eventSignature string) (chan interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}
