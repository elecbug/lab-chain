package chain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"
	"time"

	"github.com/elecbug/lab-chain/internal/logger"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Block represents a block in the blockchain
type Block struct {
	Index        uint64 // Block height
	PreviousHash []byte
	Timestamp    int64
	Transactions []*Transaction
	Miner        string
	Nonce        uint64
	Hash         []byte
	Difficulty   *big.Int // Difficulty for PoW
}

// PublishBlock serializes the block into a BlockMessage and publishes it to the pubsub topic
func (block *Block) PublishBlock(ctx context.Context, blkTopic *pubsub.Topic) error {
	log := logger.LabChainLogger

	// Wrap the block into a BlockMessage
	msg := &BlockMessage{
		Type:  "BLOCK",
		Block: block,
	}

	// Serialize the BlockMessage
	msgBytes, err := serializeBlockMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize block message: %v", err)
	}

	log.Infof("block serialized successfully: index: %d, miner: %s, nonce: %d, hash: %x",
		block.Index, block.Miner, block.Nonce, block.Hash)

	// Publish the message
	err = blkTopic.Publish(ctx, msgBytes)
	if err != nil {
		return fmt.Errorf("failed to publish block message: %v", err)
	}

	log.Infof("block published successfully: index: %d, miner: %s, nonce: %d, hash: %x",
		block.Index, block.Miner, block.Nonce, block.Hash)

	return nil
}

// createGenesisBlock creates the first block in the blockchain with a coinbase transaction
func createGenesisBlock(to string) *Block {
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
