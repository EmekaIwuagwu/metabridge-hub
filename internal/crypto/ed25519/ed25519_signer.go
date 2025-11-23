package ed25519

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/mr-tron/base58"
)

// Ed25519Signer implements UniversalSigner for Solana and NEAR chains
type Ed25519Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	chainType  types.ChainType
}

// KeystoreFile represents the JSON structure of a keystore file
type KeystoreFile struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

// NewEd25519Signer creates a new Ed25519 signer from keystore
func NewEd25519Signer(keystorePath string, password string) (*Ed25519Signer, error) {
	// Read keystore file
	data, err := os.ReadFile(keystorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore: %w", err)
	}

	// Parse JSON
	var keystore KeystoreFile
	if err := json.Unmarshal(data, &keystore); err != nil {
		return nil, fmt.Errorf("failed to parse keystore: %w", err)
	}

	// Decode private key (hex or base58)
	var privateKeyBytes []byte

	// Try hex first
	privateKeyBytes, err = hex.DecodeString(keystore.PrivateKey)
	if err != nil {
		// Try base58
		privateKeyBytes, err = base58.Decode(keystore.PrivateKey)
		if err != nil || len(privateKeyBytes) == 0 {
			return nil, fmt.Errorf("failed to decode private key: %w", err)
		}
	}

	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d",
			ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return &Ed25519Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// NewEd25519SignerFromPrivateKey creates a signer from a private key
func NewEd25519SignerFromPrivateKey(privateKeyHex string) (*Ed25519Signer, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return &Ed25519Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// NewEd25519SignerFromSeed creates a signer from a seed
func NewEd25519SignerFromSeed(seed []byte) (*Ed25519Signer, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid seed size: expected %d, got %d",
			ed25519.SeedSize, len(seed))
	}

	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return &Ed25519Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// GetScheme returns the signature scheme
func (s *Ed25519Signer) GetScheme() types.SignatureScheme {
	return types.SignatureSchemeEd25519
}

// GetPublicKey returns the public key bytes
func (s *Ed25519Signer) GetPublicKey() ([]byte, error) {
	return []byte(s.publicKey), nil
}

// GetAddress returns the address for the given chain type
func (s *Ed25519Signer) GetAddress(chainType types.ChainType) (string, error) {
	switch chainType {
	case types.ChainTypeSolana:
		// Solana address is base58 encoded public key
		return base58.Encode(s.publicKey), nil

	case types.ChainTypeNEAR:
		// NEAR uses hex encoded public key with "ed25519:" prefix
		return fmt.Sprintf("ed25519:%s", hex.EncodeToString(s.publicKey)), nil

	default:
		return "", fmt.Errorf("Ed25519 signer does not support chain type: %s", chainType)
	}
}

// Sign signs arbitrary data using Ed25519
func (s *Ed25519Signer) Sign(ctx context.Context, data []byte) ([]byte, error) {
	signature := ed25519.Sign(s.privateKey, data)
	if len(signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("invalid signature size: %d", len(signature))
	}
	return signature, nil
}

// SignTransaction signs a chain-specific transaction
func (s *Ed25519Signer) SignTransaction(ctx context.Context, tx interface{}, chainID string) (interface{}, error) {
	// For Solana and NEAR, transaction signing is chain-specific
	// This method should be implemented by chain-specific wrappers
	return nil, fmt.Errorf("SignTransaction not implemented for Ed25519 - use chain-specific methods")
}

// Verify verifies an Ed25519 signature
func (s *Ed25519Signer) Verify(data []byte, signature []byte, publicKey []byte) (bool, error) {
	if len(signature) != ed25519.SignatureSize {
		return false, fmt.Errorf("invalid signature size: %d", len(signature))
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key size: %d", len(publicKey))
	}

	return ed25519.Verify(ed25519.PublicKey(publicKey), data, signature), nil
}

// Close clears sensitive data
func (s *Ed25519Signer) Close() error {
	// Zero out private key
	for i := range s.privateKey {
		s.privateKey[i] = 0
	}
	return nil
}

// GetPublicKeyBase58 returns the public key in base58 encoding (for Solana)
func (s *Ed25519Signer) GetPublicKeyBase58() string {
	return base58.Encode(s.publicKey)
}

// GetPublicKeyHex returns the public key in hex encoding (for NEAR)
func (s *Ed25519Signer) GetPublicKeyHex() string {
	return hex.EncodeToString(s.publicKey)
}

// SignMessage signs a message and returns the signature in base58 (for Solana)
func (s *Ed25519Signer) SignMessageBase58(message []byte) (string, error) {
	signature := ed25519.Sign(s.privateKey, message)
	return base58.Encode(signature), nil
}

// SignMessageHex signs a message and returns the signature in hex (for NEAR)
func (s *Ed25519Signer) SignMessageHex(message []byte) (string, error) {
	signature := ed25519.Sign(s.privateKey, message)
	return hex.EncodeToString(signature), nil
}

// GenerateKeyPair generates a new Ed25519 key pair
func GenerateKeyPair() (*Ed25519Signer, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &Ed25519Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// ExportKeystore exports the keys to a keystore file
func (s *Ed25519Signer) ExportKeystore(filepath string) error {
	keystore := KeystoreFile{
		PrivateKey: hex.EncodeToString(s.privateKey),
		PublicKey:  hex.EncodeToString(s.publicKey),
	}

	data, err := json.MarshalIndent(keystore, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keystore: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write keystore: %w", err)
	}

	return nil
}
