package types

import (
	"context"
	"math/big"
	"time"
)

// ChainType represents the blockchain architecture
type ChainType string

const (
	ChainTypeEVM    ChainType = "EVM"
	ChainTypeSolana ChainType = "SOLANA"
	ChainTypeNEAR   ChainType = "NEAR"
)

// Environment represents the deployment environment
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTestnet     Environment = "testnet"
	EnvironmentMainnet     Environment = "mainnet"
)

// ChainInfo contains blockchain-specific information
type ChainInfo struct {
	Name        string      `json:"name"`
	Type        ChainType   `json:"type"`
	ChainID     string      `json:"chain_id"`
	NetworkID   string      `json:"network_id,omitempty"` // For NEAR
	Environment Environment `json:"environment"`
}

// TransactionStatus represents universal transaction status
type TransactionStatus struct {
	Hash        string    `json:"hash"`
	BlockNumber uint64    `json:"block_number"`
	Success     bool      `json:"success"`
	Confirmed   bool      `json:"confirmed"`
	Finalized   bool      `json:"finalized"`
	Timestamp   time.Time `json:"timestamp"`
	GasUsed     uint64    `json:"gas_used,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// BlockInfo represents universal block information
type BlockInfo struct {
	Number    uint64    `json:"number"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	TxCount   int       `json:"tx_count"`
}

// UniversalClient provides a unified interface for all blockchains
type UniversalClient interface {
	// Chain information
	GetChainType() ChainType
	GetChainID() string
	GetChainInfo() ChainInfo
	IsHealthy(ctx context.Context) bool
	Close() error

	// Block operations
	GetLatestBlockNumber(ctx context.Context) (uint64, error)
	GetBlockByNumber(ctx context.Context, number uint64) (*BlockInfo, error)
	GetBlockTime() time.Duration
	GetConfirmationBlocks() uint64

	// Transaction operations
	SendTransaction(ctx context.Context, tx interface{}) (string, error)
	GetTransactionStatus(ctx context.Context, txHash string) (*TransactionStatus, error)
	WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) error

	// Balance operations
	GetNativeBalance(ctx context.Context, address string) (*big.Int, error)
	GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error)

	// Event subscription
	SubscribeToEvents(ctx context.Context, contractAddress string, eventSignature string) (chan interface{}, error)
}

// ChainConfig represents the configuration for a blockchain
type ChainConfig struct {
	Name               string      `mapstructure:"name"`
	ChainType          ChainType   `mapstructure:"chain_type"`
	Environment        Environment `mapstructure:"environment"`
	ChainID            string      `mapstructure:"chain_id"`
	NetworkID          string      `mapstructure:"network_id"`
	RPCEndpoints       []string    `mapstructure:"rpc_endpoints"`
	WSEndpoint         string      `mapstructure:"ws_endpoint"`
	BridgeContract     string      `mapstructure:"bridge_contract"`
	BridgeProgram      string      `mapstructure:"bridge_program"`
	StartBlock         uint64      `mapstructure:"start_block"`
	StartSlot          uint64      `mapstructure:"start_slot"`
	ConfirmationBlocks uint64      `mapstructure:"confirmation_blocks"`
	ConfirmationSlots  uint64      `mapstructure:"confirmation_slots"`
	BlockTime          string      `mapstructure:"block_time"`
	MaxGasPrice        string      `mapstructure:"max_gas_price"`
	GasLimitMultiplier float64     `mapstructure:"gas_limit_multiplier"`
	MaxReorgDepth      uint64      `mapstructure:"max_reorg_depth"`
	PollInterval       string      `mapstructure:"poll_interval"`
	Commitment         string      `mapstructure:"commitment"`
	ComputeUnitPrice   string      `mapstructure:"compute_unit_price"`
	MaxRetries         int         `mapstructure:"max_retries"`
	Enabled            bool        `mapstructure:"enabled"`
}

// GetBlockTimeDuration returns block time as duration
func (c *ChainConfig) GetBlockTimeDuration() time.Duration {
	if c.BlockTime == "" {
		return 2 * time.Second // default
	}
	duration, err := time.ParseDuration(c.BlockTime)
	if err != nil {
		return 2 * time.Second
	}
	return duration
}

// GetPollIntervalDuration returns poll interval as duration
func (c *ChainConfig) GetPollIntervalDuration() time.Duration {
	if c.PollInterval == "" {
		return 5 * time.Second // default
	}
	duration, err := time.ParseDuration(c.PollInterval)
	if err != nil {
		return 5 * time.Second
	}
	return duration
}

// ClientHealth represents the health status of a blockchain client
type ClientHealth struct {
	ChainName    string        `json:"chain_name"`
	ChainType    ChainType     `json:"chain_type"`
	IsHealthy    bool          `json:"is_healthy"`
	LatestBlock  uint64        `json:"latest_block"`
	SyncedBlock  uint64        `json:"synced_block"`
	BlockLag     uint64        `json:"block_lag"`
	LastChecked  time.Time     `json:"last_checked"`
	Error        string        `json:"error,omitempty"`
	RPCEndpoint  string        `json:"rpc_endpoint"`
	ResponseTime time.Duration `json:"response_time"`
}
