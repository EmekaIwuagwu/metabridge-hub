package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/blockchain"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/listener/evm"
	nearlistener "github.com/EmekaIwuagwu/metabridge-hub/internal/listener/near"
	solanalistener "github.com/EmekaIwuagwu/metabridge-hub/internal/listener/solana"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/queue"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/rs/zerolog"
)

var (
	configPath = flag.String("config", "config/config.testnet.yaml", "Path to configuration file")
)

// EventListener is an interface that all listeners implement
type EventListener interface {
	EventChan() <-chan *types.CrossChainMessage
}

func main() {
	flag.Parse()

	// Setup logger
	logger := setupLogger()

	logger.Info().
		Str("service", "listener").
		Str("config", *configPath).
		Msg("Starting Metabridge Listener service")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	logger.Info().
		Str("environment", string(cfg.Environment)).
		Int("chains", len(cfg.Chains)).
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

	// Create and start listeners for each chain
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start listeners based on chain type
	for _, chainCfg := range cfg.Chains {
		switch chainCfg.ChainType {
		case types.ChainTypeEVM:
			evmClient, ok := clients[chainCfg.Name].(*blockchain.EVMClientAdapter)
			if !ok {
				logger.Fatal().
					Str("chain", chainCfg.Name).
					Msg("Failed to cast client to EVM client")
			}

			listener, err := evm.NewListener(evmClient.GetUnderlyingClient(), &chainCfg, logger)
			if err != nil {
				logger.Fatal().
					Err(err).
					Str("chain", chainCfg.Name).
					Msg("Failed to create EVM listener")
			}

			// Start listener
			if err := listener.Start(ctx); err != nil {
				logger.Fatal().
					Err(err).
					Str("chain", chainCfg.Name).
					Msg("Failed to start listener")
			}

			// Start event processor
			go processEvents(ctx, listener, q, db, logger, chainCfg.Name)

			logger.Info().
				Str("chain", chainCfg.Name).
				Msg("EVM listener started")

		case types.ChainTypeSolana:
			solanaClient, ok := clients[chainCfg.Name].(*blockchain.SolanaClientAdapter)
			if !ok {
				logger.Fatal().
					Str("chain", chainCfg.Name).
					Msg("Failed to cast client to Solana client")
			}

			listener, err := solanalistener.NewListener(solanaClient.GetUnderlyingClient(), &chainCfg, logger)
			if err != nil {
				logger.Fatal().
					Err(err).
					Str("chain", chainCfg.Name).
					Msg("Failed to create Solana listener")
			}

			// Start listener
			if err := listener.Start(ctx); err != nil {
				logger.Fatal().
					Err(err).
					Str("chain", chainCfg.Name).
					Msg("Failed to start listener")
			}

			// Start event processor
			go processEvents(ctx, listener, q, db, logger, chainCfg.Name)

			logger.Info().
				Str("chain", chainCfg.Name).
				Msg("Solana listener started")

		case types.ChainTypeNEAR:
			nearClient, ok := clients[chainCfg.Name].(*blockchain.NEARClientAdapter)
			if !ok {
				logger.Fatal().
					Str("chain", chainCfg.Name).
					Msg("Failed to cast client to NEAR client")
			}

			listener, err := nearlistener.NewListener(nearClient.GetUnderlyingClient(), &chainCfg, logger)
			if err != nil {
				logger.Fatal().
					Err(err).
					Str("chain", chainCfg.Name).
					Msg("Failed to create NEAR listener")
			}

			// Start listener
			if err := listener.Start(ctx); err != nil {
				logger.Fatal().
					Err(err).
					Str("chain", chainCfg.Name).
					Msg("Failed to start listener")
			}

			// Start event processor
			go processEvents(ctx, listener, q, db, logger, chainCfg.Name)

			logger.Info().
				Str("chain", chainCfg.Name).
				Msg("NEAR listener started")

		default:
			logger.Warn().
				Str("chain", chainCfg.Name).
				Str("type", string(chainCfg.ChainType)).
				Msg("Unsupported chain type")
		}
	}

	logger.Info().Msg("All listeners started")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info().Msg("Shutdown signal received")

	// Graceful shutdown
	cancel()
	logger.Info().Msg("Listener service stopped")
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

// processEvents processes events from a listener and publishes them to the queue
func processEvents(ctx context.Context, listener EventListener, q queue.Queue, db *database.DB, logger zerolog.Logger, chainName string) {
	eventLogger := logger.With().Str("chain", chainName).Str("component", "event-processor").Logger()
	eventLogger.Info().Msg("Event processor started")

	for {
		select {
		case <-ctx.Done():
			eventLogger.Info().Msg("Event processor stopped")
			return

		case msg, ok := <-listener.EventChan():
			if !ok {
				eventLogger.Warn().Msg("Event channel closed")
				return
			}

			// Save message to database
			if err := db.SaveMessage(ctx, msg); err != nil {
				eventLogger.Error().
					Err(err).
					Str("message_id", msg.ID).
					Msg("Failed to save message to database")
				continue
			}

			// Publish to queue
			if err := q.Publish(ctx, msg); err != nil {
				eventLogger.Error().
					Err(err).
					Str("message_id", msg.ID).
					Msg("Failed to publish message to queue")
				continue
			}

			eventLogger.Info().
				Str("message_id", msg.ID).
				Str("type", string(msg.Type)).
				Msg("Message published to queue")
		}
	}
}
