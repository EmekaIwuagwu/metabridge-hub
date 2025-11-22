package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/rs/zerolog"
)

// RateLimiter implements token bucket rate limiting per address
type RateLimiter struct {
	config *config.SecurityConfig
	logger zerolog.Logger

	// In-memory tracking (in production, use Redis)
	limits map[string]*AddressLimit
	mu     sync.RWMutex
}

// AddressLimit tracks rate limit for a specific address
type AddressLimit struct {
	HourlyCount     int
	HourlyResetTime time.Time
	DailyCount      int
	DailyResetTime  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *config.SecurityConfig, logger zerolog.Logger) *RateLimiter {
	limiter := &RateLimiter{
		config: config,
		logger: logger.With().Str("component", "rate_limiter").Logger(),
		limits: make(map[string]*AddressLimit),
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// CheckRateLimit checks if an address is within rate limits
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, address string) error {
	if !rl.config.EnableRateLimiting {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get or create limit for address
	limit, exists := rl.limits[address]
	if !exists {
		limit = &AddressLimit{
			HourlyResetTime: time.Now().Add(time.Hour),
			DailyResetTime:  time.Now().Add(24 * time.Hour),
		}
		rl.limits[address] = limit
	}

	// Reset hourly count if needed
	if time.Now().After(limit.HourlyResetTime) {
		limit.HourlyCount = 0
		limit.HourlyResetTime = time.Now().Add(time.Hour)
	}

	// Reset daily count if needed
	if time.Now().After(limit.DailyResetTime) {
		limit.DailyCount = 0
		limit.DailyResetTime = time.Now().Add(24 * time.Hour)
	}

	// Check hourly limit
	if rl.config.RateLimitPerHour > 0 && limit.HourlyCount >= rl.config.RateLimitPerHour {
		return fmt.Errorf("hourly rate limit exceeded: %d/%d",
			limit.HourlyCount, rl.config.RateLimitPerHour)
	}

	// Check per-address limit if configured
	if rl.config.RateLimitPerAddress > 0 && limit.HourlyCount >= rl.config.RateLimitPerAddress {
		return fmt.Errorf("address rate limit exceeded: %d/%d",
			limit.HourlyCount, rl.config.RateLimitPerAddress)
	}

	// Increment counters
	limit.HourlyCount++
	limit.DailyCount++

	rl.logger.Debug().
		Str("address", address).
		Int("hourly_count", limit.HourlyCount).
		Int("daily_count", limit.DailyCount).
		Msg("Rate limit check passed")

	return nil
}

// GetLimitInfo returns current limit info for an address
func (rl *RateLimiter) GetLimitInfo(address string) *AddressLimit {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limit, exists := rl.limits[address]
	if !exists {
		return &AddressLimit{
			HourlyResetTime: time.Now().Add(time.Hour),
			DailyResetTime:  time.Now().Add(24 * time.Hour),
		}
	}

	return &AddressLimit{
		HourlyCount:     limit.HourlyCount,
		HourlyResetTime: limit.HourlyResetTime,
		DailyCount:      limit.DailyCount,
		DailyResetTime:  limit.DailyResetTime,
	}
}

// ResetLimit manually resets limits for an address (admin function)
func (rl *RateLimiter) ResetLimit(address string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.limits, address)

	rl.logger.Info().
		Str("address", address).
		Msg("Rate limit reset")
}

// cleanup periodically removes expired entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for address, limit := range rl.limits {
			// Remove if both daily and hourly are expired
			if now.After(limit.DailyResetTime) && now.After(limit.HourlyResetTime) {
				delete(rl.limits, address)
			}
		}
		rl.mu.Unlock()

		rl.logger.Debug().
			Int("active_limits", len(rl.limits)).
			Msg("Rate limit cleanup completed")
	}
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"tracked_addresses":     len(rl.limits),
		"rate_limiting_enabled": rl.config.EnableRateLimiting,
		"hourly_limit":          rl.config.RateLimitPerHour,
		"per_address_limit":     rl.config.RateLimitPerAddress,
	}
}
