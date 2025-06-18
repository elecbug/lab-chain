package chain

import (
	"context"
	"fmt"
	"math/big"

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
		Type:   BlockMsgTypeBlock,
		Blocks: []*Block{block},
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
