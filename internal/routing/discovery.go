package routing

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RouteFinder discovers optimal routes between chains
type RouteFinder struct {
	graph   *Graph
	config  *RouteOptimizationConfig
	logger  zerolog.Logger
}

// NewRouteFinder creates a new route finder
func NewRouteFinder(graph *Graph, config *RouteOptimizationConfig, logger zerolog.Logger) *RouteFinder {
	if config == nil {
		config = DefaultOptimizationConfig()
	}

	return &RouteFinder{
		graph:  graph,
		config: config,
		logger: logger.With().Str("component", "route-finder").Logger(),
	}
}

// FindRoutes discovers all possible routes between source and destination chains
func (rf *RouteFinder) FindRoutes(ctx context.Context, query *RouteQuery) (*RouteResult, error) {
	startTime := time.Now()

	// Validate query
	if err := rf.validateQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// Check if direct route exists
	directRoute := rf.findDirectRoute(query)

	routes := []*Route{}
	if directRoute != nil {
		routes = append(routes, directRoute)
	}

	// Find multi-hop routes if max hops > 1
	maxHops := query.MaxHops
	if maxHops == 0 {
		maxHops = rf.config.MaxHops
	}

	if maxHops > 1 {
		multiHopRoutes := rf.findMultiHopRoutes(ctx, query, maxHops)
		routes = append(routes, multiHopRoutes...)
	}

	// Score and sort routes
	rf.scoreRoutes(routes, query)
	routes = rf.sortRoutes(routes)

	// Limit number of routes
	if len(routes) > rf.config.MaxRoutesToReturn {
		routes = routes[:rf.config.MaxRoutesToReturn]
	}

	// Select recommended route
	var recommendedRoute *Route
	if len(routes) > 0 {
		recommendedRoute = routes[0]
	}

	result := &RouteResult{
		Routes:           routes,
		RecommendedRoute: recommendedRoute,
		Count:            len(routes),
		SearchTime:       time.Since(startTime).Milliseconds(),
		Timestamp:        time.Now().UTC(),
	}

	rf.logger.Info().
		Str("source", query.SourceChain).
		Str("dest", query.DestChain).
		Int("routes_found", len(routes)).
		Int64("search_time_ms", result.SearchTime).
		Msg("Route search completed")

	RecordRouteDiscovery(query.SourceChain, query.DestChain, len(routes))

	return result, nil
}

// findDirectRoute checks for a direct connection between chains
func (rf *RouteFinder) findDirectRoute(query *RouteQuery) *Route {
	edge, exists := rf.graph.Edges[query.SourceChain][query.DestChain]
	if !exists || edge == nil {
		return nil
	}

	// Check if liquidity is sufficient
	if query.Amount != nil && edge.Liquidity != nil {
		if edge.Liquidity.Cmp(query.Amount) < 0 {
			rf.logger.Debug().
				Str("source", query.SourceChain).
				Str("dest", query.DestChain).
				Msg("Insufficient liquidity for direct route")
			return nil
		}
	}

	hop := Hop{
		Step:          1,
		SourceChain:   query.SourceChain,
		DestChain:     query.DestChain,
		EstimatedCost: edge.Cost,
		EstimatedTime: edge.Time,
		GasPrice:      big.NewInt(0),
		Liquidity:     edge.Liquidity,
		Status:        "PENDING",
	}

	route := &Route{
		ID:          uuid.New().String(),
		SourceChain: query.SourceChain,
		DestChain:   query.DestChain,
		Hops:        []Hop{hop},
		TotalCost:   edge.Cost,
		TotalTime:   time.Duration(edge.Time) * time.Second,
		TotalFee:    edge.Cost,
		Status:      RouteStatusPending,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	return route
}

// findMultiHopRoutes finds all multi-hop routes using modified Dijkstra's algorithm
func (rf *RouteFinder) findMultiHopRoutes(ctx context.Context, query *RouteQuery, maxHops int) []*Route {
	routes := []*Route{}

	// Use k-shortest paths algorithm to find multiple routes
	paths := rf.kShortestPaths(query.SourceChain, query.DestChain, maxHops, rf.config.MaxRoutesToReturn)

	for _, path := range paths {
		route := rf.pathToRoute(path, query)
		if route != nil {
			// Check constraints
			if rf.meetsConstraints(route, query) {
				routes = append(routes, route)
			}
		}
	}

	return routes
}

// kShortestPaths finds k shortest paths using Yen's algorithm (simplified)
func (rf *RouteFinder) kShortestPaths(source, dest string, maxHops, k int) [][]string {
	paths := [][]string{}

	// Find first shortest path
	shortestPath := rf.dijkstraPath(source, dest, maxHops)
	if shortestPath == nil {
		return paths
	}

	paths = append(paths, shortestPath)

	// Find k-1 more paths by exploring alternative routes
	for len(paths) < k {
		newPath := rf.findAlternativePath(source, dest, paths, maxHops)
		if newPath == nil {
			break
		}
		paths = append(paths, newPath)
	}

	return paths
}

// dijkstraPath finds shortest path using Dijkstra's algorithm
func (rf *RouteFinder) dijkstraPath(source, dest string, maxHops int) []string {
	// Initialize distances and previous nodes
	distances := make(map[string]float64)
	previous := make(map[string]string)
	visited := make(map[string]bool)
	hopCount := make(map[string]int)

	// Initialize all distances to infinity
	for node := range rf.graph.Nodes {
		distances[node] = math.Inf(1)
	}
	distances[source] = 0
	hopCount[source] = 0

	// Priority queue for Dijkstra's algorithm
	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &Item{
		value:    source,
		priority: 0,
	})

	for pq.Len() > 0 {
		current := heap.Pop(pq).(*Item).value

		if visited[current] {
			continue
		}
		visited[current] = true

		// Found destination
		if current == dest {
			return rf.reconstructPath(previous, source, dest)
		}

		// Check hop limit
		if hopCount[current] >= maxHops {
			continue
		}

		// Explore neighbors
		if edges, exists := rf.graph.Edges[current]; exists {
			for neighbor, edge := range edges {
				if visited[neighbor] {
					continue
				}

				// Calculate new distance
				newDistance := distances[current] + edge.Weight
				newHops := hopCount[current] + 1

				if newDistance < distances[neighbor] && newHops <= maxHops {
					distances[neighbor] = newDistance
					previous[neighbor] = current
					hopCount[neighbor] = newHops

					heap.Push(pq, &Item{
						value:    neighbor,
						priority: newDistance,
					})
				}
			}
		}
	}

	// No path found
	return nil
}

// findAlternativePath finds an alternative path by excluding edges from existing paths
func (rf *RouteFinder) findAlternativePath(source, dest string, existingPaths [][]string, maxHops int) []string {
	// Try to find path by temporarily removing edges from existing paths
	// This is a simplified version - a full implementation would use Yen's algorithm

	// For now, just try different intermediate nodes
	for intermediate := range rf.graph.Nodes {
		if intermediate == source || intermediate == dest {
			continue
		}

		// Check if this intermediate node creates a new path
		pathToIntermediate := rf.dijkstraPath(source, intermediate, maxHops/2)
		pathFromIntermediate := rf.dijkstraPath(intermediate, dest, maxHops/2)

		if pathToIntermediate != nil && pathFromIntermediate != nil {
			// Combine paths
			combinedPath := append(pathToIntermediate, pathFromIntermediate[1:]...)

			// Check if this path is different from existing ones
			if !rf.pathExists(combinedPath, existingPaths) && len(combinedPath) <= maxHops+1 {
				return combinedPath
			}
		}
	}

	return nil
}

// reconstructPath reconstructs the path from previous nodes map
func (rf *RouteFinder) reconstructPath(previous map[string]string, source, dest string) []string {
	path := []string{}
	current := dest

	for current != "" {
		path = append([]string{current}, path...)
		if current == source {
			break
		}
		current = previous[current]
	}

	if path[0] != source {
		return nil
	}

	return path
}

// pathExists checks if a path already exists in the list
func (rf *RouteFinder) pathExists(path []string, existingPaths [][]string) bool {
	for _, existingPath := range existingPaths {
		if rf.pathsEqual(path, existingPath) {
			return true
		}
	}
	return false
}

// pathsEqual checks if two paths are equal
func (rf *RouteFinder) pathsEqual(path1, path2 []string) bool {
	if len(path1) != len(path2) {
		return false
	}
	for i := range path1 {
		if path1[i] != path2[i] {
			return false
		}
	}
	return true
}

// pathToRoute converts a path (list of chain IDs) to a Route object
func (rf *RouteFinder) pathToRoute(path []string, query *RouteQuery) *Route {
	if len(path) < 2 {
		return nil
	}

	hops := []Hop{}
	totalCost := big.NewInt(0)
	totalTime := int64(0)

	for i := 0; i < len(path)-1; i++ {
		sourceChain := path[i]
		destChain := path[i+1]

		edge, exists := rf.graph.Edges[sourceChain][destChain]
		if !exists {
			return nil
		}

		hop := Hop{
			Step:          i + 1,
			SourceChain:   sourceChain,
			DestChain:     destChain,
			EstimatedCost: edge.Cost,
			EstimatedTime: edge.Time,
			GasPrice:      big.NewInt(0),
			Liquidity:     edge.Liquidity,
			Status:        "PENDING",
		}

		hops = append(hops, hop)
		totalCost.Add(totalCost, edge.Cost)
		totalTime += edge.Time
	}

	route := &Route{
		ID:          uuid.New().String(),
		SourceChain: query.SourceChain,
		DestChain:   query.DestChain,
		Hops:        hops,
		TotalCost:   totalCost,
		TotalTime:   time.Duration(totalTime) * time.Second,
		TotalFee:    totalCost,
		Status:      RouteStatusPending,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	return route
}

// meetsConstraints checks if a route meets the query constraints
func (rf *RouteFinder) meetsConstraints(route *Route, query *RouteQuery) bool {
	// Check max cost
	if query.MaxCost != nil && route.TotalCost.Cmp(query.MaxCost) > 0 {
		return false
	}

	// Check max time
	if query.MaxTime > 0 && route.TotalTime.Seconds() > float64(query.MaxTime) {
		return false
	}

	// Check liquidity for each hop
	if query.MinLiquidity != nil {
		for _, hop := range route.Hops {
			if hop.Liquidity != nil && hop.Liquidity.Cmp(query.MinLiquidity) < 0 {
				return false
			}
		}
	}

	return true
}

// validateQuery validates a route query
func (rf *RouteFinder) validateQuery(query *RouteQuery) error {
	if query.SourceChain == "" {
		return fmt.Errorf("source chain is required")
	}
	if query.DestChain == "" {
		return fmt.Errorf("destination chain is required")
	}
	if query.SourceChain == query.DestChain {
		return fmt.Errorf("source and destination chains must be different")
	}

	// Check if chains exist in graph
	if _, exists := rf.graph.Nodes[query.SourceChain]; !exists {
		return fmt.Errorf("source chain not found in graph")
	}
	if _, exists := rf.graph.Nodes[query.DestChain]; !exists {
		return fmt.Errorf("destination chain not found in graph")
	}

	return nil
}

// scoreRoutes calculates scores for all routes based on optimization criteria
func (rf *RouteFinder) scoreRoutes(routes []*Route, query *RouteQuery) {
	for _, route := range routes {
		route.Score = rf.calculateRouteScore(route, query)
	}
}

// calculateRouteScore calculates a score for a route based on multiple factors
func (rf *RouteFinder) calculateRouteScore(route *Route, query *RouteQuery) float64 {
	// Normalize factors to 0-1 range and apply weights

	// Cost score (lower is better)
	costScore := 1.0
	if route.TotalCost != nil && route.TotalCost.Cmp(big.NewInt(0)) > 0 {
		// Normalize to 0-1 (assuming max cost of 1 ETH = 1e18 wei)
		maxCost := new(big.Float).SetInt(big.NewInt(1e18))
		routeCost := new(big.Float).SetInt(route.TotalCost)
		costRatio, _ := new(big.Float).Quo(routeCost, maxCost).Float64()
		costScore = 1.0 - math.Min(costRatio, 1.0)
	}

	// Time score (faster is better)
	timeScore := 1.0
	if route.TotalTime > 0 {
		// Normalize to 0-1 (assuming max time of 1 hour)
		maxTime := float64(3600)
		timeRatio := route.TotalTime.Seconds() / maxTime
		timeScore = 1.0 - math.Min(timeRatio, 1.0)
	}

	// Hop count score (fewer hops is better)
	hopScore := 1.0 - (float64(len(route.Hops)) / float64(rf.config.MaxHops))

	// Liquidity score (higher is better)
	liquidityScore := 0.5 // Default middle score
	// This would be calculated based on actual liquidity data

	// Combine scores with weights
	weights := rf.getWeightsForOptimization(query.OptimizeFor)

	totalScore := costScore*weights.CostWeight +
		timeScore*weights.TimeWeight +
		hopScore*0.2 + // Fewer hops is always better
		liquidityScore*weights.LiquidityWeight

	return totalScore
}

// getWeightsForOptimization returns weights based on optimization preference
func (rf *RouteFinder) getWeightsForOptimization(optimizeFor string) *RouteOptimizationConfig {
	switch optimizeFor {
	case "cost":
		return &RouteOptimizationConfig{
			CostWeight:      0.6,
			TimeWeight:      0.2,
			LiquidityWeight: 0.2,
		}
	case "time":
		return &RouteOptimizationConfig{
			CostWeight:      0.2,
			TimeWeight:      0.6,
			LiquidityWeight: 0.2,
		}
	default: // "balanced"
		return rf.config
	}
}

// sortRoutes sorts routes by score (descending)
func (rf *RouteFinder) sortRoutes(routes []*Route) []*Route {
	// Simple bubble sort for small arrays
	n := len(routes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if routes[j].Score < routes[j+1].Score {
				routes[j], routes[j+1] = routes[j+1], routes[j]
			}
		}
	}
	return routes
}

// Priority Queue implementation for Dijkstra's algorithm

type Item struct {
	value    string
	priority float64
	index    int
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}
