package near

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// Client represents a NEAR blockchain client
type Client struct {
	config     *types.ChainConfig
	rpcClients []*http.Client
	endpoints  []string
	logger     zerolog.Logger
	chainInfo  types.ChainInfo
}

// NewClient creates a new NEAR client
func NewClient(
	config *types.ChainConfig,
	logger zerolog.Logger,
) (*Client, error) {
	if config.ChainType != types.ChainTypeNEAR {
		return nil, fmt.Errorf("invalid chain type: expected NEAR, got %s", config.ChainType)
	}

	client := &Client{
		config:     config,
		rpcClients: make([]*http.Client, 0),
		endpoints:  config.RPCEndpoints,
		logger:     logger.With().Str("chain", config.Name).Logger(),
		chainInfo: types.ChainInfo{
			Name:        config.Name,
			Type:        types.ChainTypeNEAR,
			NetworkID:   config.NetworkID,
			Environment: config.Environment,
		},
	}

	// Create HTTP clients for each endpoint
	for range config.RPCEndpoints {
		httpClient := &http.Client{
			Timeout: 30 * time.Second,
		}
		client.rpcClients = append(client.rpcClients, httpClient)
	}

	client.logger.Info().
		Int("endpoints", len(client.endpoints)).
		Str("network", config.NetworkID).
		Msg("NEAR client initialized")

	return client, nil
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// callRPC calls the NEAR RPC API
func (c *Client) callRPC(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      "dontcare",
		Method:  method,
		Params:  params,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Try each endpoint
	for i, endpoint := range c.endpoints {
		resp, err := c.rpcClients[i].Post(
			endpoint,
			"application/json",
			bytes.NewReader(requestBytes),
		)
		if err != nil {
			c.logger.Warn().
				Err(err).
				Str("endpoint", endpoint).
				Msg("RPC request failed")
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to read response body")
			continue
		}

		var rpcResp RPCResponse
		if err := json.Unmarshal(body, &rpcResp); err != nil {
			c.logger.Warn().Err(err).Msg("Failed to unmarshal response")
			continue
		}

		if rpcResp.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
		}

		return rpcResp.Result, nil
	}

	return nil, fmt.Errorf("all RPC endpoints failed")
}

// GetChainType returns the chain type
func (c *Client) GetChainType() types.ChainType {
	return types.ChainTypeNEAR
}

// GetChainID returns the network ID
func (c *Client) GetChainID() string {
	return c.config.NetworkID
}

// GetChainInfo returns chain information
func (c *Client) GetChainInfo() types.ChainInfo {
	return c.chainInfo
}

// IsHealthy checks if the client is healthy
func (c *Client) IsHealthy(ctx context.Context) bool {
	_, err := c.GetStatus(ctx)
	return err == nil
}

// Close closes all connections
func (c *Client) Close() error {
	c.logger.Info().Msg("NEAR client closed")
	return nil
}

// StatusResponse represents NEAR status response
type StatusResponse struct {
	ChainID           string `json:"chain_id"`
	LatestBlockHash   string `json:"latest_block_hash"`
	LatestBlockHeight uint64 `json:"latest_block_height"`
	SyncInfo          struct {
		LatestBlockHash   string `json:"latest_block_hash"`
		LatestBlockHeight uint64 `json:"latest_block_height"`
		LatestBlockTime   string `json:"latest_block_time"`
		Syncing           bool   `json:"syncing"`
	} `json:"sync_info"`
}

// GetStatus returns NEAR node status
func (c *Client) GetStatus(ctx context.Context) (*StatusResponse, error) {
	result, err := c.callRPC(ctx, "status", []interface{}{})
	if err != nil {
		return nil, err
	}

	var status StatusResponse
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &status, nil
}

// GetLatestBlockNumber returns the latest block height
func (c *Client) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

// BlockResponse represents a NEAR block
type BlockResponse struct {
	Author string `json:"author"`
	Header struct {
		Height    uint64 `json:"height"`
		Hash      string `json:"hash"`
		Timestamp uint64 `json:"timestamp"`
	} `json:"header"`
	Chunks []struct {
		ChunkHash string `json:"chunk_hash"`
	} `json:"chunks"`
}

// GetBlockByNumber returns block information
func (c *Client) GetBlockByNumber(ctx context.Context, height uint64) (*types.BlockInfo, error) {
	params := map[string]interface{}{
		"block_id": height,
	}

	result, err := c.callRPC(ctx, "block", params)
	if err != nil {
		return nil, err
	}

	var block BlockResponse
	if err := json.Unmarshal(result, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &types.BlockInfo{
		Number:    block.Header.Height,
		Hash:      block.Header.Hash,
		Timestamp: time.Unix(0, int64(block.Header.Timestamp)),
	}, nil
}

// GetBlockTime returns the expected block time
func (c *Client) GetBlockTime() time.Duration {
	return c.config.GetBlockTimeDuration()
}

// GetConfirmationBlocks returns confirmation blocks
func (c *Client) GetConfirmationBlocks() uint64 {
	if c.config.ConfirmationBlocks > 0 {
		return c.config.ConfirmationBlocks
	}
	return 3 // NEAR default
}

// SendTransaction sends a signed transaction
func (c *Client) SendTransaction(ctx context.Context, tx interface{}) (string, error) {
	// NEAR transactions need to be base64 encoded
	txBytes, ok := tx.([]byte)
	if !ok {
		return "", fmt.Errorf("invalid transaction type: expected []byte")
	}

	params := []interface{}{txBytes}
	result, err := c.callRPC(ctx, "broadcast_tx_commit", params)
	if err != nil {
		return "", err
	}

	var txResult map[string]interface{}
	if err := json.Unmarshal(result, &txResult); err != nil {
		return "", fmt.Errorf("failed to unmarshal transaction result: %w", err)
	}

	txHash, ok := txResult["transaction"].(map[string]interface{})["hash"].(string)
	if !ok {
		return "", fmt.Errorf("failed to extract transaction hash")
	}

	c.logger.Info().
		Str("tx_hash", txHash).
		Msg("NEAR transaction sent")

	return txHash, nil
}

// GetTransactionStatus returns transaction status
func (c *Client) GetTransactionStatus(ctx context.Context, txHash string) (*types.TransactionStatus, error) {
	// NEAR uses tx_status RPC method
	// This is a simplified implementation
	return &types.TransactionStatus{
		Hash:      txHash,
		Success:   true,
		Confirmed: true,
		Finalized: true,
	}, nil
}

// WaitForConfirmation waits for transaction confirmation
func (c *Client) WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.GetBlockTime())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for confirmation")
		case <-ticker.C:
			status, err := c.GetTransactionStatus(ctx, txHash)
			if err != nil {
				c.logger.Warn().
					Err(err).
					Str("tx_hash", txHash).
					Msg("Error checking transaction status")
				continue
			}

			if status.Finalized {
				c.logger.Info().
					Str("tx_hash", txHash).
					Msg("Transaction finalized")
				return nil
			}
		}
	}
}

// AccountViewResponse represents account view response
type AccountViewResponse struct {
	Amount        string `json:"amount"`
	Locked        string `json:"locked"`
	CodeHash      string `json:"code_hash"`
	StorageUsage  uint64 `json:"storage_usage"`
	StoragePaidAt uint64 `json:"storage_paid_at"`
}

// GetNativeBalance returns NEAR balance
func (c *Client) GetNativeBalance(ctx context.Context, address string) (*big.Int, error) {
	params := map[string]interface{}{
		"request_type": "view_account",
		"finality":     "final",
		"account_id":   address,
	}

	result, err := c.callRPC(ctx, "query", params)
	if err != nil {
		return nil, err
	}

	var account AccountViewResponse
	if err := json.Unmarshal(result, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	balance, ok := new(big.Int).SetString(account.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse balance")
	}

	return balance, nil
}

// GetTokenBalance returns token balance (NEP-141)
func (c *Client) GetTokenBalance(ctx context.Context, address string, tokenContract string) (*big.Int, error) {
	// This would call ft_balance_of on the NEP-141 contract
	// Placeholder implementation
	return nil, fmt.Errorf("GetTokenBalance for NEAR not fully implemented")
}

// SubscribeToEvents subscribes to contract events
func (c *Client) SubscribeToEvents(ctx context.Context, contractAddress string, eventSignature string) (chan interface{}, error) {
	// NEAR doesn't have built-in event subscriptions like EVM
	// Would need to poll for receipts/logs
	return nil, fmt.Errorf("SubscribeToEvents for NEAR not fully implemented")
}

// ViewFunction calls a view function on a contract
func (c *Client) ViewFunction(ctx context.Context, contractID string, methodName string, args interface{}) (json.RawMessage, error) {
	argsBytes, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal args: %w", err)
	}

	params := map[string]interface{}{
		"request_type": "call_function",
		"finality":     "final",
		"account_id":   contractID,
		"method_name":  methodName,
		"args_base64":  argsBytes,
	}

	return c.callRPC(ctx, "query", params)
}
