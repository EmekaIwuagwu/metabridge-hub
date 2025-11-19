package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/blockchain"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/crypto"
	evmCrypto "github.com/EmekaIwuagwu/metabridge-hub/internal/crypto/evm"
	ed25519Crypto "github.com/EmekaIwuagwu/metabridge-hub/internal/crypto/ed25519"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/queue"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/relayer"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/rs/zerolog"
)

var (
	configPath = flag.String("config", "config/config.testnet.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Setup logger
	logger := setupLogger()

	logger.Info().
		Str("service", "relayer").
		Str("config", *configPath).
		Msg("Starting Metabridge Relayer service")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	logger.Info().
		Str("environment", string(cfg.Environment)).
		Int("chains", len(cfg.Chains)).
		Int("workers", cfg.Relayer.Workers).
		Msg("Configuration loaded")

	// Connect to database
	db, err := database.NewDB(&cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	logger.Info().Msg("Database connection established")

	// Create blockchain clients
	clientFactory := blockchain.NewClientFactory(logger)
	clients, err := clientFactory.CreateAllClients(context.Background(), cfg.Chains)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create blockchain clients")
	}
	defer blockchain.CloseAllClients(clients, logger)

	logger.Info().
		Int("clients", len(clients)).
		Msg("Blockchain clients initialized")

	// Connect to message queue
	q, err := queue.NewNATSQueue(&cfg.Queue, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to message queue")
	}
	defer q.Close()

	logger.Info().Msg("Message queue connected")

	// Create signers for each chain
	signers, err := createSigners(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create signers")
	}

	logger.Info().
		Int("signers", len(signers)).
		Msg("Signers initialized")

	// Create and start relayer
	relayer, err := relayer.NewRelayer(cfg, db, q, clients, signers, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create relayer")
	}

	// Start relayer workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := relayer.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start relayer")
	}

	logger.Info().
		Int("workers", cfg.Relayer.Workers).
		Msg("Relayer workers started")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info().Msg("Shutdown signal received")

	// Graceful shutdown
	cancel()
	if err := relayer.Stop(); err != nil {
		logger.Error().Err(err).Msg("Error stopping relayer")
	}

	logger.Info().Msg("Relayer service stopped")
}

func setupLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	env := os.Getenv("BRIDGE_ENVIRONMENT")
	if env == "development" || env == "testnet" {
		logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Caller().
			Logger()
		return logger
	}

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()
}

func createSigners(cfg *config.Config, logger zerolog.Logger) (map[string]crypto.UniversalSigner, error) {
	signers := make(map[string]crypto.UniversalSigner)

	for _, chain := range cfg.Chains {
		var signer crypto.UniversalSigner
		var err error

		switch chain.ChainType {
		case types.ChainTypeEVM:
			// In production, load private key from secure storage (HSM, KMS, etc.)
			// For now, we'll create a placeholder signer
			// You would use: evmCrypto.NewECDSASigner(privateKeyHex)
			logger.Warn().
				Str("chain", chain.Name).
				Msg("Using placeholder signer - configure proper key management in production")

			// Create a test signer for development
			// In production, replace with actual key loading
			signer, err = evmCrypto.NewECDSASignerFromPrivateKey("0000000000000000000000000000000000000000000000000000000000000001")
			if err != nil {
				return nil, fmt.Errorf("failed to create EVM signer for %s: %w", chain.Name, err)
			}

		case types.ChainTypeSolana, types.ChainTypeNEAR:
			// In production, load Ed25519 private key from secure storage
			logger.Warn().
				Str("chain", chain.Name).
				Msg("Using placeholder signer - configure proper key management in production")

			// Create a test signer for development
			signer, err = ed25519Crypto.GenerateKeyPair() // Generate new key pair
			if err != nil {
				return nil, fmt.Errorf("failed to create Ed25519 signer for %s: %w", chain.Name, err)
			}

		default:
			return nil, fmt.Errorf("unsupported chain type: %s", chain.ChainType)
		}

		signers[chain.Name] = signer
		logger.Info().
			Str("chain", chain.Name).
			Str("type", string(chain.ChainType)).
			Msg("Signer created")
	}

	return signers, nil
}
