package blockchain

import (
	"context"
	"fmt"

	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/algorand"
	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/aptos"
	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/evm"
	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/near"
	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain/solana"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/rs/zerolog"
)

// ClientFactory creates blockchain clients based on chain configuration
type ClientFactory struct {
	logger zerolog.Logger
}

// NewClientFactory creates a new client factory
func NewClientFactory(logger zerolog.Logger) *ClientFactory {
	return &ClientFactory{
		logger: logger.With().Str("component", "client_factory").Logger(),
	}
}

// CreateClient creates the appropriate client based on chain type
func (f *ClientFactory) CreateClient(
	ctx context.Context,
	config *types.ChainConfig,
) (types.UniversalClient, error) {
	f.logger.Info().
		Str("chain_name", config.Name).
		Str("chain_type", string(config.ChainType)).
		Str("environment", string(config.Environment)).
		Msg("Creating blockchain client")

	switch config.ChainType {
	case types.ChainTypeEVM:
		return f.createEVMClient(ctx, config)
	case types.ChainTypeSolana:
		return f.createSolanaClient(ctx, config)
	case types.ChainTypeNEAR:
		return f.createNEARClient(ctx, config)
	case types.ChainTypeAlgorand:
		return f.createAlgorandClient(ctx, config)
	case types.ChainTypeAptos:
		return f.createAptosClient(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported chain type: %s", config.ChainType)
	}
}

// CreateAllClients creates clients for all configured chains
func (f *ClientFactory) CreateAllClients(
	ctx context.Context,
	configs []types.ChainConfig,
) (map[string]types.UniversalClient, error) {
	clients := make(map[string]types.UniversalClient)

	for i := range configs {
		config := &configs[i]

		if !config.Enabled {
			f.logger.Info().
				Str("chain_name", config.Name).
				Msg("Chain is disabled, skipping")
			continue
		}

		client, err := f.CreateClient(ctx, config)
		if err != nil {
			f.logger.Error().
				Err(err).
				Str("chain_name", config.Name).
				Msg("Failed to create client")
			// Continue with other chains instead of failing completely
			continue
		}

		clients[config.Name] = client

		f.logger.Info().
			Str("chain_name", config.Name).
			Str("chain_type", string(config.ChainType)).
			Msg("Successfully created blockchain client")
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("no blockchain clients were successfully created")
	}

	f.logger.Info().
		Int("total_clients", len(clients)).
		Msg("Blockchain clients initialized")

	return clients, nil
}

// createEVMClient creates an EVM blockchain client
func (f *ClientFactory) createEVMClient(
	ctx context.Context,
	config *types.ChainConfig,
) (types.UniversalClient, error) {
	evmClient, err := evm.NewClient(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create EVM client: %w", err)
	}

	// Wrap in adapter to implement UniversalClient interface
	return &EVMClientAdapter{
		client: evmClient,
		config: config,
		logger: f.logger,
	}, nil
}

// createSolanaClient creates a Solana blockchain client
func (f *ClientFactory) createSolanaClient(
	ctx context.Context,
	config *types.ChainConfig,
) (types.UniversalClient, error) {
	solanaClient, err := solana.NewClient(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana client: %w", err)
	}

	// Wrap in adapter
	return &SolanaClientAdapter{
		client: solanaClient,
		config: config,
		logger: f.logger,
	}, nil
}

// createNEARClient creates a NEAR blockchain client
func (f *ClientFactory) createNEARClient(
	ctx context.Context,
	config *types.ChainConfig,
) (types.UniversalClient, error) {
	nearClient, err := near.NewClient(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create NEAR client: %w", err)
	}

	// Wrap in adapter
	return &NEARClientAdapter{
		client: nearClient,
		config: config,
		logger: f.logger,
	}, nil
}

// createAlgorandClient creates an Algorand blockchain client
func (f *ClientFactory) createAlgorandClient(
	ctx context.Context,
	config *types.ChainConfig,
) (types.UniversalClient, error) {
	algorandClient, err := algorand.NewClient(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Algorand client: %w", err)
	}

	// Wrap in adapter
	return &AlgorandClientAdapter{
		client: algorandClient,
		config: config,
		logger: f.logger,
	}, nil
}

// createAptosClient creates an Aptos blockchain client
func (f *ClientFactory) createAptosClient(
	ctx context.Context,
	config *types.ChainConfig,
) (types.UniversalClient, error) {
	aptosClient, err := aptos.NewClient(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Aptos client: %w", err)
	}

	// Wrap in adapter
	return &AptosClientAdapter{
		client: aptosClient,
		config: config,
		logger: f.logger,
	}, nil
}

// CloseAllClients closes all blockchain clients
func CloseAllClients(clients map[string]types.UniversalClient, logger zerolog.Logger) {
	for name, client := range clients {
		if err := client.Close(); err != nil {
			logger.Error().
				Err(err).
				Str("chain_name", name).
				Msg("Error closing client")
		} else {
			logger.Info().
				Str("chain_name", name).
				Msg("Client closed successfully")
		}
	}
}
