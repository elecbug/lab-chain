package chain

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/elecbug/lab-chain/internal/logger"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
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
func RunSubscribeAndCollectBlock(ctx context.Context, topic *pubsub.Topic, sub *pubsub.Subscription, mempool *Mempool, chain *Chain, sender peer.ID) {
	log := logger.LabChainLogger

	go func() {
		recentRequests := make(map[uint64]time.Time)

		for {
			msg, err := sub.Next(ctx)

			if err != nil {
				log.Errorf("failed to receive block message: %v", err)
				continue
			}

			if peer.ID(msg.From) == sender {
				log.Debugf("ignoring block message from self: %s", sender)
				continue
			}

			blockMsg, err := deserializeBlockMessage(msg.Data)

			if err != nil {
				log.Warnf("invalid block message received: %v", err)
				continue
			}

			switch blockMsg.Type {
			case BlockMsgTypeBlock:
				block := blockMsg.Blocks[0]
				log.Infof("received block: index %d, miner %s, hash %x", block.Index, block.Miner, block.Hash)

				err := chain.handleIncomingBlockAdd(block, chain.Blocks[len(chain.Blocks)-1])

				if err != nil {
					log.Warnf("incoming block rejected: %v", err)

					requestBlocks(ctx, chain.Blocks[len(chain.Blocks)-1].Index+1, block.Index, topic)
				} else {
					log.Infof("block accepted into chain: index %d, hash: %x", block.Index, block.Hash)

					for _, tx := range block.Transactions {
						mempool.Remove(tx)
					}
				}

			case BlockMsgTypeReq:
				log.Infof("received block request: index %v", blockMsg.ReqIdxs)

				blocks := make([]*Block, 0)

				for _, idx := range blockMsg.ReqIdxs {
					blk := chain.GetBlockByIndex(idx)

					if blk != nil {
						blocks = append(blocks, blk)
					}
				}

				if len(blocks) != 0 {
					resp := &BlockMessage{
						Type:   BlockMsgTypeResp,
						Blocks: blocks,
					}

					if len(resp.Blocks) > 0 {
						log.Infof("sending block response: index %d ... %d", resp.Blocks[0].Index, resp.Blocks[len(resp.Blocks)-1].Index)

						data, err := serializeBlockMessage(resp)

						if err == nil {
							topic.Publish(ctx, data)
						}
					}
				}

			case BlockMsgTypeResp:
				if len(blockMsg.Blocks) == 0 {
					log.Warn("received empty block response")
					continue
				} else {
					log.Infof("received block response: index %d ... %d", blockMsg.Blocks[0].Index, blockMsg.Blocks[len(blockMsg.Blocks)-1].Index)
				}

				err = chain.connectBlocks(ctx, blockMsg.Blocks, topic, recentRequests)

				if err != nil {
					log.Warnf("response block rejected: %v", err)
				}
			}
		}
	}()
}

// handleIncomingBlockAdd handles incoming blocks and appends them to the chain if valid
func (c *Chain) handleIncomingBlockAdd(block *Block, last *Block) error {
	log := logger.LabChainLogger

	c.Mu.Lock()
	defer c.Mu.Unlock()

	if last == nil {
		log.Warn("last block is nil, cannot append new block")
		return fmt.Errorf("last block is nil, cannot append new block")
	}

	// Append to current chain
	if block.Index == last.Index+1 && bytes.Equal(block.PreviousHash, last.Hash) {
		if c.VerifyBlock(block, last) {
			return c.insertBlock(block, last)
		}
	}

	return fmt.Errorf("unacceptable block: index %d", block.Index)
}

// shouldRequest determines if we should request the given index (no repeat within 5 seconds)
func shouldRequest(idx uint64, recentRequests map[uint64]time.Time) bool {
	if t, ok := recentRequests[idx]; ok && time.Since(t) < 5*time.Second {
		return false
	}
	recentRequests[idx] = time.Now()
	return true
}

// connectBlocks connects a list of blocks to the chain, ensuring they are in the correct order
func (c *Chain) connectBlocks(ctx context.Context, blocks []*Block, topic *pubsub.Topic, recentRequests map[uint64]time.Time) error {
	log := logger.LabChainLogger

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Index < blocks[j].Index
	})

	pivot := c.GetBlockByHash(blocks[0].PreviousHash)

	if pivot == nil {
		start := indexMinux(blocks[0].Index)
		end := blocks[len(blocks)-1].Index
		for i := start; i <= end; i++ {
			if shouldRequest(i, recentRequests) {
				requestBlocks(ctx, i, i, topic)
			}
		}
		return fmt.Errorf("no common ancestor found")
	}

	// build a temporary chain with verified blocks
	temp := &Chain{Blocks: append([]*Block(nil), c.Blocks[:pivot.Index+1]...)}
	expected := pivot.Index + 1

	for _, blk := range blocks {
		if blk.Index > expected {
			// request missing blocks before this one
			for i := expected; i < blk.Index; i++ {
				if shouldRequest(i, recentRequests) {
					requestBlocks(ctx, i, i, topic)
					log.Infof("requested missing block index: %d", i)
				}
			}
		}
		// verify and append
		if temp.VerifyBlock(blk, temp.Blocks[len(temp.Blocks)-1]) {
			temp.Blocks = append(temp.Blocks, blk)
			expected = blk.Index + 1
		} else {
			return fmt.Errorf("block %d failed verification", blk.Index)
		}
	}

	// replace main chain
	c.Blocks = temp.Blocks
	log.Infof("chain successfully connected up to index: %d", c.Blocks[len(c.Blocks)-1].Index)
	return nil
}

// indexMinux returns the index minus 9, ensuring it does not go below 1
func indexMinux(u uint64) uint64 {
	if u > 10 {
		return u - 9
	} else {
		return 1
	}
}

// requestBlocks sends a request for missing blocks in the specified range to the pubsub topic
func requestBlocks(ctx context.Context, start, end uint64, topic *pubsub.Topic) {
	log := logger.LabChainLogger

	req := &BlockMessage{
		Type:    BlockMsgTypeReq,
		ReqIdxs: []uint64{},
	}

	for i := start; i <= end; i++ {
		req.ReqIdxs = append(req.ReqIdxs, i)
	}

	reqData, err := serializeBlockMessage(req)

	if err != nil {
		log.Errorf("failed to serialize block request: %v", err)
	} else {
		log.Infof("requesting missing blocks: %v", req.ReqIdxs)
		topic.Publish(ctx, reqData)
	}
}
