package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/api"
	"github.com/EmekaIwuagwu/articium-hub/internal/blockchain"
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
		Str("service", "api").
		Str("config", *configPath).
		Msg("Starting Articium API server")

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

	// Create API server
	server := api.NewServer(cfg, db, clients, logger)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Fatal().Err(err).Msg("API server failed")
		}
	}()

	logger.Info().
		Str("address", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)).
		Msg("API server started")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info().Msg("Shutdown signal received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}

	logger.Info().Msg("API server stopped")
}

func setupLogger() zerolog.Logger {
	// Use JSON logging in production
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
