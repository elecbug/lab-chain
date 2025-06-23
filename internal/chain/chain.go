package chain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/elecbug/lab-chain/internal/chain/block"
	"github.com/elecbug/lab-chain/internal/chain/tx"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/ethereum/go-ethereum/crypto"
)

// Chain represents the entire blockchain
type Chain struct {
	Blocks            []*block.Block
	Mu                sync.Mutex
	pendingBlocks     map[uint64]*block.Block
	pendingForkBlocks map[uint64]*block.Block
}

// VerifyChain checks the integrity of the blockchain starting from the genesis block
func (c *Chain) VerifyChain(genesis *block.Block) error {
	log := logger.LabChainLogger

	if c.Blocks[0].Equal(genesis) {
		log.Infof("genesis block verified successfully")
	} else {
		log.Warnf("genesis block mismatch")
		return fmt.Errorf("genesis block mismatch")
	}

	tempChain := &Chain{
		Blocks: []*block.Block{genesis},
		Mu:     sync.Mutex{},
	}

	for i := 1; i < len(c.Blocks); i++ {
		current := c.Blocks[i]
		previous := c.Blocks[i-1]

		if current.Index != previous.Index+1 && bytes.Equal(current.PreviousHash, previous.Hash) {
			if tempChain.VerifyNewBlock(current, previous) {
				tempChain.AddBlock(current)
			} else {
				log.Warnf("block %d verification failed", current.Index)
				return fmt.Errorf("block %d verification failed", current.Index)
			}
		}
	}

	log.Infof("all blocks verified successfully")

	return nil
}

// CreateTx creates a new transaction with the given parameters and signs it
func (c *Chain) CreateTx(fromPriv *ecdsa.PrivateKey, to string, amount, price *big.Int, base int) (*tx.Transaction, error) {
	log := logger.LabChainLogger

	pubKey := fromPriv.Public().(*ecdsa.PublicKey)
	fromAddr := crypto.PubkeyToAddress(*pubKey)

	t := &tx.Transaction{
		From:   fromAddr.Hex(),
		To:     to,
		Amount: amount,
		Nonce:  c.GetNonce(fromAddr.Hex(), base),
		Price:  price,
	}

	err := t.Sign(fromPriv)

	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	} else {
		log.Infof("transaction signed successfully: %s -> %s, amount: %s, price: %s, nonce: %d",
			t.From, t.To, t.Amount.String(), t.Price.String(), t.Nonce)
	}

	return t, nil
}

// InitBlockchain creates a new blockchain with a genesis block
func InitBlockchain(miner string) *Chain {
	genesis := createGenesisBlock(miner)

	c := &Chain{
		Blocks:            []*block.Block{genesis},
		pendingBlocks:     make(map[uint64]*block.Block),
		pendingForkBlocks: make(map[uint64]*block.Block),
	}

	return c
}

// createGenesisBlock creates the first block in the blockchain with a coinbase transaction
func createGenesisBlock(to string) *block.Block {
	txs := []*tx.Transaction{
		{
			From:      tx.COINBASE,
			To:        to,
			Amount:    big.NewInt(1000), // Initial reward
			Nonce:     0,
			Price:     big.NewInt(0),
			Signature: nil,
		},
	}

	header := fmt.Sprintf("0%x%d%s%d", []byte{}, time.Now().Unix(), to, 0)
	headerHash := sha256.Sum256([]byte(header))
	root := block.ComputeMerkleRoot(headerHash[:], txs)

	digest := sha256.Sum256(root.Root.Hash)
	hash := digest[:]

	return &block.Block{
		Index:        0,
		PreviousHash: []byte{},
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		Miner:        to,
		Nonce:        0,
		Hash:         hash,
		MerkleRoot:   root,
	}
}

// MineBlock mines a new block with the given parameters
func (c *Chain) MineBlock(prevHash []byte, index uint64, txs []*tx.Transaction, miner string) *block.Block {
	var nonce uint64
	var hash []byte
	var root *block.MerkleTree

	timestamp := time.Now().Unix()
	difficulty := c.calcDifficulty(30, 10)
	reward := big.NewInt(100)

	coinbaseTx := &tx.Transaction{
		From:      tx.COINBASE,
		To:        miner,
		Amount:    reward,
		Nonce:     index,
		Price:     big.NewInt(0),
		Signature: nil,
	}

	txs = append([]*tx.Transaction{coinbaseTx}, txs...)

	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Nonce < txs[j].Nonce
	})

	for {
		header := fmt.Sprintf("%d%x%d%s%d", index, prevHash, timestamp, miner, nonce)
		headerHash := sha256.Sum256([]byte(header))
		root = block.ComputeMerkleRoot(headerHash[:], txs)

		digest := sha256.Sum256(root.Root.Hash)
		hash = digest[:]

		if new(big.Int).SetBytes(hash).Cmp(difficulty) < 0 {
			break
		}

		nonce++
	}

	return &block.Block{
		Index:        index,
		PreviousHash: prevHash,
		Timestamp:    timestamp,
		Transactions: txs,
		Miner:        miner,
		Nonce:        nonce,
		Hash:         hash,
		Difficulty:   difficulty,
		MerkleRoot:   root,
	}
}

// calcDifficulty calculates the new difficulty based on recent blocks
func (c *Chain) calcDifficulty(targetIntervalSec int64, windowSize int) *big.Int {
	n := len(c.Blocks)
	if n <= windowSize {
		return big.NewInt(1).Lsh(big.NewInt(1), 240)
	}

	latest := c.Blocks[n-1]
	past := c.Blocks[n-1-windowSize]

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

// VerifyNewBlock checks if a block is valid against the previous block
func (c *Chain) VerifyNewBlock(b *block.Block, previous *block.Block) bool {
	log := logger.LabChainLogger

	// log.Infof("Verifying block: index=%d", block.Index)
	// log.Infof("Expected PreviousHash: %x", previous.Hash)
	// log.Infof("Actual PreviousHash in block: %x", block.PreviousHash)

	if previous == nil {
		if b.Index != 0 {
			log.Warn("genesis block with wrong index")
			return false
		}
		return true
	}

	if b.Index != previous.Index+1 {
		log.Infof("block index mismatch: got %d, expected %d", b.Index, previous.Index+1)
		return false
	}

	if !bytes.Equal(b.PreviousHash, previous.Hash) {
		log.Infof("previous hash mismatch")
		return false
	}

	hashInt := new(big.Int).SetBytes(b.Hash)

	if hashInt.Cmp(b.Difficulty) >= 0 {
		log.Infof("block does not meet difficulty: hash=%x, difficulty=%x", b.Hash, b.Difficulty)
		return false
	}

	for i, t := range b.Transactions {
		ok, err := t.VerifySignature()

		if err != nil || !ok {
			log.Infof("tx[%d] signature invalid: %v", i, err)
			return false
		}
	}

	tempMem := make(map[string]int, 0)

	for i, t := range b.Transactions {
		if t.From == tx.COINBASE {
			continue
		}

		required := new(big.Int).Add(t.Amount, t.Price)
		balance := c.GetBalance(t.From)

		if balance.Cmp(required) < 0 {
			log.Infof("tx[%d] insufficient balance: from=%s, need=%s, have=%s", i, t.From, required.String(), balance.String())
			return false
		}

		expected := c.GetNonce(t.From, tempMem[t.From])
		tempMem[t.From]++

		if t.Nonce != expected {
			log.Infof("tx[%d] invalid nonce: from=%s, got=%d, expected=%d", i, t.From, t.Nonce, expected)
			return false
		}
	}

	header := fmt.Sprintf("%d%x%d%s%d", b.Index, b.PreviousHash, b.Timestamp, b.Miner, b.Nonce)
	headerHash := sha256.Sum256([]byte(header))

	root := block.ComputeMerkleRoot(headerHash[:], b.Transactions)

	if b.MerkleRoot == nil || !bytes.Equal(b.MerkleRoot.Root.Hash, root.Root.Hash) {
		log.Infof("merkle root mismatch: expected=%s, actual=%s", b.MerkleRoot.Root.Hash, root.Root.Hash)
		return false
	}

	return true
}

// GetBalance calculates the balance of a given address
func (c *Chain) GetBalance(address string) *big.Int {
	balance := new(big.Int)
	seen := make(map[string]bool)

	for _, blk := range c.Blocks {
		for _, tx := range blk.Transactions {
			txHash := string(tx.Hash())

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

// AddBlock appends a verified block to the chain
func (c *Chain) AddBlock(block *block.Block) error {
	c.Blocks = append(c.Blocks, block)
	return nil
}

// Save writes the blockchain to a file as JSON
func (c *Chain) Save(path string) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal blockchain: %v", err)
	}

	return os.WriteFile(path, data, 0644)
}

// Load reads blockchain data from a file
func Load(path string) (*Chain, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("failed to read blockchain file: %v", err)
	}

	temp := &Chain{}

	if err := json.Unmarshal(data, temp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blockchain: %v", err)
	}

	c := &Chain{
		Blocks:            temp.Blocks,
		pendingBlocks:     make(map[uint64]*block.Block),
		pendingForkBlocks: make(map[uint64]*block.Block),
	}

	return c, nil
}

// GetNonce calculates the nonce for a given address
func (c *Chain) GetNonce(address string, base int) uint64 {
	var nonce uint64

	for _, blk := range c.Blocks {
		for _, tx := range blk.Transactions {
			if tx.From == address {
				nonce++
			}
		}
	}

	return nonce + uint64(base)
}

// GetBlockByIndex returns the block at the specified index
func (c *Chain) GetBlockByIndex(i uint64) *block.Block {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	if i < uint64(len(c.Blocks)) {
		return c.Blocks[i]
	}

	return nil
}

// GetBlockByHash searches the chain for a block with the given hash
func (c *Chain) GetBlockByHash(hash []byte) *block.Block {
	for _, blk := range c.Blocks {
		if bytes.Equal(blk.Hash, hash) {
			return blk
		}
	}

	for _, blk := range c.pendingForkBlocks {
		if bytes.Equal(blk.Hash, hash) {
			return blk
		}
	}

	return nil
}
