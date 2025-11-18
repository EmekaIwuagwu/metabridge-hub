package api

import (
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/routing"
	"github.com/gorilla/mux"
)

// handleFindRoutes finds optimal routes between chains
func (s *Server) handleFindRoutes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceChain  string `json:"source_chain"`
		DestChain    string `json:"dest_chain"`
		Amount       string `json:"amount"`
		TokenAddress string `json:"token_address"`
		MaxHops      int    `json:"max_hops"`
		OptimizeFor  string `json:"optimize_for"` // "cost", "time", "balanced"
		MaxCost      string `json:"max_cost,omitempty"`
		MaxTime      int64  `json:"max_time_seconds,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate required fields
	if req.SourceChain == "" {
		respondError(w, http.StatusBadRequest, "source_chain is required", nil)
		return
	}
	if req.DestChain == "" {
		respondError(w, http.StatusBadRequest, "dest_chain is required", nil)
		return
	}

	// Parse amount
	amount := big.NewInt(0)
	if req.Amount != "" {
		var ok bool
		amount, ok = new(big.Int).SetString(req.Amount, 10)
		if !ok {
			respondError(w, http.StatusBadRequest, "invalid amount format", nil)
			return
		}
	}

	// Build query
	query := &routing.RouteQuery{
		SourceChain:  req.SourceChain,
		DestChain:    req.DestChain,
		Amount:       amount,
		TokenAddress: req.TokenAddress,
		MaxHops:      req.MaxHops,
		OptimizeFor:  req.OptimizeFor,
	}

	// Parse optional fields
	if req.MaxCost != "" {
		maxCost, ok := new(big.Int).SetString(req.MaxCost, 10)
		if ok {
			query.MaxCost = maxCost
		}
	}

	if req.MaxTime > 0 {
		query.MaxTime = req.MaxTime
	}

	// Find routes
	result, err := s.routingService.FindRoutes(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to find routes", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleExecuteRoute executes a multi-hop route
func (s *Server) handleExecuteRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	routeID := vars["id"]

	execution, err := s.routingService.ExecuteRoute(r.Context(), routeID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to execute route", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"execution": execution,
		"message":   "Route execution started",
	})
}

// handleGetRoute retrieves route details
func (s *Server) handleGetRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	routeID := vars["id"]

	route, err := s.routingService.GetRouteStatus(r.Context(), routeID)
	if err != nil {
		respondError(w, http.StatusNotFound, "route not found", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"route": route,
	})
}

// handleGetChainTopology returns the routing topology
func (s *Server) handleGetChainTopology(w http.ResponseWriter, r *http.Request) {
	topology := s.routingService.GetChainTopology()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"topology": topology,
	})
}

// handleGetLiquidity returns liquidity information
func (s *Server) handleGetLiquidity(w http.ResponseWriter, r *http.Request) {
	liquidity := s.routingService.GetLiquidityInfo()

	// Convert to JSON-friendly format
	liquidityList := []map[string]interface{}{}
	for _, info := range liquidity {
		liquidityList = append(liquidityList, map[string]interface{}{
			"chain_pair":          info.ChainPair,
			"source_chain":        info.SourceChain,
			"dest_chain":          info.DestChain,
			"total_liquidity":     info.TotalLiquidity.String(),
			"available_liquidity": info.AvailableLiquidity.String(),
			"reserved_liquidity":  info.ReservedLiquidity.String(),
			"last_updated":        info.LastUpdated,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"liquidity": liquidityList,
		"count":     len(liquidityList),
	})
}

// handleGetRouteCacheStats returns cache statistics
func (s *Server) handleGetRouteCacheStats(w http.ResponseWriter, r *http.Request) {
	stats := s.routingService.GetCacheStats()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"cache_stats": stats,
	})
}

// handleInvalidateCache invalidates cache for a chain pair
func (s *Server) handleInvalidateCache(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceChain string `json:"source_chain"`
		DestChain   string `json:"dest_chain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	s.routingService.InvalidateCache(req.SourceChain, req.DestChain)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Cache invalidated successfully",
	})
}

// handleGetRouteEstimate provides a quick cost/time estimate without full route discovery
func (s *Server) handleGetRouteEstimate(w http.ResponseWriter, r *http.Request) {
	sourceChain := r.URL.Query().Get("source_chain")
	destChain := r.URL.Query().Get("dest_chain")
	amount := r.URL.Query().Get("amount")

	if sourceChain == "" || destChain == "" {
		respondError(w, http.StatusBadRequest, "source_chain and dest_chain are required", nil)
		return
	}

	// Parse amount
	amountBig := big.NewInt(0)
	if amount != "" {
		var ok bool
		amountBig, ok = new(big.Int).SetString(amount, 10)
		if !ok {
			respondError(w, http.StatusBadRequest, "invalid amount format", nil)
			return
		}
	}

	// Quick query with minimal hops
	query := &routing.RouteQuery{
		SourceChain: sourceChain,
		DestChain:   destChain,
		Amount:      amountBig,
		MaxHops:     2, // Quick estimate with max 2 hops
	}

	result, err := s.routingService.FindRoutes(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to estimate route", err)
		return
	}

	// Return simplified estimate
	estimate := map[string]interface{}{
		"source_chain": sourceChain,
		"dest_chain":   destChain,
		"available":    len(result.Routes) > 0,
	}

	if result.RecommendedRoute != nil {
		estimate["estimated_cost"] = result.RecommendedRoute.TotalCost.String()
		estimate["estimated_time_seconds"] = int64(result.RecommendedRoute.TotalTime.Seconds())
		estimate["hops"] = len(result.RecommendedRoute.Hops)
		estimate["score"] = result.RecommendedRoute.Score
	}

	respondJSON(w, http.StatusOK, estimate)
}
