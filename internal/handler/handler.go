package handler

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/elecbug/lab-chain/internal/chain/block"
	"github.com/elecbug/lab-chain/internal/chain/tx"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/user"
	"github.com/libp2p/go-libp2p/core/peer"
)

// RunSubscribeAndCollectTx listens for incoming transactions on the pubsub subscription
func RunSubscribeAndCollectTx(user *user.User) {
	go func() {
		log := logger.LabChainLogger

		sub, err := user.TxTopic.Subscribe()

		if err != nil {
			fmt.Printf("Failed to subscribe to transaction topic: %v.\n", err)

			return
		} else {
			fmt.Printf("Subscribed to transaction topic successfully.\n")
		}

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
	go func() {
		log := logger.LabChainLogger

		sub, err := user.BlockTopic.Subscribe()

		if err != nil {
			fmt.Printf("Failed to subscribe to block topic: %v.\n", err)

			return
		} else {
			fmt.Printf("Subscribed to block topic successfully.\n")
		}

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
				log.Infof("received block request from %s", peer.ID(msg.From))

				if err := handleIncomingRequestBlock(blockMsg, user); err != nil {
					log.Warnf("failed to handle block request: %v", err)
				} else {
					log.Infof("block request handled successfully, responding to %s", peer.ID(msg.From))
				}
			case block.BlockMsgTypeResp:
				log.Infof("received block response from %s", peer.ID(msg.From))

				if err := handleIncomingResponseBlock(blockMsg, user); err != nil {
					log.Warnf("failed to handle block response: %v", err)
				} else {
					log.Infof("block response handled successfully, chain updated from %s", peer.ID(msg.From))
				}
			}
		}
	}()
}

// RequestChain sends a request to the peer for the entire chain if the requested block index is out of range
func RequestChain(user *user.User) error {
	log := logger.LabChainLogger

	if user.Chain == nil {
		log.Warnf("user chain is nil, cannot request chain")
		return fmt.Errorf("user chain is nil")
	}

	if len(user.Chain.Blocks) == 0 {
		log.Warnf("user chain is empty, cannot request chain")
		return fmt.Errorf("user chain is empty")
	}

	lastBlock := user.Chain.Blocks[len(user.Chain.Blocks)-1]
	blockMsg := &block.BlockMessage{
		Type: block.BlockMsgTypeReq,
		Idx:  lastBlock.Index,
	}

	data, err := block.SerializeBlockMessage(blockMsg)

	if err != nil {
		log.Errorf("failed to serialize block message: %v", err)
		return err
	}

	if err := user.BlockTopic.Publish(user.Context, data); err != nil {
		log.Errorf("failed to publish block request: %v", err)
		return err
	}

	log.Infof("requested chain from peer %s", user.PeerID)
	return nil
}

// handleIncomingResponseBlock handles incoming block responses
func handleIncomingResponseBlock(blockMsg *block.BlockMessage, user *user.User) error {
	log := logger.LabChainLogger

	user.Chain.Mu.Lock()
	defer user.Chain.Mu.Unlock()

	if len(blockMsg.Blocks) == 0 {
		log.Warnf("received empty block response from %s", user.PeerID)
		return fmt.Errorf("empty block response")
	}

	lastBlock := blockMsg.Blocks[len(blockMsg.Blocks)-1]

	if user.Chain.Blocks[len(user.Chain.Blocks)-1].Index >= lastBlock.Index {
		log.Infof("received block response with index %d, but current chain index is %d, ignoring", lastBlock.Index, user.Chain.Blocks[len(user.Chain.Blocks)-1].Index)
		return nil
	} else {
		log.Infof("received block response with index %d, updating chain", lastBlock.Index)

		newChain := &chain.Chain{
			Blocks: blockMsg.Blocks,
		}

		if err := newChain.VerifyChain(user.Chain.Blocks[0]); err != nil {
			log.Errorf("received invalid chain from %s: %v", user.PeerID, err)
			return fmt.Errorf("invalid chain received: %v", err)
		} else {
			user.Chain.Blocks = newChain.Blocks

			log.Infof("updating chain with blocks from %s", user.PeerID)
		}

		return nil
	}
}

// handleIncomingRequestBlock handles incoming block requests and responds with the requested block
func handleIncomingRequestBlock(blockMsg *block.BlockMessage, user *user.User) error {
	log := logger.LabChainLogger

	user.Chain.Mu.Lock()
	defer user.Chain.Mu.Unlock()

	idx := blockMsg.Idx

	if idx >= uint64(len(user.Chain.Blocks)) {
		log.Infof("requested block index %d is out of range, current chain length is %d", idx, len(user.Chain.Blocks))
		log.Infof("requested chain from %s", user.PeerID)

		err := RequestChain(user)
		return err
	} else {
		log.Infof("responding to block request for index %d", idx)

		respMsg := &block.BlockMessage{
			Type:   block.BlockMsgTypeResp,
			Blocks: user.Chain.Blocks,
		}

		data, err := block.SerializeBlockMessage(respMsg)

		if err != nil {
			log.Errorf("failed to serialize block message: %v", err)
			return err
		}

		if err := user.BlockTopic.Publish(user.Context, data); err != nil {
			log.Errorf("failed to publish block response: %v", err)
			return err
		}

		return nil
	}
}

// handleIncomingBlock handles incoming blocks and appends them to the chain if valid
func handleIncomingBlock(block *block.Block, user *user.User) error {
	log := logger.LabChainLogger

	user.Chain.Mu.Lock()
	defer user.Chain.Mu.Unlock()

	last := user.Chain.Blocks[len(user.Chain.Blocks)-1]

	// Check if the parent of this block is known
	parent := user.Chain.GetBlockByHash(block.PreviousHash)
	if parent == nil {
		log.Infof("previous hash not found for block index %d", block.Index)
		return fmt.Errorf("unknown parent block: index %d", block.Index)
	}

	// Append to current chain
	if block.Index == last.Index+1 && bytes.Equal(block.PreviousHash, last.Hash) {
		if user.Chain.VerifyNewBlock(block, last) {
			return user.Chain.AddBlock(block)
		} else {
			return fmt.Errorf("block failed verification: index %d", block.Index)
		}
	}

	return fmt.Errorf("unacceptable block: index %d", block.Index)
}
