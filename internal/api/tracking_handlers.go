package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/webhooks"
	"github.com/gorilla/mux"
)

// handleTrackMessage tracks a specific message
func (s *Server) handleTrackMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	start := time.Now()
	defer func() {
		webhooks.RecordTrackingQueryLatency(time.Since(start).Seconds())
	}()

	timeline, err := s.trackingService.TrackMessage(r.Context(), messageID)
	if err != nil {
		respondError(w, http.StatusNotFound, "message not found", err)
		return
	}

	webhooks.RecordMessageTracked()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"timeline": timeline,
	})
}

// handleTrackByTxHash tracks a message by transaction hash
func (s *Server) handleTrackByTxHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txHash := vars["hash"]

	start := time.Now()
	defer func() {
		webhooks.RecordTrackingQueryLatency(time.Since(start).Seconds())
	}()

	message, err := s.trackingService.GetMessageByTxHash(r.Context(), txHash)
	if err != nil {
		respondError(w, http.StatusNotFound, "message not found", err)
		return
	}

	// Get full timeline
	timeline, err := s.trackingService.TrackMessage(r.Context(), message.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get timeline", err)
		return
	}

	webhooks.RecordMessageTracked()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  message,
		"timeline": timeline,
	})
}

// handleQueryMessages queries messages based on criteria
func (s *Server) handleQueryMessages(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		webhooks.RecordTrackingQueryLatency(time.Since(start).Seconds())
	}()

	query := &webhooks.TrackingQuery{
		MessageID:   r.URL.Query().Get("message_id"),
		TxHash:      r.URL.Query().Get("tx_hash"),
		Sender:      r.URL.Query().Get("sender"),
		Recipient:   r.URL.Query().Get("recipient"),
		SourceChain: r.URL.Query().Get("source_chain"),
		DestChain:   r.URL.Query().Get("dest_chain"),
		Status:      r.URL.Query().Get("status"),
	}

	// Parse dates
	if fromDateStr := r.URL.Query().Get("from_date"); fromDateStr != "" {
		if fromDate, err := time.Parse(time.RFC3339, fromDateStr); err == nil {
			query.FromDate = &fromDate
		}
	}

	if toDateStr := r.URL.Query().Get("to_date"); toDateStr != "" {
		if toDate, err := time.Parse(time.RFC3339, toDateStr); err == nil {
			query.ToDate = &toDate
		}
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = limit
		}
	} else {
		query.Limit = 50
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			query.Offset = offset
		}
	}

	result, err := s.trackingService.QueryMessages(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query messages", err)
		return
	}

	webhooks.RecordTrackingQuery()

	respondJSON(w, http.StatusOK, result)
}

// handleRecentMessages retrieves recent messages
func (s *Server) handleRecentMessages(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		webhooks.RecordTrackingQueryLatency(time.Since(start).Seconds())
	}()

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	messages, err := s.trackingService.GetRecentMessages(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get recent messages", err)
		return
	}

	webhooks.RecordTrackingQuery()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
	})
}

// handleMessagesByStatus retrieves messages by status
func (s *Server) handleMessagesByStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	status := vars["status"]

	start := time.Now()
	defer func() {
		webhooks.RecordTrackingQueryLatency(time.Since(start).Seconds())
	}()

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	messages, err := s.trackingService.GetMessagesByStatus(r.Context(), status, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get messages by status", err)
		return
	}

	webhooks.RecordTrackingQuery()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   status,
		"messages": messages,
		"count":    len(messages),
	})
}

// handleMessageTimeline retrieves the timeline for a message
func (s *Server) handleMessageTimeline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	start := time.Now()
	defer func() {
		webhooks.RecordTrackingQueryLatency(time.Since(start).Seconds())
	}()

	timeline, err := s.trackingService.TrackMessage(r.Context(), messageID)
	if err != nil {
		respondError(w, http.StatusNotFound, "message not found", err)
		return
	}

	webhooks.RecordMessageTracked()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"timeline": timeline,
	})
}

// handleRecordTimelineEvent records a new timeline event for a message
func (s *Server) handleRecordTimelineEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	var req struct {
		EventType   string                 `json:"event_type"`
		Description string                 `json:"description"`
		TxHash      string                 `json:"tx_hash,omitempty"`
		BlockNumber uint64                 `json:"block_number,omitempty"`
		ChainID     string                 `json:"chain_id,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.EventType == "" {
		respondError(w, http.StatusBadRequest, "event_type is required", nil)
		return
	}

	event := &webhooks.TimelineEvent{
		EventType:   req.EventType,
		Timestamp:   time.Now().UTC(),
		Description: req.Description,
		TxHash:      req.TxHash,
		BlockNumber: req.BlockNumber,
		ChainID:     req.ChainID,
		Metadata:    req.Metadata,
	}

	if err := s.trackingService.RecordEvent(r.Context(), messageID, event); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to record event", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Event recorded successfully",
		"event":   event,
	})
}

// handleTrackingStats retrieves tracking statistics
func (s *Server) handleTrackingStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		webhooks.RecordTrackingQueryLatency(time.Since(start).Seconds())
	}()

	// Get statistics from database
	query := `
		SELECT
			status,
			COUNT(*) as count,
			AVG(EXTRACT(EPOCH FROM (COALESCE(confirmed_at, NOW()) - created_at))) as avg_time_seconds
		FROM messages
		WHERE created_at > NOW() - INTERVAL '24 hours'
		GROUP BY status
	`

	rows, err := s.db.QueryContext(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get stats", err)
		return
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	totalMessages := 0

	for rows.Next() {
		var status string
		var count int
		var avgTime float64

		if err := rows.Scan(&status, &count, &avgTime); err != nil {
			continue
		}

		stats[status] = map[string]interface{}{
			"count":            count,
			"avg_time_seconds": avgTime,
		}

		totalMessages += count
	}

	webhooks.RecordTrackingQuery()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"period":         "24h",
		"total_messages": totalMessages,
		"by_status":      stats,
		"timestamp":      time.Now().UTC(),
	})
}

// handleSearchMessages searches messages with full-text search
func (s *Server) handleSearchMessages(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		respondError(w, http.StatusBadRequest, "search query parameter 'q' is required", nil)
		return
	}

	start := time.Now()
	defer func() {
		webhooks.RecordTrackingQueryLatency(time.Since(start).Seconds())
	}()

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Search across multiple fields
	query := `
		SELECT
			id, source_chain, dest_chain, sender, recipient,
			token_address, amount, nonce, data, status,
			source_tx_hash, dest_tx_hash, validator_signatures,
			created_at, updated_at, submitted_at, confirmed_at
		FROM messages
		WHERE
			id ILIKE $1 OR
			sender ILIKE $1 OR
			recipient ILIKE $1 OR
			source_tx_hash ILIKE $1 OR
			dest_tx_hash ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	searchPattern := "%" + searchTerm + "%"
	rows, err := s.db.QueryContext(r.Context(), query, searchPattern, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to search messages", err)
		return
	}
	defer rows.Close()

	messages := []map[string]interface{}{}
	for rows.Next() {
		var message struct {
			ID                  string
			SourceChain         string
			DestChain           string
			Sender              string
			Recipient           string
			TokenAddress        string
			Amount              string
			Nonce               int64
			Data                string
			Status              string
			SourceTxHash        *string
			DestTxHash          *string
			ValidatorSignatures *string
			CreatedAt           time.Time
			UpdatedAt           time.Time
			SubmittedAt         *time.Time
			ConfirmedAt         *time.Time
		}

		err := rows.Scan(
			&message.ID, &message.SourceChain, &message.DestChain,
			&message.Sender, &message.Recipient, &message.TokenAddress,
			&message.Amount, &message.Nonce, &message.Data, &message.Status,
			&message.SourceTxHash, &message.DestTxHash, &message.ValidatorSignatures,
			&message.CreatedAt, &message.UpdatedAt, &message.SubmittedAt, &message.ConfirmedAt,
		)
		if err != nil {
			continue
		}

		messageMap := map[string]interface{}{
			"id":           message.ID,
			"source_chain": message.SourceChain,
			"dest_chain":   message.DestChain,
			"sender":       message.Sender,
			"recipient":    message.Recipient,
			"amount":       message.Amount,
			"status":       message.Status,
			"created_at":   message.CreatedAt,
		}

		if message.SourceTxHash != nil {
			messageMap["source_tx_hash"] = *message.SourceTxHash
		}
		if message.DestTxHash != nil {
			messageMap["dest_tx_hash"] = *message.DestTxHash
		}

		messages = append(messages, messageMap)
	}

	webhooks.RecordTrackingQuery()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"query":    searchTerm,
		"messages": messages,
		"count":    len(messages),
	})
}
