package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/EmekaIwuagwu/articium-hub/internal/config"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

// DB represents the database connection
type DB struct {
	*sql.DB
	logger zerolog.Logger
}

// NewDB creates a new database connection
func NewDB(cfg *config.DatabaseConfig, logger zerolog.Logger) (*DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxLifetime != "" {
		lifetime, err := time.ParseDuration(cfg.MaxLifetime)
		if err == nil {
			db.SetConnMaxLifetime(lifetime)
		}
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Msg("Database connection established")

	return &DB{
		DB:     db,
		logger: logger.With().Str("component", "database").Logger(),
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	db.logger.Info().Msg("Closing database connection")
	return db.DB.Close()
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
