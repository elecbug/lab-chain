package cli

import (
	"fmt"

	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/elecbug/lab-chain/internal/user"
)

func MineFunc(user *user.User, args []string) {
	if len(args) == 1 {

		if user.MasterKey == nil {
			fmt.Printf("No master key loaded. Please load it first.\n")
			return
		}
		if user.CurrentAddress == nil {
			fmt.Printf("No current address set. Please set it first.\n")
			return
		}
		if user.Chain == nil {
			fmt.Printf("Blockchain not initialized. Please create genesis block first.\n")
			return
		}

		last := user.Chain.Blocks[len(user.Chain.Blocks)-1]

		txs := user.MemPool.PickTopTxs(20)

		b := user.Chain.MineBlock(last.Hash, last.Index+1, txs, user.CurrentAddress.Hex())
		user.Chain.Blocks = append(user.Chain.Blocks, b)

		err := b.PublishBlock(user.Context, user.BlockTopic)

		if err != nil {
			fmt.Printf("Failed to publish block: %v.\n", err)

		} else {
			fmt.Printf("Block mined and published successfully: index %d, miner %s, nonce %d, hash %x.\n",
				b.Index, b.Miner, b.Nonce, b.Hash)
		}
	} else if len(args) == 2 && args[1] == "genesis" {
		genesisFunc(user)
	} else {
		fmt.Printf("Usage: mine [genesis]\n")
		return
	}
}

func genesisFunc(user *user.User) {
	if user.MasterKey == nil {
		fmt.Printf("No master key loaded. Please load it first.\n")
		return
	}
	if user.CurrentAddress == nil {
		fmt.Printf("No current address set. Please set it first.\n")
		return
	}

	user.Chain = chain.InitBlockchain(user.CurrentAddress.Hex())

	fmt.Printf("Genesis block created successfully: index %d, miner %s, nonce %d, hash %x.\n",
		user.Chain.Blocks[0].Index,
		user.Chain.Blocks[0].Miner,
		user.Chain.Blocks[0].Nonce,
		user.Chain.Blocks[0].Hash,
	)

	b := user.Chain.Blocks[0]
	err := b.PublishBlock(user.Context, user.BlockTopic)

	if err != nil {
		fmt.Printf("Failed to publish block: %v.\n", err)

	} else {
		fmt.Printf("Block mined and published successfully: index %d, miner %s, nonce %d, hash %x.\n",
			b.Index, b.Miner, b.Nonce, b.Hash)
	}

	subscribeToTopics(user)
}
