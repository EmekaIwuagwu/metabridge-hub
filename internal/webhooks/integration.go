package webhooks

import (
	"context"
	"fmt"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/rs/zerolog"
)

// Notifier handles webhook notifications for message events
type Notifier struct {
	delivery        *DeliveryService
	trackingService *TrackingService
	logger          zerolog.Logger
}

// NewNotifier creates a new webhook notifier
func NewNotifier(
	delivery *DeliveryService,
	trackingService *TrackingService,
	logger zerolog.Logger,
) *Notifier {
	return &Notifier{
		delivery:        delivery,
		trackingService: trackingService,
		logger:          logger.With().Str("component", "webhook-notifier").Logger(),
	}
}

// NotifyMessageCreated sends webhook notifications when a message is created
func (n *Notifier) NotifyMessageCreated(ctx context.Context, message *types.CrossChainMessage) error {
	payload := map[string]interface{}{
		"message":      message,
		"source_chain": message.SourceChain,
		"dest_chain":   message.DestinationChain,
		"sender":       message.Sender,
		"recipient":    message.Recipient,
		"status":       message.Status,
	}

	event := &MessageEvent{
		Message:   message,
		Event:     EventMessageCreated,
		Timestamp: message.CreatedAt,
	}

	return n.dispatchEvent(ctx, EventMessageCreated, payload, event)
}

// NotifyMessagePending sends webhook notifications when a message is pending
func (n *Notifier) NotifyMessagePending(ctx context.Context, message *types.CrossChainMessage) error {
	payload := map[string]interface{}{
		"message":      message,
		"source_chain": message.SourceChain,
		"dest_chain":   message.DestinationChain,
		"status":       message.Status,
	}

	event := &MessageEvent{
		Message:   message,
		Event:     EventMessagePending,
		Timestamp: message.UpdatedAt,
	}

	return n.dispatchEvent(ctx, EventMessagePending, payload, event)
}

// NotifyMessageSubmitted sends webhook notifications when a message is submitted
func (n *Notifier) NotifyMessageSubmitted(ctx context.Context, message *types.CrossChainMessage, txHash string, blockNumber uint64) error {
	payload := map[string]interface{}{
		"message":      message,
		"source_chain": message.SourceChain,
		"dest_chain":   message.DestinationChain,
		"tx_hash":      txHash,
		"block_number": blockNumber,
		"status":       message.Status,
	}

	event := &MessageEvent{
		Message:     message,
		Event:       EventMessageSubmitted,
		Timestamp:   message.UpdatedAt,
		TxHash:      txHash,
		BlockNumber: blockNumber,
	}

	// Record timeline event
	if err := n.recordTimelineEvent(ctx, message.ID, "message_submitted", "Message submitted to destination chain", txHash, blockNumber, message.DestinationChain.ChainID); err != nil {
		n.logger.Warn().Err(err).Msg("Failed to record timeline event")
	}

	return n.dispatchEvent(ctx, EventMessageSubmitted, payload, event)
}

// NotifyMessageConfirmed sends webhook notifications when a message is confirmed
func (n *Notifier) NotifyMessageConfirmed(ctx context.Context, message *types.CrossChainMessage, confirmations uint64) error {
	payload := map[string]interface{}{
		"message":       message,
		"source_chain":  message.SourceChain,
		"dest_chain":    message.DestinationChain,
		"tx_hash":       message.DestTxHash,
		"confirmations": confirmations,
		"status":        message.Status,
	}

	event := &MessageEvent{
		Message:       message,
		Event:         EventMessageConfirmed,
		Timestamp:     message.UpdatedAt,
		TxHash:        message.DestTxHash,
		Confirmations: confirmations,
	}

	// Record timeline event
	if err := n.recordTimelineEvent(ctx, message.ID, "message_confirmed", fmt.Sprintf("Message confirmed with %d confirmations", confirmations), message.DestTxHash, 0, message.DestinationChain.ChainID); err != nil {
		n.logger.Warn().Err(err).Msg("Failed to record timeline event")
	}

	return n.dispatchEvent(ctx, EventMessageConfirmed, payload, event)
}

// NotifyMessageFinalized sends webhook notifications when a message is finalized
func (n *Notifier) NotifyMessageFinalized(ctx context.Context, message *types.CrossChainMessage) error {
	payload := map[string]interface{}{
		"message":      message,
		"source_chain": message.SourceChain,
		"dest_chain":   message.DestinationChain,
		"tx_hash":      message.DestTxHash,
		"status":       message.Status,
	}

	event := &MessageEvent{
		Message:   message,
		Event:     EventMessageFinalized,
		Timestamp: message.UpdatedAt,
		TxHash:    message.DestTxHash,
	}

	// Record timeline event
	if err := n.recordTimelineEvent(ctx, message.ID, "message_finalized", "Message finalized on destination chain", message.DestTxHash, 0, message.DestinationChain.ChainID); err != nil {
		n.logger.Warn().Err(err).Msg("Failed to record timeline event")
	}

	return n.dispatchEvent(ctx, EventMessageFinalized, payload, event)
}

// NotifyMessageFailed sends webhook notifications when a message fails
func (n *Notifier) NotifyMessageFailed(ctx context.Context, message *types.CrossChainMessage, errorMsg string) error {
	payload := map[string]interface{}{
		"message":       message,
		"source_chain":  message.SourceChain,
		"dest_chain":    message.DestinationChain,
		"error_message": errorMsg,
		"status":        message.Status,
	}

	event := &MessageEvent{
		Message:      message,
		Event:        EventMessageFailed,
		Timestamp:    message.UpdatedAt,
		ErrorMessage: errorMsg,
	}

	// Record timeline event
	if err := n.recordTimelineEvent(ctx, message.ID, "message_failed", fmt.Sprintf("Message failed: %s", errorMsg), "", 0, ""); err != nil {
		n.logger.Warn().Err(err).Msg("Failed to record timeline event")
	}

	return n.dispatchEvent(ctx, EventMessageFailed, payload, event)
}

// NotifyBatchCreated sends webhook notifications when a batch is created
func (n *Notifier) NotifyBatchCreated(ctx context.Context, batchID string, messageCount int) error {
	payload := map[string]interface{}{
		"batch_id":      batchID,
		"message_count": messageCount,
		"event":         "batch_created",
	}

	event := &BatchEvent{
		BatchID:      batchID,
		MessageCount: messageCount,
		Event:        EventBatchCreated,
	}

	return n.dispatchEvent(ctx, EventBatchCreated, payload, event)
}

// NotifyBatchSubmitted sends webhook notifications when a batch is submitted
func (n *Notifier) NotifyBatchSubmitted(ctx context.Context, batchID string, messageCount int, txHash string) error {
	payload := map[string]interface{}{
		"batch_id":      batchID,
		"message_count": messageCount,
		"tx_hash":       txHash,
		"event":         "batch_submitted",
	}

	event := &BatchEvent{
		BatchID:      batchID,
		MessageCount: messageCount,
		Event:        EventBatchSubmitted,
		TxHash:       txHash,
	}

	return n.dispatchEvent(ctx, EventBatchSubmitted, payload, event)
}

// NotifyBatchConfirmed sends webhook notifications when a batch is confirmed
func (n *Notifier) NotifyBatchConfirmed(ctx context.Context, batchID string, messageCount int, txHash string, gasSaved string, savingsPercent float64) error {
	payload := map[string]interface{}{
		"batch_id":        batchID,
		"message_count":   messageCount,
		"tx_hash":         txHash,
		"gas_saved_wei":   gasSaved,
		"savings_percent": savingsPercent,
		"event":           "batch_confirmed",
	}

	event := &BatchEvent{
		BatchID:        batchID,
		MessageCount:   messageCount,
		Event:          EventBatchConfirmed,
		TxHash:         txHash,
		GasSavedWei:    gasSaved,
		SavingsPercent: savingsPercent,
	}

	return n.dispatchEvent(ctx, EventBatchConfirmed, payload, event)
}

// NotifyBatchFailed sends webhook notifications when a batch fails
func (n *Notifier) NotifyBatchFailed(ctx context.Context, batchID string, messageCount int, errorMsg string) error {
	payload := map[string]interface{}{
		"batch_id":      batchID,
		"message_count": messageCount,
		"error_message": errorMsg,
		"event":         "batch_failed",
	}

	event := &BatchEvent{
		BatchID:      batchID,
		MessageCount: messageCount,
		Event:        EventBatchFailed,
		ErrorMessage: errorMsg,
	}

	return n.dispatchEvent(ctx, EventBatchFailed, payload, event)
}

// Helper methods

func (n *Notifier) dispatchEvent(ctx context.Context, eventType EventType, payload map[string]interface{}, event interface{}) error {
	// Record the event type in metrics
	RecordWebhookEvent(eventType)

	// Dispatch to all registered webhooks
	if err := n.delivery.DispatchToWebhooks(ctx, eventType, payload); err != nil {
		n.logger.Error().
			Err(err).
			Str("event_type", string(eventType)).
			Msg("Failed to dispatch webhook event")
		return err
	}

	n.logger.Debug().
		Str("event_type", string(eventType)).
		Msg("Webhook event dispatched successfully")

	return nil
}

func (n *Notifier) recordTimelineEvent(ctx context.Context, messageID, eventType, description, txHash string, blockNumber uint64, chainID string) error {
	event := &TimelineEvent{
		EventType:   eventType,
		Description: description,
		TxHash:      txHash,
		BlockNumber: blockNumber,
		ChainID:     chainID,
		Metadata:    make(map[string]interface{}),
	}

	return n.trackingService.RecordEvent(ctx, messageID, event)
}

// Example usage in message processor:
//
// // In your message processing code:
// func (p *Processor) ProcessMessage(ctx context.Context, message *types.CrossChainMessage) error {
//     // Create the message
//     if err := p.createMessage(ctx, message); err != nil {
//         return err
//     }
//
//     // Notify via webhooks
//     p.notifier.NotifyMessageCreated(ctx, message)
//
//     // Submit to blockchain
//     txHash, err := p.submitToBlockchain(ctx, message)
//     if err != nil {
//         p.notifier.NotifyMessageFailed(ctx, message, err.Error())
//         return err
//     }
//
//     // Update message status
//     message.Status = "SUBMITTED"
//     message.DestTxHash = txHash
//
//     // Notify submission
//     p.notifier.NotifyMessageSubmitted(ctx, message, txHash, blockNumber)
//
//     // Wait for confirmation
//     if err := p.waitForConfirmation(ctx, txHash); err != nil {
//         p.notifier.NotifyMessageFailed(ctx, message, err.Error())
//         return err
//     }
//
//     // Update to confirmed
//     message.Status = "CONFIRMED"
//     p.notifier.NotifyMessageConfirmed(ctx, message, confirmations)
//
//     return nil
// }
