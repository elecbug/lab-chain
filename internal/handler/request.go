package handler

import (
	"fmt"

	"github.com/elecbug/lab-chain/internal/chain/block"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/user"
)

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

	data, err := block.Serialize(blockMsg)

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
