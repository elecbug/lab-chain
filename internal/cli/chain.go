package cli

import (
	"fmt"

	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/elecbug/lab-chain/internal/user"
)

func ChainFunc(user *user.User, args []string) {
	if len(args) != 3 {
		fmt.Printf("Usage: chain <command> <file>\n")
		return
	}

	command := args[1]
	file := args[2]

	switch command {
	case "save":
		if user.Chain == nil {
			fmt.Printf("Blockchain not initialized.\n")
			return
		}

		if err := user.Chain.Save(args[2]); err != nil {
			fmt.Printf("failed to save blockchain: %v.\n", err)

		} else {
			fmt.Printf("blockchain saved successfully.\n")
		}
	case "load":
		if user.Chain != nil {
			fmt.Printf("Blockchain already loaded. Please reset first.\n")
			return
		}

		c, err := chain.Load(file)

		if err != nil {
			fmt.Printf("Failed to load blockchain: %v.\n", err)

			return
		} else {
			fmt.Printf("Blockchain loaded successfully from %s.\n", file)
		}

		user.Chain = c

		subscribeToTopics(user)
	default:
		fmt.Printf("Usage: chain <command> <file>\n")
		return
	}
}

func subscribeToTopics(user *user.User) {
	txSub, err := user.TxTopic.Subscribe()

	if err != nil {
		fmt.Printf("Failed to subscribe to transaction topic: %v.\n", err)

		return
	} else {
		fmt.Printf("Subscribed to transaction topic successfully.\n")
	}

	chain.RunSubscribeAndCollectTx(user.Context, txSub, user.MemPool, user.Chain)

	blkSub, err := user.BlockTopic.Subscribe()

	if err != nil {
		fmt.Printf("failed to subscribe to block topic: %v.\n", err)

		return
	} else {
		fmt.Printf("subscribed to block topic successfully.\n")
	}

	chain.RunSubscribeAndCollectBlock(user.Context, user.BlockTopic, blkSub, user.MemPool, user.Chain)
}
