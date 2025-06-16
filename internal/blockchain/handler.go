package blockchain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/elecbug/lab-chain/internal/logger"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// RunSubscribeAndCollectTx listens for incoming transactions on the pubsub subscription
func RunSubscribeAndCollectTx(ctx context.Context, sub *pubsub.Subscription, mempool *Mempool, chain *Blockchain) {
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
func RunSubscribeAndCollectBlock(ctx context.Context, topic *pubsub.Topic, sub *pubsub.Subscription, mempool *Mempool, chain *Blockchain) {
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
			case "BLOCK":
				log.Infof("received block: index %d, miner %s", blockMsg.Block.Index, blockMsg.Block.Miner)

				if err := chain.handleIncomingBlock(ctx, blockMsg.Block, topic); err != nil {
					log.Warnf("incoming block rejected: %v", err)
				} else {
					log.Infof("block accepted into chain: index %d, hash: %x", blockMsg.Block.Index, blockMsg.Block.Hash)

					for _, tx := range blockMsg.Block.Transactions {
						mempool.Remove(tx)
					}
				}

			case "REQ":
				log.Infof("received block request: index %d", blockMsg.ReqIdx)
				blk := chain.GetBlockByIndex(blockMsg.ReqIdx)
				if blk != nil {
					resp := &BlockMessage{
						Type:  "RESP",
						Block: blk,
					}

					data, err := serializeBlockMessage(resp)

					if err == nil {
						topic.Publish(ctx, data)
					}
				}

			case "RESP":
				log.Infof("received block response: index %d", blockMsg.Block.Index)

				if err := chain.handleIncomingBlock(ctx, blockMsg.Block, topic); err != nil {
					log.Warnf("response block rejected: %v", err)
				}
			}
		}
	}()
}

// handleIncomingBlock handles incoming blocks and detects potential forks
// If a longer valid fork is found, switches to that chain
func (bc *Blockchain) handleIncomingBlock(ctx context.Context, block *Block, blkTopic *pubsub.Topic) error {
	bc.Mu.Lock()
	defer bc.Mu.Unlock()

	log := logger.LabChainLogger

	last := bc.Blocks[len(bc.Blocks)-1]

	if block.Index == last.Index+1 && bc.VerifyBlock(block, last) {
		return bc.addBlock(block)
	}

	if block.Index > last.Index+1 {
		bc.pendingBlocks[block.Index] = block

		for i := last.Index + 1; i < block.Index; i++ {
			if _, exists := bc.pendingBlocks[i]; exists {
				continue
			}

			req := &BlockMessage{Type: "REQ", ReqIdx: i}
			data, err := serializeBlockMessage(req)

			if err == nil {
				blkTopic.Publish(ctx, data)
				log.Infof("requested missing block index %d", i)
			}
		}
		return fmt.Errorf("pending block cached: index %d", block.Index)
	}

	if block.Index <= last.Index {
		log.Infof("potential fork detected at index %d", block.Index)

		bc.pendingForkBlocks[block.Index] = block

		forkChain := []*Block{}
		current := block

		for {
			forkChain = append([]*Block{current}, forkChain...) // prepend
			parent := bc.GetBlockByHash(current.PreviousHash)

			if parent != nil {
				break
			}

			missingIdx := current.Index - 1

			if _, ok := bc.pendingForkBlocks[missingIdx]; !ok {
				req := &BlockMessage{Type: "REQ", ReqIdx: missingIdx}

				if data, err := serializeBlockMessage(req); err == nil {
					blkTopic.Publish(ctx, data)
				}

				log.Infof("missing parent block, requested index %d", missingIdx)
				return fmt.Errorf("waiting for parent block %d", missingIdx)
			}

			current = bc.pendingForkBlocks[missingIdx]
		}

		commonAncestor := bc.GetBlockByHash(current.PreviousHash)

		if commonAncestor == nil {
			return fmt.Errorf("no common ancestor found")
		}

		commonIndex := current.Index - 1

		prev := commonAncestor

		for _, b := range forkChain {
			if !bc.VerifyBlock(b, prev) {
				return fmt.Errorf("invalid block in fork chain at index %d", b.Index)
			}
			prev = b
		}

		mainTailLength := len(bc.Blocks) - int(commonIndex) - 1

		if len(forkChain) <= mainTailLength {
			return fmt.Errorf("fork chain not longer than current chain")
		}

		log.Infof("switching to longer fork chain from index %d", commonIndex)
		bc.Blocks = append(bc.Blocks[:commonIndex+1], forkChain...)

		return nil
	}

	return fmt.Errorf("unacceptable block: index %d", block.Index)
}
