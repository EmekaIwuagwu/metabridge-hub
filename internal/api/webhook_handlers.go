package api

import (
	"encoding/json"
	"net/http"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/webhooks"
	"github.com/gorilla/mux"
)

// handleRegisterWebhook registers a new webhook
func (s *Server) handleRegisterWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL          string                `json:"url"`
		Events       []webhooks.EventType  `json:"events"`
		Description  string                `json:"description"`
		SourceChains []string              `json:"source_chains,omitempty"`
		DestChains   []string              `json:"dest_chains,omitempty"`
		MinAmount    string                `json:"min_amount,omitempty"`
		MaxAmount    string                `json:"max_amount,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "url is required", nil)
		return
	}

	if len(req.Events) == 0 {
		respondError(w, http.StatusBadRequest, "at least one event type is required", nil)
		return
	}

	// For now, use a placeholder for created_by
	// In production, this should come from authentication
	createdBy := r.Header.Get("X-User-ID")
	if createdBy == "" {
		createdBy = "anonymous"
	}

	webhook := &webhooks.Webhook{
		URL:          req.URL,
		Events:       req.Events,
		Description:  req.Description,
		CreatedBy:    createdBy,
		SourceChains: req.SourceChains,
		DestChains:   req.DestChains,
		MinAmount:    req.MinAmount,
		MaxAmount:    req.MaxAmount,
	}

	if err := s.webhookRegistry.Register(r.Context(), webhook); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to register webhook", err)
		return
	}

	webhooks.RecordWebhookRegistered()
	webhooks.RecordWebhookStatusChange("", webhooks.WebhookStatusActive)

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"webhook": webhook,
		"message": "Webhook registered successfully",
	})
}

// handleListWebhooks lists all webhooks for the current user
func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	createdBy := r.Header.Get("X-User-ID")
	if createdBy == "" {
		createdBy = "anonymous"
	}

	hooks, err := s.webhookRegistry.List(r.Context(), createdBy)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list webhooks", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"webhooks": hooks,
		"count":    len(hooks),
	})
}

// handleGetWebhook retrieves a specific webhook
func (s *Server) handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	webhook, err := s.webhookRegistry.Get(r.Context(), webhookID)
	if err != nil {
		respondError(w, http.StatusNotFound, "webhook not found", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"webhook": webhook,
	})
}

// handleUpdateWebhook updates an existing webhook
func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	var req struct {
		URL          string                `json:"url"`
		Events       []webhooks.EventType  `json:"events"`
		Status       webhooks.WebhookStatus `json:"status"`
		Description  string                `json:"description"`
		SourceChains []string              `json:"source_chains,omitempty"`
		DestChains   []string              `json:"dest_chains,omitempty"`
		MinAmount    string                `json:"min_amount,omitempty"`
		MaxAmount    string                `json:"max_amount,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Get existing webhook
	webhook, err := s.webhookRegistry.Get(r.Context(), webhookID)
	if err != nil {
		respondError(w, http.StatusNotFound, "webhook not found", err)
		return
	}

	// Update fields
	if req.URL != "" {
		webhook.URL = req.URL
	}
	if len(req.Events) > 0 {
		webhook.Events = req.Events
	}
	if req.Status != "" {
		oldStatus := webhook.Status
		webhook.Status = req.Status
		webhooks.RecordWebhookStatusChange(oldStatus, req.Status)
	}
	if req.Description != "" {
		webhook.Description = req.Description
	}
	if len(req.SourceChains) > 0 {
		webhook.SourceChains = req.SourceChains
	}
	if len(req.DestChains) > 0 {
		webhook.DestChains = req.DestChains
	}
	if req.MinAmount != "" {
		webhook.MinAmount = req.MinAmount
	}
	if req.MaxAmount != "" {
		webhook.MaxAmount = req.MaxAmount
	}

	if err := s.webhookRegistry.Update(r.Context(), webhook); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update webhook", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"webhook": webhook,
		"message": "Webhook updated successfully",
	})
}

// handleDeleteWebhook deletes a webhook
func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	// Get webhook to get its status before deletion
	webhook, err := s.webhookRegistry.Get(r.Context(), webhookID)
	if err != nil {
		respondError(w, http.StatusNotFound, "webhook not found", err)
		return
	}

	if err := s.webhookRegistry.Delete(r.Context(), webhookID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete webhook", err)
		return
	}

	webhooks.RecordWebhookDeleted()
	webhooks.RecordWebhookStatusChange(webhook.Status, "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Webhook deleted successfully",
	})
}

// handlePauseWebhook pauses a webhook
func (s *Server) handlePauseWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	// Get current status
	webhook, err := s.webhookRegistry.Get(r.Context(), webhookID)
	if err != nil {
		respondError(w, http.StatusNotFound, "webhook not found", err)
		return
	}

	oldStatus := webhook.Status
	if err := s.webhookRegistry.UpdateStatus(r.Context(), webhookID, webhooks.WebhookStatusPaused); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to pause webhook", err)
		return
	}

	webhooks.RecordWebhookStatusChange(oldStatus, webhooks.WebhookStatusPaused)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Webhook paused successfully",
	})
}

// handleResumeWebhook resumes a paused webhook
func (s *Server) handleResumeWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	// Get current status
	webhook, err := s.webhookRegistry.Get(r.Context(), webhookID)
	if err != nil {
		respondError(w, http.StatusNotFound, "webhook not found", err)
		return
	}

	oldStatus := webhook.Status
	if err := s.webhookRegistry.UpdateStatus(r.Context(), webhookID, webhooks.WebhookStatusActive); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to resume webhook", err)
		return
	}

	webhooks.RecordWebhookStatusChange(oldStatus, webhooks.WebhookStatusActive)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Webhook resumed successfully",
	})
}

// handleTestWebhook sends a test event to a webhook
func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	webhook, err := s.webhookRegistry.Get(r.Context(), webhookID)
	if err != nil {
		respondError(w, http.StatusNotFound, "webhook not found", err)
		return
	}

	// Create test event
	testPayload := map[string]interface{}{
		"test":       true,
		"message":    "This is a test webhook delivery",
		"webhook_id": webhookID,
		"timestamp":  time.Now().UTC(),
	}

	event := &webhooks.WebhookEvent{
		ID:          uuid.New().String(),
		WebhookID:   webhook.ID,
		EventType:   "test.event",
		Payload:     testPayload,
		Timestamp:   time.Now().UTC(),
		DeliveryURL: webhook.URL,
	}

	// Dispatch test event
	if err := s.webhookDelivery.Dispatch(event); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to dispatch test webhook", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Test webhook dispatched",
		"event_id": event.ID,
	})
}

// handleWebhookDeliveryAttempts retrieves delivery attempts for a webhook
func (s *Server) handleWebhookDeliveryAttempts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	// Query delivery attempts from database
	query := `
		SELECT
			wa.id, wa.event_id, wa.attempt_number,
			wa.status_code, wa.success, wa.error_message,
			wa.duration_ms, wa.attempted_at, wa.next_retry_at,
			we.event_type
		FROM webhook_attempts wa
		JOIN webhook_events we ON wa.event_id = we.id
		WHERE wa.webhook_id = $1
		ORDER BY wa.attempted_at DESC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(r.Context(), query, webhookID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query delivery attempts", err)
		return
	}
	defer rows.Close()

	attempts := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, eventID, errorMessage string
			attemptNumber, statusCode int
			durationMS                int64
			success                   bool
			attemptedAt               time.Time
			nextRetryAt               *time.Time
			eventType                 string
		)

		err := rows.Scan(
			&id, &eventID, &attemptNumber,
			&statusCode, &success, &errorMessage,
			&durationMS, &attemptedAt, &nextRetryAt,
			&eventType,
		)
		if err != nil {
			continue
		}

		attempt := map[string]interface{}{
			"id":             id,
			"event_id":       eventID,
			"event_type":     eventType,
			"attempt_number": attemptNumber,
			"status_code":    statusCode,
			"success":        success,
			"error_message":  errorMessage,
			"duration_ms":    durationMS,
			"attempted_at":   attemptedAt,
			"next_retry_at":  nextRetryAt,
		}

		attempts = append(attempts, attempt)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"webhook_id": webhookID,
		"attempts":   attempts,
		"count":      len(attempts),
	})
}

// Import required packages
import (
	"time"

	"github.com/google/uuid"
)
