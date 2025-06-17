package chain

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/elecbug/lab-chain/internal/logger"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// RunSubscribeAndCollectTx listens for incoming transactions on the pubsub subscription
func RunSubscribeAndCollectTx(ctx context.Context, sub *pubsub.Subscription, mempool *Mempool, chain *Chain) {
	log := logger.LabChainLogger

	go func() {
		for {
			msg, err := sub.Next(ctx)

			if err != nil {
				log.Errorf("failed to receive pubsub message: %v", err)
				continue
			}

			tx, err := deserializeTx(msg.Data)
			if err != nil {
				log.Warnf("invalid tx: failed to deserialize: %v", err)
				continue
			}

			ok, err := tx.VerifySignature()
			if err != nil || !ok {
				log.Warnf("invalid tx: signature verification failed: %v", err)
				continue
			}

			if chain != nil {
				required := new(big.Int).Add(tx.Amount, tx.Price)
				balance := chain.GetBalance(tx.From)
				if balance.Cmp(required) < 0 {
					log.Warnf("invalid tx: insufficient balance. required: %s, actual: %s", required.String(), balance.String())
					continue
				}
			}

			txID := string(tx.Signature)
			mempool.mu.Lock()
			if _, exists := mempool.pool[txID]; !exists {
				mempool.pool[txID] = tx
				log.Infof("transaction received and stored: %s -> %s, amount: %s", tx.From, tx.To, tx.Amount.String())
			} else {
				log.Debugf("transaction already in mempool, skipping: %s", txID)
			}
			mempool.mu.Unlock()
		}
	}()
}

// RunSubscribeAndCollectBlock listens for incoming blocks and processes them accordingly
func RunSubscribeAndCollectBlock(ctx context.Context, topic *pubsub.Topic, sub *pubsub.Subscription, mempool *Mempool, chain *Chain) {
	log := logger.LabChainLogger

	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				log.Errorf("failed to receive block message: %v", err)
				continue
			}

			blockMsg, err := deserializeBlockMessage(msg.Data)
			if err != nil {
				log.Warnf("invalid block message received: %v", err)
				continue
			}

			switch blockMsg.Type {
			case BlockMsgTypeBlock:
				log.Infof("received block: index %d, miner %s", blockMsg.Block.Index, blockMsg.Block.Miner)

				if err := chain.handleIncomingBlock(blockMsg.Block); err != nil {
					log.Warnf("incoming block rejected: %v", err)
				} else {
					log.Infof("block accepted into chain: index %d, hash: %x", blockMsg.Block.Index, blockMsg.Block.Hash)

					for _, tx := range blockMsg.Block.Transactions {
						mempool.Remove(tx)
					}
				}

			case BlockMsgTypeReq:
				log.Infof("received block request: index %d", blockMsg.ReqIdx)
				blk := chain.GetBlockByIndex(blockMsg.ReqIdx)
				if blk != nil {
					resp := &BlockMessage{
						Type:  BlockMsgTypeResp,
						Block: blk,
					}

					data, err := serializeBlockMessage(resp)

					if err == nil {
						topic.Publish(ctx, data)
					}
				}

			case BlockMsgTypeResp:
				log.Infof("received block response: index %d", blockMsg.Block.Index)

				if err := chain.handleIncomingBlock(blockMsg.Block); err != nil {
					log.Warnf("response block rejected: %v", err)
				}
			}
		}
	}()
}

// handleIncomingBlock handles incoming blocks and appends them to the chain if valid
func (bc *Chain) handleIncomingBlock(block *Block) error {
	bc.Mu.Lock()
	defer bc.Mu.Unlock()

	log := logger.LabChainLogger
	last := bc.Blocks[len(bc.Blocks)-1]

	// Check if the parent of this block is known
	parent := bc.GetBlockByHash(block.PreviousHash)
	if parent == nil {
		log.Infof("previous hash not found for block index %d", block.Index)
		return fmt.Errorf("unknown parent block: index %d", block.Index)
	}

	// Append to current chain
	if block.Index == last.Index+1 && bytes.Equal(block.PreviousHash, last.Hash) {
		if bc.VerifyBlock(block, last) {
			return bc.addBlock(block)
		} else {
			return fmt.Errorf("block failed verification: index %d", block.Index)
		}
	}

	return fmt.Errorf("unacceptable block: index %d", block.Index)
}
