package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	configPath = flag.String("config", "config/config.testnet.yaml", "Path to configuration file")
	schemaDir  = flag.String("schema-dir", "internal/database", "Directory containing schema SQL files")
)

func main() {
	flag.Parse()

	// Setup logger
	logger := setupLogger()

	logger.Info().
		Str("service", "migrator").
		Str("config", *configPath).
		Msg("Starting Metabridge Database Migrator")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	logger.Info().
		Str("environment", string(cfg.Environment)).
		Str("database", cfg.Database.Database).
		Msg("Configuration loaded")

	// Connect to database
	db, err := database.NewDB(&cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	logger.Info().Msg("Database connection established")

	// Execute schema files in order
	schemaFiles := []string{
		"schema.sql",      // Main tables (chains, messages, validators, etc.)
		"auth.sql",        // Authentication tables (users, api_keys)
		"batches.sql",     // Batch processing tables
		"routes.sql",      // Multi-hop routing tables
		"webhooks.sql",    // Webhook integration tables
	}

	for _, filename := range schemaFiles {
		schemaPath := fmt.Sprintf("%s/%s", *schemaDir, filename)

		logger.Info().
			Str("schema_file", schemaPath).
			Msg("Applying schema")

		schema, err := os.ReadFile(schemaPath)
		if err != nil {
			logger.Fatal().Err(err).Str("file", schemaPath).Msg("Failed to read schema file")
		}

		if _, err := db.Exec(string(schema)); err != nil {
			logger.Fatal().Err(err).Str("file", schemaPath).Msg("Failed to execute schema")
		}

		logger.Info().
			Str("schema_file", schemaPath).
			Msg("Schema applied successfully")
	}

	logger.Info().Msg("All database schemas applied successfully")
	fmt.Println("✓ Database migration completed successfully")
}

func setupLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Logger()

	return logger
}
