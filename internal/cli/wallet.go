package cli

import (
	"fmt"
	"strconv"

	"github.com/elecbug/lab-chain/internal/user"
	"github.com/elecbug/lab-chain/internal/user/wallet"
)

func walletFunc(user *user.User, args []string) {
	if len(args) != 2 && len(args) != 3 {
		fmt.Printf("Usage: wallet <command> [args]\n")
		return
	}

	command := args[1]

	switch command {
	case "set":
		if user.MasterKey == nil {
			fmt.Printf("No master key loaded. Please load it first.\n")
			return
		}

		idxString := args[2]
		idx, err := strconv.Atoi(idxString)

		if err != nil {
			fmt.Printf("Invalid index: %v.\n", err)

			return
		}

		priv, addr, err := wallet.GenerateAddress(user.MasterKey, idx)

		if err != nil {
			fmt.Printf("Failed to generate address: %v.\n", err)

			return
		} else {
			user.CurrentPrivKey = priv
			user.CurrentAddress = addr

			fmt.Printf("Address generated successfully: index %d, address %s.\n", idx, addr.Hex())
		}
	case "balance":
		if user.MasterKey == nil {
			fmt.Printf("No master key loaded. Please load it first.")
			return
		}

		balance := user.Chain.GetBalance(user.CurrentAddress.Hex())

		fmt.Printf("Current balance: %s.\n", balance.String())
	default:
		fmt.Printf("Usage: wallet <command> [args]\n")
		return
	}
}
