package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/algorand"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// AlgorandClientAdapter adapts Algorand client to UniversalClient interface
type AlgorandClientAdapter struct {
	client *algorand.Client
	config *types.ChainConfig
	logger zerolog.Logger
}

func (a *AlgorandClientAdapter) GetChainType() types.ChainType {
	return types.ChainTypeAlgorand
}

func (a *AlgorandClientAdapter) GetChainID() string {
	return a.config.NetworkID
}

func (a *AlgorandClientAdapter) GetChainInfo() types.ChainInfo {
	return types.ChainInfo{
		Name:        a.config.Name,
		Type:        types.ChainTypeAlgorand,
		NetworkID:   a.config.NetworkID,
		Environment: a.config.Environment,
	}
}

func (a *AlgorandClientAdapter) IsHealthy(ctx context.Context) bool {
	return a.client.IsHealthy(ctx)
}

func (a *AlgorandClientAdapter) Close() error {
	return a.client.Close()
}

func (a *AlgorandClientAdapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	return a.client.GetLatestRound(ctx)
}

func (a *AlgorandClientAdapter) GetBlockByNumber(ctx context.Context, number uint64) (*types.BlockInfo, error) {
	return a.client.GetBlockByRound(ctx, number)
}

func (a *AlgorandClientAdapter) GetBlockTime() time.Duration {
	return a.config.GetBlockTimeDuration()
}

func (a *AlgorandClientAdapter) GetConfirmationBlocks() uint64 {
	return a.config.ConfirmationBlocks
}

func (a *AlgorandClientAdapter) SendTransaction(ctx context.Context, tx interface{}) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (a *AlgorandClientAdapter) GetTransactionStatus(ctx context.Context, txHash string) (*types.TransactionStatus, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *AlgorandClientAdapter) WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (a *AlgorandClientAdapter) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *AlgorandClientAdapter) GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *AlgorandClientAdapter) SubscribeToEvents(ctx context.Context, contractAddress string, eventSignature string) (chan interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}
