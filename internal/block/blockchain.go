package block

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"

	"github.com/elecbug/lab-chain/internal/logger"
)

// Blockchain represents the entire blockchain.
type Blockchain struct {
	Blocks       []*Block   // Canonical chain
	Difficulty   *big.Int   // Current PoW difficulty
	longestIndex uint64     // Highest known block index
	mu           sync.Mutex // Mutex to protect concurrent access

	// Optional: forks, orphan blocks, etc.
	Forks map[uint64][]*Block // Index-based fork map
}

// AdjustDifficulty adjusts the mining difficulty based on the time taken to mine the last few blocks.
func (bc *Blockchain) AdjustDifficulty(targetIntervalSec int64, windowSize int) {
	n := len(bc.Blocks)
	if n <= windowSize {
		return
	}

	latest := bc.Blocks[n-1]
	past := bc.Blocks[n-1-windowSize]

	actualTime := latest.Timestamp - past.Timestamp
	expectedTime := targetIntervalSec * int64(windowSize)

	oldDifficulty := new(big.Int).Set(bc.Difficulty)

	// adjustmentRatio = actual / expected
	ratioNum := big.NewInt(actualTime)
	ratioDen := big.NewInt(expectedTime)

	newDifficulty := new(big.Int).Mul(oldDifficulty, ratioNum)
	newDifficulty.Div(newDifficulty, ratioDen)

	// Ensure new difficulty is at least 1
	if newDifficulty.Cmp(big.NewInt(1)) < 0 {
		newDifficulty = big.NewInt(1)
	}

	bc.Difficulty = newDifficulty
}

// VerifyBlock checks if a block is valid compared to the current chain state
func (bc *Blockchain) VerifyBlock(block *Block, previous *Block) bool {
	if previous == nil {
		return block.Index == 0
	}

	if block.Index != previous.Index+1 {
		return false
	}
	if !bytes.Equal(block.PreviousHash, previous.Hash) {
		return false
	}

	hashInt := new(big.Int).SetBytes(block.Hash)
	if hashInt.Cmp(bc.Difficulty) >= 0 {
		return false
	}

	for _, tx := range block.Transactions {
		ok, err := tx.VerifySignature()
		if err != nil || !ok {
			return false
		}
	}

	return true
}

// HandleIncomingBlock verifies and integrates the block, resolving forks if necessary
func (bc *Blockchain) HandleIncomingBlock(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	n := len(bc.Blocks)
	if n == 0 {
		if bc.VerifyBlock(block, nil) {
			return bc.addBlock(block)
		}

		return fmt.Errorf("genesis block invalid")
	}

	last := bc.Blocks[n-1]

	if block.Index == last.Index+1 && bc.VerifyBlock(block, last) {
		return bc.addBlock(block)
	}

	// Fork handling
	if block.Index <= last.Index {
		log := logger.AppLogger
		log.Infof("received fork block: index %d (current: %d)", block.Index, last.Index)

		// Check if this fork is longer
		// (In practice, we need to track branches, here simplified)
		if block.Index > bc.longestIndex {
			log.Infof("switching to longer chain via fork block index %d", block.Index)
			bc.Blocks = bc.Blocks[:block.Index] // truncate chain (simplified)
			return bc.addBlock(block)
		}

		return fmt.Errorf("fork block ignored, not longer")
	}

	return fmt.Errorf("block rejected: invalid order or hash")
}

// addBlock appends a verified block to the chain
func (bc *Blockchain) addBlock(block *Block) error {
	bc.Blocks = append(bc.Blocks, block)
	return nil
}

// Save writes the blockchain to a file as JSON
func (bc *Blockchain) Save(path string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	data, err := json.MarshalIndent(bc, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal blockchain: %v", err)
	}

	return os.WriteFile(path, data, 0644)
}

// Load reads blockchain data from a file and replaces the in-memory state
func (bc *Blockchain) Load(path string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	data, err := os.ReadFile(path)

	if err != nil {
		return fmt.Errorf("failed to read blockchain file: %v", err)
	}

	temp := &Blockchain{}

	if err := json.Unmarshal(data, temp); err != nil {
		return fmt.Errorf("failed to unmarshal blockchain: %v", err)
	}

	bc.Blocks = temp.Blocks
	bc.Difficulty = temp.Difficulty
	bc.longestIndex = temp.longestIndex
	bc.Forks = temp.Forks

	return nil
}
