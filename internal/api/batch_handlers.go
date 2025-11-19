package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/gorilla/mux"
)

// Batch API handlers

func (s *Server) handleListBatches(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit := 50
	offset := 0
	status := r.URL.Query().Get("status")

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Query batches from database
	var batches []database.Batch
	var err error

	if status != "" {
		batches, err = s.db.GetBatchesByStatus(r.Context(), status, limit, offset)
	} else {
		batches, err = s.db.GetAllBatches(r.Context(), limit, offset)
	}

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to query batches from database")
		respondError(w, http.StatusInternalServerError, "failed to retrieve batches", err)
		return
	}

	total, _ := s.db.GetBatchesCount(r.Context())

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"batches": batches,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"filter": map[string]string{
			"status": status,
		},
	})
}

func (s *Server) handleGetBatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	batchID := vars["id"]

	// Query batch from database
	batch, err := s.db.GetBatch(r.Context(), batchID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "batch not found", err)
		} else {
			s.logger.Error().Err(err).Str("batch_id", batchID).Msg("Failed to get batch")
			respondError(w, http.StatusInternalServerError, "failed to retrieve batch", err)
		}
		return
	}

	// Get messages in this batch
	messages, err := s.db.GetBatchMessages(r.Context(), batchID)
	if err != nil {
		s.logger.Warn().Err(err).Str("batch_id", batchID).Msg("Failed to get batch messages")
		// Continue without messages
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"batch":    batch,
		"messages": messages,
	})
}

func (s *Server) handleBatchStats(w http.ResponseWriter, r *http.Request) {
	// Get batch statistics from database
	totalBatches, err := s.db.GetBatchesCount(r.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get batches count")
		totalBatches = 0
	}

	totalMessages, err := s.db.GetTotalMessagesBatched(r.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get total messages batched")
		totalMessages = 0
	}

	avgBatchSize, err := s.db.GetAverageBatchSize(r.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get average batch size")
		avgBatchSize = 0
	}

	batchesToday, err := s.db.GetBatchesToday(r.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get batches today")
		batchesToday = 0
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total_batches":           totalBatches,
		"total_messages_batched":  totalMessages,
		"total_gas_saved":         "0", // TODO: Calculate from batch records
		"average_batch_size":      avgBatchSize,
		"average_savings_percent": 0, // TODO: Calculate savings percentage
		"batches_today":           batchesToday,
	})
}

func (s *Server) handleBatchEfficiency(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	batchID := vars["id"]

	// TODO: Calculate batch efficiency metrics

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"batch_id":           batchID,
		"message_count":      0,
		"gas_saved_wei":      "0",
		"gas_saved_eth":      "0",
		"savings_percentage": 0.0,
		"cost_per_message":   "0",
	})
}

type SubmitToBatchRequest struct {
	MessageID       string `json:"message_id"`
	SourceChain     string `json:"source_chain"`
	DestinationChain string `json:"dest_chain"`
	Priority        string `json:"priority"` // "low", "normal", "high"
}

func (s *Server) handleSubmitToBatch(w http.ResponseWriter, r *http.Request) {
	var req SubmitToBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if req.MessageID == "" {
		respondError(w, http.StatusBadRequest, "message_id is required", nil)
		return
	}

	// TODO: Add message to batch aggregator
	// This would interact with the batcher service

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":     "pending",
		"message":    "Message added to batch queue",
		"message_id": req.MessageID,
		"estimated_batch_time": "30s",
	})
}
