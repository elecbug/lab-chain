package blockchain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elecbug/lab-chain/internal/logger"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Block represents a block in the blockchain.
type Block struct {
	Index        uint64 // Block height
	PreviousHash []byte
	Timestamp    int64
	Transactions []*Transaction
	Miner        string
	Nonce        uint64
	Hash         []byte
}

// PublishBlock serializes the block and publishes it to the pubsub topic.
func PublishBlock(ctx context.Context, blkTopic *pubsub.Topic, block *Block) error {
	log := logger.AppLogger

	txBs, err := serializeBlock(block)

	if err != nil {
		return fmt.Errorf("failed to serialize transaction: %v", err)
	} else {
		log.Infof("block serialized successfully: index: %d, miner: %s, nonce: %d, hash: %x",
			block.Index, block.Miner, block.Nonce, block.Hash)
	}

	err = blkTopic.Publish(ctx, txBs)

	if err != nil {
		return fmt.Errorf("failed to publish transaction: %v", err)
	} else {
		log.Infof("block published successfully: index: %d, miner: %s, nonce: %d, hash: %x",
			block.Index, block.Miner, block.Nonce, block.Hash)
	}

	return nil
}

// RunSubscribeAndCollectBlock listens for incoming blocks and adds them to the chain if valid or handles fork resolution
func RunSubscribeAndCollectBlock(ctx context.Context, sub *pubsub.Subscription, chain *Blockchain) {
	log := logger.AppLogger

	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				log.Errorf("failed to receive block: %v", err)
				continue
			}

			block, err := deserializeBlock(msg.Data)

			if err != nil {
				log.Warnf("invalid block received: cannot deserialize: %v", err)
				continue
			}

			log.Infof("received block: index %d, miner %s", block.Index, block.Miner)

			if err := chain.HandleIncomingBlock(block); err != nil {
				log.Warnf("incoming block rejected: %v", err)
			} else {
				log.Infof("block accepted into chain: index %d, hash: %x", block.Index, block.Hash)
			}
		}
	}()
}

// serialize and deserialize functions for block
func serializeBlock(tx *Block) ([]byte, error) {
	jsonBytes, err := json.Marshal(tx)

	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %v", err)
	}

	return jsonBytes, nil
}

// deserialize converts JSON bytes back into a block object
func deserializeBlock(data []byte) (*Block, error) {
	var tx Block

	err := json.Unmarshal(data, &tx)

	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	return &tx, nil
}
