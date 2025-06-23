package cli

import (
	"fmt"

	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/elecbug/lab-chain/internal/handler"
	"github.com/elecbug/lab-chain/internal/user"
)

func chainFunc(user *user.User, args []string) {
	if len(args) < 2 {
		fmt.Printf("Usage: chain <command> [file]\n")
		return
	}

	command := args[1]

	switch command {
	case "save":
		file := args[2]

		if user.Chain == nil {
			fmt.Printf("Blockchain not initialized.\n")
			return
		}

		if err := user.Chain.Save(file); err != nil {
			fmt.Printf("Failed to save blockchain: %v.\n", err)

		} else {
			fmt.Printf("Blockchain saved successfully.\n")
		}
	case "load":
		file := args[2]

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
	case "request":
		if user.Chain == nil {
			fmt.Printf("Blockchain not initialized.\n")
			return
		}

		if err := handler.RequestChain(user); err != nil {
			fmt.Printf("Failed to request blocks: %v.\n", err)
		} else {
			fmt.Printf("Block request sent successfully.\n")
		}
	default:
		fmt.Printf("Usage: chain <command> <file>\n")
		return
	}
}

func subscribeToTopics(user *user.User) {
	handler.RunSubscribeAndCollectTx(user)

	handler.RunSubscribeAndCollectBlock(user)
}
