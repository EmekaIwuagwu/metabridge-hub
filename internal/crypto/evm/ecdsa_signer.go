package evm

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// ECDSASigner implements UniversalSigner for EVM chains using ECDSA (secp256k1)
type ECDSASigner struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	address    common.Address
}

// NewECDSASigner creates a new ECDSA signer from keystore
func NewECDSASigner(keystorePath string, password string) (*ECDSASigner, error) {
	// Load keystore
	ks := keystore.NewKeyStore(keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)

	if len(ks.Accounts()) == 0 {
		return nil, fmt.Errorf("no accounts found in keystore: %s", keystorePath)
	}

	// Get first account
	account := ks.Accounts()[0]

	// Unlock account
	if err := ks.Unlock(account, password); err != nil {
		return nil, fmt.Errorf("failed to unlock account: %w", err)
	}

	// Get private key
	key, err := ks.Export(account, password, password)
	if err != nil {
		return nil, fmt.Errorf("failed to export key: %w", err)
	}

	privateKey, err := keystore.DecryptKey(key, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	publicKey := privateKey.PrivateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &ECDSASigner{
		privateKey: privateKey.PrivateKey,
		publicKey:  publicKeyECDSA,
		address:    address,
	}, nil
}

// NewECDSASignerFromPrivateKey creates a signer from a private key hex string
func NewECDSASignerFromPrivateKey(privateKeyHex string) (*ECDSASigner, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &ECDSASigner{
		privateKey: privateKey,
		publicKey:  publicKeyECDSA,
		address:    address,
	}, nil
}

// GetScheme returns the signature scheme
func (s *ECDSASigner) GetScheme() types.SignatureScheme {
	return types.SignatureSchemeECDSA
}

// GetPublicKey returns the public key bytes
func (s *ECDSASigner) GetPublicKey() ([]byte, error) {
	return crypto.FromECDSAPub(s.publicKey), nil
}

// GetAddress returns the Ethereum address
func (s *ECDSASigner) GetAddress(chainType types.ChainType) (string, error) {
	if chainType != types.ChainTypeEVM {
		return "", fmt.Errorf("ECDSA signer only supports EVM chains")
	}
	return s.address.Hex(), nil
}

// Sign signs arbitrary data using Ethereum's personal_sign format
func (s *ECDSASigner) Sign(ctx context.Context, data []byte) ([]byte, error) {
	// Hash the data
	hash := crypto.Keccak256Hash(data)

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	return signature, nil
}

// SignHash signs a hash directly
func (s *ECDSASigner) SignHash(hash []byte) ([]byte, error) {
	signature, err := crypto.Sign(hash, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}
	return signature, nil
}

// SignTransaction signs an Ethereum transaction
func (s *ECDSASigner) SignTransaction(ctx context.Context, tx interface{}, chainID string) (interface{}, error) {
	ethTx, ok := tx.(*ethtypes.Transaction)
	if !ok {
		return nil, fmt.Errorf("invalid transaction type: expected *types.Transaction")
	}

	// Parse chain ID
	chainIDBigInt, ok := new(big.Int).SetString(chainID, 10)
	if !ok {
		return nil, fmt.Errorf("invalid chain ID: %s", chainID)
	}

	// Create signer for this chain ID
	signer := ethtypes.NewEIP155Signer(chainIDBigInt)

	// Sign the transaction
	signedTx, err := ethtypes.SignTx(ethTx, signer, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return signedTx, nil
}

// Verify verifies an ECDSA signature
func (s *ECDSASigner) Verify(data []byte, signature []byte, publicKey []byte) (bool, error) {
	// Hash the data
	hash := crypto.Keccak256Hash(data)

	// Recover public key from signature
	recoveredPubKey, err := crypto.SigToPub(hash.Bytes(), signature)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Compare addresses
	recoveredAddress := crypto.PubkeyToAddress(*recoveredPubKey)
	expectedPubKey, err := crypto.UnmarshalPubkey(publicKey)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	expectedAddress := crypto.PubkeyToAddress(*expectedPubKey)

	return recoveredAddress == expectedAddress, nil
}

// RecoverAddress recovers the Ethereum address from a signature
func (s *ECDSASigner) RecoverAddress(hash []byte, signature []byte) (common.Address, error) {
	pubKey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to recover public key: %w", err)
	}
	return crypto.PubkeyToAddress(*pubKey), nil
}

// Close clears sensitive data
func (s *ECDSASigner) Close() error {
	// Zero out private key
	if s.privateKey != nil {
		s.privateKey.D.SetInt64(0)
	}
	return nil
}

// GetEthereumAddress returns the Ethereum address
func (s *ECDSASigner) GetEthereumAddress() common.Address {
	return s.address
}

// SignEthereumMessage signs a message with Ethereum's personal_sign format
func (s *ECDSASigner) SignEthereumMessage(message []byte) ([]byte, error) {
	// Add Ethereum signed message prefix
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixedMessage))

	// Sign
	signature, err := crypto.Sign(hash.Bytes(), s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	// Adjust V value for Ethereum compatibility
	signature[64] += 27

	return signature, nil
}
