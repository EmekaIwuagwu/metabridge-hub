package routing

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Route discovery metrics
	RoutesDiscovered = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_routes_discovered_total",
			Help: "Total number of routes discovered",
		},
		[]string{"source_chain", "dest_chain"},
	)

	RouteDiscoveryLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_route_discovery_latency_seconds",
		Help:    "Route discovery latency in seconds",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
	})

	// Route execution metrics
	RoutesExecuted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_routes_executed_total",
		Help: "Total number of routes executed",
	})

	RoutesCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_routes_completed_total",
		Help: "Total number of routes completed successfully",
	})

	RoutesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_routes_failed_total",
		Help: "Total number of routes that failed",
	})

	RouteExecutionTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_route_execution_time_seconds",
		Help:    "Total time to execute a route from start to finish",
		Buckets: []float64{60, 300, 600, 1200, 1800, 3600, 7200},
	})

	// Route hop metrics
	RouteHopCount = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_route_hop_count",
		Help:    "Number of hops in executed routes",
		Buckets: []float64{1, 2, 3, 4, 5},
	})

	HopsCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_route_hops_completed_total",
		Help: "Total number of route hops completed",
	})

	HopsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_route_hops_failed_total",
		Help: "Total number of route hops that failed",
	})

	// Route cost metrics
	RouteTotalCost = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_route_total_cost_wei",
		Help:    "Total cost of route execution in wei",
		Buckets: prometheus.ExponentialBuckets(1e15, 2, 10),
	})

	// Cache metrics
	RouteCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_route_cache_hits_total",
		Help: "Total number of route cache hits",
	})

	RouteCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_route_cache_misses_total",
		Help: "Total number of route cache misses",
	})

	RouteCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_route_cache_size",
		Help: "Current number of entries in route cache",
	})

	// Liquidity metrics
	ChainPairLiquidity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_chain_pair_liquidity_wei",
			Help: "Available liquidity for chain pairs in wei",
		},
		[]string{"source_chain", "dest_chain"},
	)

	LiquidityReservations = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_liquidity_reservations_total",
		Help: "Total number of liquidity reservations",
	})

	LiquidityReleases = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_liquidity_releases_total",
		Help: "Total number of liquidity releases",
	})

	InsufficientLiquidity = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_insufficient_liquidity_total",
			Help: "Total number of times insufficient liquidity was encountered",
		},
		[]string{"source_chain", "dest_chain"},
	)

	// Graph metrics
	GraphNodes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_routing_graph_nodes",
		Help: "Number of nodes (chains) in routing graph",
	})

	GraphEdges = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bridge_routing_graph_edges",
		Help: "Number of edges (chain pairs) in routing graph",
	})

	GraphUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_routing_graph_updates_total",
		Help: "Total number of graph updates",
	})

	// Route quality metrics
	RouteScores = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bridge_route_scores",
		Help:    "Distribution of route quality scores",
		Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	})

	OptimalRouteSelected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bridge_optimal_route_selected_total",
		Help: "Total number of times the optimal route was selected",
	})
)

// Record functions for easier metric recording

// RecordRouteDiscovery records a route discovery event
func RecordRouteDiscovery(sourceChain, destChain string, routesFound int) {
	RoutesDiscovered.WithLabelValues(sourceChain, destChain).Add(float64(routesFound))
}

// RecordRouteDiscoveryLatency records route discovery latency
func RecordRouteDiscoveryLatency(seconds float64) {
	RouteDiscoveryLatency.Observe(seconds)
}

// RecordRouteExecution records a route execution
func RecordRouteExecution() {
	RoutesExecuted.Inc()
}

// RecordRouteCompleted records a completed route
func RecordRouteCompleted(hops int, executionTime float64, totalCost float64) {
	RoutesCompleted.Inc()
	RouteHopCount.Observe(float64(hops))
	RouteExecutionTime.Observe(executionTime)
	RouteTotalCost.Observe(totalCost)
}

// RecordRouteFailed records a failed route
func RecordRouteFailed() {
	RoutesFailed.Inc()
}

// RecordHopCompleted records a completed hop
func RecordHopCompleted() {
	HopsCompleted.Inc()
}

// RecordHopFailed records a failed hop
func RecordHopFailed() {
	HopsFailed.Inc()
}

// RecordRouteCacheHit records a cache hit
func RecordRouteCacheHit() {
	RouteCacheHits.Inc()
}

// RecordRouteCacheMiss records a cache miss
func RecordRouteCacheMiss() {
	RouteCacheMisses.Inc()
}

// RecordRouteCacheSet records a cache set operation
func RecordRouteCacheSet() {
	// Cache size would be updated separately
}

// SetRouteCacheSize sets the current cache size
func SetRouteCacheSize(size int) {
	RouteCacheSize.Set(float64(size))
}

// RecordLiquidity records liquidity for a chain pair
func RecordLiquidity(sourceChain, destChain string, liquidityWei float64) {
	ChainPairLiquidity.WithLabelValues(sourceChain, destChain).Set(liquidityWei)
}

// RecordLiquidityUpdate records a liquidity update
func RecordLiquidityUpdate(sourceChain, destChain string) {
	// Liquidity value would be set separately
}

// RecordLiquidityReservation records a liquidity reservation
func RecordLiquidityReservation() {
	LiquidityReservations.Inc()
}

// RecordLiquidityRelease records a liquidity release
func RecordLiquidityRelease() {
	LiquidityReleases.Inc()
}

// RecordInsufficientLiquidity records insufficient liquidity event
func RecordInsufficientLiquidity(sourceChain, destChain string) {
	InsufficientLiquidity.WithLabelValues(sourceChain, destChain).Inc()
}

// SetGraphSize sets the graph size metrics
func SetGraphSize(nodes, edges int) {
	GraphNodes.Set(float64(nodes))
	GraphEdges.Set(float64(edges))
}

// RecordGraphUpdate records a graph update
func RecordGraphUpdate() {
	GraphUpdates.Inc()
}

// RecordRouteScore records a route score
func RecordRouteScore(score float64) {
	RouteScores.Observe(score)
}

// RecordOptimalRoute records selection of optimal route
func RecordOptimalRoute() {
	OptimalRouteSelected.Inc()
}
