package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/elecbug/lab-chain/internal/logger"
)

// Blockchain represents the entire blockchain
type Blockchain struct {
	Blocks            []*Block          // Canonical chain
	Difficulty        *big.Int          // Current PoW difficulty
	Mu                sync.Mutex        // Mutex to protect concurrent access
	pendingBlocks     map[uint64]*Block // Cache for missing blocks
	pendingForkBlocks map[uint64]*Block // Cache for pending fork blocks

	// Optional: forks, orphan blocks, etc.
	Forks map[uint64][]*Block // Index-based fork map
}

// InitBlockchain creates a new blockchain with a genesis block
func InitBlockchain(miner string) *Blockchain {
	genesis := createGenesisBlock(miner)

	bc := &Blockchain{
		Blocks:            []*Block{genesis},
		Difficulty:        big.NewInt(1).Lsh(big.NewInt(1), 240),
		Forks:             make(map[uint64][]*Block),
		pendingBlocks:     make(map[uint64]*Block),
		pendingForkBlocks: make(map[uint64]*Block),
	}

	return bc
}

// MineBlock mines a new block with the given parameters
func (bc *Blockchain) MineBlock(prevHash []byte, index uint64, txs []*Transaction, miner string) *Block {
	var nonce uint64
	var hash []byte
	timestamp := time.Now().Unix()
	bc.adjustDifficulty(20, 10)
	target := bc.Difficulty

	totalFee := big.NewInt(0)

	for _, tx := range txs {
		if tx.From != "COINBASE" {
			totalFee.Add(totalFee, tx.Price)
		}
	}

	reward := big.NewInt(100)
	reward.Add(reward, totalFee) // Add transaction fees to the reward

	coinbaseTx := &Transaction{
		From:      "COINBASE",
		To:        miner,
		Amount:    reward,
		Nonce:     0,
		Price:     big.NewInt(0),
		Signature: nil,
	}

	txs = append([]*Transaction{coinbaseTx}, txs...)

	for {
		header := fmt.Sprintf("%d%x%d%s%d", index, prevHash, timestamp, miner, nonce)
		headerHash := sha256.Sum256([]byte(header))
		fullData := append(headerHash[:], serializeTxs(txs)...)

		digest := sha256.Sum256(fullData)
		hash = digest[:]

		if new(big.Int).SetBytes(hash).Cmp(target) < 0 {
			break
		}
		nonce++
	}

	return &Block{
		Index:        index,
		PreviousHash: prevHash,
		Timestamp:    timestamp,
		Transactions: txs,
		Miner:        miner,
		Nonce:        nonce,
		Hash:         hash,
	}
}

// adjustDifficulty adjusts the mining difficulty based on the time taken to mine the last few blocks
func (bc *Blockchain) adjustDifficulty(targetIntervalSec int64, windowSize int) {
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

// VerifyBlock checks if a block is valid against the previous block and the current difficulty
func (bc *Blockchain) VerifyBlock(block *Block, previous *Block) bool {
	log := logger.LabChainLogger

	if previous == nil {
		if block.Index != 0 {
			log.Warn("genesis block with wrong index")
			return false
		}
		return true
	}

	if block.Index != previous.Index+1 {
		log.Infof("block index mismatch: got %d, expected %d", block.Index, previous.Index+1)
		return false
	}
	if !bytes.Equal(block.PreviousHash, previous.Hash) {
		log.Infof("previous hash mismatch")
		return false
	}

	hashInt := new(big.Int).SetBytes(block.Hash)
	if hashInt.Cmp(bc.Difficulty) >= 0 {
		log.Infof("block does not meet difficulty: hash=%x, difficulty=%x", block.Hash, bc.Difficulty)
		return false
	}

	expectedNonces := make(map[string]uint64)

	for i, tx := range block.Transactions {
		ok, err := tx.VerifySignature()
		if err != nil || !ok {
			log.Infof("tx[%d] signature invalid: %v", i, err)
			return false
		}
	}

	for i, tx := range block.Transactions {
		if tx.From == "COINBASE" {
			continue
		}

		// Balance check
		required := new(big.Int).Add(tx.Amount, tx.Price)
		balance := bc.GetBalance(tx.From)
		if balance.Cmp(required) < 0 {
			log.Infof("tx[%d] insufficient balance: from=%s, need=%s, have=%s", i, tx.From, required.String(), balance.String())
			return false
		}

		// Nonce check
		expected, ok := expectedNonces[tx.From]
		if !ok {
			expected = bc.GetNonce(tx.From)
		}

		if tx.Nonce != expected {
			log.Infof("tx[%d] invalid nonce: from=%s, got=%d, expected=%d", i, tx.From, tx.Nonce, expected)
			return false
		}

		expectedNonces[tx.From] = expected + 1
	}

	return true
}

// GetBalance calculates the balance of a given address by iterating through all blocks,
// while ignoring duplicate transactions (same hash)
func (bc *Blockchain) GetBalance(address string) *big.Int {
	balance := new(big.Int)
	seen := make(map[string]bool) // track seen transaction hashes

	for _, blk := range bc.Blocks {
		for _, tx := range blk.Transactions {
			txHash := string(tx.hash())

			if seen[txHash] {
				continue // skip duplicate transaction
			}

			seen[txHash] = true

			if tx.From == address {
				balance.Sub(balance, tx.Amount)
			}
			if tx.To == address {
				balance.Add(balance, tx.Amount)
			}
		}
	}

	return balance
}

// addBlock appends a verified block to the chain
func (bc *Blockchain) addBlock(block *Block) error {
	bc.Blocks = append(bc.Blocks, block)
	return nil
}

// Save writes the blockchain to a file as JSON
func (bc *Blockchain) Save(path string) error {
	bc.Mu.Lock()
	defer bc.Mu.Unlock()

	data, err := json.MarshalIndent(bc, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal blockchain: %v", err)
	}

	return os.WriteFile(path, data, 0644)
}

// Load reads blockchain data from a file and replaces the in-memory state
func Load(path string) (*Blockchain, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("failed to read blockchain file: %v", err)
	}

	temp := &Blockchain{}

	if err := json.Unmarshal(data, temp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blockchain: %v", err)
	}

	bc := &Blockchain{
		Blocks:            temp.Blocks,
		Difficulty:        temp.Difficulty,
		Forks:             temp.Forks,
		pendingBlocks:     make(map[uint64]*Block),
		pendingForkBlocks: make(map[uint64]*Block),
	}

	return bc, nil
}

// GetNonce calculates the nonce for a given address by counting the number of transactions sent from that address
func (bc *Blockchain) GetNonce(address string) uint64 {
	var nonce uint64
	for _, blk := range bc.Blocks {
		for _, tx := range blk.Transactions {
			if tx.From == address {
				nonce++
			}
		}
	}
	return nonce
}

// GetBlockByIndex returns the block at the specified index, or nil if not found
func (bc *Blockchain) GetBlockByIndex(i uint64) *Block {
	bc.Mu.Lock()
	defer bc.Mu.Unlock()

	if i < uint64(len(bc.Blocks)) {
		return bc.Blocks[i]
	}
	return nil
}

// GetBlockByHash searches the chain for a block with the given hash.
// Returns the block if found, or nil otherwise.
func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	bc.Mu.Lock()
	defer bc.Mu.Unlock()

	for _, blk := range bc.Blocks {
		if string(blk.Hash) == string(hash) {
			return blk
		}
	}
	return nil
}
