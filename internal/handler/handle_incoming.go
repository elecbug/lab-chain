package handler

import (
	"bytes"
	"fmt"

	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/elecbug/lab-chain/internal/chain/block"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/user"
)

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

		data, err := block.Serialize(respMsg)

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
