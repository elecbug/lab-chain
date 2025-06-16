package block

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/transaction"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Block represents a block in the blockchain.
type Block struct {
	Index        uint64 // Block height
	PreviousHash []byte
	Timestamp    int64
	Transactions []*transaction.Transaction
	Miner        string
	Nonce        uint64
	Hash         []byte
}

// MineBlock creates a new block with the given parameters.
func MineBlock(prevHash []byte, index uint64, txs []*transaction.Transaction, miner string, difficultyTarget *big.Int) *Block {
	var nonce uint64
	var hash []byte
	timestamp := time.Now().Unix()

	for {
		header := fmt.Sprintf("%d%x%d%s%d", index, prevHash, timestamp, miner, nonce)
		headerHash := sha256.Sum256([]byte(header))             // 1차: 헤더
		fullData := append(headerHash[:], serializeTxs(txs)...) // 2차: 트랜잭션 포함

		digest := sha256.Sum256(fullData)
		hash = digest[:]

		if new(big.Int).SetBytes(hash).Cmp(difficultyTarget) < 0 {
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

// PublishBlock serializes the block and publishes it to the pubsub topic.
func PublishBlock(ctx context.Context, blkTopic *pubsub.Topic, block *Block) error {
	log := logger.AppLogger

	txBs, err := serialize(block)

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

			block, err := deserialize(msg.Data)

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

// serializeTxs serializes the transactions into a byte slice.
func serializeTxs(txs []*transaction.Transaction) []byte {
	var data []byte

	for _, tx := range txs {
		b, _ := json.Marshal(tx)
		data = append(data, b...)
	}

	return data
}

// serialize and deserialize functions for block
func serialize(tx *Block) ([]byte, error) {
	jsonBytes, err := json.Marshal(tx)

	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %v", err)
	}

	return jsonBytes, nil
}

// deserialize converts JSON bytes back into a block object
func deserialize(data []byte) (*Block, error) {
	var tx Block

	err := json.Unmarshal(data, &tx)

	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	return &tx, nil
}
