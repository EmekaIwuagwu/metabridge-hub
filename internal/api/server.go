package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/auth"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/routing"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/webhooks"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

// Server represents the API server
type Server struct {
	config          *config.Config
	db              *database.DB
	router          *mux.Router
	server          *http.Server
	logger          zerolog.Logger
	clients         map[string]types.UniversalClient
	webhookRegistry *webhooks.Registry
	webhookDelivery *webhooks.DeliveryService
	trackingService *webhooks.TrackingService
	routingService  *routing.Service
	authMiddleware  *auth.Middleware
	authHandler     *auth.Handler
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	db *database.DB,
	clients map[string]types.UniversalClient,
	logger zerolog.Logger,
) *Server {
	router := mux.NewRouter()

	// Initialize webhook and tracking services
	webhookRegistry := webhooks.NewRegistry(db, logger)
	trackingService := webhooks.NewTrackingService(db, logger)
	webhookDelivery := webhooks.NewDeliveryService(nil, webhookRegistry, db, logger)

	// Initialize routing service
	routingService := routing.NewService(db, nil, logger)

	// Initialize authentication
	authConfig := getAuthConfig()
	authMiddleware := auth.NewMiddleware(authConfig, db, logger)
	authHandler := auth.NewHandler(db, authConfig, logger)

	s := &Server{
		config:          cfg,
		db:              db,
		router:          router,
		logger:          logger.With().Str("component", "api").Logger(),
		clients:         clients,
		webhookRegistry: webhookRegistry,
		webhookDelivery: webhookDelivery,
		trackingService: trackingService,
		routingService:  routingService,
		authMiddleware:  authMiddleware,
		authHandler:     authHandler,
	}

	// Start webhook delivery service
	go webhookDelivery.Start(context.Background())

	// Start routing service
	go routingService.Start(context.Background())

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/ready", s.handleReady).Methods("GET")

	// API v1
	v1 := s.router.PathPrefix("/v1").Subrouter()

	// Chain endpoints
	v1.HandleFunc("/chains", s.handleListChains).Methods("GET")
	v1.HandleFunc("/chains/{chain}/status", s.handleChainStatus).Methods("GET")
	v1.HandleFunc("/chains/status", s.handleAllChainsStatus).Methods("GET")

	// Bridge endpoints
	v1.HandleFunc("/bridge/token", s.handleBridgeToken).Methods("POST")
	v1.HandleFunc("/bridge/nft", s.handleBridgeNFT).Methods("POST")

	// Message endpoints
	v1.HandleFunc("/messages", s.handleListMessages).Methods("GET")
	v1.HandleFunc("/messages/{id}", s.handleGetMessage).Methods("GET")
	v1.HandleFunc("/messages/{id}/status", s.handleMessageStatus).Methods("GET")

	// Batch endpoints
	v1.HandleFunc("/batches", s.handleListBatches).Methods("GET")
	v1.HandleFunc("/batches/stats", s.handleBatchStats).Methods("GET")
	v1.HandleFunc("/batches/{id}", s.handleGetBatch).Methods("GET")
	v1.HandleFunc("/batches/{id}/efficiency", s.handleBatchEfficiency).Methods("GET")
	v1.HandleFunc("/batches/submit", s.handleSubmitToBatch).Methods("POST")

	// Statistics endpoints
	v1.HandleFunc("/stats", s.handleStats).Methods("GET")
	v1.HandleFunc("/stats/{chain}", s.handleChainStats).Methods("GET")

	// Transaction endpoints
	v1.HandleFunc("/transactions/{hash}", s.handleGetTransaction).Methods("GET")

	// Webhook endpoints
	v1.HandleFunc("/webhooks", s.handleRegisterWebhook).Methods("POST")
	v1.HandleFunc("/webhooks", s.handleListWebhooks).Methods("GET")
	v1.HandleFunc("/webhooks/{id}", s.handleGetWebhook).Methods("GET")
	v1.HandleFunc("/webhooks/{id}", s.handleUpdateWebhook).Methods("PUT")
	v1.HandleFunc("/webhooks/{id}", s.handleDeleteWebhook).Methods("DELETE")
	v1.HandleFunc("/webhooks/{id}/pause", s.handlePauseWebhook).Methods("POST")
	v1.HandleFunc("/webhooks/{id}/resume", s.handleResumeWebhook).Methods("POST")
	v1.HandleFunc("/webhooks/{id}/test", s.handleTestWebhook).Methods("POST")
	v1.HandleFunc("/webhooks/{id}/attempts", s.handleWebhookDeliveryAttempts).Methods("GET")

	// Tracking endpoints
	v1.HandleFunc("/track/{id}", s.handleTrackMessage).Methods("GET")
	v1.HandleFunc("/track/tx/{hash}", s.handleTrackByTxHash).Methods("GET")
	v1.HandleFunc("/track/query", s.handleQueryMessages).Methods("GET")
	v1.HandleFunc("/track/recent", s.handleRecentMessages).Methods("GET")
	v1.HandleFunc("/track/status/{status}", s.handleMessagesByStatus).Methods("GET")
	v1.HandleFunc("/track/{id}/timeline", s.handleMessageTimeline).Methods("GET")
	v1.HandleFunc("/track/{id}/events", s.handleRecordTimelineEvent).Methods("POST")
	v1.HandleFunc("/track/stats", s.handleTrackingStats).Methods("GET")
	v1.HandleFunc("/track/search", s.handleSearchMessages).Methods("GET")

	// Routing endpoints
	v1.HandleFunc("/routes/find", s.handleFindRoutes).Methods("POST")
	v1.HandleFunc("/routes/{id}", s.handleGetRoute).Methods("GET")
	v1.HandleFunc("/routes/{id}/execute", s.handleExecuteRoute).Methods("POST")
	v1.HandleFunc("/routes/topology", s.handleGetChainTopology).Methods("GET")
	v1.HandleFunc("/routes/liquidity", s.handleGetLiquidity).Methods("GET")
	v1.HandleFunc("/routes/cache/stats", s.handleGetRouteCacheStats).Methods("GET")
	v1.HandleFunc("/routes/cache/invalidate", s.handleInvalidateCache).Methods("POST")
	v1.HandleFunc("/routes/estimate", s.handleGetRouteEstimate).Methods("GET")

	// Authentication endpoints (public)
	authRouter := s.router.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/login", s.authHandler.HandleLogin).Methods("POST")
	authRouter.HandleFunc("/refresh", s.authHandler.HandleRefreshToken).Methods("POST")
	authRouter.HandleFunc("/me", s.authHandler.HandleGetMe).Methods("GET")
	authRouter.HandleFunc("/api-keys", s.authHandler.HandleCreateAPIKey).Methods("POST")
	authRouter.HandleFunc("/api-keys", s.authHandler.HandleListAPIKeys).Methods("GET")
	authRouter.HandleFunc("/api-keys/{id}", s.authHandler.HandleRevokeAPIKey).Methods("DELETE")

	// Apply global middleware (order matters!)
	s.router.Use(s.recoverMiddleware)
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)
	s.router.Use(s.authMiddleware.RateLimit)

	// Apply auth middleware to v1 routes only
	v1.Use(s.authMiddleware.AuthRequired)
}

// Start starts the API server
func (s *Server) Start() error {
	s.logger.Info().
		Str("address", s.server.Addr).
		Msg("Starting API server")

	if s.config.Server.TLSEnabled {
		return s.server.ListenAndServeTLS(
			s.config.Server.TLSCertPath,
			s.config.Server.TLSKeyPath,
		)
	}

	return s.server.ListenAndServe()
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info().Msg("Stopping API server")
	return s.server.Shutdown(ctx)
}

// Health check handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "healthy",
		"service":     "metabridge-api",
		"environment": s.config.Environment,
		"timestamp":   time.Now().UTC(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check database
	if err := s.db.HealthCheck(r.Context()); err != nil {
		respondError(w, http.StatusServiceUnavailable, "database not ready", err)
		return
	}

	// Check at least one blockchain client is healthy
	healthyClients := 0
	for _, client := range s.clients {
		if client.IsHealthy(r.Context()) {
			healthyClients++
		}
	}

	if healthyClients == 0 {
		respondError(w, http.StatusServiceUnavailable, "no healthy blockchain clients", nil)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ready",
		"healthy_chains": healthyClients,
		"total_chains":   len(s.clients),
	})
}

// Middleware

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call next handler
		next.ServeHTTP(w, r)

		// Log request
		s.logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Dur("duration", time.Since(start)).
			Msg("HTTP request")
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment, default to * for development
		allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "*"
		}

		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if allowedOrigins == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			// Check if origin is in allowed list
			allowedList := strings.Split(allowedOrigins, ",")
			for _, allowed := range allowedList {
				if strings.TrimSpace(allowed) == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error().
					Interface("error", err).
					Str("path", r.URL.Path).
					Msg("Panic recovered")

				respondError(w, http.StatusInternalServerError, "internal server error", nil)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but can't change response at this point
		log.Printf("Error encoding JSON: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error":  message,
		"status": status,
	}

	if err != nil {
		response["details"] = err.Error()
	}

	respondJSON(w, status, response)
}

// getAuthConfig returns authentication configuration from environment variables
func getAuthConfig() *auth.AuthConfig {
	config := auth.DefaultAuthConfig()

	// JWT Secret
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		config.JWTSecret = secret
	}

	// JWT Expiration (in hours)
	if expiry := os.Getenv("JWT_EXPIRATION_HOURS"); expiry != "" {
		if hours, err := strconv.Atoi(expiry); err == nil && hours > 0 {
			config.JWTExpirationHours = hours
		}
	}

	// Rate Limit
	if rateLimit := os.Getenv("RATE_LIMIT_PER_MINUTE"); rateLimit != "" {
		if limit, err := strconv.Atoi(rateLimit); err == nil && limit > 0 {
			config.RateLimitPerMinute = limit
		}
	}

	// Require Authentication (default: true)
	if requireAuth := os.Getenv("REQUIRE_AUTH"); requireAuth != "" {
		config.RequireAuth = requireAuth != "false"
	}

	// API Key Enabled (default: true)
	if apiKeyEnabled := os.Getenv("API_KEY_ENABLED"); apiKeyEnabled != "" {
		config.APIKeyEnabled = apiKeyEnabled != "false"
	}

	return config
}
