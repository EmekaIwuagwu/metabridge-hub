package routing

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service provides high-level routing functionality
type Service struct {
	graphBuilder     *GraphBuilder
	routeFinder      *RouteFinder
	cache            *RouteCache
	liquidityTracker *LiquidityTracker
	db               *database.DB
	config           *RouteOptimizationConfig
	logger           zerolog.Logger
}

// NewService creates a new routing service
func NewService(
	db *database.DB,
	config *RouteOptimizationConfig,
	logger zerolog.Logger,
) *Service {
	if config == nil {
		config = DefaultOptimizationConfig()
	}

	graphBuilder := NewGraphBuilder(db, logger)
	liquidityTracker := NewLiquidityTracker(logger)
	cache := NewRouteCache(config.CacheTTL, logger)

	return &Service{
		graphBuilder:     graphBuilder,
		liquidityTracker: liquidityTracker,
		cache:            cache,
		db:               db,
		config:           config,
		logger:           logger.With().Str("component", "routing-service").Logger(),
	}
}

// Start starts the routing service
func (s *Service) Start(ctx context.Context) error {
	s.logger.Info().Msg("Starting routing service")

	// Build initial graph
	if err := s.graphBuilder.BuildGraph(ctx); err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	// Initialize route finder with graph
	graph := s.graphBuilder.GetGraph()
	s.routeFinder = NewRouteFinder(graph, s.config, s.logger)

	// Start periodic graph refresh (every 5 minutes)
	go s.graphBuilder.StartPeriodicRefresh(ctx, 5*time.Minute)

	// Start periodic liquidity refresh (every 2 minutes)
	go s.liquidityTracker.StartPeriodicRefresh(ctx, 2*time.Minute)

	// Start periodic cache cleanup (every 10 minutes)
	if s.config.CacheEnabled {
		go s.cache.StartPeriodicCleanup(ctx, 10*time.Minute)
	}

	// Update metrics
	s.updateMetrics()

	s.logger.Info().Msg("Routing service started successfully")

	return nil
}

// FindRoutes discovers optimal routes between chains
func (s *Service) FindRoutes(ctx context.Context, query *RouteQuery) (*RouteResult, error) {
	start := time.Now()

	// Check cache first
	if s.config.CacheEnabled {
		if routes, found := s.cache.Get(query); found {
			result := &RouteResult{
				Routes:           routes,
				RecommendedRoute: routes[0],
				Count:            len(routes),
				SearchTime:       0,
				Timestamp:        time.Now().UTC(),
			}
			return result, nil
		}
		RecordRouteCacheMiss()
	}

	// Get latest graph
	graph := s.graphBuilder.GetGraph()
	s.routeFinder.graph = graph

	// Enrich query with liquidity information
	s.enrichQueryWithLiquidity(query)

	// Find routes
	result, err := s.routeFinder.FindRoutes(ctx, query)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if s.config.CacheEnabled && len(result.Routes) > 0 {
		s.cache.Set(query, result.Routes)
	}

	// Save routes to database
	for _, route := range result.Routes {
		if err := s.saveRoute(ctx, route); err != nil {
			s.logger.Warn().
				Err(err).
				Str("route_id", route.ID).
				Msg("Failed to save route")
		}
	}

	// Record metrics
	RecordRouteDiscoveryLatency(time.Since(start).Seconds())

	if result.RecommendedRoute != nil {
		RecordRouteScore(result.RecommendedRoute.Score)
		RecordOptimalRoute()
	}

	return result, nil
}

// ExecuteRoute executes a multi-hop route
func (s *Service) ExecuteRoute(ctx context.Context, routeID string) (*RouteExecution, error) {
	// Get route from database
	route, err := s.getRoute(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("route not found: %w", err)
	}

	s.logger.Info().
		Str("route_id", routeID).
		Int("hops", len(route.Hops)).
		Msg("Starting route execution")

	RecordRouteExecution()

	execution := &RouteExecution{
		RouteID:       routeID,
		CurrentHop:    0,
		TotalHops:     len(route.Hops),
		Status:        RouteStatusExecuting,
		CompletedHops: []string{},
		FailedHops:    []string{},
		StartedAt:     time.Now().UTC(),
		LastUpdate:    time.Now().UTC(),
	}

	// Execute each hop
	for i, hop := range route.Hops {
		execution.CurrentHop = i + 1

		s.logger.Info().
			Int("hop", i+1).
			Int("total", len(route.Hops)).
			Str("source", hop.SourceChain).
			Str("dest", hop.DestChain).
			Msg("Executing hop")

		// Try to reserve liquidity
		if hop.Liquidity != nil {
			if err := s.liquidityTracker.ReserveLiquidity(hop.SourceChain, hop.DestChain, hop.Liquidity); err != nil {
				s.logger.Error().
					Err(err).
					Str("source", hop.SourceChain).
					Str("dest", hop.DestChain).
					Msg("Failed to reserve liquidity")

				execution.Status = RouteStatusFailed
				execution.ErrorMessage = fmt.Sprintf("insufficient liquidity at hop %d", i+1)
				RecordInsufficientLiquidity(hop.SourceChain, hop.DestChain)
				RecordRouteFailed()
				return execution, err
			}
			RecordLiquidityReservation()
		}

		// Execute hop (this would call the actual bridge transaction)
		if err := s.executeHop(ctx, &hop); err != nil {
			s.logger.Error().
				Err(err).
				Int("hop", i+1).
				Msg("Hop execution failed")

			execution.FailedHops = append(execution.FailedHops, hop.DestChain)
			execution.Status = RouteStatusFailed
			execution.ErrorMessage = fmt.Sprintf("hop %d failed: %s", i+1, err.Error())

			// Release reserved liquidity
			if hop.Liquidity != nil {
				s.liquidityTracker.ReleaseLiquidity(hop.SourceChain, hop.DestChain, hop.Liquidity)
				RecordLiquidityRelease()
			}

			RecordHopFailed()
			RecordRouteFailed()
			return execution, err
		}

		execution.CompletedHops = append(execution.CompletedHops, hop.DestChain)
		execution.LastUpdate = time.Now().UTC()
		RecordHopCompleted()

		s.logger.Info().
			Int("hop", i+1).
			Msg("Hop completed successfully")
	}

	// All hops completed
	execution.Status = RouteStatusCompleted
	route.Status = RouteStatusCompleted
	now := time.Now().UTC()
	route.CompletedAt = &now

	// Update route in database
	s.updateRoute(ctx, route)

	// Record completion metrics
	executionTime := time.Since(execution.StartedAt).Seconds()
	totalCostFloat, _ := new(big.Float).SetInt(route.TotalCost).Float64()
	RecordRouteCompleted(len(route.Hops), executionTime, totalCostFloat)

	s.logger.Info().
		Str("route_id", routeID).
		Int("hops", len(route.Hops)).
		Float64("execution_time", executionTime).
		Msg("Route execution completed")

	return execution, nil
}

// GetRouteStatus returns the execution status of a route
func (s *Service) GetRouteStatus(ctx context.Context, routeID string) (*Route, error) {
	return s.getRoute(ctx, routeID)
}

// GetChainTopology returns the current routing topology
func (s *Service) GetChainTopology() *ChainGraph {
	return s.graphBuilder.GetChainTopology()
}

// GetLiquidityInfo returns liquidity information for all chain pairs
func (s *Service) GetLiquidityInfo() map[string]*LiquidityInfo {
	return s.liquidityTracker.GetAllLiquidity()
}

// GetCacheStats returns cache statistics
func (s *Service) GetCacheStats() map[string]interface{} {
	return s.cache.GetStats()
}

// InvalidateCache invalidates routes involving a chain pair
func (s *Service) InvalidateCache(sourceChain, destChain string) {
	s.cache.InvalidateChainPair(sourceChain, destChain)
}

// Private methods

func (s *Service) enrichQueryWithLiquidity(query *RouteQuery) {
	// Add liquidity information to help with route selection
	if info, exists := s.liquidityTracker.GetLiquidity(query.SourceChain, query.DestChain); exists {
		if query.MinLiquidity == nil && info.AvailableLiquidity != nil {
			query.MinLiquidity = info.AvailableLiquidity
		}
	}
}

func (s *Service) executeHop(ctx context.Context, hop *Hop) error {
	// In production, this would:
	// 1. Create a cross-chain message
	// 2. Submit it to the bridge
	// 3. Wait for confirmation
	// For now, simulate execution

	s.logger.Debug().
		Str("source", hop.SourceChain).
		Str("dest", hop.DestChain).
		Msg("Simulating hop execution")

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Generate mock transaction hash
	hop.MessageID = uuid.New().String()
	hop.TxHash = fmt.Sprintf("0x%s", uuid.New().String())
	hop.Status = "CONFIRMED"

	return nil
}

func (s *Service) saveRoute(ctx context.Context, route *Route) error {
	query := `
		INSERT INTO routes (
			id, source_chain, dest_chain, total_hops,
			total_cost, total_time_seconds, total_fee,
			score, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		route.ID,
		route.SourceChain,
		route.DestChain,
		len(route.Hops),
		route.TotalCost.String(),
		int64(route.TotalTime.Seconds()),
		route.TotalFee.String(),
		route.Score,
		route.Status,
		route.CreatedAt,
		route.UpdatedAt,
	)

	return err
}

func (s *Service) getRoute(ctx context.Context, routeID string) (*Route, error) {
	query := `
		SELECT
			id, source_chain, dest_chain, total_hops,
			total_cost, total_time_seconds, total_fee,
			score, status, created_at, updated_at
		FROM routes
		WHERE id = $1
	`

	row := s.db.QueryRowContext(ctx, query, routeID)

	route := &Route{
		Hops: []Hop{},
	}

	var totalCostStr, totalFeeStr string
	var totalTimeSeconds int64

	err := row.Scan(
		&route.ID,
		&route.SourceChain,
		&route.DestChain,
		&totalTimeSeconds,
		&totalCostStr,
		&totalTimeSeconds,
		&totalFeeStr,
		&route.Score,
		&route.Status,
		&route.CreatedAt,
		&route.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	route.TotalCost, _ = new(big.Int).SetString(totalCostStr, 10)
	route.TotalFee, _ = new(big.Int).SetString(totalFeeStr, 10)
	route.TotalTime = time.Duration(totalTimeSeconds) * time.Second

	return route, nil
}

func (s *Service) updateRoute(ctx context.Context, route *Route) error {
	query := `
		UPDATE routes
		SET status = $2, updated_at = $3, completed_at = $4
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		route.ID,
		route.Status,
		route.UpdatedAt,
		route.CompletedAt,
	)

	return err
}

func (s *Service) updateMetrics() {
	graph := s.graphBuilder.GetGraph()
	nodeCount := len(graph.Nodes)

	edgeCount := 0
	for _, edges := range graph.Edges {
		edgeCount += len(edges)
	}

	SetGraphSize(nodeCount, edgeCount)

	// Update cache size
	stats := s.cache.GetStats()
	if entries, ok := stats["total_entries"].(int); ok {
		SetRouteCacheSize(entries)
	}

	// Update liquidity metrics
	for _, info := range s.liquidityTracker.GetAllLiquidity() {
		if info.AvailableLiquidity != nil {
			liquidityFloat, _ := new(big.Float).SetInt(info.AvailableLiquidity).Float64()
			RecordLiquidity(info.SourceChain, info.DestChain, liquidityFloat)
		}
	}
}
