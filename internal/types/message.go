package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageType represents the type of cross-chain message
type MessageType string

const (
	MessageTypeTokenTransfer MessageType = "TOKEN_TRANSFER"
	MessageTypeNFTTransfer   MessageType = "NFT_TRANSFER"
	MessageTypeGeneric       MessageType = "GENERIC_MESSAGE"
)

// MessageStatus represents the processing status of a message
type MessageStatus string

const (
	MessageStatusPending    MessageStatus = "PENDING"
	MessageStatusValidating MessageStatus = "VALIDATING"
	MessageStatusProcessing MessageStatus = "PROCESSING"
	MessageStatusCompleted  MessageStatus = "COMPLETED"
	MessageStatusFailed     MessageStatus = "FAILED"
	MessageStatusRetrying   MessageStatus = "RETRYING"
)

// CrossChainMessage represents a universal cross-chain message
type CrossChainMessage struct {
	// Message identification
	ID    string      `json:"id" db:"id"`
	Type  MessageType `json:"type" db:"message_type"`
	Nonce uint64      `json:"nonce" db:"nonce"`

	// Source chain info
	SourceChain  ChainInfo `json:"source_chain" db:"-"`
	SourceTxHash string    `json:"source_tx_hash" db:"source_tx_hash"`
	SourceBlock  uint64    `json:"source_block" db:"source_block"`

	// Destination chain info
	DestinationChain ChainInfo `json:"destination_chain" db:"-"`
	DestTxHash       string    `json:"dest_tx_hash,omitempty" db:"dest_tx_hash"`
	DestBlock        uint64    `json:"dest_block,omitempty" db:"dest_block"`

	// Addresses
	Sender    Address `json:"sender" db:"-"`
	Recipient Address `json:"recipient" db:"-"`

	// Payload (serialized)
	Payload        json.RawMessage        `json:"payload" db:"payload"`
	DecodedPayload interface{}            `json:"decoded_payload,omitempty" db:"-"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	// Processing state
	Status    MessageStatus `json:"status" db:"status"`
	Attempts  int           `json:"attempts" db:"attempts"`
	LastError string        `json:"last_error,omitempty" db:"last_error"`

	// Validation
	ValidatorSignatures []ValidatorSignature `json:"validator_signatures" db:"-"`
	RequiredSignatures  int                  `json:"required_signatures" db:"required_signatures"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" db:"processed_at"`
}

// NewCrossChainMessage creates a new cross-chain message
func NewCrossChainMessage(
	msgType MessageType,
	sourceChain, destChain ChainInfo,
	sender, recipient Address,
	payload interface{},
) (*CrossChainMessage, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &CrossChainMessage{
		ID:               uuid.New().String(),
		Type:             msgType,
		SourceChain:      sourceChain,
		DestinationChain: destChain,
		Sender:           sender,
		Recipient:        recipient,
		Payload:          payloadBytes,
		DecodedPayload:   payload,
		Status:           MessageStatusPending,
		Attempts:         0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         make(map[string]interface{}),
	}, nil
}

// DecodePayload decodes the message payload into the appropriate struct
func (m *CrossChainMessage) DecodePayload() error {
	switch m.Type {
	case MessageTypeTokenTransfer:
		var payload TokenTransferPayload
		if err := json.Unmarshal(m.Payload, &payload); err != nil {
			return err
		}
		m.DecodedPayload = payload
	case MessageTypeNFTTransfer:
		var payload NFTTransferPayload
		if err := json.Unmarshal(m.Payload, &payload); err != nil {
			return err
		}
		m.DecodedPayload = payload
	default:
		var payload interface{}
		if err := json.Unmarshal(m.Payload, &payload); err != nil {
			return err
		}
		m.DecodedPayload = payload
	}
	return nil
}

// TokenTransferPayload represents a token transfer payload
type TokenTransferPayload struct {
	TokenAddress  Address `json:"token_address"`
	Amount        string  `json:"amount"`         // String to handle different decimals
	TokenStandard string  `json:"token_standard"` // ERC20, SPL, NEP141
	Decimals      uint8   `json:"decimals"`
	Symbol        string  `json:"symbol,omitempty"`
	Name          string  `json:"name,omitempty"`
}

// NFTTransferPayload represents an NFT transfer payload
type NFTTransferPayload struct {
	ContractAddress Address `json:"contract_address"`
	TokenID         string  `json:"token_id"`
	TokenURI        string  `json:"token_uri"`
	Standard        string  `json:"standard"` // ERC721, ERC1155, Metaplex, NEP171
	Metadata        string  `json:"metadata,omitempty"`
	Amount          uint64  `json:"amount,omitempty"` // For ERC1155
}

// ValidatorSignature represents a validator's signature on a message
type ValidatorSignature struct {
	ValidatorAddress string    `json:"validator_address" db:"validator_address"`
	Signature        []byte    `json:"signature" db:"signature"`
	SignatureScheme  string    `json:"signature_scheme" db:"signature_scheme"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
}

// MessageFilter represents filters for querying messages
type MessageFilter struct {
	SourceChain      string
	DestinationChain string
	Status           MessageStatus
	MessageType      MessageType
	Sender           string
	Recipient        string
	FromBlock        uint64
	ToBlock          uint64
	Limit            int
	Offset           int
}

// MessageStats represents statistics about bridge messages
type MessageStats struct {
	TotalMessages      int64         `json:"total_messages"`
	PendingMessages    int64         `json:"pending_messages"`
	ProcessingMessages int64         `json:"processing_messages"`
	CompletedMessages  int64         `json:"completed_messages"`
	FailedMessages     int64         `json:"failed_messages"`
	AvgProcessingTime  time.Duration `json:"avg_processing_time"`
	TotalVolume        string        `json:"total_volume"` // In USD
	LastProcessedAt    *time.Time    `json:"last_processed_at,omitempty"`
}
