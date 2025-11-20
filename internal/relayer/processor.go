package relayer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/crypto"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/monitoring"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/security"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/gagliardetto/solana-go"
	"github.com/rs/zerolog"
)

// Processor processes cross-chain messages and broadcasts them to destination chains
type Processor struct {
	clients   map[string]types.UniversalClient
	signers   map[string]crypto.UniversalSigner
	db        *database.DB
	config    *config.Config
	validator *security.Validator
	logger    zerolog.Logger
	chainCfg  map[string]*types.ChainConfig
}

// NewProcessor creates a new message processor
func NewProcessor(
	clients map[string]types.UniversalClient,
	signers map[string]crypto.UniversalSigner,
	db *database.DB,
	cfg *config.Config,
	validator *security.Validator,
	logger zerolog.Logger,
) *Processor {
	chainCfg := make(map[string]*types.ChainConfig)
	for _, chain := range cfg.Chains {
		chainCfg[chain.Name] = &chain
	}

	return &Processor{
		clients:   clients,
		signers:   signers,
		db:        db,
		config:    cfg,
		validator: validator,
		logger:    logger.With().Str("component", "processor").Logger(),
		chainCfg:  chainCfg,
	}
}

// ProcessMessage processes a cross-chain message
func (p *Processor) ProcessMessage(ctx context.Context, msg *types.CrossChainMessage) error {
	startTime := time.Now()
	defer func() {
		monitoring.MessagesProcessingDuration.WithLabelValues(
			msg.SourceChain.Name,
			msg.DestinationChain.Name,
		).Observe(time.Since(startTime).Seconds())
	}()

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("source", msg.SourceChain.Name).
		Str("destination", msg.DestinationChain.Name).
		Str("type", string(msg.Type)).
		Msg("Processing message")

	// Validate message security
	if err := p.validator.ValidateMessage(ctx, msg); err != nil {
		p.logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Msg("Message failed security validation")
		monitoring.MessagesTotal.WithLabelValues(msg.SourceChain.Name, msg.DestinationChain.Name, string(msg.Type), "failed").Inc()
		return fmt.Errorf("security validation failed: %w", err)
	}

	// Check if message already processed
	status, err := p.db.GetMessageStatus(ctx, msg.ID)
	if err == nil && status == types.MessageStatusCompleted {
		p.logger.Warn().
			Str("message_id", msg.ID).
			Msg("Message already processed, skipping")
		return nil
	}

	// Verify validator signatures
	if err := p.verifySignatures(ctx, msg); err != nil {
		p.logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Msg("Signature verification failed")
		monitoring.MessagesTotal.WithLabelValues(msg.SourceChain.Name, msg.DestinationChain.Name, string(msg.Type), "failed").Inc()
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Process based on destination chain type
	destClient, ok := p.clients[msg.DestinationChain.Name]
	if !ok {
		return fmt.Errorf("client not found for chain: %s", msg.DestinationChain.Name)
	}

	var txHash string
	switch destClient.GetChainType() {
	case types.ChainTypeEVM:
		txHash, err = p.processEVMMessage(ctx, msg, destClient)
	case types.ChainTypeSolana:
		txHash, err = p.processSolanaMessage(ctx, msg, destClient)
	case types.ChainTypeNEAR:
		txHash, err = p.processNEARMessage(ctx, msg, destClient)
	default:
		return fmt.Errorf("unsupported chain type: %s", destClient.GetChainType())
	}

	if err != nil {
		p.logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Msg("Failed to broadcast transaction")
		monitoring.MessagesTotal.WithLabelValues(msg.SourceChain.Name, msg.DestinationChain.Name, string(msg.Type), "failed").Inc()
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Update message status
	if err := p.db.UpdateMessageStatus(ctx, msg.ID, types.MessageStatusCompleted, txHash); err != nil {
		p.logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Msg("Failed to update message status")
		// Don't return error - transaction was broadcast successfully
	}

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("tx_hash", txHash).
		Str("destination", msg.DestinationChain.Name).
		Msg("Message processed successfully")

	duration := time.Since(startTime).Seconds()
	monitoring.RecordMessageProcessed(msg.SourceChain.Name, msg.DestinationChain.Name, string(msg.Type), "completed", duration)
	return nil
}

// verifySignatures verifies validator signatures on the message
func (p *Processor) verifySignatures(ctx context.Context, msg *types.CrossChainMessage) error {
	// Get required signature threshold based on environment
	requiredSigs := p.config.Security.RequiredSignatures

	if len(msg.ValidatorSignatures) < requiredSigs {
		return fmt.Errorf("insufficient signatures: got %d, need %d",
			len(msg.ValidatorSignatures), requiredSigs)
	}

	// Verify each signature
	validSigs := 0
	seenValidators := make(map[string]bool)

	for _, sig := range msg.ValidatorSignatures {
		// Check for duplicate validators
		if seenValidators[sig.ValidatorAddress] {
			p.logger.Warn().
				Str("validator", sig.ValidatorAddress).
				Msg("Duplicate signature from validator")
			continue
		}
		seenValidators[sig.ValidatorAddress] = true

		// Verify signature
		if err := p.verifyValidatorSignature(ctx, msg, &sig); err != nil {
			p.logger.Warn().
				Err(err).
				Str("validator", sig.ValidatorAddress).
				Msg("Invalid signature from validator")
			continue
		}

		validSigs++
	}

	if validSigs < requiredSigs {
		return fmt.Errorf("insufficient valid signatures: got %d, need %d",
			validSigs, requiredSigs)
	}

	p.logger.Info().
		Str("message_id", msg.ID).
		Int("valid_signatures", validSigs).
		Int("required", requiredSigs).
		Msg("Signature verification passed")

	return nil
}

// verifyValidatorSignature verifies a single validator signature
func (p *Processor) verifyValidatorSignature(ctx context.Context, msg *types.CrossChainMessage, sig *types.ValidatorSignature) error {
	// Create message hash for verification
	msgHash, err := p.createMessageHash(msg)
	if err != nil {
		return fmt.Errorf("failed to create message hash: %w", err)
	}

	// Get validator's chain type to determine signature scheme
	// For simplicity, we'll use the source chain's type
	// In production, you might want to use a specific validator registry
	sourceClient := p.clients[msg.SourceChain.Name]
	chainType := sourceClient.GetChainType()

	// Convert signature to hex string
	sigHex := hex.EncodeToString(sig.Signature)

	// Verify signature based on chain type
	switch chainType {
	case types.ChainTypeEVM:
		return crypto.VerifyECDSASignature(msgHash, sigHex, sig.ValidatorAddress)
	case types.ChainTypeSolana, types.ChainTypeNEAR:
		return crypto.VerifyEd25519Signature(msgHash, sigHex, sig.ValidatorAddress)
	default:
		return fmt.Errorf("unsupported chain type for signature verification")
	}
}

// createMessageHash creates a deterministic hash of the message for signing
func (p *Processor) createMessageHash(msg *types.CrossChainMessage) ([]byte, error) {
	// Create a canonical representation of the message
	data := struct {
		ID            string
		Type          types.MessageType
		SourceChainID string
		DestChainID   string
		Sender        string
		Recipient     string
		Payload       json.RawMessage
		Nonce         uint64
		Timestamp     int64
	}{
		ID:            msg.ID,
		Type:          msg.Type,
		SourceChainID: msg.SourceChain.ChainID,
		DestChainID:   msg.DestinationChain.ChainID,
		Sender:        msg.Sender.Raw,
		Recipient:     msg.Recipient.Raw,
		Payload:       msg.Payload,
		Nonce:         msg.Nonce,
		Timestamp:     msg.CreatedAt.Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return crypto.Keccak256(jsonData), nil
}

// processEVMMessage processes a message for EVM chains
func (p *Processor) processEVMMessage(ctx context.Context, msg *types.CrossChainMessage, client types.UniversalClient) (string, error) {
	p.logger.Debug().
		Str("message_id", msg.ID).
		Msg("Processing EVM message")

	// Get chain configuration
	chainCfg, ok := p.chainCfg[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("chain config not found: %s", msg.DestinationChain.Name)
	}

	// Get signer for this chain
	signer, ok := p.signers[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("signer not found for chain: %s", msg.DestinationChain.Name)
	}

	// Build transaction based on message type
	var tx *ethTypes.Transaction
	var err error

	switch msg.Type {
	case types.MessageTypeTokenTransfer:
		tx, err = p.buildEVMTokenUnlockTx(msg, chainCfg)
	case types.MessageTypeNFTTransfer:
		tx, err = p.buildEVMNFTUnlockTx(msg, chainCfg)
	default:
		return "", fmt.Errorf("unsupported message type: %s", msg.Type)
	}

	if err != nil {
		return "", fmt.Errorf("failed to build transaction: %w", err)
	}

	// Sign transaction
	signedTx, err := signer.SignTransaction(ctx, tx, msg.DestinationChain.ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Broadcast transaction
	txHash, err := client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for confirmation if needed
	if chainCfg.ConfirmationBlocks > 0 {
		p.logger.Debug().
			Str("tx_hash", txHash).
			Uint64("confirmations", chainCfg.ConfirmationBlocks).
			Msg("Waiting for transaction confirmation")

		// In production, you would implement proper confirmation waiting
		// For now, we'll just return the tx hash
	}

	return txHash, nil
}

// buildEVMTokenUnlockTx builds a token unlock transaction for EVM chains
func (p *Processor) buildEVMTokenUnlockTx(msg *types.CrossChainMessage, chainCfg *types.ChainConfig) (*ethTypes.Transaction, error) {
	// Parse payload
	var payload types.TokenTransferPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Build contract call data
	// unlockToken(bytes32 messageId, address recipient, address token, uint256 amount, bytes[] signatures)

	contractABI, err := abi.JSON(nil) // In production, load actual bridge ABI
	if err != nil {
		return nil, err
	}

	// Prepare signature array
	signatures := make([][]byte, len(msg.ValidatorSignatures))
	for i, sig := range msg.ValidatorSignatures {
		signatures[i] = []byte(sig.Signature)
	}

	// Parse amount
	amount := new(big.Int)
	amount.SetString(payload.Amount, 10)

	// Pack function call
	data, err := contractABI.Pack(
		"unlockToken",
		[32]byte{}, // messageId (convert msg.ID to bytes32)
		common.HexToAddress(msg.Recipient.Raw),
		common.HexToAddress(payload.TokenAddress.Raw),
		amount,
		signatures,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack function call: %w", err)
	}

	// Create transaction
	// In production, you would:
	// 1. Get nonce from account
	// 2. Estimate gas
	// 3. Get current gas price
	// 4. Build proper transaction with all parameters

	tx := ethTypes.NewTransaction(
		0, // nonce - should be fetched
		common.HexToAddress(chainCfg.BridgeContract),
		big.NewInt(0),           // value
		300000,                  // gas limit - should be estimated
		big.NewInt(20000000000), // gas price - should be fetched
		data,
	)

	return tx, nil
}

// buildEVMNFTUnlockTx builds an NFT unlock transaction for EVM chains
func (p *Processor) buildEVMNFTUnlockTx(msg *types.CrossChainMessage, chainCfg *types.ChainConfig) (*ethTypes.Transaction, error) {
	// Parse payload
	var payload types.NFTTransferPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Similar to token unlock but for NFTs
	// unlockNFT(bytes32 messageId, address recipient, address nftContract, uint256 tokenId, bytes[] signatures)

	// Implementation similar to buildEVMTokenUnlockTx
	// Returning placeholder for now
	return nil, fmt.Errorf("NFT unlock not fully implemented")
}

// processSolanaMessage processes a message for Solana
func (p *Processor) processSolanaMessage(ctx context.Context, msg *types.CrossChainMessage, client types.UniversalClient) (string, error) {
	p.logger.Debug().
		Str("message_id", msg.ID).
		Msg("Processing Solana message")

	// Get chain configuration
	chainCfg, ok := p.chainCfg[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("chain config not found: %s", msg.DestinationChain.Name)
	}

	// Get signer for Solana
	signer, ok := p.signers[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("signer not found for Solana")
	}

	// Build transaction based on message type
	var tx interface{}
	var err error

	switch msg.Type {
	case types.MessageTypeTokenTransfer:
		tx, err = p.buildSolanaTokenUnlockTx(ctx, msg, chainCfg, signer)
	case types.MessageTypeNFTTransfer:
		tx, err = p.buildSolanaNFTUnlockTx(ctx, msg, chainCfg, signer)
	default:
		return "", fmt.Errorf("unsupported message type: %s", msg.Type)
	}

	if err != nil {
		return "", fmt.Errorf("failed to build transaction: %w", err)
	}

	// Send transaction
	txHash, err := client.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("signature", txHash).
		Msg("Solana transaction sent")

	// Wait for confirmation if needed
	if chainCfg.ConfirmationBlocks > 0 {
		p.logger.Debug().
			Str("signature", txHash).
			Uint64("confirmations", chainCfg.ConfirmationBlocks).
			Msg("Waiting for Solana transaction confirmation")

		if err := client.WaitForConfirmation(ctx, txHash, 60*time.Second); err != nil {
			p.logger.Warn().
				Err(err).
				Str("signature", txHash).
				Msg("Confirmation wait failed, but transaction was sent")
			// Don't fail - transaction was broadcast
		}
	}

	return txHash, nil
}

// processNEARMessage processes a message for NEAR
func (p *Processor) processNEARMessage(ctx context.Context, msg *types.CrossChainMessage, client types.UniversalClient) (string, error) {
	p.logger.Debug().
		Str("message_id", msg.ID).
		Msg("Processing NEAR message")

	// Get chain configuration
	chainCfg, ok := p.chainCfg[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("chain config not found: %s", msg.DestinationChain.Name)
	}

	// Get signer for NEAR
	signer, ok := p.signers[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("signer not found for NEAR")
	}

	// Build transaction based on message type
	var tx interface{}
	var err error

	switch msg.Type {
	case types.MessageTypeTokenTransfer:
		tx, err = p.buildNEARTokenUnlockTx(ctx, msg, chainCfg, signer)
	case types.MessageTypeNFTTransfer:
		tx, err = p.buildNEARNFTUnlockTx(ctx, msg, chainCfg, signer)
	default:
		return "", fmt.Errorf("unsupported message type: %s", msg.Type)
	}

	if err != nil {
		return "", fmt.Errorf("failed to build transaction: %w", err)
	}

	// Send transaction
	txHash, err := client.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("tx_hash", txHash).
		Msg("NEAR transaction sent")

	// Wait for confirmation if needed
	if chainCfg.ConfirmationBlocks > 0 {
		p.logger.Debug().
			Str("tx_hash", txHash).
			Uint64("confirmations", chainCfg.ConfirmationBlocks).
			Msg("Waiting for NEAR transaction confirmation")

		if err := client.WaitForConfirmation(ctx, txHash, 60*time.Second); err != nil {
			p.logger.Warn().
				Err(err).
				Str("tx_hash", txHash).
				Msg("Confirmation wait failed, but transaction was sent")
			// Don't fail - transaction was broadcast
		}
	}

	return txHash, nil
}

// buildSolanaTokenUnlockTx builds a Solana token unlock transaction
func (p *Processor) buildSolanaTokenUnlockTx(ctx context.Context, msg *types.CrossChainMessage, chainCfg *types.ChainConfig, signer crypto.UniversalSigner) (*solana.Transaction, error) {
	// Parse payload
	var payload types.TokenTransferPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Parse bridge program ID
	bridgeProgramID, err := solana.PublicKeyFromBase58(chainCfg.BridgeContract)
	if err != nil {
		return nil, fmt.Errorf("invalid bridge program ID: %w", err)
	}

	// Parse recipient public key
	recipientPubkey, err := solana.PublicKeyFromBase58(msg.Recipient.Raw)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient public key: %w", err)
	}

	// Parse token mint address
	tokenMint, err := solana.PublicKeyFromBase58(payload.TokenAddress.Raw)
	if err != nil {
		return nil, fmt.Errorf("invalid token mint address: %w", err)
	}

	// Get signer's public key
	signerPubkey, err := p.getSolanaSignerPublicKey(signer)
	if err != nil {
		return nil, fmt.Errorf("failed to get signer public key: %w", err)
	}

	// Build unlock instruction data
	// Format: [unlock_discriminator(8), message_id(32), amount(8), signatures_count(1), signatures...]
	instructionData := make([]byte, 0)

	// Add discriminator for "unlock_token" instruction (simplified)
	unlockDiscriminator := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	instructionData = append(instructionData, unlockDiscriminator...)

	// Add message ID (convert to 32 bytes)
	messageIDBytes := []byte(msg.ID)
	if len(messageIDBytes) > 32 {
		messageIDBytes = messageIDBytes[:32]
	} else {
		// Pad to 32 bytes
		for len(messageIDBytes) < 32 {
			messageIDBytes = append(messageIDBytes, 0)
		}
	}
	instructionData = append(instructionData, messageIDBytes...)

	// Parse and add amount (8 bytes, little endian)
	amount := new(big.Int)
	amount.SetString(payload.Amount, 10)
	amountBytes := make([]byte, 8)
	amountBytesSlice := amount.Bytes()
	// Copy to little endian
	for i := 0; i < len(amountBytesSlice) && i < 8; i++ {
		amountBytes[i] = amountBytesSlice[len(amountBytesSlice)-1-i]
	}
	instructionData = append(instructionData, amountBytes...)

	// Add number of validator signatures
	instructionData = append(instructionData, byte(len(msg.ValidatorSignatures)))

	// Add validator signatures
	for _, sig := range msg.ValidatorSignatures {
		sigBytes := []byte(sig.Signature)
		// Ensure signature is exactly 64 bytes for Ed25519
		if len(sigBytes) > 64 {
			sigBytes = sigBytes[:64]
		}
		instructionData = append(instructionData, sigBytes...)
	}

	// Derive bridge vault PDA (Program Derived Address)
	// Seeds: ["vault", token_mint]
	vaultSeeds := [][]byte{
		[]byte("vault"),
		tokenMint.Bytes(),
	}
	vaultPDA, _, err := solana.FindProgramAddress(vaultSeeds, bridgeProgramID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive vault PDA: %w", err)
	}

	// Derive recipient token account (Associated Token Account)
	recipientTokenAccount, _, err := solana.FindAssociatedTokenAddress(recipientPubkey, tokenMint)
	if err != nil {
		return nil, fmt.Errorf("failed to derive recipient token account: %w", err)
	}

	// Build instruction with all required accounts
	// Note: solana.Instruction type may vary by library version
	// Using GenericInstruction or wrapping in NewInstruction if needed
	instruction := solana.GenericInstruction{
		ProgID: bridgeProgramID,
		AccountValues: []*solana.AccountMeta{
			{PublicKey: signerPubkey, IsSigner: true, IsWritable: false},            // Relayer signer
			{PublicKey: bridgeProgramID, IsSigner: false, IsWritable: false},        // Bridge program
			{PublicKey: vaultPDA, IsSigner: false, IsWritable: true},                // Token vault
			{PublicKey: recipientPubkey, IsSigner: false, IsWritable: false},        // Recipient
			{PublicKey: recipientTokenAccount, IsSigner: false, IsWritable: true},   // Recipient token account
			{PublicKey: tokenMint, IsSigner: false, IsWritable: false},              // Token mint
			{PublicKey: solana.TokenProgramID, IsSigner: false, IsWritable: false},  // Token program
			{PublicKey: solana.SystemProgramID, IsSigner: false, IsWritable: false}, // System program
		},
		DataBytes: instructionData,
	}

	// Get recent blockhash (would need to call Solana client)
	// For now, use a placeholder
	recentBlockhash := solana.Hash{} // In production, fetch from network

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{&instruction},
		recentBlockhash,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("recipient", recipientPubkey.String()).
		Str("amount", payload.Amount).
		Msg("Built Solana token unlock transaction")

	return tx, nil
}

// buildSolanaNFTUnlockTx builds a Solana NFT unlock transaction
func (p *Processor) buildSolanaNFTUnlockTx(ctx context.Context, msg *types.CrossChainMessage, chainCfg *types.ChainConfig, signer crypto.UniversalSigner) (*solana.Transaction, error) {
	// Parse payload
	var payload types.NFTTransferPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Similar to token unlock but for NFTs
	// Would use Metaplex standard for NFT transfers

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("nft_contract", payload.ContractAddress.Raw).
		Str("token_id", payload.TokenID).
		Msg("Building Solana NFT unlock transaction")

	// Placeholder - full implementation would handle Metaplex NFT standard
	return nil, fmt.Errorf("Solana NFT unlock not fully implemented")
}

// buildNEARTokenUnlockTx builds a NEAR token unlock transaction
func (p *Processor) buildNEARTokenUnlockTx(ctx context.Context, msg *types.CrossChainMessage, chainCfg *types.ChainConfig, signer crypto.UniversalSigner) ([]byte, error) {
	// Parse payload
	var payload types.TokenTransferPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Build NEAR function call transaction
	// Method: unlock_token
	// Args: {message_id, recipient, token, amount, signatures}

	type UnlockArgs struct {
		MessageID  string   `json:"message_id"`
		Recipient  string   `json:"recipient"`
		Token      string   `json:"token"`
		Amount     string   `json:"amount"`
		Signatures []string `json:"signatures"`
	}

	// Collect validator signatures
	signatures := make([]string, len(msg.ValidatorSignatures))
	for i, sig := range msg.ValidatorSignatures {
		signatures[i] = hex.EncodeToString(sig.Signature)
	}

	args := UnlockArgs{
		MessageID:  msg.ID,
		Recipient:  msg.Recipient.Raw,
		Token:      payload.TokenAddress.Raw,
		Amount:     payload.Amount,
		Signatures: signatures,
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal args: %w", err)
	}

	// Build NEAR transaction
	// In production, you would:
	// 1. Get signer account ID
	// 2. Get nonce (access key nonce)
	// 3. Get recent block hash
	// 4. Build proper transaction structure
	// 5. Sign with Ed25519

	// Simplified transaction structure
	type NEARAction struct {
		FunctionCall struct {
			MethodName string `json:"method_name"`
			Args       string `json:"args"`
			Gas        uint64 `json:"gas"`
			Deposit    string `json:"deposit"`
		} `json:"FunctionCall"`
	}

	type NEARTransaction struct {
		SignerID   string       `json:"signer_id"`
		PublicKey  string       `json:"public_key"`
		Nonce      uint64       `json:"nonce"`
		ReceiverID string       `json:"receiver_id"`
		Actions    []NEARAction `json:"actions"`
		BlockHash  string       `json:"block_hash"`
	}

	// This would be properly constructed in production
	tx := NEARTransaction{
		SignerID:   "relayer.near", // Would get from signer
		ReceiverID: chainCfg.BridgeContract,
		Actions: []NEARAction{
			{
				FunctionCall: struct {
					MethodName string `json:"method_name"`
					Args       string `json:"args"`
					Gas        uint64 `json:"gas"`
					Deposit    string `json:"deposit"`
				}{
					MethodName: "unlock_token",
					Args:       string(argsJSON),
					Gas:        100000000000000, // 100 TGas
					Deposit:    "0",
				},
			},
		},
		Nonce:     1,             // Would fetch from access key
		BlockHash: "placeholder", // Would fetch recent block hash
	}

	// Serialize and sign transaction
	txBytes, err := json.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// In production, sign with Ed25519 signer
	// signedTx, err := signer.SignTransaction(ctx, txBytes, chainCfg.ChainID)

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("recipient", msg.Recipient.Raw).
		Str("amount", payload.Amount).
		Msg("Built NEAR token unlock transaction")

	return txBytes, nil
}

// buildNEARNFTUnlockTx builds a NEAR NFT unlock transaction
func (p *Processor) buildNEARNFTUnlockTx(ctx context.Context, msg *types.CrossChainMessage, chainCfg *types.ChainConfig, signer crypto.UniversalSigner) ([]byte, error) {
	// Parse payload
	var payload types.NFTTransferPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Similar to token unlock but for NFTs using NEP-171 standard
	type UnlockNFTArgs struct {
		MessageID   string   `json:"message_id"`
		Recipient   string   `json:"recipient"`
		NFTContract string   `json:"nft_contract"`
		TokenID     string   `json:"token_id"`
		Signatures  []string `json:"signatures"`
	}

	// Collect validator signatures
	signatures := make([]string, len(msg.ValidatorSignatures))
	for i, sig := range msg.ValidatorSignatures {
		signatures[i] = hex.EncodeToString(sig.Signature)
	}

	args := UnlockNFTArgs{
		MessageID:   msg.ID,
		Recipient:   msg.Recipient.Raw,
		NFTContract: payload.ContractAddress.Raw,
		TokenID:     payload.TokenID,
		Signatures:  signatures,
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal args: %w", err)
	}

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("nft_contract", payload.ContractAddress.Raw).
		Str("token_id", payload.TokenID).
		Msg("Built NEAR NFT unlock transaction")

	// Build similar transaction structure as token unlock
	// Placeholder implementation
	return argsJSON, nil
}

// getSolanaSignerPublicKey extracts the public key from a Solana signer
func (p *Processor) getSolanaSignerPublicKey(signer crypto.UniversalSigner) (solana.PublicKey, error) {
	// Get public key from signer
	// This would depend on the signer implementation
	// For now, return a placeholder
	return solana.PublicKey{}, nil // Would extract from actual signer
}
