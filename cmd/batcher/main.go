package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/batching"
	"github.com/EmekaIwuagwu/articium-hub/internal/config"
	"github.com/EmekaIwuagwu/articium-hub/internal/database"
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
		Str("service", "batcher").
		Str("config", *configPath).
		Msg("Starting Articium Batch Aggregator")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	logger.Info().
		Str("environment", string(cfg.Environment)).
		Msg("Configuration loaded")

	// Connect to database
	db, err := database.NewDB(&cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	logger.Info().Msg("Database connection established")

	// Create batch configuration
	batchConfig := &batching.BatchConfig{
		MaxBatchSize:          100,
		MinBatchSize:          5,
		MaxWaitTime:           30 * time.Second,
		MinSubmissionInterval: 10 * time.Second,
		EnabledChainPairs:     make(map[string]bool),
	}

	// Enable batching for all configured chain pairs
	for _, chain := range cfg.Chains {
		for _, destChain := range cfg.Chains {
			if chain.Name != destChain.Name {
				key := chain.Name + "-" + destChain.Name
				batchConfig.EnabledChainPairs[key] = true
				logger.Info().
					Str("source", chain.Name).
					Str("dest", destChain.Name).
					Msg("Enabled batching for chain pair")
			}
		}
	}

	// Create aggregator
	aggregator := batching.NewAggregator(batchConfig, db, logger)

	// Start aggregator
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := aggregator.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start aggregator")
	}

	logger.Info().Msg("Batch aggregator started successfully")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Print stats periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			logger.Info().Msg("Shutdown signal received")
			cancel()

			// Graceful shutdown
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()

			if err := aggregator.Stop(shutdownCtx); err != nil {
				logger.Error().Err(err).Msg("Error during shutdown")
			}

			logger.Info().Msg("Batcher stopped")
			return

		case <-ticker.C:
			// Print statistics
			stats := aggregator.GetBatchStats()
			logger.Info().
				Int("pending_batches", stats.PendingBatchCount).
				Int("pending_messages", stats.PendingMessageCount).
				Str("total_value_locked", stats.TotalValueLocked.String()).
				Msg("Batch aggregator statistics")
		}
	}
}

func setupLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	env := os.Getenv("BRIDGE_ENVIRONMENT")
	if env == "development" || env == "testnet" {
		// Pretty logging for development
		logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Caller().
			Logger()
		return logger
	}

	// Structured JSON logging for production
	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()
}
