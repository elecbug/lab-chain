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
	Blocks            []*Block
	Mu                sync.Mutex
	pendingBlocks     map[uint64]*Block
	pendingForkBlocks map[uint64]*Block
	Forks             map[uint64][]*Block
}

// InitBlockchain creates a new blockchain with a genesis block
func InitBlockchain(miner string) *Blockchain {
	genesis := createGenesisBlock(miner)

	bc := &Blockchain{
		Blocks:            []*Block{genesis},
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
	difficulty := bc.calcDifficulty(20, 10)
	totalFee := big.NewInt(0)
	for _, tx := range txs {
		if tx.From != "COINBASE" {
			totalFee.Add(totalFee, tx.Price)
		}
	}

	reward := big.NewInt(100)
	reward.Add(reward, totalFee)

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

		if new(big.Int).SetBytes(hash).Cmp(difficulty) < 0 {
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
		Difficulty:   difficulty,
	}
}

// calcDifficulty calculates the new difficulty based on recent blocks
func (bc *Blockchain) calcDifficulty(targetIntervalSec int64, windowSize int) *big.Int {
	n := len(bc.Blocks)
	if n <= windowSize {
		return big.NewInt(1).Lsh(big.NewInt(1), 240)
	}

	latest := bc.Blocks[n-1]
	past := bc.Blocks[n-1-windowSize]

	actualTime := latest.Timestamp - past.Timestamp
	expectedTime := targetIntervalSec * int64(windowSize)

	ratioNum := big.NewInt(actualTime)
	ratioDen := big.NewInt(expectedTime)
	newDifficulty := new(big.Int).Mul(latest.Difficulty, ratioNum)
	newDifficulty.Div(newDifficulty, ratioDen)

	if newDifficulty.Cmp(big.NewInt(1)) < 0 {
		newDifficulty = big.NewInt(1)
	}

	return newDifficulty
}

// VerifyBlock checks if a block is valid against the previous block
func (bc *Blockchain) VerifyBlock(block *Block, previous *Block) bool {
	log := logger.LabChainLogger

	// log.Infof("Verifying block: index=%d", block.Index)
	// log.Infof("Expected PreviousHash: %x", previous.Hash)
	// log.Infof("Actual PreviousHash in block: %x", block.PreviousHash)

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
	if hashInt.Cmp(block.Difficulty) >= 0 {
		log.Infof("block does not meet difficulty: hash=%x, difficulty=%x", block.Hash, block.Difficulty)
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

		required := new(big.Int).Add(tx.Amount, tx.Price)
		balance := bc.GetBalance(tx.From)
		if balance.Cmp(required) < 0 {
			log.Infof("tx[%d] insufficient balance: from=%s, need=%s, have=%s", i, tx.From, required.String(), balance.String())
			return false
		}

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

// GetBalance calculates the balance of a given address
func (bc *Blockchain) GetBalance(address string) *big.Int {
	balance := new(big.Int)
	seen := make(map[string]bool)

	for _, blk := range bc.Blocks {
		for _, tx := range blk.Transactions {
			txHash := string(tx.hash())
			if seen[txHash] {
				continue
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

// Load reads blockchain data from a file
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
		Forks:             temp.Forks,
		pendingBlocks:     make(map[uint64]*Block),
		pendingForkBlocks: make(map[uint64]*Block),
	}

	return bc, nil
}

// GetNonce calculates the nonce for a given address
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

// GetBlockByIndex returns the block at the specified index
func (bc *Blockchain) GetBlockByIndex(i uint64) *Block {
	bc.Mu.Lock()
	defer bc.Mu.Unlock()
	if i < uint64(len(bc.Blocks)) {
		return bc.Blocks[i]
	}
	return nil
}

// GetBlockByHash searches the chain for a block with the given hash
func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	for _, blk := range bc.Blocks {
		if bytes.Equal(blk.Hash, hash) {
			return blk
		}
	}
	for _, blk := range bc.pendingForkBlocks {
		if bytes.Equal(blk.Hash, hash) {
			return blk
		}
	}
	return nil
}
