package crypto

import (
	"github.com/EmekaIwuagwu/articium-hub/internal/crypto/ed25519"
	"github.com/EmekaIwuagwu/articium-hub/internal/crypto/evm"
	"github.com/EmekaIwuagwu/articium-hub/internal/types"
)

// NewECDSASigner creates a new ECDSA signer (for EVM chains)
func NewECDSASigner(keystorePath string, password string) (UniversalSigner, error) {
	return evm.NewECDSASigner(keystorePath, password)
}

// NewEd25519Signer creates a new Ed25519 signer (for Solana/NEAR)
func NewEd25519Signer(keystorePath string, password string) (UniversalSigner, error) {
	return ed25519.NewEd25519Signer(keystorePath, password)
}

// GetSignerForChain returns the appropriate signer type for a chain
func GetSignerForChain(chainType types.ChainType) (types.SignatureScheme, error) {
	return types.GetSchemeForChain(chainType)
}
