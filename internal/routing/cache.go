package routing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// RouteCache provides caching for discovered routes with TTL
type RouteCache struct {
	cache  map[string]*CachedRoute
	mu     sync.RWMutex
	ttl    time.Duration
	logger zerolog.Logger
}

// NewRouteCache creates a new route cache
func NewRouteCache(ttl time.Duration, logger zerolog.Logger) *RouteCache {
	return &RouteCache{
		cache:  make(map[string]*CachedRoute),
		ttl:    ttl,
		logger: logger.With().Str("component", "route-cache").Logger(),
	}
}

// Get retrieves routes from cache if they exist and haven't expired
func (rc *RouteCache) Get(query *RouteQuery) ([]*Route, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	key := rc.generateKey(query)
	cached, exists := rc.cache[key]

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().UTC().After(cached.ExpiresAt) {
		return nil, false
	}

	// Increment hit count
	cached.HitCount++

	rc.logger.Debug().
		Str("key", key).
		Int("hit_count", cached.HitCount).
		Msg("Cache hit")

	RecordRouteCacheHit()

	return cached.Routes, true
}

// Set stores routes in cache
func (rc *RouteCache) Set(query *RouteQuery, routes []*Route) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	key := rc.generateKey(query)
	now := time.Now().UTC()

	cached := &CachedRoute{
		Query:     query,
		Routes:    routes,
		CachedAt:  now,
		ExpiresAt: now.Add(rc.ttl),
		HitCount:  0,
	}

	rc.cache[key] = cached

	rc.logger.Debug().
		Str("key", key).
		Int("routes", len(routes)).
		Time("expires_at", cached.ExpiresAt).
		Msg("Routes cached")

	RecordRouteCacheSet()
}

// Invalidate removes a specific cache entry
func (rc *RouteCache) Invalidate(query *RouteQuery) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	key := rc.generateKey(query)
	delete(rc.cache, key)

	rc.logger.Debug().
		Str("key", key).
		Msg("Cache entry invalidated")
}

// InvalidateChainPair invalidates all cache entries involving a chain pair
func (rc *RouteCache) InvalidateChainPair(sourceChain, destChain string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	removed := 0
	for key, cached := range rc.cache {
		// Check if any hop involves this chain pair
		for _, route := range cached.Routes {
			for _, hop := range route.Hops {
				if hop.SourceChain == sourceChain && hop.DestChain == destChain {
					delete(rc.cache, key)
					removed++
					break
				}
			}
		}
	}

	if removed > 0 {
		rc.logger.Info().
			Str("source", sourceChain).
			Str("dest", destChain).
			Int("removed", removed).
			Msg("Cache entries invalidated for chain pair")
	}
}

// Clear removes all cache entries
func (rc *RouteCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache = make(map[string]*CachedRoute)

	rc.logger.Info().Msg("Cache cleared")
}

// CleanExpired removes expired cache entries
func (rc *RouteCache) CleanExpired() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now().UTC()
	removed := 0

	for key, cached := range rc.cache {
		if now.After(cached.ExpiresAt) {
			delete(rc.cache, key)
			removed++
		}
	}

	if removed > 0 {
		rc.logger.Debug().
			Int("removed", removed).
			Msg("Expired cache entries cleaned")
	}
}

// StartPeriodicCleanup starts periodic cleanup of expired entries
func (rc *RouteCache) StartPeriodicCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			rc.logger.Info().Msg("Stopping cache cleanup")
			return
		case <-ticker.C:
			rc.CleanExpired()
		}
	}
}

// GetStats returns cache statistics
func (rc *RouteCache) GetStats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	totalHits := 0
	for _, cached := range rc.cache {
		totalHits += cached.HitCount
	}

	return map[string]interface{}{
		"total_entries": len(rc.cache),
		"total_hits":    totalHits,
		"ttl_seconds":   rc.ttl.Seconds(),
	}
}

// generateKey generates a cache key from a route query
func (rc *RouteCache) generateKey(query *RouteQuery) string {
	// Create a deterministic key from query parameters
	data, _ := json.Marshal(map[string]interface{}{
		"source_chain": query.SourceChain,
		"dest_chain":   query.DestChain,
		"amount":       query.Amount.String(),
		"max_hops":     query.MaxHops,
		"optimize_for": query.OptimizeFor,
	})

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// LiquidityTracker tracks liquidity for chain pairs
type LiquidityTracker struct {
	liquidity map[string]*LiquidityInfo
	mu        sync.RWMutex
	logger    zerolog.Logger
}

// NewLiquidityTracker creates a new liquidity tracker
func NewLiquidityTracker(logger zerolog.Logger) *LiquidityTracker {
	return &LiquidityTracker{
		liquidity: make(map[string]*LiquidityInfo),
		logger:    logger.With().Str("component", "liquidity-tracker").Logger(),
	}
}

// UpdateLiquidity updates liquidity information for a chain pair
func (lt *LiquidityTracker) UpdateLiquidity(sourceChain, destChain string, total, available, reserved *big.Int) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	key := lt.pairKey(sourceChain, destChain)

	info := &LiquidityInfo{
		ChainPair:          key,
		SourceChain:        sourceChain,
		DestChain:          destChain,
		TotalLiquidity:     total,
		AvailableLiquidity: available,
		ReservedLiquidity:  reserved,
		LastUpdated:        time.Now().UTC(),
		UpdateInterval:     5 * time.Minute,
	}

	lt.liquidity[key] = info

	lt.logger.Debug().
		Str("chain_pair", key).
		Str("available", available.String()).
		Msg("Liquidity updated")

	RecordLiquidityUpdate(sourceChain, destChain)
}

// GetLiquidity retrieves liquidity information for a chain pair
func (lt *LiquidityTracker) GetLiquidity(sourceChain, destChain string) (*LiquidityInfo, bool) {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	key := lt.pairKey(sourceChain, destChain)
	info, exists := lt.liquidity[key]

	return info, exists
}

// CheckAvailability checks if sufficient liquidity is available
func (lt *LiquidityTracker) CheckAvailability(sourceChain, destChain string, amount *big.Int) bool {
	info, exists := lt.GetLiquidity(sourceChain, destChain)
	if !exists {
		return false
	}

	return info.AvailableLiquidity.Cmp(amount) >= 0
}

// ReserveLiquidity reserves liquidity for a transaction
func (lt *LiquidityTracker) ReserveLiquidity(sourceChain, destChain string, amount *big.Int) error {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	key := lt.pairKey(sourceChain, destChain)
	info, exists := lt.liquidity[key]

	if !exists {
		return fmt.Errorf("liquidity info not found for %s", key)
	}

	if info.AvailableLiquidity.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient liquidity: available %s, required %s",
			info.AvailableLiquidity.String(), amount.String())
	}

	// Update liquidity
	info.AvailableLiquidity.Sub(info.AvailableLiquidity, amount)
	info.ReservedLiquidity.Add(info.ReservedLiquidity, amount)
	info.LastUpdated = time.Now().UTC()

	lt.logger.Info().
		Str("chain_pair", key).
		Str("amount", amount.String()).
		Str("remaining", info.AvailableLiquidity.String()).
		Msg("Liquidity reserved")

	return nil
}

// ReleaseLiquidity releases reserved liquidity
func (lt *LiquidityTracker) ReleaseLiquidity(sourceChain, destChain string, amount *big.Int) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	key := lt.pairKey(sourceChain, destChain)
	info, exists := lt.liquidity[key]

	if !exists {
		return
	}

	// Update liquidity
	info.AvailableLiquidity.Add(info.AvailableLiquidity, amount)
	info.ReservedLiquidity.Sub(info.ReservedLiquidity, amount)
	info.LastUpdated = time.Now().UTC()

	lt.logger.Debug().
		Str("chain_pair", key).
		Str("amount", amount.String()).
		Msg("Liquidity released")
}

// GetAllLiquidity returns all liquidity information
func (lt *LiquidityTracker) GetAllLiquidity() map[string]*LiquidityInfo {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	// Return a copy
	result := make(map[string]*LiquidityInfo)
	for k, v := range lt.liquidity {
		result[k] = v
	}

	return result
}

// RefreshLiquidity refreshes liquidity from on-chain data
func (lt *LiquidityTracker) RefreshLiquidity(ctx context.Context) error {
	// In production, this would query actual on-chain liquidity
	// For now, we'll use placeholder values

	lt.logger.Debug().Msg("Refreshing liquidity data")

	// Default chain pairs with placeholder liquidity
	chainPairs := [][2]string{
		{"polygon", "ethereum"},
		{"polygon", "bsc"},
		{"polygon", "avalanche"},
		{"ethereum", "bsc"},
		{"ethereum", "avalanche"},
		{"bsc", "avalanche"},
	}

	for _, pair := range chainPairs {
		source := pair[0]
		dest := pair[1]

		// Placeholder values
		total := big.NewInt(1e20)      // 100 ETH
		available := big.NewInt(8e19)  // 80 ETH
		reserved := big.NewInt(2e19)   // 20 ETH

		lt.UpdateLiquidity(source, dest, total, available, reserved)
		lt.UpdateLiquidity(dest, source, total, available, reserved) // Bidirectional
	}

	return nil
}

// StartPeriodicRefresh starts periodic liquidity refresh
func (lt *LiquidityTracker) StartPeriodicRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial refresh
	lt.RefreshLiquidity(ctx)

	for {
		select {
		case <-ctx.Done():
			lt.logger.Info().Msg("Stopping liquidity refresh")
			return
		case <-ticker.C:
			if err := lt.RefreshLiquidity(ctx); err != nil {
				lt.logger.Error().
					Err(err).
					Msg("Failed to refresh liquidity")
			}
		}
	}
}

// pairKey generates a key for a chain pair
func (lt *LiquidityTracker) pairKey(sourceChain, destChain string) string {
	return fmt.Sprintf("%s-%s", sourceChain, destChain)
}
