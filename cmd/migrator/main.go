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
	schemaPath = flag.String("schema", "internal/database/schema.sql", "Path to schema SQL file")
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

	// Read schema file
	schema, err := os.ReadFile(*schemaPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to read schema file")
	}

	logger.Info().
		Str("schema_file", *schemaPath).
		Msg("Schema file loaded")

	// Execute schema
	if _, err := db.Exec(string(schema)); err != nil {
		logger.Fatal().Err(err).Msg("Failed to execute schema")
	}

	logger.Info().Msg("Database schema applied successfully")

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
