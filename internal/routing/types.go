package routing

import (
	"math/big"
	"time"
)

// RouteStatus represents the status of a multi-hop route
type RouteStatus string

const (
	RouteStatusPending   RouteStatus = "PENDING"
	RouteStatusExecuting RouteStatus = "EXECUTING"
	RouteStatusCompleted RouteStatus = "COMPLETED"
	RouteStatusFailed    RouteStatus = "FAILED"
	RouteStatusPartial   RouteStatus = "PARTIAL"
)

// Route represents a multi-hop path between chains
type Route struct {
	ID          string        `json:"id"`
	SourceChain string        `json:"source_chain"`
	DestChain   string        `json:"dest_chain"`
	Hops        []Hop         `json:"hops"`
	TotalCost   *big.Int      `json:"total_cost"`
	TotalTime   time.Duration `json:"total_time_seconds"`
	TotalFee    *big.Int      `json:"total_fee"`
	Score       float64       `json:"score"`
	Status      RouteStatus   `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	ExecutedAt  *time.Time    `json:"executed_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

// Hop represents a single step in a multi-hop route
type Hop struct {
	Step           int      `json:"step"`
	SourceChain    string   `json:"source_chain"`
	DestChain      string   `json:"dest_chain"`
	BridgeContract string   `json:"bridge_contract"`
	EstimatedCost  *big.Int `json:"estimated_cost"`
	EstimatedTime  int64    `json:"estimated_time_seconds"`
	GasPrice       *big.Int `json:"gas_price"`
	Liquidity      *big.Int `json:"liquidity"`
	MessageID      string   `json:"message_id,omitempty"`
	TxHash         string   `json:"tx_hash,omitempty"`
	Status         string   `json:"status"`
}

// ChainPair represents a direct connection between two chains
type ChainPair struct {
	SourceChain    string    `json:"source_chain"`
	DestChain      string    `json:"dest_chain"`
	Available      bool      `json:"available"`
	Liquidity      *big.Int  `json:"liquidity"`
	BaseFee        *big.Int  `json:"base_fee"`
	GasPrice       *big.Int  `json:"gas_price"`
	AverageTime    int64     `json:"average_time_seconds"`
	SuccessRate    float64   `json:"success_rate"`
	LastUpdated    time.Time `json:"last_updated"`
	BridgeContract string    `json:"bridge_contract"`
}

// RouteQuery represents a route search request
type RouteQuery struct {
	SourceChain  string   `json:"source_chain"`
	DestChain    string   `json:"dest_chain"`
	Amount       *big.Int `json:"amount"`
	TokenAddress string   `json:"token_address"`
	MaxHops      int      `json:"max_hops"`
	OptimizeFor  string   `json:"optimize_for"` // "cost", "time", "balanced"
	MaxCost      *big.Int `json:"max_cost,omitempty"`
	MaxTime      int64    `json:"max_time_seconds,omitempty"`
	MinLiquidity *big.Int `json:"min_liquidity,omitempty"`
}

// RouteResult represents the result of a route search
type RouteResult struct {
	Routes           []*Route  `json:"routes"`
	RecommendedRoute *Route    `json:"recommended_route,omitempty"`
	Count            int       `json:"count"`
	SearchTime       int64     `json:"search_time_ms"`
	Timestamp        time.Time `json:"timestamp"`
}

// Graph represents the chain connectivity graph
type Graph struct {
	Nodes map[string]*Node
	Edges map[string]map[string]*Edge
}

// Node represents a blockchain in the routing graph
type Node struct {
	ChainID      string
	ChainName    string
	Active       bool
	TotalVolume  *big.Int
	LastActivity time.Time
}

// Edge represents a connection between two chains
type Edge struct {
	SourceChain string
	DestChain   string
	Weight      float64 // Combined score of cost, time, and reliability
	Cost        *big.Int
	Time        int64
	Liquidity   *big.Int
	SuccessRate float64
	LastUpdated time.Time
}

// RouteExecution represents an executing multi-hop route
type RouteExecution struct {
	RouteID       string      `json:"route_id"`
	CurrentHop    int         `json:"current_hop"`
	TotalHops     int         `json:"total_hops"`
	Status        RouteStatus `json:"status"`
	CompletedHops []string    `json:"completed_hops"`
	FailedHops    []string    `json:"failed_hops,omitempty"`
	StartedAt     time.Time   `json:"started_at"`
	LastUpdate    time.Time   `json:"last_update"`
	ErrorMessage  string      `json:"error_message,omitempty"`
}

// LiquidityInfo represents liquidity information for a chain pair
type LiquidityInfo struct {
	ChainPair          string        `json:"chain_pair"`
	SourceChain        string        `json:"source_chain"`
	DestChain          string        `json:"dest_chain"`
	TotalLiquidity     *big.Int      `json:"total_liquidity"`
	AvailableLiquidity *big.Int      `json:"available_liquidity"`
	ReservedLiquidity  *big.Int      `json:"reserved_liquidity"`
	LastUpdated        time.Time     `json:"last_updated"`
	UpdateInterval     time.Duration `json:"update_interval"`
}

// RouteOptimizationConfig represents configuration for route optimization
type RouteOptimizationConfig struct {
	MaxHops           int           `json:"max_hops"`
	MaxSearchTime     time.Duration `json:"max_search_time"`
	MaxRoutesToReturn int           `json:"max_routes_to_return"`
	CostWeight        float64       `json:"cost_weight"`
	TimeWeight        float64       `json:"time_weight"`
	LiquidityWeight   float64       `json:"liquidity_weight"`
	ReliabilityWeight float64       `json:"reliability_weight"`
	MinSuccessRate    float64       `json:"min_success_rate"`
	CacheEnabled      bool          `json:"cache_enabled"`
	CacheTTL          time.Duration `json:"cache_ttl"`
}

// DefaultOptimizationConfig returns default optimization configuration
func DefaultOptimizationConfig() *RouteOptimizationConfig {
	return &RouteOptimizationConfig{
		MaxHops:           3,
		MaxSearchTime:     5 * time.Second,
		MaxRoutesToReturn: 5,
		CostWeight:        0.3,
		TimeWeight:        0.3,
		LiquidityWeight:   0.2,
		ReliabilityWeight: 0.2,
		MinSuccessRate:    0.95,
		CacheEnabled:      true,
		CacheTTL:          5 * time.Minute,
	}
}

// RoutingMetrics represents metrics for a route
type RoutingMetrics struct {
	RouteID         string        `json:"route_id"`
	TotalHops       int           `json:"total_hops"`
	TotalCost       string        `json:"total_cost"`
	TotalTime       time.Duration `json:"total_time"`
	SuccessRate     float64       `json:"success_rate"`
	AverageGasPrice string        `json:"average_gas_price"`
	ExecutionTime   time.Duration `json:"execution_time"`
}

// RouteStats represents statistics for route execution
type RouteStats struct {
	TotalRoutes      int64   `json:"total_routes"`
	SuccessfulRoutes int64   `json:"successful_routes"`
	FailedRoutes     int64   `json:"failed_routes"`
	AverageHops      float64 `json:"average_hops"`
	AverageCost      string  `json:"average_cost"`
	AverageTime      int64   `json:"average_time_seconds"`
	TotalVolume      string  `json:"total_volume"`
}

// CachedRoute represents a cached route with TTL
type CachedRoute struct {
	Query     *RouteQuery `json:"query"`
	Routes    []*Route    `json:"routes"`
	CachedAt  time.Time   `json:"cached_at"`
	ExpiresAt time.Time   `json:"expires_at"`
	HitCount  int         `json:"hit_count"`
}

// RouteHealthCheck represents health status of a chain pair
type RouteHealthCheck struct {
	SourceChain  string    `json:"source_chain"`
	DestChain    string    `json:"dest_chain"`
	Healthy      bool      `json:"healthy"`
	Liquidity    *big.Int  `json:"liquidity"`
	LastSuccess  time.Time `json:"last_success"`
	LastFailure  time.Time `json:"last_failure"`
	FailureCount int       `json:"failure_count"`
	Message      string    `json:"message,omitempty"`
}

// ChainGraph represents the overall routing topology
type ChainGraph struct {
	Chains      []string            `json:"chains"`
	Connections map[string][]string `json:"connections"`
	UpdatedAt   time.Time           `json:"updated_at"`
}
