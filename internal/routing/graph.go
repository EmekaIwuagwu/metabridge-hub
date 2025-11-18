package routing

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/rs/zerolog"
)

// GraphBuilder builds and maintains the chain routing graph
type GraphBuilder struct {
	graph  *Graph
	db     *database.DB
	logger zerolog.Logger
	mu     sync.RWMutex
}

// NewGraphBuilder creates a new graph builder
func NewGraphBuilder(db *database.DB, logger zerolog.Logger) *GraphBuilder {
	return &GraphBuilder{
		graph: &Graph{
			Nodes: make(map[string]*Node),
			Edges: make(map[string]map[string]*Edge),
		},
		db:     db,
		logger: logger.With().Str("component", "graph-builder").Logger(),
	}
}

// BuildGraph builds the initial routing graph from database
func (gb *GraphBuilder) BuildGraph(ctx context.Context) error {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	gb.logger.Info().Msg("Building routing graph")

	// Load active chains as nodes
	if err := gb.loadNodes(ctx); err != nil {
		return fmt.Errorf("failed to load nodes: %w", err)
	}

	// Load chain pairs as edges
	if err := gb.loadEdges(ctx); err != nil {
		return fmt.Errorf("failed to load edges: %w", err)
	}

	gb.logger.Info().
		Int("nodes", len(gb.graph.Nodes)).
		Int("edges", gb.countEdges()).
		Msg("Routing graph built successfully")

	return nil
}

// GetGraph returns a read-only copy of the graph
func (gb *GraphBuilder) GetGraph() *Graph {
	gb.mu.RLock()
	defer gb.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	graphCopy := &Graph{
		Nodes: make(map[string]*Node),
		Edges: make(map[string]map[string]*Edge),
	}

	for k, v := range gb.graph.Nodes {
		graphCopy.Nodes[k] = v
	}

	for k, v := range gb.graph.Edges {
		graphCopy.Edges[k] = make(map[string]*Edge)
		for k2, v2 := range v {
			graphCopy.Edges[k][k2] = v2
		}
	}

	return graphCopy
}

// UpdateEdge updates an edge in the graph
func (gb *GraphBuilder) UpdateEdge(sourceChain, destChain string, edge *Edge) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	if gb.graph.Edges[sourceChain] == nil {
		gb.graph.Edges[sourceChain] = make(map[string]*Edge)
	}

	gb.graph.Edges[sourceChain][destChain] = edge

	gb.logger.Debug().
		Str("source", sourceChain).
		Str("dest", destChain).
		Msg("Edge updated")
}

// AddNode adds a node to the graph
func (gb *GraphBuilder) AddNode(chainID string, node *Node) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	gb.graph.Nodes[chainID] = node

	gb.logger.Debug().
		Str("chain_id", chainID).
		Msg("Node added")
}

// RemoveNode removes a node and all its edges from the graph
func (gb *GraphBuilder) RemoveNode(chainID string) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	delete(gb.graph.Nodes, chainID)

	// Remove all edges to/from this node
	delete(gb.graph.Edges, chainID)
	for _, edges := range gb.graph.Edges {
		delete(edges, chainID)
	}

	gb.logger.Debug().
		Str("chain_id", chainID).
		Msg("Node removed")
}

// UpdateLiquidity updates liquidity for a chain pair
func (gb *GraphBuilder) UpdateLiquidity(sourceChain, destChain string, liquidity *big.Int) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	if edge, exists := gb.graph.Edges[sourceChain][destChain]; exists {
		edge.Liquidity = liquidity
		edge.LastUpdated = time.Now().UTC()

		gb.logger.Debug().
			Str("source", sourceChain).
			Str("dest", destChain).
			Str("liquidity", liquidity.String()).
			Msg("Liquidity updated")
	}
}

// UpdateCost updates cost for a chain pair
func (gb *GraphBuilder) UpdateCost(sourceChain, destChain string, cost *big.Int) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	if edge, exists := gb.graph.Edges[sourceChain][destChain]; exists {
		edge.Cost = cost
		edge.Weight = gb.calculateEdgeWeight(edge)
		edge.LastUpdated = time.Now().UTC()

		gb.logger.Debug().
			Str("source", sourceChain).
			Str("dest", destChain).
			Str("cost", cost.String()).
			Msg("Cost updated")
	}
}

// UpdateSuccessRate updates success rate for a chain pair
func (gb *GraphBuilder) UpdateSuccessRate(sourceChain, destChain string, successRate float64) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	if edge, exists := gb.graph.Edges[sourceChain][destChain]; exists {
		edge.SuccessRate = successRate
		edge.Weight = gb.calculateEdgeWeight(edge)
		edge.LastUpdated = time.Now().UTC()

		gb.logger.Debug().
			Str("source", sourceChain).
			Str("dest", destChain).
			Float64("success_rate", successRate).
			Msg("Success rate updated")
	}
}

// RefreshGraph refreshes the graph with latest data from database
func (gb *GraphBuilder) RefreshGraph(ctx context.Context) error {
	gb.logger.Debug().Msg("Refreshing routing graph")

	// Rebuild the graph
	return gb.BuildGraph(ctx)
}

// StartPeriodicRefresh starts periodic graph refresh
func (gb *GraphBuilder) StartPeriodicRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			gb.logger.Info().Msg("Stopping periodic graph refresh")
			return
		case <-ticker.C:
			if err := gb.RefreshGraph(ctx); err != nil {
				gb.logger.Error().
					Err(err).
					Msg("Failed to refresh graph")
			}
		}
	}
}

// GetChainTopology returns the current chain connectivity topology
func (gb *GraphBuilder) GetChainTopology() *ChainGraph {
	gb.mu.RLock()
	defer gb.mu.RUnlock()

	chains := []string{}
	for chainID := range gb.graph.Nodes {
		chains = append(chains, chainID)
	}

	connections := make(map[string][]string)
	for source, edges := range gb.graph.Edges {
		connections[source] = []string{}
		for dest := range edges {
			connections[source] = append(connections[source], dest)
		}
	}

	return &ChainGraph{
		Chains:      chains,
		Connections: connections,
		UpdatedAt:   time.Now().UTC(),
	}
}

// Private methods

func (gb *GraphBuilder) loadNodes(ctx context.Context) error {
	// Query active chains from database
	query := `
		SELECT DISTINCT chain_id, name
		FROM (
			SELECT source_chain as chain_id, source_chain as name FROM messages
			UNION
			SELECT dest_chain as chain_id, dest_chain as name FROM messages
		) chains
	`

	rows, err := gb.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query chains: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var chainID, chainName string
		if err := rows.Scan(&chainID, &chainName); err != nil {
			continue
		}

		node := &Node{
			ChainID:      chainID,
			ChainName:    chainName,
			Active:       true,
			TotalVolume:  big.NewInt(0),
			LastActivity: time.Now().UTC(),
		}

		gb.graph.Nodes[chainID] = node
	}

	// Add hardcoded chains if not present (fallback)
	defaultChains := []string{"polygon", "ethereum", "bsc", "avalanche"}
	for _, chainID := range defaultChains {
		if _, exists := gb.graph.Nodes[chainID]; !exists {
			node := &Node{
				ChainID:      chainID,
				ChainName:    chainID,
				Active:       true,
				TotalVolume:  big.NewInt(0),
				LastActivity: time.Now().UTC(),
			}
			gb.graph.Nodes[chainID] = node
		}
	}

	return nil
}

func (gb *GraphBuilder) loadEdges(ctx context.Context) error {
	// Query chain pairs statistics
	query := `
		SELECT
			source_chain,
			dest_chain,
			COUNT(*) as total_messages,
			COUNT(*) FILTER (WHERE status = 'CONFIRMED') as confirmed_messages,
			AVG(EXTRACT(EPOCH FROM (COALESCE(confirmed_at, NOW()) - created_at))) as avg_time
		FROM messages
		WHERE created_at > NOW() - INTERVAL '7 days'
		GROUP BY source_chain, dest_chain
		HAVING COUNT(*) > 0
	`

	rows, err := gb.db.QueryContext(ctx, query)
	if err != nil {
		// If query fails, create default edges between all nodes
		gb.createDefaultEdges()
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var sourceChain, destChain string
		var totalMessages, confirmedMessages int
		var avgTime float64

		if err := rows.Scan(&sourceChain, &destChain, &totalMessages, &confirmedMessages, &avgTime); err != nil {
			continue
		}

		successRate := float64(confirmedMessages) / float64(totalMessages)

		// Estimate cost (in wei) - this would be fetched from blockchain in production
		estimatedCost := big.NewInt(1e16) // 0.01 ETH

		edge := &Edge{
			SourceChain: sourceChain,
			DestChain:   destChain,
			Cost:        estimatedCost,
			Time:        int64(avgTime),
			Liquidity:   big.NewInt(1e20), // Placeholder: 100 ETH
			SuccessRate: successRate,
			LastUpdated: time.Now().UTC(),
		}

		edge.Weight = gb.calculateEdgeWeight(edge)

		if gb.graph.Edges[sourceChain] == nil {
			gb.graph.Edges[sourceChain] = make(map[string]*Edge)
		}

		gb.graph.Edges[sourceChain][destChain] = edge
	}

	// Ensure all node pairs have edges (for testing)
	gb.createDefaultEdges()

	return nil
}

func (gb *GraphBuilder) createDefaultEdges() {
	// Create bidirectional edges between common chains
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

		// Only create if both nodes exist
		if _, sourceExists := gb.graph.Nodes[source]; !sourceExists {
			continue
		}
		if _, destExists := gb.graph.Nodes[dest]; !destExists {
			continue
		}

		// Create edge source -> dest
		if gb.graph.Edges[source] == nil {
			gb.graph.Edges[source] = make(map[string]*Edge)
		}
		if _, exists := gb.graph.Edges[source][dest]; !exists {
			edge := &Edge{
				SourceChain: source,
				DestChain:   dest,
				Cost:        big.NewInt(1e16), // 0.01 ETH
				Time:        300,               // 5 minutes
				Liquidity:   big.NewInt(1e20), // 100 ETH
				SuccessRate: 0.98,
				LastUpdated: time.Now().UTC(),
			}
			edge.Weight = gb.calculateEdgeWeight(edge)
			gb.graph.Edges[source][dest] = edge
		}

		// Create edge dest -> source (bidirectional)
		if gb.graph.Edges[dest] == nil {
			gb.graph.Edges[dest] = make(map[string]*Edge)
		}
		if _, exists := gb.graph.Edges[dest][source]; !exists {
			edge := &Edge{
				SourceChain: dest,
				DestChain:   source,
				Cost:        big.NewInt(1e16),
				Time:        300,
				Liquidity:   big.NewInt(1e20),
				SuccessRate: 0.98,
				LastUpdated: time.Now().UTC(),
			}
			edge.Weight = gb.calculateEdgeWeight(edge)
			gb.graph.Edges[dest][source] = edge
		}
	}
}

func (gb *GraphBuilder) calculateEdgeWeight(edge *Edge) float64 {
	// Weight is a combination of cost, time, and reliability
	// Lower weight is better

	// Normalize cost (assuming max 0.1 ETH)
	costWeight := 0.0
	if edge.Cost != nil {
		maxCost := big.NewInt(1e17) // 0.1 ETH
		costRatio := new(big.Float).Quo(
			new(big.Float).SetInt(edge.Cost),
			new(big.Float).SetInt(maxCost),
		)
		costFloat, _ := costRatio.Float64()
		costWeight = costFloat * 0.4 // 40% weight
	}

	// Normalize time (assuming max 1 hour)
	timeWeight := 0.0
	if edge.Time > 0 {
		maxTime := float64(3600)
		timeWeight = (float64(edge.Time) / maxTime) * 0.3 // 30% weight
	}

	// Reliability weight (inverse of success rate)
	reliabilityWeight := (1.0 - edge.SuccessRate) * 0.3 // 30% weight

	totalWeight := costWeight + timeWeight + reliabilityWeight

	return totalWeight
}

func (gb *GraphBuilder) countEdges() int {
	count := 0
	for _, edges := range gb.graph.Edges {
		count += len(edges)
	}
	return count
}
