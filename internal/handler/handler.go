package handler

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/elecbug/lab-chain/internal/chain/block"
	"github.com/elecbug/lab-chain/internal/chain/tx"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/user"
	"github.com/libp2p/go-libp2p/core/peer"
)

// RunSubscribeAndCollectTx listens for incoming transactions on the pubsub subscription
func RunSubscribeAndCollectTx(user *user.User) {
	log := logger.LabChainLogger

	sub, err := user.TxTopic.Subscribe()

	if err != nil {
		fmt.Printf("Failed to subscribe to transaction topic: %v.\n", err)

		return
	} else {
		fmt.Printf("Subscribed to transaction topic successfully.\n")
	}

	go func() {
		for {
			msg, err := sub.Next(user.Context)

			if err != nil {
				log.Errorf("failed to receive pubsub message: %v", err)
				continue
			}

			t, err := tx.DeserializeTx(msg.Data)
			if err != nil {
				log.Warnf("invalid tx: failed to deserialize: %v", err)
				continue
			}

			ok, err := t.VerifySignature()
			if err != nil || !ok {
				log.Warnf("invalid tx: signature verification failed: %v", err)
				continue
			}

			if user.Chain != nil {
				required := new(big.Int).Add(t.Amount, t.Price)
				balance := user.Chain.GetBalance(t.From)
				if balance.Cmp(required) < 0 {
					log.Warnf("invalid tx: insufficient balance. required: %s, actual: %s", required.String(), balance.String())
					continue
				}
			}

			txID := string(t.Signature)
			if user.MemPool.Add(txID, t) {
				log.Infof("transaction received and stored: %s -> %s, amount: %s", t.From, t.To, t.Amount.String())
			} else {
				log.Debugf("transaction already in mp, skipping: %s", txID)
			}
		}
	}()
}

// RunSubscribeAndCollectBlock listens for incoming blocks and processes them accordingly
func RunSubscribeAndCollectBlock(user *user.User) {
	log := logger.LabChainLogger

	sub, err := user.BlockTopic.Subscribe()

	if err != nil {
		fmt.Printf("Failed to subscribe to block topic: %v.\n", err)

		return
	} else {
		fmt.Printf("Subscribed to block topic successfully.\n")
	}

	go func() {
		for {
			msg, err := sub.Next(user.Context)

			if user.PeerID == peer.ID(msg.From) {
				log.Debugf("ignoring block message from self: %s", user.PeerID)
				continue
			}

			if err != nil {
				log.Errorf("failed to receive block message: %v", err)
				continue
			}

			blockMsg, err := block.DeserializeBlockMessage(msg.Data)

			if err != nil {
				log.Warnf("invalid block message received: %v", err)
				continue
			}

			switch blockMsg.Type {
			case block.BlockMsgTypeBlock:
				log.Infof("received block: index %d, miner %s", blockMsg.Blocks[0].Index, blockMsg.Blocks[0].Miner)

				if err := handleIncomingBlock(blockMsg.Blocks[0], user); err != nil {
					log.Warnf("incoming block rejected: %v", err)
				} else {
					log.Infof("block accepted into chain: index %d, hash: %x", blockMsg.Blocks[0].Index, blockMsg.Blocks[0].Hash)

					for _, tx := range blockMsg.Blocks[0].Transactions {
						user.MemPool.Remove(tx)
					}
				}

			case block.BlockMsgTypeReq:
			case block.BlockMsgTypeResp:
			}
		}
	}()
}

// handleIncomingBlock handles incoming blocks and appends them to the chain if valid
func handleIncomingBlock(block *block.Block, user *user.User) error {
	user.Chain.Mu.Lock()
	defer user.Chain.Mu.Unlock()

	log := logger.LabChainLogger
	last := user.Chain.Blocks[len(user.Chain.Blocks)-1]

	// Check if the parent of this block is known
	parent := user.Chain.GetBlockByHash(block.PreviousHash)
	if parent == nil {
		log.Infof("previous hash not found for block index %d", block.Index)
		return fmt.Errorf("unknown parent block: index %d", block.Index)
	}

	// Append to current chain
	if block.Index == last.Index+1 && bytes.Equal(block.PreviousHash, last.Hash) {
		if user.Chain.VerifyBlock(block, last, user.MemPool) {
			return user.Chain.AddBlock(block)
		} else {
			return fmt.Errorf("block failed verification: index %d", block.Index)
		}
	}

	return fmt.Errorf("unacceptable block: index %d", block.Index)
}
