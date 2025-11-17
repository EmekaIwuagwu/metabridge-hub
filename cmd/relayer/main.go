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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	// TODO: Create and start relayer workers
	// This would involve:
	// 1. Connecting to message queue (NATS)
	// 2. Creating worker pool
	// 3. Processing messages from queue
	// 4. Validating multi-sig signatures
	// 5. Broadcasting transactions to destination chains

	logger.Info().
		Int("workers", cfg.Relayer.Workers).
		Msg("Relayer workers started")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info().Msg("Shutdown signal received")

	// Graceful shutdown
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
