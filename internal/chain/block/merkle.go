package block

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/elecbug/lab-chain/internal/chain/tx"
)

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Hash  []byte
}

// MerkleTree represents a Merkle tree structure
type MerkleTree struct {
	Root *MerkleNode
}

// Equal compares two Merkle trees for equality
func (m *MerkleTree) Equal(target *MerkleTree) bool {
	node := m.Root
	targetNode := target.Root

	var compareNodes func(a, b *MerkleNode) bool

	compareNodes = func(a, b *MerkleNode) bool {
		if a == nil && b == nil {
			return true
		}
		if a == nil || b == nil {
			return false
		}
		if string(a.Hash) != string(b.Hash) {
			return false
		}

		return compareNodes(a.Left, b.Left) && compareNodes(a.Right, b.Right)
	}

	return compareNodes(node, targetNode)
}

// ComputeMerkleRoot computes the Merkle root of a list of transactions
func ComputeMerkleRoot(header []byte, txs []*tx.Transaction) *MerkleTree {
	var data = [][]byte{header}

	for _, tx := range txs {
		b, _ := json.Marshal(tx)
		data = append(data, b)
	}

	tree := buildMerkleTree(data)
	return tree
}

// hashPair computes the hash of two byte slices concatenated together
func hashPair(left, right []byte) []byte {
	h := sha256.New()
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

// buildMerkleTree constructs a Merkle tree from the provided data slices
func buildMerkleTree(data [][]byte) *MerkleTree {
	var nodes []*MerkleNode

	// leaf nodes
	for _, datum := range data {
		hash := sha256.Sum256(datum)
		nodes = append(nodes, &MerkleNode{Hash: hash[:]})
	}

	// build tree
	for len(nodes) > 1 {
		var level []*MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			var left = nodes[i]
			var right *MerkleNode

			if i+1 < len(nodes) {
				right = nodes[i+1]
			} else {
				right = &MerkleNode{Hash: nodes[i].Hash}
			}

			parentHash := hashPair(left.Hash, right.Hash)
			level = append(level, &MerkleNode{Left: left, Right: right, Hash: parentHash})
		}

		nodes = level
	}

	return &MerkleTree{Root: nodes[0]}
}
