package batching

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
)

// MerkleTree represents a Merkle tree for batch verification
type MerkleTree struct {
	Root   *MerkleNode
	Leaves []*MerkleNode
	Layers [][]*MerkleNode
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Hash   string
	Left   *MerkleNode
	Right  *MerkleNode
	Parent *MerkleNode
	Index  int
}

// MerkleProof represents a proof for a specific message in the batch
type MerkleProof struct {
	MessageID string
	Index     int
	Siblings  []string
	Root      string
}

// BuildMerkleTree constructs a Merkle tree from messages
func BuildMerkleTree(messages []*types.CrossChainMessage) (*MerkleTree, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("cannot build merkle tree from empty message list")
	}

	tree := &MerkleTree{
		Leaves: make([]*MerkleNode, len(messages)),
		Layers: make([][]*MerkleNode, 0),
	}

	// Create leaf nodes
	leafLayer := make([]*MerkleNode, len(messages))
	for i, msg := range messages {
		hash := hashMessage(msg)
		node := &MerkleNode{
			Hash:  hash,
			Index: i,
		}
		leafLayer[i] = node
		tree.Leaves[i] = node
	}
	tree.Layers = append(tree.Layers, leafLayer)

	// Build tree bottom-up
	currentLayer := leafLayer
	for len(currentLayer) > 1 {
		nextLayer := make([]*MerkleNode, 0)

		for i := 0; i < len(currentLayer); i += 2 {
			left := currentLayer[i]
			var right *MerkleNode

			// If odd number of nodes, duplicate the last one
			if i+1 < len(currentLayer) {
				right = currentLayer[i+1]
			} else {
				right = left
			}

			// Create parent node
			parentHash := hashPair(left.Hash, right.Hash)
			parent := &MerkleNode{
				Hash:  parentHash,
				Left:  left,
				Right: right,
				Index: i / 2,
			}

			left.Parent = parent
			right.Parent = parent

			nextLayer = append(nextLayer, parent)
		}

		tree.Layers = append(tree.Layers, nextLayer)
		currentLayer = nextLayer
	}

	// Root is the last remaining node
	tree.Root = currentLayer[0]

	return tree, nil
}

// GetProof generates a Merkle proof for a message at given index
func (t *MerkleTree) GetProof(index int) (*MerkleProof, error) {
	if index < 0 || index >= len(t.Leaves) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	proof := &MerkleProof{
		Index:    index,
		Root:     t.Root.Hash,
		Siblings: make([]string, 0),
	}

	currentNode := t.Leaves[index]
	currentIndex := index

	// Traverse up the tree
	for currentNode.Parent != nil {
		parent := currentNode.Parent

		// Determine sibling
		var sibling *MerkleNode
		if parent.Left == currentNode {
			sibling = parent.Right
		} else {
			sibling = parent.Left
		}

		proof.Siblings = append(proof.Siblings, sibling.Hash)

		currentNode = parent
		currentIndex = currentIndex / 2
	}

	return proof, nil
}

// VerifyProof verifies a Merkle proof
func VerifyProof(proof *MerkleProof, messageHash string) bool {
	currentHash := messageHash
	index := proof.Index

	for _, siblingHash := range proof.Siblings {
		// Determine order based on index
		if index%2 == 0 {
			// Current is left
			currentHash = hashPair(currentHash, siblingHash)
		} else {
			// Current is right
			currentHash = hashPair(siblingHash, currentHash)
		}
		index = index / 2
	}

	return currentHash == proof.Root
}

// GetAllProofs generates proofs for all messages in the batch
func (t *MerkleTree) GetAllProofs() ([]*MerkleProof, error) {
	proofs := make([]*MerkleProof, len(t.Leaves))

	for i := range t.Leaves {
		proof, err := t.GetProof(i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate proof for index %d: %w", i, err)
		}
		proofs[i] = proof
	}

	return proofs, nil
}

// GetRoot returns the Merkle root hash
func (t *MerkleTree) GetRoot() string {
	if t.Root == nil {
		return ""
	}
	return t.Root.Hash
}

// Helper functions

func hashMessage(msg *types.CrossChainMessage) string {
	hasher := sha256.New()

	// Hash all message fields
	hasher.Write([]byte(msg.ID))
	hasher.Write([]byte(msg.SourceChain.Name))
	hasher.Write([]byte(msg.DestinationChain.Name))
	hasher.Write([]byte(msg.Sender.Raw))
	hasher.Write([]byte(msg.Recipient.Raw))
	hasher.Write(msg.Payload)

	return hex.EncodeToString(hasher.Sum(nil))
}

func hashPair(left, right string) string {
	hasher := sha256.New()
	hasher.Write([]byte(left))
	hasher.Write([]byte(right))
	return hex.EncodeToString(hasher.Sum(nil))
}

// BatchMerkleData holds Merkle tree data for a batch
type BatchMerkleData struct {
	Root      string
	Proofs    map[string]*MerkleProof // messageID -> proof
	Tree      *MerkleTree
	BatchSize int
}

// GenerateBatchMerkleData creates complete Merkle data for a batch
func GenerateBatchMerkleData(batch *Batch) (*BatchMerkleData, error) {
	// Build Merkle tree
	tree, err := BuildMerkleTree(batch.Messages)
	if err != nil {
		return nil, fmt.Errorf("failed to build merkle tree: %w", err)
	}

	// Generate all proofs
	proofs, err := tree.GetAllProofs()
	if err != nil {
		return nil, fmt.Errorf("failed to generate proofs: %w", err)
	}

	// Map proofs by message ID
	proofMap := make(map[string]*MerkleProof)
	for i, proof := range proofs {
		if i < len(batch.Messages) {
			proof.MessageID = batch.Messages[i].ID
			proofMap[batch.Messages[i].ID] = proof
		}
	}

	data := &BatchMerkleData{
		Root:      tree.GetRoot(),
		Proofs:    proofMap,
		Tree:      tree,
		BatchSize: len(batch.Messages),
	}

	// Update batch with Merkle root
	batch.MerkleRoot = data.Root

	return data, nil
}
