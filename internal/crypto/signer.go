package crypto

import (
	"context"
	"fmt"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
)

// UniversalSigner provides a unified signing interface for all blockchain types
type UniversalSigner interface {
	// GetScheme returns the signature scheme (ECDSA or Ed25519)
	GetScheme() types.SignatureScheme

	// GetPublicKey returns the public key bytes
	GetPublicKey() ([]byte, error)

	// GetAddress returns the address for the given chain type
	GetAddress(chainType types.ChainType) (string, error)

	// Sign signs arbitrary data
	Sign(ctx context.Context, data []byte) ([]byte, error)

	// SignTransaction signs a chain-specific transaction
	SignTransaction(ctx context.Context, tx interface{}, chainID string) (interface{}, error)

	// Verify verifies a signature
	Verify(data []byte, signature []byte, publicKey []byte) (bool, error)

	// Close closes the signer and clears sensitive data
	Close() error
}

// SignerFactory creates appropriate signers based on chain type
type SignerFactory struct {
	keystorePath string
}

// NewSignerFactory creates a new signer factory
func NewSignerFactory(keystorePath string) *SignerFactory {
	return &SignerFactory{
		keystorePath: keystorePath,
	}
}

// CreateSigner creates a signer for the given chain type
func (f *SignerFactory) CreateSigner(
	chainType types.ChainType,
	password string,
) (UniversalSigner, error) {
	switch chainType {
	case types.ChainTypeEVM:
		return NewECDSASigner(f.keystorePath, password)
	case types.ChainTypeSolana, types.ChainTypeNEAR:
		return NewEd25519Signer(f.keystorePath, password)
	default:
		return nil, fmt.Errorf("unsupported chain type for signer: %s", chainType)
	}
}

// CreateMultiChainSigners creates signers for all supported chain types
func (f *SignerFactory) CreateMultiChainSigners(
	evmPassword, solanaPassword, nearPassword string,
) (map[types.ChainType]UniversalSigner, error) {
	signers := make(map[types.ChainType]UniversalSigner)

	// Create EVM signer
	evmSigner, err := f.CreateSigner(types.ChainTypeEVM, evmPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create EVM signer: %w", err)
	}
	signers[types.ChainTypeEVM] = evmSigner

	// Create Solana signer
	solanaSigner, err := f.CreateSigner(types.ChainTypeSolana, solanaPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana signer: %w", err)
	}
	signers[types.ChainTypeSolana] = solanaSigner

	// Create NEAR signer
	nearSigner, err := f.CreateSigner(types.ChainTypeNEAR, nearPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create NEAR signer: %w", err)
	}
	signers[types.ChainTypeNEAR] = nearSigner

	return signers, nil
}

// CloseAll closes all signers
func CloseAll(signers map[types.ChainType]UniversalSigner) {
	for chainType, signer := range signers {
		if err := signer.Close(); err != nil {
			// Log error but continue
			fmt.Printf("Error closing signer for %s: %v\n", chainType, err)
		}
	}
}
