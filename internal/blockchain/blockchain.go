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

// Blockchain represents the entire blockchain.
type Blockchain struct {
	Blocks       []*Block   // Canonical chain
	Difficulty   *big.Int   // Current PoW difficulty
	longestIndex uint64     // Highest known block index
	Mu           sync.Mutex // Mutex to protect concurrent access

	// Optional: forks, orphan blocks, etc.
	Forks map[uint64][]*Block // Index-based fork map
}

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

// CreateGenesisBlock creates the first block in the blockchain with a coinbase transaction
func CreateGenesisBlock(to string) *Block {
	txs := []*Transaction{
		{
			From:      "COINBASE",
			To:        to,
			Amount:    big.NewInt(1000), // Initial reward
			Nonce:     0,
			Price:     big.NewInt(0),
			Signature: nil,
		},
	}

	header := fmt.Sprintf("0%x%d%s%d", []byte{}, time.Now().Unix(), to, 0)
	headerHash := sha256.Sum256([]byte(header))
	fullData := append(headerHash[:], serializeTxs(txs)...)
	hash := sha256.Sum256(fullData)

	return &Block{
		Index:        0,
		PreviousHash: []byte{},
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		Miner:        to,
		Nonce:        0,
		Hash:         hash[:],
	}
}

// InitBlockchain creates a new blockchain with a genesis block
func InitBlockchain(miner string) *Blockchain {
	genesis := CreateGenesisBlock(miner)

	bc := &Blockchain{
		Blocks:       []*Block{genesis},
		Difficulty:   big.NewInt(1).Lsh(big.NewInt(1), 240), // 초기 난이도 설정 (예: 2^240)
		longestIndex: 0,
		Forks:        make(map[uint64][]*Block),
	}

	return bc
}

// serializeTxs serializes the transactions into a byte slice.
func serializeTxs(txs []*Transaction) []byte {
	var data []byte

	for _, tx := range txs {
		b, _ := json.Marshal(tx)
		data = append(data, b...)
	}

	return data
}

// adjustDifficulty adjusts the mining difficulty based on the time taken to mine the last few blocks.
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

	for _, tx := range block.Transactions {
		ok, err := tx.VerifySignature()
		if err != nil || !ok {
			return false
		}

		required := new(big.Int).Add(tx.Amount, tx.Price)
		balance := bc.GetBalance(tx.From)
		if balance.Cmp(required) < 0 {
			return false
		}
	}

	return true
}

// HandleIncomingBlock verifies and integrates the block, resolving forks if necessary
func (bc *Blockchain) HandleIncomingBlock(block *Block) error {
	bc.Mu.Lock()
	defer bc.Mu.Unlock()

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

// GetBalance calculates the balance of a given address by iterating through all blocks
func (bc *Blockchain) GetBalance(address string) *big.Int {
	balance := new(big.Int)
	for _, blk := range bc.Blocks {
		for _, tx := range blk.Transactions {
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
func (bc *Blockchain) Load(path string) error {
	bc.Mu.Lock()
	defer bc.Mu.Unlock()

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
